// Package tfmap transpose les ressources d'un plan Terraform vers le modèle
// normalisé commun de Pépin, PILOTÉ PAR UNE SPEC YAML (déclarative) — la même
// grammaire de projection (map + transforms) que la collecte live
// (internal/collect). Objectif : aucun code de mapping Go par provider ; chaque
// provider fournit un mapping-terraform.yaml. Anti-invention (§2) : une garde
// valide les attributs référencés contre le schéma réel du provider Terraform.
package tfmap

import (
	"fmt"
	"strings"

	yaml "go.yaml.in/yaml/v3"

	"github.com/stephrobert/pepin/internal/collect"
	"github.com/stephrobert/pepin/internal/i18n"
	"github.com/stephrobert/pepin/internal/model"
	"github.com/stephrobert/pepin/internal/tfparse"
)

// ResourceSpec : transposition d'un type de ressource Terraform (tf_type) vers un
// type normalisé commun. `map`/`transforms` suivent la grammaire de collect.Project ;
// `const` pose des attributs littéraux ; `region` nomme le champ de localisation.
type ResourceSpec struct {
	TFType     string            `yaml:"tf_type"`
	Type       string            `yaml:"type"`
	ID         string            `yaml:"id"`
	Region     string            `yaml:"region"`
	Items      string            `yaml:"items"` // bloc répété à éclater (ex. "inbound_rule[*]") : 1 ressource TF -> N
	Map        map[string]string `yaml:"map"`
	Transforms map[string]any    `yaml:"transforms"`
	Const      map[string]any    `yaml:"const"`
}

// Spec est la configuration de mapping Terraform d'un provider.
type Spec struct {
	Provider  string         `yaml:"provider"`
	Resources []ResourceSpec `yaml:"resources"`
}

// Parse lit une spec de mapping Terraform (YAML).
func Parse(raw []byte) (Spec, error) {
	var s Spec
	if err := yaml.Unmarshal(raw, &s); err != nil {
		return Spec{}, fmt.Errorf(i18n.T("spec de mapping Terraform invalide : %w", "invalid Terraform mapping spec: %w"), err)
	}
	return s, nil
}

// Apply transpose les ressources du plan vers le modèle commun selon la spec.
// Plusieurs ResourceSpec peuvent viser le même tf_type (ex. SG : inbound_rule +
// outbound_rule). Une spec avec `items` éclate un bloc répété en N ressources.
//
// Les types non déclarés ne sont plus ignorés EN SILENCE : ceux qui appartiennent
// au fournisseur scanné sont enregistrés dans l'état de collecte. Ils n'y sont pas
// une incomplétude — aucun contrôle ne les lit, donc aucun verdict n'en dépend —
// mais un plan qui contient dix ressources dont Pépin n'en projette que six ne
// doit pas laisser croire qu'il a été audité en entier.
//
// Le filtre sur le PRÉFIXE du fournisseur est délibéré : un plan porte
// légitimement des `random_password`, `tls_private_key` ou `null_resource` que
// Pépin n'a jamais prétendu auditer, et les signaler ferait du relevé une liste
// que personne ne lit.
func Apply(spec Spec, resources []tfparse.Resource) model.Inventory {
	var inv model.Inventory
	// idParAdresse : l'identité que CHAQUE ressource du plan prend dans l'inventaire.
	// Sert à la seconde passe, qui remplace une adresse injectée par cette identité.
	idParAdresse := map[string]string{}
	// injectes[i] : les attributs de `inv.Resources[i]` comblés depuis une référence.
	// On ne réécrit qu'eux : une valeur qui ressemblerait à une adresse sans en venir
	// ne doit pas être touchée.
	var injectes []map[string]bool
	var idAttr []string
	declared := map[string]bool{}
	for _, rs := range spec.Resources {
		declared[rs.TFType] = true
	}
	prefix := spec.Provider + "_"
	for _, res := range resources {
		if !declared[res.Type] {
			if strings.HasPrefix(res.Type, prefix) {
				inv.Collection.RecordUnmapped(res.Type, 1)
			}
			continue
		}
		for _, rs := range spec.Resources {
			if rs.TFType != res.Type {
				continue
			}
			// Les références comblent les valeurs AVANT l'extraction des items : un
			// mapping en mode `items` lit son porteur par `_parent.<champ>`, et
			// `_parent` est cette carte de valeurs, attachée à chaque item. Combler
			// après l'extraction ne toucherait que l'item, jamais son parent.
			valeurs := withReferences(res.Values, rs.Map, res.References)
			items := []any{any(valeurs)}
			if rs.Items != "" {
				items = collect.ExtractItems(valeurs, rs.Items)
			}
			region := regionOf(rs, res.Values)
			// La source atteste le TYPE de ressource du plan, pas l'adresse : l'adresse
			// est déjà l'identifiant de la ressource, le type est ce qui dit d'où la
			// valeur a été lue. Une valeur issue d'un plan n'est PAS une observation de
			// la configuration effective, et l'origine `terraform-plan` le porte.
			src := collect.Source{Origin: model.OriginTerraform, Ref: res.Type}
			for _, it := range items {
				attrs, prov := collect.ProjectAttested(it, rs.Map, rs.Transforms, src)
				attestReferences(&prov, rs.Map, res)
				for k, v := range rs.Const {
					attrs[k] = v
				}
				collect.AttestConst(&prov, rs.Const, "descriptor:const")
				id, _ := attrs[rs.ID].(string)
				if id == "" {
					id = res.Address
				}
				name := id
				if n, ok := res.Values["name"].(string); ok && n != "" {
					name = n
				}
				idParAdresse[res.Address] = id
				injectes = append(injectes, champsInjectes(rs.Map, res))
				idAttr = append(idAttr, rs.ID)
				inv.Resources = append(inv.Resources, model.Resource{
					Provider: spec.Provider, Type: rs.Type, ID: id, Name: name, Region: region,
					Attributes: attrs, Provenance: prov,
					// L'origine voyage avec la ressource : une spec `items` éclate un bloc
					// répété en N ressources normalisées, qui viennent toutes du MÊME bloc
					// HCL. Leur donner la même origine est fidèle — c'est bien ce bloc qu'il
					// faut corriger.
					Source: sourceRefOf(res.Origin),
				})
			}
		}
	}
	resoudreVersLIdentite(inv.Resources, injectes, idAttr, idParAdresse)
	return inv
}

// resoudreVersLIdentite remplace, dans les attributs comblés depuis une référence,
// l'ADRESSE de la ressource visée par l'IDENTITÉ qu'elle porte dans l'inventaire.
//
// Une référence désigne une ressource ; ce qui la nomme dans l'inventaire n'est pas
// forcément son adresse. Une instance de base managée qui porte un `name` s'identifie
// par ce nom, quand une ressource sans identifiant lisible retombe sur son adresse.
//
// Sans cette passe, une même base obtenait DEUX sujets sur un plan : la règle dérivée
// de son ACL la nommait `scaleway_rdb_instance.exposed`, celles dérivées de l'instance
// `pepin-rdb`. Trois écarts, deux noms — et une dérogation écrite sur l'un ratait
// l'autre, en silence.
//
// La réécriture est bornée aux attributs INJECTÉS : une valeur qui ressemblerait à une
// adresse sans en venir n'est jamais touchée.
func resoudreVersLIdentite(rs []model.Resource, injectes []map[string]bool, idAttr []string, idParAdresse map[string]string) {
	if len(idParAdresse) == 0 {
		return
	}
	for i := range rs {
		if i >= len(injectes) || len(injectes[i]) == 0 {
			continue
		}
		// L'IDENTITÉ d'abord : elle est dérivée d'un attribut, donc si cet attribut a
		// été comblé par une référence, l'identité porte une adresse. La recalculer
		// après coup serait fragile ; la réécrire ici la garde alignée sur l'attribut.
		if injectes[i][idAttr[i]] {
			if id, ok := idParAdresse[rs[i].ID]; ok {
				if rs[i].Name == rs[i].ID {
					rs[i].Name = id
				}
				rs[i].ID = id
			}
		}
		for attr := range injectes[i] {
			switch v := rs[i].Attributes[attr].(type) {
			case string:
				if id, ok := idParAdresse[v]; ok {
					rs[i].Attributes[attr] = id
				}
			case []any:
				for j, e := range v {
					if adr, ok := e.(string); ok {
						if id, ok := idParAdresse[adr]; ok {
							v[j] = id
						}
					}
				}
			}
		}
	}
}

// champsInjectes rend les attributs qu'une référence a comblés sur cette ressource.
func champsInjectes(mapping map[string]string, res tfparse.Resource) map[string]bool {
	out := map[string]bool{}
	for attr, brut := range mapping {
		chemin, ok := champArgument(brut)
		if !ok {
			continue
		}
		if _, present := res.Values[chemin]; present {
			continue
		}
		if len(res.References[chemin]) > 0 {
			out[attr] = true
		}
	}
	return out
}

// sourceRefOf transpose l'origine d'une ressource du plan vers le modèle commun.
// Une origine vide reste ABSENTE (pointeur nil) : porter un objet à trois champs
// vides dans chaque ressource d'un inventaire live donnerait l'apparence d'une
// information qui n'existe pas.
func sourceRefOf(o tfparse.Origin) *model.SourceRef {
	if o.Empty() {
		return nil
	}
	return &model.SourceRef{File: o.File, Line: o.Line, Module: o.Module}
}

// withReferences complète un item du plan avec les ADRESSES que l'exploitant a
// référencées là où le plan n'a pas encore de valeur.
//
// L'injection se fait AVANT la projection, plutôt qu'à côté : la spec, ses
// transforms et l'attestation s'appliquent alors sans rien savoir de la manœuvre, et
// il n'existe qu'un seul chemin de projection à maintenir.
//
// Trois bornes, et chacune retire un moyen de se tromper :
//
//   - on ne complète QUE ce que le plan n'a pas. Une valeur présente gagne toujours,
//     donc un plan appliqué (`values`, tout résolu) est strictement inchangé ;
//   - on ne complète que ce qui désigne un ARGUMENT de la ressource : un chemin
//     simple (`vm_id`), ou `_parent.<champ>` — qui, en mode `items`, désigne un
//     argument du porteur et non une structure imbriquée. `a.b` reste écarté : il
//     descend dans une structure, et le compléter mêlerait deux espaces de noms.
//     Sans le cas `_parent.`, une base de données obtenait DEUX sujets sur un plan :
//     la règle dérivée de l'ACL nommait la ressource ACL, celles dérivées de
//     l'instance nommaient la base, et une dérogation écrite sur l'une ratait l'autre ;
//   - une seule adresse est projetée comme SCALAIRE, plusieurs comme LISTE. C'est la
//     forme que la spec attend de la source native, et le transform `list` de la spec
//     retombe sur ses pieds dans les deux cas.
func withReferences(valeurs map[string]any, mapping map[string]string, refs map[string][]string) map[string]any {
	if len(refs) == 0 || valeurs == nil {
		return valeurs
	}
	var complete map[string]any
	for _, brut := range mapping {
		chemin, ok := champArgument(brut)
		if !ok {
			continue
		}
		if _, present := valeurs[chemin]; present {
			continue
		}
		adresses := refs[chemin]
		if len(adresses) == 0 {
			continue
		}
		if complete == nil {
			complete = make(map[string]any, len(valeurs)+1)
			for k, v := range valeurs {
				complete[k] = v
			}
		}
		if len(adresses) == 1 {
			complete[chemin] = adresses[0]
			continue
		}
		liste := make([]any, len(adresses))
		for i, a := range adresses {
			liste[i] = a
		}
		complete[chemin] = liste
	}
	if complete == nil {
		return valeurs
	}
	return complete
}

// champArgument rend le nom de l'ARGUMENT qu'un chemin de mapping désigne, et faux si
// le chemin descend dans une structure.
//
// `vm_id` désigne un argument. `_parent.instance_id` aussi : en mode `items`, `_parent`
// est la carte des valeurs du porteur, donc son suffixe est un argument de la
// ressource. `audit.0.endpoint` non — c'est une descente, et une référence n'y répond
// pas.
func champArgument(chemin string) (string, bool) {
	chemin = strings.TrimPrefix(chemin, "_parent.")
	if chemin == "" || strings.ContainsAny(chemin, ".[") {
		return "", false
	}
	return chemin, true
}

// attestReferences corrige l'attestation des attributs COMPLÉTÉS par une référence.
//
// Sans cela, la provenance dirait que la valeur a été lue dans `planned_values` à
// l'attribut du même nom, ce qui est faux : elle vient de `configuration`, et elle
// porte une adresse plutôt que la valeur du champ. Une traçabilité qui désigne le
// mauvais endroit est pire que son absence — c'est la règle que le modèle de
// provenance énonce lui-même pour les appels d'API.
func attestReferences(prov *model.Provenance, mapping map[string]string, res tfparse.Resource) {
	for attr, brut := range mapping {
		chemin, ok := champArgument(brut)
		if !ok {
			continue
		}
		if _, present := res.Values[chemin]; present {
			continue
		}
		if len(res.References[chemin]) == 0 {
			continue
		}
		prov.Attest(attr, model.Attestation{
			Origin:   model.OriginTerraform,
			Source:   "configuration:" + res.Type,
			Path:     chemin + ".references",
			Observed: true,
			Derived:  true,
		})
	}
}

// regionOf lit la région d'une ressource du plan, en lui appliquant le transform que
// la spec déclare pour son champ source.
//
// Un plan localise par la ZONE, pas par la région : Scaleway écrit `fr-par-1` sur un
// serveur, Outscale `eu-west-2a` sur une VM. Lire la valeur brute, comme on le faisait,
// posait donc `fr-par-1` en guise de région — un nom qu'aucun catalogue ne connaît, et
// qui rendait le contrôle de souveraineté muet là où la localisation est pourtant
// écrite en clair.
//
// Le transform se déclare sous le NOM DU CHAMP SOURCE (`transforms: {zone:
// region_of_zone}`), parce que la région n'est pas un attribut de `map` et n'a donc pas
// de nom commun sous lequel l'indexer.
func regionOf(rs ResourceSpec, values map[string]any) string {
	if rs.Region == "" {
		return ""
	}
	t := map[string]any{}
	if tr, ok := rs.Transforms[rs.Region]; ok {
		t["region"] = tr
	}
	out := collect.Project(values, map[string]string{"region": rs.Region}, t)
	r, _ := out["region"].(string)
	return r
}

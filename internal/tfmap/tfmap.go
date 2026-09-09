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
			items := []any{any(res.Values)}
			if rs.Items != "" {
				items = collect.ExtractItems(res.Values, rs.Items)
			}
			region := regionOf(rs, res.Values)
			// La source atteste le TYPE de ressource du plan, pas l'adresse : l'adresse
			// est déjà l'identifiant de la ressource, le type est ce qui dit d'où la
			// valeur a été lue. Une valeur issue d'un plan n'est PAS une observation de
			// la configuration effective, et l'origine `terraform-plan` le porte.
			src := collect.Source{Origin: model.OriginTerraform, Ref: res.Type}
			for _, it := range items {
				it = withReferences(it, rs.Map, res.References)
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
	return inv
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
//   - on ne complète que les chemins SIMPLES. `_parent.x` ou `a.b` désignent une
//     structure, pas un argument de la configuration : les compléter mêlerait deux
//     espaces de noms ;
//   - une seule adresse est projetée comme SCALAIRE, plusieurs comme LISTE. C'est la
//     forme que la spec attend de la source native, et le transform `list` de la spec
//     retombe sur ses pieds dans les deux cas.
func withReferences(item any, mapping map[string]string, refs map[string][]string) any {
	if len(refs) == 0 {
		return item
	}
	valeurs, ok := item.(map[string]any)
	if !ok {
		return item
	}
	var complete map[string]any
	for _, chemin := range mapping {
		if chemin == "" || strings.ContainsAny(chemin, ".[") {
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
		return item
	}
	return complete
}

// attestReferences corrige l'attestation des attributs COMPLÉTÉS par une référence.
//
// Sans cela, la provenance dirait que la valeur a été lue dans `planned_values` à
// l'attribut du même nom, ce qui est faux : elle vient de `configuration`, et elle
// porte une adresse plutôt que la valeur du champ. Une traçabilité qui désigne le
// mauvais endroit est pire que son absence — c'est la règle que le modèle de
// provenance énonce lui-même pour les appels d'API.
func attestReferences(prov *model.Provenance, mapping map[string]string, res tfparse.Resource) {
	for attr, chemin := range mapping {
		if chemin == "" || strings.ContainsAny(chemin, ".[") {
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

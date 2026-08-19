package assess

import (
	"sort"
	"strconv"
	"strings"

	"github.com/stephrobert/pepin/internal/model"
	"github.com/stephrobert/scankit/assessment"
)

// La provenance côté assessment : dire, pour l'attribut DÉCISIF de chaque contrôle,
// d'où vient la donnée et si elle a réellement été observée.
//
// Le choix de conception, et son coût. Porter la provenance DANS chaque valeur
// (`{"value": false, "observed": true, "source": …}`) obligerait à réécrire les
// cinquante-neuf règles, qui lisent toutes `attributes.<nom>` — et une règle
// réécrite est une règle qui peut changer de verdict. L'attestation vit donc à
// côté, dans un index parallèle porté par la ressource (model.Provenance), et
// l'assessment n'en publie qu'un RÉSUMÉ, sur les attributs qui décident d'un
// « pass ». Le modèle grossit à la collecte (une entrée par attribut mappé), pas
// dans le rapport.
//
// Ce que ce résumé rend visible, et qui ne l'était pas : deux contrôles franchissent
// aujourd'hui leur garde d'attribut grâce à un littéral de descripteur
// (`const: {encrypted: true}` chez Exoscale, `const: {scope: account}` chez
// Outscale). Leur verdict ne change pas — ce lot ne touche pas au verrou — mais
// leur preuve dit désormais « derived:descriptor:const » au lieu de laisser croire
// à une mesure. Resserrer le verrou est une décision, et une décision se prend en
// voyant.

// AttrOrigin résume la provenance d'UN attribut sur les ressources d'un type.
type AttrOrigin struct {
	// Sources : les étiquettes « <origine>:<source> » distinctes rencontrées, triées.
	Sources []string
	// Observed : ressources dont la source portait réellement le champ.
	Observed int
	// Total : ressources du type qui déclarent cet attribut (attesté ou non).
	Total int
}

// Label rend le résumé sous une forme compacte et SANS PROSE : il voyage dans
// `evidence.source`, que des outils parsent, et une chaîne traduite y serait un
// piège (le même bundle dirait deux choses selon la langue de son auteur).
func (o AttrOrigin) Label() string {
	return strings.Join(o.Sources, " + ") + " observed=" +
		strconv.Itoa(o.Observed) + "/" + strconv.Itoa(o.Total)
}

// ProvenanceIndex : type de ressource -> attribut -> résumé de provenance.
type ProvenanceIndex map[string]map[string]AttrOrigin

// ProvenanceOf construit l'index depuis l'inventaire évalué. Il accepte les deux
// formes que le scan manipule : les ressources typées (collecte live, plan
// Terraform) et la forme générique d'un export JSON relu — dont l'input.json d'un
// bundle scellé, pour que `verify --re-derive` reconstruise le MÊME assessment.
//
// Un export sans `provenance` produit un index vide : une attestation absente reste
// absente, elle ne s'invente pas. C'est la raison pour laquelle un inventaire
// importé n'a pas d'origine « export » : Pépin n'a pas observé cet inventaire, il
// l'a reçu.
func ProvenanceOf(input any) ProvenanceIndex {
	idx := ProvenanceIndex{}
	m, ok := input.(map[string]any)
	if !ok {
		return idx
	}
	switch rs := m["resources"].(type) {
	case []model.Resource:
		for _, r := range rs {
			idx.add(r.Type, r.Provenance)
		}
	case []any:
		for _, it := range rs {
			rm, ok := it.(map[string]any)
			if !ok {
				continue
			}
			typ, _ := rm["type"].(string)
			raw, _ := rm["provenance"].(map[string]any)
			idx.add(typ, provenanceFromJSON(raw))
		}
	}
	for _, byAttr := range idx {
		for attr, o := range byAttr {
			sort.Strings(o.Sources)
			byAttr[attr] = o
		}
	}
	return idx
}

// add fusionne l'attestation d'une ressource dans l'index de son type.
func (idx ProvenanceIndex) add(typ string, prov model.Provenance) {
	if typ == "" || len(prov) == 0 {
		return
	}
	if idx[typ] == nil {
		idx[typ] = map[string]AttrOrigin{}
	}
	for attr, att := range prov {
		o := idx[typ][attr]
		o.Total++
		if att.Observed {
			o.Observed++
		}
		if label := attestationLabel(att); label != "" && !containsStr(o.Sources, label) {
			o.Sources = append(o.Sources, label)
		}
		idx[typ][attr] = o
	}
}

// attestationLabel rend « <origine>:<source> » (ou la seule origine si la source
// est vide). Un `derived` porte en plus la marque `~` quand il est calculé sur une
// source réelle (agrégat), pour ne pas le confondre avec une valeur recopiée.
func attestationLabel(a model.Attestation) string {
	if a.Origin == "" {
		return ""
	}
	label := string(a.Origin)
	if a.Source != "" {
		label += ":" + a.Source
	}
	if a.Derived && a.Origin != model.OriginDerived {
		label += " (derived)"
	}
	return label
}

// provenanceFromJSON relit une carte d'attestations telle qu'elle sort d'un
// input.json. Défensif : une entrée mal formée est ignorée, jamais une panique —
// le bundle vient d'un tiers.
func provenanceFromJSON(raw map[string]any) model.Provenance {
	if len(raw) == 0 {
		return nil
	}
	out := make(model.Provenance, len(raw))
	for attr, v := range raw {
		am, ok := v.(map[string]any)
		if !ok {
			continue
		}
		origin, _ := am["origin"].(string)
		source, _ := am["source"].(string)
		path, _ := am["path"].(string)
		observed, _ := am["observed"].(bool)
		derived, _ := am["derived"].(bool)
		out[attr] = model.Attestation{
			Origin: model.Origin(origin), Source: source, Path: path,
			Observed: observed, Derived: derived,
		}
	}
	return out
}

// WithProvenance annote chaque résultat dont le contrôle a un attribut DÉCISIF avec
// l'attribut en question et l'attestation de sa donnée.
//
// C'est une PASSE POSTÉRIEURE, et elle l'est par conception : elle n'écrit que
// `evidence.attribute` et `evidence.source`, ne lit jamais un statut et n'en écrit
// aucun. La non-régression des verdicts n'est donc pas une propriété à vérifier au
// cas par cas, elle est structurelle — et TestProvenanceNeverMovesAVerdict la
// mesure quand même, parce qu'une propriété qu'on affirme sans la mesurer est une
// propriété qu'on espère.
func WithProvenance(a assessment.Assessment, idx ProvenanceIndex, controlType map[string]string) assessment.Assessment {
	if len(idx) == 0 {
		return a
	}
	res := append([]assessment.Result(nil), a.Results...)
	for i := range res {
		attrs := requiredAttr[res[i].Control]
		if len(attrs) == 0 {
			continue
		}
		byAttr := idx[controlType[res[i].Control]]
		if byAttr == nil {
			continue
		}
		var attested []string
		var labels []string
		for _, attr := range attrs {
			o, ok := byAttr[attr]
			if !ok {
				continue
			}
			attested = append(attested, attr)
			labels = append(labels, attr+"="+o.Label())
		}
		if len(attested) == 0 {
			continue
		}
		res[i].Evidence.Attribute = strings.Join(attested, " / ")
		res[i].Evidence.Source = strings.Join(labels, " ; ")
	}
	a.Results = res
	return a
}

func containsStr(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

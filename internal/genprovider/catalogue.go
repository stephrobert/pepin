package genprovider

import (
	"sort"

	"github.com/stephrobert/pepin/internal/eimpolicy"
	"github.com/stephrobert/pepin/internal/objectstorage"
	"github.com/stephrobert/pepin/internal/oks"
)

// InventoryCatalogue énumère le vocabulaire de l'inventaire normalisé : chaque
// type de ressource et les attributs communs que Pépin sait y projeter, tous
// fournisseurs et toutes sources confondus.
//
// Il est DÉRIVÉ des descripteurs chargés et des collecteurs Go, jamais recopié :
// une liste tenue à la main à côté du code diverge au premier attribut ajouté, et
// une énumération fausse est pire qu'une énumération absente. C'est cette
// dérivation qui permet de geler le vocabulaire (cmd/testdata/frozen/inventory.json)
// sans qu'un contributeur ait à penser à mettre une table à jour.
//
// Ce que le catalogue dit : « cet attribut existe dans le modèle, et voici sur
// quel type ». Ce qu'il ne dit PAS : qu'il est présent sur une ressource donnée
// d'un scan donné — cette question-là est celle de la provenance.
func InventoryCatalogue() map[string][]string {
	byType := map[string]map[string]bool{}
	add := func(typ string, attrs ...string) {
		if typ == "" {
			return
		}
		if byType[typ] == nil {
			byType[typ] = map[string]bool{}
		}
		for _, a := range attrs {
			if a != "" {
				byType[typ][a] = true
			}
		}
	}
	for _, d := range registry {
		for _, r := range d.Collecte.Resources {
			add(r.Type, keysOf(r.Map)...)
			add(r.Type, keysOfAny(r.Const)...)
			add(r.Type, r.Aggregate)
		}
		for _, r := range d.MappingTerraform.Resources {
			add(r.Type, keysOf(r.Map)...)
			add(r.Type, keysOfAny(r.Const)...)
		}
		// La ressource synthétique de souveraineté n'est collectée nulle part : ses
		// attributs viennent du descripteur, et elle appartient pourtant au modèle.
		if res, ok := (GenericProvider{desc: d}).GovernanceResource(); ok {
			add(res.Type, keysOfAny(res.Attributes)...)
		}
	}
	// Les collecteurs Go (stockage objet, Kubernetes managé, politiques EIM inline)
	// projettent hors du moteur YAML : leurs attributs sont déclarés à côté de leur
	// code, et lus ici — même principe, une seule source.
	add("object_storage_bucket", objectstorage.BucketAttributes()...)
	add("kubernetes_cluster", oks.ClusterAttributes()...)
	add("iam_policy", eimpolicy.PolicyAttributes()...)

	out := make(map[string][]string, len(byType))
	for typ, attrs := range byType {
		list := make([]string, 0, len(attrs))
		for a := range attrs {
			list = append(list, a)
		}
		sort.Strings(list)
		out[typ] = list
	}
	return out
}

func keysOf(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func keysOfAny(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

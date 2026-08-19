package cmd

import (
	"encoding/json"
	"os"
	"testing"
)

// FuzzInventoryWalk — l'export d'inventaire est fourni par un TIERS
// (`pepin scan <provider> export.json`), décodé en `any`, puis parcouru par
// resourceTypesOf et attrsByTypeOf avant toute évaluation.
//
// Ces deux fonctions décident de ce que le verdict a le droit d'affirmer :
// attrsByTypeOf alimente le verrou de capacité, celui qui autorise ou non un
// « pass ». Une panique y ferait tomber le scan sur une entrée choisie par
// l'audité ; une lecture trop permissive lui ferait affirmer une conformité
// jamais mesurée. Les deux sont couverts ici : pas de panique, et un attribut
// n'est jamais compté comme collecté si sa valeur est une collection vide.
func FuzzInventoryWalk(f *testing.F) {
	for _, p := range []string{
		"../examples/scaleway/inventory.json",
		"../examples/scaleway/inventory-ok.json",
	} {
		if b, err := os.ReadFile(p); err == nil {
			f.Add(b)
		}
	}
	f.Add([]byte(`{"resources":[]}`))
	f.Add([]byte(`{"resources":null}`))
	f.Add([]byte(`{"resources":"pas-un-tableau"}`))
	f.Add([]byte(`{"resources":[{"type":"bucket","attributes":null}]}`))
	f.Add([]byte(`{"resources":[{"type":"bucket","attributes":"boom"}]}`))
	f.Add([]byte(`{"resources":[{"type":123,"attributes":{"acl":[]}}]}`))
	f.Add([]byte(`[]`))
	f.Add([]byte(`null`))
	f.Add([]byte(`"chaine"`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var input any
		if err := json.Unmarshal(data, &input); err != nil {
			return // un JSON invalide est refusé en amont, pas notre sujet
		}

		// Aucun des deux parcours ne doit paniquer, quelle que soit la forme.
		_ = resourceTypesOf(input)
		attrs := attrsByTypeOf(input)

		// Le verrou de capacité repose sur une propriété précise : une valeur
		// vide n'est PAS une collecte. C'est ce qui empêche `statements: []`,
		// toujours posé par le collecteur IAM, de faire conclure « conforme »
		// sur zéro information. On la vérifie ici sur des entrées arbitraires.
		m, ok := input.(map[string]any)
		if !ok {
			return
		}
		rs, ok := m["resources"].([]any)
		if !ok {
			return
		}
		for _, it := range rs {
			rm, ok := it.(map[string]any)
			if !ok {
				continue
			}
			typ, ok := rm["type"].(string)
			if !ok || typ == "" {
				continue
			}
			am, ok := rm["attributes"].(map[string]any)
			if !ok {
				continue
			}
			for k, v := range am {
				switch vv := v.(type) {
				case []any:
					if len(vv) == 0 && attrs[typ][k] {
						t.Fatalf("attribut %q de type %q compté comme collecté alors qu'il est une liste vide", k, typ)
					}
				case map[string]any:
					if len(vv) == 0 && attrs[typ][k] {
						t.Fatalf("attribut %q de type %q compté comme collecté alors qu'il est un objet vide", k, typ)
					}
				case nil:
					if attrs[typ][k] {
						t.Fatalf("attribut %q de type %q compté comme collecté alors qu'il est nul", k, typ)
					}
				}
			}
		}
	})
}

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
// jamais mesurée.
//
// L'INVARIANT A CHANGÉ AVEC L'ISSUE #227, et la raison mérite d'être écrite parce
// qu'elle explique pourquoi un test de fuzzing peut devenir faux.
//
// Il disait : une collection VIDE n'est jamais collectée. C'était un garde-fou par
// PROCURATION. Ce qu'il protégeait réellement, c'est qu'un document de politique
// illisible ne fasse pas conclure : `IAMPolicyStatements` rendait `[]` aussi bien pour
// « ce document n'a pas pu être analysé » que pour « cette politique n'accorde rien »,
// et le verrou se défendait en se méfiant de tous les vides. Le prix était lourd : une
// instance SANS aucun groupe de sécurité — l'écart même qu'un contrôle cherche — faisait
// taire ce contrôle, sur elle et sur ses voisines.
//
// Le bouchon a été retiré à la source : le parseur rend désormais `nil` quand il n'a pas
// su lire, et les collecteurs OMETTENT alors l'attribut. Le verrou peut donc répondre sur
// la PRÉSENCE, et ce test garde les deux propriétés qui tiennent vraiment :
//
//	nil n'est jamais collecté                      — un scalaire nul n'établit rien
//	un attribut absent d'UNE ressource du type      — c'est l'intersection, et c'est elle
//	n'est jamais collecté pour ce type                qui empêche l'union de faire conclure
//	                                                  sur une ressource jamais observée
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
	// #227 : une liste vide OBSERVÉE, et une ressource voisine qui porte la même clé.
	f.Add([]byte(`{"resources":[{"type":"vm","attributes":{"sg":[]}},{"type":"vm","attributes":{"sg":["a"]}}]}`))
	// Une valeur NULLE explicite : elle ne dit pas ce que la valeur est, seulement
	// qu'il n'y en a pas — absent et inconnu s'y confondent, donc elle n'établit rien.
	f.Add([]byte(`{"resources":[{"type":"vm","attributes":{"sg":null}}]}`))
	// Et le contre-exemple : la clé manque à une ressource du même type.
	f.Add([]byte(`{"resources":[{"type":"vm","attributes":{"sg":[]}},{"type":"vm","attributes":{}}]}`))
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

		m, ok := input.(map[string]any)
		if !ok {
			return
		}
		rs, ok := m["resources"].([]any)
		if !ok {
			return
		}
		// Les ressources par type, telles que l'entrée les donne.
		parType := map[string][]map[string]any{}
		for _, it := range rs {
			rm, ok := it.(map[string]any)
			if !ok {
				continue
			}
			typ, ok := rm["type"].(string)
			if !ok || typ == "" {
				continue
			}
			// Une ressource porteuse de PROVENANCE sort du test : un attribut absent des
			// `attributes` mais attesté a été CHERCHÉ, et le verrou le compte à ce titre
			// (ADR-0007). Le mesurer ici demanderait de réimplémenter cette règle, donc
			// de tester le test.
			if _, atteste := rm["provenance"]; atteste {
				continue
			}
			if _, atteste := m["provenance"]; atteste {
				return
			}
			am, ok := rm["attributes"].(map[string]any)
			if !ok {
				continue
			}
			parType[typ] = append(parType[typ], am)
		}

		for typ, ressources := range parType {
			for _, am := range ressources {
				for k, v := range am {
					// `nil` n'établit rien : un scalaire nul ne dit pas ce que la valeur
					// est, seulement qu'il n'y en a pas — absent et inconnu s'y confondent.
					if v == nil && attrs[typ][k] {
						t.Fatalf("attribut %q de type %q compté comme collecté alors qu'il est nul", k, typ)
					}
				}
			}
			// L'INTERSECTION : un attribut qu'une seule ressource du type ne porte pas
			// n'est pas collecté POUR CE TYPE. C'est ce qui empêche une ressource
			// porteuse d'ouvrir la porte du `pass` à ses voisines jamais observées.
			for _, am := range ressources {
				for k := range am {
					manquant := false
					for _, autre := range ressources {
						if _, present := autre[k]; !present {
							manquant = true
							break
						}
					}
					if manquant && attrs[typ][k] {
						t.Fatalf("attribut %q de type %q compté comme collecté alors qu'une ressource de ce type ne le porte pas — c'est l'union, pas l'intersection", k, typ)
					}
				}
			}
		}
	})
}

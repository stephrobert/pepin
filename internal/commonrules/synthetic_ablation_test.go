package commonrules_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/stephrobert/scankit/engine"

	"github.com/stephrobert/pepin/internal/commonrules"
)

// La porte d'ablation du corpus (ablation_test.go) mesure le comportement RÉEL, mais
// elle ne voit que ce que les fixtures atteignent : 65 ablations sur 6 types de
// ressource. Les règles corrigées dans ce jalon — VM publique, journalisation,
// passthrough TLS — n'y sont exercées par aucune fixture.
//
// Celle-ci ne dépend d'aucun corpus. Elle SYNTHÉTISE, pour chaque contrôle qui
// déclare un attribut décisif, une ressource minimale du type qu'il lit, puis retire
// cet attribut et exige qu'aucun écart n'APPARAISSE.
//
// Les deux portes sont complémentaires et aucune ne remplace l'autre :
//
//   - le corpus mesure des configurations que personne n'a conçues pour ces règles,
//     donc il attrape ce que l'auteur n'avait pas prévu ;
//   - la synthèse couvre TOUTES les règles qui déclarent leur donnée décisive, donc
//     elle ne laisse aucune règle hors de portée faute de fixture.
//
// CE QU'ELLE NE VOIT PAS, mesuré plutôt que supposé. Les deux défauts corrigés dans
// ce jalon ont été réintroduits pour l'éprouver, et elle n'attrape NI l'un NI l'autre :
//
//   - `network_peering_cross_organization` exige deux comptes DIFFÉRENTS et non vides.
//     La sonde peuple tous les attributs de la même valeur, donc les deux comptes sont
//     égaux et la règle ne se déclenche jamais — ni avant, ni après l'ablation. Une
//     règle dont le `deny` dépend d'une COMBINAISON reste hors de portée.
//   - `loadbalancer_logging_enabled` lit `access_log` puis `is_enabled` à travers un
//     `object.get` IMBRIQUÉ. La sonde met une chaîne dans `access_log`, le second
//     `object.get` retombe sur son défaut, et la règle se déclenche DÉJÀ dans la base
//     — le cas est alors écarté, faute de base silencieuse.
//
// Cette porte couvre donc les règles à condition SIMPLE, qui sont la majorité. Elle
// ne remplace ni les tests unitaires Rego, ni la porte de corpus, ni l'extension du
// corpus elle-même (#123), qui reste le seul moyen d'éprouver une règle sur une
// configuration que son auteur n'a pas conçue.

// probeValues — valeurs plausibles pour un attribut. On en essaie PLUSIEURS : une
// règle se tait sur l'une et se déclenche sur l'autre, et seule une base silencieuse
// laisse voir un écart qui APPARAÎT.
func probeValues(attr string) []any {
	switch {
	case attr == "user_data":
		return []any{"#!/bin/sh\necho ok\n"}
	case attr == "region":
		return []any{"fr-par"}
	case attr == "state":
		return []any{"available", "active"}
	// Un attribut d'IDENTITÉ sert souvent de `subject` au finding, et le modèle
	// partagé exige une chaîne : lui donner un booléen fait échouer le moteur
	// lui-même, pas la règle. On ne sonde donc que des chaînes.
	case strings.HasSuffix(attr, "_id"), attr == "id", strings.Contains(attr, "name"):
		return []any{"probe-1"}
	default:
		return []any{true, false, "x", []any{"x"}}
	}
}

// reDefaulted capte `object.get(<quoi que ce soit>, "attr", <défaut>)` : un attribut
// lu AVEC une valeur de repli. C'est exactement la forme du défaut que l'ADR-0014
// interdit — l'attribut manque, le repli prend sa place, et ce repli produit un
// écart que personne n'a observé.
//
// Dériver du Rego plutôt que de `requiredAttr` était nécessaire : le défaut du
// peering vivait dans `state`, que ce contrôle ne déclarait PAS comme décisif. Une
// porte pilotée par la déclaration serait passée à côté — vérifié en réintroduisant
// le défaut, qu'elle n'attrapait pas.
var reDefaulted = regexp.MustCompile(`object\.get\([^,]+,\s*"([a-z_]+)"\s*,`)

// attrsReadByRules rend, par type de ressource, les attributs qu'une règle lit avec
// un repli, et les codes que ce fichier émet.
func attrsReadByRules(t *testing.T) map[string]map[string][]string {
	t.Helper()
	dir := filepath.Join("rules")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("lecture des règles : %v", err)
	}
	reType := regexp.MustCompile(`resources_of_type\("([a-z_]+)"\)`)
	reCode := regexp.MustCompile(`"code":\s*"([a-z0-9_]+)"`)

	out := map[string]map[string][]string{} // code -> type -> attributs
	for _, e := range entries {
		n := e.Name()
		if !strings.HasSuffix(n, ".rego") || strings.HasSuffix(n, "_test.rego") || n == "lib.rego" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, n)) //nolint:gosec // chemin du dépôt
		if err != nil {
			t.Fatalf("lecture de %s : %v", n, err)
		}
		text := string(b)
		types := reType.FindAllStringSubmatch(text, -1)
		if len(types) == 0 {
			continue
		}
		seen := map[string]bool{}
		var attrs []string
		for _, m := range reDefaulted.FindAllStringSubmatch(text, -1) {
			if seen[m[1]] {
				continue
			}
			seen[m[1]] = true
			attrs = append(attrs, m[1])
		}
		if len(attrs) == 0 {
			continue
		}
		sort.Strings(attrs)
		for _, c := range reCode.FindAllStringSubmatch(text, -1) {
			if out[c[1]] == nil {
				out[c[1]] = map[string][]string{}
			}
			out[c[1]][types[0][1]] = attrs
		}
	}
	return out
}

func findingCodes(t *testing.T, ctx context.Context, inv map[string]any) map[string]bool {
	t.Helper()
	raw, err := engineEvaluate(ctx, inv)
	if err != nil {
		t.Fatalf("évaluation : %v", err)
	}
	out := map[string]bool{}
	for _, f := range raw {
		out[f.Code] = true
	}
	return out
}

// syntheticExceptions — les couples (contrôle, attribut) dont le repli est
// DÉLIBÉRÉ et défendable, avec sa raison. Chacun se relit comme une décision.
//
// Le critère qui les sépare d'un défaut : vers quoi le repli fait-il pencher ?
// Le défaut interdit par l'ADR-0014 fait conclure à un ÉCART sur une donnée
// jamais observée. Un repli qui fait au contraire ENTRER une ressource dans le
// périmètre d'un contrôle est l'inverse : il produit au pire un faux positif
// visible et corrigeable, là où le repli inverse produirait un faux vert.
var syntheticExceptions = map[string]string{
	// Scaleway expose ExpiresAt comme un *time.Time (providers/scaleway.yaml,
	// contrat access_key, etat: verifie). Un nil EST la façon dont l'API dit
	// « aucune expiration » : l'absence est ici l'observation. Ambiguïté restante
	// suivie en #121.
	"iam_accesskey_expiration_set\x00expiration_date": "contrat Scaleway : ExpiresAt est un *time.Time, un nil signifie « aucune expiration »",

	// `editable` absent ⇒ le rôle est traité comme éditable, donc DANS le périmètre.
	// Le repli inverse exclurait silencieusement tout rôle dont le fournisseur
	// n'expose pas la capacité — un faux vert. Quatre règles partagent ce repli, et
	// il penche du bon côté.
	"iam_role_key_lifetime_bounded\x00editable":      "repli vers l'inclusion : ne pas savoir si un rôle est prédéfini ne doit pas l'exclure du contrôle",
	"iam_role_no_admin_privileges\x00editable":       "idem",
	"iam_role_source_ip_restricted\x00editable":      "idem",
	"iam_policy_no_privilege_escalation\x00editable": "idem",
}

// TestNoRuleCreatesAFindingWhenItsDecidingAttributeDisappears est la porte générique
// de l'issue #104.
//
// Pour chaque contrôle qui déclare un attribut décisif, on construit une ressource du
// type qu'il lit, on la peuple, puis on retire l'attribut. Un écart qui APPARAÎT à ce
// moment-là est un écart déduit d'une absence — exactement ce que l'ADR-0014 interdit,
// et ce que trois règles de ce jalon faisaient.
func TestNoRuleCreatesAFindingWhenItsDecidingAttributeDisappears(t *testing.T) {
	ctx := context.Background()
	byCode := attrsReadByRules(t)
	if len(byCode) == 0 {
		t.Fatal("aucun attribut à repli trouvé : la porte ne mesurerait rien")
	}

	codes := make([]string, 0, len(byCode))
	for c := range byCode {
		codes = append(codes, c)
	}
	sort.Strings(codes)

	var covered int
	for _, code := range codes {
		for typ, attrs := range byCode[code] {
			for _, attr := range attrs {
				for _, v := range probeValues(attr) {
					// Seul l'attribut CIBLÉ prend la valeur de sonde. Les autres
					// gardent une chaîne neutre : les peupler tous avec un booléen
					// produisait des findings dont le `subject` était un booléen,
					// puisque certains de ces attributs SONT le sujet (vm_id, name).
					base := map[string]any{}
					for _, a := range attrs {
						base[a] = "x"
					}
					base[attr] = v
					inv := map[string]any{"resources": []any{map[string]any{
						"provider": "outscale", "type": typ, "id": "probe-1",
						"name": "probe-1", "region": "eu-west-2", "attributes": base,
					}}}

					before := findingCodes(t, ctx, inv)
					if before[code] {
						continue // la règle se déclenche déjà : cette base ne mesure rien
					}

					ablated := deepCopyAny(t, inv)
					rs := ablated["resources"].([]any)
					at := rs[0].(map[string]any)["attributes"].(map[string]any)
					delete(at, attr)

					covered++
					if _, allowed := syntheticExceptions[code+"\x00"+attr]; allowed {
						continue
					}
					if findingCodes(t, ctx, ablated)[code] {
						t.Errorf("%s : retirer l'attribut %q fait APPARAÎTRE l'écart.\n"+
							"  La règle a défaussé un attribut absent sur une valeur, et cette valeur a\n"+
							"  produit un écart que rien n'a observé (ADR-0014). Rendre le repli neutre,\n"+
							"  ou déclarer l'incertitude (ADR-0015) si la règle sait qu'elle ne sait pas.",
							code, attr)
					}
				}
			}
		}
	}

	if covered == 0 {
		t.Fatal("aucune ablation synthétique effectuée : la porte ne mesure rien")
	}
	t.Logf("%d ablation(s) synthétique(s) sur %d contrôle(s), dérivées du Rego", covered, len(codes))
}

func deepCopyAny(t *testing.T, m map[string]any) map[string]any {
	t.Helper()
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("clonage : %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("clonage : %v", err)
	}
	return out
}

type probeFinding struct {
	Code string `json:"code"`
}

func engineEvaluate(ctx context.Context, inv map[string]any) ([]probeFinding, error) {
	raw, err := engine.Evaluate(ctx, inv, commonrules.FS())
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var out []probeFinding
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

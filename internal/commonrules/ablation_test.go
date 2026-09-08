package commonrules_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stephrobert/scankit/engine"

	"github.com/stephrobert/pepin/internal/commonrules"
)

// L'invariant, en une phrase : RETIRER un attribut ne doit jamais faire APPARAÎTRE
// un écart.
//
// Si moins d'information produit plus de findings, c'est que la règle a déduit
// quelque chose de l'absence — elle a défaussé un attribut manquant sur une valeur
// qui se trouve être la mauvaise. C'est le défaut que l'ADR-0014 interdit, et il
// s'était logé dans deux règles sans qu'aucun test ne le voie : `state` absent
// présumé « actif », `is_enabled` absent présumé faux.
//
// Ce que cette porte a de particulier : elle n'exige d'aucune règle qu'elle déclare
// son attribut décisif. Elle ne lit pas le code, elle mesure le comportement. Un
// contrôle écrit demain est couvert sans avoir rien à déclarer.
//
// Ce qu'elle NE voit pas, et qui est écrit pour qu'on ne s'y trompe pas :
//
//   - Le cas où la donnée est PRÉSENTE et inexploitable — une région renseignée mais
//     hors des tables de classification. Rien n'est retiré, donc rien n'apparaît.
//     C'est le domaine de l'ADR-0015, pas d'ici.
//   - Un défaut qui ne se manifeste que sur une combinaison d'attributs retirés :
//     on n'ablate qu'un attribut à la fois, parce que le produit cartésien serait
//     ingérable et que les défauts observés sont tous unitaires.
//   - Une règle qui n'est déclenchée par aucune fixture : on ne mesure que ce que le
//     corpus atteint. C'est la même limite que la dette de véracité, et elle se
//     rembourse par le même chemin — plus de corpus.
//
// Voir docs/adr/0014-jamais-fabriquer-une-donnee-absente.md et l'issue #104.

// ablationExceptions énumère les couples (contrôle, attribut) pour lesquels
// l'apparition d'un écart à l'ablation est LÉGITIME, avec sa raison.
//
// Une exception se justifie quand l'absence de l'attribut EST la non-conformité, et
// que le contrôle sait par ailleurs distinguer « non collecté » de « absent chez le
// fournisseur » — sans quoi elle réintroduit exactement le défaut qu'on traque.
//
// Toute entrée porte sa raison et se relit comme une décision, pas comme un
// contournement. La porte a d'abord signalé ce cas ; c'est le CONTRAT qui a
// tranché, pas le confort.
var ablationExceptions = map[string]string{
	// Scaleway expose `ExpiresAt (*time.Time)` — un POINTEUR (providers/scaleway.yaml,
	// contrat `access_key`, etat: verifie). Un pointeur nul est la façon dont l'API
	// dit « aucune expiration ». Ici, l'absence de l'attribut EST l'observation, et
	// exiger sa présence rendrait le contrôle aveugle au cas qu'il existe pour voir.
	//
	// La limite subsiste et elle est réelle : l'absence conflate « pointeur nul
	// observé » et « attribut jamais collecté ». Les distinguer demande de lire la
	// PROVENANCE, qui indexe les attributs CHERCHÉS même non exposés (ADR-0007) —
	// une règle n'y a pas accès aujourd'hui. Suivi en #121.
	"iam_accesskey_expiration_set\x00expiration_date": "contrat Scaleway : ExpiresAt est un *time.Time, un nil signifie « aucune expiration »",

	// Une snapshot sans `volume_id` n'est attribuable à AUCUN volume : le lien est
	// ce qui la fait compter. Son absence n'est donc pas un repli défavorable mais
	// une donnée structurellement inexploitable, et la RÈGLE ne peut rien y faire —
	// elle ne voit pas la différence entre « aucune snapshot ne correspond » et
	// « le lien n'a pas été collecté ».
	//
	// Cette exception RESTE donc, mais elle ne consigne plus une dette : #133 a
	// traité le problème là où il se traite, un cran plus haut. Le contrat de
	// décision déclare désormais `volume_id` décisif SUR LE TYPE snapshot, et
	// l'assessment requalifie les écarts en `not-evaluated` quand ce lien manque —
	// le faux positif de masse (tous les volumes « non sauvegardés » alors que rien
	// ne l'a été observé) ne peut plus sortir. Ce que cette ligne dit aujourd'hui,
	// c'est seulement que la porte d'ablation mesure la règle, pas la chaîne.
	"blockstorage_volume_snapshots_exist\x00volume_id": "le lien vers le volume est structurel : sans lui la snapshot n'est attribuable à rien ; le faux positif de masse est traité au verrou de capacité, par type (#133)",
}

type finding struct {
	Code    string `json:"code"`
	Subject string `json:"subject"`
}

// On compare les CODES, pas les couples (code, sujet). Retirer un attribut
// d'identité — un `policy_name`, un `name` — déplace le sujet d'un écart qui
// existait déjà, et le couple paraîtrait neuf. Ce bruit noierait le signal. La
// contrepartie est assumée : un même code qui se met à toucher un AUTRE sujet
// passe sous le radar. La famille de défauts traquée ici est « un code apparaît »,
// et elle est intégralement couverte.
func key(f finding) string { return f.Code }

// corpus charge les inventaires JSON du dépôt. Les plans Terraform sont exclus :
// ils passent par le mapper, et l'ablation d'un attribut normalisé n'y a pas de
// sens direct.
func corpus(t *testing.T) map[string]map[string]any {
	t.Helper()
	root := filepath.Join("..", "..")
	out := map[string]map[string]any{}
	// Le corpus DÉDIÉ (testdata/corpus) complète les fixtures d'exemple. Il vit ici et
	// non dans examples/ parce qu'il ne sert qu'à cette porte : le mettre dans
	// examples/ ferait dériver la documentation générée à chaque ajout, ce qui
	// dissuaderait d'en ajouter — exactement l'effet inverse de celui recherché.
	//
	// Chaque tenant y est CONFORME : l'ablation part d'une base silencieuse, sans quoi
	// l'écart y serait déjà présent et son apparition ne se verrait pas.
	groups := [][]string{{filepath.Join("testdata", "corpus", "*.json")}}
	for _, prov := range []string{"scaleway", "outscale", "exoscale"} {
		groups = append(groups, []string{
			filepath.Join(root, "examples", prov, "*.json"),
			filepath.Join(root, "references", "tenants", prov, "*", "plan.json"),
		})
	}
	for _, patterns := range groups {
		var all []string
		for _, pat := range patterns {
			m, _ := filepath.Glob(pat)
			all = append(all, m...)
		}
		for _, p := range all {
			b, err := os.ReadFile(p) //nolint:gosec // chemins du dépôt, énumérés ici
			if err != nil {
				continue
			}
			var inv map[string]any
			if json.Unmarshal(b, &inv) != nil {
				continue
			}
			if _, ok := inv["resources"].([]any); !ok {
				continue
			}
			out[strings.TrimPrefix(p, root+string(filepath.Separator))] = inv
		}
	}
	if len(out) == 0 {
		t.Fatal("corpus vide : la porte ne mesurerait rien")
	}
	return out
}

func evaluate(t *testing.T, ctx context.Context, inv map[string]any) map[string]bool {
	t.Helper()
	raw, err := engine.Evaluate(ctx, inv, commonrules.FS())
	if err != nil {
		t.Fatalf("évaluation : %v", err)
	}
	b, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("sérialisation des findings : %v", err)
	}
	var fs []finding
	if err := json.Unmarshal(b, &fs); err != nil {
		t.Fatalf("relecture des findings : %v", err)
	}
	out := map[string]bool{}
	for _, f := range fs {
		out[key(f)] = true
	}
	return out
}

// deepCopy clone l'inventaire par sérialisation : l'ablation ne doit jamais altérer
// le corpus partagé entre les sous-tests.
func deepCopy(t *testing.T, inv map[string]any) map[string]any {
	t.Helper()
	b, err := json.Marshal(inv)
	if err != nil {
		t.Fatalf("clonage : %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("clonage : %v", err)
	}
	return out
}

// TestRemovingAnAttributeNeverCreatesAFinding est LA porte de l'ADR-0014.
//
// Pour chaque ressource du corpus et chaque attribut qu'elle porte, on retire
// l'attribut, on réévalue, et on exige qu'aucun écart NOUVEAU n'apparaisse. Un écart
// qui DISPARAÎT est normal : on vient de retirer la donnée qui l'établissait.
func TestRemovingAnAttributeNeverCreatesAFinding(t *testing.T) {
	ctx := context.Background()
	var checked int
	seenTypes := map[string]bool{}

	for name, inv := range corpus(t) {
		base := evaluate(t, ctx, inv)
		resources, _ := inv["resources"].([]any)

		for i := range resources {
			res, ok := resources[i].(map[string]any)
			if !ok {
				continue
			}
			attrs, ok := res["attributes"].(map[string]any)
			if !ok {
				continue // un plan Terraform porte `values`, pas `attributes` : rien à ablater
			}
			if ty, ok := res["type"].(string); ok {
				seenTypes[ty] = true
			}
			names := make([]string, 0, len(attrs))
			for a := range attrs {
				names = append(names, a)
			}
			sort.Strings(names)

			for _, attr := range names {
				ablated := deepCopy(t, inv)
				rs, _ := ablated["resources"].([]any)
				r, _ := rs[i].(map[string]any)
				at, _ := r["attributes"].(map[string]any)
				delete(at, attr)

				after := evaluate(t, ctx, ablated)
				checked++

				for k := range after {
					if base[k] {
						continue // l'écart existait déjà : rien de nouveau
					}
					code := k
					if why, allowed := ablationExceptions[code+"\x00"+attr]; allowed {
						t.Logf("exception admise — %s sur l'absence de %q : %s", code, attr, why)
						continue
					}
					t.Errorf(
						"%s : retirer l'attribut %q de la ressource %d fait APPARAÎTRE l'écart %s.\n"+
							"  Un attribut absent a été défaussé sur une valeur, et cette valeur a produit un écart\n"+
							"  que rien n'a observé (ADR-0014). Rendre le défaut neutre, ou déclarer l'incertitude\n"+
							"  (ADR-0015) si la règle sait qu'elle ne sait pas.",
						name, attr, i, code)
				}
			}
		}
	}

	if checked == 0 {
		t.Fatal("aucune ablation effectuée : la porte ne mesure rien")
	}

	// Le chiffre qui compte n'est pas le nombre d'ablations, c'est ce que le corpus
	// ATTEINT. Une règle dont aucun type n'est présent n'est pas éprouvée, et son
	// silence ressemble à s'y méprendre à un succès.
	//
	// Mesuré à l'écriture : le corpus JSON porte 6 types de ressources. Le défaut du
	// peering corrigé dans ce même lot n'aurait PAS été attrapé ici, faute d'une
	// seule ressource `network_peering` dans une fixture. Le dire est le minimum ;
	// l'issue #123 étend le corpus.
	t.Logf("%d ablations sur %d type(s) de ressource — la couverture borne ce que cette porte peut voir",
		checked, len(seenTypes))
	for _, ty := range sortedKeys(seenTypes) {
		t.Logf("  type éprouvé : %s", ty)
	}
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

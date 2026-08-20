package commonrules_test

// Le profil par défaut existe DEUX FOIS : en Go (internal/policy, ce que le scan
// injecte dans `input.config`) et en Rego (lib.rego, le repli quand aucune
// configuration n'est présente — `opa test`, une règle externe, ou l'input.json
// d'une version antérieure).
//
// Deux définitions d'une même chose finissent par diverger, et cette divergence-là
// serait invisible : les règles rendraient un verdict sous un profil, la
// documentation et les contraintes normatives en décriraient un autre. Ce test
// mesure l'égalité au lieu de l'affirmer — il évalue les MÊMES inventaires par les
// DEUX chemins et exige des findings identiques.

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stephrobert/scankit/engine"

	"github.com/stephrobert/pepin/internal/commonrules"
	"github.com/stephrobert/pepin/internal/policy"
)

// Des inventaires qui exercent les QUATRE contrôles réglables. Les fixtures du
// dépôt ne portent ni réseau ni volume : s'en contenter ferait passer le test sur
// la moitié du sujet.
func inventories() map[string]map[string]any {
	res := func(rs ...map[string]any) map[string]any {
		return map[string]any{"evaluated_at": "2026-07-19T12:00:00Z", "resources": rs}
	}
	tag := func(k, v string) map[string]any { return map[string]any{"key": k, "value": v} }
	return map[string]map[string]any{
		"réseau sans étiquette": res(map[string]any{
			"provider": "outscale", "type": "network", "id": "n1",
			"attributes": map[string]any{"name": "prod", "tags": []any{}},
		}),
		"réseau à étiquette hors sujet": res(map[string]any{
			"provider": "outscale", "type": "network", "id": "n1",
			"attributes": map[string]any{"tags": []any{tag("foo", "bar")}},
		}),
		"réseau documenté par alias": res(map[string]any{
			"provider": "outscale", "type": "network", "id": "n1",
			"attributes": map[string]any{"tags": []any{tag("Team", "sre"), tag("Project", "p"), tag("Env", "prod")}},
		}),
		"ressource facturable sans étiquette": res(map[string]any{
			"provider": "scaleway", "type": "compute_instance", "id": "i1", "name": "web",
			"attributes": map[string]any{"tags": []any{}},
		}),
		"ressource facturable étiquetée à l'ancienne": res(map[string]any{
			"provider": "scaleway", "type": "compute_instance", "id": "i1", "name": "web",
			"attributes": map[string]any{"tags": []any{
				tag("CostCenter", "cc"), tag("Project", "p"), tag("Env", "prod"), tag("Owner", "sre"),
			}},
		}),
		"base managée sans étiquette": res(map[string]any{
			"provider": "scaleway", "type": "managed_database", "id": "db1",
			"attributes": map[string]any{"tags": []any{}},
		}),
		"volume sans snapshot": res(map[string]any{
			"provider": "outscale", "type": "blockstorage_volume", "id": "vol-1",
			"attributes": map[string]any{"volume_id": "vol-1", "state": "in-use"},
		}),
		"volume avec snapshot terminée": res(
			map[string]any{
				"provider": "outscale", "type": "blockstorage_volume", "id": "vol-1",
				"attributes": map[string]any{"volume_id": "vol-1", "state": "in-use"},
			},
			map[string]any{
				"provider": "outscale", "type": "blockstorage_snapshot", "id": "snap-1",
				"attributes": map[string]any{"volume_id": "vol-1", "creation_date": "2026-07-18T12:00:00Z", "state": "completed"},
			},
		),
		"volume avec snapshot en erreur": res(
			map[string]any{
				"provider": "outscale", "type": "blockstorage_volume", "id": "vol-1",
				"attributes": map[string]any{"volume_id": "vol-1", "state": "in-use"},
			},
			map[string]any{
				"provider": "outscale", "type": "blockstorage_snapshot", "id": "snap-1",
				"attributes": map[string]any{"volume_id": "vol-1", "creation_date": "2026-07-18T12:00:00Z", "state": "error"},
			},
		),
		"user-data à secret confirmé": res(map[string]any{
			"provider": "exoscale", "type": "compute_instance", "id": "i1",
			"attributes": map[string]any{"vm_id": "i1", "user_data": "-----BEGIN RSA PRIVATE KEY-----\nZZZZ\n-----END RSA PRIVATE KEY-----"},
		}),
		"user-data à heuristique générique": res(map[string]any{
			"provider": "exoscale", "type": "compute_instance", "id": "i1",
			"attributes": map[string]any{"vm_id": "i1", "user_data": "#cloud-config\npassword=hunter2000plus\n"},
		}),
	}
}

// TestTheRegoFallbackProfileEqualsTheGoDefaultProfile : sur chaque inventaire, le
// jeu de findings est le MÊME avec la configuration Go injectée et sans aucune
// configuration. C'est la mesure de l'égalité des deux profils par défaut, et
// c'est aussi ce qui garantit que l'injection de `input.config` n'a, par
// elle-même, déplacé aucun verdict.
func TestTheRegoFallbackProfileEqualsTheGoDefaultProfile(t *testing.T) {
	ctx := context.Background()
	for name, inv := range inventories() {
		t.Run(name, func(t *testing.T) {
			withCfg := map[string]any{}
			for k, v := range inv {
				withCfg[k] = v
			}
			withCfg["config"] = policy.Defaults()

			bare, err := engine.Evaluate(ctx, inv, commonrules.FS())
			if err != nil {
				t.Fatalf("évaluation sans configuration : %v", err)
			}
			injected, err := engine.Evaluate(ctx, withCfg, commonrules.FS())
			if err != nil {
				t.Fatalf("évaluation avec la configuration par défaut : %v", err)
			}
			if a, b := mustJSON(t, bare), mustJSON(t, injected); a != b {
				t.Errorf("le repli Rego et le profil Go par défaut divergent.\n  sans config : %s\n  avec config : %s", a, b)
			}
		})
	}
}

// TestTheInjectedConfigurationIsActuallyRead est l'autre moitié : si les règles
// ignoraient `input.config`, le test ci-dessus serait vert pour la pire des
// raisons — les deux chemins seraient le même. Une configuration DIFFÉRENTE doit
// donc produire un résultat différent.
func TestTheInjectedConfigurationIsActuallyRead(t *testing.T) {
	ctx := context.Background()
	inv := inventories()["user-data à heuristique générique"]
	strict := map[string]any{}
	for k, v := range inv {
		strict[k] = v
	}
	strict["config"] = policy.Resolve(&policy.Controls{Secrets: &policy.Secrets{MinConfidence: "high"}})

	bare, err := engine.Evaluate(ctx, inv, commonrules.FS())
	if err != nil {
		t.Fatalf("évaluation sans configuration : %v", err)
	}
	raised, err := engine.Evaluate(ctx, strict, commonrules.FS())
	if err != nil {
		t.Fatalf("évaluation à seuil relevé : %v", err)
	}
	if len(bare) == 0 {
		t.Fatal("l'inventaire témoin ne produit aucun finding : le test ne mesure rien")
	}
	if len(raised) >= len(bare) {
		t.Errorf("un seuil de confiance relevé n'a rien changé : les règles ne lisent pas `input.config` (%d → %d findings)", len(bare), len(raised))
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("sérialisation : %v", err)
	}
	return string(b)
}

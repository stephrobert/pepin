package docgen

// Les régions générées de la page « plan Terraform contre scan live ».
//
// La divergence n'y est pas racontée, elle est MONTRÉE : l'extrait du plan committé qui
// marque un attribut « unknown after apply », le statut que le scan en tire, et le statut que
// le même contrôle prend quand la même donnée est présente. C'est le faux positif corrigé en
// v0.1.1 (une VM rattachée à un groupe de sécurité créé par le même plan, signalée « sans
// groupe de sécurité »), et il se rejoue depuis les fixtures du dépôt.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// L'inventaire qui tient le rôle de la source live pour la démonstration : la MÊME instance
// que le plan, telle que le collecteur la normalise depuis `api/instance/v1` Server
// (`vm_id` ← Server.ID, `security_group_ids` ← [SecurityGroup.ID], `tags` ← Server.Tags,
// mapping de providers/scaleway.yaml). Aucun compte Scaleway n'a été appelé pour l'écrire :
// c'est un inventaire d'exemple, et la page le dit.
const driftInventory = `{
  "provider": "scaleway",
  "resources": [
    {
      "provider": "scaleway",
      "type": "compute_instance",
      "id": "3f2b1c00-0000-4a00-9000-000000000001",
      "name": "web",
      "region": "fr-par",
      "attributes": {
        "vm_id": "3f2b1c00-0000-4a00-9000-000000000001",
        "state": "running",
        "security_group_ids": ["b1a2c3d4-0000-4a00-9000-000000000002"],
        "tags": [{"key": "env", "value": "prod"}]
      }
    }
  ]
}`

// driftControl est le contrôle dont le statut change avec la source. Sa règle porte la garde
// de capacité posée en v0.1.1 (`"security_group_ids" in object.keys(vm.attributes)`).
const driftControl = "compute_instance_has_security_group"

// driftBlocks assemble les régions de la page de comparaison.
func driftBlocks(lang string, c captures) map[string]string {
	t := driftText(lang)
	return map[string]string{
		"drift-plan-unknown":      Fence("json", c.planUnknown),
		"drift-plan-status":       assessmentControl(t, c.assessment, driftControl),
		"drift-live-fixture":      Fence("json", driftInventory),
		"drift-live-status":       assessmentControl(t, c.driftLive, driftControl),
		"drift-bool-plan":         Fence("json", c.planBoolAsString),
		"drift-bool-finding":      findingsWithCode(t, c.outscalePlanJSON, "CLD-STO-2"),
		"drift-source-provenance": Fence("json", sourceProvenance(c.assessment)+"\n"+sourceProvenance(c.driftLive)),
	}
}

// planExcerpt rend un extrait RÉEL d'un plan committé : l'entrée `resource_changes` d'une
// ressource, réduite aux clés demandées. Rien n'est reformulé ; ce qui est retiré l'est par
// sélection de clés, et la page dit qu'il s'agit d'un extrait.
func planExcerpt(root, plan, address string, keys []string) (string, error) {
	raw, err := os.ReadFile(filepath.Join(root, plan)) // #nosec G304 -- fixture du dépôt.
	if err != nil {
		return "", fmt.Errorf("lecture du plan %s : %w", plan, err)
	}
	var doc struct {
		ResourceChanges []struct {
			Address string `json:"address"`
			Type    string `json:"type"`
			Change  struct {
				After        map[string]any `json:"after"`
				AfterUnknown map[string]any `json:"after_unknown"`
			} `json:"change"`
		} `json:"resource_changes"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return "", fmt.Errorf("plan %s illisible : %w", plan, err)
	}
	for _, rc := range doc.ResourceChanges {
		if rc.Address != address {
			continue
		}
		out := map[string]any{"address": rc.Address, "type": rc.Type}
		after := map[string]any{}
		unknown := map[string]any{}
		for _, k := range keys {
			if v, ok := rc.Change.After[k]; ok {
				after[k] = v
			}
			if v, ok := rc.Change.AfterUnknown[k]; ok {
				unknown[k] = v
			}
		}
		out["change"] = map[string]any{"after": after, "after_unknown": unknown}
		return mustIndent(out), nil
	}
	return "", fmt.Errorf("ressource %q absente de %s : l'extrait ne montrerait rien", address, plan)
}

// assessmentControl extrait d'une sortie `--format assessment` le résultat d'UN contrôle.
func assessmentControl(t driftStrings, c Capture, control string) string {
	var doc map[string]any
	if err := json.Unmarshal([]byte(c.Stdout), &doc); err != nil {
		return Fence("text", strings.ReplaceAll(t.noResult, "%s", control))
	}
	results, _ := doc["results"].([]any)
	for _, it := range results {
		r, ok := it.(map[string]any)
		if !ok {
			continue
		}
		if ctl, _ := r["control"].(string); ctl == control {
			return Fence("json", mustIndent(r))
		}
	}
	return Fence("text", strings.ReplaceAll(t.noResult, "%s", control))
}

// findingsWithCode rend les findings d'un code donné, tels que `--format json` les a produits.
func findingsWithCode(t driftStrings, c Capture, code string) string {
	var doc struct {
		Findings []map[string]any `json:"findings"`
	}
	if err := json.Unmarshal([]byte(c.Stdout), &doc); err != nil {
		return Fence("text", strings.ReplaceAll(t.noFinding, "%s", code))
	}
	var picked []map[string]any
	for _, f := range doc.Findings {
		if got, _ := f["code"].(string); got == code {
			picked = append(picked, f)
		}
	}
	if len(picked) == 0 {
		return Fence("text", strings.ReplaceAll(t.noFinding, "%s", code))
	}
	sort.Slice(picked, func(i, j int) bool {
		a, _ := picked[i]["subject"].(string)
		b, _ := picked[j]["subject"].(string)
		return a < b
	})
	return Fence("json", mustIndent(picked[0]))
}

// sourceProvenance isole le champ que l'assessment porte pour dire d'où venait la donnée.
// C'est ce qui distingue deux rapports du même tenant, et c'est scellé dans le bundle.
func sourceProvenance(c Capture) string {
	var doc map[string]any
	if err := json.Unmarshal([]byte(c.Stdout), &doc); err != nil {
		return `{"run": {"source": "?"}}`
	}
	run, _ := doc["run"].(map[string]any)
	src, _ := run["source"].(string)
	return `{"run": {"source": "` + src + `"}}`
}

// driftStrings porte les libellés des régions de la page de comparaison.
type driftStrings struct {
	noResult, noFinding string
}

func driftText(lang string) driftStrings {
	if lang == "fr" {
		return driftStrings{
			noResult:  "(aucun résultat pour le contrôle %s sur ce scan)",
			noFinding: "(aucun écart %s sur ce scan)",
		}
	}
	return driftStrings{
		noResult:  "(no result for control %s in this scan)",
		noFinding: "(no %s deviation in this scan)",
	}
}

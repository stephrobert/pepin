package tfmap

import (
	"encoding/json"
	"os/exec"
	"strings"
)

// tfBlock reflète un bloc de schéma Terraform (attributs + blocs imbriqués).
type tfBlock struct {
	Attributes map[string]json.RawMessage `json:"attributes"`
	BlockTypes map[string]struct {
		Block tfBlock `json:"block"`
	} `json:"block_types"`
}

// CheckSchema ANCRE une spec de mapping sur le code réel du provider Terraform :
// il exécute `terraform providers schema -json` dans dir et vérifie que chaque
// attribut source référencé par la spec existe dans le schéma. Pour une spec avec
// `items` (bloc répété éclaté), les chemins sont validés contre le schéma DU BLOC
// (ex. inbound_rule.port), `_parent.*` étant le conteneur. Retourne les écarts et
// ok=false si terraform/le schéma sont indisponibles. Mutualisé pour tous les providers.
func CheckSchema(dir string, spec Spec) (missing []string, ok bool) {
	tf, err := exec.LookPath("terraform")
	if err != nil {
		return nil, false
	}
	// #nosec G204 -- binaire résolu par LookPath, arguments constants ; lecture du schéma uniquement.
	cmd := exec.Command(tf, "providers", "schema", "-json")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return nil, false
	}
	var doc struct {
		ProviderSchemas map[string]struct {
			ResourceSchemas map[string]struct {
				Block tfBlock `json:"block"`
			} `json:"resource_schemas"`
		} `json:"provider_schemas"`
	}
	if json.Unmarshal(out, &doc) != nil {
		return nil, false
	}

	blocks := map[string]tfBlock{} // tf_type -> bloc racine
	for _, ps := range doc.ProviderSchemas {
		for name, rs := range ps.ResourceSchemas {
			blocks[name] = rs.Block
		}
	}

	for _, rs := range spec.Resources {
		root, found := blocks[rs.TFType]
		if !found {
			missing = append(missing, rs.TFType+" : type absent du schéma du provider")
			continue
		}
		// Schéma de référence pour les attributs du `map` : le bloc éclaté si `items`.
		ref := root
		if rs.Items != "" {
			blockName := rootField(rs.Items)
			sub, ok := root.BlockTypes[blockName]
			if !ok {
				missing = append(missing, rs.TFType+"."+blockName+" : bloc absent du schéma")
				continue
			}
			ref = sub.Block
		}
		for _, p := range mapPaths(rs) {
			r := rootField(p)
			if r == "" || r == "_parent" { // conteneur injecté à l'éclatement
				continue
			}
			if !has(ref, r) && !has(root, r) {
				missing = append(missing, rs.TFType+"."+r+" : attribut absent du schéma (invention ou dérive ?)")
			}
		}
	}
	return missing, true
}

// mapPaths retourne tous les chemins source d'une spec (valeurs du map éclatées
// sur la coalescence ||, plus region).
func mapPaths(rs ResourceSpec) []string {
	var out []string
	for _, p := range rs.Map {
		out = append(out, strings.Split(p, "||")...)
	}
	if rs.Region != "" {
		out = append(out, rs.Region)
	}
	return out
}

func has(b tfBlock, field string) bool {
	if _, ok := b.Attributes[field]; ok {
		return true
	}
	_, ok := b.BlockTypes[field]
	return ok
}

// rootField extrait le premier segment d'un chemin (avant ., [, ou ||).
func rootField(path string) string {
	for _, sep := range []string{"||", ".", "["} {
		if i := strings.Index(path, sep); i >= 0 {
			path = path[:i]
		}
	}
	return strings.TrimSpace(path)
}

package tfmap

import (
	"encoding/json"
	"os/exec"
	"sort"
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
		missing = append(missing, defautsInterdits(rs, ref, root)...)
	}
	return missing, true
}

// drapeaux d'un attribut, tels que le schéma du provider les publie.
type drapeaux struct {
	Required bool `json:"required"`
	Optional bool `json:"optional"`
	Computed bool `json:"computed"`
}

// defautsInterdits refuse une déclaration `default:` que le schéma ne permet pas.
//
// Un `default:X` dit « cet argument, laissé vide par l'auteur, vaut X ». Cette
// phrase n'a de sens que pour un argument dont le plan ÉCRIT le silence, c'est-à-dire
// un `optional` qui n'est pas `computed` (ADR-0024).
//
//	computed  — le provider peut calculer la valeur à l'apply, donc Terraform la
//	            range dans `after_unknown` et elle est ABSENTE de `planned_values`.
//	            Le plan ne sait pas ; déclarer ce qu'elle vaut serait fabriquer une
//	            donnée absente, ce qu'ADR-0014 interdit. En pratique la déclaration
//	            ne tirerait jamais — mais une déclaration inerte est pire qu'absente,
//	            parce qu'elle fait CROIRE que le contrôle conclut.
//	required  — l'argument est toujours écrit, donc le défaut ne tire jamais. C'est
//	            un no-op trompeur, et il en existait un : `inbound_rule.action` chez
//	            Scaleway, déclaré `default:accept` alors que le schéma l'exige.
func defautsInterdits(rs ResourceSpec, ref, root tfBlock) []string {
	var out []string
	for attr, t := range rs.Transforms {
		if !porteUnDefaut(t) {
			continue
		}
		chemin, ok := rs.Map[attr]
		if !ok {
			continue
		}
		for _, p := range strings.Split(chemin, "||") {
			champ := rootField(strings.TrimSpace(p))
			if champ == "" || champ == "_parent" {
				continue
			}
			d, trouve := drapeauxDe(ref, champ)
			if !trouve {
				d, trouve = drapeauxDe(root, champ)
			}
			if !trouve {
				continue // déjà signalé par la vérification d'existence ci-dessus.
			}
			switch {
			case d.Computed:
				out = append(out, rs.TFType+"."+champ+" : `default:` sur un attribut COMPUTED — "+
					"son silence part en after_unknown, donc le plan ne le connaît pas (ADR-0024)")
			case d.Required:
				out = append(out, rs.TFType+"."+champ+" : `default:` sur un attribut REQUIRED — "+
					"il est toujours écrit, le défaut ne tire jamais (no-op trompeur)")
			}
		}
	}
	sort.Strings(out)
	return out
}

// porteUnDefaut dit si un transform déclaré contient un `default:`. Un transform
// est soit une chaîne, soit une LISTE appliquée dans l'ordre (`["default:x", list]`).
func porteUnDefaut(t any) bool {
	switch v := t.(type) {
	case string:
		return strings.HasPrefix(v, "default:")
	case []any:
		for _, e := range v {
			if porteUnDefaut(e) {
				return true
			}
		}
	}
	return false
}

func drapeauxDe(b tfBlock, champ string) (drapeaux, bool) {
	raw, ok := b.Attributes[champ]
	if !ok {
		return drapeaux{}, false
	}
	var d drapeaux
	if json.Unmarshal(raw, &d) != nil {
		return drapeaux{}, false
	}
	return d, true
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

package veracity

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	yaml "go.yaml.in/yaml/v3"
)

// Le FORMAT d'un scénario de véracité, et ce qu'il engage.
//
// Un fichier décrit UN chemin (contrôle × fournisseur × source) et les verdicts
// qu'il prouve. Le nom du fichier n'a aucune valeur : c'est son contenu qui dit
// ce qu'il couvre, pour qu'un scénario renommé ne puisse pas silencieusement
// changer ce qu'il prétend prouver.
//
// # Par où le scénario ENTRE dans la chaîne
//
// C'est le point qui décide de la valeur de tout ce paquet, parce que l'incident
// fondateur de cette vague est une donnée qui n'arrivait jamais jusqu'à une règle
// pourtant correcte. Trois points d'entrée, du plus fidèle au moins fidèle :
//
//   - `api:` — des réponses d'API canned, servies à la spec `collecte` RÉELLE du
//     descripteur. La chaîne éprouvée part de la réponse HTTP : collecteur,
//     pagination, jointures, projection, transforms, garde de capacité, Rego,
//     assessment. C'est le seul point d'entrée qui aurait attrapé l'incident.
//   - `plan:` — un plan Terraform minimal, passé au mapper réel du descripteur.
//     Équivalent pour la source Terraform.
//   - `inventory:` — un inventaire normalisé, qui entre à la frontière du modèle.
//     Il n'éprouve PAS le collecteur. Réservé aux unités que la spec `collecte`
//     ne produit pas (stockage objet, Kubernetes managé, politiques inline), dont
//     le collecteur est du Go et non une spec.
//
// Une porte refuse un `inventory:` là où un `api:` était possible
// (TestNoScenarioEntersLowerThanItCould) : sans elle, la voie facile
// désarmerait le contrat sans que rien ne rougisse.

// Entry nomme le point d'entrée d'un scénario dans la chaîne.
type Entry string

// Les trois points d'entrée, du plus fidèle au moins fidèle.
const (
	EntryAPI       Entry = "api"
	EntryPlan      Entry = "plan"
	EntryInventory Entry = "inventory"
)

// Scenario est un cas : une entrée, et le verdict qu'elle doit produire.
type Scenario struct {
	// Expect : le verdict attendu pour le contrôle du fichier.
	Expect Verdict `yaml:"expect"`
	// Why : ce que le cas met en scène, en une phrase. Obligatoire — un scénario
	// dont personne ne sait ce qu'il montre est un scénario que personne n'ose
	// corriger quand il rougit.
	Why string `yaml:"why"`
	// API : corps de réponse JSON, indexés par chemin d'endpoint (la valeur est
	// servie telle quelle à toute méthode).
	API map[string]string `yaml:"api"`
	// Deny : chemins d'endpoint que l'API REFUSE (403). Distinct d'un endpoint
	// simplement non servi (404) : les deux rendent l'unité incomplète, mais ils ne
	// se corrigent pas de la même façon — l'un est un droit à accorder, l'autre un
	// service absent du tenant. Un scénario qui veut éprouver le refus doit pouvoir
	// le dire, sinon il éprouve autre chose que ce qu'il annonce.
	Deny []string `yaml:"deny"`
	// Plan : un plan Terraform minimal (`planned_values`, `configuration`).
	Plan map[string]any `yaml:"plan"`
	// Inventory : un inventaire normalisé (`resources`, `collection`).
	Inventory map[string]any `yaml:"inventory"`
}

// Entry rend le point d'entrée du scénario.
func (s Scenario) Entry() Entry {
	switch {
	case len(s.API) > 0 || len(s.Deny) > 0:
		return EntryAPI
	case len(s.Plan) > 0:
		return EntryPlan
	default:
		return EntryInventory
	}
}

// File est un fichier de scénarios : un chemin, et les cas qui le prouvent.
type File struct {
	Path      string     `yaml:"-"` // chemin du fichier, pour les messages d'échec
	Control   string     `yaml:"control"`
	Provider  string     `yaml:"provider"`
	Source    string     `yaml:"source"` // live | terraform
	Scenarios []Scenario `yaml:"scenarios"`
}

// PathOf rend le chemin couvert par le fichier.
func (f File) PathOf() Path {
	return Path{Control: f.Control, Provider: f.Provider, Source: f.Source}
}

// LoadScenarios lit tous les fichiers de scénarios d'un répertoire.
func LoadScenarios(dir string) ([]File, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("lecture de %s : %w", dir, err)
	}
	var out []File
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		raw, rerr := os.ReadFile(path) // #nosec G304 -- fixture du dépôt, chemin construit depuis un répertoire de test.
		if rerr != nil {
			return nil, fmt.Errorf("lecture de %s : %w", path, rerr)
		}
		var f File
		if uerr := yaml.Unmarshal(raw, &f); uerr != nil {
			return nil, fmt.Errorf("scénario %s invalide : %w", path, uerr)
		}
		f.Path = path
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

// Covered rend, par chemin, les verdicts que les scénarios prouvent.
func Covered(files []File) map[Path][]Verdict {
	out := map[Path][]Verdict{}
	for _, f := range files {
		p := f.PathOf()
		for _, s := range f.Scenarios {
			out[p] = append(out[p], s.Expect)
		}
	}
	return out
}

// LoadLedger lit le registre de dette committé (lignes non vides, hors
// commentaires `#`).
func LoadLedger(path string) ([]string, error) {
	raw, err := os.ReadFile(path) // #nosec G304 -- fixture du dépôt.
	if err != nil {
		return nil, fmt.Errorf("lecture du registre %s : %w", path, err)
	}
	var out []string
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	return out, nil
}

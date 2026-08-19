// Package tfparse lit la sortie `terraform show -json` (un plan Terraform) et en
// extrait les ressources sous une forme générique, indépendante du provider.
//
// On consomme le plan plutôt que le HCL brut pour la FIDÉLITÉ : `planned_values`
// porte des valeurs entièrement résolues (références, variables, locals, modules,
// fonctions comme jsonencode), ce qui permet aux règles de corrélation de
// fonctionner — un audit du HCL brut les manquerait silencieusement.
//
// La projection des types Terraform (`scaleway_*`, `outscale_*`) vers le modèle
// normalisé agnostique de Pépin est faite par chaque provider (mapper dédié).
package tfparse

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/stephrobert/pepin/internal/i18n"
)

// Resource est une ressource Terraform extraite du plan : son type HCL (ex.
// "outscale_security_group_rule"), son nom local, son adresse complète, et ses
// valeurs résolues (`values` du plan).
type Resource struct {
	Type    string
	Name    string
	Address string
	Values  map[string]any
}

// plan reflète la structure minimale de `terraform show -json`. La sortie d'un
// fichier de plan porte `planned_values` (valeurs futures, certains champs
// calculés encore inconnus) ; celle d'un état appliqué porte `values` (tout est
// résolu). On accepte les deux.
type plan struct {
	PlannedValues *valuesBlock `json:"planned_values"`
	Values        *valuesBlock `json:"values"`
}

type valuesBlock struct {
	RootModule module `json:"root_module"`
}

type module struct {
	Resources    []planResource `json:"resources"`
	ChildModules []module       `json:"child_modules"`
}

type planResource struct {
	Address string         `json:"address"`
	Type    string         `json:"type"`
	Name    string         `json:"name"`
	Values  map[string]any `json:"values"`
}

// ParsePlan lit un fichier `terraform show -json` et retourne les ressources de
// tous les modules (racine + enfants), triées de façon déterministe.
func ParsePlan(path string) ([]Resource, error) {
	raw, err := os.ReadFile(path) // #nosec G304 -- chemin de plan Terraform fourni par l'utilisateur en argument CLI, lu en seule lecture.
	if err != nil {
		return nil, fmt.Errorf(i18n.T("lecture du plan %s : %w", "reading the plan %s: %w"), path, err)
	}
	var p plan
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf(i18n.T("plan terraform JSON invalide : %w", "invalid Terraform JSON plan: %w"), err)
	}
	root := p.PlannedValues
	if root == nil {
		root = p.Values
	}
	if root == nil {
		return nil, errors.New(i18n.T("plan terraform sans bloc planned_values ni values (sortie de `terraform show -json` attendue)", "Terraform plan with neither a planned_values nor a values block (`terraform show -json` output expected)"))
	}
	var out []Resource
	collect(&out, root.RootModule)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Address < out[j].Address })
	return out, nil
}

// collect parcourt récursivement un module et ses enfants (équivalent du walk
// d'osc-policy sur planned_values).
func collect(out *[]Resource, m module) {
	for _, r := range m.Resources {
		// Une ressource sans type n'est évaluable par AUCUNE règle : toutes
		// sélectionnent par `resources_of_type(...)`. La laisser entrer ajoute une
		// entrée que rien ne contrôle et pollue le relevé des types collectés, sur
		// lequel repose le verrou de capacité. Trouvé par FuzzParsePlan, sur un plan
		// où `resources: [{}]` — encoding/json apparie les clés sans tenir compte de
		// la casse, donc un plan forgé atteint ce chemin plus facilement qu'il n'y
		// paraît.
		if r.Type == "" {
			continue
		}
		*out = append(*out, Resource{Type: r.Type, Name: r.Name, Address: r.Address, Values: r.Values})
	}
	for _, child := range m.ChildModules {
		collect(out, child)
	}
}

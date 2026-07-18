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
	"fmt"
	"os"
	"sort"
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
		return nil, fmt.Errorf("lecture du plan %s : %w", path, err)
	}
	var p plan
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("plan terraform JSON invalide : %w", err)
	}
	root := p.PlannedValues
	if root == nil {
		root = p.Values
	}
	if root == nil {
		return nil, fmt.Errorf("plan terraform sans bloc planned_values ni values (sortie de `terraform show -json` attendue)")
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
		*out = append(*out, Resource{Type: r.Type, Name: r.Name, Address: r.Address, Values: r.Values})
	}
	for _, child := range m.ChildModules {
		collect(out, child)
	}
}

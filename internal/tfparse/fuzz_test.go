package tfparse

import (
	"os"
	"path/filepath"
	"testing"
)

// FuzzParsePlan — un plan Terraform est une entrée NON FIABLE : il est produit
// par le dépôt audité, pas par pepin. CLAUDE.md §5 impose de le parser
// défensivement et interdit la panique sur une telle entrée.
//
// L'audit de livraison a éprouvé neuf formes malformées à la main (null, [],
// types inversés, imbrication à 8 000 niveaux) sans provoquer de panique. Ce
// fuzz rend la vérification continue plutôt que ponctuelle : toute forme que
// personne n'a imaginée est couverte par construction.
//
// Le contrat éprouvé est simple et volontairement faible : ParsePlan rend un
// résultat OU une erreur, jamais une panique, et jamais une ressource sans type.
// Un plan absurde a le droit d'être refusé ; il n'a pas le droit de faire tomber
// le scanner, car c'est le tiers audité qui choisit ce fichier.
func FuzzParsePlan(f *testing.F) {
	// Graines : les plans réels du dépôt, plus les formes dégénérées connues.
	for _, p := range []string{
		"../../examples/scaleway/terraform/plan.json",
		"../../examples/exoscale/terraform/plan.json",
		"../../examples/exoscale/lab-imds/plan.json",
	} {
		if b, err := os.ReadFile(p); err == nil {
			f.Add(b)
		}
	}
	f.Add([]byte(`{}`))
	f.Add([]byte(`null`))
	f.Add([]byte(`[]`))
	f.Add([]byte(`{"planned_values":null}`))
	f.Add([]byte(`{"planned_values":{"root_module":{"resources":"pas-un-tableau"}}}`))
	f.Add([]byte(`{"planned_values":{"root_module":{"child_modules":[{"child_modules":[]}]}}}`))
	f.Add([]byte(`{"values":{"root_module":{"resources":[{"type":123,"values":null}]}}}`))

	dir := f.TempDir()
	f.Fuzz(func(t *testing.T, data []byte) {
		path := filepath.Join(dir, "plan.json")
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Skip() // défaut d'écriture : hors sujet pour ce qu'on mesure
		}
		res, err := ParsePlan(path)
		if err != nil {
			return // refuser une entrée absurde est le comportement attendu
		}
		for i, r := range res {
			if r.Type == "" {
				t.Fatalf("ressource %d acceptée sans type : une règle la classerait nulle part", i)
			}
		}
	})
}

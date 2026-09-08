package genprovider_test

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/stephrobert/pepin/internal/genprovider"
)

var (
	reReadsType = regexp.MustCompile(`resources_of_type\("([a-z_]+)"\)`)
	reEmitsCode = regexp.MustCompile(`"code":\s*"([a-z0-9_]+)"`)
)

// TestControlTypesMatchTheRules confronte la table déclarée à ce que les règles
// LISENT réellement.
//
// Sans cette porte, `extraControlTypes` serait une seconde table à maintenir à la
// main, et deux tables divergent — celle qui diverge étant toujours celle qu'on
// lit. Ici la source de vérité reste le Rego : la table le déclare, ce test le
// vérifie, et la CI casse dans les deux sens.
//
// Ce qu'il ne peut PAS voir : un type lu à travers un helper de lib.rego plutôt que
// par un `resources_of_type` littéral. Aucune règle ne le fait aujourd'hui ; si
// l'une venait à le faire, ce test la déclarerait conforme à tort. C'est la limite
// d'une analyse textuelle, et elle est écrite plutôt que tue.
func TestControlTypesMatchTheRules(t *testing.T) {
	dir := filepath.Join("..", "commonrules", "rules")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("lecture des règles : %v", err)
	}

	var checked int
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

		read := map[string]bool{}
		for _, m := range reReadsType.FindAllStringSubmatch(text, -1) {
			read[m[1]] = true
		}
		if len(read) == 0 {
			continue
		}
		codes := map[string]bool{}
		for _, m := range reEmitsCode.FindAllStringSubmatch(text, -1) {
			codes[m[1]] = true
		}

		for code := range codes {
			declared := map[string]bool{}
			for _, ty := range genprovider.ControlTypes(code) {
				declared[ty] = true
			}
			checked++

			for ty := range read {
				if !declared[ty] {
					t.Errorf("%s : le contrôle %q lit le type %q sans le déclarer.\n"+
						"  Une collecte incomplète sur ce type ne le dégraderait donc pas, et il\n"+
						"  conclurait sur une donnée jamais arrivée (ADR-0006). Ajouter le type à\n"+
						"  extraControlTypes.", n, code, ty)
				}
			}
			for ty := range declared {
				if !read[ty] {
					t.Errorf("%s : le contrôle %q déclare le type %q qu'il ne lit pas.\n"+
						"  Un type déclaré en trop dégrade le contrôle sans raison : c'est un faux\n"+
						"  `not-evaluated`, moins grave qu'un faux `pass` mais tout aussi faux.",
						n, code, ty)
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("aucun contrôle vérifié : la porte ne mesure rien")
	}
	t.Logf("%d couple(s) contrôle/règle vérifié(s)", checked)
}

// TestEveryMultiTypeControlIsDeclared est le contre-test : il énumère les règles qui
// lisent PLUSIEURS types et exige que chacune ait sa déclaration. Le test précédent
// pourrait passer sur une table vide si aucune règle ne lisait rien ; celui-ci
// interdit ce vert-là.
func TestEveryMultiTypeControlIsDeclared(t *testing.T) {
	dir := filepath.Join("..", "commonrules", "rules")
	entries, _ := os.ReadDir(dir)
	var multi []string
	for _, e := range entries {
		n := e.Name()
		if !strings.HasSuffix(n, ".rego") || strings.HasSuffix(n, "_test.rego") || n == "lib.rego" {
			continue
		}
		b, _ := os.ReadFile(filepath.Join(dir, n)) //nolint:gosec // chemin du dépôt
		read := map[string]bool{}
		for _, m := range reReadsType.FindAllStringSubmatch(string(b), -1) {
			read[m[1]] = true
		}
		if len(read) < 2 {
			continue
		}
		for _, m := range reEmitsCode.FindAllStringSubmatch(string(b), -1) {
			if len(genprovider.ControlTypes(m[1])) < 2 {
				multi = append(multi, m[1])
			}
		}
	}
	sort.Strings(multi)
	if len(multi) > 0 {
		t.Errorf("contrôle(s) lisant plusieurs types sans les déclarer : %v", multi)
	}
}

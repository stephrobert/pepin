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

	// Une règle peut aussi filtrer sur un ENSEMBLE nommé de types :
	//
	//	_located_types := {"compute_instance", "load_balancer"}
	//	...
	//	r.type in _located_types
	//
	// C'est par là que `governance_resource_region_in_eu` est passé : il lit sept
	// types, n'en déclarait aucun, et la porte le trouvait conforme parce qu'elle ne
	// cherchait que des `resources_of_type` littéraux.
	reTypeSet    = regexp.MustCompile(`(?s)(_[a-z_]+)\s*:=\s*\{([^}]*)\}`)
	reInTypeSet  = regexp.MustCompile(`\.type\s+in\s+(_[a-z_]+)`)
	reSetMembers = regexp.MustCompile(`"([a-z_]+)"`)
)

// typesReadViaSets rend les types qu'une règle lit à travers un ensemble nommé
// comparé à `.type`. Seuls les ensembles RÉELLEMENT confrontés à `.type` comptent :
// une règle qui définit un ensemble de protocoles ou de ports ne déclare pas des
// types de ressource.
func typesReadViaSets(text string) map[string]bool {
	used := map[string]bool{}
	for _, m := range reInTypeSet.FindAllStringSubmatch(text, -1) {
		used[m[1]] = true
	}
	out := map[string]bool{}
	for _, m := range reTypeSet.FindAllStringSubmatch(text, -1) {
		if !used[m[1]] {
			continue
		}
		for _, mm := range reSetMembers.FindAllStringSubmatch(m[2], -1) {
			out[mm[1]] = true
		}
	}
	return out
}

// TestControlTypesMatchTheRules confronte la table déclarée à ce que les règles
// LISENT réellement.
//
// Sans cette porte, `extraControlTypes` serait une seconde table à maintenir à la
// main, et deux tables divergent — celle qui diverge étant toujours celle qu'on
// lit. Ici la source de vérité reste le Rego : la table le déclare, ce test le
// vérifie, et la CI casse dans les deux sens.
//
// Il dérive DEUX formes : `resources_of_type("t")`, et un ensemble nommé de types
// confronté à `.type`. La seconde a été ajoutée après coup, parce que la première
// laissait passer exactement le contrôle qui en avait le plus besoin — celui de la
// souveraineté, qui lit sept types sans jamais écrire `resources_of_type`.
//
// Ce qu'il ne peut PAS voir : un type lu à travers un helper de lib.rego. Aucune
// règle ne le fait aujourd'hui ; si l'une venait à le faire, ce test la déclarerait
// conforme à tort. C'est la limite d'une analyse textuelle, et elle est écrite
// plutôt que tue — la version précédente de cette phrase disait la même chose et
// s'était trompée sur l'étendue du trou, ce qui est l'argument pour la mesurer.
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
		for ty := range typesReadViaSets(text) {
			read[ty] = true
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

package cmd

// La référence CLI est une PROMESSE : elle dit ce qu'un intégrateur a le droit de brancher.
// Une promesse qui oublie un drapeau n'est pas incomplète, elle est fausse — l'intégrateur
// conclut que l'option n'existe pas.
//
// Ce test lie donc la page à la MÊME source que la surface gelée (cliSurface, cf.
// frozen_test.go) : tout verbe, tout drapeau public et tout code de sortie de la CLI vivante
// doit apparaître dans docs/reference/cli.md et dans sa version française, et tout code de
// sortie doit apparaître dans docs/reference/exit-codes.md. Ajouter un drapeau sans
// documenter passe alors au rouge, ici, avant la revue.

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// cliReferencePages : les deux versions de la référence CLI (l'anglais est la langue
// primaire, le français sa contrepartie synchronisée).
var cliReferencePages = []string{
	filepath.Join("..", "docs", "reference", "cli.md"),
	filepath.Join("..", "docs", "reference", "cli.fr.md"),
}

// exitCodePages : les deux versions de la page des codes de sortie.
var exitCodePages = []string{
	filepath.Join("..", "docs", "reference", "exit-codes.md"),
	filepath.Join("..", "docs", "reference", "exit-codes.fr.md"),
}

func TestEveryPublicCLIFlagIsDocumented(t *testing.T) {
	surface := cliSurface()
	verbs, ok := surface["verbs"].(map[string]any)
	if !ok || len(verbs) == 0 {
		t.Fatal("aucun verbe observé sur la CLI : le test ne mesurerait plus rien")
	}
	for _, page := range cliReferencePages {
		content := readDocPage(t, page)
		for verb, raw := range verbs {
			if !strings.Contains(content, "pepin "+verb) {
				t.Errorf("%s : le verbe `pepin %s` n'est pas documenté", page, verb)
			}
			spec, _ := raw.(map[string]any)
			flags, _ := spec["flags"].(map[string]string)
			for name := range flags {
				if !strings.Contains(content, "--"+name) {
					t.Errorf("%s : le drapeau `--%s` de `pepin %s` n'est pas documenté "+
						"(un drapeau public non documenté n'existe pas pour l'intégrateur)",
						page, name, verb)
				}
			}
		}
	}
}

func TestEveryExitCodeIsDocumented(t *testing.T) {
	codes, ok := cliSurface()["exit_codes"].(map[string]int)
	if !ok || len(codes) == 0 {
		t.Fatal("aucun code de sortie observé : le test ne mesurerait plus rien")
	}
	for _, page := range append(append([]string{}, cliReferencePages...), exitCodePages...) {
		content := readDocPage(t, page)
		for name, code := range codes {
			// La forme cherchée est celle des tableaux générés (`| **3** |`) : chercher le
			// chiffre nu serait satisfait par n'importe quel nombre de la page, et le
			// contrôle ne mesurerait plus rien.
			if !strings.Contains(content, "**"+strconv.Itoa(code)+"**") {
				t.Errorf("%s : le code de sortie %d (`%s`) n'est pas documenté", page, code, name)
			}
		}
	}
}

// TestTheDocumentedFlagCheckWouldCatchAnOmission : le contrôle ci-dessus ne vaut que s'il
// SAIT échouer. Un drapeau inventé, absent des pages, doit être signalé — sans quoi le test
// mesurerait sa propre panne plutôt que la documentation.
func TestTheDocumentedFlagCheckWouldCatchAnOmission(t *testing.T) {
	for _, page := range cliReferencePages {
		content := readDocPage(t, page)
		if strings.Contains(content, "--drapeau-qui-nexiste-pas") {
			t.Errorf("%s : le témoin du contrôle est présent dans la page ; le test ne discrimine plus rien", page)
		}
	}
}

// readDocPage lit une page de documentation et échoue si elle manque ou si elle est vide :
// une page absente ne doit pas rendre le contrôle vert.
func readDocPage(t *testing.T, page string) string {
	t.Helper()
	raw, err := os.ReadFile(page) // #nosec G304 -- chemin d'une liste constante du paquet.
	if err != nil {
		t.Fatalf("%s : %v — la référence CLI fait partie de la surface publique", page, err)
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		t.Fatalf("%s : page vide", page)
	}
	return string(raw)
}

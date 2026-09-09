package fuzz_test

import (
	"os"
	"path/filepath"
	"testing"
)

// Le corpus de fuzzing versionné est un COÛT PERMANENT : chaque graine est rejouée à
// chaque `go test`. Il est donc plafonné, et le plafond se garde — sans quoi une
// promotion enthousiaste allongerait la suite de tests sans que personne ne le décide.
//
// L'autre moitié de la garde n'a pas besoin d'être écrite : une graine qui fait tomber
// sa cible fait rougir `go test` lui-même. C'est précisément pour cela que
// `tools/fuzz/promote.py` éprouve chaque candidate AVANT de l'accepter, et qu'une
// graine fautive entre au dépôt avec la correction qu'elle motive, jamais avant.

// plafondParCible doit rester en phase avec MAX_PAR_CIBLE de tools/fuzz/promote.py.
// Une marge est laissée pour une promotion délibérée faite avec `--max` plus haut :
// le but est d'attraper une dérive, pas d'interdire une décision.
const plafondParCible = 64

func TestTheVersionedFuzzCorpusStaysBounded(t *testing.T) {
	racine := filepath.Join("..", "..")
	cibles := []string{
		filepath.Join(racine, "internal", "tfparse", "testdata", "fuzz", "FuzzParsePlan"),
		filepath.Join(racine, "cmd", "testdata", "fuzz", "FuzzInventoryWalk"),
	}
	total := 0
	for _, dir := range cibles {
		entrees, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("lecture du corpus %s : %v", dir, err)
		}
		n := 0
		for _, e := range entrees {
			if !e.IsDir() {
				n++
			}
		}
		total += n
		if n == 0 {
			t.Errorf("%s : corpus vide — une campagne repartirait de rien", dir)
		}
		if n > plafondParCible {
			t.Errorf("%s : %d graines, plafond %d.\n"+
				"  Chaque graine est rejouée à chaque `go test` : le corpus est un coût permanent.\n"+
				"  Relever le plafond est une décision, pas un effet de bord d'une promotion.", dir, n, plafondParCible)
		}
	}
	if total == 0 {
		t.Fatal("aucune graine versionnée : la garde ne mesure rien")
	}
	t.Logf("%d graine(s) versionnée(s) sur %d cible(s)", total, len(cibles))
}

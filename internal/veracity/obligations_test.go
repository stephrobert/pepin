package veracity_test

import (
	"sync"
	"testing"

	"github.com/stephrobert/pepin/internal/veracity"

	"github.com/stephrobert/pepin/internal/docgen"
)

// Les OBLIGATIONS sont dérivées de la matrice de couverture, qui est elle-même
// calculée depuis le référentiel, les descripteurs et le verrou du « pass » de
// l'assessment — les mêmes fonctions que celles qu'emprunte le scan. Rien n'est
// recopié : un contrôle ajouté, un fournisseur activé ou un attribut projeté
// change les obligations sans que personne n'ait à s'en souvenir.

var (
	obligationsOnce sync.Once
	obligationsMem  map[veracity.Path][]veracity.Verdict
	obligationsErr  error
)

// obligationsOf calcule les obligations une fois pour tous les tests du paquet :
// BuildMatrix charge les descripteurs et le référentiel, et le refaire à chaque
// test ne mesurerait rien de plus.
func obligationsOf(t *testing.T) map[veracity.Path][]veracity.Verdict {
	t.Helper()
	obligationsOnce.Do(func() {
		// BuildMatrix charge et enregistre les descripteurs lui-même (une seule fois
		// pour le processus) : les enregistrer ici en plus ferait paniquer le registre
		// sur un doublon. La langue, elle, n'entre pas dans le calcul — seules les
		// PROSES des motifs en dépendent, et les obligations n'en lisent aucune.
		m, err := docgen.BuildMatrix(repoRoot, "en")
		if err != nil {
			obligationsErr = err
			return
		}
		obligationsMem = veracity.Obligations(docgen.VeracityCells(m))
	})
	if obligationsErr != nil {
		t.Fatalf("calcul des obligations : %v", obligationsErr)
	}
	return obligationsMem
}

// TestTheObligationsAreDerivedAndNotEmpty : la porte anti-harnais-en-panne. Un
// calcul d'obligations qui rendrait zéro chemin ferait passer le registre de
// dette au vert en ne mesurant plus rien — la panne qui se mesure elle-même
// plutôt que son sujet.
func TestTheObligationsAreDerivedAndNotEmpty(t *testing.T) {
	obligations := obligationsOf(t)
	if len(obligations) < 100 {
		t.Fatalf("%d chemins avec obligation : la matrice en connaît beaucoup plus, le calcul ne mesure plus rien", len(obligations))
	}
	// Les trois verdicts d'un chemin `supported` et le verdict unique d'un chemin
	// `not-applicable` doivent tous deux apparaître : sans quoi la table de
	// correspondance aurait été réduite à un seul cas sans que rien ne rougisse.
	var sawTriple, sawSingleNA bool
	for _, want := range obligations {
		switch {
		case len(want) == 3:
			sawTriple = true
		case len(want) == 1 && want[0] == veracity.NotApplicable:
			sawSingleNA = true
		}
	}
	if !sawTriple {
		t.Error("aucun chemin n'exige les trois verdicts d'un chemin concluant")
	}
	if !sawSingleNA {
		t.Error("aucun chemin n'exige un `not-applicable` : les contrats de fournisseurs en déclarent pourtant")
	}
}

// TestAnUnsupportedPathCarriesNoObligation : Pépin ne conclut rien sur un chemin
// non supporté ; exiger un verdict d'un tel chemin obligerait à truquer une
// fixture pour le faire répondre.
func TestAnUnsupportedPathCarriesNoObligation(t *testing.T) {
	m, err := docgen.BuildMatrix(repoRoot, "en")
	if err != nil {
		t.Fatalf("BuildMatrix : %v", err)
	}
	obligations := veracity.Obligations(docgen.VeracityCells(m))
	checked := 0
	for _, row := range m.Rows {
		for provider, bySource := range row.Cells {
			for source, cell := range bySource {
				if cell.Status != docgen.Unsupported {
					continue
				}
				checked++
				if _, ok := obligations[veracity.Path{Control: row.Code, Provider: provider, Source: string(source)}]; ok {
					t.Errorf("%s / %s / %s est `unsupported` et porte pourtant une obligation", row.Code, provider, source)
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("aucune cellule `unsupported` : le test ne mesure rien")
	}
}

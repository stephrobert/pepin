package quality_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stephrobert/pepin/internal/quality"

	"github.com/stephrobert/pepin/internal/veracity"
)

const repoRoot = "../.."

func embedded(t *testing.T) quality.Snapshot {
	t.Helper()
	s, err := quality.Embedded()
	if err != nil {
		t.Fatalf("carte embarquée illisible : %v", err)
	}
	return s
}

// TestTheMapNeverExceedsTheLedger est LA porte de ce paquet.
//
// La carte publie des chiffres de couverture ; le registre de dette publie ce qui
// reste dû. Ce sont deux vues d'une seule mesure, et si elles pouvaient diverger,
// c'est la carte qu'on lirait — un tableau de bord se lit, un registre de 395
// lignes ne se lit pas. La porte les confronte donc verdict par verdict : la somme
// des verdicts manquants au registre doit être EXACTEMENT ce que la carte annonce
// comme restant à prouver.
func TestTheMapNeverExceedsTheLedger(t *testing.T) {
	lines, err := veracity.LoadLedger(filepath.Join(repoRoot, "internal/veracity/testdata/debt.txt"))
	if err != nil {
		t.Fatalf("chargement du registre : %v", err)
	}
	if len(lines) == 0 {
		t.Fatal("registre vide : la porte ne mesurerait rien")
	}
	// Une ligne : `<contrôle> <fournisseur> <source> <verdicts,manquants>`.
	fromLedger := 0
	for _, l := range lines {
		fields := strings.Fields(l)
		if len(fields) != 4 {
			t.Fatalf("ligne de registre illisible : %q", l)
		}
		fromLedger += len(strings.Split(fields[3], ","))
	}
	s := embedded(t)
	remaining := s.Obligations - s.Proven
	if remaining != fromLedger {
		t.Errorf("la carte annonce %d verdicts restant à prouver, le registre en compte %d.\n"+
			"  Les deux dérivent de la même mesure : lancer `mise run veracity-update` puis `mise run gen-docs`.",
			remaining, fromLedger)
	}
}

// Il n'y a délibérément PAS de test « l'instantané embarqué est celui qui est
// committé » : `go:embed` lit le fichier à la compilation, donc un tel test
// comparerait le fichier à lui-même et passerait toujours — un test qui mesure sa
// propre tautologie plutôt que son sujet. Ce qui garde réellement l'instantané est
// TestGeneratedDocsAreUpToDate (internal/docgen) : il le RÉGÉNÈRE depuis les
// artefacts du dépôt et refuse qu'il en diverge.

// TestTheMapIsDerivedAndNotEmpty : la porte anti-harnais-en-panne. Une carte à
// zéro partout passerait toutes les autres portes en ne mesurant plus rien, et
// afficherait « 0 % » partout — ce qui a l'air honnête et ne l'est pas.
func TestTheMapIsDerivedAndNotEmpty(t *testing.T) {
	s := embedded(t)
	for _, c := range []struct {
		name string
		n    int
	}{
		{"contrôles", s.Controls}, {"chemins", s.Paths},
		{"obligations", s.Obligations}, {"verdicts prouvés", s.Proven},
		{"chemins live", s.LivePaths}, {"tenants", s.Tenants},
	} {
		if c.n <= 0 {
			t.Errorf("%s vaut %d : la carte ne mesure plus rien", c.name, c.n)
		}
	}
	if len(s.ByControl) != s.Controls {
		t.Errorf("%d contrôles au référentiel mais %d dans la carte : un contrôle sans chemin serait invisible",
			s.Controls, len(s.ByControl))
	}
	if len(s.Canary) == 0 {
		t.Error("aucun relevé de canari dans la carte")
	}
	// Les quatre verdicts doivent être représentés : réduire la table à un seul cas
	// ferait tomber les pourcentages sans que rien ne rougisse.
	for _, v := range []veracity.Verdict{
		veracity.Fail, veracity.Pass, veracity.NotEvaluated, veracity.NotApplicable,
	} {
		if s.Verdicts[string(v)].Required == 0 {
			t.Errorf("aucun verdict %q à prouver : la ventilation ne mesure plus rien", v)
		}
	}
}

// TestNoPublishedFigureFlattersTheMeasure : les invariants qui empêchent la carte
// d'annoncer mieux que ce qui est mesuré.
func TestNoPublishedFigureFlattersTheMeasure(t *testing.T) {
	s := embedded(t)
	if s.Proven > s.Obligations {
		t.Errorf("%d verdicts prouvés pour %d à prouver", s.Proven, s.Obligations)
	}
	if s.PathsProven > s.Paths {
		t.Errorf("%d chemins prouvés pour %d chemins", s.PathsProven, s.Paths)
	}
	if s.LiveValidated > s.LivePaths {
		t.Errorf("%d chemins validés en live pour %d chemins live", s.LiveValidated, s.LivePaths)
	}
	if s.Counterwitnesses > s.Tenants {
		t.Errorf("%d contre-témoins pour %d tenants", s.Counterwitnesses, s.Tenants)
	}
	// La somme des ventilations est le total : une ventilation qui ne se boucle pas
	// laisserait publier un pourcentage par verdict sans rapport avec le total.
	sumRequired, sumProven := 0, 0
	for _, c := range s.Verdicts {
		sumRequired += c.Required
		sumProven += c.Proven
	}
	if sumRequired != s.Obligations || sumProven != s.Proven {
		t.Errorf("la ventilation par verdict donne %d/%d, le total annonce %d/%d",
			sumProven, sumRequired, s.Proven, s.Obligations)
	}
}

// TestLiveValidationNeedsAnAuthenticatedRecord : le zéro de « validé en live » est
// DÉRIVÉ, pas écrit. Un relevé de canari est non authentifié : il atteste qu'un
// endpoint refuse, jamais qu'un droit suffisant rende 200 sur un tenant réel. Si ce
// compteur pouvait monter sur un relevé non authentifié, la carte publierait comme
// « validé en live » ce qui n'est qu'une accessibilité d'endpoint.
func TestLiveValidationNeedsAnAuthenticatedRecord(t *testing.T) {
	s := embedded(t)
	authenticated := false
	for _, c := range s.Canary {
		if c.Authenticated {
			authenticated = true
		}
	}
	if authenticated {
		t.Skip("un relevé authentifié existe désormais : ce test ne mesure plus le cas nominal")
	}
	if s.LiveValidated != 0 {
		t.Errorf("« validé en live » vaut %d alors qu'aucun relevé n'est authentifié : "+
			"un refus non authentifié ne vaut pas validation d'un contrôle", s.LiveValidated)
	}
}

// TestNoFigurePathIsAbsolute : l'instantané est committé et embarqué. Un chemin
// absolu y porterait le disque du mainteneur — inutile à un lecteur, et un
// renseignement gratuit sur sa machine.
func TestNoFigurePathIsAbsolute(t *testing.T) {
	for code, proof := range embedded(t).ByControl {
		for _, list := range [][]string{proof.Scenarios, proof.Tenants, proof.RegoTests} {
			for _, p := range list {
				if strings.HasPrefix(p, "/") || strings.Contains(p, ":\\") {
					t.Errorf("%s : chemin absolu dans la carte : %q", code, p)
				}
			}
		}
	}
}

// TestPercentNeverReadsFullOnAnEmptyDenominator : « tout est prouvé » et « il n'y
// avait rien à prouver » ne se lisent pas de la même façon, et le second ne doit
// jamais s'afficher 100 %.
func TestPercentNeverReadsFullOnAnEmptyDenominator(t *testing.T) {
	if got := quality.Percent(0, 0); got != 0 {
		t.Errorf("Percent(0, 0) = %d, attendu 0", got)
	}
	if got := quality.Percent(1, 3); got != 33 {
		t.Errorf("Percent(1, 3) = %d, attendu 33", got)
	}
	if got := quality.Percent(3, 3); got != 100 {
		t.Errorf("Percent(3, 3) = %d, attendu 100", got)
	}
}

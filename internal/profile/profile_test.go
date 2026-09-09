package profile

import (
	"testing"

	"github.com/stephrobert/scankit/finding"
)

func f(code, sev, cat, conf string) finding.Finding {
	return finding.Finding{
		Code: code, Severity: sev,
		Labels: map[string]string{"category": cat, "confidence": conf},
	}
}

// TestTheDefaultProfileFiltersNothing — l'invariant qui rend cette fonctionnalité
// sûre à livrer.
//
// Changer un défaut ferait passer au vert, en silence et sans que personne ne l'ait
// décidé, une chaîne qui échoue aujourd'hui. Ce serait le faux vert que ce projet
// combat, déplacé dans un réglage — donc invisible, donc pire.
func TestTheDefaultProfileFiltersNothing(t *testing.T) {
	tous := []finding.Finding{
		f("a", "critical", "security", "confirmed"),
		f("b", "high", "compliance", "contextual"),
		f("c", "low", "sovereignty", "heuristic"),
		f("d", "medium", "hygiene", "probable"),
		{Code: "e", Severity: "high"}, // sans aucun label
	}
	retenus, ecartes := Split(All, tous)
	if len(retenus) != len(tous) || len(ecartes) != 0 {
		t.Errorf("le profil par défaut a filtré : %d retenus, %d écartés sur %d",
			len(retenus), len(ecartes), len(tous))
	}
}

// TestAnUnlabelledFindingIsAlwaysRetained : le repli penche vers PLUS de sévérité.
//
// Une étiquette absente ne doit pas faire disparaître un écart d'une porte de CI.
// C'est le sens qui compte : dans l'autre, une règle qui oublierait son label
// sortirait silencieusement de toutes les portes.
func TestAnUnlabelledFindingIsAlwaysRetained(t *testing.T) {
	nu := finding.Finding{Code: "sans-label", Severity: "critical"}
	for _, p := range Names() {
		if !Retains(p, nu) {
			t.Errorf("profil %q : un finding sans label a été mis de côté", p)
		}
	}
}

// TestAnUnknownProfileRetainsEverything : un nom inconnu ne doit jamais ALLÉGER une
// porte. La commande le refuse en amont ; si ce refus tombait, le repli doit rester
// du côté sûr.
func TestAnUnknownProfileRetainsEverything(t *testing.T) {
	if !Retains("paranoid", f("a", "low", "hygiene", "heuristic")) {
		t.Error("un profil inconnu a filtré : le repli doit retenir")
	}
}

func TestEachProfileRetainsWhatItNames(t *testing.T) {
	cas := []struct {
		profil string
		f      finding.Finding
		veut   bool
	}{
		{Security, f("a", "high", "security", "confirmed"), true},
		{Security, f("b", "high", "security", "probable"), true},
		// La confiance est ce qui distingue « grave » de « certain » : un écart dont
		// la règle dit elle-même qu'il dépend d'un contexte invisible ne casse pas
		// une chaîne, mais il reste au rapport.
		{Security, f("c", "high", "security", "contextual"), false},
		{Security, f("d", "critical", "compliance", "confirmed"), false},
		{Compliance, f("e", "medium", "compliance", "contextual"), true},
		{Compliance, f("f", "low", "hygiene", "heuristic"), true},
		{Compliance, f("g", "critical", "security", "confirmed"), false},
		{Sovereignty, f("h", "high", "sovereignty", "confirmed"), true},
		{Sovereignty, f("i", "critical", "security", "confirmed"), false},
	}
	for _, c := range cas {
		if got := Retains(c.profil, c.f); got != c.veut {
			t.Errorf("profil %q, finding %s (%s/%s) : retenu=%v, attendu %v",
				c.profil, c.f.Code, c.f.Label("category"), c.f.Label("confidence"), got, c.veut)
		}
	}
}

// TestSetAsideNamesWhatItHidAndWhetherItWasSevere : ce que la porte met de côté doit
// être NOMMÉ et son poids connu — c'est ce qui empêche de rendre 0.
func TestSetAsideNamesWhatItHidAndWhetherItWasSevere(t *testing.T) {
	codes, grave := SetAside([]finding.Finding{
		f("zeta", "low", "hygiene", "heuristic"),
		f("alpha", "medium", "compliance", "probable"),
		f("alpha", "medium", "compliance", "probable"), // même code, deux sujets
	})
	if len(codes) != 2 || codes[0] != "alpha" || codes[1] != "zeta" {
		t.Errorf("codes = %v, attendu [alpha zeta] dédoublonnés et triés", codes)
	}
	if grave {
		t.Error("aucun critical/high mis de côté, et pourtant signalé comme grave")
	}
	if _, grave := SetAside([]finding.Finding{f("x", "HIGH", "compliance", "confirmed")}); !grave {
		t.Error("un `high` mis de côté n'a pas été signalé (la casse ne doit pas compter)")
	}
}

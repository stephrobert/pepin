package referentiel

import (
	"strings"
	"testing"
)

func TestLookupKnownCode(t *testing.T) {
	ctl, ok := Lookup("objectstorage_bucket_public_access")
	if !ok {
		t.Fatal("contrôle objectstorage_bucket_public_access introuvable")
	}
	if ctl.SCSL() != "CLD-STO-1" {
		t.Errorf("SCSL attendu CLD-STO-1, obtenu %q", ctl.SCSL())
	}
	if ctl.Severite != "critical" {
		t.Errorf("sévérité attendue critical, obtenue %q", ctl.Severite)
	}
}

func TestLookupUnknownCode(t *testing.T) {
	if _, ok := Lookup("inexistant_xxx"); ok {
		t.Fatal("un code inconnu ne devrait pas être trouvé")
	}
}

// Cohérence du catalogue : chaque contrôle a un code, une famille, une sévérité
// valide et au moins une exigence SCSL (l'index est la source de vérité).
func TestCatalogueCoherent(t *testing.T) {
	valides := map[string]bool{"critical": true, "high": true, "medium": true, "low": true}
	seen := map[string]bool{}
	for code, ctl := range byCode {
		if code == "" || strings.ContainsAny(code, " ") {
			t.Errorf("code invalide : %q", code)
		}
		if seen[code] {
			t.Errorf("code dupliqué : %q", code)
		}
		seen[code] = true
		if ctl.Famille == "" {
			t.Errorf("%s : famille manquante", code)
		}
		if !valides[ctl.Severite] {
			t.Errorf("%s : sévérité invalide %q", code, ctl.Severite)
		}
		if ctl.SCSL() == "" {
			t.Errorf("%s : aucune exigence SCSL (CLD-*)", code)
		}
	}
}

package referentiel

import (
	"os"
	"testing"
)

// TestSCSLCoherence : aucun contrôle Pépin ne doit mapper une exigence SCSL qui
// en cite un AUTRE dans son outillage (désalignement de mapping après évolution
// de l'index). Cette garde automatise la détection des désalignements et pilote
// la roadmap (cf. commande `pepin scsl`). Ignorée si l'index n'est pas présent.
func TestSCSLCoherence(t *testing.T) {
	if _, err := os.Stat(frameworkExigences); err != nil {
		t.Skipf("index SCSL indisponible (%s) — vérification ignorée", frameworkExigences)
	}
	exs, err := ParseCLDExigences(frameworkExigences)
	if err != nil {
		t.Fatalf("lecture index SCSL : %v", err)
	}
	if len(exs) == 0 {
		t.Fatal("aucune exigence CLD parsée — format de l'index inattendu")
	}
	desal, _ := AnalyseCoherence(exs)
	for _, d := range desal {
		t.Errorf("désalignement : contrôle %q — %s", d.Code, d.Detail)
	}
}

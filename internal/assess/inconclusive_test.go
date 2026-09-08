package assess

import (
	"testing"

	"github.com/stephrobert/scankit/assessment"
	"github.com/stephrobert/scankit/finding"

	"github.com/stephrobert/pepin/referentiel"
)

func inconclusiveFinding() finding.Finding {
	return finding.Finding{
		Code:     "governance_resource_region_in_eu",
		Severity: "medium",
		Subject:  "vm-1",
		Message:  "région non cataloguée",
		Labels:   map[string]string{"provider": "scaleway", LabelInconclusive: "true"},
	}
}

// TestAnInconclusiveFindingIsNotEvaluated tient l'invariant central de l'ADR-0015 :
// une règle qui CONSTATE son incapacité à conclure ne produit pas un écart. Sans
// cette porte, le message « ni établie ni infirmée » continuerait d'être rendu
// comme un `fail`, ce qui est un faux rouge que la règle dément elle-même.
func TestAnInconclusiveFindingIsNotEvaluated(t *testing.T) {
	controls := map[string]referentiel.Control{
		"governance_resource_region_in_eu": {Code: "governance_resource_region_in_eu"},
	}
	asmt := Build("scaleway", controls, []finding.Finding{inconclusiveFinding()},
		map[string]bool{}, nil, map[string]bool{}, map[string]string{},
		map[string]map[string]bool{}, assessment.Run{})

	var seen bool
	for _, r := range asmt.Results {
		if r.Control != "governance_resource_region_in_eu" {
			continue
		}
		seen = true
		if r.Status == assessment.Fail {
			t.Fatalf("un constat d'incertitude est rendu comme un écart : %s", r.Status)
		}
		if r.Status != assessment.NotEvaluated {
			t.Fatalf("statut attendu not-evaluated, obtenu %s", r.Status)
		}
	}
	if !seen {
		// L'invisibilité serait le fail-open que ce mécanisme existe pour empêcher.
		t.Fatal("le constat d'incertitude a disparu du rapport")
	}
}

// TestAnInconclusiveFindingNeverProducesExitOne prouve la seconde moitié : le
// constat sort des décomptes de sévérité. Sans elle, le statut serait juste et la
// porte de CI mentirait quand même.
func TestAnInconclusiveFindingNeverProducesExitOne(t *testing.T) {
	real := finding.Finding{
		Code: "network_securitygroup_x", Severity: "critical", Subject: "sg-1",
		Labels: map[string]string{"provider": "scaleway"},
	}

	deviations, inconclusive := SplitInconclusive([]finding.Finding{inconclusiveFinding(), real})

	if len(inconclusive) != 1 {
		t.Fatalf("constats d'incertitude attendus : 1, obtenus %d", len(inconclusive))
	}
	if len(deviations) != 1 || deviations[0].Code != "network_securitygroup_x" {
		t.Fatalf("seul l'écart réel doit peser dans la porte, obtenu %+v", deviations)
	}

	only, _ := SplitInconclusive([]finding.Finding{inconclusiveFinding()})
	if len(only) != 0 {
		t.Fatal("un scan dont les seuls constats sont incertains ne doit peser aucun écart")
	}
}

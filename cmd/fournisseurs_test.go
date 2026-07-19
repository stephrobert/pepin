package cmd

import (
	"os"
	"testing"

	"github.com/stephrobert/pepin/internal/genprovider"
	"github.com/stephrobert/pepin/referentiel"
)

// governanceMultiType : contrôles de gouvernance évalués via la ressource synthétique
// governance_provider (ou multi-types), sans type de contrat unique à confronter.
var governanceMultiType = map[string]bool{
	"governance_resource_region_in_eu":  true,
	"governance_provider_sovereignty":   true,
	"governance_resource_required_tags": true,
}

// TestFournisseursAreCollected : un contrôle ne peut lister un fournisseur QUE si ce
// provider collecte réellement le type de ressource visé (contrat present, état verifie
// ou a_verifier). Un type `absent` ou non déclaré = fausse déclaration de couverture
// (l'inflation exacte que le projet dénonce). L'état verifie/a_verifier est ensuite
// distingué à l'évaluation (assess : Pass vs NotEvaluated), pas ici.
func TestFournisseursAreCollected(t *testing.T) {
	if err := genprovider.RegisterAll(os.DirFS(".."), "providers"); err != nil {
		t.Fatalf("chargement des providers : %v", err)
	}
	for code, ctl := range referentiel.All() {
		if governanceMultiType[code] {
			continue
		}
		typ := genprovider.ControlType(code)
		if typ == "" {
			t.Errorf("contrôle %q : type de ressource non résolu (ControlType manquant)", code)
			continue
		}
		for _, p := range ctl.Fournisseurs {
			switch genprovider.TypeEtat(p, typ) {
			case "verifie", "a_verifier":
				// le provider collecte ce type : couverture légitime.
			default:
				t.Errorf("contrôle %q liste le fournisseur %q, mais son contrat ne collecte pas le type %q (état %q) — couverture non fondée",
					code, p, typ, genprovider.TypeEtat(p, typ))
			}
		}
	}
}

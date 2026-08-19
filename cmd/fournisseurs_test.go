package cmd

import (
	"os"
	"sync"
	"testing"

	"github.com/stephrobert/pepin/internal/genprovider"
	"github.com/stephrobert/pepin/referentiel"
)

// registerOnce : provider.Register PANIQUE sur un doublon, donc le chargement des
// descripteurs ne peut avoir lieu qu'une fois par binaire de test. Plusieurs tests
// en ont besoin (la couverture déclarée, la surface gelée de l'inventaire) et
// l'ordre d'exécution ne doit décider de rien : une surface gelée qui dépendrait de
// qui a tourné en premier gèlerait tantôt une chose, tantôt une autre.
var registerOnce sync.Once

// ensureProvidersRegistered charge les descripteurs du dépôt, une seule fois.
func ensureProvidersRegistered(t *testing.T) {
	t.Helper()
	var err error
	registerOnce.Do(func() { err = genprovider.RegisterAll(os.DirFS(".."), "providers") })
	if err != nil {
		t.Fatalf("chargement des providers : %v", err)
	}
}

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
	ensureProvidersRegistered(t)
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

package cmd

import (
	"strings"
	"testing"

	"github.com/stephrobert/pepin/referentiel"
)

// Chaque finding imprime un lien de documentation, et ce lien ne servait aucune page :
// la racine concaténée à plat (`…/scsl/CLD-NET-1`) renvoyait le lecteur à l'accueil du
// site. Le référentiel est publié par FAMILLE, chaque exigence y étant une ancre.
//
// Cette garde vaut pour la suite : elle refuse une famille d'exigences ajoutée au
// référentiel sans sa page. Elle ne demande aucun réseau — elle ne peut donc pas
// prouver qu'une page RÉPOND, seulement qu'aucun code n'est laissé sans destination,
// ce qui est exactement le défaut qu'elle existe pour empêcher.
// famillesSansPage — les familles d'exigences dont la page publiée n'est pas connue.
// Chacune coûte un lien de documentation absent sur tous ses contrôles ; aucune ne
// coûte un lien FAUX, ce qui est la seule propriété qu'on refuse de perdre.
//
// La table de l'issue #138 en listait huit ; l'index SCSL gelé en porte neuf. La
// neuvième a été trouvée par cette garde même, à sa première exécution — c'est
// l'argument pour la dériver du référentiel plutôt que de la recopier.
var famillesSansPage = map[string]string{
	"K8S": "12 exigences à l'index gelé, absentes de la table de #138 ; slug de la page à confirmer",
}

func TestEverySCSLFamilyHasItsDocPage(t *testing.T) {
	var verifies int
	for code, ctl := range referentiel.All() {
		for _, id := range ctl.Scsl {
			verifies++
			if url := scslDocURL(id); url != "" {
				continue
			}
			parts := strings.Split(id, "-")
			famille := id
			if len(parts) == 3 {
				famille = parts[1]
			}
			if _, connue := famillesSansPage[famille]; connue {
				continue
			}
			t.Errorf("contrôle %q : l'exigence %q ne produit aucun lien (famille %q inconnue).\n"+
				"  Ses findings s'imprimeraient sans documentation. Ajouter la page de cette\n"+
				"  famille à scslFamilyPage, ou consigner la lacune dans famillesSansPage.",
				code, id, famille)
		}
	}
	if verifies == 0 {
		t.Fatal("aucune exigence vérifiée : la garde ne mesure rien")
	}
	t.Logf("%d exigence(s) SCSL vérifiée(s), %d famille(s) sans page connue", verifies, len(famillesSansPage))
}

// Une lacune CONSIGNÉE doit rester réelle : une famille qui gagne sa page et reste
// inscrite ici masquerait la suivante. Le registre est une porte dans les deux sens,
// comme celui de la dette de véracité.
func TestNoStaleGapInTheDocPageRegistry(t *testing.T) {
	for famille := range famillesSansPage {
		if _, a := scslFamilyPage[famille]; a {
			t.Errorf("famille %q : consignée comme sans page, alors que scslFamilyPage en déclare une. "+
				"Retirer l'entrée de famillesSansPage.", famille)
		}
	}
}

// Un lien FAUX coûte plus cher que pas de lien : il est suivi, et celui qui le suit
// croit avoir lu la bonne page. Ce qui n'est pas une exigence SCSL n'en produit donc
// aucun — notamment le code de check agnostique, qui n'a pas de page publiée.
func TestNoDocLinkIsInventedForANonSCSLCode(t *testing.T) {
	for _, code := range []string{
		"objectstorage_bucket_public_access", // check agnostique
		"CLD-XXX-1",                          // famille inconnue
		"CLD-NET",                            // tronqué
		"SCSL-CLD-NET-1",                     // forme préfixée, non dépréfixée
		"",
	} {
		if url := scslDocURL(code); url != "" {
			t.Errorf("code %q : lien inventé %q", code, url)
		}
	}
}

// Le lien construit doit être exactement celui que la publication sert : page de la
// famille, puis ancre en minuscules.
func TestDocLinkShape(t *testing.T) {
	got := scslDocURL("CLD-NET-1")
	want := "https://blog.stephane-robert.info/docs/securiser/socle/referentiel/cloud/" +
		"exposition-filtrage-reseau/#socle-cld-net-1"
	if got != want {
		t.Errorf("lien construit :\n  %s\nattendu :\n  %s", got, want)
	}
}

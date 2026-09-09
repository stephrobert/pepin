package genprovider_test

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/stephrobert/pepin/internal/genprovider"
)

// Le catalogue des régions existe DEUX fois, et c'est délibéré : la CLI ne lit pas de
// Rego, une règle ne lit pas de descripteur. Ce qui n'est pas négociable, c'est qu'ils
// disent la même chose — sans quoi `pepin scan --region ch-gva-2` avertirait qu'une
// région est inconnue pendant que la règle de souveraineté la catalogue, ou l'inverse.
//
// La garde lit le Rego au motif plutôt qu'en l'évaluant : elle ne cherche pas à
// comprendre les règles, seulement à voir si les deux listes divergent.

var (
	blocRegions = regexp.MustCompile(`(?s)(_eu_regions|_trusted_regions|_noneu_regions)\s*:=\s*\{(.*?)\n\}`)
	parFournis  = regexp.MustCompile(`"([a-z]+)":\s*(\{[^}]*\}|set\(\))`)
	littéral    = regexp.MustCompile(`"([^"]+)"`)
)

// regionsDesRegles rend, par fournisseur, l'union des trois tables de lib.rego.
func regionsDesRegles(t *testing.T) map[string][]string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "commonrules", "rules", "lib.rego"))
	if err != nil {
		t.Fatalf("lecture de lib.rego : %v", err)
	}
	out := map[string]map[string]bool{}
	blocs := blocRegions.FindAllStringSubmatch(string(raw), -1)
	if len(blocs) != 3 {
		t.Fatalf("%d table(s) de régions trouvée(s) dans lib.rego, 3 attendues : la garde ne mesure plus ce qu'elle croit", len(blocs))
	}
	for _, b := range blocs {
		for _, e := range parFournis.FindAllStringSubmatch(b[2], -1) {
			if out[e[1]] == nil {
				out[e[1]] = map[string]bool{}
			}
			for _, r := range littéral.FindAllStringSubmatch(e[2], -1) {
				out[e[1]][r[1]] = true
			}
		}
	}
	plat := map[string][]string{}
	for p, set := range out {
		for r := range set {
			plat[p] = append(plat[p], r)
		}
		sort.Strings(plat[p])
	}
	return plat
}

// TestTheRegionCatalogueMatchesTheRules : le `regions:` d'un descripteur est
// exactement l'union des régions que les règles de souveraineté cataloguent.
func TestTheRegionCatalogueMatchesTheRules(t *testing.T) {
	desRegles := regionsDesRegles(t)
	if len(desRegles) == 0 {
		t.Fatal("aucune région lue dans lib.rego : la garde ne mesure rien")
	}
	vus := map[string]bool{}
	for nom, d := range descripteursDuDepot(t) {
		attendu, catalogué := desRegles[nom]
		vus[nom] = true
		got := append([]string(nil), d.Regions...)
		sort.Strings(got)
		if !catalogué {
			// Un fournisseur que les règles ne cataloguent pas ne doit pas inventer sa
			// propre liste : ce serait une connaissance sans source, et l'avertissement
			// deviendrait faux dans les deux sens.
			if len(got) > 0 {
				t.Errorf("providers/%s.yaml déclare des régions (%s) qu'aucune table de lib.rego ne catalogue",
					nom, strings.Join(got, ", "))
			}
			continue
		}
		if strings.Join(got, ",") != strings.Join(attendu, ",") {
			t.Errorf("providers/%s.yaml et lib.rego divergent :\n  descripteur : %s\n  règles      : %s\n"+
				"  Les deux listes portent la MÊME connaissance ; la CLI lit l'une, les règles l'autre.",
				nom, strings.Join(got, ", "), strings.Join(attendu, ", "))
		}
	}
	for nom := range desRegles {
		if !vus[nom] {
			t.Errorf("lib.rego catalogue des régions pour %q, qui n'a pas de descripteur", nom)
		}
	}
}

// descripteursDuDepot charge les descripteurs COMMITTÉS, plutôt que le registre du
// processus.
//
// Le registre n'est peuplé que par le binaire, qui enregistre les providers au
// démarrage : le lire depuis un test le trouve vide, et une garde qui parcourt une
// carte vide passe en ne mesurant rien. C'est une panne silencieuse qui s'est déjà
// produite ici, et le `t.Fatal` sur la carte vide est ce qui l'empêche de revenir.
func descripteursDuDepot(t *testing.T) map[string]genprovider.Descriptor {
	t.Helper()
	entrees, err := os.ReadDir(filepath.Join("..", "..", "providers"))
	if err != nil {
		t.Fatalf("lecture des descripteurs : %v", err)
	}
	out := map[string]genprovider.Descriptor{}
	for _, e := range entrees {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		d, lerr := genprovider.Load(os.DirFS(filepath.Join("..", "..", "providers")), e.Name())
		if lerr != nil {
			t.Fatalf("chargement de %s : %v", e.Name(), lerr)
		}
		out[strings.TrimSuffix(e.Name(), ".yaml")] = d
	}
	if len(out) == 0 {
		t.Fatal("aucun descripteur chargé : la garde ne mesure rien")
	}
	return out
}

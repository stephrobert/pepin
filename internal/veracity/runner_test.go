package veracity_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stephrobert/pepin/internal/veracity"

	"github.com/stephrobert/pepin/internal/collect"
	"github.com/stephrobert/pepin/internal/genprovider"
	"github.com/stephrobert/pepin/internal/tenants"
)

// coveredNow rend TOUT ce qui est prouvé aujourd'hui : les scénarios écrits à la
// main, et les verdicts que les tenants de référence rendent sur des
// configurations tierces (internal/tenants). Les deux artefacts prouvent la même
// chose — un verdict atteint de bout en bout à travers le binaire — et un seul
// compteur les additionne, pour qu'il n'existe pas deux chiffres de couverture.
func coveredNow(t *testing.T, files []veracity.File) map[veracity.Path][]veracity.Verdict {
	t.Helper()
	fromTenants, err := tenants.Covered(repoRoot)
	if err != nil {
		t.Fatalf("couverture des tenants de référence : %v", err)
	}
	return veracity.Merge(veracity.Covered(files), fromTenants)
}

// Le HARNAIS : il exécute le BINAIRE, pas une reconstitution de ce que le binaire
// ferait. Un harnais qui rejouerait assess.Build à sa façon mesurerait sa propre
// copie de la chaîne, et divergerait du produit au premier changement — c'est
// exactement la panne qui se mesure elle-même plutôt que son sujet.

const repoRoot = "../.."

// scenarioDir et ledgerPath : le contrat de véracité vit dans un seul endroit.
const (
	scenarioDir = "testdata/scenarios"
	ledgerPath  = "testdata/debt.txt"
)

// buildPepin compile le binaire à éprouver. Compiler plutôt que réutiliser un
// ./pepin qui traînerait : une porte adossée à un artefact qu'elle n'a pas
// fabriqué mesure ce qui traîne sur le disque, pas le code.
func buildPepin(t *testing.T) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "pepin")
	cmd := exec.Command("go", "build", "-o", out, ".") // #nosec G204 -- arguments constants.
	cmd.Dir = repoRoot
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compilation du binaire à mesurer : %v\n%s", err, b)
	}
	return out
}

// statusOf exécute un scan et rend le statut du contrôle visé dans l'assessment.
func statusOf(t *testing.T, bin, provider, control string, args ...string) string {
	t.Helper()
	full := append([]string{"scan", provider}, args...)
	full = append(full, "--format", "assessment")
	cmd := exec.Command(bin, full...) // #nosec G204 -- binaire compilé par le test.
	cmd.Dir = repoRoot
	cmd.Env = []string{"NO_COLOR=1", "TERM=dumb", "PEPIN_LANG=en", "HOME=" + t.TempDir()}
	var out, errb strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &errb
	_ = cmd.Run() // un code de sortie non nul est un résultat, pas une panne

	var doc struct {
		Results []struct {
			Control string `json:"control"`
			Status  string `json:"status"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(out.String()), &doc); err != nil {
		t.Fatalf("assessment illisible (%v)\nstdout: %s\nstderr: %s", err, out.String(), errb.String())
	}
	for _, r := range doc.Results {
		if r.Control == control {
			return r.Status
		}
	}
	// Un contrôle absent de l'assessment n'est PAS un verdict : c'est un chemin
	// hors périmètre du scan, et le scénario se trompe de fournisseur ou de code.
	return "(absent)"
}

// inventoryFrom produit le fichier d'inventaire que le scan évaluera, selon le
// point d'entrée du scénario.
//
// Pour un point d'entrée `api`, la spec `collecte` RÉELLE du descripteur est
// exécutée contre un serveur qui sert les réponses canned : c'est le collecteur,
// sa pagination, ses jointures et sa projection qui tournent. L'inventaire
// normalisé qui en sort est écrit tel quel, puis scanné — la frontière du modèle
// est le seul endroit où la chaîne est coupée, et c'est la MÊME coupure que celle
// d'un bundle scellé relu par `verify --re-derive`.
func inventoryFrom(t *testing.T, f veracity.File, s veracity.Scenario) (path string, terraform bool) {
	t.Helper()
	dir := t.TempDir()
	switch s.Entry() {
	case veracity.EntryPlan:
		return writeJSON(t, filepath.Join(dir, "plan.json"), s.Plan), true
	case veracity.EntryInventory:
		inv := map[string]any{"provider": f.Provider}
		for k, v := range s.Inventory {
			inv[k] = v
		}
		return writeJSON(t, filepath.Join(dir, "inventory.json"), inv), false
	default:
		return writeJSON(t, filepath.Join(dir, "inventory.json"), collectFromAPI(t, f, s)), false
	}
}

// collectFromAPI exécute la spec de collecte du descripteur contre un serveur qui
// sert les réponses déclarées par le scénario. Un endpoint non déclaré répond 404,
// ce qui marque son unité incomplète : un scénario ne sert que ce dont son
// contrôle a besoin, et les autres contrôles n'en sont pas affectés.
func collectFromAPI(t *testing.T, f veracity.File, s veracity.Scenario) map[string]any {
	t.Helper()
	desc, ok := genprovider.Descriptors()[f.Provider]
	if !ok {
		t.Fatalf("descripteur %q introuvable", f.Provider)
	}
	denied := map[string]bool{}
	for _, p := range s.Deny {
		denied[p] = true
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if denied[r.URL.Path] {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"Errors":[{"Code":"AccessDenied"}]}`))
			return
		}
		body, served := s.API[r.URL.Path]
		if !served {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"not served by this scenario"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	spec := desc.Collecte
	spec.Provider = f.Provider
	spec.BaseURL = srv.URL
	// Les variables de chemin sont celles qu'un scan réel résout depuis les
	// identifiants. Des valeurs fixes suffisent : ce sont les CHEMINS d'endpoints
	// que le scénario doit connaître, pas leur contenu.
	vars := map[string]string{"region": "fr-par", "zone": "fr-par-1", "org": "org-test", "host": "outscale.com"}
	inv, err := collect.Collect(context.Background(), srv.Client(), spec, nil, vars)
	if err != nil {
		t.Fatalf("%s : collecte impossible : %v", f.Path, err)
	}
	out := map[string]any{"provider": f.Provider, "resources": inv.Resources}
	if !inv.Collection.Empty() {
		out["collection"] = inv.Collection
	}
	return out
}

func writeJSON(t *testing.T, path string, doc any) string {
	t.Helper()
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatalf("sérialisation du scénario : %v", err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("écriture de %s : %v", path, err)
	}
	return path
}

// TestEveryScenarioReachesItsVerdict : chaque scénario committé produit le
// verdict qu'il annonce, sur la chaîne complète et à travers le binaire.
func TestEveryScenarioReachesItsVerdict(t *testing.T) {
	files, err := veracity.LoadScenarios(scenarioDir)
	if err != nil {
		t.Fatalf("chargement des scénarios : %v", err)
	}
	if len(files) == 0 {
		t.Fatal("aucun scénario : le contrat de véracité ne mesure rien")
	}
	// Les descripteurs sont chargés par le calcul des obligations. Un scénario
	// `api:` a besoin de la spec `collecte` réelle : sans cet appel, ce test
	// passerait tantôt (si un autre l'a chargée avant) et échouerait tantôt.
	obligationsOf(t)
	bin := buildPepin(t)
	for _, f := range files {
		for i, s := range f.Scenarios {
			name := f.Control + "/" + f.Provider + "/" + f.Source + "/" + string(s.Expect)
			t.Run(name, func(t *testing.T) {
				if strings.TrimSpace(s.Why) == "" {
					t.Fatalf("%s (scénario %d) : `why` manquant — un cas dont on ignore ce qu'il montre est un cas que personne n'ose corriger", f.Path, i)
				}
				path, tf := inventoryFrom(t, f, s)
				args := []string{path}
				if tf {
					args = []string{"--terraform", path}
				}
				got := statusOf(t, bin, f.Provider, f.Control, args...)
				if got != string(s.Expect) {
					t.Errorf("%s\n  cas      : %s\n  attendu  : %s\n  obtenu   : %s",
						f.Path, s.Why, s.Expect, got)
				}
			})
		}
	}
}

// TestNoScenarioEntersLowerThanItCould : la porte qui empêche d'affaiblir le
// contrat sans que rien ne rougisse. Un scénario `live` dont le type de ressource
// EST produit par la spec `collecte` du descripteur doit entrer par l'API. Entrer
// par l'inventaire y sauterait le collecteur — c'est-à-dire précisément l'endroit
// où l'incident fondateur de cette vague s'est produit.
func TestNoScenarioEntersLowerThanItCould(t *testing.T) {
	files, err := veracity.LoadScenarios(scenarioDir)
	if err != nil {
		t.Fatalf("chargement des scénarios : %v", err)
	}
	obligationsOf(t) // charge les descripteurs (cf. TestEveryScenarioReachesItsVerdict)
	descs := genprovider.Descriptors()
	for _, f := range files {
		if f.Source != "live" {
			continue
		}
		typ := genprovider.ControlType(f.Control)
		produced := false
		for _, r := range descs[f.Provider].Collecte.Resources {
			if r.Type == typ {
				produced = true
				break
			}
		}
		if !produced {
			continue
		}
		for _, s := range f.Scenarios {
			if s.Entry() == veracity.EntryInventory {
				t.Errorf("%s : le type %q est produit par la spec `collecte` de %s ; le scénario « %s » doit entrer par `api:` et non par `inventory:` — sinon le collecteur n'est pas éprouvé",
					f.Path, typ, f.Provider, s.Why)
			}
		}
	}
}

// TestScenariosDeclareAPathThatExists : un scénario qui viserait un contrôle ou
// un fournisseur inexistant passerait au vert sans rien prouver.
func TestScenariosDeclareAPathThatExists(t *testing.T) {
	files, err := veracity.LoadScenarios(scenarioDir)
	if err != nil {
		t.Fatalf("chargement des scénarios : %v", err)
	}
	obligations := obligationsOf(t)
	for _, f := range files {
		if f.Source != "live" && f.Source != "terraform" {
			t.Errorf("%s : source %q inconnue (live | terraform)", f.Path, f.Source)
			continue
		}
		want, ok := obligations[f.PathOf()]
		if !ok {
			t.Errorf("%s : le chemin %s n'a aucune obligation — contrôle inconnu, fournisseur inconnu, ou chemin sur lequel Pépin ne conclut rien",
				f.Path, f.PathOf())
			continue
		}
		expected := map[veracity.Verdict]bool{}
		for _, v := range want {
			expected[v] = true
		}
		for _, s := range f.Scenarios {
			if !expected[s.Expect] {
				t.Errorf("%s : le verdict %q n'est pas atteignable sur ce chemin (attendus : %v)",
					f.Path, s.Expect, want)
			}
		}
	}
}

// TestTheVeracityDebtLedgerIsExact est la porte qui rend le contrat CONTRAIGNANT.
//
// Elle échoue dans les deux sens, et les deux comptent :
//
//   - une obligation non prouvée et absente du registre : c'est le contrôle ajouté
//     sans ses scénarios, qui casse la CI comme l'exige l'issue ;
//   - une ligne de registre qui ne correspond plus à aucune obligation : c'est une
//     dette payée qu'on n'a pas rayée, ou un chemin disparu. Un registre qui
//     surestime la dette est aussi faux qu'un registre qui la sous-estime.
func TestTheVeracityDebtLedgerIsExact(t *testing.T) {
	files, err := veracity.LoadScenarios(scenarioDir)
	if err != nil {
		t.Fatalf("chargement des scénarios : %v", err)
	}
	got := veracity.Debt(obligationsOf(t), coveredNow(t, files))
	// La régénération est EXPLICITE et jamais automatique : un registre que le test
	// réécrirait tout seul serait un registre qui ne dit plus rien, et la dette
	// grandirait sans que personne ne la voie grandir.
	if os.Getenv("PEPIN_UPDATE_VERACITY") != "" {
		writeLedger(t, got)
	}
	want, err := veracity.LoadLedger(ledgerPath)
	if err != nil {
		t.Fatalf("chargement du registre : %v", err)
	}
	added, removed := diff(want, got)
	for _, l := range added {
		t.Errorf("dette NON consignée : %s\n  Un chemin sans ses scénarios doit être écrit au registre (%s), ou prouvé.", l, ledgerPath)
	}
	for _, l := range removed {
		t.Errorf("dette consignée à tort : %s\n  Elle est prouvée, ou le chemin n'existe plus : retirer la ligne de %s.", l, ledgerPath)
	}
}

// diff rend ce qui est dans `got` sans être dans `want`, et l'inverse.
func diff(want, got []string) (added, removed []string) {
	inWant := map[string]bool{}
	for _, l := range want {
		inWant[l] = true
	}
	inGot := map[string]bool{}
	for _, l := range got {
		inGot[l] = true
		if !inWant[l] {
			added = append(added, l)
		}
	}
	for _, l := range want {
		if !inGot[l] {
			removed = append(removed, l)
		}
	}
	return added, removed
}

// writeLedger réécrit le registre committé. Il porte son en-tête en clair : un
// fichier de dette qu'on ouvre sans savoir ce qu'il compte est un fichier qu'on
// referme.
func writeLedger(t *testing.T, lines []string) {
	t.Helper()
	counts := veracity.Count(obligationsOf(t), coveredNow(t, mustScenarios(t)))
	header := []string{
		"# Registre de dette du contrat de véracité — RÉGÉNÉRÉ, jamais édité à la main.",
		"#   mise run veracity-update",
		"#",
		"# Une ligne = un chemin contrôle × fournisseur × source, et les verdicts qu'il",
		"# ne sait pas encore prouver de bout en bout (réponse d'API ou plan → collecteur →",
		"# normalisation → garde de capacité → Rego → assessment → verdict).",
		"#",
		"# Une ligne qui disparaît est une dette payée. Une ligne qui apparaît sans",
		"# scénario est un contrôle ajouté sans sa preuve, et la CI la refuse.",
		"#",
		fmt.Sprintf("# chemins : %d · entierement prouves : %d · obligations : %d · restantes : %d",
			counts.Paths, counts.PathsProven, counts.Obligations, counts.Remaining),
		"",
	}
	body := strings.Join(append(header, lines...), "\n") + "\n"
	if err := os.WriteFile(ledgerPath, []byte(body), 0o600); err != nil {
		t.Fatalf("écriture du registre : %v", err)
	}
	t.Logf("registre régénéré : %d ligne(s) de dette", len(lines))
}

// mustScenarios charge les scénarios ou fait échouer le test.
func mustScenarios(t *testing.T) []veracity.File {
	t.Helper()
	files, err := veracity.LoadScenarios(scenarioDir)
	if err != nil {
		t.Fatalf("chargement des scénarios : %v", err)
	}
	return files
}

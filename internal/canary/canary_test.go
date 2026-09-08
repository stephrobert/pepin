package canary_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stephrobert/pepin/internal/canary"

	"github.com/stephrobert/pepin/internal/genprovider"
)

const repoRoot = "../.."

func load(t *testing.T) []canary.Record {
	t.Helper()
	recs, err := canary.Load(repoRoot)
	if err != nil {
		t.Fatalf("chargement des relevés : %v", err)
	}
	if len(recs) == 0 {
		t.Fatal("aucun relevé de canari : rien n'est mesuré contre un vrai plan de contrôle")
	}
	return recs
}

// TestEveryCloudProviderHasACanaryRecord : la COMPLÉTUDE, vérifiée en CI parce
// qu'elle ne dépend pas de la date. Un fournisseur cloud ajouté sans son relevé
// publierait une capacité live que personne n'a jamais vue répondre.
//
// Les fournisseurs hors périmètre cloud en sont exclus, et pour une raison de
// fait : `kubernetes` s'authentifie par kubeconfig et son `base_url` est le
// serveur que ce fichier nomme. Il n'existe aucun plan de contrôle public à
// interroger, donc aucun refus à mesurer.
func TestEveryCloudProviderHasACanaryRecord(t *testing.T) {
	if err := genprovider.RegisterAll(os.DirFS(repoRoot), "providers"); err != nil {
		// Déjà enregistrés par un autre test du binaire : ce n'est pas une panne.
		t.Logf("enregistrement des descripteurs : %v", err)
	}
	have := canary.ByProvider(load(t))
	checked := 0
	for name, desc := range genprovider.Descriptors() {
		if desc.Scope != "" && desc.Scope != "cloud" {
			continue
		}
		checked++
		if _, ok := have[name]; !ok {
			t.Errorf("le fournisseur cloud %q n'a aucun relevé de canari\n"+
				"  Le produire : tools/release/canary.sh %s", name, name)
		}
	}
	if checked == 0 {
		t.Fatal("aucun fournisseur cloud examiné : le test ne mesure rien")
	}
}

// TestEveryCanaryRecordIsSubstantive : un relevé dont toutes les unités sont
// injoignables parle du réseau du mainteneur, pas du fournisseur. Le consigner
// ferait lire une régression du plan de contrôle là où il n'y a qu'un proxy.
func TestEveryCanaryRecordIsSubstantive(t *testing.T) {
	for _, r := range load(t) {
		t.Run(r.Provider, func(t *testing.T) {
			if !r.Substantive() {
				t.Errorf("%s : %d unité(s), toutes injoignables — ce relevé ne mesure pas le fournisseur",
					r.Path, len(r.Units))
			}
			if r.Authenticated {
				t.Errorf("%s : `authenticated: true` — le canari n'a jamais d'identifiant, "+
					"et un relevé authentifié aurait pu en écrire un", r.Path)
			}
			if _, err := r.RecordedOn(); err != nil {
				t.Errorf("%s : %v — un relevé qu'on ne sait pas dater ne peut rien attester", r.Path, err)
			}
		})
	}
}

// credentialShaped : les formes que prennent les identifiants des trois
// fournisseurs souverains, y compris les jetons SYNTHÉTIQUES que canary.sh
// exporte. Aucun ne doit atteindre un relevé — le canari n'en produit pas, et
// c'est ce test qui permet de l'AFFIRMER plutôt que de l'espérer.
var credentialShaped = []*regexp.Regexp{
	regexp.MustCompile(`SCW[A-Z0-9]{17}`),                     // clé d'accès Scaleway
	regexp.MustCompile(`EXO[A-Za-z0-9]{20}`),                  // clé d'accès Exoscale
	regexp.MustCompile(`\bCANARY[A-Z]{5,}`),                   // les jetons synthétiques du script
	regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}`), // UUID d'organisation/projet
	regexp.MustCompile(`(?i)\b(authorization|x-auth-token)\b`),
}

// TestNoCanaryRecordCarriesACredentialOrATenant : le relevé ne porte que des FAITS
// d'endpoint. Ni identifiant, ni query string (le seul endroit où une valeur de
// tenant pourrait se glisser dans une URL), ni corps de réponse.
func TestNoCanaryRecordCarriesACredentialOrATenant(t *testing.T) {
	for _, r := range load(t) {
		t.Run(r.Provider, func(t *testing.T) {
			raw, err := os.ReadFile(r.Path) // #nosec G304 -- artefact du dépôt.
			if err != nil {
				t.Fatalf("lecture de %s : %v", r.Path, err)
			}
			for _, re := range credentialShaped {
				if m := re.Find(raw); m != nil {
					t.Errorf("%s porte une forme d'identifiant (%s) : %q", r.Path, re, m)
				}
			}
			for _, u := range r.Units {
				if strings.Contains(u.Path, "?") {
					t.Errorf("%s : le chemin de %q porte une query string (%q) — c'est là qu'une valeur de tenant se glisse",
						r.Path, u.Unit, u.Path)
				}
			}
		})
	}
}

// TestTheFreshnessWindowJudgesTheDate éprouve la fenêtre DANS LES DEUX SENS. Une
// fonction de fraîcheur qui rendrait toujours « frais » désarmerait le préflight
// sans que rien ne rougisse.
func TestTheFreshnessWindowJudgesTheDate(t *testing.T) {
	rec := canary.Record{Recorded: "2026-01-01"}
	day, err := rec.RecordedOn()
	if err != nil {
		t.Fatalf("RecordedOn : %v", err)
	}
	if rec.Stale(day.Add(canary.MaxAge - time.Hour)) {
		t.Error("un relevé DANS la fenêtre est déclaré périmé")
	}
	if !rec.Stale(day.Add(canary.MaxAge + time.Hour)) {
		t.Error("un relevé HORS de la fenêtre est déclaré frais : le préflight ne mesurerait plus rien")
	}
	// Une date illisible vaut périmée : on ne peut pas attester la fraîcheur de ce
	// qu'on ne sait pas dater.
	if !(canary.Record{Recorded: "hier"}).Stale(time.Now()) {
		t.Error("une date illisible est déclarée fraîche")
	}
}

// TestASubstantiveRecordNeedsOneAnswer éprouve le filtre « substantiel » dans les
// deux sens, sur des relevés construits à la main : c'est lui qui distingue « le
// fournisseur a bougé » de « le mainteneur est derrière un proxy ».
func TestASubstantiveRecordNeedsOneAnswer(t *testing.T) {
	allDown := canary.Record{Units: []canary.Unit{
		{Unit: "a", Verdict: canary.Unreachable},
		{Unit: "b", Verdict: canary.Unreachable},
	}}
	if allDown.Substantive() {
		t.Error("un relevé entièrement injoignable est déclaré substantiel")
	}
	if (canary.Record{}).Substantive() {
		t.Error("un relevé sans unité est déclaré substantiel")
	}
	// Un seul 404 suffit à parler du fournisseur : c'est même le signal recherché.
	oneMoved := canary.Record{Units: []canary.Unit{
		{Unit: "a", Verdict: canary.Unreachable},
		{Unit: "b", Verdict: canary.Moved},
	}}
	if !oneMoved.Substantive() {
		t.Error("un relevé portant un 404 est déclaré non substantiel : c'est pourtant la régression cherchée")
	}
}

// TestTheRecordedSummaryMatchesItsUnits : le résumé est un décompte, pas une
// annonce. S'il pouvait diverger des unités, c'est le résumé qu'on lirait.
func TestTheRecordedSummaryMatchesItsUnits(t *testing.T) {
	for _, r := range load(t) {
		t.Run(r.Provider, func(t *testing.T) {
			got := map[string]int{}
			for _, u := range r.Units {
				got[u.Verdict]++
			}
			want := map[string]int{
				canary.Answered: r.Summary.Answered,
				canary.Moved:    r.Summary.Moved, canary.Unreachable: r.Summary.Unreachable,
			}
			for verdict, n := range want {
				if got[verdict] != n {
					t.Errorf("%s : le résumé annonce %d %q, les unités en portent %d",
						r.Path, n, verdict, got[verdict])
				}
			}
		})
	}
}

// TestThePreflightCitesTheSameFreshnessWindow : la fenêtre est écrite à deux
// endroits — la constante Go, que la documentation cite, et le script de
// préflight, qui l'applique. Deux nombres finissent toujours par diverger, et
// celui qui divergerait ici est celui qui décide de taguer.
func TestThePreflightCitesTheSameFreshnessWindow(t *testing.T) {
	const script = "../../tools/release/preflight.sh"
	raw, err := os.ReadFile(script)
	if err != nil {
		t.Fatalf("lecture de %s : %v", script, err)
	}
	m := regexp.MustCompile(`CANARY_MAX_AGE_DAYS=(\d+)`).FindSubmatch(raw)
	if m == nil {
		t.Fatalf("%s ne fixe aucun CANARY_MAX_AGE_DAYS : la fraîcheur du canari n'est plus vérifiée avant un tag", script)
	}
	wantDays := int(canary.MaxAge / (24 * time.Hour))
	if got := string(m[1]); got != strconv.Itoa(wantDays) {
		t.Errorf("le préflight exige %s jours, canary.MaxAge en vaut %d", got, wantDays)
	}
}

// TestTheCanaryDirectoryHoldsOnlyRecords : un fichier étranger déposé là serait
// chargé comme un relevé, ou ignoré en silence. Les deux sont mauvais.
func TestTheCanaryDirectoryHoldsOnlyRecords(t *testing.T) {
	entries, err := os.ReadDir(filepath.Join(repoRoot, canary.Dir))
	if err != nil {
		t.Fatalf("lecture du dossier des relevés : %v", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			t.Errorf("%s n'est pas un relevé : ce dossier ne porte que des *.yaml générés par canary.sh", e.Name())
		}
	}
}

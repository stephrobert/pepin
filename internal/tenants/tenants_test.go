package tenants_test

// Les portes des tenants de référence.
//
// Elles mesurent le PRODUIT : le binaire est compilé, lancé sur un plan tiers, et
// c'est ce qu'il imprime qui est comparé. Un harnais qui rejouerait assess.Build à
// sa façon mesurerait sa propre copie de la chaîne.

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stephrobert/pepin/internal/genprovider"
	"github.com/stephrobert/pepin/internal/tenants"
)

const repoRoot = "../.."

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

// result est ce qu'on lit d'un assessment : le statut et la sévérité par contrôle.
type result struct {
	status   string
	severity string
}

// scan exécute le binaire sur le plan d'un tenant et rend, par contrôle, son
// verdict. Un contrôle apparaît une fois : plusieurs écarts sur un même contrôle
// sont un même verdict.
func scan(t *testing.T, bin string, ten tenants.Tenant) map[string]result {
	t.Helper()
	plan, err := filepath.Abs(ten.PlanPath())
	if err != nil {
		t.Fatalf("chemin du plan : %v", err)
	}
	cmd := exec.Command(bin, "scan", ten.Provider, "--terraform", plan, "--format", "assessment") // #nosec G204 -- binaire compilé par le test.
	cmd.Dir = repoRoot
	cmd.Env = []string{"NO_COLOR=1", "TERM=dumb", "PEPIN_LANG=en", "HOME=" + t.TempDir()}
	var out, errb strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &errb
	_ = cmd.Run() // un code de sortie non nul est un résultat, pas une panne

	var doc struct {
		Results []struct {
			Control  string `json:"control"`
			Status   string `json:"status"`
			Severity string `json:"severity"`
		} `json:"results"`
	}
	if uerr := json.Unmarshal([]byte(out.String()), &doc); uerr != nil {
		t.Fatalf("%s : assessment illisible (%v)\nstderr: %s", ten.Name, uerr, errb.String())
	}
	got := map[string]result{}
	for _, r := range doc.Results {
		// `fail` l'emporte sur un autre statut du même contrôle : c'est le verdict
		// que l'assessment publie pour lui.
		if prev, seen := got[r.Control]; seen && prev.status == "fail" {
			continue
		}
		got[r.Control] = result{status: r.Status, severity: r.Severity}
	}
	return got
}

func load(t *testing.T) []tenants.Tenant {
	t.Helper()
	list, err := tenants.Load(repoRoot)
	if err != nil {
		t.Fatalf("chargement des tenants : %v", err)
	}
	if len(list) == 0 {
		t.Fatal("aucun tenant de référence : la non-régression ne mesure plus rien")
	}
	return list
}

// TestEveryReferenceTenantProducesItsExpectedVerdicts est la porte de
// non-régression : sur une configuration que personne n'a écrite pour Pépin, le
// verdict rendu est celui qui est consigné.
//
// Elle échoue dans les DEUX sens, et c'est ce qui la rend utile : un verdict qui
// bascule vers `fail` est un faux positif candidat, un verdict qui bascule vers
// `pass` est un faux négatif candidat. Aucun des deux ne se régénère sans avoir
// dit lequel c'était.
func TestEveryReferenceTenantProducesItsExpectedVerdicts(t *testing.T) {
	list := load(t)
	bin := buildPepin(t)
	update := os.Getenv("PEPIN_UPDATE_TENANTS") != ""
	for _, ten := range list {
		t.Run(ten.Name, func(t *testing.T) {
			got := scan(t, bin, ten)
			if update {
				flat := map[string]string{}
				for code, r := range got {
					flat[code] = r.status
				}
				if err := tenants.WriteExpected(ten, flat); err != nil {
					t.Fatalf("régénération : %v", err)
				}
			}
			want, err := tenants.LoadExpected(ten.ExpectedPath())
			if err != nil {
				t.Fatalf("%s : %v", ten.Name, err)
			}
			if len(want) == 0 {
				t.Fatalf("%s : relevé vide — un tenant sans verdict attendu ne mesure rien", ten.Name)
			}
			for code, w := range want {
				g, seen := got[code]
				if !seen {
					t.Errorf("%s : le contrôle %s a disparu de l'assessment (attendu : %s)", ten.Name, code, w)
					continue
				}
				if g.status != w {
					t.Errorf("%s : %s\n  consigné : %s\n  obtenu   : %s\n"+
						"  Un verdict qui bascule sur une configuration tierce INCHANGÉE est un faux positif\n"+
						"  ou un faux négatif. Dire lequel, puis `mise run tenants-update`.",
						ten.Name, code, w, g.status)
				}
			}
			for code, g := range got {
				if _, seen := want[code]; !seen {
					t.Errorf("%s : le contrôle %s (%s) est apparu et n'est pas au relevé — `mise run tenants-update`",
						ten.Name, code, g.status)
				}
			}
		})
	}
}

// TestEveryReferenceTenantDeclaresItsProvenance : un plan committé sans amont
// vérifiable est un plan que le dépôt a fini par s'écrire à lui-même, c'est-à-dire
// la fixture auto-confirmante qu'on cherchait à quitter.
func TestEveryReferenceTenantDeclaresItsProvenance(t *testing.T) {
	sha := regexp.MustCompile(`^[0-9a-f]{40}$`)
	for _, ten := range load(t) {
		t.Run(ten.Name, func(t *testing.T) {
			if !strings.HasPrefix(ten.Upstream.Repo, "https://") {
				t.Errorf("amont non vérifiable : repo = %q", ten.Upstream.Repo)
			}
			if !sha.MatchString(ten.Upstream.Commit) {
				t.Errorf("commit %q : un SHA complet est attendu — une branche bouge, un commit non", ten.Upstream.Commit)
			}
			if ten.Upstream.Licence == "" {
				t.Error("licence de l'amont non déclarée : un artefact dérivé d'un code tiers se cite")
			}
			if ten.Upstream.Retrieved == "" {
				t.Error("date de récupération absente")
			}
			if ten.Upstream.Path == "" {
				t.Error("chemin du module dans l'amont absent")
			}
			// Bilinguisme : tout texte destiné à un lecteur s'écrit deux fois (CLAUDE.md §1.2).
			for _, c := range []struct{ name, fr, en string }{
				{"title", ten.Title, ten.TitleEn},
				{"why", ten.Why, ten.WhyEn},
			} {
				if strings.TrimSpace(c.fr) == "" {
					t.Errorf("%s : version française absente", c.name)
				}
				if strings.TrimSpace(c.en) == "" {
					t.Errorf("%s_en : version anglaise absente", c.name)
				}
			}
			if ten.Source != "terraform" && ten.Source != "live" {
				t.Errorf("source %q inconnue (live | terraform)", ten.Source)
			}
		})
	}
}

// TestNoReferenceTenantPlanCarriesMoreThanPepinReads : un plan de tenant ne porte
// que ce que internal/tfparse lit. Les blocs écartés (`variables`,
// `provider_config`, `prior_state`, `resource_changes`) sont ceux où un plan pris
// sur un tenant RÉEL porte ses identifiants — la garde est de sécurité avant
// d'être de taille.
func TestNoReferenceTenantPlanCarriesMoreThanPepinReads(t *testing.T) {
	for _, ten := range load(t) {
		t.Run(ten.Name, func(t *testing.T) {
			extra, err := tenants.CheckPlanShape(ten.PlanPath())
			if err != nil {
				t.Fatalf("%v", err)
			}
			if len(extra) > 0 {
				t.Errorf("le plan porte des blocs que Pépin ne lit pas : %s\n"+
					"  Les réduire (scripts/reference-tenant.sh) : c'est là qu'un plan réel cacherait ses secrets.",
					strings.Join(extra, ", "))
			}
		})
	}
}

// TestEveryPostureIsTheOneMeasured : la posture annoncée est celle que le scan
// constate. Un tenant annoncé « durci » qui rendrait un écart critical/high
// mentirait sur ce qu'il prouve, et le corpus perdrait son contre-témoin.
func TestEveryPostureIsTheOneMeasured(t *testing.T) {
	list := load(t)
	bin := buildPepin(t)
	for _, ten := range list {
		t.Run(ten.Name, func(t *testing.T) {
			got := scan(t, bin, ten)
			var fails, severe []string
			for code, r := range got {
				if r.status != "fail" {
					continue
				}
				fails = append(fails, code)
				if r.severity == "critical" || r.severity == "high" {
					severe = append(severe, code)
				}
			}
			switch ten.Posture {
			case tenants.PostureHardened:
				if len(severe) > 0 {
					t.Errorf("posture « durcie » démentie : écarts critical/high sur %s", strings.Join(severe, ", "))
				}
			case tenants.PostureExposed:
				if len(fails) == 0 {
					t.Errorf("posture « exposée » démentie : aucun écart relevé — le tenant ne montre plus ce qu'il annonce")
				}
			default:
				t.Errorf("posture %q inconnue (%s | %s)", ten.Posture, tenants.PostureExposed, tenants.PostureHardened)
			}
		})
	}
}

// TestEveryCloudProviderHasAReferenceTenant : un fournisseur sans tenant de
// référence n'a que ses propres fixtures pour se juger.
func TestEveryCloudProviderHasAReferenceTenant(t *testing.T) {
	if err := genprovider.RegisterAll(os.DirFS(repoRoot), "providers"); err != nil &&
		!strings.Contains(err.Error(), "déjà enregistré") {
		t.Logf("enregistrement des descripteurs : %v", err)
	}
	have := map[string]bool{}
	for _, ten := range load(t) {
		have[ten.Provider] = true
	}
	nonCloud := genprovider.NonCloudProviders()
	for name := range genprovider.Descriptors() {
		if nonCloud[name] {
			continue // portée intra-cluster : un plan Terraform ne la décrit pas
		}
		if !have[name] {
			t.Errorf("le fournisseur %q n'a aucun tenant de référence : il ne se juge que sur ses propres fixtures", name)
		}
	}
}

// TestTheSubstantiveFilterRejectsAnEmptyNotEvaluated : le filtre qui décide ce
// qu'un tenant PAIE au contrat de véracité doit savoir refuser. Sans ce cas, on ne
// saurait pas que le filtre filtre.
func TestTheSubstantiveFilterRejectsAnEmptyNotEvaluated(t *testing.T) {
	const code = "objectstorage_bucket_public_access" // type visé : object_storage_bucket
	empty := map[string]bool{"": true}
	full := map[string]bool{"": true, "object_storage_bucket": true}

	if tenants.Substantive(code, "not-evaluated", empty) {
		t.Error("un « not-evaluated » sur un tenant sans bucket est compté comme une preuve : le filtre ne filtre plus")
	}
	if !tenants.Substantive(code, "not-evaluated", full) {
		t.Error("un « not-evaluated » sur un tenant QUI a un bucket doit compter : c'est la garde de capacité qui est éprouvée")
	}
	for _, st := range []string{"fail", "pass", "not-applicable"} {
		if !tenants.Substantive(code, st, empty) {
			t.Errorf("un %q conclut sur une donnée réelle : il compte toujours", st)
		}
	}
}

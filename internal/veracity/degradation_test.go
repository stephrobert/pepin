package veracity_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stephrobert/pepin/internal/veracity"

	"github.com/stephrobert/pepin/internal/collect"
	"github.com/stephrobert/pepin/internal/genprovider"
	"github.com/stephrobert/pepin/referentiel"
)

// LES CHEMINS DE DÉGRADATION, et la seule garantie qui compte : **jamais un
// `pass`**.
//
// Ces cas ne se prouvent pas contrôle par contrôle — ils ne dépendent d'aucun
// contrôle en particulier, mais de la chaîne. Un seul jeu les couvre donc pour
// tous les contrôles du type touché, et la garantie est vérifiée sur TOUS ces
// contrôles à la fois : c'est plus fort que de la vérifier sur un témoin choisi.
//
// La liste vient de l'issue #43, et chaque entrée est produite pour de vrai —
// un vrai serveur qui refuse, un vrai serveur qui tronque, un vrai plan dont
// l'attribut est absent — jamais simulée par un drapeau interne.

// statusesOfType rend les statuts de tous les contrôles qui lisent `typ`, tels
// que l'assessment les publie.
func statusesOfType(t *testing.T, bin, provider, typ string, args ...string) map[string]string {
	t.Helper()
	full := append([]string{"scan", provider}, args...)
	full = append(full, "--format", "assessment")
	cmd := exec.Command(bin, full...) // #nosec G204 -- binaire compilé par le test.
	cmd.Dir = repoRoot
	cmd.Env = []string{"NO_COLOR=1", "TERM=dumb", "PEPIN_LANG=en", "HOME=" + t.TempDir()}
	var out, errb strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &errb
	_ = cmd.Run()

	var doc struct {
		Results []struct {
			Control string `json:"control"`
			Status  string `json:"status"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(out.String()), &doc); err != nil {
		t.Fatalf("assessment illisible (%v)\nstdout: %s\nstderr: %s", err, out.String(), errb.String())
	}
	got := map[string]string{}
	for _, r := range doc.Results {
		if genprovider.ControlType(r.Control) == typ {
			got[r.Control] = r.Status
		}
	}
	if len(got) == 0 {
		t.Fatalf("aucun contrôle ne lit le type %q : le cas de dégradation ne mesure rien", typ)
	}
	return got
}

// assertNoPass est la garantie elle-même.
func assertNoPass(t *testing.T, mode string, statuses map[string]string) {
	t.Helper()
	for control, status := range statuses {
		if status == string(veracity.Pass) {
			t.Errorf("dégradation « %s » : le contrôle %s a rendu `pass`. Un périmètre non lu ne prouve aucune conformité.",
				mode, control)
		}
	}
}

// liveInventory exécute la spec de collecte réelle contre un serveur donné et
// écrit l'inventaire normalisé qui en sort.
func liveInventory(t *testing.T, provider string, srv *httptest.Server) string {
	t.Helper()
	spec := genprovider.Descriptors()[provider].Collecte
	spec.Provider = provider
	spec.BaseURL = srv.URL
	vars := map[string]string{"region": "fr-par", "zone": "fr-par-1", "org": "org-test", "host": "outscale.com"}
	inv, err := collect.Collect(context.Background(), srv.Client(), spec, nil, vars)
	if err != nil {
		t.Fatalf("collecte : %v", err)
	}
	out := map[string]any{"provider": provider, "resources": inv.Resources}
	if !inv.Collection.Empty() {
		out["collection"] = inv.Collection
	}
	return writeJSON(t, filepath.Join(t.TempDir(), "inventory.json"), out)
}

// TestNoDegradationEverProducesAPass parcourt les modes de dégradation d'une
// collecte live et vérifie, pour CHAQUE contrôle du type touché, qu'aucun ne
// conclut à la conformité.
func TestNoDegradationEverProducesAPass(t *testing.T) {
	obligationsOf(t) // charge les descripteurs
	bin := buildPepin(t)

	const sgPath = "/instance/v1/zones/fr-par-1/security_groups"
	const rulesPath = "/instance/v1/zones/fr-par-1/security_groups/sg-web/rules"

	cases := []struct {
		mode    string
		handler http.HandlerFunc
	}{
		{
			// Droits insuffisants : l'API refuse la liste des groupes de sécurité.
			mode: "403 sur la liste parente",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == sgPath {
					w.WriteHeader(http.StatusForbidden)
					_, _ = w.Write([]byte(`{"message":"insufficient permissions"}`))
					return
				}
				w.WriteHeader(http.StatusNotFound)
			},
		},
		{
			// La liste parente répond, la jointure enfant est refusée : le cas le plus
			// trompeur, parce que le tenant a bien des groupes de sécurité et que
			// l'inventaire de règles est vide sans être vrai.
			mode: "403 sur la jointure enfant",
			handler: func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case sgPath:
					_, _ = w.Write([]byte(`{"security_groups":[{"id":"sg-web","name":"web"}]}`))
				case rulesPath:
					w.WriteHeader(http.StatusForbidden)
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			},
		},
		{
			// Réponse partielle : la première page passe, la suivante est refusée.
			mode: "reponse partielle (page 2 refusee)",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != sgPath {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				if r.URL.Query().Get("page") != "1" {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				// Une page PLEINE (la spec demande 100) : le moteur en redemandera une.
				var sgs []string
				for i := 0; i < 100; i++ {
					sgs = append(sgs, `{"id":"sg-web","name":"web"}`)
				}
				_, _ = w.Write([]byte(`{"security_groups":[` + strings.Join(sgs, ",") + `]}`))
			},
		},
		{
			// Service indisponible : ni refus ni réponse, une erreur de service.
			mode: "service indisponible (500)",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
			},
		},
		{
			// Réponse illisible : un portail d'authentification interposé rend du HTML.
			// C'est la voie la plus courte vers un faux vert si « illisible » est
			// confondu avec « aucune ressource ».
			mode: "reponse illisible (HTML au lieu de JSON)",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("<html><body>SSO login</body></html>"))
			},
		},
	}

	for _, c := range cases {
		t.Run(c.mode, func(t *testing.T) {
			srv := httptest.NewServer(c.handler)
			defer srv.Close()
			path := liveInventory(t, "scaleway", srv)
			assertNoPass(t, c.mode, statusesOfType(t, bin, "scaleway", "security_group_rule", path))
		})
	}
}

// TestAnUnknownAfterApplyAttributeNeverProducesAPass : sur un plan, un attribut
// encore inconnu est simplement ABSENT de `planned_values`. Le fabriquer à une
// valeur vide franchirait la garde de capacité et conclurait sur du vide — c'est
// la configuration la plus COURANTE qui produit ce cas, celle d'une ressource
// dont un attribut est calculé par le même plan.
func TestAnUnknownAfterApplyAttributeNeverProducesAPass(t *testing.T) {
	obligationsOf(t)
	bin := buildPepin(t)
	plan := map[string]any{
		"format_version": "1.2",
		"planned_values": map[string]any{
			"root_module": map[string]any{
				"resources": []any{
					map[string]any{
						"address": "scaleway_instance_server.web",
						"type":    "scaleway_instance_server",
						"name":    "web",
						// `security_group` et `user_data` sont « known after apply » : ils
						// n'apparaissent pas. `tags` non plus.
						"values": map[string]any{"id": "srv-1", "name": "web"},
					},
				},
			},
		},
	}
	path := writeJSON(t, filepath.Join(t.TempDir(), "plan.json"), plan)
	assertNoPass(t, "unknown after apply",
		statusesOfType(t, bin, "scaleway", "compute_instance", "--terraform", path))
}

// TestAnUnknownResourceTypeIsNeverSilentlyIgnored : un type que le plan porte et
// qu'aucune spec ne projette est ENREGISTRÉ. Pépin ne prétend pas l'auditer, mais
// il ne fait pas semblant de ne pas l'avoir vu.
func TestAnUnknownResourceTypeIsNeverSilentlyIgnored(t *testing.T) {
	obligationsOf(t)
	bin := buildPepin(t)
	plan := map[string]any{
		"format_version": "1.2",
		"planned_values": map[string]any{
			"root_module": map[string]any{
				"resources": []any{
					map[string]any{
						"address": "scaleway_lb.front",
						"type":    "scaleway_lb",
						"name":    "front",
						"values":  map[string]any{"id": "lb-1", "name": "front"},
					},
				},
			},
		},
	}
	path := writeJSON(t, filepath.Join(t.TempDir(), "plan.json"), plan)

	cmd := exec.Command(bin, "scan", "scaleway", "--terraform", path, "--format", "json") // #nosec G204 -- binaire compilé par le test.
	cmd.Dir = repoRoot
	cmd.Env = []string{"NO_COLOR=1", "TERM=dumb", "PEPIN_LANG=en", "HOME=" + t.TempDir()}
	var out, errb strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &errb
	_ = cmd.Run()

	var doc struct {
		Collection struct {
			Unmapped []struct {
				Type  string `json:"type"`
				Count int    `json:"count"`
			} `json:"unmapped"`
		} `json:"collection"`
	}
	if err := json.Unmarshal([]byte(out.String()), &doc); err != nil {
		t.Fatalf("sortie JSON illisible : %v\n%s", err, out.String())
	}
	found := false
	for _, u := range doc.Collection.Unmapped {
		if u.Type == "scaleway_lb" && u.Count == 1 {
			found = true
		}
	}
	if !found {
		t.Errorf("le type non projeté doit être enregistré, obtenu %+v", doc.Collection.Unmapped)
	}
	if !strings.Contains(errb.String(), "scaleway_lb") {
		t.Errorf("il doit aussi être DIT, pas seulement enregistré :\n%s", errb.String())
	}
}

// TestTheDegradationSuiteCoversEveryControlOfItsType : la porte anti-témoin. Une
// suite de dégradation qui ne regarderait qu'un contrôle laisserait les autres
// libres de conclure ; on vérifie donc que le type choisi en porte plusieurs.
func TestTheDegradationSuiteCoversEveryControlOfItsType(t *testing.T) {
	n := 0
	for code := range referentiel.All() {
		if genprovider.ControlType(code) == "security_group_rule" {
			n++
		}
	}
	if n < 2 {
		t.Fatalf("le type security_group_rule ne porte que %d contrôle(s) : la garantie « jamais pass » ne serait vérifiée que sur un témoin", n)
	}
}

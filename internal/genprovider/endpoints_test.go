package genprovider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"

	yaml "go.yaml.in/yaml/v3"

	"github.com/stephrobert/pepin/internal/collect"
)

// LA PORTE « DÉCLARÉ ≠ APPELÉ », adossée à un ENREGISTREMENT.
//
// Un descripteur DÉCLARE des endpoints ; rien, jusqu'ici, ne prouvait que le
// collecteur les ÉMET. C'est la forme exacte de l'incident fondateur des
// politiques EIM inline : la règle était juste, la donnée n'arrivait jamais, et
// aucun test Rego ne pouvait le voir. Un endpoint déclaré et jamais appelé est
// un contrôle qui ne mesure rien, et qui conclut sur du vide.
//
// # Pourquoi un enregistrement, et pas un serveur écrit pour l'occasion
//
// La tentation est d'opposer au collecteur un serveur qui répond « ce que la
// spec attend ». Cette porte-là ne mesure rien : elle se construit depuis la
// spec qu'elle prétend éprouver, donc un `items:` faux la ferait servir le
// tableau faux, et elle resterait verte. Une panne qui se mesure elle-même
// mesure sa propre panne.
//
// Les réponses rejouées ici viennent donc d'AILLEURS : d'une session de collecte
// réellement enregistrée (testdata/transcripts/, issue #84). Le collecteur les
// reçoit sans qu'aucune ait été écrite en fonction de lui. Un `items:` qui ne
// nomme pas le tableau de la réponse fait alors rendre zéro item à la liste
// parente, l'enfant n'est plus appelé, et la porte rougit.
//
// # Ce que cette porte NE prouve PAS
//
// L'enregistrement a été fait contre un ÉMULATEUR LOCAL, jamais contre le
// fournisseur : ce dépôt ne détient aucun identifiant cloud. Il établit ce que
// Pépin FAIT, à savoir les endpoints émis, les jointures qui tirent et les
// variables qui se substituent. Il n'établit rien de ce que le fournisseur RÉPOND : ni les
// noms et types exacts des champs, ni les bornes réelles de pagination, ni le
// comportement de limitation de débit. Cela reste dû à un scan réel.

const transcriptDir = "testdata/transcripts"

// transcriptManifest décrit ce qu'une transcription est, et ce qu'elle n'a pas
// exercé. Un enregistrement sans son manifeste serait un fichier qu'on croit
// exhaustif.
type transcriptManifest struct {
	Provider    string            `yaml:"provider"`
	Outil       string            `yaml:"outil"`
	Enregistre  string            `yaml:"enregistre_le"`
	APIHost     string            `yaml:"hote"`
	Vars        map[string]string `yaml:"vars"`
	NonObserves []struct {
		Endpoint string `yaml:"endpoint"`
		Raison   string `yaml:"raison"`
	} `yaml:"non_observes"`
}

// exchange est une ligne de transcription, réduite à ce que le rejeu emploie.
type exchange struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Host   string `json:"host"`
	Status int    `json:"status"`
	Res    struct {
		Body json.RawMessage `json:"body"`
	} `json:"res"`
}

// loadTranscript lit un manifeste et la transcription qu'il décrit.
func loadTranscript(t *testing.T, provider string) (transcriptManifest, []exchange) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(transcriptDir, provider+".yaml"))
	if err != nil {
		t.Fatalf("manifeste de transcription de %s : %v", provider, err)
	}
	var m transcriptManifest
	if uerr := yaml.Unmarshal(raw, &m); uerr != nil {
		t.Fatalf("manifeste de %s illisible : %v", provider, uerr)
	}
	if m.APIHost == "" || len(m.Vars) == 0 {
		t.Fatalf("manifeste de %s : `hote` et `vars` sont ce qui rend la transcription rejouable", provider)
	}
	lines, err := os.ReadFile(filepath.Join(transcriptDir, provider+".jsonl"))
	if err != nil {
		t.Fatalf("transcription de %s : %v", provider, err)
	}
	var out []exchange
	for _, l := range strings.Split(string(lines), "\n") {
		if strings.TrimSpace(l) == "" {
			continue
		}
		var e exchange
		if jerr := json.Unmarshal([]byte(l), &e); jerr != nil {
			t.Fatalf("transcription de %s : ligne illisible : %v", provider, jerr)
		}
		out = append(out, e)
	}
	if len(out) == 0 {
		t.Fatalf("transcription de %s vide : la porte ne mesure plus rien", provider)
	}
	return m, out
}

// declaredEndpoints rend les endpoints que la spec `collecte` annonce, listes
// parentes comprises, chacun avec son libellé.
func declaredEndpoints(spec collect.Spec) map[string]string {
	out := map[string]string{}
	for _, r := range spec.Resources {
		if fe := r.ForEach; fe != nil {
			out[strip(fe.Path)] = r.Type + " / liste parente"
			out[strip(r.Path)] = r.Type + " / enfant de la jointure sur " + strip(fe.Path)
			continue
		}
		out[strip(r.Path)] = r.Type
	}
	return out
}

// strip retire la partie requête d'un chemin déclaré : c'est l'endpoint qui
// atteste la donnée, pas ses paramètres de page.
func strip(p string) string { return strings.SplitN(p, "?", 2)[0] }

// TestTheRecordedCollectionStillHappens rejoue une session enregistrée : le
// collecteur d'aujourd'hui doit émettre exactement les appels que celui du jour
// de l'enregistrement a émis, ni moins (une donnée cesserait d'arriver), ni plus
// (un appel que rien n'a jamais observé).
func TestTheRecordedCollectionStillHappens(t *testing.T) {
	for name, desc := range loadAllDescriptors(t) {
		if len(desc.Collecte.Resources) == 0 {
			continue
		}
		if _, err := os.Stat(filepath.Join(transcriptDir, name+".yaml")); err != nil {
			// Un fournisseur sans transcription n'est pas une porte au vert : c'est
			// une mesure qui n'a pas été faite, et elle se dit.
			t.Logf("%s : aucune transcription, donc le câblage de sa spec `collecte` n'est PAS mesuré (docs/guides/tracing-api-calls.md)", name)
			continue
		}
		t.Run(name, func(t *testing.T) {
			m, exchanges := loadTranscript(t, name)

			// Les réponses à rejouer, indexées par verbe+chemin. La transcription
			// peut porter plusieurs scans : la première réponse d'un endpoint fait foi.
			replay := map[string]exchange{}
			var recorded []string
			for _, e := range exchanges {
				if e.Host != m.APIHost {
					continue // stockage objet, Kubernetes managé : collecteurs Go, hors spec
				}
				k := e.Method + " " + e.Path
				if _, dup := replay[k]; dup {
					continue
				}
				replay[k] = e
				recorded = append(recorded, k)
			}
			if len(recorded) == 0 {
				t.Fatalf("aucun échange sur %s : `hote` ne correspond pas à la transcription", m.APIHost)
			}
			sort.Strings(recorded)

			var mu sync.Mutex
			got := map[string]bool{}
			var unknown []string

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, io.LimitReader(r.Body, 1<<20))
				k := r.Method + " " + r.URL.Path
				mu.Lock()
				defer mu.Unlock()
				e, known := replay[k]
				if !known {
					unknown = append(unknown, k)
					w.WriteHeader(http.StatusNotFound)
					_, _ = w.Write([]byte(`{"error":"cet appel n'est pas dans la transcription"}`))
					return
				}
				got[k] = true
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(e.Status)
				if len(e.Res.Body) > 0 {
					_, _ = w.Write(e.Res.Body)
				}
			}))
			defer srv.Close()

			spec := desc.Collecte
			spec.Provider = name
			spec.BaseURL = srv.URL + basePath(desc.Collecte.BaseURL)
			vars := map[string]string{}
			for k, v := range m.Vars {
				vars[k] = v
			}
			// Une erreur de collecte n'est pas un échec : la transcription contient de
			// vrais refus (404, 501) et ils doivent être rejoués tels quels. Ce que
			// cette porte mesure est ce qui a été DEMANDÉ.
			_, _ = collect.Collect(context.Background(), srv.Client(), spec, nil, vars)

			for _, k := range recorded {
				if !got[k] {
					t.Errorf("appel ENREGISTRÉ que le collecteur n'émet plus : %s\n"+
						"  Une session réelle l'a émis le %s ; la spec d'aujourd'hui ne l'émet pas.\n"+
						"  Une donnée a cessé d'arriver, et aucun test de règle ne peut le voir.\n"+
						"  Vérifier `items:` de la liste parente, la substitution des variables de\n"+
						"  chemin, et le verbe HTTP.", k, m.Enregistre)
				}
			}
			if len(unknown) > 0 {
				sort.Strings(unknown)
				t.Errorf("appel(s) que la transcription ne connaît pas : %s\n"+
					"  Le collecteur atteint un endpoint qu'aucune session enregistrée n'a jamais\n"+
					"  vu : il est DÉCLARÉ mais pas MESURÉ. Ré-enregistrer (mise run trace) et\n"+
					"  relire la transcription avant de la committer.", strings.Join(dedupe(unknown), ", "))
			}
		})
	}
}

// TestEveryDeclaredEndpointIsObservedOrDeclaredUnobserved : le registre des
// endpoints que la session n'a PAS exercés est EXACT.
//
// Porte dans les deux sens, et les deux comptent. Un endpoint déclaré qui
// n'apparaît nulle part, ni dans la transcription ni dans le registre, est
// une couverture qu'on s'accorde sans l'avoir mesurée. Une ligne de registre qui
// ne correspond plus à rien est une dette payée qu'on n'a pas rayée, et un
// registre qui surestime la dette est aussi faux que celui qui la sous-estime.
func TestEveryDeclaredEndpointIsObservedOrDeclaredUnobserved(t *testing.T) {
	for name, desc := range loadAllDescriptors(t) {
		if len(desc.Collecte.Resources) == 0 {
			continue
		}
		if _, err := os.Stat(filepath.Join(transcriptDir, name+".yaml")); err != nil {
			continue
		}
		t.Run(name, func(t *testing.T) {
			m, exchanges := loadTranscript(t, name)
			base := basePath(desc.Collecte.BaseURL)

			seen := map[string]bool{}
			for _, e := range exchanges {
				if e.Host == m.APIHost {
					seen[strings.TrimPrefix(e.Path, base)] = true
				}
			}

			declaredUnobserved := map[string]string{}
			for _, u := range m.NonObserves {
				declaredUnobserved[u.Endpoint] = u.Raison
				if strings.TrimSpace(u.Raison) == "" {
					t.Errorf("%s : l'endpoint non observé %q n'a pas de raison écrite. Un trou dont on ignore la cause est un trou que personne ne bouche.", name, u.Endpoint)
				}
			}

			computed := map[string]bool{}
			for ep, label := range declaredEndpoints(desc.Collecte) {
				if matchesRecorded(ep, seen) {
					continue
				}
				computed[ep] = true
				if _, declared := declaredUnobserved[ep]; !declared {
					t.Errorf("endpoint DÉCLARÉ, jamais observé, et absent du registre : %s (%s)\n"+
						"  providers/%s.yaml l'annonce ; aucune session enregistrée ne l'a vu passer.\n"+
						"  Soit la session l'exerce (ré-enregistrer), soit le registre `non_observes`\n"+
						"  de %s/%s.yaml dit pourquoi elle ne le peut pas.",
						ep, label, name, transcriptDir, name)
				}
			}
			for ep := range declaredUnobserved {
				if !computed[ep] {
					t.Errorf("registre `non_observes` faux : %s y figure alors que la transcription l'a bien vu, ou qu'il n'est plus déclaré.\n"+
						"  Retirer la ligne de %s/%s.yaml.", ep, transcriptDir, name)
				}
			}
		})
	}
}

// matchesRecorded dit si un endpoint déclaré correspond à un chemin enregistré.
// Les `{…}` d'un chemin acceptent tout segment : leur valeur vient des
// identifiants ou de la liste parente, jamais de la spec.
func matchesRecorded(declared string, seen map[string]bool) bool {
	if seen[declared] {
		return true
	}
	if !strings.Contains(declared, "{") {
		return false
	}
	var b strings.Builder
	b.WriteString(`^`)
	parts := regexp.MustCompile(`\{[^}]*\}`).Split(declared, -1)
	for i, part := range parts {
		b.WriteString(regexp.QuoteMeta(part))
		if i < len(parts)-1 {
			b.WriteString(`[^/]+`)
		}
	}
	b.WriteString(`$`)
	re := regexp.MustCompile(b.String())
	for p := range seen {
		if re.MatchString(p) {
			return true
		}
	}
	return false
}

// basePath rend le préfixe de chemin d'un base_url (`/api/v1`, `/v2`), qui fait
// partie de l'endpoint réellement émis et donc du chemin enregistré.
func basePath(baseURL string) string {
	i := strings.Index(baseURL, "://")
	if i < 0 {
		return ""
	}
	rest := baseURL[i+3:]
	j := strings.Index(rest, "/")
	if j < 0 {
		return ""
	}
	return strings.TrimSuffix(rest[j:], "/")
}

// dedupe rend une liste sans doublon, l'ordre conservé.
func dedupe(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

package genprovider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stephrobert/pepin/internal/collect"
	"github.com/stephrobert/pepin/internal/provider"
	"github.com/stephrobert/pepin/internal/tfmap"
)

const providersDir = "../../providers"

// TestProvidersValid : chaque providers/<nom>.yaml respecte le CONTRAT
// (genprovider.Validate) — identité, auth, identifiants, sources non vides.
func TestProvidersValid(t *testing.T) {
	res, err := ValidateAll(os.DirFS(providersDir), ".")
	if err != nil {
		t.Fatalf("validation des providers : %v", err)
	}
	if len(res) == 0 {
		t.Fatal("aucun provider trouvé")
	}
	for name, errs := range res {
		for _, e := range errs {
			t.Errorf("%s : %s", name, e)
		}
	}
}

// TestProviderMappingsMatchSchema ANCRE le mapping Terraform de chaque provider
// sur le schéma réel du provider Terraform (anti-invention §2). Ignoré par
// provider si terraform/le schéma ne sont pas disponibles.
func TestProviderMappingsMatchSchema(t *testing.T) {
	entries, err := os.ReadDir(providersDir)
	if err != nil {
		t.Skipf("dossier providers indisponible : %v", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".yaml")
		desc, err := Load(os.DirFS(providersDir), e.Name())
		if err != nil {
			t.Errorf("%s : %v", name, err)
			continue
		}
		missing, ok := tfmap.CheckSchema(filepath.Join("../../examples", name, "terraform"), desc.MappingTerraform)
		if !ok {
			continue
		}
		for _, m := range missing {
			t.Errorf("%s : %s", name, m)
		}
	}
}

// TestContractVerifiedTypesAreCollected — invariant de cohérence (audit A.3) : un type de
// ressource déclaré `verifie` au contrat DOIT être réellement collecté (live) ou mappé
// (Terraform), sinon les règles qui le consomment sont MORTES (chargées mais sans donnée).
func TestContractVerifiedTypesAreCollected(t *testing.T) {
	entries, err := os.ReadDir(providersDir)
	if err != nil {
		t.Skipf("dossier providers indisponible : %v", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".yaml")
		desc, err := Load(os.DirFS(providersDir), e.Name())
		if err != nil {
			t.Errorf("%s : %v", name, err)
			continue
		}
		collected := map[string]bool{}
		for _, r := range desc.Collecte.Resources {
			collected[r.Type] = true
		}
		for _, r := range desc.MappingTerraform.Resources {
			collected[r.Type] = true
		}
		// Le stockage objet est collecté par le collecteur S3 partagé (desc.S3), hors spec `collecte`.
		if desc.S3.Endpoint != "" {
			collected["object_storage_bucket"] = true
		}
		// Les clusters Kubernetes managés sont collectés par le collecteur Go OKS (desc.OKS).
		if desc.OKS.Endpoint != "" {
			collected["kubernetes_cluster"] = true
		}
		for typ, tc := range desc.Contrat.Types {
			if tc.Etat == "verifie" && !collected[typ] {
				t.Errorf("%s : type %q déclaré `verifie` au contrat mais NI collecté NI mappé (règle orpheline) — le collecter, ou passer à `a_verifier`/`absent`", name, typ)
			}
		}
	}
}

// TestEveryContractJustificationIsBilingual : toute justification de
// non-applicabilité porte sa traduction anglaise.
//
// Ces justifications ne sont pas de la décoration : ce sont elles qu'un auditeur
// lit dans `--format assessment`, dans l'OSCAL et dans le bundle scellé, en face
// d'un `not-applicable`. Un N/A sans motif n'est pas opposable ; un N/A dont le
// motif bascule au français dans un rapport anglais ne l'est pas davantage pour
// l'auditeur qui ne lit pas le français.
//
// Le contrôle porte aussi sur l'ABSENCE d'accent dans la version anglaise :
// c'est le même critère que pour le référentiel et les règles, appliqué à la
// troisième source de prose du produit.
func TestEveryContractJustificationIsBilingual(t *testing.T) {
	entries, err := os.ReadDir(providersDir)
	if err != nil {
		t.Fatalf("dossier providers indisponible : %v", err)
	}
	check := func(t *testing.T, where, fr, en string) {
		t.Helper()
		if strings.TrimSpace(fr) == "" {
			return // pas de justification française : rien à traduire (repli générique, lui traduit dans le code)
		}
		if strings.TrimSpace(en) == "" {
			t.Errorf("%s : `reason` sans `reason_en` — un rapport anglais afficherait cette justification en français", where)
			return
		}
		if en == fr {
			t.Errorf("%s : `reason_en` est identique au français — la traduction n'a pas été faite", where)
		}
		for _, r := range en {
			if r > 127 && !strings.ContainsRune("—…«»·≥≠→⚠✓", r) {
				t.Errorf("%s : `reason_en` porte le caractère non ASCII %q", where, r)
				break
			}
		}
	}
	seen := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".yaml")
		d, lerr := Load(os.DirFS(providersDir), e.Name())
		if lerr != nil {
			t.Errorf("%s : %v", name, lerr)
			continue
		}
		for _, na := range d.Contrat.NonApplicable {
			check(t, name+"/non_applicable/"+na.Control, na.Reason, na.ReasonEn)
			seen++
		}
		for typ, tc := range d.Contrat.Types {
			if tc.Etat != "absent" {
				continue
			}
			check(t, name+"/types/"+typ, tc.Reason, tc.ReasonEn)
		}
	}
	if seen == 0 {
		t.Fatal("aucune justification de non-applicabilité lue : le test ne mesurerait rien")
	}
}

func TestCredentialFileSecretsAreNotProjectedIntoInventory(t *testing.T) {
	const secret = "fake-credential-marker"

	tests := []struct {
		name, format, pathEnv, contents string
	}{
		{
			name:    "scaleway",
			format:  "scw",
			pathEnv: "SCW_CONFIG_PATH",
			contents: `access_key: fake-access-key
secret_key: fake-credential-marker
`,
		},
		{
			name:     "outscale",
			format:   "osc",
			pathEnv:  "OSC_CONFIG_FILE",
			contents: `{"default":{"access_key":"fake-access-key","secret_key":"fake-credential-marker"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			credentialsPath := filepath.Join(t.TempDir(), "credentials")
			if err := os.WriteFile(credentialsPath, []byte(tt.contents), 0o600); err != nil {
				t.Fatalf("écriture des identifiants de test : %v", err)
			}
			t.Setenv(tt.pathEnv, credentialsPath)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("X-Auth-Token") != secret {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				_, _ = w.Write([]byte(`{"items":[{"id":"safe-resource","secret_key":"` + secret + `"}]}`))
			}))
			defer server.Close()

			g := GenericProvider{desc: Descriptor{Name: "test"}}
			g.desc.Auth.Type = "header"
			g.desc.Auth.Header = "X-Auth-Token"
			g.desc.Auth.Value = "{secret_key}"
			g.desc.Credentials.File = credFile{Format: tt.format}
			g.desc.Collecte = collect.Spec{
				BaseURL: server.URL,
				Resources: []collect.ResourceSpec{{
					Type:  "test_resource",
					Path:  "/resources",
					Items: "items",
					ID:    "id",
					Map:   map[string]string{"id": "id"},
				}},
			}

			inv, err := g.Collect(context.Background(), provider.Config{})
			if err != nil {
				t.Fatalf("collecte de test : %v", err)
			}
			if len(inv.Resources) != 1 || inv.Resources[0].ID != "safe-resource" {
				t.Fatalf("la collecte authentifiée n'a pas produit la ressource attendue : %+v", inv.Resources)
			}

			encoded, err := json.Marshal(inv)
			if err != nil {
				t.Fatalf("sérialisation de l'inventaire : %v", err)
			}
			if strings.Contains(string(encoded), secret) {
				t.Fatal("le secret d'authentification est entré dans l'inventaire normalisé")
			}
		})
	}
}

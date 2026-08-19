package genprovider

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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

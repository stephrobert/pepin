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

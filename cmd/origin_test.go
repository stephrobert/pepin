package cmd

// L'origine Terraform, mesurée sur le PRODUIT : on compile le binaire, on scanne
// le plan d'exemple du dépôt — dont les `.tf` sont à côté — et on lit ce que les
// formats analysables portent.
//
// La contrepartie est aussi importante que le cas nominal : sur une source qui
// n'est pas un plan, l'origine doit être ABSENTE, proprement, sans rien casser.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestAPlanFindingCarriesItsFileAndLine : le cas nominal, et la vérification qui
// compte — la ligne désignée porte réellement l'en-tête du bloc. Un test qui se
// contenterait de constater « une ligne est présente » accepterait n'importe quel
// nombre, c'est-à-dire exactement le défaut qu'il devrait attraper.
func TestAPlanFindingCarriesItsFileAndLine(t *testing.T) {
	bin := buildPepin(t)
	stdout, _, _ := runScan(t, bin, "scan", "scaleway", "-t", exemptFixture, "--format", "json")

	var doc struct {
		Findings []struct {
			Subject string            `json:"subject"`
			Labels  map[string]string `json:"labels"`
		} `json:"findings"`
	}
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("sortie JSON illisible : %v", err)
	}
	if len(doc.Findings) == 0 {
		t.Fatal("la fixture doit produire des findings")
	}

	planDir := filepath.Dir(filepath.Join(repoRoot, exemptFixture))
	located := 0
	for _, f := range doc.Findings {
		file := f.Labels["tf_file"]
		if file == "" {
			continue
		}
		located++
		line, err := strconv.Atoi(f.Labels["tf_line"])
		if err != nil || line <= 0 {
			t.Errorf("%s : ligne illisible (%q)", f.Subject, f.Labels["tf_line"])
			continue
		}
		raw, rerr := os.ReadFile(filepath.Join(planDir, file)) // #nosec G304 -- fixture du dépôt.
		if rerr != nil {
			t.Errorf("%s : le fichier désigné est illisible : %v", f.Subject, rerr)
			continue
		}
		lines := strings.Split(string(raw), "\n")
		if line > len(lines) {
			t.Errorf("%s : ligne %d hors du fichier %s (%d lignes)", f.Subject, line, file, len(lines))
			continue
		}
		if !strings.HasPrefix(strings.TrimSpace(lines[line-1]), "resource ") {
			t.Errorf("%s : la ligne %d de %s ne porte pas un en-tête de bloc : %q",
				f.Subject, line, file, lines[line-1])
		}
	}
	if located == 0 {
		t.Fatal("aucun finding du plan ne porte son origine — le mécanisme ne mesure rien")
	}
}

// TestASARIFResultIsAnnotatedOnTheGuiltyLine : l'exigence de l'issue #67 pour les
// formats analysables. C'est cette `region` qui fait qu'une forge annote la ligne
// fautive plutôt que le fichier de plan.
func TestASARIFResultIsAnnotatedOnTheGuiltyLine(t *testing.T) {
	bin := buildPepin(t)
	stdout, _, _ := runScan(t, bin, "scan", "scaleway", "-t", exemptFixture, "--format", "sarif")

	var doc struct {
		Runs []struct {
			Results []struct {
				RuleID    string `json:"ruleId"`
				Locations []struct {
					PhysicalLocation struct {
						ArtifactLocation struct {
							URI string `json:"uri"`
						} `json:"artifactLocation"`
						Region struct {
							StartLine int `json:"startLine"`
						} `json:"region"`
					} `json:"physicalLocation"`
				} `json:"locations"`
			} `json:"results"`
		} `json:"runs"`
	}
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("SARIF illisible : %v", err)
	}
	if len(doc.Runs) != 1 || len(doc.Runs[0].Results) == 0 {
		t.Fatal("le SARIF doit porter une exécution et des résultats")
	}
	annotated := 0
	for _, r := range doc.Runs[0].Results {
		if len(r.Locations) != 1 {
			t.Errorf("%s : %d localisations, attendu 1", r.RuleID, len(r.Locations))
			continue
		}
		loc := r.Locations[0].PhysicalLocation
		if strings.HasSuffix(loc.ArtifactLocation.URI, ".tf") && loc.Region.StartLine > 0 {
			annotated++
		}
	}
	if annotated == 0 {
		t.Fatalf("aucun résultat SARIF n'est annoté sur une ligne de `.tf` :\n%s", stdout)
	}
}

// TestALiveOrExportedInventoryCarriesNoFabricatedOrigin : la réserve explicite de
// l'issue. Sur une source qui n'est pas un plan, l'information n'existe pas ; son
// absence ne doit ni être comblée, ni faire échouer quoi que ce soit.
func TestALiveOrExportedInventoryCarriesNoFabricatedOrigin(t *testing.T) {
	bin := buildPepin(t)
	stdout, _, code := runScan(t, bin, "scan", "scaleway", "examples/scaleway/inventory.json", "--format", "json")
	if code == exitErreur {
		t.Fatalf("un inventaire sans origine ne doit rien faire échouer :\n%s", stdout)
	}

	var doc struct {
		Findings []struct {
			Subject string            `json:"subject"`
			Labels  map[string]string `json:"labels"`
		} `json:"findings"`
	}
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("sortie JSON illisible : %v", err)
	}
	if len(doc.Findings) == 0 {
		t.Fatal("la fixture doit produire des findings")
	}
	for _, f := range doc.Findings {
		for _, label := range []string{"tf_file", "tf_line", "tf_module"} {
			if v, present := f.Labels[label]; present {
				t.Errorf("%s : le label %s ne devrait pas exister hors plan Terraform (valeur %q)",
					f.Subject, label, v)
			}
		}
	}
}

// TestTheSARIFOfANonPlanScanStaysWellFormed : la passe qui injecte la
// localisation ne doit rien casser quand il n'y a rien à injecter.
func TestTheSARIFOfANonPlanScanStaysWellFormed(t *testing.T) {
	bin := buildPepin(t)
	stdout, _, _ := runScan(t, bin, "scan", "scaleway", "examples/scaleway/inventory.json", "--format", "sarif")
	var doc map[string]any
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("SARIF illisible : %v", err)
	}
	if doc["version"] != "2.1.0" {
		t.Errorf("version SARIF %v, attendu 2.1.0", doc["version"])
	}
	runs, _ := doc["runs"].([]any)
	if len(runs) != 1 {
		t.Fatalf("%d exécutions, attendu 1", len(runs))
	}
}

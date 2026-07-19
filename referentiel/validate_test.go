package referentiel

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	yaml "go.yaml.in/yaml/v3"
)

// rulesDir : règles communes (relatif au package referentiel).
const rulesDir = "../internal/commonrules/rules"

// frameworkExigences : index SCSL (dépôt frère, comme ../scankit). Absent =
// vérification SCSL ignorée (environnement sans le framework).
const frameworkExigences = "../../framework-scsl/api/v1/exigences.json"

// readRules concatène le texte de toutes les règles .rego (hors tests).
func readRules(t *testing.T) string {
	t.Helper()
	entries, err := os.ReadDir(rulesDir)
	if err != nil {
		t.Fatalf("lecture de %s : %v", rulesDir, err)
	}
	var sb strings.Builder
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".rego") || strings.HasSuffix(e.Name(), "_test.rego") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(rulesDir, e.Name()))
		if err != nil {
			t.Fatalf("lecture de %s : %v", e.Name(), err)
		}
		sb.Write(b)
		sb.WriteByte('\n')
	}
	return sb.String()
}

// TestActiveControlsHaveRule : tout contrôle actif (au moins un fournisseur) doit
// être émis par une règle (le code apparaît en littéral dans un .rego commun).
func TestActiveControlsHaveRule(t *testing.T) {
	rules := readRules(t)
	for code, ctl := range byCode {
		if len(ctl.Fournisseurs) == 0 {
			continue
		}
		if !strings.Contains(rules, `"`+code+`"`) {
			t.Errorf("contrôle actif %q (fournisseurs %v) : aucune règle ne l'émet", code, ctl.Fournisseurs)
		}
	}
}

// TestRuleCodesAreControlled : tout code émis par une règle doit exister dans
// controles.yaml (pas de finding orphelin sans contrôle/SCSL).
func TestRuleCodesAreControlled(t *testing.T) {
	rules := readRules(t)
	// `code` peut être passé via "code": "X" OU en argument littéral d'un
	// constructeur de finding : on extrait tout littéral ressemblant à un code.
	re := regexp.MustCompile(`"([a-z][a-z0-9]+(?:_[a-z0-9]+){2,})"`)
	for _, m := range re.FindAllStringSubmatch(rules, -1) {
		cand := m[1]
		// Filtrer : un code de finding a un préfixe de service connu.
		if !hasServicePrefix(cand) {
			continue
		}
		if _, ok := byCode[cand]; !ok {
			t.Errorf("règle émet le code %q absent de controles.yaml", cand)
		}
	}
}

func hasServicePrefix(s string) bool {
	for _, p := range []string{"network_", "compute_", "objectstorage_", "blockstorage_", "iam_", "kubernetes_", "loadbalancer_", "governance_"} {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

// TestSCSLReferencesExist : toute exigence scsl (CLD-*) d'un contrôle existe dans
// l'index SCSL du framework (anti-invention, §2 / discipline ancrage-scsl-r).
func TestSCSLReferencesExist(t *testing.T) {
	raw, err := os.ReadFile(frameworkExigences)
	if err != nil {
		t.Skipf("index SCSL indisponible (%s) — vérification ignorée", frameworkExigences)
	}
	known := map[string]bool{}
	for _, m := range regexp.MustCompile(`CLD-[A-Z0-9]+-[0-9]+`).FindAllString(string(raw), -1) {
		known[m] = true
	}
	for code, ctl := range byCode {
		for _, s := range ctl.Scsl {
			if !known[s] {
				t.Errorf("contrôle %q référence une exigence SCSL inconnue : %s", code, s)
			}
		}
	}
}

// TestCatalogueConsistency : tout code marqué `implemente` au catalogue existe
// bien dans controles.yaml.
func TestCatalogueConsistency(t *testing.T) {
	raw, err := os.ReadFile("catalogue.yaml")
	if err != nil {
		t.Fatalf("lecture catalogue.yaml : %v", err)
	}
	var cat struct {
		Catalogue []struct {
			Code   string `yaml:"code"`
			Statut string `yaml:"statut"`
		} `yaml:"catalogue"`
	}
	if err := yaml.Unmarshal(raw, &cat); err != nil {
		t.Fatalf("catalogue.yaml invalide : %v", err)
	}
	for _, e := range cat.Catalogue {
		if e.Statut == "implemente" {
			if _, ok := byCode[e.Code]; !ok {
				t.Errorf("catalogue : %q marqué implemente mais absent de controles.yaml", e.Code)
			}
		}
	}
}

// frameworksDir : catalogues de normes (une vue par norme), relatif au package.
const frameworksDir = "frameworks"

// loadFrameworkIDs charge frameworks/*.yaml et retourne, par code de norme, l'ensemble
// des identifiants d'exigence définis (repris VERBATIM du texte officiel).
func loadFrameworkIDs(t *testing.T) map[string]map[string]bool {
	t.Helper()
	entries, err := os.ReadDir(frameworksDir)
	if err != nil {
		t.Fatalf("lecture de %s : %v", frameworksDir, err)
	}
	out := map[string]map[string]bool{}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(frameworksDir, e.Name()))
		if err != nil {
			t.Fatalf("lecture de %s : %v", e.Name(), err)
		}
		var fw struct {
			Code     string `yaml:"code"`
			Controls []struct {
				ID string `yaml:"id"`
			} `yaml:"controls"`
		}
		if err := yaml.Unmarshal(b, &fw); err != nil {
			t.Fatalf("%s invalide : %v", e.Name(), err)
		}
		ids := map[string]bool{}
		for _, c := range fw.Controls {
			ids[c.ID] = true
		}
		out[fw.Code] = ids
	}
	return out
}

// TestFrameworkReferencesExist : tout identifiant d'exigence mappé par un contrôle
// (frameworks: {norme: [ids]}) doit être défini dans le catalogue de cette norme
// (frameworks/<norme>.yaml). Anti-invention symétrique de TestSCSLReferencesExist :
// interdit qu'un mapping cite un numéro d'exigence qui n'existe pas dans le texte.
func TestFrameworkReferencesExist(t *testing.T) {
	defined := loadFrameworkIDs(t)
	for code, ctl := range byCode {
		for fw, ids := range ctl.Frameworks {
			known, ok := defined[fw]
			if !ok {
				t.Errorf("contrôle %q mappe une norme inconnue (pas de frameworks/%s.yaml) : %s", code, fw, fw)
				continue
			}
			for _, id := range ids {
				if !known[id] {
					t.Errorf("contrôle %q référence %s %q, absent du catalogue frameworks/%s.yaml", code, fw, id, fw)
				}
			}
		}
	}
}

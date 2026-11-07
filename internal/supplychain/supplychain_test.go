package supplychain_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Les deux portes de l'ADR-0016. Elles lisent les workflows comme du texte : c'est
// grossier, et c'est exactement ce qui convient. Un épinglage ou une somme sont des
// propriétés SYNTAXIQUES, et les vérifier sur le texte publié évite d'avoir à faire
// confiance à une abstraction qui pourrait, elle, mentir.

func repoRoot(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..")
}

func workflows(t *testing.T) map[string]string {
	t.Helper()
	dir := filepath.Join(repoRoot(t), ".github", "workflows")
	names, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("lecture des workflows : %v", err)
	}
	out := map[string]string{}
	for _, n := range names {
		if n.IsDir() || !strings.HasSuffix(n.Name(), ".yml") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, n.Name())) //nolint:gosec // chemin du dépôt
		if err != nil {
			t.Fatalf("lecture de %s : %v", n.Name(), err)
		}
		out[n.Name()] = string(b)
	}
	if len(out) == 0 {
		t.Fatal("aucun workflow : la porte ne mesurerait rien")
	}
	return out
}

var usesRe = regexp.MustCompile(`uses:\s*(\S+)`)
var fullSHA = regexp.MustCompile(`@[0-9a-f]{40}$`)

// TestEveryActionIsPinnedToAFullSHA : une étiquette de version se déplace, donc elle
// n'épingle rien. C'est le vecteur le plus documenté de l'écosystème, et le dépôt
// tenait déjà la propriété — 79 sur 79 — sans que rien ne l'empêche de la perdre.
func TestEveryActionIsPinnedToAFullSHA(t *testing.T) {
	var checked int
	for name, body := range workflows(t) {
		for _, m := range usesRe.FindAllStringSubmatch(body, -1) {
			ref := m[1]
			if strings.HasPrefix(ref, "./") {
				continue // action locale du dépôt : rien à épingler
			}
			checked++
			if !fullSHA.MatchString(ref) {
				t.Errorf("%s : %q n'est pas épinglé par un SHA de 40 caractères.\n"+
					"  Une étiquette se déplace : elle ne garantit pas ce qui sera exécuté.", name, ref)
			}
		}
	}
	if checked == 0 {
		t.Fatal("aucune action tierce trouvée : la porte ne mesure rien")
	}
	t.Logf("%d référence(s) d'action vérifiée(s)", checked)
}

// TestEveryPublishedArtefactIsChecksummed est la porte que le SBOM a coûtée.
//
// `checksums.txt` est signé, et une signature sur un condensé de condensés ne couvre
// que ce qui y figure. Un artefact publié hors de ce fichier n'est couvert par
// AUCUN des chemins de vérification documentés — quand bien même il serait attesté
// par ailleurs, ce que l'attestation SBOM fait pour son contenu mais pas pour le
// fichier publié.
//
// La mesure porte sur le texte du workflow : ce qui est téléversé, contre ce qui est
// haché. Toute divergence est une omission, pas une subtilité.
func TestEveryPublishedArtefactIsChecksummed(t *testing.T) {
	body := workflows(t)["release.yml"]
	if body == "" {
		t.Fatal("release.yml introuvable")
	}

	// Les fichiers que la release téléverse, tels que le workflow les nomme.
	published := map[string]bool{}
	for _, m := range regexp.MustCompile(`(?m)^\s+(?:dist/)?([a-z0-9][a-z0-9._-]*\.(?:json|jsonl|txt|bundle))\s*$`).
		FindAllStringSubmatch(body, -1) {
		published[m[1]] = true
	}
	// checksums.txt ne peut pas se contenir lui-même, et le bundle signe checksums.txt.
	delete(published, "checksums.txt")
	delete(published, "checksums.txt.cosign.bundle")
	// La provenance SLSA est une enveloppe DSSE SIGNÉE : elle porte sa propre
	// vérification, et `gh attestation verify` la contrôle sans rien d'autre.
	// L'exiger dans checksums.txt serait un cercle — elle est produite APRÈS les
	// sommes, puisqu'elle atteste les binaires que ces sommes décrivent.
	delete(published, "provenance.intoto.jsonl")

	line := regexp.MustCompile(`sha256sum ([^>]+)> checksums\.txt`).FindStringSubmatch(body)
	if line == nil {
		t.Fatal("aucune commande sha256sum vers checksums.txt : la couverture n'est plus vérifiable")
	}
	hashed := line[1]

	for f := range published {
		if !strings.Contains(hashed, f) {
			t.Errorf("l'artefact %q est publié mais n'entre dans aucune somme de checksums.txt.\n"+
				"  La signature cosign porte sur checksums.txt : ce qui n'y figure pas n'est couvert\n"+
				"  par aucun chemin de vérification documenté (ADR-0016).\n"+
				"  Commande de hachage actuelle : sha256sum %s> checksums.txt", f, hashed)
		}
	}
	t.Logf("artefacts publiés contrôlés : %d", len(published))
}

// installerActions énumère les actions dont le RÔLE est d'installer un outil, avec
// le nom de l'entrée qui en fige la version.
//
// La liste est explicite plutôt que devinée : « cette action installe-t-elle quelque
// chose » n'est pas une question qu'un test peut poser au texte d'un workflow. Une
// action ajoutée ici est une décision ; une action installatrice absente d'ici est
// un trou, et c'est le prix de l'honnêteté de cette porte.
var installerActions = map[string]string{
	"jdx/mise-action":           "version",
	"actions/setup-go":          "go-version-file",
	"actions/setup-python":      "python-version",
	"actions/setup-node":        "node-version",
	"sigstore/cosign-installer": "cosign-release",
	"getplumber/plumber":        "version",
}

// TestEveryInstallerActionPinsItsTool ferme le trou que l'ADR-0016 n'avait pas nommé.
//
// Épingler une action par SHA ne fige QUE l'action. Ce qu'elle télécharge ensuite
// reste choisi par l'amont si personne ne le dit — et le 2026-09-08, `mise-action`
// sans `version:` a résolu vers une étiquette dont les binaires n'étaient pas
// publiés. 404, cinq fois, et tout le dépôt à l'arrêt.
//
// Une chaîne ne vaut que son maillon le moins figé.
func TestEveryInstallerActionPinsItsTool(t *testing.T) {
	var checked int
	for name, body := range workflows(t) {
		lines := strings.Split(body, "\n")
		for i, l := range lines {
			m := usesRe.FindStringSubmatch(l)
			if m == nil {
				continue
			}
			action := strings.SplitN(strings.TrimPrefix(m[1], "./"), "@", 2)[0]
			key, isInstaller := installerActions[action]
			if !isInstaller {
				continue
			}
			checked++
			// La version se déclare dans le bloc `with:` de CETTE étape : on lit
			// jusqu'à la prochaine étape, marquée par un tiret de liste.
			var block []string
			for j := i + 1; j < len(lines); j++ {
				if strings.HasPrefix(strings.TrimSpace(lines[j]), "- ") {
					break
				}
				block = append(block, lines[j])
			}
			if !strings.Contains(strings.Join(block, "\n"), key+":") {
				t.Errorf("%s ligne %d : %q installe un outil sans figer sa version (%q manquant).\n"+
					"  Épingler l'action ne fige que l'action ; ce qu'elle télécharge reste choisi\n"+
					"  par l'amont. Voir ADR-0016 et l'incident mise v2026.9.3.",
					name, i+1, action, key)
			}
		}
	}
	if checked == 0 {
		t.Fatal("aucune action installatrice trouvée : la porte ne mesure rien")
	}
	t.Logf("%d action(s) installatrice(s) vérifiée(s)", checked)
}

func TestReleaseRequiresSuccessfulPlumber(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash is required to exercise the release preflight")
	}
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "tools", "release", "preflight.sh"))
	if err != nil {
		t.Fatal(err)
	}
	script := string(raw)
	start := strings.Index(script, "ci_jobs=")
	if start < 0 {
		t.Fatal("release CI check not found")
	}
	end := strings.Index(script[start:], "\necho\n")
	if end < 0 {
		t.Fatal("release CI check boundary not found")
	}
	check := script[start : start+end]
	for _, tc := range []struct {
		name       string
		plumber    string
		wantPassed bool
	}{
		{"successful", "success", true},
		{"failed", "failure", false},
		{"skipped", "skipped", false},
		{"pending", "null", false},
		{"missing", "", false},
		{"mixed attempts", "failure\nPlumber (trust policy)\tsuccess", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runs := "Build + Test\tsuccess\n"
			if tc.plumber != "" {
				runs += "Plumber (trust policy)\t" + tc.plumber + "\n"
			}
			cmd := exec.Command(bash, "-c", `
set -eu
failures=0
DIM="" OFF=""
ok() { printf '%s\n' "$1"; }
ko() { printf '%s: %s\n' "$1" "$2"; failures=$((failures + 1)); }
git() { if [ "$1" = remote ]; then echo origin; else echo test-sha; fi; }
gh() { printf '%s' "$FIXTURE_CHECK_RUNS"; }
`+check+`
[ "$failures" -eq 0 ]
`)
			cmd.Env = append(os.Environ(), "FIXTURE_CHECK_RUNS="+runs)
			out, err := cmd.CombinedOutput()
			if (err == nil) != tc.wantPassed {
				t.Fatalf("passed=%t, want %t: %v\n%s", err == nil, tc.wantPassed, err, out)
			}
		})
	}
}

func TestPlumberJobKeepsItsTrustBoundaryAndScoreGate(t *testing.T) {
	_, job, found := strings.Cut(workflows(t)["ci.yml"], "\n  plumber:\n")
	if !found {
		t.Fatal("Plumber job missing from CI")
	}
	if next := regexp.MustCompile(`(?m)^  [a-zA-Z0-9_-]+:\s*$`).FindStringIndex(job); next != nil {
		job = job[:next[0]]
	}
	for _, required := range []string{
		"name: Plumber (trust policy)",
		"if: github.event_name == 'push' && github.ref == 'refs/heads/main'",
		"environment: security",
		"persist-credentials: false",
		"secrets.PLUMBER_ADMIN_TOKEN",
		"verify-attestation: 'true'",
		"config-file: .plumber.yaml",
		"min-points: '100'",
		"score-push: 'true'",
	} {
		if !strings.Contains(job, required) {
			t.Errorf("Plumber job is missing %q", required)
		}
	}

	if strings.Contains(job, "continue-on-error:") || strings.Contains(job, "soft-fail:") {
		t.Fatal("Plumber gate must not be weakened")
	}
}

func TestOSVPRScanRejectsIncompleteResults(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash is required to exercise the OSV result guard")
	}
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq is required to exercise the OSV result guard")
	}
	body := workflows(t)["osv-scanner.yml"]
	if !strings.Contains(body, "\n  osv-scan:\n") {
		t.Fatal("OSV job ID must preserve the historical Code Scanning analysis key")
	}
	if strings.Contains(body, "osv-scanner-reusable-pr.yml@") {
		t.Fatal("upstream PR checkout persists credentials; use the local scan steps")
	}
	if !strings.Contains(body, "persist-credentials: false") {
		t.Fatal("OSV PR checkout must not persist credentials")
	}
	if regexp.MustCompile(`(?m)^\s+scan-args:\s+--`).MatchString(body) {
		t.Fatal("OSV Docker actions require newline-separated scan arguments, not one combined argument")
	}
	block := regexp.MustCompile(`(?s)- name: Require two completed scans\n        run: \|\n(.*?)(?:\n      - name:)`).
		FindStringSubmatch(body)
	if block == nil {
		t.Fatal("OSV completed-scan guard missing")
	}
	for _, tc := range []struct {
		name string
		old  string
		new  string
		pass bool
	}{
		{"two valid empty scans", `{"results":[]}`, `{"results":[]}`, true},
		{"missing old scan", "", `{"results":[]}`, false},
		{"missing new scan", `{"results":[]}`, "", false},
		{"malformed scan", `{"results":[]}`, `{`, false},
		{"wrong shape", `{"results":[]}`, `{"results":null}`, false},
		{"unrelated object", `{"results":[]}`, `{}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, data := range map[string]string{
				"old-results.json": tc.old,
				"new-results.json": tc.new,
			} {
				if data != "" {
					if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0o600); err != nil {
						t.Fatal(err)
					}
				}
			}
			cmd := exec.Command(bash, "-c", "set -eu\n"+block[1])
			cmd.Dir = dir
			out, err := cmd.CombinedOutput()
			if (err == nil) != tc.pass {
				t.Fatalf("passed=%t, want %t: %v\n%s", err == nil, tc.pass, err, out)
			}
		})
	}
}

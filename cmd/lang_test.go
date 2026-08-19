package cmd

// Les critères d'acceptation du bilinguisme, mesurés sur le PRODUIT et non sur
// une intention : on compile le binaire, on l'exécute, on lit ce qu'il imprime.
//
// Deux invariants, et ils tirent dans des directions opposées, ce qui est le
// point : l'anglais ne doit plus laisser passer un mot de français, et le
// français ne doit pas avoir bougé d'un caractère. Le second est celui qu'on
// oublie : une internationalisation réussie qui déplacerait la sortie française
// casserait les portes de CI de tous ses utilisateurs actuels.

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"unicode"
)

// repoRoot est la racine du dépôt vue depuis le paquet cmd.
const repoRoot = ".."

// scanFixture : le plan Terraform non conforme livré dans le dépôt. Choisi parce
// qu'il déclenche neuf contrôles de sept familles : la sortie couvre messages,
// remédiations, titres, verdict, bandeau et avertissement de portée.
const scanFixture = "examples/scaleway/terraform/plan.json"

// buildPepin compile le binaire à mesurer. Compiler plutôt que réutiliser un
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

// runPepin exécute le binaire dans un environnement ENTIÈREMENT explicite : la
// résolution de langue lit l'environnement, donc en hériter ferait dépendre le
// résultat de la machine qui lance les tests.
func runPepin(t *testing.T, bin string, env []string, args ...string) (stdout, stderr string) {
	t.Helper()
	cmd := exec.Command(bin, args...) // #nosec G204 -- binaire compilé par le test, arguments constants.
	cmd.Dir = repoRoot
	cmd.Env = append([]string{"NO_COLOR=1", "TERM=dumb", "HOME=" + t.TempDir()}, env...)
	var o, e strings.Builder
	cmd.Stdout = &o
	cmd.Stderr = &e
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if !asExitError(err, &ee) {
			t.Fatalf("exécution de pepin %s : %v", strings.Join(args, " "), err)
		}
		// Un code de sortie non nul est le RÉSULTAT attendu d'un scan non conforme.
	}
	return o.String(), e.String()
}

// asExitError évite d'importer errors pour un unique As.
func asExitError(err error, target **exec.ExitError) bool {
	if ee, ok := err.(*exec.ExitError); ok { //nolint:errorlint // exec.Cmd.Run rend l'erreur sans l'envelopper.
		*target = ee
		return true
	}
	return false
}

// TestFrenchScanOutputHasNotMovedOneCharacter compare la sortie française à une
// capture prise AVANT l'internationalisation (binaire de `main`, committée sous
// testdata/lang/). Le français est la langue de référence : le rendre bilingue ne
// devait rien y changer, et « rien » se prouve octet par octet.
//
// La référence n'est pas une capture du code d'aujourd'hui, ce qui reviendrait à
// se comparer à soi-même. C'est la sortie de la version publiée.
func TestFrenchScanOutputHasNotMovedOneCharacter(t *testing.T) {
	bin := buildPepin(t)
	// LANG seul, sans PEPIN_LANG ni LC_ALL : c'est le chemin de résolution du poste
	// francophone ordinaire, celui qu'un repli trop zélé casserait.
	stdout, stderr := runPepin(t, bin, []string{"LANG=fr_FR.UTF-8"},
		"scan", "scaleway", "--terraform", scanFixture)

	for _, c := range []struct{ golden, got, flux string }{
		{"testdata/lang/scan-fr.stdout", stdout, "stdout"},
		{"testdata/lang/scan-fr.stderr", stderr, "stderr"},
	} {
		want, err := os.ReadFile(c.golden) // #nosec G304 -- fixture du dépôt.
		if err != nil {
			t.Fatalf("référence %s absente : %v", c.golden, err)
		}
		if string(want) != c.got {
			t.Errorf("la sortie française (%s) a bougé : %s", c.flux, firstDiff(string(want), c.got))
		}
	}
}

// TestEnglishScanOutputCarriesNoAccentedLetter est le critère d'acceptation
// « LANG=en ne produit aucun caractère accenté hors noms de ressources ».
//
// La réserve « hors noms de ressources » est appliquée MÉCANIQUEMENT, pas laissée
// au jugement : un mot accenté n'est toléré que s'il figure tel quel dans la
// fixture scannée, donc s'il vient de la donnée auditée et non du texte de Pépin.
// Le plan d'exemple porte justement de la prose française dans une `description`
// Terraform : le jour où le rapport la citera, ce test dira que c'est une donnée
// de l'inventaire, pas une régression de traduction.
//
// Seules les LETTRES sont regardées. Le bandeau ASCII, « — », « … », « · », « ⓘ »
// et les émojis de sévérité sont de la ponctuation ou des symboles : ils ne
// trahissent aucune langue et sont légitimes dans les deux.
func TestEnglishScanOutputCarriesNoAccentedLetter(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join(repoRoot, scanFixture)) // #nosec G304 -- fixture du dépôt.
	if err != nil {
		t.Fatalf("lecture de la fixture : %v", err)
	}
	bin := buildPepin(t)
	// Les quatre surfaces qu'un anglophone rencontre : le rapport terminal, le
	// document parsable le plus riche (evidence et motifs de non-évaluation y sont
	// tous rendus), et les deux niveaux d'aide.
	cases := []struct {
		name string
		args []string
	}{
		{"scan table", []string{"scan", "scaleway", "--terraform", scanFixture}},
		{"scan assessment", []string{"scan", "scaleway", "--terraform", scanFixture, "--format", "assessment"}},
		{"aide racine", []string{"--help"}},
		{"aide scan", []string{"scan", "--help"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			stdout, stderr := runPepin(t, bin, []string{"LANG=en_US.UTF-8"}, c.args...)
			for flux, out := range map[string]string{"stdout": stdout, "stderr": stderr} {
				for _, w := range accentedWordsNotFromInput(out, string(fixture)) {
					t.Errorf("sortie anglaise (%s) : mot accenté %q, absent de l'inventaire scanné — "+
						"il vient donc du texte de Pépin", flux, w)
				}
			}
		})
	}
}

// accentedWordsNotFromInput rend les mots accentués d'une sortie qui ne figurent
// PAS dans les données scannées. Un mot repris de l'inventaire (nom de ressource,
// étiquette, description saisie par l'exploitant) est cité tel quel par le rapport
// et n'a pas à être traduit ; tout autre mot accenté est du français resté dans
// une sortie anglaise.
func accentedWordsNotFromInput(out, input string) []string {
	var bad []string
	for _, line := range strings.Split(out, "\n") {
		for _, tok := range strings.Fields(line) {
			word := strings.Trim(tok, `«»"'.,;:!?()[]{}|─`)
			if _, ok := firstAccentedLetter(word); !ok {
				continue
			}
			if strings.Contains(input, word) {
				continue // valeur venue de l'inventaire audité
			}
			bad = append(bad, word)
		}
	}
	return bad
}

// TestTheLangFlagBeatsTheEnvironment : --lang l'emporte sur l'environnement, et
// une valeur inconnue retombe sur l'anglais sans erreur. Vérifié de bout en bout
// (le résolveur a ses propres tests ; ici on mesure le câblage réel de la CLI,
// qui lit --lang dans os.Args avant que cobra n'existe).
func TestTheLangFlagBeatsTheEnvironment(t *testing.T) {
	bin := buildPepin(t)
	cases := []struct {
		name   string
		env    []string
		args   []string
		expect string
	}{
		{"--lang=fr sur un environnement anglais", []string{"LANG=en_US.UTF-8"},
			[]string{"--lang=fr", "scan", "scaleway", "--terraform", scanFixture}, "Verdict : NON CONFORME"},
		{"--lang en séparé", []string{"LANG=fr_FR.UTF-8"},
			[]string{"--lang", "en", "scan", "scaleway", "--terraform", scanFixture}, "Verdict: NON-COMPLIANT"},
		{"valeur inconnue : repli anglais, sans erreur", []string{"LANG=fr_FR.UTF-8"},
			[]string{"--lang=klingon", "scan", "scaleway", "--terraform", scanFixture}, "Verdict: NON-COMPLIANT"},
		{"PEPIN_LANG bat LC_ALL", []string{"PEPIN_LANG=fr", "LC_ALL=en_US.UTF-8"},
			[]string{"scan", "scaleway", "--terraform", scanFixture}, "Verdict : NON CONFORME"},
		{"aucune variable : repli anglais", nil,
			[]string{"scan", "scaleway", "--terraform", scanFixture}, "Verdict: NON-COMPLIANT"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			stdout, _ := runPepin(t, bin, c.env, c.args...)
			if !strings.Contains(stdout, c.expect) {
				t.Errorf("verdict attendu %q absent de la sortie", c.expect)
			}
		})
	}
}

// TestTheTranslationLabelsNeverReachTheOutput : `message_en` et `remediation_en`
// sont un TRANSPORT entre la règle Rego et le rendu, pas une donnée du rapport.
// Les laisser dans `labels` ferait voyager les deux langues dans chaque finding
// de `--format json`, du SARIF et de l'assessment scellé, et changerait le digest
// du bundle pour une raison qui ne regarde pas la posture du tenant.
func TestTheTranslationLabelsNeverReachTheOutput(t *testing.T) {
	bin := buildPepin(t)
	for _, format := range []string{"json", "assessment", "sarif"} {
		for _, lang := range []string{"fr_FR.UTF-8", "en_US.UTF-8"} {
			stdout, _ := runPepin(t, bin, []string{"LANG=" + lang},
				"scan", "scaleway", "--terraform", scanFixture, "--format", format)
			for _, label := range []string{"message_en", "remediation_en"} {
				if strings.Contains(stdout, label) {
					t.Errorf("--format %s (LANG=%s) : le label de transport %q est publié dans la sortie",
						format, lang, label)
				}
			}
		}
	}
}

// firstAccentedLetter rend la première lettre non ASCII d'une chaîne.
func firstAccentedLetter(s string) (rune, bool) {
	for _, r := range s {
		if r > unicode.MaxASCII && unicode.IsLetter(r) {
			return r, true
		}
	}
	return 0, false
}

// firstDiff localise le premier écart entre deux sorties.
func firstDiff(want, got string) string {
	w := strings.Split(want, "\n")
	g := strings.Split(got, "\n")
	for i := 0; i < len(w) && i < len(g); i++ {
		if w[i] != g[i] {
			return "ligne " + itoa(i+1) + "\n  référence : " + w[i] + "\n  observée  : " + g[i]
		}
	}
	return "longueurs différentes (" + itoa(len(w)) + " lignes de référence, " + itoa(len(g)) + " observées)"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

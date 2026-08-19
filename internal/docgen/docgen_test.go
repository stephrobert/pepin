package docgen

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// repoRoot est la racine du dépôt, vue depuis internal/docgen.
const repoRoot = "../.."

// TestGeneratedDocsAreUpToDate est la PORTE : la documentation dérivée committée doit être
// exactement celle que le dépôt produit aujourd'hui. Elle régénère tout (matrice de couverture
// calculée depuis le référentiel et les descripteurs, sorties capturées en exécutant réellement
// le binaire) et compare au disque.
//
// Le contrôle porte sur le COMPORTEMENT, pas sur une intention : ajouter un contrôle, changer
// une sévérité, modifier un mapping Terraform, corriger une règle qui déplace une sortie — tout
// cela fait échouer ce test tant que `mise run gen-docs` n'a pas été relancé et le résultat
// committé. Une page de couverture périmée ne peut donc pas atteindre `main`.
func TestGeneratedDocsAreUpToDate(t *testing.T) {
	bin, err := ResolveBinary(repoRoot, t.TempDir())
	if err != nil {
		t.Fatalf("binaire de capture indisponible : %v", err)
	}
	want, err := Generate(repoRoot, bin)
	if err != nil {
		t.Fatalf("génération : %v", err)
	}
	if len(want) == 0 {
		t.Fatal("le générateur n'a produit aucun fichier : le test ne mesurerait plus rien")
	}
	for page, expected := range want {
		got, rerr := os.ReadFile(filepath.Join(repoRoot, page)) // #nosec G304 -- page d'une liste constante du paquet.
		if rerr != nil {
			t.Errorf("%s : %v — lancer `mise run gen-docs`", page, rerr)
			continue
		}
		if string(got) != expected {
			t.Errorf("%s est périmé : %s.\nLancer `mise run gen-docs` et committer le résultat.",
				page, firstDifference(string(got), expected))
		}
	}
}

// TestTheGeneratorActuallyRunsTheBinary vérifie que la capture passe bien par une EXÉCUTION :
// un harnais qui rendrait des sorties vides ferait passer le test de fraîcheur sur des pages
// vides, c'est-à-dire qu'il mesurerait sa propre panne plutôt que son sujet.
func TestTheGeneratorActuallyRunsTheBinary(t *testing.T) {
	bin, err := ResolveBinary(repoRoot, t.TempDir())
	if err != nil {
		t.Fatalf("binaire de capture indisponible : %v", err)
	}
	c, err := captureAll(repoRoot, bin)
	if err != nil {
		t.Fatalf("capture : %v", err)
	}
	if !strings.Contains(c.vulnerable.Stdout, "Verdict") {
		t.Error("le scan de l'exemple non conforme ne porte pas de verdict : capture douteuse")
	}
	if c.vulnerable.Exit != 1 {
		t.Errorf("scan non conforme : code de sortie %d, attendu 1", c.vulnerable.Exit)
	}
	if c.fixed.Exit != 0 {
		t.Errorf("scan corrigé : code de sortie %d, attendu 0", c.fixed.Exit)
	}
	if c.missingFile.Exit != 2 {
		t.Errorf("fichier absent : code de sortie %d, attendu 2", c.missingFile.Exit)
	}
	if c.empty.Exit != 3 {
		t.Errorf("inventaire vide : code de sortie %d, attendu 3 (rien mesuré)", c.empty.Exit)
	}
	if c.taglessStr.Exit != 3 {
		t.Errorf("écarts medium seuls avec --strict : code de sortie %d, attendu 3", c.taglessStr.Exit)
	}
	if c.tagless.Exit != 0 {
		t.Errorf("écarts medium seuls sans --strict : code de sortie %d, attendu 0", c.tagless.Exit)
	}
}

// TestTheCapturedBannerCarriesNoBuildVersion : la version est injectée au build, donc une page
// qui la figerait divergerait à chaque construction. La substitution est ancrée sur « v<version> » ;
// ce test vérifie qu'elle a bien opéré ET qu'elle n'a pas mangé la sortie au passage (le repli
// « dev » d'un build hors dépôt est un sous-mot de « deviations »).
func TestTheCapturedBannerCarriesNoBuildVersion(t *testing.T) {
	bin, err := ResolveBinary(repoRoot, t.TempDir())
	if err != nil {
		t.Fatalf("binaire de capture indisponible : %v", err)
	}
	r := Runner{Bin: bin, Root: repoRoot}
	ver, err := r.Run("version")
	if err != nil {
		t.Fatalf("version : %v", err)
	}
	raw := strings.TrimPrefix(strings.TrimSpace(ver.Stdout), "pépin ")
	c, err := captureAll(repoRoot, bin)
	if err != nil {
		t.Fatalf("capture : %v", err)
	}
	if strings.Contains(c.vulnerable.Stderr, "v"+raw) {
		t.Errorf("le bandeau capturé porte encore la version de build %q", raw)
	}
	if !strings.Contains(c.vulnerable.Stderr, "v"+versionPlaceholder) {
		t.Errorf("le bandeau capturé ne porte pas le marqueur de version : substitution muette")
	}
	if !strings.Contains(c.vulnerable.Stdout, "Total deviations") {
		t.Error("la substitution de version a altéré le corps du rapport (« Total deviations » perdu)")
	}
}

// TestTheMatrixCoversEveryControlAndProvider : la matrice n'a pas de trou. Une ligne manquante
// serait invisible dans une page de 57 lignes, et c'est exactement le genre d'absence qui fait
// mentir une documentation de couverture.
func TestTheMatrixCoversEveryControlAndProvider(t *testing.T) {
	m, err := BuildMatrix(repoRoot)
	if err != nil {
		t.Fatalf("matrice : %v", err)
	}
	if len(m.Rows) == 0 || len(m.CloudProviders) == 0 {
		t.Fatal("matrice vide")
	}
	for _, want := range []string{"scaleway", "outscale", "exoscale"} {
		if !contains(m.CloudProviders, want) {
			t.Errorf("fournisseur %q absent de la matrice", want)
		}
	}
	for _, r := range m.Rows {
		for _, p := range append(append([]string{}, m.CloudProviders...), m.OtherProviders...) {
			for _, src := range []Source{SourceTerraform, SourceLive} {
				c, ok := r.Cells[p][src]
				if !ok {
					t.Errorf("%s / %s / %s : case absente", r.Code, p, src)
					continue
				}
				if c.Status != Supported && c.Reason == "" {
					t.Errorf("%s / %s / %s : statut %q sans motif — un statut sans motif n'est pas opposable",
						r.Code, p, src, c.Status)
				}
			}
		}
	}
}

// linkRe repère les liens Markdown relatifs (on ignore les liens absolus et les ancres pures).
var linkRe = regexp.MustCompile(`\[[^\]]*\]\(([^)#]+)(#[^)]*)?\)`)

// TestInternalDocLinksResolve : tout lien relatif d'une page de docs/ (et des pages racine
// bilingues) pointe un fichier qui existe. Un lien mort dans une documentation de conformité
// est une promesse non tenue, et il ne se voit pas à la relecture.
func TestInternalDocLinksResolve(t *testing.T) {
	var pages []string
	err := filepath.WalkDir(filepath.Join(repoRoot, "docs"), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".md") {
			pages = append(pages, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("parcours de docs/ : %v", err)
	}
	for _, name := range []string{"README.md", "README.fr.md"} {
		pages = append(pages, filepath.Join(repoRoot, name))
	}
	if len(pages) == 0 {
		t.Fatal("aucune page trouvée : le test ne mesurerait rien")
	}
	for _, page := range pages {
		raw, rerr := os.ReadFile(page) // #nosec G304 -- pages découvertes sous docs/ du dépôt.
		if rerr != nil {
			t.Errorf("%s : %v", page, rerr)
			continue
		}
		for _, m := range linkRe.FindAllStringSubmatch(string(raw), -1) {
			target := strings.TrimSpace(m[1])
			if target == "" || strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") {
				continue
			}
			resolved := filepath.Join(filepath.Dir(page), target)
			if _, serr := os.Stat(resolved); serr != nil {
				t.Errorf("%s : lien relatif mort vers %q", rel(page), target)
			}
		}
	}
}

// rel raccourcit un chemin pour les messages d'échec.
func rel(p string) string {
	if r, err := filepath.Rel(repoRoot, p); err == nil {
		return r
	}
	return p
}

// firstDifference localise le premier écart entre deux rendus, pour que l'échec dise OÙ ça
// diverge plutôt que de recracher deux pages entières.
func firstDifference(got, want string) string {
	g := strings.Split(got, "\n")
	w := strings.Split(want, "\n")
	for i := 0; i < len(g) && i < len(w); i++ {
		if g[i] != w[i] {
			return "première divergence ligne " + itoa(i+1) + "\n  committé : " + truncate(g[i]) + "\n  généré   : " + truncate(w[i])
		}
	}
	return "longueurs différentes (" + itoa(len(g)) + " lignes committées, " + itoa(len(w)) + " générées)"
}

func truncate(s string) string {
	const max = 120
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
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

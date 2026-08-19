package docgen

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// injectedPages : les pages écrites à la main qui portent des régions générées. La page reste
// de la prose (elle explique, elle argumente) ; ce qui s'y périme — sorties de commande,
// tableaux dérivés du référentiel — est injecté.
var injectedPages = []string{
	"docs/getting-started/quickstart.md",
	"docs/getting-started/quickstart.fr.md",
	"docs/getting-started/understanding-a-scan.md",
	"docs/getting-started/understanding-a-scan.fr.md",
	"docs/concepts/assessment-model.md",
	"docs/concepts/assessment-model.fr.md",
	"docs/concepts/scope.md",
	"docs/concepts/scope.fr.md",
	"docs/known-limitations.md",
	"docs/known-limitations.fr.md",
	"docs/reference/cli.md",
	"docs/reference/cli.fr.md",
	"docs/reference/exit-codes.md",
	"docs/reference/exit-codes.fr.md",
	"docs/reference/output-formats.md",
	"docs/reference/output-formats.fr.md",
	"docs/concepts/terraform-vs-live.md",
	"docs/concepts/terraform-vs-live.fr.md",
	"docs/guides/remediation.md",
	"docs/guides/remediation.fr.md",
	"docs/guides/evidence-bundles.md",
	"docs/guides/evidence-bundles.fr.md",
	"docs/guides/github-actions.md",
	"docs/guides/github-actions.fr.md",
	"docs/guides/gitlab-ci.md",
	"docs/guides/gitlab-ci.fr.md",
	"docs/providers/scaleway.md",
	"docs/providers/scaleway.fr.md",
	"docs/providers/outscale.md",
	"docs/providers/outscale.fr.md",
	"docs/providers/exoscale.md",
	"docs/providers/exoscale.fr.md",
}

// generatedPages : les pages entièrement calculées, sans une ligne écrite à la main.
var generatedPages = []string{
	"docs/coverage.md",
	"docs/coverage.fr.md",
}

// fullyGeneratedDirs : les dossiers dont TOUT le contenu Markdown est produit par le
// générateur. Ce qui s'y trouve sans y être produit est une orpheline, et se supprime.
var fullyGeneratedDirs = []string{controlsDir}

// blockRe repère une région générée :
//
//	<!-- pepin:gen id -->
//	… contenu remplacé …
//	<!-- /pepin:gen id -->
//
// Le marqueur porte l'identifiant aux DEUX extrémités : une région tronquée par une édition
// manuelle ne se referme alors pas silencieusement sur la suivante.
var blockRe = regexp.MustCompile(`(?s)<!-- pepin:gen ([a-z0-9-]+) -->\n.*?<!-- /pepin:gen ([a-z0-9-]+) -->`)

// Generate rend le contenu ATTENDU de chaque fichier documentaire dérivé, indexé par chemin
// relatif à la racine du dépôt. Le générateur (tools/docgen) l'écrit ; le test de régénération
// le compare à ce qui est committé. Les deux empruntent le même chemin de code : il n'existe
// pas de version « de vérification » qui pourrait diverger de la version « de production ».
func Generate(root, bin string) (map[string]string, error) {
	rem, err := RemediationCoverages(root)
	if err != nil {
		return nil, err
	}
	proofs, err := RemediationProofs(root)
	if err != nil {
		return nil, err
	}
	families, err := familyOrder()
	if err != nil {
		return nil, err
	}
	// Une campagne de captures PAR LANGUE : Pépin est bilingue, donc la page anglaise
	// doit montrer la sortie anglaise et la page française la sortie française. Une
	// campagne unique recopiée dans les deux pages ferait mentir l'une des deux, et
	// c'est précisément le défaut que l'internationalisation vient corriger.
	// Idem pour la matrice, qui porte de la prose (titres de contrôles, motifs) :
	// une par langue, sinon la page française citerait des motifs anglais.
	captureByLang := map[string]captures{}
	matrixByLang := map[string]Matrix{}
	for _, lang := range []string{"fr", "en"} {
		c, cerr := captureAll(root, bin, lang)
		if cerr != nil {
			return nil, cerr
		}
		captureByLang[lang] = c
		m, merr := BuildMatrix(root, lang)
		if merr != nil {
			return nil, merr
		}
		matrixByLang[lang] = m
	}

	out := map[string]string{}
	for _, page := range generatedPages {
		lang := langOf(page)
		out[page] = matrixByLang[lang].coveragePage(lang)
	}
	// Le catalogue des contrôles : une page par contrôle et par langue, plus son index.
	// La liste des pages n'est pas énumérée ici — elle EST le référentiel, donc un contrôle
	// ajouté produit sa page sans qu'on pense à l'inscrire quelque part.
	for _, lang := range []string{"fr", "en"} {
		for page, body := range controlPages(lang, matrixByLang[lang], proofs, families) {
			out[page] = body
		}
	}
	for _, page := range injectedPages {
		lang := langOf(page)
		blocks := buildBlocks(lang, matrixByLang[lang], captureByLang[lang], rem)
		raw, rerr := os.ReadFile(filepath.Join(root, page)) // #nosec G304 -- chemin d'une liste constante du paquet.
		if rerr != nil {
			return nil, fmt.Errorf("lecture de %s : %w", page, rerr)
		}
		injected, ierr := inject(page, string(raw), blocks)
		if ierr != nil {
			return nil, ierr
		}
		out[page] = injected
	}
	return out, nil
}

// Write écrit sur disque le résultat de Generate et rend la liste des fichiers modifiés.
func Write(root, bin string) ([]string, error) {
	want, err := Generate(root, bin)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(want))
	for p := range want {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	var changed []string
	for _, p := range paths {
		full := filepath.Join(root, p)
		current, rerr := os.ReadFile(full) // #nosec G304 -- chemin d'une liste constante du paquet.
		if rerr == nil && string(current) == want[p] {
			continue
		}
		if mkerr := os.MkdirAll(filepath.Dir(full), 0o750); mkerr != nil {
			return nil, mkerr
		}
		if werr := os.WriteFile(full, []byte(want[p]), 0o600); werr != nil {
			return nil, fmt.Errorf("écriture de %s : %w", p, werr)
		}
		changed = append(changed, p)
	}
	removed, err := prune(root, want)
	if err != nil {
		return nil, err
	}
	return append(changed, removed...), nil
}

// prune supprime, dans les dossiers ENTIÈREMENT générés, les pages que le générateur ne
// produit plus. Sans cela, retirer un contrôle du référentiel laisserait sa page en place :
// le test de fraîcheur compare le contenu de ce qu'il génère et ne verrait jamais l'orpheline,
// c'est-à-dire une page qui décrit un contrôle que le produit n'a plus.
func prune(root string, want map[string]string) ([]string, error) {
	var removed []string
	for _, dir := range fullyGeneratedDirs {
		stale, err := stalePages(root, dir, want)
		if err != nil {
			return nil, err
		}
		for _, p := range stale {
			if rerr := os.Remove(filepath.Join(root, p)); rerr != nil {
				return nil, fmt.Errorf("suppression de %s : %w", p, rerr)
			}
			removed = append(removed, "(supprimé) "+p)
		}
	}
	return removed, nil
}

// stalePages rend les pages présentes sur le disque dans un dossier entièrement généré que le
// générateur ne produit pas. Exposé au test de fraîcheur, qui doit signaler l'orpheline plutôt
// que de la corriger en silence.
func stalePages(root, dir string, want map[string]string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(root, dir))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("lecture de %s : %w", dir, err)
	}
	var stale []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		page := dir + "/" + e.Name()
		if _, ok := want[page]; !ok {
			stale = append(stale, page)
		}
	}
	sort.Strings(stale)
	return stale, nil
}

// inject remplace chaque région générée d'une page par son bloc. Un identifiant inconnu, ou
// des marqueurs dépareillés, sont des ERREURS : une région qu'on croit générée et qui ne l'est
// pas est exactement la documentation périmée qu'on cherche à rendre impossible.
func inject(page, content string, blocks map[string]string) (string, error) {
	var bad error
	out := blockRe.ReplaceAllStringFunc(content, func(m string) string {
		sub := blockRe.FindStringSubmatch(m)
		id, closing := sub[1], sub[2]
		if id != closing {
			bad = fmt.Errorf("%s : région générée « %s » refermée par « %s »", page, id, closing)
			return m
		}
		body, ok := blocks[id]
		if !ok {
			bad = fmt.Errorf("%s : région générée « %s » inconnue du générateur (internal/docgen)", page, id)
			return m
		}
		return "<!-- pepin:gen " + id + " -->\n" + strings.TrimRight(body, "\n") + "\n<!-- /pepin:gen " + id + " -->"
	})
	if bad != nil {
		return "", bad
	}
	if strings.Count(content, "<!-- pepin:gen ") != strings.Count(content, "<!-- /pepin:gen ") {
		return "", fmt.Errorf("%s : marqueurs de région générée dépareillés", page)
	}
	return out, nil
}

// langOf déduit la langue d'une page de son nom : `*.fr.md` est la version française, tout le
// reste est la version anglaise (primaire).
func langOf(page string) string {
	if strings.HasSuffix(page, ".fr.md") {
		return "fr"
	}
	return "en"
}

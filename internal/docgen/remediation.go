package docgen

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/stephrobert/pepin/referentiel"
)

// RemediationCoverage compte, par fournisseur, les contrôles actifs qui disposent d'une PREUVE
// de remédiation sous references/remediation/. Même convention que `mise run check-remediation`
// (scripts/check-remediation.py) : une note `<code>.md`, ou un module Terraform autonome
// `<code>/*.tf`. Recalculé ici en Go pour que la page ne dépende pas d'un interpréteur Python
// et de ses dépendances au moment de la génération.
//
// À NE PAS confondre avec la remédiation TEXTUELLE portée par chaque finding, qui est, elle,
// garantie par TestEveryFindingCarriesRemediation.
type RemediationCoverage struct {
	Provider string
	Covered  int
	Total    int
	Missing  []string
}

// RemediationCoverages calcule la couverture pour chaque fournisseur cité par le référentiel,
// triée par nom.
func RemediationCoverages(root string) ([]RemediationCoverage, error) {
	base := filepath.Join(root, "references", "remediation")
	byProvider := map[string][]string{}
	for code, ctl := range referentiel.All() {
		for _, p := range ctl.Fournisseurs {
			byProvider[p] = append(byProvider[p], code)
		}
	}
	names := make([]string, 0, len(byProvider))
	for p := range byProvider {
		names = append(names, p)
	}
	sort.Strings(names)

	out := make([]RemediationCoverage, 0, len(names))
	for _, p := range names {
		codes := byProvider[p]
		sort.Strings(codes)
		cov := RemediationCoverage{Provider: p, Total: len(codes)}
		for _, code := range codes {
			ok, err := hasProof(base, p, code)
			if err != nil {
				return nil, err
			}
			if ok {
				cov.Covered++
				continue
			}
			cov.Missing = append(cov.Missing, code)
		}
		out = append(out, cov)
	}
	return out, nil
}

// RemediationProofs rend, par fournisseur puis par code, le chemin RELATIF À LA RACINE de la
// preuve de remédiation déposée : le dossier du module Terraform, ou la note Markdown. Sert au
// catalogue des contrôles, qui lie la preuve plutôt que d'affirmer qu'elle existe.
func RemediationProofs(root string) (map[string]map[string]string, error) {
	base := filepath.Join(root, "references", "remediation")
	out := map[string]map[string]string{}
	for code, ctl := range referentiel.All() {
		for _, p := range ctl.Fournisseurs {
			path, err := proofPath(base, p, code)
			if err != nil {
				return nil, err
			}
			if path == "" {
				continue
			}
			if out[p] == nil {
				out[p] = map[string]string{}
			}
			out[p][code] = path
		}
	}
	return out, nil
}

// proofPath rend le chemin de la preuve, ou "" s'il n'y en a pas. Même règle d'existence que
// hasProof : les deux s'appuient sur la MÊME lecture, sans quoi la page pourrait lier une
// preuve que le compteur ignore.
func proofPath(base, provider, code string) (string, error) {
	rel := "references/remediation/" + provider + "/" + code
	if _, err := os.Stat(filepath.Join(base, provider, code+".md")); err == nil {
		return rel + ".md", nil
	}
	ok, err := hasProof(base, provider, code)
	if err != nil || !ok {
		return "", err
	}
	return rel, nil
}

// hasProof applique la règle de `check-remediation` : une note Markdown, ou un dossier
// contenant au moins un `.tf`.
func hasProof(base, provider, code string) (bool, error) {
	if _, err := os.Stat(filepath.Join(base, provider, code+".md")); err == nil {
		return true, nil
	}
	dir := filepath.Join(base, provider, code)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("lecture de %s : %w", dir, err)
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".tf") {
			return true, nil
		}
	}
	return false, nil
}

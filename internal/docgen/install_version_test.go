package docgen

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// La page d'installation est la PREMIÈRE page qu'on recopie, et elle épinglait
// `v0.1.0` — la version dont ce dépôt écrit lui-même, dans
// `examples/github-actions/pepin.yml`, que son installeur « refusait TOUTE
// installation ». Une page qui décrit un produit disparu est pire qu'une page
// absente, parce qu'elle inspire confiance.
//
// La garde s'adosse au CHANGELOG plutôt qu'aux tags git : un `git clone` de CI peut
// arriver sans tags, et une doc générée depuis un état absent serait instable. Le
// CHANGELOG, lui, est dans le dépôt — c'est la même discipline que partout ailleurs
// ici : ce qui se vérifie doit être lisible dans l'arbre.

// versionEpinglee capture un numéro de version dans un épinglage d'exemple.
var versionEpinglee = regexp.MustCompile(`\b[vV]?(\d+\.\d+\.\d+)\b`)

// derniereVersionPubliee lit la version en tête du CHANGELOG : la première
// section `## [X.Y.Z]`, en ignorant `## [Unreleased]`.
func derniereVersionPubliee(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(repoRoot, "CHANGELOG.md"))
	if err != nil {
		t.Fatalf("lecture du CHANGELOG : %v", err)
	}
	re := regexp.MustCompile(`(?m)^## \[(\d+\.\d+\.\d+)\]`)
	m := re.FindSubmatch(raw)
	if m == nil {
		t.Fatal("aucune version publiée dans le CHANGELOG : la garde ne mesure rien")
	}
	return string(m[1])
}

// TestTheInstallPagePinsTheLatestRelease : tout numéro de version épinglé dans la
// page d'installation est celui de la dernière release.
//
// La garde porte sur TOUTES les occurrences, pas sur une ligne repère : l'écart
// d'origine touchait le binaire, l'image, l'action ET le modèle GitLab, et n'en
// corriger qu'une partie aurait laissé la page à moitié fausse — ce qui se remarque
// encore moins.
func TestTheInstallPagePinsTheLatestRelease(t *testing.T) {
	attendu := derniereVersionPubliee(t)
	for _, page := range []string{"docs/install.md", "docs/install.fr.md"} {
		chemin := filepath.Join(repoRoot, page)
		raw, err := os.ReadFile(chemin) // #nosec G304 -- page du dépôt.
		if err != nil {
			t.Fatalf("lecture de %s : %v", page, err)
		}
		vues := map[string]bool{}
		for _, ligne := range strings.Split(string(raw), "\n") {
			// Les mentions HISTORIQUES sont légitimes et ne doivent pas rougir : elles
			// nomment une version passée pour dire ce qui n'y marchait pas. On ne
			// regarde donc que les lignes qui ÉPINGLENT, c'est-à-dire celles qui
			// portent une URL de release, une image, une référence d'action ou une
			// entrée `version:`.
			if !ligneDepinglage(ligne) {
				continue
			}
			for _, m := range versionEpinglee.FindAllStringSubmatch(ligne, -1) {
				vues[m[1]] = true
			}
		}
		if len(vues) == 0 {
			t.Errorf("%s : aucun épinglage trouvé — la garde ne mesure plus rien", page)
			continue
		}
		var liste []string
		for v := range vues {
			liste = append(liste, v)
		}
		sort.Strings(liste)
		for _, v := range liste {
			if v != attendu {
				t.Errorf("%s épingle %s, alors que la dernière release est %s.\n"+
					"  Versions épinglées : %s\n"+
					"  La page d'installation est la première qu'on recopie ; elle ne doit pas\n"+
					"  proposer une version que le dépôt ne publie plus.", page, v, attendu, strings.Join(liste, ", "))
			}
		}
	}
}

// ligneDepinglage dit si une ligne propose une version À INSTALLER, par opposition à
// une ligne qui en cite une pour raconter ce qu'elle valait.
func ligneDepinglage(ligne string) bool {
	for _, motif := range []string{
		"releases/download/",
		"ghcr.io/stephrobert/pepin:",
		"gh release download",
		"pepin-scan@",
		"raw.githubusercontent.com/stephrobert/pepin/",
	} {
		if strings.Contains(ligne, motif) {
			return true
		}
	}
	return strings.HasPrefix(strings.TrimSpace(ligne), "version:")
}

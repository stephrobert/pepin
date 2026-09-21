package genprovider_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Toute déclaration `default:` du mapping Terraform cite sa SOURCE.
//
// # Ce qu'un `default:` affirme
//
// `default:X` dit « cet argument, que l'auteur du HCL n'a pas écrit, vaut X ». Ce
// n'est pas une convention de Pépin : c'est une affirmation sur le CONTRAT du
// provider — sur ce qu'il envoie à l'API quand l'argument est omis. Une affirmation
// de cette nature se vérifie dans le code ou la documentation du fournisseur, et
// `CLAUDE.md` §2 interdit de la poser autrement.
//
// # Pourquoi une porte, et pas une consigne
//
// Les quatre premières déclarations de ce dépôt ont été écrites sans source, et
// l'une d'elles portait sur un attribut `required` : elle ne pouvait pas tirer, et
// personne ne s'en est aperçu pendant des mois. Une déclaration fausse ne se voit
// pas — elle produit un verdict plausible. Celle qui portait sur `cidrs` alimente
// `network_securitygroup_allow_ingress_from_internet_to_tcp_port_22`, un contrôle
// CRITICAL : une source erronée y fabriquerait un écart critique sur un tenant
// tiers inchangé.
//
// La porte ne juge pas la source, elle exige qu'il y en ait une. Le reste est la
// revue.
func TestEveryTerraformDefaultCitesItsSource(t *testing.T) {
	entries, err := os.ReadDir("../../providers")
	if err != nil {
		t.Fatalf("lecture des descripteurs : %v", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		t.Run(e.Name(), func(t *testing.T) {
			raw, rerr := os.ReadFile(filepath.Join("../../providers", e.Name())) // #nosec G304 -- descripteur du dépôt.
			if rerr != nil {
				t.Fatalf("lecture : %v", rerr)
			}
			lignes := strings.Split(string(raw), "\n")
			dansMapping := false
			for i, l := range lignes {
				// Les clés de premier niveau délimitent les blocs : `mapping_terraform:`
				// ouvre, toute autre clé non indentée ferme.
				if len(l) > 0 && l[0] != ' ' && l[0] != '#' && strings.Contains(l, ":") {
					dansMapping = strings.HasPrefix(l, "mapping_terraform:")
				}
				if !dansMapping || !strings.Contains(l, "default:") {
					continue
				}
				if sourceProche(lignes, i) {
					continue
				}
				t.Errorf("%s:%d — `default:` sans source citée :\n    %s\n"+
					"  Un `default:` affirme ce que le provider envoie à l'API quand\n"+
					"  l'argument est omis. C'est un fait de son contrat, pas une convention\n"+
					"  de Pépin : il se cite (CLAUDE.md §2, ADR-0024).\n"+
					"  Ajouter au-dessus un commentaire « # source: … » nommant le fichier du\n"+
					"  provider, ou la page de documentation, qui l'établit.",
					e.Name(), i+1, strings.TrimRight(l, " "))
			}
		})
	}
}

// sourceProche cherche un `# source:` sur la ligne elle-même, ou dans le BLOC DE
// COMMENTAIRES CONTIGU qui la précède.
//
// Le critère n'est pas une fenêtre de N lignes — un nombre arbitraire serait tantôt
// trop court pour une source qui mérite d'être expliquée, tantôt assez long pour
// attraper la source de la déclaration d'à côté. Le bloc contigu, lui, désigne sans
// ambiguïté CETTE déclaration : il s'arrête à la première ligne qui n'est pas un
// commentaire.
func sourceProche(lignes []string, i int) bool {
	if strings.Contains(lignes[i], "# source:") {
		return true
	}
	for j := i - 1; j >= 0; j-- {
		l := strings.TrimSpace(lignes[j])
		if !strings.HasPrefix(l, "#") {
			return false // fin du bloc de commentaires
		}
		if strings.Contains(l, "# source:") {
			return true
		}
	}
	return false
}

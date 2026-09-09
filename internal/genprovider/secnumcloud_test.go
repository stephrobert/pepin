package genprovider

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Une qualification SecNumCloud porte sur un PÉRIMÈTRE de régions, pas sur un
// fournisseur entier. Celle d'Outscale couvre `cloudgouv-eu-west-1` et elle seule ; un
// tenant en `eu-west-2` lisait pourtant « SecNumCloud qualifié » dans un `pass` de
// souveraineté. C'est la première affirmation qu'un auditeur conteste, et tout
// l'argument de l'outil est qu'il n'affirme jamais plus qu'il n'a observé.

func souv(statut string, regions ...string) Souverainete {
	return Souverainete{SecNumCloud: statut, SecNumCloudRegions: regions}
}

func TestTheQualificationIsScopedToItsRegions(t *testing.T) {
	for _, c := range []struct {
		nom     string
		s       Souverainete
		region  string
		attendu string
	}{
		{"région dans le périmètre", souv("qualifie", "cloudgouv-eu-west-1"), "cloudgouv-eu-west-1", "qualifie"},
		// LE cas de l'issue : la région est connue, et elle est hors périmètre. Ce
		// n'est pas « non qualifié » — le fournisseur l'est —, c'est « pas ici ».
		{"région hors périmètre", souv("qualifie", "cloudgouv-eu-west-1"), "eu-west-2", SecNumCloudHorsPerimetre},
		// Un plan Terraform n'a pas de région. Ne pas savoir n'est pas savoir que non :
		// on ne tranche donc ni dans un sens ni dans l'autre (ADR-0014).
		{"région inconnue", souv("qualifie", "cloudgouv-eu-west-1"), "", SecNumCloudPerimetreInconnu},
		// LES CONTRE-EXEMPLES : sans périmètre déclaré, rien ne change. On ne fabrique
		// pas une restriction que le descripteur n'énonce pas — les deux autres
		// fournisseurs seraient sinon dégradés sans qu'aucune source ne le justifie.
		{"aucun périmètre déclaré", souv("qualifie"), "eu-west-2", "qualifie"},
		{"non qualifié, avec région", souv("non"), "eu-west-2", "non"},
		{"en cours, avec périmètre", souv("en_cours", "cloudgouv-eu-west-1"), "eu-west-2", "en_cours"},
		{"statut vide", souv(""), "eu-west-2", ""},
	} {
		t.Run(c.nom, func(t *testing.T) {
			if got := secNumCloudPour(c.s, c.region); got != c.attendu {
				t.Errorf("région %q → %q, attendu %q", c.region, got, c.attendu)
			}
		})
	}
}

// TestTheQualifiedRegionsAreInTheProviderCatalogue : un périmètre ne peut nommer
// qu'une région que le fournisseur a. Une faute de frappe y rendrait la qualification
// inatteignable — donc `hors_perimetre` partout — sans que rien ne rougisse.
func TestTheQualifiedRegionsAreInTheProviderCatalogue(t *testing.T) {
	vus := 0
	for nom, d := range descripteursDuDepotInterne(t) {
		for _, r := range d.Souverainete.SecNumCloudRegions {
			vus++
			if !contient(d.Regions, r) {
				t.Errorf("providers/%s.yaml : le périmètre SecNumCloud nomme %q, absente du catalogue de régions (%v)",
					nom, r, d.Regions)
			}
		}
	}
	if vus == 0 {
		t.Skip("aucun périmètre SecNumCloud déclaré")
	}
}

func contient(l []string, v string) bool {
	for _, e := range l {
		if e == v {
			return true
		}
	}
	return false
}

// descripteursDuDepotInterne charge les descripteurs COMMITTÉS depuis le paquet
// interne. Doublon assumé de son homologue du test externe : le registre du processus
// est vide hors du binaire, et une garde qui parcourt une carte vide passe sans rien
// mesurer.
func descripteursDuDepotInterne(t *testing.T) map[string]Descriptor {
	t.Helper()
	entrees, err := os.ReadDir(filepath.Join("..", "..", "providers"))
	if err != nil {
		t.Fatalf("lecture des descripteurs : %v", err)
	}
	out := map[string]Descriptor{}
	for _, e := range entrees {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		d, lerr := Load(os.DirFS(filepath.Join("..", "..", "providers")), e.Name())
		if lerr != nil {
			t.Fatalf("chargement de %s : %v", e.Name(), lerr)
		}
		out[strings.TrimSuffix(e.Name(), ".yaml")] = d
	}
	if len(out) == 0 {
		t.Fatal("aucun descripteur chargé : la garde ne mesure rien")
	}
	return out
}

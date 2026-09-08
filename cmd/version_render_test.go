package cmd

import (
	"strings"
	"testing"
)

// La version est injectée au build par `git describe`, qui rend `v0.3.0-18-g25888fc`,
// alors que le repli codé en dur est nu (`0.1.0-dev`). Ses lecteurs supposaient chacun
// une forme, et deux d'entre eux se contredisaient : le bandeau ajoutait un « v » à une
// chaîne qui en portait déjà un, imprimant `vv0.3.0-18-g25888fc` sur CHAQUE scan — dans
// le GIF de démonstration comme dans toute capture d'écran de la documentation.
//
// Ces deux tests éprouvent les deux formes canoniques sur les deux formes d'entrée, ce
// qui est le seul moyen de ne pas refaire le même pari.
func TestVersionRendersExactlyOnePrefix(t *testing.T) {
	original := version
	t.Cleanup(func() { version = original })

	for _, entree := range []string{"0.3.0", "v0.3.0", "0.1.0-dev", "v0.3.0-18-g25888fc"} {
		version = entree

		if got := displayVersion(); !strings.HasPrefix(got, "v") || strings.HasPrefix(got, "vv") {
			t.Errorf("version %q : affichée %q — il en faut un « v », et un seul", entree, got)
		}
		if got := bareVersion(); strings.HasPrefix(got, "v") {
			t.Errorf("version %q : forme nue %q — les surfaces parsables ne portent pas le préfixe",
				entree, got)
		}
		// Le rendu partagé ajoute son propre « v » à ce qu'on lui passe : c'est la
		// composition exacte qui produisait le défaut.
		if got := "v" + bareVersion(); strings.HasPrefix(got, "vv") {
			t.Errorf("version %q : le bandeau imprimerait %q", entree, got)
		}
	}
}

// Les deux formes doivent rester cohérentes entre elles, quelle que soit l'entrée :
// `displayVersion` est exactement `bareVersion` précédée d'un « v ». Sans cette
// vérification, corriger l'une et oublier l'autre reproduirait le désaccord d'origine.
func TestVersionFormsAgreeWithEachOther(t *testing.T) {
	original := version
	t.Cleanup(func() { version = original })

	for _, entree := range []string{"0.3.0", "v0.3.0", "", "v"} {
		version = entree
		if displayVersion() != "v"+bareVersion() {
			t.Errorf("version %q : affichée %q, nue %q — les deux formes divergent",
				entree, displayVersion(), bareVersion())
		}
	}
}

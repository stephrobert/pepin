package cmd

import (
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// Une sous-commande inconnue était acceptée avec le code 0, et dans un cas elle
// exécutait SILENCIEUSEMENT autre chose que ce qui avait été tapé : `pepin provider
// inexistant` lançait `provider list`.
//
// Ce n'est pas une coquetterie d'ergonomie. `pepin control list` est une supposition
// naturelle — `provider list` existe, on essaie la forme symétrique — et une étape de
// pipeline écrite `pepin control list --json > controls.json` réussissait, écrivait un
// écran d'aide dans le fichier, et le job passait au vert. Cet outil met par ailleurs
// un soin particulier à ce que ses codes de sortie veuillent dire quelque chose, ce qui
// est précisément ce qui faisait ressortir l'écart.

// TestEveryCommandValidatesItsArguments parcourt l'ARBRE plutôt qu'une liste écrite à
// la main.
//
// Une liste se périme à la première commande ajoutée, et celle qu'on oublie est
// justement celle qui repartira en silence. En marchant l'arbre, une commande nouvelle
// est tenue sans que personne n'ait à y penser.
func TestEveryCommandValidatesItsArguments(t *testing.T) {
	var vus int
	var marcher func(c *cobra.Command, chemin string)
	marcher = func(c *cobra.Command, chemin string) {
		if c != rootCmd { // la racine rejette déjà une commande inconnue, c'est cobra
			vus++
			if c.Args == nil {
				t.Errorf("commande %q : aucun validateur d'arguments.\n"+
					"  Un argument qu'elle ne comprend pas serait IGNORÉ — au mieux un écran\n"+
					"  d'aide et un code 0, au pire l'exécution d'une autre commande que celle\n"+
					"  tapée. Poser `Args: cobra.NoArgs` si elle n'en prend aucun, ou le\n"+
					"  validateur qui décrit ceux qu'elle prend.", chemin)
			}
			// Une commande NON exécutable ne valide pas ses arguments : cobra rend
			// l'aide et s'arrête avant. Poser `Args` sans `RunE` donne donc l'illusion
			// d'une garde — c'est exactement ce qui laissait `control list` rendre 0.
			if c.Args != nil && !c.Runnable() {
				t.Errorf("commande %q : `Args` déclaré sur une commande NON exécutable.\n"+
					"  Cobra rend l'aide avant de valider : la garde ne s'applique jamais.\n"+
					"  Lui donner un `RunE`, même trivial (`return cmd.Help()`).", chemin)
			}
		}
		for _, e := range c.Commands() {
			marcher(e, strings.TrimSpace(chemin+" "+e.Name()))
		}
	}
	marcher(rootCmd, "pepin")
	if vus == 0 {
		t.Fatal("aucune commande parcourue : la garde ne mesure rien")
	}
	t.Logf("%d commande(s) parcourue(s)", vus)
}

// TestAnUnknownSubcommandExitsTwo mesure sur le BINAIRE ce que la garde d'arbre
// vérifie sur la structure.
//
// Les deux sont nécessaires : la première dit qu'un validateur EXISTE, celle-ci qu'il
// produit le bon code. Un validateur posé sur une commande non exécutable satisfait la
// première et ne change rien au comportement — c'est exactement ce qui laissait
// `control list` rendre 0.
func TestAnUnknownSubcommandExitsTwo(t *testing.T) {
	bin := buildPepin(t)
	cas := [][]string{
		{"control", "list"},          // la supposition naturelle, symétrique de `provider list`
		{"provider", "inexistant"},   // exécutait `provider list` en silence
		{"provider", "list", "trop"}, // trouvée par la garde d'arbre
		{"scsl", "inexistant"},
		{"version", "parasite"},
		{"nimportequoi"}, // la racine, qui était déjà correcte
	}
	for _, args := range cas {
		if code := exitCodeOfArgs(t, bin, args...); code != exitErreur {
			t.Errorf("pepin %v rend %d, attendu %d.\n"+
				"  Une sous-commande incomprise qui rend 0 fait passer une étape de pipeline\n"+
				"  au vert, en écrivant au mieux un écran d'aide dans le fichier de sortie.",
				args, code, exitErreur)
		}
	}
	// Et les formes VALIDES doivent tenir : une correction qui refuserait tout serait
	// pire que le défaut.
	for _, args := range [][]string{{"provider"}, {"provider", "list"}, {"control"}, {"version"}} {
		if code := exitCodeOfArgs(t, bin, args...); code != 0 {
			t.Errorf("pepin %v rend %d : la correction refuse une forme valide", args, code)
		}
	}
}

// exitCodeOfArgs lance le binaire et rend son code de sortie : ici, le code EST la
// mesure, donc une erreur d'exécution ne doit pas faire échouer le test.
func exitCodeOfArgs(t *testing.T, bin string, args ...string) int {
	t.Helper()
	c := exec.Command(bin, args...)
	c.Dir = repoRoot
	if err := c.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return ee.ExitCode()
		}
		t.Fatalf("exécution de %v : %v", args, err)
	}
	return 0
}

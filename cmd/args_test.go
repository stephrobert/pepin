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

// exitCodeOfArgs lance le binaire et rend son code de sortie.
//
// Le code EST la mesure dans plusieurs gardes de ce paquet — sous-commande inconnue,
// discordance de fournisseur, profil de porte —, donc une sortie non nulle ne doit pas
// faire échouer le test. Un seul exemplaire : deux helpers identiques divergent.
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

// TestTheRootDescriptionNamesTheProvidersThatExist — l'issue #150.
//
// La première phrase qu'un nouvel utilisateur lit annonçait OVH — une entrée de feuille
// de route, pas un fournisseur — et omettait Kubernetes, qui en est un. La liste est
// désormais DÉRIVÉE du registre : une liste recopiée se périme au premier fournisseur
// ajouté ou retiré, et personne ne relit une phrase d'accueil.
func TestTheRootDescriptionNamesTheProvidersThatExist(t *testing.T) {
	bin := buildPepin(t)
	stdout, _ := runPepin(t, bin, nil, "--help")

	// Les fournisseurs sont lus depuis le BINAIRE, pas depuis le processus de test :
	// le registre se peuple à partir des descripteurs du dépôt, et un test qui
	// tournerait dans un autre répertoire mesurerait un registre vide.
	liste, _ := runPepin(t, bin, nil, "provider", "list")
	var enregistres []string
	for _, l := range strings.Split(liste, "\n") {
		champs := strings.Fields(l)
		if len(champs) > 0 && !strings.HasPrefix(champs[0], "/") && champs[0] != "" {
			if n := champs[0]; strings.ToLower(n) == n && !strings.Contains(n, "é") {
				enregistres = append(enregistres, n)
			}
		}
	}
	if len(enregistres) == 0 {
		t.Fatal("aucun fournisseur listé par le binaire : le test ne mesure rien")
	}
	for _, p := range enregistres {
		if !strings.Contains(stdout, p) {
			t.Errorf("la description racine omet le fournisseur %q, qui est enregistré", p)
		}
	}
	// Le cas EXACT du rapport : un nom annoncé qui n'existe pas.
	for _, p := range enregistres {
		if p == "ovh" {
			return // le jour où il existe, cette garde s'efface d'elle-même
		}
	}
	if strings.Contains(strings.ToLower(stdout), "ovh") {
		t.Error("la description racine annonce OVH, qui n'est pas un fournisseur enregistré")
	}
}

// TestTheSCSLIndexHasNoMaintainerDefault — l'issue #154.
//
// Le défaut de `--index` était un chemin relatif remontant hors du répertoire courant
// vers un dépôt dont le lecteur n'a jamais entendu parler : la disposition locale d'un
// mainteneur échappée dans un binaire publié. L'erreur était juste et ne disait rien.
func TestTheSCSLIndexHasNoMaintainerDefault(t *testing.T) {
	bin := buildPepin(t)
	_, stderr := runPepin(t, bin, nil, "scsl")

	// Ce qui était fautif n'est pas de MENTIONNER ce chemin — l'exemple aide — c'est
	// de le poser en DÉFAUT, comme s'il allait de soi sur la machine du lecteur.
	aide, _ := runPepin(t, bin, nil, "scsl", "--help")
	if strings.Contains(aide, `(default "../framework-scsl`) {
		t.Error("`--index` porte encore le chemin d'un mainteneur comme valeur par défaut")
	}
	// Le message doit dire ce QU'EST le fichier, et que la commande est un outil de
	// maintenance : sans cela, le lecteur ne peut pas savoir si c'est à lui d'agir.
	for _, attendu := range []string{"--index", "SCSL", "MAINTENANCE"} {
		if !strings.Contains(strings.ToUpper(stderr), strings.ToUpper(attendu)) {
			t.Errorf("le message n'explique pas %q :\n%s", attendu, stderr)
		}
	}
	if code := exitCodeOfArgs(t, bin, "scsl"); code != exitErreur {
		t.Errorf("`pepin scsl` sans index rend %d, attendu %d", code, exitErreur)
	}
}

// TestAnUnknownFormatIsRefused — l'issue #149.
//
// Une valeur inconnue de `--format` était ignorée sans un mot : le scan tournait et
// imprimait la table. Le cas dangereux n'est pas l'humain qui tape `xml` à un prompt et
// le remarque ; c'est l'étape de pipeline écrite `--format oscal` éditée en
// `--format oscal2`, ou la variable qui s'évalue à autre chose que prévu. Le job garde
// sa sémantique de code de sortie, publie un fichier, et l'artefact de conformité que
// toute la chaîne en aval croit être de l'OSCAL est un tableau colorié.
//
// La liste des formats valides est LUE depuis le code, pas recopiée : une liste
// recopiée se périme au premier format ajouté, et le format oublié serait justement
// celui que personne n'éprouve.
func TestAnUnknownFormatIsRefused(t *testing.T) {
	bin := buildPepin(t)
	const fixture = "examples/scaleway/inventory.json"

	if len(scanFormats) == 0 {
		t.Fatal("aucun format déclaré : le test ne mesure rien")
	}
	for _, f := range scanFormats {
		if code := exitCodeOfArgs(t, bin, "scan", "scaleway", fixture, "--format", f); code == exitErreur {
			t.Errorf("le format valide %q rend %d : la correction refuse ce qu'elle doit accepter", f, code)
		}
	}
	for _, f := range []string{"xml", "oscal2", "TABLE", "", "jsonl"} {
		if code := exitCodeOfArgs(t, bin, "scan", "scaleway", fixture, "--format", f); code != exitErreur {
			t.Errorf("le format inconnu %q rend %d, attendu %d.\n"+
				"  Un pipeline publierait un tableau colorié là où il croit écrire de l'OSCAL,\n"+
				"  avec le code de sortie d'un scan réussi.", f, code, exitErreur)
		}
	}
}

// TestNothingMeasuredNeverRendersAsCompliant — l'issue #158.
//
// Quand rien n'a pu être mesuré, le rapport montrait une coche VERTE, « aucun écart sur
// le périmètre audité », et quatre compteurs à zéro — immédiatement au-dessus d'un
// verdict INDÉTERMINÉ qui disait l'inverse. Trois signaux littéralement vrais et
// collectivement trompeurs.
//
// Le mode d'échec est HUMAIN, pas machine : le code de sortie était déjà 3, et une
// automatisation se comportait correctement. C'est la personne qui survole un terminal,
// ou la capture collée dans un ticket, qui voit une coche et une rangée de zéros.
//
// Les deux sens sont éprouvés. Un cas RÉELLEMENT conforme doit garder sa coche : une
// correction qui les rendrait identiques n'aurait fait que déplacer la confusion.
func TestNothingMeasuredNeverRendersAsCompliant(t *testing.T) {
	bin := buildPepin(t)

	// Rien de mesuré : un plan dont aucune ressource n'entre dans un contrôle.
	vide, _ := runPepin(t, bin, []string{"LANG=fr_FR.UTF-8"},
		"scan", "scaleway", "--terraform", "references/tenants/scaleway/kubic/plan.json")
	if strings.Contains(vide, "✓") {
		t.Error("la coche verte subsiste alors que rien n'a été mesuré : c'est ce que l'œil retient")
	}
	if strings.Contains(vide, "Aucun écart") {
		t.Error("« aucun écart » subsiste alors que rien n'a été regardé")
	}
	if strings.Contains(vide, "CRITICAL") {
		t.Error("les compteurs subsistent : quatre zéros se lisent comme un feu vert")
	}
	if !strings.Contains(vide, "Aucun contrôle n'a pu être mesuré") {
		t.Errorf("le rendu ne nomme pas la cause :\n%s", vide)
	}
	// Le code de sortie et le rendu doivent dire la MÊME chose : c'est leur divergence
	// qui faisait le défaut.
	if code := exitCodeOfArgs(t, bin, "scan", "scaleway", "--terraform",
		"references/tenants/scaleway/kubic/plan.json"); code != exitStrict {
		t.Errorf("code %d, attendu %d : le rendu et la porte ne disent plus la même chose", code, exitStrict)
	}

	// Et le cas RÉELLEMENT conforme garde tout ce qui le distingue.
	ok, _ := runPepin(t, bin, []string{"LANG=fr_FR.UTF-8"},
		"scan", "scaleway", "examples/scaleway/inventory-ok.json")
	for _, want := range []string{"✓", "Aucun écart", "CRITICAL"} {
		if !strings.Contains(ok, want) {
			t.Errorf("le cas conforme a perdu %q : la correction a rendu les deux cas identiques", want)
		}
	}
}

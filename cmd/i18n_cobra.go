package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// L'ossature que COBRA imprime — « Usage », « Available Commands », « Flags », le
// « help for <cmd> » de chaque commande, et les erreurs de nombre d'arguments.
//
// C'est la seconde source de l'issue #151, et elle est distincte de celle du rapport :
// les libellés du rapport viennent du moteur de rendu partagé, ceux-ci du framework de
// ligne de commande. L'écran d'aide mélangeait les deux registres — descriptions de
// commandes en français, structure en anglais —, ce qui se lit comme une traduction
// inachevée plutôt que comme un choix.
//
// Ce qui reste en anglais est ÉCRIT plus bas, plutôt que passé sous silence.

// usageTemplate reprend le gabarit de cobra en rendant traduisibles ses seuls
// libellés. La structure (les conditions, les boucles) est celle de cobra : la
// recopier pour la traduire est le prix à payer, le gabarit n'étant pas composable.
func usageTemplate() string {
	return `{{.UseLine}}` + "\n" + `{{if .HasAvailableSubCommands}}` + "\n" +
		usageWord() + `:
  {{.CommandPath}} [` + commandWord() + `]{{end}}{{if gt (len .Aliases) 0}}

` + aliasesWord() + `:
  {{.NameAndAliases}}{{end}}{{if .HasAvailableSubCommands}}

` + availableWord() + `:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

` + flagsWord() + `:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

` + globalFlagsWord() + `:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableSubCommands}}

` + moreInfoLine() + `{{end}}` + "\n"
}

func usageWord() string       { return tr("Utilisation", "Usage") }
func commandWord() string     { return tr("commande", "command") }
func aliasesWord() string     { return tr("Alias", "Aliases") }
func availableWord() string   { return tr("Commandes disponibles", "Available Commands") }
func flagsWord() string       { return tr("Drapeaux", "Flags") }
func globalFlagsWord() string { return tr("Drapeaux globaux", "Global Flags") }

func moreInfoLine() string {
	return tr(
		`Lancer « {{.CommandPath}} [commande] --help » pour en savoir plus sur une commande.`,
		`Use "{{.CommandPath}} [command] --help" for more information about a command.`)
}

// localizeCobra applique le gabarit et les libellés du framework à tout l'arbre.
//
// Appelée après `localize`, et avant que cobra ne construise la moindre aide : un
// gabarit posé plus tard ne servirait qu'aux commandes non encore rendues.
func localizeCobra(root *cobra.Command) {
	root.SetUsageTemplate(usageTemplate())
	var marcher func(c *cobra.Command)
	marcher = func(c *cobra.Command) {
		// `help for <cmd>` : cobra le fabrique à la volée, donc il faut d'abord
		// forcer sa création pour pouvoir le réécrire.
		c.InitDefaultHelpFlag()
		if f := c.Flags().Lookup("help"); f != nil {
			f.Usage = tr("aide pour ", "help for ") + c.Name()
		}
		for _, e := range c.Commands() {
			marcher(e)
		}
	}
	marcher(root)
	// L'erreur de DRAPEAU inconnu vient de pflag, et cobra offre un point d'accroche
	// pour la réécrire. Celle de COMMANDE inconnue (« unknown command … Did you mean
	// this? ») est construite au fond de `Command.Find`, sans hook : la traduire
	// exigerait de reconnaître son texte anglais, donc de dépendre de la formulation
	// d'une dépendance. Elle reste en anglais, et c'est écrit plutôt que tu —
	// `TestTheUntranslatedCobraResidueIsKnown` la tient inventoriée.
	root.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
		msg := err.Error()
		if nom, ok := strings.CutPrefix(msg, "unknown flag: "); ok {
			return fmt.Errorf(tr("drapeau inconnu : %s\n%s", "unknown flag: %s\n%s"), nom, c.UseLine())
		}
		if nom, ok := strings.CutPrefix(msg, "unknown shorthand flag: "); ok {
			return fmt.Errorf(tr("drapeau court inconnu : %s\n%s", "unknown shorthand flag: %s\n%s"), nom, c.UseLine())
		}
		return err
	})
	// La commande `help` elle-même, que cobra ajoute silencieusement.
	root.InitDefaultHelpCmd()
	for _, c := range root.Commands() {
		if c.Name() == "help" {
			c.Short = tr("Aide sur n'importe quelle commande", "Help about any command")
		}
	}
}

// exactArgs, rangeArgs — les validateurs de cobra rendent leurs erreurs en anglais
// (« accepts between 1 and 2 arg(s), received 0 »), au milieu d'un message d'erreur
// préfixé « erreur : ». Ces équivalents disent la même chose dans la langue résolue.
func rangeArgs(min, max int) cobra.PositionalArgs {
	return func(c *cobra.Command, args []string) error {
		if len(args) >= min && len(args) <= max {
			return nil
		}
		return fmt.Errorf(tr(
			"%s attend entre %d et %d argument(s), %d reçu(s)\n%s",
			"%s accepts between %d and %d arg(s), %d received\n%s"),
			c.CommandPath(), min, max, len(args), c.UseLine())
	}
}

func exactArgs(n int) cobra.PositionalArgs {
	return func(c *cobra.Command, args []string) error {
		if len(args) == n {
			return nil
		}
		return fmt.Errorf(tr(
			"%s attend exactement %d argument(s), %d reçu(s)\n%s",
			"%s accepts exactly %d arg(s), %d received\n%s"),
			c.CommandPath(), n, len(args), c.UseLine())
	}
}

func noArgs() cobra.PositionalArgs {
	return func(c *cobra.Command, args []string) error {
		if len(args) == 0 {
			return nil
		}
		sous := make([]string, 0, len(c.Commands()))
		for _, e := range c.Commands() {
			if e.IsAvailableCommand() {
				sous = append(sous, e.Name())
			}
		}
		if len(sous) == 0 {
			return fmt.Errorf(tr(
				"%s n'attend aucun argument, %q reçu",
				"%s takes no argument, got %q"), c.CommandPath(), args[0])
		}
		return fmt.Errorf(tr(
			"sous-commande inconnue %q pour %q\n  sous-commandes disponibles : %s",
			"unknown subcommand %q for %q\n  available subcommands: %s"),
			args[0], c.CommandPath(), strings.Join(sous, ", "))
	}
}

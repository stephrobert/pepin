// Package cmd porte l'interface en ligne de commande de Pépin (cobra).
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/stephrobert/pepin/internal/i18n"
)

var rootCmd = &cobra.Command{
	Use: "pepin",
	// Les chaînes d'aide sont RÉÉCRITES par localize() une fois la langue résolue
	// (cf. cmd/i18n.go). Les littéraux français restent la référence lisible.
	Short: "Pépin — trouve les pépins de votre cloud souverain",
	Long: "Pépin — CSPM multi-cloud souverain.\n\n" +
		"Évalue la posture d'un cloud (OVH, Scaleway, Exoscale, Outscale…) contre un\n" +
		"référentiel commun ancré sur SCSL, SecNumCloud, CIS et ISO.",
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute lance la racine. Codes de sortie : 0 conforme · 1 non-conformité · 2 erreur.
//
// La LANGUE est résolue en tout premier, avant que cobra ne construise la moindre
// aide : --lang est lu directement dans os.Args, parce qu'au moment où cobra en
// connaîtrait la valeur, les `Short`/`Long` seraient déjà figés.
func Execute() {
	i18n.Set(i18n.Resolve(langFromArgs(os.Args[1:]), os.Getenv))
	localize()
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, errStyle.Render(tr("erreur : ", "error: "))+err.Error())
		os.Exit(exitErreur)
	}
}

func init() {
	// Drapeau PERSISTANT : la langue vaut pour la racine et toutes ses
	// sous-commandes. Enregistré pour que `--help` le documente et qu'une valeur
	// inconnue ne provoque pas d'erreur de parsing ; sa lecture EFFECTIVE se fait
	// plus tôt, dans Execute.
	rootCmd.PersistentFlags().StringVar(&langFlag, "lang", "",
		"langue de l'interface : fr | en (défaut : PEPIN_LANG, puis LC_ALL/LANG, sinon en)")
	rootCmd.AddCommand(scanCmd, providerCmd, versionCmd, scslCmd, controlCmd)
}

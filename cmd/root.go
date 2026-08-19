// Package cmd porte l'interface en ligne de commande de Pépin (cobra).
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "pepin",
	Short: "Pépin — trouve les pépins de votre cloud souverain",
	Long: "Pépin — CSPM multi-cloud souverain.\n\n" +
		"Évalue la posture d'un cloud (OVH, Scaleway, Exoscale, Outscale…) contre un\n" +
		"référentiel commun ancré sur SCSL, SecNumCloud, CIS et ISO.",
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute lance la racine. Codes de sortie : 0 conforme · 1 non-conformité · 2 erreur.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, errStyle.Render("erreur : ")+err.Error())
		os.Exit(exitErreur)
	}
}

func init() {
	rootCmd.AddCommand(scanCmd, providerCmd, versionCmd, scslCmd)
}

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// version est surchargée au build via -ldflags.
var version = "0.1.0-dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Afficher la version",
	Run: func(_ *cobra.Command, _ []string) {
		// L'accent tombe en anglais : `pepin version` est la sortie la plus
		// susceptible d'être coupée, collée et comparée par un script.
		_, _ = fmt.Println(tr("pépin ", "pepin ") + version)
	},
}

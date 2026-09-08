package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// version est surchargée au build via -ldflags.
//
// Sa forme n'est PAS stable d'un build à l'autre : le repli ci-dessous est nu, tandis
// que `git describe` rend une chaîne préfixée (`v0.3.0-18-g25888fc`). Ses trois
// lecteurs supposaient chacun une forme différente, et deux d'entre eux se
// contredisaient — le bandeau ajoutait un « v » à une chaîne qui en portait déjà un,
// d'où le `vv0.3.0…` imprimé à chaque scan. On ne lit donc plus cette variable
// directement : `bareVersion` et `displayVersion` en donnent les deux seules formes.
var version = "0.1.0-dev"

// bareVersion rend la version SANS préfixe : c'est la forme des surfaces parsables
// (`tool.version` d'un assessment, d'un bundle, d'un OSCAL) et celle qu'on passe à un
// rendu qui ajoute lui-même son « v ».
func bareVersion() string { return strings.TrimPrefix(version, "v") }

// displayVersion rend la version telle qu'on la MONTRE, avec un « v » et un seul.
func displayVersion() string { return "v" + bareVersion() }

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Afficher la version",
	Run: func(_ *cobra.Command, _ []string) {
		// L'accent tombe en anglais : `pepin version` est la sortie la plus
		// susceptible d'être coupée, collée et comparée par un script.
		_, _ = fmt.Println(tr("pépin ", "pepin ") + displayVersion())
	},
}

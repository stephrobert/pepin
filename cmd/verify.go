package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/stephrobert/pepin/internal/assess"
)

var verifyCmd = &cobra.Command{
	Use:   "verify <dossier-bundle>",
	Short: "Vérifier l'intégrité d'un bundle de preuve (checksums)",
	Long: "Recalcule l'empreinte SHA-256 de chaque fichier listé dans checksums.txt et signale\n" +
		"toute altération. C'est la vérification tierce d'un bundle produit par `scan --seal` :\n" +
		"un auditeur confirme que l'assessment, l'OSCAL et le manifeste n'ont pas été modifiés.\n" +
		"La signature (cosign/gpg de checksums.txt) reste à l'identité de l'opérateur.",
	Args: cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		if err := assess.VerifyBundle(args[0]); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(os.Stdout, "✓ bundle intègre : %s\n", args[0])
		return nil
	},
}

func init() {
	rootCmd.AddCommand(verifyCmd)
}

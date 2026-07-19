package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/stephrobert/pepin/internal/assess"
)

var (
	verifyPubKey string // clé publique cosign (vérification par clé)
	verifyBundle string // bundle de signature cosign (défaut : <dossier>/checksums.txt.bundle)
)

var verifyCmd = &cobra.Command{
	Use:   "verify <dossier-bundle>",
	Short: "Vérifier l'intégrité (et la signature) d'un bundle de preuve",
	Long: "Recalcule l'empreinte SHA-256 de chaque fichier listé dans checksums.txt et signale\n" +
		"toute altération. C'est la vérification tierce d'un bundle produit par `scan --seal`.\n\n" +
		"Sans --pubkey, seule l'intégrité (altération accidentelle) est vérifiée : un attaquant\n" +
		"peut régénérer fichiers + checksums. Avec --pubkey, la SIGNATURE cosign de checksums.txt\n" +
		"est vérifiée (non-répudiation) — l'opérateur ayant scellé le bundle avec (cosign 3.x) :\n" +
		"  cosign sign-blob --key cosign.key --bundle checksums.txt.bundle checksums.txt",
	Args: cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		dir := args[0]
		if err := assess.VerifyBundle(dir); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(os.Stdout, "✓ bundle intègre : %s\n", dir)

		if verifyPubKey == "" {
			_, _ = fmt.Fprintln(os.Stdout, "  (intégrité seule ; fournir --pubkey pour vérifier la signature cosign)")
			return nil
		}
		if err := verifyCosign(dir, verifyPubKey, verifyBundle); err != nil {
			return err
		}
		_, _ = fmt.Fprintln(os.Stdout, "✓ signature cosign valide (non-répudiation)")
		return nil
	},
}

// verifyCosign vérifie la signature cosign de checksums.txt via le binaire `cosign` (shell-out :
// pas de dépendance lourde embarquée). Échoue clairement si cosign est absent ou la signature
// invalide/absente.
func verifyCosign(dir, pubKey, bundlePath string) error {
	if _, err := exec.LookPath("cosign"); err != nil {
		return fmt.Errorf("cosign introuvable dans le PATH — requis pour --pubkey")
	}
	checksums := filepath.Join(dir, "checksums.txt")
	if bundlePath == "" {
		bundlePath = checksums + ".bundle"
	}
	if _, err := os.Stat(bundlePath); err != nil {
		return fmt.Errorf("bundle de signature absent (%s) — sceller : cosign sign-blob --key cosign.key --bundle %s %s", bundlePath, bundlePath, checksums)
	}
	// #nosec G204 -- arguments issus des options CLI de l'opérateur, pas d'une source distante.
	out, err := exec.Command("cosign", "verify-blob", "--key", pubKey, "--bundle", bundlePath, checksums).CombinedOutput()
	if err != nil {
		return fmt.Errorf("signature cosign INVALIDE : %s", string(out))
	}
	return nil
}

func init() {
	verifyCmd.Flags().StringVar(&verifyPubKey, "pubkey", "", "clé publique cosign pour vérifier la signature de checksums.txt")
	verifyCmd.Flags().StringVar(&verifyBundle, "bundle", "", "bundle de signature cosign (défaut : <dossier>/checksums.txt.bundle)")
	rootCmd.AddCommand(verifyCmd)
}

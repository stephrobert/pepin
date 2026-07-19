package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/stephrobert/pepin/internal/assess"
	"github.com/stephrobert/pepin/internal/commonrules"
	"github.com/stephrobert/pepin/internal/provider"
	"github.com/stephrobert/pepin/referentiel"
	"github.com/stephrobert/scankit/assessment"
	"github.com/stephrobert/scankit/engine"
)

var (
	verifyPubKey   string // clé publique cosign (vérification par clé)
	verifyBundle   string // bundle de signature cosign (défaut : <dossier>/checksums.txt.bundle)
	verifyReDerive bool   // rejouer le Rego sur input.json et comparer à l'assessment scellé
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
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := args[0]
		if err := assess.VerifyBundle(dir); err != nil {
			return err
		}
		signed := verifyPubKey != ""
		if !signed {
			// Sans signature, l'intégrité ne détecte que l'altération ACCIDENTELLE : un attaquant
			// régénère fichiers + checksums. On le dit sans ambiguïté (pas de « ✓ » rassurant).
			_, _ = fmt.Fprintf(os.Stdout, "⚠ bundle cohérent en interne : %s\n  (intégrité ACCIDENTELLE seulement — NON opposable sans --pubkey pour la signature cosign)\n", dir)
		} else {
			if err := verifyCosign(dir, verifyPubKey, verifyBundle); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(os.Stdout, "✓ bundle intègre et signé (non-répudiation) : %s\n", dir)
		}

		if verifyReDerive {
			if err := reDerive(cmd.Context(), dir); err != nil {
				return err
			}
			_, _ = fmt.Fprintln(os.Stdout, "✓ re-dérivation FIDÈLE : l'assessment scellé découle bien de input.json")
		}
		return nil
	},
}

// reDerive relit input.json du bundle, réexécute les règles communes, reconstruit l'assessment
// et le compare (base canonique) à l'assessment SCELLÉ. C'est la seule vérification réellement
// opposable : sans elle, un bundle intègre/signé peut attester un résultat qui ne découle PAS de
// son propre input.json (la signature n'atteste que des octets, jamais l'exactitude d'évaluation).
func reDerive(ctx context.Context, dir string) error {
	sealedRaw, err := os.ReadFile(filepath.Join(dir, "assessment.json")) // #nosec G304 -- dossier de bundle de l'opérateur.
	if err != nil {
		return fmt.Errorf("lecture de assessment.json : %w", err)
	}
	var sealed assessment.Assessment
	if err := json.Unmarshal(sealedRaw, &sealed); err != nil {
		return fmt.Errorf("assessment.json invalide : %w", err)
	}
	inputRaw, err := os.ReadFile(filepath.Join(dir, "input.json")) // #nosec G304 -- dossier de bundle de l'opérateur.
	if err != nil {
		return fmt.Errorf("re-dérivation impossible : input.json absent du bundle (%w)", err)
	}
	var input any
	if err := json.Unmarshal(inputRaw, &input); err != nil {
		return fmt.Errorf("input.json invalide : %w", err)
	}
	name := sealed.Run.Target.Provider
	if _, ok := provider.Get(name); !ok {
		return fmt.Errorf("re-dérivation : provider %q du bundle inconnu de ce binaire", name)
	}
	// input.json est DÉJÀ augmenté (governance) et porte evaluated_at : on ne ré-augmente ni ne
	// ré-horodate (withGovernance est idempotent, evaluated_at préservé) — on réévalue tel quel.
	findings, err := engine.Evaluate(ctx, input, commonrules.FS())
	if err != nil {
		return fmt.Errorf("re-dérivation : %w", err)
	}
	got := assess.Build(name, referentiel.All(), findings, resourceTypesOf(input),
		providerNAReasons(name), providerVerified(name), controlTypes(), sealed.Run)

	gotJSON, _ := json.Marshal(assess.Canonical(got).Results)
	sealedJSON, _ := json.Marshal(assess.Canonical(sealed).Results)
	if string(gotJSON) != string(sealedJSON) {
		return fmt.Errorf("re-dérivation DIVERGE de l'assessment scellé : le bundle n'atteste PAS fidèlement input.json (résultat fabriqué ou config différente)")
	}
	if d := configDigest(); d != sealed.Run.Ruleset.Digest {
		_, _ = fmt.Fprintf(os.Stdout, "  note : config actuelle (%s) ≠ config scellée (%s) — re-dérivation faite avec le référentiel/règles COURANTS\n", short(d), short(sealed.Run.Ruleset.Digest))
	}
	return nil
}

// short abrège une empreinte pour l'affichage.
func short(d string) string {
	if len(d) > 20 {
		return d[:20] + "…"
	}
	return d
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
	verifyCmd.Flags().BoolVar(&verifyReDerive, "re-derive", false, "rejouer les règles sur input.json et vérifier que l'assessment scellé en découle (opposabilité forte)")
	rootCmd.AddCommand(verifyCmd)
}

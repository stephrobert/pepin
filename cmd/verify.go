package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/stephrobert/pepin/internal/assess"
	"github.com/stephrobert/pepin/internal/commonrules"
	"github.com/stephrobert/pepin/internal/i18n"
	"github.com/stephrobert/pepin/internal/provider"
	"github.com/stephrobert/pepin/referentiel"
	"github.com/stephrobert/scankit/assessment"
	"github.com/stephrobert/scankit/engine"
	"github.com/stephrobert/scankit/finding"
	screport "github.com/stephrobert/scankit/report"
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
			_, _ = fmt.Fprintf(os.Stdout, tr(
				"⚠ bundle cohérent en interne : %s\n  (intégrité ACCIDENTELLE seulement — NON opposable sans --pubkey pour la signature cosign)\n",
				"⚠ bundle internally consistent: %s\n  (ACCIDENTAL integrity only — NOT defensible without --pubkey for the cosign signature)\n"), dir)
		} else {
			if err := verifyCosign(dir, verifyPubKey, verifyBundle); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(os.Stdout, tr(
				"✓ bundle intègre et signé (non-répudiation) : %s\n",
				"✓ bundle intact and signed (non-repudiation): %s\n"), dir)
		}

		if verifyReDerive {
			if err := reDerive(cmd.Context(), dir); err != nil {
				return err
			}
			_, _ = fmt.Fprintln(os.Stdout, tr(
				"✓ re-dérivation FIDÈLE : l'assessment scellé découle bien de input.json",
				"✓ FAITHFUL re-derivation: the sealed assessment does follow from input.json"))
		}
		return nil
	},
}

// reDeriveOSCAL re-rend l'OSCAL depuis l'assessment re-dérivé et le compare à celui du
// bundle. Sans ce contrôle, l'artefact normatif restait falsifiable en silence.
func reDeriveOSCAL(dir string, got assessment.Assessment) error {
	sealedOSCAL, err := os.ReadFile(filepath.Join(dir, "assessment-oscal.json")) // #nosec G304 -- dossier de bundle de l'opérateur.
	if err != nil {
		if os.IsNotExist(err) {
			return nil // bundle sans OSCAL : rien à comparer
		}
		return fmt.Errorf(tr("lecture de assessment-oscal.json : %w", "reading assessment-oscal.json: %w"), err)
	}
	var buf bytes.Buffer
	if err := screport.OSCAL(&buf, got); err != nil {
		return fmt.Errorf(tr("re-rendu OSCAL : %w", "re-rendering OSCAL: %w"), err)
	}
	var a, b any
	if err := json.Unmarshal(sealedOSCAL, &a); err != nil {
		return fmt.Errorf(tr("assessment-oscal.json invalide : %w", "invalid assessment-oscal.json: %w"), err)
	}
	if err := json.Unmarshal(buf.Bytes(), &b); err != nil {
		return fmt.Errorf(tr("OSCAL re-rendu invalide : %w", "invalid re-rendered OSCAL: %w"), err)
	}
	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	if string(ja) != string(jb) {
		return errors.New(tr(
			"re-dérivation DIVERGE sur l'OSCAL : assessment-oscal.json n'est pas le rendu de l'assessment (artefact normatif fabriqué)",
			"re-derivation DIVERGES on the OSCAL: assessment-oscal.json is not the rendering of the assessment (fabricated normative artifact)"))
	}
	return nil
}

// checkProvenance vérifie l'invariant du scan : un seul instant d'évaluation, gravé à la
// fois dans input.evaluated_at et dans Run.Timestamp. Les désaligner est le moyen le plus
// simple d'antidater un rapport.
func checkProvenance(input any, sealed assessment.Assessment) error {
	m, ok := input.(map[string]any)
	if !ok {
		return nil
	}
	evaluatedAt, _ := m["evaluated_at"].(string)
	if evaluatedAt == "" || sealed.Run.Timestamp == "" {
		return nil
	}
	if evaluatedAt != sealed.Run.Timestamp {
		return fmt.Errorf(tr(
			"provenance INCOHÉRENTE : input.evaluated_at (%s) ≠ Run.Timestamp (%s) — le scan grave un instant unique, cet écart trahit un antidatage",
			"INCONSISTENT provenance: input.evaluated_at (%s) ≠ Run.Timestamp (%s) — a scan carves a single instant, this gap betrays a backdating"),
			evaluatedAt, sealed.Run.Timestamp)
	}
	return nil
}

// reDerive relit input.json du bundle, réexécute les règles communes, reconstruit l'assessment
// et le compare (base canonique) à l'assessment SCELLÉ. C'est la seule vérification réellement
// opposable : sans elle, un bundle intègre/signé peut attester un résultat qui ne découle PAS de
// son propre input.json (la signature n'atteste que des octets, jamais l'exactitude d'évaluation).
func reDerive(ctx context.Context, dir string) error {
	sealedRaw, err := os.ReadFile(filepath.Join(dir, "assessment.json")) // #nosec G304 -- dossier de bundle de l'opérateur.
	if err != nil {
		return fmt.Errorf(tr("lecture de assessment.json : %w", "reading assessment.json: %w"), err)
	}
	var sealed assessment.Assessment
	if err := json.Unmarshal(sealedRaw, &sealed); err != nil {
		return fmt.Errorf(tr("assessment.json invalide : %w", "invalid assessment.json: %w"), err)
	}
	inputRaw, err := os.ReadFile(filepath.Join(dir, "input.json")) // #nosec G304 -- dossier de bundle de l'opérateur.
	if err != nil {
		return fmt.Errorf(tr("re-dérivation impossible : input.json absent du bundle (%w)",
			"re-derivation impossible: input.json is missing from the bundle (%w)"), err)
	}
	var input any
	if err := json.Unmarshal(inputRaw, &input); err != nil {
		return fmt.Errorf(tr("input.json invalide : %w", "invalid input.json: %w"), err)
	}
	name := sealed.Run.Target.Provider
	if _, ok := provider.Get(name); !ok {
		return fmt.Errorf(tr("re-dérivation : provider %q du bundle inconnu de ce binaire",
			"re-derivation: provider %q of the bundle is unknown to this binary"), name)
	}
	// input.json est DÉJÀ augmenté (governance) et porte evaluated_at : on ne ré-augmente ni ne
	// ré-horodate (withGovernance est idempotent, evaluated_at préservé) — on réévalue tel quel.
	findings, err := engine.Evaluate(ctx, input, commonrules.FS())
	if err != nil {
		return fmt.Errorf(tr("re-dérivation : %w", "re-derivation: %w"), err)
	}
	sealedJSON, _ := json.Marshal(assess.Canonical(sealed).Results)

	// Un bundle porte la LANGUE du scan qui l'a produit ; le vérificateur, lui, tourne
	// dans la sienne. Comparer sans en tenir compte ferait diverger un bundle
	// parfaitement fidèle pour la seule raison que son lecteur ne parle pas la même
	// langue que son auteur — un faux positif de falsification, le pire verdict qu'un
	// vérificateur puisse rendre. On rejoue donc dans les deux langues : ce qui est
	// comparé (statuts, sujets, références, provenance) est identique, seule la
	// formulation change, et une seule concordance suffit à établir la fidélité.
	got, matched := reDeriveInEitherLanguage(name, findings, input, sealed.Run, string(sealedJSON))
	if !matched {
		return errors.New(tr(
			"re-dérivation DIVERGE de l'assessment scellé : le bundle n'atteste PAS fidèlement input.json (résultat fabriqué ou config différente)",
			"re-derivation DIVERGES from the sealed assessment: the bundle does NOT faithfully attest input.json (fabricated result, or a different configuration)"))
	}
	// L'OSCAL est l'artefact qu'un AUDITEUR ingère : ne vérifier que assessment.json
	// laissait falsifier l'OSCAL (tous les échecs passés en succès) sans que la
	// re-dérivation ne bronche. On le RE-REND depuis l'assessment re-dérivé et on compare.
	if err := reDeriveOSCAL(dir, got); err != nil {
		return err
	}
	// La provenance doit rester cohérente : le scan grave UN SEUL instant, partagé entre
	// input.evaluated_at et Run.Timestamp. Une divergence trahit un antidatage.
	if err := checkProvenance(input, sealed); err != nil {
		return err
	}
	if d := configDigest(); d != sealed.Run.Ruleset.Digest {
		_, _ = fmt.Fprintf(os.Stdout, tr(
			"  note : config actuelle (%s) ≠ config scellée (%s) — re-dérivation faite avec le référentiel/règles COURANTS\n",
			"  note: current config (%s) ≠ sealed config (%s) — re-derivation done with the CURRENT reference and rules\n"),
			short(d), short(sealed.Run.Ruleset.Digest))
	}
	return nil
}

// reDeriveInEitherLanguage reconstruit l'assessment dans la langue courante puis dans
// l'autre, et rend le premier qui coïncide avec les résultats scellés. Le second retour
// dit si l'une des deux a coïncidé ; à défaut, le premier essai est rendu pour que
// l'appelant dispose d'un assessment non nul.
func reDeriveInEitherLanguage(name string, findings []finding.Finding, input any, run assessment.Run, sealedJSON string) (assessment.Assessment, bool) {
	restore := i18n.Current()
	defer i18n.Set(restore)

	other := i18n.FR
	if restore == i18n.FR {
		other = i18n.EN
	}
	var first assessment.Assessment
	for i, lang := range []i18n.Lang{restore, other} {
		i18n.Set(lang)
		f := cloneFindings(findings)
		localizeFindings(f)
		got := assess.Build(name, referentiel.All(), f, resourceTypesOf(input),
			providerNAReasons(name), providerVerified(name), controlTypes(), attrsByTypeOf(input), run)
		b, _ := json.Marshal(assess.Canonical(got).Results)
		if string(b) == sealedJSON {
			return got, true
		}
		if i == 0 {
			first = got
		}
	}
	return first, false
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
		return errors.New(tr("cosign introuvable dans le PATH — requis pour --pubkey",
			"cosign not found in PATH — required for --pubkey"))
	}
	checksums := filepath.Join(dir, "checksums.txt")
	if bundlePath == "" {
		bundlePath = checksums + ".bundle"
	}
	if _, err := os.Stat(bundlePath); err != nil {
		return fmt.Errorf(tr(
			"bundle de signature absent (%s) — sceller : cosign sign-blob --key cosign.key --bundle %s %s",
			"signature bundle missing (%s) — seal it with: cosign sign-blob --key cosign.key --bundle %s %s"),
			bundlePath, bundlePath, checksums)
	}
	// #nosec G204 -- arguments issus des options CLI de l'opérateur, pas d'une source distante.
	out, err := exec.Command("cosign", "verify-blob", "--key", pubKey, "--bundle", bundlePath, checksums).CombinedOutput()
	if err != nil {
		return fmt.Errorf(tr("signature cosign INVALIDE : %s", "INVALID cosign signature: %s"), string(out))
	}
	return nil
}

func init() {
	verifyCmd.Flags().StringVar(&verifyPubKey, "pubkey", "", "clé publique cosign pour vérifier la signature de checksums.txt")
	verifyCmd.Flags().StringVar(&verifyBundle, "bundle", "", "bundle de signature cosign (défaut : <dossier>/checksums.txt.bundle)")
	verifyCmd.Flags().BoolVar(&verifyReDerive, "re-derive", false, "rejouer les règles sur input.json et vérifier que l'assessment scellé en découle (opposabilité forte)")
	rootCmd.AddCommand(verifyCmd)
}

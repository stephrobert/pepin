package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/stephrobert/pepin/internal/assess"
	"github.com/stephrobert/scankit/assessment"
)

var diffFormat string

var diffCmd = &cobra.Command{
	Use:   "diff <bundle-avant> <bundle-après>",
	Short: "Comparer deux bundles de preuve",
	Long: "Vérifie l'intégrité interne de deux bundles scellés puis compare leurs verdicts et leurs ressources.\n\n" +
		"Les changements de version d'outil ou de règles sont signalés. La perte de visibilité n'est jamais présentée comme une correction.\n" +
		"Cette commande ne vérifie pas les signatures détachées : utiliser `pepin verify --pubkey` pour authentifier l'auteur.",
	Args: exactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if diffFormat != "table" && diffFormat != "json" {
			return fmt.Errorf(tr("format inconnu %q (valeurs : table, json)", "unknown format %q (values: table, json)"), diffFormat)
		}
		diff, err := assess.DiffBundles(args[0], args[1])
		if err != nil {
			return err
		}
		if diffFormat == "json" {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(diff)
		}
		return renderBundleDiff(cmd.OutOrStdout(), diff)
	},
}

func renderBundleDiff(w io.Writer, diff assess.BundleDiff) error {
	if _, err := fmt.Fprintf(w, "%s\n", tr("Comparaison de bundles", "Bundle comparison")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  %s: %s (%s)\n", tr("avant", "before"), diff.Before.Generated, diff.Before.Source); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  %s: %s (%s)\n", tr("après", "after"), diff.After.Generated, diff.After.Source); err != nil {
		return err
	}
	for _, change := range diff.VersionChanges {
		if _, err := fmt.Fprintf(w, "  %s: %s → %s\n", change.Component, change.Before, change.After); err != nil {
			return err
		}
	}

	s := diff.Summary
	if _, err := fmt.Fprintf(w, "\n%s\n", tr("Changements de verdict", "Verdict changes")); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  %s: %d · %s: %d · %s: %d · %s: %d · %s: %d\n",
		tr("corrigés", "fixed"), s.Fixed,
		tr("régressions", "regressions"), s.Regressed,
		tr("visibilité perdue", "visibility lost"), s.VisibilityLost,
		tr("visibilité rétablie", "visibility restored"), s.VisibilityRestored,
		tr("autres changements", "other changes"), s.StatusChanged); err != nil {
		return err
	}
	if s.ResultsAdded+s.ResultsRemoved > 0 {
		if _, err := fmt.Fprintf(w, "  %s: +%d / -%d\n",
			tr("résultats", "results"), s.ResultsAdded, s.ResultsRemoved); err != nil {
			return err
		}
	}
	for _, change := range diff.ResultChanges {
		if _, err := fmt.Fprintf(w, "  %s %s [%s]: %s → %s\n",
			resultChangeLabel(change.Kind), change.Control, change.Subject,
			statusLabel(change.Before), statusLabel(change.After)); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w, "\n%s\n", tr("Ressources", "Resources")); err != nil {
		return err
	}
	if !diff.ResourcesCompared {
		_, err := fmt.Fprintf(w, "  %s\n", tr(
			"comparaison indisponible : input.json manque dans au moins un bundle",
			"comparison unavailable: input.json is missing from at least one bundle"))
		return err
	}
	if _, err := fmt.Fprintf(w, "  +%d / -%d\n", s.ResourcesAdded, s.ResourcesRemoved); err != nil {
		return err
	}
	for _, change := range diff.ResourceChanges {
		if _, err := fmt.Fprintf(w, "  %s %s/%s/%s\n",
			resultChangeLabel(change.Kind), change.Provider, change.Type, change.ID); err != nil {
			return err
		}
	}
	if len(diff.ResultChanges) == 0 && len(diff.ResourceChanges) == 0 {
		_, err := fmt.Fprintf(w, "  %s\n", tr("aucun changement de verdict ni de ressource", "no verdict or resource changes"))
		return err
	}
	return nil
}

func resultChangeLabel(kind string) string {
	switch kind {
	case "fixed":
		return tr("corrigé", "fixed")
	case "regressed":
		return tr("régression", "regression")
	case "visibility-lost":
		return tr("visibilité perdue", "visibility lost")
	case "visibility-restored":
		return tr("visibilité rétablie", "visibility restored")
	case "added":
		return "+"
	case "removed":
		return "-"
	default:
		return tr("changé", "changed")
	}
}

func statusLabel(status assessment.Status) string {
	if status == "" {
		return tr("absent", "absent")
	}
	return strings.ReplaceAll(string(status), "-", " ")
}

func init() {
	diffCmd.Flags().StringVar(&diffFormat, "format", "table", "format de sortie : table | json")
}

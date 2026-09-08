package cmd

// `pepin control explain` — pourquoi Pépin a le droit d'affirmer un verdict.
//
// Un scanner de posture qui annonce un résultat sans pouvoir dire d'où il le
// tient demande qu'on le croie. Pour un outil qui vise l'opposabilité, c'est le
// défaut central : un auditeur ne peut pas opposer une affirmation, il oppose une
// chaîne de preuve. Cette commande rend cette chaîne, pour un contrôle et un
// fournisseur : les appels d'API qui alimentent la décision, les attributs
// décisifs, les conditions exactes d'un `pass`, les tests qui l'éprouvent, et la
// date de la dernière validation live.
//
// Elle lit la MÊME source que la carte de qualité de détection (l'instantané
// internal/quality, généré et committé) : deux calculs de couverture divergent
// toujours, et celui qui diverge est celui qu'on lit.

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/stephrobert/pepin/internal/assess"
	"github.com/stephrobert/pepin/internal/genprovider"
	"github.com/stephrobert/pepin/internal/i18n"
	"github.com/stephrobert/pepin/internal/quality"
	"github.com/stephrobert/pepin/referentiel"
)

var explainProvider string

var controlCmd = &cobra.Command{
	Use:   "control",
	Short: "Inspecter les contrôles du référentiel commun",
}

var controlExplainCmd = &cobra.Command{
	Use:   "explain <code>",
	Short: "Expliquer d'où vient le verdict d'un contrôle, et ce qui l'éprouve",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		code := args[0]
		ctl, ok := referentiel.Lookup(code)
		if !ok {
			return fmt.Errorf(tr("contrôle inconnu : %s", "unknown control: %s"), code)
		}
		snap, err := quality.Embedded()
		if err != nil {
			return err
		}
		providers := ctl.Fournisseurs
		if explainProvider != "" {
			if !contains(providers, explainProvider) {
				return fmt.Errorf(tr(
					"le contrôle %s n'est pas déclaré pour le fournisseur %s (déclarés : %s)",
					"control %s is not declared for provider %s (declared: %s)"),
					code, explainProvider, strings.Join(providers, ", "))
			}
			providers = []string{explainProvider}
		}
		out := cmd.OutOrStdout()
		explainHeader(out, ctl)
		for _, p := range providers {
			explainProviderSection(out, ctl, p, snap)
		}
		explainProof(out, code, snap)
		return nil
	},
}

// contains dit si une liste porte une valeur.
func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func explainHeader(out interface{ Write([]byte) (int, error) }, ctl referentiel.Control) {
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, eyebrow.Render(brandEyebrow())+muted.Render("  "+tr(
		"pourquoi ce verdict est opposable", "why this verdict is defensible")))
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, titre.Render(ctl.Code))
	_, _ = fmt.Fprintln(out, "  "+ctl.TitreIn(i18n.Current()))
	_, _ = fmt.Fprintf(out, "  %s %s\n", muted.Render(tr("sévérité :", "severity:")), ctl.Severite)
	if scsl := strings.Join(ctl.Scsl, ", "); scsl != "" {
		_, _ = fmt.Fprintf(out, "  %s %s\n", muted.Render(tr("exigence SCSL :", "SCSL requirement:")), scsl)
	}
	typ := genprovider.ControlType(ctl.Code)
	if typ == "" {
		typ = tr("(aucun : contrôle transverse)", "(none: cross-cutting control)")
	}
	_, _ = fmt.Fprintf(out, "  %s %s\n", muted.Render(tr("type de ressource lu :", "resource type read:")), typ)

	// Les attributs DÉCISIFS : ceux dont la présence conditionne un « pass ». Sans
	// eux le scan ne conclut pas — c'est le verrou qui empêche un faux vert quand la
	// donnée n'a pas été collectée.
	if attrs := assess.RequiredAttrs()[ctl.Code]; len(attrs) > 0 {
		_, _ = fmt.Fprintf(out, "  %s %s\n",
			muted.Render(tr("attributs décisifs :", "deciding attributes:")),
			strings.Join(attrs, tr(" ou ", " or ")))
	} else {
		_, _ = fmt.Fprintf(out, "  %s %s\n",
			muted.Render(tr("attributs décisifs :", "deciding attributes:")),
			tr("aucun — le contrôle se juge à la présence d'un écart",
				"none — the control is judged on the presence of a deviation"))
	}
	_, _ = fmt.Fprintln(out)
}

// explainProviderSection rend, pour un fournisseur, ce qui alimente la décision et
// ce qu'il faut pour qu'elle conclue.
func explainProviderSection(out interface{ Write([]byte) (int, error) }, ctl referentiel.Control, p string, snap quality.Snapshot) {
	_, _ = fmt.Fprintln(out, titre.Render(p))

	if reason := genprovider.NonApplicableReasonIn(i18n.Current(), p, ctl.Code); reason != "" {
		_, _ = fmt.Fprintf(out, "  %s %s\n\n",
			muted.Render(tr("non applicable :", "not applicable:")), reason)
		return
	}

	desc := genprovider.Descriptors()[p]
	typ := genprovider.ControlType(ctl.Code)

	// Les APPELS D'API qui alimentent la décision : la spec `collecte` du
	// descripteur, jointures comprises. C'est le câblage réel, pas une paraphrase —
	// un endpoint qui n'est pas ici n'alimente rien.
	var calls []string
	for _, r := range desc.Collecte.Resources {
		if r.Type != typ {
			continue
		}
		method := r.Method
		if method == "" {
			method = "GET"
		}
		base := desc.Collecte.BaseURL
		if r.BaseURL != "" {
			base = r.BaseURL
		}
		if r.ForEach != nil {
			calls = append(calls, fmt.Sprintf("%s %s%s", method, base, r.ForEach.Path)+
				muted.Render(tr("  (liste parente)", "  (parent listing)")))
		}
		calls = append(calls, fmt.Sprintf("%s %s%s", method, base, r.Path))
	}
	// Trois types ne viennent PAS de la spec `collecte` mais d'un collecteur Go
	// partagé. Les omettre ferait dire à la commande « aucun appel » là où le scan en
	// fait bel et bien — c'est-à-dire exactement le genre d'angle mort que cette
	// commande existe pour supprimer.
	switch {
	case typ == "object_storage_bucket" && desc.S3.Endpoint != "":
		calls = append(calls, "GET "+desc.S3.Endpoint+
			muted.Render(tr("  (collecteur S3)", "  (S3 collector)")))
	case typ == "kubernetes_cluster" && desc.OKS.Endpoint != "":
		calls = append(calls, "GET "+desc.OKS.Endpoint+
			muted.Render(tr("  (collecteur Kubernetes managé)", "  (managed-Kubernetes collector)")))
	case typ == "iam_policy" && desc.EIM.InlinePolicies:
		calls = append(calls, muted.Render(tr(
			"collecteur de politiques inline (EIM)", "inline policy collector (EIM)")))
	}
	_, _ = fmt.Fprintln(out, "  "+muted.Render(tr("appels d'API (collecte live) :", "API calls (live collection):")))
	if len(calls) == 0 {
		_, _ = fmt.Fprintln(out, "    "+tr(
			"aucun — la spec `collecte` de ce fournisseur ne produit pas ce type",
			"none — this provider's `collecte` spec does not produce this type"))
	}
	for _, c := range calls {
		_, _ = fmt.Fprintln(out, "    "+c)
	}

	// Les ressources Terraform qui alimentent la même décision sur un plan.
	var tfTypes []string
	for _, r := range desc.MappingTerraform.Resources {
		if r.Type == typ {
			tfTypes = append(tfTypes, r.TFType)
		}
	}
	sort.Strings(tfTypes)
	_, _ = fmt.Fprintf(out, "  %s %s\n", muted.Render(tr("ressources Terraform :", "Terraform resources:")),
		orNone(dedupe(tfTypes)))

	// Les CONDITIONS D'UN PASS, dans l'ordre où assess.Build les évalue. Elles sont
	// citées avec leur valeur pour CE couple : une condition énoncée dans l'abstrait
	// n'aide personne à comprendre pourquoi son scan a rendu `not-evaluated`.
	verified := assess.Verified(p, ctl.Code)
	_, _ = fmt.Fprintln(out, "  "+muted.Render(tr("conditions d'un `pass` :", "conditions for a `pass`:")))
	_, _ = fmt.Fprintf(out, "    %s %s\n", mark(true), tr(
		"aucun écart relevé par la règle commune", "no deviation raised by the common rule"))
	_, _ = fmt.Fprintf(out, "    %s %s\n", mark(verified), tr(
		"le contrat du fournisseur déclare le type `verifie`", "the provider contract declares the type `verifie`"))
	_, _ = fmt.Fprintf(out, "    %s %s\n", mark(true), tr(
		"l'inventaire porte au moins une ressource de ce type",
		"the inventory carries at least one resource of this type"))
	_, _ = fmt.Fprintf(out, "    %s %s\n", mark(true), tr(
		"un attribut décisif a réellement été collecté",
		"a deciding attribute was actually collected"))
	_, _ = fmt.Fprintln(out, "    "+muted.Render(tr(
		"(les trois dernières dépendent du scan ; la deuxième est figée par le contrat)",
		"(the last three depend on the scan; the second is fixed by the contract)")))

	// Ce que ce chemin sait PROUVER, de bout en bout et à travers le binaire.
	for _, path := range snap.ByControl[ctl.Code].Paths {
		if path.Provider != p {
			continue
		}
		_, _ = fmt.Fprintf(out, "  %s %s — %s %s / %s %s\n",
			muted.Render(tr("source", "source")), path.Source,
			muted.Render(tr("couverture :", "coverage:")), path.Status,
			muted.Render(tr("verdicts prouvés :", "verdicts proven:")),
			provenSummary(path))
	}
	_, _ = fmt.Fprintln(out)
}

// provenSummary rend « prouvés / à prouver », et nomme ce qui manque : un compteur
// seul ne dit pas quel verdict reste dû.
func provenSummary(p quality.PathProof) string {
	proven := map[string]bool{}
	for _, v := range p.Proven {
		proven[v] = true
	}
	var missing []string
	for _, v := range p.Required {
		if !proven[v] {
			missing = append(missing, v)
		}
	}
	s := fmt.Sprintf("%d/%d", len(p.Proven), len(p.Required))
	if len(missing) > 0 {
		s += muted.Render(tr(" (reste : ", " (left: ") + strings.Join(missing, ", ") + ")")
	}
	return s
}

// explainProof rend les artefacts qui éprouvent le contrôle, et la date de la
// dernière validation live.
func explainProof(out interface{ Write([]byte) (int, error) }, code string, snap quality.Snapshot) {
	proof := snap.ByControl[code]
	// Ces artefacts sont indexés par CONTRÔLE, pas par chemin : un scénario écrit
	// pour un fournisseur éprouve la règle commune, qui est la même pour tous. Le
	// titre le dit, parce qu'avec `--provider` la liste citerait sinon des tenants
	// d'un autre fournisseur sans prévenir — et un lecteur y lirait une couverture
	// qu'il n'a pas demandée.
	_, _ = fmt.Fprintln(out, titre.Render(tr(
		"ce qui l'éprouve (tous fournisseurs confondus)",
		"what tests it (across all providers)")))
	_, _ = fmt.Fprintf(out, "  %s %s\n", muted.Render(tr("tests Rego :", "Rego tests:")), orNone(proof.RegoTests))
	_, _ = fmt.Fprintf(out, "  %s %s\n", muted.Render(tr("scénarios de véracité :", "veracity scenarios:")), orNone(proof.Scenarios))
	_, _ = fmt.Fprintf(out, "  %s %s\n", muted.Render(tr("tenants de référence :", "reference tenants:")), orNone(proof.Tenants))

	// La DATE DE DERNIÈRE VALIDATION LIVE. Un relevé de canari n'est PAS une
	// validation live : il est non authentifié, donc il prouve qu'un endpoint refuse,
	// jamais qu'un droit suffisant rende 200 sur un tenant réel. Le dire ici plutôt
	// que d'afficher la date du canari est la différence entre une preuve et une
	// apparence de preuve.
	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprintln(out, titre.Render(tr("dernière validation live", "last live validation")))
	var live string
	for _, c := range snap.Canary {
		if c.Authenticated {
			live = c.Recorded
		}
	}
	if live == "" {
		_, _ = fmt.Fprintln(out, "  "+tr(
			"jamais — aucun scan authentifié contre un tenant réel n'est consigné.",
			"never — no authenticated scan against a real tenant is recorded."))
		_, _ = fmt.Fprintln(out, "  "+muted.Render(tr(
			"Les relevés de canari (references/canary/) interrogent les vrais plans de contrôle",
			"The canary records (references/canary/) query the real control planes")))
		_, _ = fmt.Fprintln(out, "  "+muted.Render(tr(
			"SANS identifiant : ils attestent qu'un endpoint existe et refuse, pas un verdict.",
			"WITHOUT a credential: they attest that an endpoint exists and refuses, not a verdict.")))
	} else {
		_, _ = fmt.Fprintf(out, "  %s\n", live)
	}
	for _, c := range snap.Canary {
		_, _ = fmt.Fprintf(out, "  %s %s %s\n", muted.Render(tr("canari", "canary")+" "+c.Provider),
			c.Recorded, muted.Render(fmt.Sprintf(tr("(%d ont répondu, %d déplacés, %d injoignables)",
				"(%d answered, %d moved, %d unreachable)"), c.Answered, c.Moved, c.Unreachable)))
	}
	_, _ = fmt.Fprintln(out)
}

// mark rend une pastille de condition. Une condition qui dépend du scan est rendue
// neutre : la marquer « remplie » ici mentirait, puisque rien n'a été scanné.
func mark(fixed bool) string {
	if fixed {
		return "·"
	}
	return "✘"
}

func orNone(list []string) string {
	if len(list) == 0 {
		return muted.Render(tr("aucun", "none"))
	}
	return strings.Join(list, ", ")
}

func dedupe(list []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range list {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

func init() {
	controlExplainCmd.Flags().StringVar(&explainProvider, "provider", "",
		"limiter l'explication à un fournisseur")
	controlCmd.AddCommand(controlExplainCmd)
}

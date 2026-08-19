package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"

	"github.com/stephrobert/pepin/internal/genprovider"
	"github.com/stephrobert/pepin/referentiel"
)

var scslIndex string

// scslCmd vérifie la cohérence des contrôles Pépin vis-à-vis de l'index SCSL
// (module posture cloud) et imprime la roadmap des providers : désalignements de
// mapping, exigences citant un code Pépin manquant, et exigences outillables par
// Pépin non encore couvertes.
var scslCmd = &cobra.Command{
	Use:   "scsl",
	Short: "Vérifier la cohérence avec l'index SCSL et piloter la roadmap",
	RunE: func(_ *cobra.Command, _ []string) error {
		exs, err := referentiel.ParseCLDExigences(scslIndex)
		if err != nil {
			return err
		}
		desal, roadmap := referentiel.AnalyseCoherence(exs)

		// Couverture par famille + codes Pépin couvrant chaque exigence.
		ctrls := referentiel.All()
		covered := map[string]bool{}
		coveredBy := map[string][]string{}
		for code, c := range ctrls {
			for _, s := range c.Scsl {
				covered[s] = true
				coveredBy[s] = append(coveredBy[s], code)
			}
		}
		texte := map[string]string{}
		for _, e := range exs {
			texte[e.ID] = e.Texte
		}
		type stat struct{ tot, cov int }
		fams := map[string]*stat{}
		var order []string
		for _, e := range exs {
			s, ok := fams[e.Famille]
			if !ok {
				s = &stat{}
				fams[e.Famille] = s
				order = append(order, e.Famille)
			}
			s.tot++
			if covered[e.ID] {
				s.cov++
			}
		}
		sort.Strings(order)

		fmt.Println()
		fmt.Println(eyebrow.Render(brandEyebrow()) + muted.Render(fmt.Sprintf(tr(
			"  cohérence SCSL — %d exigences cloud (CLD-*)",
			"  SCSL consistency — %d cloud requirements (CLD-*)"), len(exs))))

		// Socle essentiel : exigences `essentielle` (fondamentaux). Ce sont des
		// AGRÉGATS des contrôles fins — satisfaits dès que leurs sous-contrôles le
		// sont. CLD-0-1 (inventaire/CSPM) l'est par l'exécution même de Pépin.
		essComp := map[string][]string{
			"CLD-GEN-2": {"CLD-IAM-1", "CLD-IAM-3"}, // IAM moindre privilège + MFA admin
			"CLD-GEN-3": {"CLD-NET-1", "CLD-STO-1"}, // rien d'exposé par défaut
			"CLD-GEN-4": {"CLD-CHF-2", "CLD-LOG-1"}, // chiffrement au repos + journalisation
		}
		essCov := func(id string) bool {
			if covered[id] {
				return true
			}
			comp := essComp[id]
			if len(comp) == 0 {
				return id == "CLD-GEN-1" // inventaire/CSPM : assuré par Pépin lui-même
			}
			for _, c := range comp {
				if !covered[c] {
					return false
				}
			}
			return true
		}
		var ess []referentiel.CLDExigence
		for _, e := range exs {
			if e.Essentielle {
				ess = append(ess, e)
			}
		}
		if len(ess) > 0 {
			n := 0
			for _, e := range ess {
				if essCov(e.ID) {
					n++
				}
			}
			fmt.Println()
			fmt.Println(titre.Render(fmt.Sprintf(tr("  Socle essentiel  %d/%d", "  Essential baseline  %d/%d"), n, len(ess))) +
				muted.Render(tr("  (fondamentaux — agrégat des contrôles fins)", "  (fundamentals — an aggregate of the fine-grained controls)")))
			for _, e := range ess {
				mark := errStyle.Render("·")
				if essCov(e.ID) {
					mark = titre.Render("✓")
				}
				txt := []rune(e.Texte)
				if len(txt) > 66 {
					txt = txt[:66]
				}
				fmt.Printf("    %s %-9s %s\n", mark, e.ID, muted.Render(string(txt)))
			}
		}

		fmt.Println()
		fmt.Println(titre.Render(tr("  Couverture par famille", "  Coverage by family")))
		totCov, tot := 0, 0
		for _, f := range order {
			s := fams[f]
			tot += s.tot
			totCov += s.cov
			fmt.Printf("    %-6s %2d/%-2d\n", f, s.cov, s.tot)
		}
		fmt.Printf("    %-6s %2d/%-2d\n", "TOTAL", totCov, tot)

		// Matrice de couverture par provider (exigence × provider), d'après le
		// champ `fournisseurs` (providers qui collectent le type visé par le contrôle).
		provSet := map[string]bool{}
		covProv := map[string]map[string]bool{} // provider -> ensemble d'exigences
		for _, c := range ctrls {
			for _, p := range c.Fournisseurs {
				provSet[p] = true
				if covProv[p] == nil {
					covProv[p] = map[string]bool{}
				}
				for _, s := range c.Scsl {
					covProv[p][s] = true
				}
			}
		}
		var provs []string
		for p := range provSet {
			provs = append(provs, p)
		}
		sort.Strings(provs)

		var covIDs []string
		for id := range covered {
			covIDs = append(covIDs, id)
		}
		sort.Strings(covIDs)

		// nonApplicable : tous les contrôles couvrant l'exigence visent un type
		// dont le contrat du provider est `absent` (API n'expose pas) → non testable.
		nonApplicable := func(p, e string) bool {
			codes := coveredBy[e]
			if len(codes) == 0 {
				return false
			}
			for _, c := range codes {
				if !genprovider.ControlNonApplicable(p, c) {
					return false
				}
			}
			return true
		}

		okMark := lipgloss.NewStyle().Foreground(lipgloss.Color("#3ddc84")).Bold(true).Render("✓")
		noMark := muted.Render("·")
		naMark := lipgloss.NewStyle().Foreground(colMuted).Render("n/a")
		headers := append([]string{tr("Exigence", "Requirement")}, provs...)
		var rows [][]string
		for _, id := range covIDs {
			row := []string{id}
			for _, p := range provs {
				switch {
				case covProv[p][id]:
					row = append(row, okMark)
				case nonApplicable(p, id):
					row = append(row, naMark)
				default:
					row = append(row, noMark)
				}
			}
			rows = append(rows, row)
		}
		totalRow := []string{"TOTAL"}
		for _, p := range provs {
			totalRow = append(totalRow, fmt.Sprintf("%d", len(covProv[p])))
		}
		rows = append(rows, totalRow)

		tbl := table.New().
			Border(lipgloss.RoundedBorder()).
			BorderStyle(muted).
			Headers(headers...).
			Rows(rows...).
			StyleFunc(func(row, col int) lipgloss.Style {
				s := lipgloss.NewStyle().Padding(0, 1)
				if row == table.HeaderRow || col > 0 {
					s = s.Align(lipgloss.Center)
				}
				if row == table.HeaderRow || row == len(rows)-1 {
					s = s.Bold(true)
				}
				return s
			})
		fmt.Println()
		fmt.Println(titre.Render(tr("  Couverture par provider", "  Coverage by provider")))
		fmt.Println(tbl)

		// Roadmap de PARITÉ : une exigence couverte par un provider est observable,
		// donc faisable chez les autres. On liste, par provider, les exigences
		// couvertes ailleurs mais pas chez lui (applicabilité : contrat de providers/<nom>.yaml).
		fmt.Println()
		fmt.Println(titre.Render(tr("  Roadmap de parité par provider", "  Parity roadmap by provider")) +
			muted.Render(tr("  (couvert ailleurs → à étendre)", "  (covered elsewhere → to extend)")))
		// Les providers d'une AUTRE portée (ex. `kubernetes`, qui audite l'intérieur d'un
		// cluster) sont exclus de la parité : leurs exigences ne sont pas atteignables par
		// un plan de contrôle cloud, et réciproquement. Les comparer produirait une roadmap
		// absurde (réclamer du RBAC in-cluster à un cloud, ou des buckets à Kubernetes).
		nonCloud := genprovider.NonCloudProviders()
		for _, p := range provs {
			if nonCloud[p] {
				continue
			}
			var extend, na []string
			for _, id := range covIDs {
				others := false
				for _, q := range provs {
					if q != p && !nonCloud[q] && covProv[q][id] {
						others = true
					}
				}
				if !others || covProv[p][id] {
					continue
				}
				if nonApplicable(p, id) {
					na = append(na, id)
				} else {
					extend = append(extend, id)
				}
			}
			if len(extend) == 0 && len(na) == 0 {
				fmt.Printf("    %-9s %s\n", p, muted.Render(tr("parité atteinte ✓", "parity reached ✓")))
				continue
			}
			label := p
			if len(extend) > 0 {
				fmt.Printf("    %-9s %s %s\n", label, titre.Render(fmt.Sprintf(tr("(%d à étendre)", "(%d to extend)"), len(extend))), muted.Render(strings.Join(extend, ", ")))
				label = ""
			}
			if len(na) > 0 {
				fmt.Printf("    %-9s %s %s\n", label, muted.Render(fmt.Sprintf(tr(
					"(%d non applicable — API n'expose pas)",
					"(%d not applicable — the API does not expose it)"), len(na))), muted.Render(strings.Join(na, ", ")))
			}
		}

		// Exigences couvertes (triées), avec le(s) code(s) Pépin qui les traitent.
		var ids []string
		for id := range coveredBy {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		fmt.Println()
		fmt.Println(titre.Render(fmt.Sprintf(tr("  Couvertes (%d)", "  Covered (%d)"), len(ids))))
		for _, id := range ids {
			codes := coveredBy[id]
			sort.Strings(codes)
			fmt.Printf("    %s  %s\n", titre.Render(id), muted.Render(texte[id]))
			fmt.Printf("      %s\n", muted.Render("→ "+strings.Join(codes, ", ")))
		}

		fmt.Println()
		if len(desal) == 0 {
			fmt.Println(titre.Render(tr("  Désalignements", "  Misalignments")) + muted.Render(tr("  aucun ✓", "  none ✓")))
		} else {
			fmt.Println(errStyle.Render(fmt.Sprintf(tr(
				"  Désalignements (%d) — mapping `scsl` à corriger",
				"  Misalignments (%d) — the `scsl` mapping must be fixed"), len(desal))))
			for _, d := range desal {
				fmt.Printf("    %s  %s\n", errStyle.Render(d.Scsl), d.Detail)
			}
		}

		fmt.Println()
		fmt.Println(titre.Render(fmt.Sprintf(tr(
			"  Roadmap (%d) — exigences outillables par Pépin, non couvertes",
			"  Roadmap (%d) — requirements Pepin could tool, not covered yet"), len(roadmap))))
		for _, r := range roadmap {
			tag := tr("non couverte", "not covered")
			if r.Type == "a_implementer" {
				tag = tr("code cité absent : ", "cited code missing: ") + r.Code
			}
			fmt.Printf("    %s  %s %s\n", titre.Render(r.Scsl), muted.Render("["+tag+"]"), r.Detail)
		}
		fmt.Println()
		return nil
	},
}

func init() {
	scslCmd.Flags().StringVar(&scslIndex, "index", "../framework-scsl/api/v1/exigences.json",
		"chemin de l'API SCSL (api/v1/exigences.json du framework)")
}

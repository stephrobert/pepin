package docgen

import (
	"fmt"
	"sort"
	"strings"
)

// mark est le symbole d'un statut dans la matrice. Volontairement contrasté : une case qui
// n'est pas ✅ doit se voir à la lecture rapide, sinon la matrice rassure au lieu d'informer.
func mark(s Status) string {
	switch s {
	case Supported:
		return "✅"
	case Partial:
		return "◐"
	case NotApplicable:
		return "∅"
	default:
		return "✗"
	}
}

// coveragePage rend docs/coverage.md (lang="en") ou docs/coverage.fr.md (lang="fr").
func (m Matrix) coveragePage(lang string) string {
	t := coverageText(lang)
	var b strings.Builder
	if lang == "fr" {
		b.WriteString("> [🇬🇧 English](coverage.md) · 🇫🇷 Français\n\n")
	} else {
		b.WriteString("> 🇬🇧 English · [🇫🇷 Français](coverage.fr.md)\n\n")
	}
	b.WriteString(generatedBanner(lang, "mise run gen-docs"))
	b.WriteString("\n# " + t.title + "\n\n")
	b.WriteString(t.intro + "\n\n")

	b.WriteString("## " + t.legendTitle + "\n\n")
	b.WriteString("| " + t.colMark + " | " + t.colStatus + " | " + t.colMeans + " |\n|---|---|---|\n")
	for _, s := range []Status{Supported, Partial, NotApplicable, Unsupported} {
		_, _ = fmt.Fprintf(&b, "| %s | `%s` | %s |\n", mark(s), s, t.legend[s])
	}
	b.WriteString("\n" + t.legendNote + "\n\n")

	b.WriteString("## " + t.summaryTitle + "\n\n")
	b.WriteString(m.summaryTable(t))
	b.WriteString("\n" + t.matrixTitle + "\n\n")
	b.WriteString(m.matrixTable(t))
	b.WriteString("\n## " + t.reasonsTitle + "\n\n")
	b.WriteString(t.reasonsIntro + "\n\n")
	b.WriteString(m.reasonsTable(t))
	if len(m.OtherProviders) > 0 {
		b.WriteString("\n## " + t.otherTitle + "\n\n")
		b.WriteString(t.otherIntro + "\n\n")
		b.WriteString(m.otherTable(t))
	}
	b.WriteString("\n## " + t.countsTitle + "\n\n")
	b.WriteString(m.countsTable(t))
	return b.String()
}

// summaryTable agrège la couverture par famille du référentiel et par fournisseur : un
// contrôle compte ✅ pour un fournisseur dès qu'AU MOINS UNE source sait le conclure.
func (m Matrix) summaryTable(t coverageStrings) string {
	families := map[string]bool{}
	for _, r := range m.Rows {
		families[r.Family] = true
	}
	fams := make([]string, 0, len(families))
	for f := range families {
		fams = append(fams, f)
	}
	sort.Strings(fams)

	var b strings.Builder
	b.WriteString("| " + t.colFamily + " | " + t.colControls + " |")
	for _, p := range m.CloudProviders {
		b.WriteString(" " + p + " |")
	}
	b.WriteString("\n|---|---:|")
	for range m.CloudProviders {
		b.WriteString("---|")
	}
	b.WriteString("\n")
	for _, f := range fams {
		total := 0
		per := map[string]map[Status]int{}
		for _, p := range m.CloudProviders {
			per[p] = map[Status]int{}
		}
		for _, r := range m.Rows {
			if r.Family != f {
				continue
			}
			total++
			for _, p := range m.CloudProviders {
				per[p][bestStatus(r, p)]++
			}
		}
		_, _ = fmt.Fprintf(&b, "| `%s` | %d |", f, total)
		for _, p := range m.CloudProviders {
			c := per[p]
			_, _ = fmt.Fprintf(&b, " %s %d · %s %d · %s %d · %s %d |",
				mark(Supported), c[Supported], mark(Partial), c[Partial],
				mark(NotApplicable), c[NotApplicable], mark(Unsupported), c[Unsupported])
		}
		b.WriteString("\n")
	}
	return b.String()
}

// bestStatus rend le meilleur statut atteint par un contrôle chez un fournisseur, toutes
// sources confondues : c'est la question qu'on se pose avant d'adopter l'outil (« ce contrôle
// est-il mesurable ici, d'une façon ou d'une autre ? »).
func bestStatus(r Row, provider string) Status {
	order := map[Status]int{Supported: 3, Partial: 2, NotApplicable: 1, Unsupported: 0}
	best := Unsupported
	for _, src := range []Source{SourceTerraform, SourceLive} {
		if c, ok := r.Cells[provider][src]; ok && order[c.Status] > order[best] {
			best = c.Status
		}
	}
	return best
}

func (m Matrix) matrixTable(t coverageStrings) string {
	var b strings.Builder
	b.WriteString("| " + t.colControl + " | " + t.colSeverity + " | SCSL |")
	for _, p := range m.CloudProviders {
		_, _ = fmt.Fprintf(&b, " %s TF | %s live |", p, p)
	}
	b.WriteString("\n|---|---|---|")
	for range m.CloudProviders {
		b.WriteString(":-:|:-:|")
	}
	b.WriteString("\n")
	for _, r := range m.Rows {
		_, _ = fmt.Fprintf(&b, "| `%s` | %s | %s |", r.Code, r.Severity, strings.Join(r.SCSL, ", "))
		for _, p := range m.CloudProviders {
			b.WriteString(" " + mark(r.Cells[p][SourceTerraform].Status) + " |")
			b.WriteString(" " + mark(r.Cells[p][SourceLive].Status) + " |")
		}
		b.WriteString("\n")
	}
	return b.String()
}

// reasonsTable détaille chaque case qui n'est pas ✅ ALORS QUE le contrôle est déclaré pour ce
// fournisseur. C'est la partie utile : « non supporté » sans motif n'est pas opposable, et une
// case non applicable sans justification serait rejetée par un auditeur.
func (m Matrix) reasonsTable(t coverageStrings) string {
	var b strings.Builder
	b.WriteString("| " + t.colControl + " | " + t.colProvider + " | " + t.colSource + " | " + t.colStatus + " | " + t.colReason + " |\n|---|---|---|---|---|\n")
	rows := 0
	for _, r := range m.Rows {
		for _, p := range m.CloudProviders {
			for _, src := range []Source{SourceTerraform, SourceLive} {
				c := r.Cells[p][src]
				if c.Status == Supported {
					continue
				}
				if c.Undeclared {
					continue // trivial : déjà lisible dans la matrice, et sans information
				}
				_, _ = fmt.Fprintf(&b, "| `%s` | %s | %s | %s `%s` | %s |\n",
					r.Code, p, src, mark(c.Status), c.Status, oneLine(c.Reason))
				rows++
			}
		}
	}
	if rows == 0 {
		return t.noReasons + "\n"
	}
	return b.String()
}

// otherTable rend les fournisseurs d'une autre portée (in-cluster) dans leur propre tableau :
// les mettre dans la matrice cloud suggérerait une parité qui n'a aucun sens.
func (m Matrix) otherTable(t coverageStrings) string {
	var b strings.Builder
	b.WriteString("| " + t.colControl + " | " + t.colSeverity + " |")
	for _, p := range m.OtherProviders {
		b.WriteString(" " + p + " |")
	}
	b.WriteString("\n|---|---|")
	for range m.OtherProviders {
		b.WriteString(":-:|")
	}
	b.WriteString("\n")
	for _, r := range m.Rows {
		shown := false
		for _, p := range m.OtherProviders {
			if r.Cells[p][SourceLive].Status != Unsupported {
				shown = true
			}
		}
		if !shown {
			continue
		}
		_, _ = fmt.Fprintf(&b, "| `%s` | %s |", r.Code, r.Severity)
		for _, p := range m.OtherProviders {
			b.WriteString(" " + mark(r.Cells[p][SourceLive].Status) + " |")
		}
		b.WriteString("\n")
	}
	return b.String()
}

// countsTable donne les totaux par fournisseur et par source — le chiffre qu'on cite, et
// qu'on ne veut donc surtout pas maintenir à la main.
func (m Matrix) countsTable(t coverageStrings) string {
	var b strings.Builder
	b.WriteString("| " + t.colProvider + " | " + t.colSource + " | " + mark(Supported) + " `supported` | " +
		mark(Partial) + " `partial` | " + mark(NotApplicable) + " `not-applicable` | " +
		mark(Unsupported) + " `unsupported` |\n|---|---|---:|---:|---:|---:|\n")
	row := func(p string, src Source) {
		c := map[Status]int{}
		for _, r := range m.Rows {
			c[r.Cells[p][src].Status]++
		}
		_, _ = fmt.Fprintf(&b, "| %s | %s | %d | %d | %d | %d |\n",
			p, src, c[Supported], c[Partial], c[NotApplicable], c[Unsupported])
	}
	for _, p := range m.CloudProviders {
		row(p, SourceTerraform)
		row(p, SourceLive)
	}
	// Les fournisseurs d'une autre portée n'ont pas de mapping Terraform du tout : afficher une
	// ligne « terraform : 0 supporté » se lirait comme un support cassé, alors que la source
	// n'existe pas pour eux.
	for _, p := range m.OtherProviders {
		row(p, SourceLive)
	}
	return b.String()
}

// oneLine aplatit une justification multi-lignes du contrat pour tenir dans une cellule de
// tableau Markdown, sans en retirer un mot.
func oneLine(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "|", "\\|")
	return strings.Join(strings.Fields(s), " ")
}

// generatedBanner : l'avertissement qui empêche une édition manuelle de se perdre au
// prochain `mise run gen-docs`.
func generatedBanner(lang, cmd string) string {
	if lang == "fr" {
		return "<!-- PAGE GÉNÉRÉE : ne pas éditer à la main. Régénérer avec `" + cmd + "`. -->\n"
	}
	return "<!-- GENERATED PAGE — do not edit by hand. Regenerate with `" + cmd + "`. -->\n"
}

// coverageStrings porte les libellés de la page dans une langue.
type coverageStrings struct {
	title, intro                                      string
	legendTitle, legendNote                           string
	legend                                            map[Status]string
	summaryTitle, matrixTitle, reasonsTitle           string
	reasonsIntro, noReasons                           string
	otherTitle, otherIntro, countsTitle               string
	colMark, colStatus, colMeans, colFamily           string
	colControls, colControl, colSeverity, colProvider string
	colSource, colReason                              string
}

func coverageText(lang string) coverageStrings {
	if lang == "fr" {
		return coverageStrings{
			title: "Matrice de couverture",
			intro: "Ce que Pépin sait réellement mesurer, par contrôle, par fournisseur et par source.\n" +
				"La page est **calculée** depuis `referentiel/controles.yaml` (codes, sévérités, `fournisseurs`),\n" +
				"`providers/*.yaml` (spec de collecte live, mapping Terraform, contrat d'API) et le verrou du\n" +
				"« pass » de `internal/assess`. Elle décrit ce que Pépin **peut conclure**, pas le résultat d'un\n" +
				"scan donné : un contrôle marqué ✅ peut parfaitement rendre `not-evaluated` sur un inventaire\n" +
				"qui ne contient aucune ressource du type visé.\n\n" +
				"**Ce que la colonne « live » est, et ce qu'elle n'est pas.** Elle est DÉRIVÉE des\n" +
				"descripteurs : elle dit quel type de ressource la spec `collecte` déclare produire, pas ce\n" +
				"qu'un fournisseur a effectivement répondu. Deux choses sont désormais MESURÉES, et pas\n" +
				"seulement déclarées : que les endpoints annoncés partent réellement sur le réseau, et que\n" +
				"les jointures parent/enfant tirent. Cela vient d'un enregistrement d'appels réels, rejoué à\n" +
				"chaque build (`internal/genprovider/testdata/transcripts/`, cf.\n" +
				"[Tracer les appels réels](guides/tracing-api-calls.fr.md)). Cet enregistrement a été pris\n" +
				"contre un ÉMULATEUR LOCAL : il prouve ce que Pépin fait, jamais ce que le fournisseur\n" +
				"répond. Les noms et types de champs du contrat natif, les bornes réelles de pagination et\n" +
				"le comportement devant un refus de droits restent NON OBSERVÉS ; ils sont dus à un scan\n" +
				"réel, que ce dépôt ne fait pas puisqu'il ne détient aucun identifiant cloud.\n\n" +
				"Les titres de contrôles et les justifications proviennent du référentiel et des contrats de\n" +
				"fournisseurs, qui sont bilingues : cette page cite la version française, la page anglaise\n" +
				"cite la version anglaise. Le français reste la langue de référence du contenu normatif.",
			legendTitle: "Légende",
			legend: map[Status]string{
				Supported:     "la source produit le type de ressource visé, le contrat le déclare `verifie`, et l'attribut décisif est projeté. Pépin peut rendre `pass` ou `fail`.",
				Partial:       "la source produit le type, mais le verrou du « pass » ne peut pas être levé (contrat non `verifie`, ou attribut décisif non projeté par cette source). Le scan rend `not-evaluated`, jamais un vert silencieux.",
				NotApplicable: "le contrat du fournisseur déclare le contrôle non testable, avec sa justification (mécanisme inexistant côté API, ou type de ressource absent). Le scan rend `not-applicable` accompagné de ce motif.",
				Unsupported:   "le contrôle n'est pas déclaré pour ce fournisseur dans le référentiel, ou cette source ne produit aucune ressource du type qu'il lit. Rien ne sera conclu depuis cette source.",
			},
			legendNote: "**◐ n'est pas « à moitié conforme ».** C'est « Pépin ne peut pas décider depuis cette source »,\n" +
				"et le rapport le dit à chaque scan, contrôle par contrôle, avec le motif.",
			summaryTitle: "Synthèse par famille du référentiel",
			matrixTitle:  "## Matrice complète (contrôle × fournisseur × source)",
			reasonsTitle: "Pourquoi une case n'est pas ✅",
			reasonsIntro: "Une case qui n'est pas ✅ **alors que le contrôle est déclaré pour ce fournisseur** porte\n" +
				"toujours son motif. Les cases « contrôle non déclaré pour ce fournisseur » ne sont pas reprises\n" +
				"ici : la matrice les montre déjà, et elles n'apprennent rien de plus.",
			noReasons:   "_Aucune : toutes les cases déclarées sont pleinement observables._",
			otherTitle:  "Fournisseur d'une autre portée : `kubernetes` (in-cluster)",
			otherIntro:  "Ce fournisseur audite l'état **dans** un cluster (RBAC, Pod Security, NetworkPolicy), pas un\nplan de contrôle cloud. Le comparer en parité avec un cloud n'aurait pas de sens : aucun des\ndeux ne peut couvrir la portée de l'autre. Une seule source : la collecte live via kubeconfig.",
			countsTitle: "Totaux",
			colMark:     "Symbole", colStatus: "Statut", colMeans: "Ce que Pépin peut conclure",
			colFamily: "Famille", colControls: "Contrôles", colControl: "Contrôle",
			colSeverity: "Sévérité", colProvider: "Fournisseur", colSource: "Source", colReason: "Motif",
		}
	}
	return coverageStrings{
		title: "Coverage matrix",
		intro: "What Pépin actually measures, per control, per provider and per source.\n" +
			"This page is **computed** from `referentiel/controles.yaml` (codes, severities, `fournisseurs`),\n" +
			"`providers/*.yaml` (live collection spec, Terraform mapping, API contract) and the `pass` lock in\n" +
			"`internal/assess`. It describes what Pépin **can conclude**, not the result of any given scan: a\n" +
			"control marked ✅ may well return `not-evaluated` on an inventory that contains no resource of the\n" +
			"type it reads.\n\n" +
			"**What the \"live\" column is, and what it is not.** It is DERIVED from the descriptors: it\n" +
			"says which resource type the `collecte` spec declares it produces, not what a provider\n" +
			"actually answered. Two things are now MEASURED rather than merely declared: that the announced\n" +
			"endpoints really go out on the wire, and that parent/child joins fire. That comes from a\n" +
			"recording of real calls, replayed on every build (`internal/genprovider/testdata/transcripts/`,\n" +
			"see [Tracing real API calls](guides/tracing-api-calls.md)). That recording was taken against a\n" +
			"LOCAL EMULATOR: it proves what Pépin does, never what the provider answers. The field names and\n" +
			"types of the native contract, the real pagination bounds and the behaviour on a permission\n" +
			"refusal remain UNOBSERVED; they are owed to a real scan, which this repository does not run\n" +
			"because it holds no cloud credentials.\n\n" +
			"Control titles and justifications come from the reference and from the provider contracts, which\n" +
			"are bilingual: this page quotes their English version, the French page quotes the French one.\n" +
			"French remains the reference language of the normative content.",
		legendTitle: "Legend",
		legend: map[Status]string{
			Supported:     "the source produces the resource type the control reads, the contract marks it `verifie`, and the deciding attribute is projected. Pépin can return `pass` or `fail`.",
			Partial:       "the source produces the type, but the `pass` lock cannot be lifted (contract not `verifie`, or the deciding attribute is not projected by this source). The scan returns `not-evaluated` — never a silent green.",
			NotApplicable: "the provider contract declares the control untestable, with its justification (no such mechanism in the API, or the resource type does not exist). The scan returns `not-applicable` together with that reason.",
			Unsupported:   "the control is not declared for this provider in the reference, or this source produces no resource of the type it reads. Nothing will be concluded from this source.",
		},
		legendNote: "**◐ does not mean \"half compliant\".** It means \"Pépin cannot decide from this source\",\n" +
			"and the report says so on every scan, control by control, with the reason.",
		summaryTitle: "Summary by reference family",
		matrixTitle:  "## Full matrix (control × provider × source)",
		reasonsTitle: "Why a cell is not ✅",
		reasonsIntro: "Every cell that is not ✅ **while the control is declared for that provider** carries its\n" +
			"reason. Cells that merely say \"control not declared for this provider\" are left out: the matrix\n" +
			"already shows them, and they add nothing.",
		noReasons:   "_None: every declared cell is fully observable._",
		otherTitle:  "Different-scope provider: `kubernetes` (in-cluster)",
		otherIntro:  "This provider audits the state **inside** a cluster (RBAC, Pod Security, NetworkPolicy), not a\ncloud control plane. Comparing it with a cloud for parity would be meaningless: neither can cover\nthe other's scope. One source only: live collection through a kubeconfig.",
		countsTitle: "Totals",
		colMark:     "Mark", colStatus: "Status", colMeans: "What Pépin can conclude",
		colFamily: "Family", colControls: "Controls", colControl: "Control",
		colSeverity: "Severity", colProvider: "Provider", colSource: "Source", colReason: "Reason",
	}
}

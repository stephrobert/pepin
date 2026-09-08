package docgen

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/stephrobert/pepin/internal/canary"
	"github.com/stephrobert/pepin/internal/quality"
	"github.com/stephrobert/pepin/internal/tenants"
	"github.com/stephrobert/pepin/internal/veracity"
	"github.com/stephrobert/pepin/referentiel"
)

// regoRulesDir : les règles communes, où se lisent les tests qui citent un code.
const regoRulesDir = "internal/commonrules/rules"

// BuildQuality dérive la carte de qualité de détection depuis les artefacts du
// dépôt. Un seul calcul, consommé par la page générée ET par `pepin control
// explain` (via l'instantané committé) : deux calculs divergeraient, et celui qui
// diverge est celui qu'on lit.
func BuildQuality(root string, m Matrix) (quality.Snapshot, error) {
	files, err := veracity.LoadScenarios(filepath.Join(root, veracityScenarios))
	if err != nil {
		return quality.Snapshot{}, err
	}
	fromTenants, err := tenants.Covered(root)
	if err != nil {
		return quality.Snapshot{}, err
	}
	tenantsByPath, err := tenants.CoveredBy(root)
	if err != nil {
		return quality.Snapshot{}, err
	}
	list, err := tenants.Load(root)
	if err != nil {
		return quality.Snapshot{}, err
	}
	records, err := canary.Load(root)
	if err != nil {
		return quality.Snapshot{}, err
	}
	regoTests, err := regoTestsByControl(root)
	if err != nil {
		return quality.Snapshot{}, err
	}

	// Les chemins sont rendus RELATIFS à la racine du dépôt. `root` est absolu quand
	// le générateur tourne, et un instantané committé qui porterait le disque du
	// mainteneur serait à la fois inutile pour un lecteur et un renseignement gratuit
	// sur sa machine.
	scenariosByPath := map[veracity.Path][]string{}
	for _, f := range files {
		rel, rerr := filepath.Rel(root, f.Path)
		if rerr != nil {
			rel = f.Path
		}
		scenariosByPath[f.PathOf()] = append(scenariosByPath[f.PathOf()], filepath.ToSlash(rel))
	}
	// Les contre-témoins : des tenants TIERS déclarés durcis, sur lesquels Pépin ne
	// relève aucun écart critical/high. TestEveryPostureIsTheOneMeasured confronte
	// cette déclaration au scan, donc ce compteur est mesuré et non annoncé.
	counterwitnesses := 0
	for _, t := range list {
		if t.Posture == tenants.PostureHardened {
			counterwitnesses++
		}
	}
	return quality.Compute(quality.Inputs{
		Cells:              VeracityCells(m),
		Covered:            veracity.Merge(veracity.Covered(files), fromTenants),
		Controls:           len(referentiel.All()),
		Records:            records,
		ScenariosByPath:    scenariosByPath,
		TenantsByPath:      tenantsByPath,
		RegoTestsByControl: regoTests,
		Counterwitnesses:   counterwitnesses,
		TenantsTotal:       len(list),
	}), nil
}

// regoTestsByControl rend, par code de contrôle, les fichiers `*_test.rego` qui le
// citent. Le lien est fait par la MENTION du code dans le test, pas par une table
// tenue à la main : une table se périme au premier renommage, une mention non.
func regoTestsByControl(root string) (map[string][]string, error) {
	dir := filepath.Join(root, regoRulesDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("lecture de %s : %w", dir, err)
	}
	codes := referentiel.All()
	out := map[string][]string{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), "_test.rego") {
			continue
		}
		raw, rerr := os.ReadFile(filepath.Join(dir, e.Name())) // #nosec G304 -- règles du dépôt.
		if rerr != nil {
			return nil, fmt.Errorf("lecture de %s : %w", e.Name(), rerr)
		}
		body := string(raw)
		for code := range codes {
			if strings.Contains(body, code) {
				out[code] = append(out[code], regoRulesDir+"/"+e.Name())
			}
		}
	}
	for code := range out {
		sort.Strings(out[code])
	}
	return out, nil
}

// verdictRow est la ligne d'un verdict dans la ventilation : sa clé, son nom
// affiché, et ce qu'il met en scène.
//
// Ces quatre libellés vivent dans leur propre table plutôt que dans qualityText,
// et l'arbitrage mérite sa ligne : un champ de structure nommé d'après le verdict
// `pass` déclenche le G101 de gosec (« identifiant en dur »), qui apparie un NOM de
// champ contenant « pass » avec une chaîne littérale. C'est un faux positif, mais la
// correction acceptée ici est le renommage, jamais un `//nolint` — et une table
// ordonnée, indexée par la clé du verdict, est de toute façon la meilleure forme :
// elle porte l'ORDRE d'affichage, que quatre champs indépendants ne portaient pas.
type verdictRow struct {
	key    string
	name   string
	staged string
}

// verdictRows rend les quatre verdicts dans l'ordre où la page les présente.
func verdictRows(lang string) []verdictRow {
	if lang == "fr" {
		return []verdictRow{
			{string(veracity.Fail), "`fail`", "une configuration vulnérable est détectée"},
			{string(veracity.Pass), "`pass`", "une configuration réellement correcte est confirmée"},
			{string(veracity.NotEvaluated), "`not-evaluated`", "l'attribut décisif manque, et le scan refuse de conclure"},
			{string(veracity.NotApplicable), "`not-applicable`", "le contrat du fournisseur déclare le mécanisme inexistant"},
		}
	}
	return []verdictRow{
		{string(veracity.Fail), "`fail`", "a vulnerable configuration is detected"},
		{string(veracity.Pass), "`pass`", "a genuinely correct configuration is confirmed"},
		{string(veracity.NotEvaluated), "`not-evaluated`", "the deciding attribute is missing, and the scan refuses to conclude"},
		{string(veracity.NotApplicable), "`not-applicable`", "the provider's contract declares the mechanism non-existent"},
	}
}

// qualityText porte les libellés de la page, dans une langue.
type qualityText struct {
	title, intro, ruleTitle, ruleBody                      string
	figures, colFigure, colCount                           string
	controls, paths, pathsProven, obligations, provenWord  string
	veracityTitle, veracityIntro                           string
	colVerdict, colMeaning, colRequired, colProven, colPct string
	liveTitle, liveBody, liveRow                           string
	canaryTitle, canaryIntro                               string
	colProvider, colRecorded, colAnswered, colMoved, colUn string
	fpTitle, fpBody, counterwitnesses, tenantsWord         string
	blindTitle, blindBody                                  string
	generated                                              string
}

func qualityStrings(lang string) qualityText {
	if lang == "fr" {
		return qualityText{
			title: "Carte de qualité de détection",
			intro: "Ce que Pépin peut PROUVER de ses propres verdicts, et ce qu'il ne peut pas.\n" +
				"Chaque chiffre de cette page est dérivé des artefacts du dépôt — obligations\ncalculées depuis la matrice de couverture, scénarios de véracité, tenants de\nréférence, relevés de canari. Aucun n'est saisi.",
			ruleTitle: "La règle",
			ruleBody: "**Aucun chiffre publié ici ne peut être meilleur que ce qui est mesuré.** Un\n" +
				"pourcentage sans mesure derrière est un faux vert déplacé dans un tableau de\nbord, et il y est pire qu'ailleurs : personne ne relit un tableau de bord.\n\n" +
				"Les chiffres sont donc laids, et c'est le point. « %d contrôles » ne dit rien de\nla qualité d'une détection ; « %d verdicts prouvés sur %d » dit où en est le\nproduit, et rétrécit dans le bon sens à chaque scénario écrit.",
			figures: "Les chiffres", colFigure: "Chiffre", colCount: "Nombre",
			controls:    "Contrôles au référentiel",
			paths:       "Chemins contrôle × fournisseur × source sur lesquels Pépin conclut",
			pathsProven: "Chemins dont TOUS les verdicts atteignables sont prouvés de bout en bout",
			obligations: "Verdicts à prouver au total", provenWord: "Verdicts prouvés",
			veracityTitle: "Couverture de véracité, par verdict",
			veracityIntro: "Un chemin doit prouver les verdicts qu'il peut réellement ATTEINDRE, pas quatre\n" +
				"partout : exiger un `not-applicable` d'un chemin où le mécanisme existe\ndemanderait d'inventer une non-applicabilité.",
			colVerdict: "Verdict", colMeaning: "Ce qu'il met en scène",
			colRequired: "À prouver", colProven: "Prouvés", colPct: "%",
			liveTitle: "Validé en live",
			liveBody: "Un scan canari interroge le VRAI plan de contrôle d'un fournisseur, mais **sans\n" +
				"identifiant** : il prouve qu'un endpoint existe et refuse, jamais qu'un droit\n*suffisant* rende `200` sur un tenant réel. Il ne vaut donc pas validation live\nd'un contrôle.\n\n" +
				"Ce compteur ne s'incrémente que sur un relevé **authentifié**, et il n'en existe\naucun. Le zéro est dérivé, pas écrit : le jour où un mainteneur consigne un\nrelevé authentifié, il montera tout seul.",
			liveRow:     "Chemins dont la source est une collecte live",
			canaryTitle: "Ce que les vrais plans de contrôle ont répondu",
			canaryIntro: "Une requête non authentifiée par endpoint déclaré, à la qualification de release.\n" +
				"Un endpoint qui répond existe et se résout ; un `moved` (404) dit qu'il a bougé.",
			colProvider: "Fournisseur", colRecorded: "Relevé le", colAnswered: "Ont répondu",
			colMoved: "Déplacés", colUn: "Injoignables",
			fpTitle: "Faux positifs",
			fpBody: "Le dépôt ne tient pas de registre de faux positifs, et en publier un compte serait\n" +
				"exactement la saisie que cette page refuse. Ce qui est MESURÉ, c'est le\ncontre-témoin : un tenant tiers déclaré durci sur lequel Pépin ne relève aucun\nécart `critical`/`high`. C'est le seul endroit où un faux positif se voit, et une\nporte le vérifie à chaque build.",
			counterwitnesses: "Tenants tiers durcis sans écart critical/high (contre-témoins)",
			tenantsWord:      "Tenants de référence au total",
			blindTitle:       "Mesures hors de portée",
			blindBody: "Elles sont documentées plutôt que comblées : cf. [Limites connues](known-limitations.fr.md)\n" +
				"et le registre de dette `internal/veracity/testdata/debt.txt`, qui nomme ligne à\nligne chaque verdict restant à prouver.",
			generated: "<!-- Page GÉNÉRÉE par internal/docgen. Ne pas éditer à la main. -->",
		}
	}
	return qualityText{
		title: "Detection quality map",
		intro: "What Pépin can PROVE about its own verdicts, and what it cannot.\n" +
			"Every figure on this page is derived from the repository's artefacts —\nobligations computed from the coverage matrix, veracity scenarios, reference\ntenants, canary records. None is typed in.",
		ruleTitle: "The rule",
		ruleBody: "**No figure published here can be better than what is measured.** A percentage\n" +
			"with no measurement behind it is a false green moved into a dashboard, and it is\nworse there than anywhere else: nobody re-reads a dashboard.\n\n" +
			"The figures are therefore ugly, and that is the point. \"%d controls\" says nothing\nabout the quality of a detection; \"%d verdicts proven out of %d\" says where the\nproduct stands, and shrinks the right way with every scenario written.",
		figures: "The figures", colFigure: "Figure", colCount: "Count",
		controls:    "Controls in the reference",
		paths:       "Control x provider x source paths on which Pépin concludes",
		pathsProven: "Paths whose EVERY reachable verdict is proven end to end",
		obligations: "Verdicts to prove in total", provenWord: "Verdicts proven",
		veracityTitle: "Veracity coverage, by verdict",
		veracityIntro: "A path must prove the verdicts it can actually REACH, not four everywhere:\n" +
			"demanding a `not-applicable` from a path where the mechanism exists would mean\ninventing a non-applicability.",
		colVerdict: "Verdict", colMeaning: "What it stages",
		colRequired: "To prove", colProven: "Proven", colPct: "%",
		liveTitle: "Validated live",
		liveBody: "A canary scan queries a provider's REAL control plane, but **with no credential**:\n" +
			"it proves an endpoint exists and refuses, never that a *sufficient* right returns\n`200` on a real tenant. It therefore does not count as live validation of a\ncontrol.\n\n" +
			"This counter only moves on an **authenticated** record, and none exists. The zero\nis derived, not written: the day a maintainer records an authenticated run, it\nwill rise on its own.",
		liveRow:     "Paths whose source is a live collection",
		canaryTitle: "What the real control planes answered",
		canaryIntro: "One unauthenticated request per declared endpoint, at release qualification.\n" +
			"An endpoint that answers exists and resolves; a `moved` (404) says it has shifted.",
		colProvider: "Provider", colRecorded: "Recorded", colAnswered: "Answered",
		colMoved: "Moved", colUn: "Unreachable",
		fpTitle: "False positives",
		fpBody: "The repository keeps no false-positive register, and publishing a count would be\n" +
			"exactly the data entry this page refuses. What is MEASURED is the\ncounter-witness: a third-party tenant declared hardened on which Pépin raises no\n`critical`/`high` deviation. It is the only place a false positive shows up, and\na gate checks it on every build.",
		counterwitnesses: "Hardened third-party tenants with no critical/high deviation (counter-witnesses)",
		tenantsWord:      "Reference tenants in total",
		blindTitle:       "Measurements out of reach",
		blindBody: "They are documented rather than papered over: see [Known limitations](known-limitations.md)\n" +
			"and the debt ledger `internal/veracity/testdata/debt.txt`, which names every\nverdict left to prove, line by line.",
		generated: "<!-- GENERATED page (internal/docgen). Do not edit by hand. -->",
	}
}

// qualityPage rend la page entière depuis l'instantané. Aucune prose n'y décrit un
// chiffre : chaque nombre est lu dans le Snapshot, donc la page ne peut pas mentir
// sans que l'instantané mente d'abord — et lui est gardé par la porte de fraîcheur.
func qualityPage(lang string, s quality.Snapshot) string {
	t := qualityStrings(lang)
	var b strings.Builder
	label := "🇬🇧 English · [🇫🇷 Français](detection-quality.fr.md)"
	if lang == "fr" {
		label = "[🇬🇧 English](detection-quality.md) · 🇫🇷 Français"
	}
	b.WriteString("> " + label + "\n\n")
	b.WriteString(t.generated + "\n\n")
	b.WriteString("# " + t.title + "\n\n" + t.intro + "\n\n")
	b.WriteString("## " + t.ruleTitle + "\n\n")
	b.WriteString(fmt.Sprintf(t.ruleBody, s.Controls, s.Proven, s.Obligations) + "\n\n")

	b.WriteString("## " + t.figures + "\n\n")
	b.WriteString("| " + t.colFigure + " | " + t.colCount + " |\n|---|---:|\n")
	row := func(label string, n int) { _, _ = fmt.Fprintf(&b, "| %s | %d |\n", label, n) }
	row(t.controls, s.Controls)
	row(t.paths, s.Paths)
	row(t.pathsProven, s.PathsProven)
	row(t.obligations, s.Obligations)
	row(t.provenWord, s.Proven)
	b.WriteString("\n")

	b.WriteString("## " + t.veracityTitle + "\n\n" + t.veracityIntro + "\n\n")
	b.WriteString("| " + t.colVerdict + " | " + t.colMeaning + " | " + t.colRequired +
		" | " + t.colProven + " | " + t.colPct + " |\n|---|---|---:|---:|---:|\n")
	for _, v := range verdictRows(lang) {
		c := s.Verdicts[v.key]
		_, _ = fmt.Fprintf(&b, "| %s | %s | %d | %d | %d |\n",
			v.name, v.staged, c.Required, c.Proven, quality.Percent(c.Proven, c.Required))
	}
	_, _ = fmt.Fprintf(&b, "| **%s** | | **%d** | **%d** | **%d** |\n\n",
		blockText(lang).totalWord, s.Obligations, s.Proven, quality.Percent(s.Proven, s.Obligations))

	b.WriteString("## " + t.liveTitle + "\n\n" + t.liveBody + "\n\n")
	b.WriteString("| " + t.colFigure + " | " + t.colCount + " |\n|---|---:|\n")
	row(t.liveRow, s.LivePaths)
	_, _ = fmt.Fprintf(&b, "| %s | **%d %%** |\n\n", t.liveTitle, quality.Percent(s.LiveValidated, s.LivePaths))

	b.WriteString("## " + t.canaryTitle + "\n\n" + t.canaryIntro + "\n\n")
	b.WriteString("| " + t.colProvider + " | " + t.colRecorded + " | " + t.colAnswered +
		" | " + t.colMoved + " | " + t.colUn + " |\n|---|---|---:|---:|---:|\n")
	for _, c := range s.Canary {
		_, _ = fmt.Fprintf(&b, "| `%s` | %s | %d | %d | %d |\n",
			c.Provider, c.Recorded, c.Answered, c.Moved, c.Unreachable)
	}
	b.WriteString("\n")

	b.WriteString("## " + t.fpTitle + "\n\n" + t.fpBody + "\n\n")
	b.WriteString("| " + t.colFigure + " | " + t.colCount + " |\n|---|---:|\n")
	row(t.counterwitnesses, s.Counterwitnesses)
	row(t.tenantsWord, s.Tenants)
	b.WriteString("\n")

	b.WriteString("## " + t.blindTitle + "\n\n" + t.blindBody + "\n")
	return b.String()
}

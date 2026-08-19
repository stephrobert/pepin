package docgen

// Le catalogue des contrôles, DÉRIVÉ du référentiel.
//
// Une page par contrôle et un index, calculés depuis `referentiel/controles.yaml` (code,
// famille, sévérité, prose bilingue, mappings normatifs, fournisseurs déclarés), les
// descripteurs `providers/*.yaml` (ce que chaque source sait produire), le verrou du « pass »
// de `internal/assess` et les preuves de remédiation présentes sous `references/remediation/`.
//
// Rien n'y est recopié à la main : ajouter un contrôle, changer une sévérité, retirer un
// fournisseur ou déposer une preuve de remédiation change ces pages, et
// TestGeneratedDocsAreUpToDate refuse une documentation qui ne suivrait pas.

import (
	"fmt"
	"sort"
	"strings"

	yaml "go.yaml.in/yaml/v3"

	"github.com/stephrobert/pepin/internal/i18n"
	"github.com/stephrobert/pepin/referentiel"
)

// controlsDir est le dossier des pages de contrôles, relatif à la racine du dépôt. Il est
// ENTIÈREMENT généré : Write y supprime ce que le générateur ne produit plus, sans quoi le
// retrait d'un contrôle laisserait une page orpheline qu'aucun test ne verrait.
const controlsDir = "docs/controls"

// familyOrder rend les familles du référentiel DANS LEUR ORDRE DE DÉCLARATION. L'ordre porte
// une intention éditoriale (IAM d'abord, gouvernance en dernier) qu'un tri alphabétique
// détruirait. La liste est lue dans le référentiel embarqué : la page ne peut pas connaître
// une famille que le référentiel ignore.
func familyOrder() ([]string, error) {
	var doc struct {
		Familles []struct {
			Code string `yaml:"code"`
		} `yaml:"familles"`
	}
	if err := yaml.Unmarshal(referentiel.Raw(), &doc); err != nil {
		return nil, fmt.Errorf("lecture des familles du référentiel : %w", err)
	}
	if len(doc.Familles) == 0 {
		return nil, fmt.Errorf("aucune famille lue dans le référentiel : le catalogue serait vide")
	}
	out := make([]string, 0, len(doc.Familles))
	for _, f := range doc.Familles {
		out = append(out, f.Code)
	}
	return out, nil
}

// controlPages rend l'index et la page de chaque contrôle, pour une langue.
func controlPages(lang string, m Matrix, proofs map[string]map[string]string, families []string) map[string]string {
	t := controlText(lang)
	out := map[string]string{controlsDir + "/" + pageName("index", lang): m.controlIndex(t, lang, proofs, families)}
	for _, r := range m.Rows {
		out[controlsDir+"/"+pageName(r.Code, lang)] = m.controlPage(t, lang, r, proofs)
	}
	return out
}

// pageName rend le nom de fichier d'une page dans une langue : l'anglais est primaire, le
// français porte le suffixe `.fr`.
func pageName(base, lang string) string {
	if lang == "fr" {
		return base + ".fr.md"
	}
	return base + ".md"
}

// switcher rend le sélecteur de langue en tête de page, vers la contrepartie de MÊME nom.
func switcher(base, lang string) string {
	if lang == "fr" {
		return "> [🇬🇧 English](" + base + ".md) · 🇫🇷 Français\n\n"
	}
	return "> 🇬🇧 English · [🇫🇷 Français](" + base + ".fr.md)\n\n"
}

// providersOf rend tous les fournisseurs de la matrice, cloud puis autre portée.
func (m Matrix) providersOf() []string {
	return append(append([]string{}, m.CloudProviders...), m.OtherProviders...)
}

// controlIndex rend docs/controls/index.md : la liste complète, par famille, avec ce qui
// permet de trier au premier coup d'œil — sévérité, exigence SCSL gelée, fournisseurs actifs,
// et présence d'une preuve de remédiation déployable.
func (m Matrix) controlIndex(t controlStrings, lang string, proofs map[string]map[string]string, families []string) string {
	var b strings.Builder
	b.WriteString(switcher("index", lang))
	b.WriteString(generatedBanner(lang, "mise run gen-docs"))
	b.WriteString("\n# " + t.indexTitle + "\n\n")
	b.WriteString(t.indexIntro + "\n\n")

	b.WriteString("## " + t.figuresTitle + "\n\n")
	b.WriteString(m.indexFigures(t, proofs))

	b.WriteString("\n## " + t.readingTitle + "\n\n")
	b.WriteString(t.readingBody + "\n")

	byFamily := map[string][]Row{}
	for _, r := range m.Rows {
		byFamily[r.Family] = append(byFamily[r.Family], r)
	}
	// Une famille présente au référentiel mais absente de l'en-tête `familles:` ne doit pas
	// disparaître silencieusement du catalogue : elle est rendue après les autres.
	seen := map[string]bool{}
	order := append([]string{}, families...)
	for _, f := range order {
		seen[f] = true
	}
	extra := make([]string, 0)
	for f := range byFamily {
		if !seen[f] {
			extra = append(extra, f)
		}
	}
	sort.Strings(extra)
	order = append(order, extra...)

	for _, fam := range order {
		rows := byFamily[fam]
		if len(rows) == 0 {
			continue
		}
		_, _ = fmt.Fprintf(&b, "\n## `%s`\n\n", fam)
		b.WriteString("| " + t.colControl + " | " + t.colSeverity + " | SCSL | " +
			t.colActiveFor + " | " + t.colProofs + " |\n|---|---|---|---|---|\n")
		for _, r := range rows {
			active := m.declaredProviders(r)
			state := strings.Join(codeList(active), ", ")
			if len(active) == 0 {
				state = "_" + t.dormant + "_"
			}
			_, _ = fmt.Fprintf(&b, "| [`%s`](%s) %s | %s | %s | %s | %s |\n",
				r.Code, pageName(r.Code, lang), r.Title, r.Severity,
				strings.Join(codeList(r.SCSL), ", "), state, proofCount(t, proofs, active, r.Code))
		}
	}

	b.WriteString("\n## " + t.dormantTitle + "\n\n")
	b.WriteString(t.dormantIntro + "\n\n")
	b.WriteString(m.dormantTable(t, lang))
	return b.String()
}

// indexFigures donne les chiffres du catalogue : contrôles, contrôles actifs, dormants,
// répartition par sévérité, preuves de remédiation. Ce sont les chiffres qu'on cite, donc
// exactement ceux qu'on ne veut pas maintenir à la main.
func (m Matrix) indexFigures(t controlStrings, proofs map[string]map[string]string) string {
	bySeverity := map[string]int{}
	active, dormant := 0, 0
	pairs, proven := 0, 0
	for _, r := range m.Rows {
		bySeverity[r.Severity]++
		declared := m.declaredProviders(r)
		if len(declared) == 0 {
			dormant++
		} else {
			active++
		}
		for _, p := range declared {
			pairs++
			if proofs[p][r.Code] != "" {
				proven++
			}
		}
	}
	var b strings.Builder
	b.WriteString("| " + t.colFigure + " | " + t.colCount + " |\n|---|---:|\n")
	_, _ = fmt.Fprintf(&b, "| %s | %d |\n", t.figTotal, len(m.Rows))
	_, _ = fmt.Fprintf(&b, "| %s | %d |\n", t.figActive, active)
	_, _ = fmt.Fprintf(&b, "| %s | %d |\n", t.figDormant, dormant)
	for _, s := range []string{"critical", "high", "medium", "low"} {
		_, _ = fmt.Fprintf(&b, "| `%s` | %d |\n", s, bySeverity[s])
	}
	_, _ = fmt.Fprintf(&b, "| %s | %d / %d |\n", t.figProofs, proven, pairs)
	return b.String()
}

// dormantTable liste les contrôles écrits mais déclarés pour AUCUN fournisseur. Les laisser
// se fondre dans la masse ferait lire le catalogue comme une couverture.
func (m Matrix) dormantTable(t controlStrings, lang string) string {
	var b strings.Builder
	b.WriteString("| " + t.colControl + " | " + t.colSeverity + " | " + t.colFamily + " |\n|---|---|---|\n")
	rows := 0
	for _, r := range m.Rows {
		if len(m.declaredProviders(r)) > 0 {
			continue
		}
		_, _ = fmt.Fprintf(&b, "| [`%s`](%s) %s | %s | `%s` |\n",
			r.Code, pageName(r.Code, lang), r.Title, r.Severity, r.Family)
		rows++
	}
	if rows == 0 {
		return t.noDormant + "\n"
	}
	return b.String()
}

// controlPage rend docs/controls/<code>.md : tout ce que le dépôt sait d'un contrôle, sans
// qu'un lecteur ait à ouvrir le référentiel, les descripteurs et le code du verrou.
func (m Matrix) controlPage(t controlStrings, lang string, r Row, proofs map[string]map[string]string) string {
	ctl, ok := referentiel.Lookup(r.Code)
	var b strings.Builder
	b.WriteString(switcher(r.Code, lang))
	b.WriteString(generatedBanner(lang, "mise run gen-docs"))
	b.WriteString("\n# `" + r.Code + "`\n\n")
	b.WriteString("**" + r.Title + "**\n\n")
	b.WriteString("[" + t.backToIndex + "](" + pageName("index", lang) + ")\n\n")

	b.WriteString(m.identityTable(t, r, proofs))

	b.WriteString("\n## " + t.whyTitle + "\n\n")
	if ok {
		b.WriteString(paragraph(ctl.DescriptionIn(i18n.Lang(lang))) + "\n")
	}
	b.WriteString("\n" + t.whyNote + "\n")

	b.WriteString("\n## " + t.mappingTitle + "\n\n")
	b.WriteString(t.mappingIntro + "\n\n")
	b.WriteString(mappingTable(t, r, ctl))

	b.WriteString("\n## " + t.whereTitle + "\n\n")
	b.WriteString(t.whereIntro + "\n\n")
	b.WriteString(m.whereTable(t, r))
	b.WriteString("\n" + t.reasonsIntro + "\n\n")
	b.WriteString(m.whyNotTable(t, r))

	b.WriteString("\n## " + t.concludeTitle + "\n\n")
	b.WriteString(m.concludeTable(t, r))

	b.WriteString("\n## " + t.investigateTitle + "\n\n")
	b.WriteString(m.investigateBody(t, r))

	b.WriteString("\n## " + t.remediateTitle + "\n\n")
	if ok {
		b.WriteString(paragraph(ctl.RemediationIn(i18n.Lang(lang))) + "\n\n")
	}
	b.WriteString(m.proofTable(t, r, proofs))
	b.WriteString("\n" + t.proofNote + "\n")

	b.WriteString("\n## " + t.verifyTitle + "\n\n")
	b.WriteString(m.verifyBody(t, r))

	b.WriteString("\n## " + t.seeAlsoTitle + "\n\n")
	b.WriteString(t.seeAlso + "\n")
	return b.String()
}

// identityTable rend la carte d'identité du contrôle : ce qu'il est, ce qu'il lit, et ce dont
// sa décision dépend.
func (m Matrix) identityTable(t controlStrings, r Row, proofs map[string]map[string]string) string {
	declared := m.declaredProviders(r)
	typ := "_" + t.noType + "_"
	if r.Type != "" {
		typ = "`" + r.Type + "`"
	}
	attrs := "_" + t.noAttr + "_"
	if len(r.RequiredAttrs) > 0 {
		attrs = strings.Join(codeList(r.RequiredAttrs), " / ")
	}
	state := t.stateActive
	if len(declared) == 0 {
		state = t.stateDormant
	}
	proven := 0
	for _, p := range declared {
		if proofs[p][r.Code] != "" {
			proven++
		}
	}
	var b strings.Builder
	b.WriteString("| " + t.colField + " | " + t.colValue + " |\n|---|---|\n")
	_, _ = fmt.Fprintf(&b, "| %s | `%s` |\n", t.rowCode, r.Code)
	_, _ = fmt.Fprintf(&b, "| %s | `%s` |\n", t.rowFamily, r.Family)
	_, _ = fmt.Fprintf(&b, "| %s | `%s` |\n", t.rowSeverity, r.Severity)
	_, _ = fmt.Fprintf(&b, "| %s | %s |\n", t.rowSCSL, orNone(t, strings.Join(codeList(r.SCSL), ", ")))
	_, _ = fmt.Fprintf(&b, "| %s | %s |\n", t.rowType, typ)
	_, _ = fmt.Fprintf(&b, "| %s | %s |\n", t.rowAttrs, attrs)
	_, _ = fmt.Fprintf(&b, "| %s | %s |\n", t.rowState, state)
	_, _ = fmt.Fprintf(&b, "| %s | %s |\n", t.rowDeclared, orNone(t, strings.Join(codeList(declared), ", ")))
	_, _ = fmt.Fprintf(&b, "| %s | %d / %d |\n", t.rowProofs, proven, len(declared))
	return b.String()
}

// mappingTable rend les correspondances normatives EXACTES du référentiel : l'exigence SCSL
// gelée d'abord, puis les frameworks. Aucune n'est reformulée ici.
func mappingTable(t controlStrings, r Row, ctl referentiel.Control) string {
	var b strings.Builder
	b.WriteString("| " + t.colFramework + " | " + t.colRefs + " |\n|---|---|\n")
	if len(r.SCSL) > 0 {
		_, _ = fmt.Fprintf(&b, "| `scsl` | %s |\n", strings.Join(codeList(r.SCSL), ", "))
	}
	names := make([]string, 0, len(ctl.Frameworks))
	for fw := range ctl.Frameworks {
		names = append(names, fw)
	}
	sort.Strings(names)
	for _, fw := range names {
		ids := ctl.Frameworks[fw]
		if len(ids) == 0 {
			continue
		}
		_, _ = fmt.Fprintf(&b, "| `%s` | %s |\n", fw, strings.Join(codeList(ids), ", "))
	}
	return b.String()
}

// whereTable rend la couverture du contrôle, fournisseur par fournisseur et source par source.
func (m Matrix) whereTable(t controlStrings, r Row) string {
	var b strings.Builder
	b.WriteString("| " + t.colProvider + " | " + t.colTerraform + " | " + t.colLive + " |\n|---|:-:|:-:|\n")
	for _, p := range m.CloudProviders {
		_, _ = fmt.Fprintf(&b, "| %s | %s | %s |\n", p,
			mark(r.Cells[p][SourceTerraform].Status), mark(r.Cells[p][SourceLive].Status))
	}
	for _, p := range m.OtherProviders {
		_, _ = fmt.Fprintf(&b, "| %s | %s | %s |\n", p, t.noSource, mark(r.Cells[p][SourceLive].Status))
	}
	return b.String()
}

// whyNotTable détaille chaque case qui n'est pas ✅ alors que le contrôle est déclaré pour ce
// fournisseur. Même règle que la matrice de couverture : un statut sans motif n'est pas
// opposable, et un « non déclaré » n'apprend rien de plus que le tableau ci-dessus.
func (m Matrix) whyNotTable(t controlStrings, r Row) string {
	var b strings.Builder
	b.WriteString("| " + t.colProvider + " | " + t.colSource + " | " + t.colStatus + " | " + t.colReason + " |\n|---|---|---|---|\n")
	rows := 0
	for _, p := range m.providersOf() {
		for _, src := range []Source{SourceTerraform, SourceLive} {
			c := r.Cells[p][src]
			if c.Status == Supported || c.Undeclared {
				continue
			}
			_, _ = fmt.Fprintf(&b, "| %s | %s | %s `%s` | %s |\n", p, src, mark(c.Status), c.Status, oneLine(c.Reason))
			rows++
		}
	}
	if rows == 0 {
		return t.noReason + "\n"
	}
	return b.String()
}

// concludeTable dit, statut par statut, OÙ ce contrôle peut réellement l'atteindre. C'est la
// question qu'un auditeur pose : « ce vert, il vaut pour quelle source ? ».
func (m Matrix) concludeTable(t controlStrings, r Row) string {
	var pass, fail, ne, na []string
	for _, p := range m.providersOf() {
		for _, src := range []Source{SourceTerraform, SourceLive} {
			c := r.Cells[p][src]
			label := p + " / " + string(src)
			switch c.Status {
			case Supported:
				pass = append(pass, label)
				fail = append(fail, label)
			case Partial:
				fail = append(fail, label)
				ne = append(ne, label)
			case NotApplicable:
				na = append(na, label)
			case Unsupported:
			}
		}
	}
	var b strings.Builder
	b.WriteString("| " + t.colStatus + " | " + t.colMeans + " | " + t.colWhere + " |\n|---|---|---|\n")
	_, _ = fmt.Fprintf(&b, "| `fail` | %s | %s |\n", t.meansFail, orNone(t, strings.Join(fail, " · ")))
	_, _ = fmt.Fprintf(&b, "| `pass` | %s | %s |\n", t.meansCompliant, orNone(t, strings.Join(pass, " · ")))
	_, _ = fmt.Fprintf(&b, "| `not-applicable` | %s | %s |\n", t.meansNA, orNone(t, strings.Join(na, " · ")))
	_, _ = fmt.Fprintf(&b, "| `not-evaluated` | %s | %s |\n", t.meansNE, orNone(t, strings.Join(ne, " · ")))
	b.WriteString("\n" + t.concludeNote + "\n")
	return b.String()
}

// investigateBody dit quoi regarder côté fournisseur : le type normalisé lu, les attributs
// dont la décision dépend, et les descripteurs où leur projection se lit.
func (m Matrix) investigateBody(t controlStrings, r Row) string {
	var b strings.Builder
	if r.Type == "" {
		b.WriteString("- " + t.investNoType + "\n")
	} else {
		b.WriteString("- " + t.investType + " `" + r.Type + "`\n")
	}
	if len(r.RequiredAttrs) > 0 {
		b.WriteString("- " + t.investAttrs + " " + strings.Join(codeList(r.RequiredAttrs), " / ") + "\n")
		b.WriteString("- " + t.investGuard + "\n")
	} else {
		b.WriteString("- " + t.investNoAttrs + "\n")
	}
	declared := m.declaredProviders(r)
	if len(declared) > 0 {
		b.WriteString("- " + t.investDescriptors + " ")
		links := make([]string, 0, len(declared))
		for _, p := range declared {
			links = append(links, "[`providers/"+p+".yaml`](../../providers/"+p+".yaml)")
		}
		b.WriteString(strings.Join(links, " · ") + "\n")
	}
	b.WriteString("- " + t.investRule + "\n")
	return b.String()
}

// proofTable rend, par fournisseur déclaré, la preuve de remédiation DÉPLOYABLE présente au
// dépôt — ou son absence. L'absence est dite, elle n'est pas masquée : c'est le chantier que
// `mise run check-remediation` suit.
func (m Matrix) proofTable(t controlStrings, r Row, proofs map[string]map[string]string) string {
	declared := m.declaredProviders(r)
	if len(declared) == 0 {
		return t.proofDormant + "\n"
	}
	var b strings.Builder
	b.WriteString("| " + t.colProvider + " | " + t.colProof + " |\n|---|---|\n")
	for _, p := range declared {
		path := proofs[p][r.Code]
		if path == "" {
			_, _ = fmt.Fprintf(&b, "| %s | _%s_ |\n", p, t.proofMissing)
			continue
		}
		_, _ = fmt.Fprintf(&b, "| %s | [`%s`](../../%s) |\n", p, path, path)
	}
	return b.String()
}

// verifyBody rend la commande qui confirme la correction. Le fournisseur cité n'est pas
// choisi au hasard : c'est un de ceux dont CETTE source sait conclure, sinon la commande
// montrée rendrait `not-evaluated` sur un tenant parfaitement corrigé. Quand aucune source ne
// sait lever le verrou du « pass », la page le dit au lieu de proposer une vérification qui
// ne vérifie rien.
func (m Matrix) verifyBody(t controlStrings, r Row) string {
	tf, tfExact := m.witness(r, SourceTerraform)
	live, liveExact := m.witness(r, SourceLive)
	if tf == "" && live == "" {
		return t.verifyNone
	}
	var cmds []string
	if tf != "" {
		cmds = append(cmds, "# "+t.verifyTF, "./pepin scan "+tf+" --terraform plan.json --format assessment")
	}
	if live != "" {
		if len(cmds) > 0 {
			cmds = append(cmds, "")
		}
		cmds = append(cmds, "# "+t.verifyLive, "./pepin scan "+live+" --live --format assessment")
	}
	var b strings.Builder
	b.WriteString(Fence("bash", strings.Join(cmds, "\n")) + "\n\n")
	b.WriteString(fmt.Sprintf(t.verifyBodyFmt, r.Code) + "\n")
	if !tfExact || !liveExact {
		b.WriteString("\n" + t.verifyPartial + "\n")
	}
	return b.String()
}

// witness rend un fournisseur témoin pour une source : de préférence un qui sait conclure
// (`supported`), à défaut un qui produit le type sans pouvoir lever le verrou (`partial`). Le
// second retour dit si le témoin est du premier genre.
func (m Matrix) witness(r Row, src Source) (string, bool) {
	fallback := ""
	for _, p := range m.declaredProviders(r) {
		switch r.Cells[p][src].Status {
		case Supported:
			return p, true
		case Partial:
			if fallback == "" {
				fallback = p
			}
		case NotApplicable, Unsupported:
		}
	}
	return fallback, fallback == ""
}

// declaredProviders rend les fournisseurs pour lesquels le contrôle est DÉCLARÉ au référentiel
// (`fournisseurs:`), dans l'ordre de la matrice. La liste vient du référentiel et non des cases :
// une non-applicabilité justifiée est rendue AVANT la porte de déclaration, donc lire les cases
// comptait comme déclarés des couples que le référentiel ignore.
func (m Matrix) declaredProviders(r Row) []string {
	var out []string
	for _, p := range m.providersOf() {
		if contains(r.Declared, p) {
			out = append(out, p)
		}
	}
	return out
}

// proofCount rend « n / m » : les preuves de remédiation présentes sur les couples déclarés.
func proofCount(t controlStrings, proofs map[string]map[string]string, declared []string, code string) string {
	if len(declared) == 0 {
		return t.none
	}
	n := 0
	for _, p := range declared {
		if proofs[p][code] != "" {
			n++
		}
	}
	return fmt.Sprintf("%d / %d", n, len(declared))
}

// codeList met chaque valeur en code Markdown, pour qu'un identifiant ne se lise pas comme de
// la prose.
func codeList(vals []string) []string {
	out := make([]string, 0, len(vals))
	for _, v := range vals {
		out = append(out, "`"+v+"`")
	}
	return out
}

// orNone rend un marqueur explicite plutôt qu'une cellule vide : une cellule vide se lit
// comme un oubli de rendu, pas comme une absence constatée.
func orNone(t controlStrings, s string) string {
	if strings.TrimSpace(s) == "" {
		return t.none
	}
	return s
}

// paragraph aplatit la prose du référentiel (pliée par le YAML) en un paragraphe.
func paragraph(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

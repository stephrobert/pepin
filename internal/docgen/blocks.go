package docgen

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/stephrobert/pepin/internal/assess"
	"github.com/stephrobert/pepin/internal/genprovider"
	"github.com/stephrobert/pepin/referentiel"
)

// Les deux inventaires minimaux que la documentation fait écrire au lecteur pour observer les
// codes de sortie 3. Ils sont créés dans un dossier jetable au moment de la capture, et rendus
// tels quels dans la page : ce que le lecteur colle est ce que le générateur a exécuté.
const (
	emptyInventory = `{
  "provider": "scaleway",
  "resources": []
}`
	taglessInventory = `{
  "provider": "scaleway",
  "resources": [
    {
      "provider": "scaleway",
      "type": "compute_instance",
      "id": "srv-demo",
      "name": "srv-demo",
      "region": "fr-par",
      "attributes": {
        "vm_id": "srv-demo",
        "security_group_ids": ["sg-front"],
        "tags": []
      }
    }
  ]
}`
)

// Chemins des fixtures du dépôt utilisées par les captures.
const (
	planVulnerable = "examples/scaleway/terraform/plan.json"
	planFixed      = "examples/scaleway/terraform-fixed/plan.json"
	planMissing    = "examples/scaleway/plan-absent.json"
)

// captures rassemble toutes les exécutions réelles de `pepin` dont la documentation montre la
// sortie. Aucune autre source n'est admise dans une page.
type captures struct {
	vulnerable  Capture
	fixed       Capture
	assessment  Capture // --format assessment sur le plan non conforme
	missingFile Capture
	empty       Capture
	tagless     Capture
	taglessStr  Capture
	providers   Capture
}

// captureAll exécute la totalité des commandes documentées. Les deux inventaires en ligne sont
// écrits dans un dossier jetable ; la commande AFFICHÉE, elle, porte le nom de fichier relatif
// que le lecteur aura créé.
func captureAll(root, bin string) (captures, error) {
	r := Runner{Bin: bin, Root: root}
	tmp, err := os.MkdirTemp("", "pepin-docgen-")
	if err != nil {
		return captures{}, err
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	emptyPath := filepath.Join(tmp, "empty-inventory.json")
	taglessPath := filepath.Join(tmp, "tagless-inventory.json")
	if werr := os.WriteFile(emptyPath, []byte(emptyInventory+"\n"), 0o600); werr != nil {
		return captures{}, werr
	}
	if werr := os.WriteFile(taglessPath, []byte(taglessInventory+"\n"), 0o600); werr != nil {
		return captures{}, werr
	}

	var c captures
	steps := []struct {
		into    *Capture
		args    []string
		display []string
	}{
		{&c.vulnerable, []string{"scan", "scaleway", "--terraform", planVulnerable}, nil},
		{&c.fixed, []string{"scan", "scaleway", "--terraform", planFixed}, nil},
		{&c.assessment, []string{"scan", "scaleway", "--terraform", planVulnerable, "--format", "assessment"}, nil},
		{&c.missingFile, []string{"scan", "scaleway", planMissing}, nil},
		{&c.empty, []string{"scan", "scaleway", emptyPath}, []string{"scan", "scaleway", "empty-inventory.json"}},
		{&c.tagless, []string{"scan", "scaleway", taglessPath}, []string{"scan", "scaleway", "tagless-inventory.json"}},
		{&c.taglessStr, []string{"scan", "scaleway", taglessPath, "--strict"}, []string{"scan", "scaleway", "tagless-inventory.json", "--strict"}},
		{&c.providers, []string{"provider", "list"}, nil},
	}
	for _, s := range steps {
		got, rerr := r.Run(s.args...)
		if rerr != nil {
			return captures{}, rerr
		}
		if s.display != nil {
			got.Args = s.display
		}
		*s.into = got
	}
	if strings.TrimSpace(c.vulnerable.Stdout) == "" {
		return captures{}, fmt.Errorf("le scan de %s n'a rien produit : capture inexploitable", planVulnerable)
	}
	// La VERSION est injectée au build (`git describe`) : elle diffère d'une machine et d'un
	// commit à l'autre, et le bandeau la porte. Sans neutralisation, la page divergerait à
	// chaque build sans qu'aucun comportement n'ait bougé — un contrôle qui crie pour rien
	// finit désarmé. On la remplace partout, et la page le dit.
	ver, verr := r.Run("version")
	if verr != nil {
		return captures{}, verr
	}
	if v := strings.TrimPrefix(strings.TrimSpace(ver.Stdout), "pépin "); len(v) >= 3 {
		for _, capt := range []*Capture{&c.vulnerable, &c.fixed, &c.assessment, &c.missingFile,
			&c.empty, &c.tagless, &c.taglessStr, &c.providers} {
			capt.Stdout = strings.ReplaceAll(capt.Stdout, v, versionPlaceholder)
			capt.Stderr = strings.ReplaceAll(capt.Stderr, v, versionPlaceholder)
		}
	}
	return c, nil
}

// versionPlaceholder remplace la version du binaire dans les sorties capturées.
const versionPlaceholder = "<version>"

// buildBlocks assemble toutes les régions injectables pour une langue.
func buildBlocks(lang string, m Matrix, c captures, rem []RemediationCoverage) map[string]string {
	t := blockText(lang)
	b := map[string]string{
		"scope-disclaimer":          Fence("text", wrapDisclaimer(assess.ScopeDisclaimer)),
		"scan-vulnerable-head":      Fence("text", Head(c.vulnerable.Stdout, 20)),
		"scan-vulnerable-tail":      Fence("text", Tail(c.vulnerable.Stdout, 22)),
		"scan-vulnerable-full":      Fence("text", c.vulnerable.Stdout),
		"scan-vulnerable-banner":    Fence("text", c.vulnerable.Stderr),
		"scan-fixed-full":           Fence("text", c.fixed.Stdout),
		"provider-list":             Fence("text", c.providers.Stdout),
		"fixture-empty-inventory":   Fence("json", emptyInventory),
		"fixture-tagless-inventory": Fence("json", taglessInventory),
		"exit-codes":                exitCodeTable(t, c),
		"assessment-run":            assessmentRunBlock(c),
		"assessment-counts":         assessmentCountsTable(t, c),
		"assessment-fail":           assessmentExtract(c, "fail", "objectstorage_bucket_public_access"),
		"assessment-pass":           assessmentExtract(c, "pass", "network_securitygroup_allow_ingress_from_internet_to_all_ports"),
		"assessment-na":             assessmentExtract(c, "not-applicable", "blockstorage_volume_encryption"),
		"assessment-ne":             assessmentExtract(c, "not-evaluated", "compute_instance_public_ip_with_open_securitygroup"),
		"not-evaluated-reasons":     notEvaluatedReasons(t, c),
		"required-attrs":            requiredAttrTable(t),
		"never-pass":                neverPassTable(t, m),
		"single-source":             singleSourceTable(t, m),
		"not-applicable-list":       notApplicableTable(t, m),
		"remediation-coverage":      remediationTable(t, rem),
		"coverage-totals":           m.countsTable(coverageText(lang)),
		"control-counts":            controlCountsTable(t, m),
	}
	return b
}

// wrapDisclaimer replie l'avertissement de portée à une largeur lisible, sans changer un mot :
// c'est la MÊME chaîne que celle imprimée à chaque scan (assess.ScopeDisclaimer).
func wrapDisclaimer(s string) string {
	const width = 84
	var out []string
	line := ""
	for _, w := range strings.Fields(s) {
		if line != "" && len(line)+1+len(w) > width {
			out = append(out, line)
			line = w
			continue
		}
		if line == "" {
			line = w
			continue
		}
		line += " " + w
	}
	if line != "" {
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func exitCodeTable(t blockStrings, c captures) string {
	rows := []struct {
		situation string
		cap       Capture
	}{
		{t.exitCompliant, c.fixed},
		{t.exitNonCompliance, c.vulnerable},
		{t.exitError, c.missingFile},
		{t.exitNothing, c.empty},
		{t.exitMediumPlain, c.tagless},
		{t.exitMediumStrict, c.taglessStr},
	}
	var b strings.Builder
	b.WriteString("| " + t.colSituation + " | " + t.colCommand + " | " + t.colExit + " |\n|---|---|:-:|\n")
	for _, r := range rows {
		_, _ = fmt.Fprintf(&b, "| %s | `%s` | **%d** |\n", r.situation, r.cap.Command(), r.cap.Exit)
	}
	return b.String()
}

// assessmentRunBlock rend l'enveloppe de provenance d'un scan réel. Les champs VOLATILES
// (horodatage, version, empreintes) sont remplacés par un marqueur explicite : ils changent à
// chaque exécution, et une page qui les figerait mentirait dès le commit suivant.
func assessmentRunBlock(c captures) string {
	doc := parseAssessment(c)
	if doc == nil {
		return Fence("json", "{}")
	}
	run, _ := doc["run"].(map[string]any)
	normalizeRun(run)
	return Fence("json", mustIndent(map[string]any{"run": run}))
}

// normalizeRun neutralise ce qui varie d'une exécution à l'autre.
func normalizeRun(run map[string]any) {
	if run == nil {
		return
	}
	run["timestamp"] = "<horodatage RFC3339 du scan>"
	if tool, ok := run["tool"].(map[string]any); ok {
		tool["version"] = "<version>"
		tool["digest"] = "<empreinte du binaire>"
	}
	if rs, ok := run["ruleset"].(map[string]any); ok {
		rs["digest"] = "<empreinte règles + descripteurs + référentiel>"
	}
}

// assessmentExtract rend UN résultat portant le statut demandé, tel que le scan l'a produit :
// `preferred` d'abord (le contrôle que la prose de la page commente), sinon le premier par ordre
// de code. Aucun champ n'est ajouté ni retiré. Le repli garantit qu'un contrôle qui disparaîtrait
// du scan d'exemple ne casse pas la génération : elle montrerait un autre résultat du même
// statut, jamais un exemple fabriqué.
func assessmentExtract(c captures, status, preferred string) string {
	doc := parseAssessment(c)
	if doc == nil {
		return Fence("json", "{}")
	}
	results, _ := doc["results"].([]any)
	var picked []map[string]any
	for _, it := range results {
		r, ok := it.(map[string]any)
		if !ok {
			continue
		}
		if s, _ := r["status"].(string); s == status {
			picked = append(picked, r)
		}
	}
	if len(picked) == 0 {
		return Fence("text", "(aucun résultat « "+status+" » sur ce scan)")
	}
	sort.Slice(picked, func(i, j int) bool {
		a, _ := picked[i]["control"].(string)
		b, _ := picked[j]["control"].(string)
		return a < b
	})
	for _, r := range picked {
		if ctl, _ := r["control"].(string); ctl == preferred {
			return Fence("json", mustIndent(r))
		}
	}
	return Fence("json", mustIndent(picked[0]))
}

func assessmentCountsTable(t blockStrings, c captures) string {
	doc := parseAssessment(c)
	counts := map[string]int{}
	if doc != nil {
		results, _ := doc["results"].([]any)
		for _, it := range results {
			if r, ok := it.(map[string]any); ok {
				s, _ := r["status"].(string)
				counts[s]++
			}
		}
	}
	var b strings.Builder
	b.WriteString("| " + t.colStatus + " | " + t.colCount + " |\n|---|---:|\n")
	for _, s := range []string{"pass", "fail", "not-applicable", "not-evaluated"} {
		_, _ = fmt.Fprintf(&b, "| `%s` | %d |\n", s, counts[s])
	}
	return b.String()
}

// notEvaluatedReasons énumère les motifs DISTINCTS de non-évaluation observés sur le scan
// d'exemple, avec un contrôle témoin : c'est la preuve qu'un `not-evaluated` dit toujours sur
// quoi il bute.
func notEvaluatedReasons(t blockStrings, c captures) string {
	doc := parseAssessment(c)
	type entry struct {
		reason  string
		count   int
		witness string
	}
	byReason := map[string]*entry{}
	if doc != nil {
		results, _ := doc["results"].([]any)
		for _, it := range results {
			r, ok := it.(map[string]any)
			if !ok {
				continue
			}
			if s, _ := r["status"].(string); s != "not-evaluated" {
				continue
			}
			ev, _ := r["evidence"].(map[string]any)
			obs, _ := ev["observed"].(string)
			ctl, _ := r["control"].(string)
			key := generalizeReason(obs)
			if e, ok := byReason[key]; ok {
				e.count++
				if ctl < e.witness {
					e.witness = ctl
				}
				continue
			}
			byReason[key] = &entry{reason: key, count: 1, witness: ctl}
		}
	}
	keys := make([]string, 0, len(byReason))
	for k := range byReason {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString("| " + t.colReason + " | " + t.colCount + " | " + t.colWitness + " |\n|---|---:|---|\n")
	for _, k := range keys {
		e := byReason[k]
		_, _ = fmt.Fprintf(&b, "| %s | %d | `%s` |\n", oneLine(e.reason), e.count, e.witness)
	}
	return b.String()
}

// quotedName repère un nom (type, attribut, état) cité par le moteur entre guillemets français.
var quotedName = regexp.MustCompile(`«[^»]*»`)

// generalizeReason remplace CHAQUE nom cité d'un motif par un marqueur générique, pour
// regrouper les motifs de MÊME NATURE sans en inventer le libellé. Chaque span est traité
// séparément : englober du premier au dernier guillemet avalerait le texte intermédiaire, et
// la ligne du tableau ne dirait plus ce que le moteur dit.
func generalizeReason(s string) string {
	return quotedName.ReplaceAllString(s, "« … »")
}

// requiredAttrTable rend la table des attributs décisifs telle qu'elle est APPLIQUÉE
// (assess.RequiredAttrs), pas une paraphrase.
func requiredAttrTable(t blockStrings) string {
	req := assess.RequiredAttrs()
	codes := make([]string, 0, len(req))
	for c := range req {
		codes = append(codes, c)
	}
	sort.Strings(codes)
	var b strings.Builder
	b.WriteString("| " + t.colControl + " | " + t.colType + " | " + t.colDeciding + " |\n|---|---|---|\n")
	for _, c := range codes {
		attrs := req[c]
		sort.Strings(attrs)
		quoted := make([]string, 0, len(attrs))
		for _, a := range attrs {
			quoted = append(quoted, "`"+a+"`")
		}
		typ := genprovider.ControlType(c)
		if typ == "" {
			typ = t.noTypeWord // contrôle transverse : aucun type de ressource visé
		} else {
			typ = "`" + typ + "`"
		}
		_, _ = fmt.Fprintf(&b, "| `%s` | %s | %s |\n", c, typ, strings.Join(quoted, " "+t.orWord+" "))
	}
	return b.String()
}

// neverPassTable liste les contrôles qui ne peuvent conclure « pass » chez AUCUN fournisseur
// et depuis AUCUNE source. Ce sont des angles morts structurels, et les taire serait la faute
// que cette page existe pour éviter.
func neverPassTable(t blockStrings, m Matrix) string {
	var b strings.Builder
	b.WriteString("| " + t.colControl + " | " + t.colSeverity + " | " + t.colReason + " |\n|---|---|---|\n")
	rows := 0
	providers := append(append([]string{}, m.CloudProviders...), m.OtherProviders...)
	for _, r := range m.Rows {
		if !declaredSomewhere(r, providers) {
			continue
		}
		reason := ""
		can := false
		for _, p := range providers {
			for _, src := range []Source{SourceTerraform, SourceLive} {
				c := r.Cells[p][src]
				if c.Status == Supported {
					can = true
				}
				if c.Status == Partial && reason == "" {
					reason = c.Reason
				}
			}
		}
		if can || reason == "" {
			continue
		}
		_, _ = fmt.Fprintf(&b, "| `%s` | %s | %s |\n", r.Code, r.Severity, oneLine(reason))
		rows++
	}
	if rows == 0 {
		return t.none + "\n"
	}
	return b.String()
}

func declaredSomewhere(r Row, providers []string) bool {
	ctl, ok := referentiel.Lookup(r.Code)
	if !ok {
		return false
	}
	for _, p := range providers {
		if contains(ctl.Fournisseurs, p) {
			return true
		}
	}
	return false
}

// singleSourceTable liste les couples (contrôle, fournisseur) observables depuis UNE SEULE
// source. Le référentiel déclare le fournisseur ; la page dit par quel chemin, et par lequel
// il ne passe pas.
func singleSourceTable(t blockStrings, m Matrix) string {
	var b strings.Builder
	b.WriteString("| " + t.colControl + " | " + t.colProvider + " | " + t.colOnlyVia + " | " + t.colReason + " |\n|---|---|---|---|\n")
	rows := 0
	for _, r := range m.Rows {
		for _, p := range m.CloudProviders {
			tf := r.Cells[p][SourceTerraform]
			live := r.Cells[p][SourceLive]
			switch {
			case tf.Status == Supported && live.Status != Supported:
				_, _ = fmt.Fprintf(&b, "| `%s` | %s | terraform | %s |\n", r.Code, p, oneLine(live.Reason))
				rows++
			case live.Status == Supported && tf.Status != Supported:
				_, _ = fmt.Fprintf(&b, "| `%s` | %s | live | %s |\n", r.Code, p, oneLine(tf.Reason))
				rows++
			}
		}
	}
	if rows == 0 {
		return t.none + "\n"
	}
	return b.String()
}

// notApplicableTable rend chaque non-applicabilité DÉCLARÉE avec sa justification : c'est ce
// qu'un auditeur lit en premier, et un N/A sans motif n'est pas opposable.
func notApplicableTable(t blockStrings, m Matrix) string {
	var b strings.Builder
	b.WriteString("| " + t.colControl + " | " + t.colProvider + " | " + t.colJustification + " |\n|---|---|---|\n")
	rows := 0
	providers := append(append([]string{}, m.CloudProviders...), m.OtherProviders...)
	for _, r := range m.Rows {
		for _, p := range providers {
			c := r.Cells[p][SourceLive]
			if c.Status != NotApplicable {
				continue
			}
			_, _ = fmt.Fprintf(&b, "| `%s` | %s | %s |\n", r.Code, p, oneLine(c.Reason))
			rows++
		}
	}
	if rows == 0 {
		return t.none + "\n"
	}
	return b.String()
}

func remediationTable(t blockStrings, rem []RemediationCoverage) string {
	var b strings.Builder
	b.WriteString("| " + t.colProvider + " | " + t.colProofs + " |\n|---|---:|\n")
	covered, total := 0, 0
	for _, c := range rem {
		covered += c.Covered
		total += c.Total
		_, _ = fmt.Fprintf(&b, "| %s | %d / %d |\n", c.Provider, c.Covered, c.Total)
	}
	_, _ = fmt.Fprintf(&b, "| **%s** | **%d / %d** |\n", t.totalWord, covered, total)
	return b.String()
}

// controlCountsTable donne les chiffres du référentiel : contrôles catalogués, contrôles
// déclarés pour au moins un fournisseur, et répartition par sévérité.
func controlCountsTable(t blockStrings, m Matrix) string {
	bySeverity := map[string]int{}
	declared := 0
	providers := append(append([]string{}, m.CloudProviders...), m.OtherProviders...)
	for _, r := range m.Rows {
		bySeverity[r.Severity]++
		if declaredSomewhere(r, providers) {
			declared++
		}
	}
	var b strings.Builder
	b.WriteString("| " + t.colFigure + " | " + t.colCount + " |\n|---|---:|\n")
	_, _ = fmt.Fprintf(&b, "| %s | %d |\n", t.figControls, len(m.Rows))
	_, _ = fmt.Fprintf(&b, "| %s | %d |\n", t.figDeclared, declared)
	for _, s := range []string{"critical", "high", "medium", "low"} {
		_, _ = fmt.Fprintf(&b, "| `%s` | %d |\n", s, bySeverity[s])
	}
	return b.String()
}

func parseAssessment(c captures) map[string]any {
	var doc map[string]any
	if err := json.Unmarshal([]byte(c.assessment.Stdout), &doc); err != nil {
		return nil
	}
	return doc
}

func mustIndent(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(b)
}

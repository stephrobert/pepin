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
	"github.com/stephrobert/pepin/internal/i18n"
	"github.com/stephrobert/pepin/internal/model"
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
	// L'inventaire du code de sortie 4 : UN écart high, et un seul, pour que la page
	// montre une dérogation entière plutôt qu'une liste. La règle SSH ouvert à
	// l'Internet est le cas d'école de l'exception légitime, le bastion.
	bastionInventory = `{
  "provider": "scaleway",
  "resources": [
    {
      "provider": "scaleway",
      "type": "security_group_rule",
      "id": "vm-bastion",
      "name": "vm-bastion",
      "region": "fr-par",
      "attributes": {
        "security_group_id": "sg-bastion",
        "direction": "inbound",
        "action": "accept",
        "protocol": "tcp",
        "port_from": 22,
        "port_to": 22,
        "cidrs": ["0.0.0.0/0"],
        "description": "Acces d administration du bastion"
      }
    }
  ]
}`
	// La dérogation qui couvre cet écart : datée, justifiée, attribuée. C'est le
	// fichier exact que la page fait écrire au lecteur.
	exceptionsFile = `exceptions:
  - control: network_securitygroup_allow_ingress_from_internet_to_tcp_port_22
    resource: sg-bastion
    justification: "Bastion administre, acces restreint par IP source en amont"
    expires_at: 2099-12-31
    owner: platform-security
    approved_by: security@example.org
`
)

// remediationWitness : le contrôle que le guide de remédiation suit d'un scan à l'autre. Il
// est CRITIQUE, actif chez les trois fournisseurs, et l'exemple corrigé du dépôt le fait
// passer de `fail` à `pass` — c'est-à-dire qu'il démontre la boucle complète plutôt que la
// simple disparition d'un écart.
const remediationWitness = "objectstorage_bucket_public_access"

// Chemins des fixtures du dépôt utilisées par les captures.
const (
	planVulnerable = "examples/scaleway/terraform/plan.json"
	planFixed      = "examples/scaleway/terraform-fixed/plan.json"
	planMissing    = "examples/scaleway/plan-absent.json"
	planOutscale   = "examples/outscale/terraform/plan.json"
)

// captures rassemble toutes les exécutions réelles de `pepin` dont la documentation montre la
// sortie. Aucune autre source n'est admise dans une page.
type captures struct {
	vulnerable Capture
	fixed      Capture
	assessment Capture // --format assessment sur le plan non conforme
	// assessmentFixed : le MÊME scan sur le plan corrigé. C'est l'avant/après d'une
	// remédiation, mesuré et non raconté : le même contrôle y passe de `fail` à `pass`.
	assessmentFixed Capture
	missingFile     Capture
	empty           Capture
	tagless         Capture
	taglessStr      Capture
	// Dérogations : le même inventaire, avec une exemption valide puis échue, et
	// l'assessment qui montre le statut `exempted`.
	exempted        Capture
	exemptedExpired Capture
	exemptedAsmt    Capture
	providers       Capture

	// Référence CLI : une aide par verbe, dans la langue de la page. Les captures y sont
	// tenues par POINTEUR : une valeur de carte n'est pas adressable, et la neutralisation
	// de la version de build doit pouvoir les réécrire.
	help map[string]*Capture
	// Formats de sortie : le MÊME scan rendu dans chaque format parsable.
	jsonReport Capture
	sarif      Capture
	oscal      Capture
	// Plan contre live : le contrôle qui change de statut avec la source, et l'attribut
	// qu'un plan rend en chaîne là où l'API rend un booléen.
	driftLive        Capture // assessment sur l'inventaire de démonstration
	outscalePlanJSON Capture // --format json sur le plan Outscale
	planUnknown      string  // extrait du plan : l'attribut « unknown after apply »
	planBoolAsString string  // extrait du plan : le booléen rendu en chaîne
	// Pages de fournisseurs : un scan réel du plan d'exemple de chacun.
	providerScans map[string]*Capture
	// Bundle de preuve : sceller, vérifier, altérer, caviarder.
	bundle bundleCaptures
	// Surfaces gelées : la version de forme de chacune, lue dans sa fixture.
	frozenVersions map[string]int
	cliSurface     cliFrozen
	// Fichiers d'exemple montrés tels quels (pipelines de CI), indexés par identifiant
	// de région.
	exampleFiles map[string]string
}

// all rend chaque capture du jeu. La neutralisation de la version de build s'y adosse : une
// liste énumérée à la main y oublierait une capture au premier ajout, et la page concernée
// divergerait à chaque build sans qu'aucun comportement n'ait bougé.
func (c *captures) all() []*Capture {
	out := []*Capture{&c.vulnerable, &c.fixed, &c.assessment, &c.assessmentFixed, &c.missingFile, &c.empty,
		&c.tagless, &c.taglessStr, &c.exempted, &c.exemptedExpired, &c.exemptedAsmt,
		&c.providers, &c.jsonReport, &c.sarif, &c.oscal,
		&c.driftLive, &c.outscalePlanJSON,
		&c.bundle.seal, &c.bundle.verify, &c.bundle.reDerive, &c.bundle.tampered,
		&c.bundle.redact, &c.bundle.redactRD, &c.bundle.crossVerify}
	for _, m := range []map[string]*Capture{c.help, c.providerScans} {
		names := make([]string, 0, len(m))
		for k := range m {
			names = append(names, k)
		}
		sort.Strings(names)
		for _, n := range names {
			out = append(out, m[n])
		}
	}
	return out
}

// captureAll exécute la totalité des commandes documentées. Les deux inventaires en ligne sont
// écrits dans un dossier jetable ; la commande AFFICHÉE, elle, porte le nom de fichier relatif
// que le lecteur aura créé.
func captureAll(root, bin, lang string) (captures, error) {
	r := Runner{Bin: bin, Root: root, Lang: lang}
	tmp, err := os.MkdirTemp("", "pepin-docgen-")
	if err != nil {
		return captures{}, err
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	emptyPath := filepath.Join(tmp, "empty-inventory.json")
	taglessPath := filepath.Join(tmp, "tagless-inventory.json")
	bastionPath := filepath.Join(tmp, "bastion-inventory.json")
	exceptionsPath := filepath.Join(tmp, "exceptions.yaml")
	expiredPath := filepath.Join(tmp, "exceptions-expired.yaml")
	for path, body := range map[string]string{
		emptyPath:      emptyInventory + "\n",
		taglessPath:    taglessInventory + "\n",
		bastionPath:    bastionInventory + "\n",
		exceptionsPath: exceptionsFile,
		expiredPath:    strings.ReplaceAll(exceptionsFile, "2099-12-31", "2020-01-01"),
	} {
		if werr := os.WriteFile(path, []byte(body), 0o600); werr != nil {
			return captures{}, werr
		}
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
		{&c.assessmentFixed, []string{"scan", "scaleway", "--terraform", planFixed, "--format", "assessment"}, nil},
		{&c.missingFile, []string{"scan", "scaleway", planMissing}, nil},
		{&c.empty, []string{"scan", "scaleway", emptyPath}, []string{"scan", "scaleway", "empty-inventory.json"}},
		{&c.tagless, []string{"scan", "scaleway", taglessPath}, []string{"scan", "scaleway", "tagless-inventory.json"}},
		{&c.taglessStr, []string{"scan", "scaleway", taglessPath, "--strict"}, []string{"scan", "scaleway", "tagless-inventory.json", "--strict"}},
		{&c.exempted, []string{"scan", "scaleway", bastionPath, "--exceptions", exceptionsPath},
			[]string{"scan", "scaleway", "bastion-inventory.json", "--exceptions", "exceptions.yaml"}},
		{&c.exemptedExpired, []string{"scan", "scaleway", bastionPath, "--exceptions", expiredPath},
			[]string{"scan", "scaleway", "bastion-inventory.json", "--exceptions", "exceptions-expired.yaml"}},
		{&c.exemptedAsmt, []string{"scan", "scaleway", bastionPath, "--exceptions", exceptionsPath, "--format", "assessment"},
			[]string{"scan", "scaleway", "bastion-inventory.json", "--exceptions", "exceptions.yaml", "--format", "assessment"}},
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
	ver, verr := r.Run("version")
	if verr != nil {
		return captures{}, verr
	}
	version := trimToolName(ver.Stdout)

	if err := c.captureReference(r, root, tmp); err != nil {
		return captures{}, err
	}
	bc, berr := captureBundle(r, tmp, version)
	if berr != nil {
		return captures{}, berr
	}
	c.bundle = bc

	// La VERSION est injectée au build (`git describe`) : elle diffère d'une machine et d'un
	// commit à l'autre, et le bandeau la porte. Sans neutralisation, la page divergerait à
	// chaque build sans qu'aucun comportement n'ait bougé — un contrôle qui crie pour rien
	// finit désarmé.
	//
	// La substitution est ANCRÉE sur la forme que le bandeau imprime (« v<version> ») et non
	// sur le jeton nu. Sans ancrage, un binaire construit hors dépôt (version de repli
	// « dev ») ferait remplacer « dev » à l'intérieur de « Total deviations », ce qui
	// corromprait la sortie montrée au lecteur.
	//
	// Les documents PARSABLES, eux, portent la version nue (`"version": "0.2.0"`) : la
	// forme citée y est donc celle du champ JSON, sans quoi la page de formats divergerait
	// à chaque build.
	if version != "" {
		for _, capt := range c.all() {
			capt.Stdout = strings.ReplaceAll(capt.Stdout, "v"+version, "v"+versionPlaceholder)
			capt.Stderr = strings.ReplaceAll(capt.Stderr, "v"+version, "v"+versionPlaceholder)
			capt.Stdout = strings.ReplaceAll(capt.Stdout, `"`+version+`"`, `"`+versionPlaceholder+`"`)
			capt.Stderr = strings.ReplaceAll(capt.Stderr, `"`+version+`"`, `"`+versionPlaceholder+`"`)
		}
	}
	return c, nil
}

// exampleSources : les pipelines d'exemple que les guides d'intégration montrent, avec
// l'identifiant de la région qui les porte. Ce sont des fichiers exécutables du dépôt
// (actionlint les valide), pas des extraits écrits dans une page.
var exampleSources = map[string]string{
	"example-github-workflow": "examples/github-actions/pepin.yml",
	"example-gitlab-template": "examples/gitlab-ci/pepin.gitlab-ci.yml",
	"example-gitlab-pipeline": "examples/gitlab-ci/.gitlab-ci.yml",
}

// captureReference joue les exécutions dont vivent les pages de référence (aides de la CLI,
// formats de sortie), la page de comparaison plan/live et les pages de fournisseurs, puis lit
// les surfaces gelées et les extraits de plans committés.
func (c *captures) captureReference(r Runner, root, tmp string) error {
	surface, err := loadCLISurface(root)
	if err != nil {
		return err
	}
	if err := frozenVerbsAreDocumented(surface); err != nil {
		return err
	}
	c.cliSurface = surface
	c.frozenVersions = map[string]int{}
	for _, name := range []string{"cli", "findings", "assessment", "bundle", "inventory"} {
		v, _, ferr := loadFrozen(root, name)
		if ferr != nil {
			return ferr
		}
		c.frozenVersions[name] = v
	}

	c.help = map[string]*Capture{}
	for _, verb := range append([]string{""}, documentedVerbs...) {
		args := append(strings.Fields(verb), "--help")
		got, rerr := r.Run(args...)
		if rerr != nil {
			return rerr
		}
		if strings.TrimSpace(got.Stdout) == "" {
			return fmt.Errorf("`pepin %s --help` n'a rien écrit sur la sortie standard : aide inexploitable", verb)
		}
		h := got
		c.help[verb] = &h
	}

	driftPath := filepath.Join(tmp, "live-inventory.json")
	if werr := os.WriteFile(driftPath, []byte(driftInventory+"\n"), 0o600); werr != nil {
		return werr
	}
	steps := []struct {
		into    *Capture
		args    []string
		display []string
	}{
		{&c.jsonReport, []string{"scan", "scaleway", "--terraform", planVulnerable, "--format", "json"}, nil},
		{&c.sarif, []string{"scan", "scaleway", "--terraform", planVulnerable, "--format", "sarif"}, nil},
		{&c.oscal, []string{"scan", "scaleway", "--terraform", planVulnerable, "--format", "oscal"}, nil},
		{&c.driftLive, []string{"scan", "scaleway", driftPath, "--format", "assessment"},
			[]string{"scan", "scaleway", "live-inventory.json", "--format", "assessment"}},
		{&c.outscalePlanJSON, []string{"scan", "outscale", "--terraform", planOutscale, "--format", "json"}, nil},
	}
	for _, s := range steps {
		got, rerr := r.Run(s.args...)
		if rerr != nil {
			return rerr
		}
		if s.display != nil {
			got.Args = s.display
		}
		*s.into = got
	}

	// Les documents normatifs portent un horodatage, des UUID dérivés de cet horodatage et
	// l'empreinte de provenance du binaire : neutralisés, sans quoi la page de formats
	// divergerait à chaque exécution du générateur.
	c.oscal.Stdout = normalizeVolatile(c.oscal.Stdout)
	c.sarif.Stdout = normalizeVolatile(c.sarif.Stdout)

	c.providerScans = map[string]*Capture{}
	for _, name := range documentedProviders {
		got, rerr := r.Run("scan", name, "--terraform", "examples/"+name+"/terraform/plan.json")
		if rerr != nil {
			return rerr
		}
		if strings.TrimSpace(got.Stdout) == "" {
			return fmt.Errorf("le scan du plan d'exemple de %s n'a rien produit : capture inexploitable", name)
		}
		s := got
		c.providerScans[name] = &s
	}

	// Les pipelines d'exemple sont MONTRÉS depuis leur fichier, jamais recopiés : un
	// extrait recopié se périme au premier changement d'épinglage, et une doc de CI qui
	// donne un SHA obsolète fait installer autre chose que ce qu'elle annonce.
	c.exampleFiles = map[string]string{}
	for id, rel := range exampleSources {
		raw, rerr := os.ReadFile(filepath.Join(root, rel)) // #nosec G304 -- chemin d'une table constante du paquet.
		if rerr != nil {
			return fmt.Errorf("lecture du pipeline d'exemple %s : %w", rel, rerr)
		}
		if strings.TrimSpace(string(raw)) == "" {
			return fmt.Errorf("pipeline d'exemple %s vide : la page ne montrerait rien", rel)
		}
		c.exampleFiles[id] = string(raw)
	}

	// Les deux extraits de plan que la page de comparaison commente. Ils sont LUS dans les
	// fixtures committées : un extrait recopié survivrait à la disparition de ce qu'il
	// illustre, et c'est précisément la divergence que la page dénonce.
	c.planUnknown, err = planExcerpt(root, planVulnerable, "scaleway_instance_server.web",
		[]string{"security_group_id", "public_ips", "id"})
	if err != nil {
		return err
	}
	c.planBoolAsString, err = planExcerpt(root, planOutscale, "outscale_image_launch_permission.public_omi",
		[]string{"image_id", "permission_additions"})
	return err
}

// versionPlaceholder remplace la version du binaire dans les sorties capturées.
const versionPlaceholder = "<version>"

// trimToolName isole la version dans la sortie de `pepin version`, quelle que soit la
// langue : le nom de l'outil s'écrit « pépin » en français et « pepin » en anglais.
func trimToolName(out string) string {
	v := strings.TrimSpace(out)
	for _, prefix := range []string{"pépin ", "pepin "} {
		v = strings.TrimPrefix(v, prefix)
	}
	return v
}

// buildBlocks assemble toutes les régions injectables pour une langue.
func buildBlocks(lang string, m Matrix, c captures, rem []RemediationCoverage) map[string]string {
	t := blockText(lang)
	b := map[string]string{
		"scope-disclaimer":          Fence("text", wrapDisclaimer(assess.ScopeDisclaimerIn(i18n.Lang(lang)))),
		"scan-vulnerable-head":      Fence("text", Head(c.vulnerable.Stdout, 20)),
		"scan-vulnerable-tail":      Fence("text", Tail(c.vulnerable.Stdout, 22)),
		"scan-vulnerable-full":      Fence("text", c.vulnerable.Stdout),
		"scan-vulnerable-banner":    Fence("text", c.vulnerable.Stderr),
		"scan-fixed-full":           Fence("text", c.fixed.Stdout),
		"scan-header":               Fence("text", Head(c.vulnerable.Stdout, 4)),
		"scan-control-encryption":   Fence("text", controlBlock(t, c.vulnerable.Stdout, "CLD-CHF-2")),
		"scan-control-objectstore":  Fence("text", controlBlock(t, c.vulnerable.Stdout, "CLD-STO-1")),
		"provider-list":             Fence("text", c.providers.Stdout),
		"fixture-empty-inventory":   Fence("json", emptyInventory),
		"fixture-tagless-inventory": Fence("json", taglessInventory),
		"exit-codes":                exitCodeTable(t, c),
		"exit-run-clean":            consoleRun(c.fixed, Tail(c.fixed.Stdout, 6)),
		"exit-run-nonconformity":    consoleRun(c.vulnerable, Tail(c.vulnerable.Stdout, 6)),
		"exit-run-error":            consoleRun(c.missingFile, c.missingFile.Stderr),
		"exit-run-nothing":          consoleRun(c.empty, Tail(c.empty.Stdout, 6)),
		"exit-run-strict":           consoleRun(c.taglessStr, Tail(c.taglessStr.Stdout, 6)),
		"exit-run-medium-plain":     consoleRun(c.tagless, Tail(c.tagless.Stdout, 6)),
		"fixture-bastion-inventory": Fence("json", bastionInventory),
		"fixture-exceptions":        Fence("yaml", strings.TrimRight(exceptionsFile, "\n")),
		"exit-run-exempted":         consoleRun(c.exempted, Tail(c.exempted.Stdout, 14)),
		"exit-run-expired":          consoleRun(c.exemptedExpired, c.exemptedExpired.Stderr),
		"assessment-exempted":       assessmentExtract(t, c.withAssessment(c.exemptedAsmt), "exempted", "network_securitygroup_allow_ingress_from_internet_to_tcp_port_22"),
		"assessment-run":            assessmentRunBlock(c),
		"assessment-counts":         assessmentCountsTable(t, c),
		"assessment-fail":           assessmentExtract(t, c, "fail", "objectstorage_bucket_public_access"),
		"assessment-pass":           assessmentExtract(t, c, "pass", "network_securitygroup_allow_ingress_from_internet_to_all_ports"),
		"assessment-na":             assessmentExtract(t, c, "not-applicable", "blockstorage_volume_encryption"),
		"assessment-ne":             assessmentExtract(t, c, "not-evaluated", "compute_instance_public_ip_with_open_securitygroup"),
		"not-evaluated-reasons":     notEvaluatedReasons(t, c),
		"required-attrs":            requiredAttrTable(t),
		"never-pass":                neverPassTable(t, m),
		"single-source":             singleSourceTable(t, m),
		"not-applicable-list":       notApplicableTable(t, m),
		"remediation-coverage":      remediationTable(t, rem),
		"remediation-before":        assessmentControl(driftText(lang), c.assessment, remediationWitness),
		"remediation-after":         assessmentControl(driftText(lang), c.assessmentFixed, remediationWitness),
		"coverage-totals":           m.countsTable(coverageText(lang)),
		"control-counts":            controlCountsTable(t, m),
		"inventory-format":          inventoryFormatBlock(t),
		"inventory-types":           inventoryTypesTable(t),
	}
	// Les régions des pages de la vague 2. Elles vivent dans leurs propres fichiers (la
	// référence CLI, les formats, la comparaison plan/live, le bundle, les fournisseurs)
	// pour que chacune reste lisible, et se fondent ici en un seul index de régions.
	for id, body := range c.exampleFiles {
		b[id] = Fence("yaml", body)
	}
	for _, extra := range []map[string]string{
		referenceBlocks(lang, c),
		formatBlocks(lang, c),
		driftBlocks(lang, c),
		bundleBlocks(lang, c),
		providerBlocks(lang, m, c.providerScans),
	} {
		for id, body := range extra {
			b[id] = body
		}
	}
	return b
}

// referenceBlocks assemble les régions de la référence CLI.
func referenceBlocks(lang string, c captures) map[string]string {
	t := refText(lang)
	out := map[string]string{
		"cli-verbs":        cliVerbsTable(t, c.cliSurface),
		"cli-exit-codes":   cliExitCodesTable(t, c.cliSurface),
		"surface-versions": surfaceVersionsTable(t, c.frozenVersions),
	}
	for verb, capt := range c.help {
		out[helpBlockID(verb)] = Fence("text", capt.Stdout)
	}
	return out
}

// formatBlocks assemble les régions de la page des formats de sortie : le MÊME scan, rendu
// dans chaque format, et l'extrait qui montre ce qu'un pipeline y lira.
func formatBlocks(lang string, c captures) map[string]string {
	_ = lang // les extraits sont des documents, pas des libellés : rien à traduire ici
	return map[string]string{
		"format-json-summary": jsonField(c.jsonReport, "summary"),
		"format-json-finding": jsonField(c.jsonReport, "findings", "0"),
		"format-sarif-head":   Fence("json", Head(c.sarif.Stdout, 22)),
		"format-sarif-result": jsonField(c.sarif, "runs", "0", "results", "0"),
		"format-oscal-head":   Fence("json", Head(c.oscal.Stdout, 24)),
	}
}

// bundleBlocks assemble les régions du guide du bundle de preuve.
func bundleBlocks(lang string, c captures) map[string]string {
	t := bundleText(lang)
	b := c.bundle
	return map[string]string{
		"bundle-seal":         consoleRun(b.seal, linesWithPrefix(b.seal.Stderr, "pepin:")),
		"bundle-files":        bundleFilesTable(t, b.files),
		"bundle-manifest":     Fence("json", b.manifest),
		"bundle-checksums":    Fence("text", b.checksums),
		"bundle-verify":       consoleRun(b.verify, b.verify.Stdout),
		"bundle-rederive":     consoleRun(b.reDerive, b.reDerive.Stdout),
		"bundle-tampered":     consoleRun(b.tampered, b.tampered.Stdout+b.tampered.Stderr),
		"bundle-redact":       Fence("json", b.redacted),
		"bundle-redact-rd":    consoleRun(b.redactRD, b.redactRD.Stdout+b.redactRD.Stderr),
		"bundle-cross-lang":   crossLangTable(t, b.crossLang),
		"bundle-cross-verify": consoleRun(b.crossVerify, b.crossVerify.Stdout),
	}
}

// controlBlock extrait du rapport terminal le bloc d'UN contrôle, de la règle horizontale qui
// l'ouvre jusqu'à sa ligne de documentation. Sert à commenter un écart précis sans recopier
// à la main un fragment de sortie : recopié, il survivrait à un changement de rendu et
// contredirait le rapport complet affiché quelques lignes plus haut.
//
// Un contrôle absent du scan rend une note EXPLICITE plutôt qu'un bloc vide : la page doit
// dire qu'elle n'a pas l'exemple, jamais faire semblant de l'avoir.
func controlBlock(t blockStrings, stdout, code string) string {
	lines := strings.Split(stdout, "\n")
	start := -1
	for i, l := range lines {
		if strings.Contains(l, "·  "+code+"  ·") {
			start = i
			break
		}
	}
	if start < 0 {
		return strings.ReplaceAll(t.noDeviationFor, "%s", code)
	}
	if start > 0 && strings.HasPrefix(strings.TrimSpace(lines[start-1]), "─") {
		start-- // la règle horizontale qui ouvre le bloc
	}
	for j := start; j < len(lines); j++ {
		if strings.Contains(lines[j], "↳ docs:") {
			return strings.Join(lines[start:j+1], "\n")
		}
	}
	return strings.Join(lines[start:], "\n")
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
		{t.exitExempted, c.exempted},
		{t.exitExpired, c.exemptedExpired},
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
func assessmentExtract(t blockStrings, c captures, status, preferred string) string {
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
		return Fence("text", strings.ReplaceAll(t.noResultFor, "%s", status))
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
			key := generalizeReason(obs, t.quotedPlaceholder)
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

// quotedName repère un nom (type, attribut, état) cité par le moteur : guillemets
// français en français, guillemets droits en anglais. Le moteur cite dans la langue
// du rapport, la généralisation doit donc reconnaître les deux.
var quotedName = regexp.MustCompile(`«[^»]*»|"[^"]*"`)

// generalizeReason remplace CHAQUE nom cité d'un motif par un marqueur générique, pour
// regrouper les motifs de MÊME NATURE sans en inventer le libellé. Chaque span est traité
// séparément : englober du premier au dernier guillemet avalerait le texte intermédiaire, et
// la ligne du tableau ne dirait plus ce que le moteur dit.
func generalizeReason(s, placeholder string) string {
	return quotedName.ReplaceAllString(s, placeholder)
}

// requiredAttrTable rend la table des attributs décisifs telle qu'elle est APPLIQUÉE
// (assess.RequiredAttrs), pas une paraphrase.
// inventoryFormatBlock rend la chaîne de format du schéma d'inventaire, telle que le
// code la publie. Recopiée dans la page, elle vieillirait au premier incrément.
func inventoryFormatBlock(t blockStrings) string {
	return Fence("text", model.InventoryFormat)
}

// inventoryTypesTable énumère le vocabulaire de l'inventaire : chaque type et ses
// attributs communs. DÉRIVÉ des descripteurs chargés et des collecteurs, donc
// impossible à laisser diverger — une énumération fausse est pire qu'absente.
func inventoryTypesTable(t blockStrings) string {
	cat := genprovider.InventoryCatalogue()
	types := make([]string, 0, len(cat))
	for typ := range cat {
		types = append(types, typ)
	}
	sort.Strings(types)
	var b strings.Builder
	b.WriteString("| " + t.colType + " | " + t.colAttributes + " |\n|---|---|\n")
	for _, typ := range types {
		quoted := make([]string, 0, len(cat[typ]))
		for _, a := range cat[typ] {
			quoted = append(quoted, "`"+a+"`")
		}
		_, _ = fmt.Fprintf(&b, "| `%s` | %s |\n", typ, strings.Join(quoted, " "))
	}
	return b.String()
}

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

// withAssessment rend une COPIE du jeu de captures dont l'assessment est celui
// passé en argument. Les extracteurs lisent `c.assessment` ; plusieurs pages ont
// besoin d'un autre scan (ici celui qui porte une dérogation) sans qu'il faille
// dupliquer l'extracteur.
func (c captures) withAssessment(a Capture) captures {
	c.assessment = a
	return c
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

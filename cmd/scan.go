package cmd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/stephrobert/pepin/internal/assess"
	"github.com/stephrobert/pepin/internal/commonrules"
	"github.com/stephrobert/pepin/internal/genprovider"
	"github.com/stephrobert/pepin/internal/model"
	"github.com/stephrobert/pepin/internal/provider"
	"github.com/stephrobert/pepin/internal/scoring"
	"github.com/stephrobert/pepin/internal/tfparse"
	"github.com/stephrobert/pepin/referentiel"
	"github.com/stephrobert/scankit/assessment"
	"github.com/stephrobert/scankit/engine"
	"github.com/stephrobert/scankit/finding"
	screport "github.com/stephrobert/scankit/report"
	scscoring "github.com/stephrobert/scankit/scoring"
)

var (
	scanFormat     string
	policyDirs     []string
	scanTF         bool
	scanLive       bool
	scanRegion     string
	scanProfile    string
	scanS3Endpoint string
	scanSeal       string
	scanTimestamp  string // instant d'évaluation (RFC3339 UTC), partagé input.evaluated_at + Run.Timestamp
)

var scanCmd = &cobra.Command{
	Use:   "scan <provider> [export.json]",
	Short: "Évaluer la posture d'un cloud contre les politiques",
	Long: "Évalue un inventaire contre les règles embarquées du provider (+ règles\n" +
		"externes via --policy-dir). Trois sources : un export JSON normalisé, un plan\n" +
		"Terraform (--terraform), ou une collecte live de l'API (--live).",
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		path := ""
		if len(args) > 1 {
			path = args[1]
		}
		p, ok := provider.Get(name)
		if !ok {
			return fmt.Errorf("provider inconnu : %q (voir `pepin providers`)", name)
		}
		if !scanLive && path == "" {
			return fmt.Errorf("préciser un fichier (export JSON ou plan Terraform), ou utiliser --live")
		}
		scanTimestamp = time.Now().UTC().Format(time.RFC3339) // un seul instant, partagé input + run

		opts := scanReportOptions(name, scanSource(path))

		// En mode terminal, afficher le bandeau dès le lancement (avant la
		// collecte/évaluation, qui peut être longue en live), pas à la fin.
		if scanFormat == "table" {
			_ = screport.Banner(os.Stderr, opts) // best-effort banner to stderr
		}

		input, err := loadInput(cmd.Context(), p, path)
		if err != nil {
			return err
		}
		// Injecte la ressource synthétique de souveraineté (CLD-GVN-4), métadonnée
		// du fournisseur indépendante de l'inventaire collecté/importé.
		input = withGovernance(p, input)
		// Horodatage d'évaluation porté PAR l'input : les règles sensibles au temps s'y
		// ancrent (au lieu de l'horloge), donc le bundle (input.json) rejoue le même verdict.
		if m, ok := input.(map[string]any); ok {
			m["evaluated_at"] = scanTimestamp
		}

		// Toutes les règles sont communes ; le provider ne fournit que la
		// collecte (la source). On peut ajouter des règles externes via --policy-dir.
		sources := []fs.FS{commonrules.FS()}
		for _, dir := range policyDirs {
			sources = append(sources, os.DirFS(dir))
		}

		findings, err := engine.Evaluate(cmd.Context(), input, sources...)
		if err != nil {
			return err
		}
		// Build the opposable assessment from the AGNOSTIC-coded findings, BEFORE enrichment
		// rewrites Code to the SCSL id: typed statuses (fail/pass/not-evaluated), exact
		// normative references, and run provenance (tool/ruleset digests, target, timestamp).
		rtypes := resourceTypesOf(input)
		asmt := assess.Build(name, referentiel.All(), findings, rtypes, providerNAReasons(name), providerVerified(name), controlTypes(), buildRun(name, rtypes))

		// Bundle de preuve horodaté et hashé (opposabilité : intégrité + non-répudiation).
		if scanSeal != "" {
			inputJSON, merr := json.MarshalIndent(input, "", "  ") // l'inventaire EXACT évalué
			if merr != nil {
				return fmt.Errorf("sérialisation de l'inventaire évalué : %w", merr)
			}
			cs, err := assess.WriteBundle(scanSeal, asmt, inputJSON)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(os.Stderr, "pepin: bundle de preuve écrit dans %s — sceller : cosign sign-blob %s\n", scanSeal, cs)
		}

		enrichFromReferentiel(findings)
		res := scoring.Summarize(findings)
		opts.SummaryHeadline = verdictHeadline(res, asmt, asmt.Run.Source)

		switch scanFormat {
		case "json":
			if err := renderJSON(findings, res); err != nil {
				return err
			}
		case "assessment":
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(asmt); err != nil {
				return err
			}
		case "oscal":
			if err := screport.OSCAL(os.Stdout, asmt); err != nil {
				return err
			}
		case "sarif":
			if err := screport.SARIF(os.Stdout, opts, path, findings); err != nil {
				return err
			}
		default:
			_ = screport.Terminal(os.Stdout, opts, findings, scscoring.Summarize(findings)) // best-effort render
		}
		if !res.Conforme {
			os.Exit(1)
		}
		return nil
	},
}

func init() {
	scanCmd.Flags().StringVarP(&scanFormat, "format", "f", "table", "format de sortie : table | json | assessment | oscal | sarif")
	scanCmd.Flags().StringArrayVarP(&policyDirs, "policy-dir", "p", nil,
		"répertoire de règles externes (.rego), répétable — chargé sans recompilation")
	scanCmd.Flags().BoolVarP(&scanTF, "terraform", "t", false,
		"auditer un plan Terraform (`terraform show -json`) au lieu d'un export d'inventaire")
	scanCmd.Flags().BoolVar(&scanLive, "live", false,
		"collecter l'inventaire en direct via l'API du provider (identifiants requis)")
	scanCmd.Flags().StringVar(&scanRegion, "region", "", "région cible pour la collecte live")
	scanCmd.Flags().StringVar(&scanProfile, "profile", "", "profil d'identifiants pour la collecte live (ex. ~/.osc/config.json)")
	scanCmd.Flags().StringVar(&scanS3Endpoint, "s3-endpoint", "", "endpoint S3 custom pour le stockage objet (collecte live ; ex. MinIO http://localhost:9000)")
	scanCmd.Flags().StringVar(&scanSeal, "seal", "", "écrire un bundle de preuve opposable (assessment + OSCAL + manifest + checksums) dans ce dossier")
}

// enrichFromReferentiel rattache chaque finding à l'index SCSL. Les règles Rego
// émettent un identifiant de check agnostique ; on le résout dans le référentiel
// pour exposer le `code` SCSL (CLD-*) comme identifiant de contrôle, conserver le
// check agnostique en `labels.check`, et poser le titre stable du contrôle. Les
// règles restent ainsi découplées de la numérotation SCSL.
func enrichFromReferentiel(findings []finding.Finding) {
	for i := range findings {
		ctl, ok := referentiel.Lookup(findings[i].Code)
		if !ok {
			continue
		}
		if findings[i].Labels == nil {
			findings[i].Labels = map[string]string{}
		}
		findings[i].Labels["check"] = findings[i].Code
		findings[i].Title = ctl.Titre
		if ctl.SCSL() != "" {
			findings[i].Code = ctl.SCSL()
		}
	}
}

// scanSource décrit la provenance des données pour le rapport. En live, le
// chemin est vide : on affiche le mode, le profil et la région à la place.
func scanSource(path string) string {
	if !scanLive {
		return path
	}
	src := "collecte live"
	if scanProfile != "" {
		src += " · profil " + scanProfile
	}
	if scanRegion != "" {
		src += " · région " + scanRegion
	}
	return src
}

// resourceTypesOf collecte l'ensemble des types de ressources présents dans l'inventaire
// évalué, en gérant la forme live/Terraform ([]model.Resource) et la forme export JSON.
func resourceTypesOf(input any) map[string]bool {
	out := map[string]bool{}
	m, ok := input.(map[string]any)
	if !ok {
		return out
	}
	switch rs := m["resources"].(type) {
	case []model.Resource:
		for _, r := range rs {
			if r.Type != "" {
				out[r.Type] = true
			}
		}
	case []any:
		for _, it := range rs {
			if rm, ok := it.(map[string]any); ok {
				if t, ok := rm["type"].(string); ok && t != "" {
					out[t] = true
				}
			}
		}
	}
	return out
}

// providerNAReasons collecte, pour chaque contrôle non applicable au provider scanné, sa
// justification opposable (contrat du provider : mécanisme inexistant, ou type de ressource
// absent de l'API). Un contrôle sans justification n'est PAS marqué non applicable.
func providerNAReasons(provider string) map[string]string {
	out := map[string]string{}
	for code := range referentiel.All() {
		if r := genprovider.NonApplicableReason(provider, code); r != "" {
			out[code] = r
		}
	}
	return out
}

// governanceProviderReaders : contrôles de gouvernance dont la donnée est la ressource
// synthétique `governance_provider` (toujours construite depuis le descripteur). Leur source
// est donc intrinsèquement collectée ; les autres contrôles de gouvernance (ex. étiquettes,
// qui dépendent de la collecte d'attributs par ressource) ne sont PAS présumés vérifiés.
var governanceProviderReaders = map[string]bool{
	"governance_resource_region_in_eu": true,
	"governance_provider_sovereignty":  true,
}

// controlTypes retourne, par code de contrôle, le type de ressource normalisé qu'il évalue
// (genprovider.ControlType). assess s'en sert pour tester la présence du TYPE EXACT dans
// l'inventaire — un « pass » exige une ressource du type visé, pas d'une famille voisine.
func controlTypes() map[string]string {
	out := map[string]string{}
	for code := range referentiel.All() {
		out[code] = genprovider.ControlType(code)
	}
	return out
}

// providerVerified indique, par contrôle, si le contrat du provider CONFIRME que la donnée
// nécessaire est réellement collectée (état `verifie` du type visé). C'est le verrou d'un
// « pass » opposable : sans confirmation, assess reste sur NotEvaluated plutôt que d'affirmer
// une conformité qu'une garde de capacité a pu court-circuiter silencieusement.
func providerVerified(provider string) map[string]bool {
	out := map[string]bool{}
	for code := range referentiel.All() {
		switch {
		case governanceProviderReaders[code]:
			out[code] = true
		case genprovider.ControlType(code) != "":
			out[code] = genprovider.TypeEtat(provider, genprovider.ControlType(code)) == "verifie"
		default:
			out[code] = false
		}
	}
	return out
}

// buildRun assemble l'enveloppe de provenance du scan : empreintes outil + jeu de règles,
// cible, horodatage, source et périmètre (types de ressources audités). C'est ce qui rend le
// résultat attribuable et reproductible — donc opposable.
func buildRun(provider string, rtypes map[string]bool) assessment.Run {
	source := "export"
	switch {
	case scanLive:
		source = "live-api"
	case scanTF:
		source = "terraform-plan"
	}
	included := make([]string, 0, len(rtypes))
	for t := range rtypes {
		included = append(included, t)
	}
	sort.Strings(included)
	return assessment.Run{
		Tool:      assessment.Component{Name: "pepin", Version: version},
		Ruleset:   assessment.Component{Name: "pepin-config", Digest: configDigest()},
		Target:    assessment.Target{Provider: provider, Region: scanRegion, Platform: provider},
		Timestamp: scanTimestamp,
		Source:    source,
		Scope:     assessment.Scope{Included: included},
	}
}

// configDigest empreinte TOUT ce qui détermine le résultat : les règles Rego communes, les
// descripteurs providers (verified/N/A/projection/souveraineté), le référentiel (sévérités,
// références, fournisseurs) ET les répertoires de règles externes (--policy-dir). Deux
// configurations différentes ne peuvent PAS produire le même digest sous le même résultat.
func configDigest() string {
	h := sha256.New()
	h.Write([]byte(assess.RulesetDigest(commonrules.FS()))) // règles communes embarquées
	h.Write([]byte(genprovider.DescriptorsDigest()))        // descripteurs providers
	h.Write(referentiel.Raw())                              // référentiel (mappings/sévérités/fournisseurs)
	for _, dir := range policyDirs {                        // règles externes éventuelles
		h.Write([]byte(dir))
		h.Write([]byte(assess.RulesetDigest(os.DirFS(dir))))
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

// loadInput charge l'entrée à évaluer : collecte live de l'API (--live), plan
// Terraform (--terraform, projeté par le mapper du provider), ou export
// d'inventaire JSON déjà normalisé.
func loadInput(ctx context.Context, p provider.Provider, path string) (any, error) {
	if scanLive {
		cfg := provider.Config{Profile: scanProfile, Region: scanRegion}
		if scanS3Endpoint != "" {
			cfg.Options = map[string]string{"s3_endpoint": scanS3Endpoint}
		}
		resources, err := p.Collect(ctx, cfg)
		if err != nil {
			return nil, err
		}
		return map[string]any{"provider": p.Name(), "resources": resources}, nil
	}
	if scanTF {
		mapper, ok := p.(provider.TerraformMapper)
		if !ok {
			return nil, fmt.Errorf("le provider %q ne supporte pas l'audit Terraform", p.Name())
		}
		resources, err := tfparse.ParsePlan(path)
		if err != nil {
			return nil, err
		}
		mapped, err := mapper.MapTerraform(resources)
		if err != nil {
			return nil, err
		}
		return map[string]any{"provider": p.Name(), "resources": mapped}, nil
	}

	// #nosec G304 -- path est l'export choisi par l'utilisateur en argument CLI, lu en seule lecture ; pas de traversée à craindre.
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var input any
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil, fmt.Errorf("export JSON invalide : %w", err)
	}
	return input, nil
}

// withGovernance ajoute, le cas échéant, la ressource synthétique de souveraineté
// du fournisseur (CLD-GVN-4) à l'inventaire évalué, quel que soit le mode (live,
// Terraform, export). Les FAITS de souveraineté sont déclarés dans le YAML du
// provider (providers/<nom>.yaml, section `souverainete`) ; ici, code générique.
func withGovernance(p provider.Provider, input any) any {
	gp, ok := p.(provider.GovernanceProvider)
	if !ok {
		return input
	}
	res, ok := gp.GovernanceResource()
	if !ok {
		return input
	}
	m, ok := input.(map[string]any)
	if !ok {
		return input
	}
	switch rs := m["resources"].(type) {
	case []model.Resource:
		m["resources"] = append(rs, res)
	case []any:
		var rm any
		if b, err := json.Marshal(res); err == nil && json.Unmarshal(b, &rm) == nil {
			m["resources"] = append(rs, rm)
		}
	}
	return m
}

// scslDocBase est la racine de la doc SCSL ; on y concatène l'id de l'exigence
// SCSL (CLD-*) du contrôle pour pointer sa page (explication + remédiation).
const scslDocBase = "https://stephane-robert.info/scsl/"

// docURL pointe la page SCSL du finding. Après enrichissement, `code` est l'id
// SCSL (CLD-*) ; sinon le check agnostique sert de repli.
func docURL(f finding.Finding) string {
	return scslDocBase + f.Code
}

// verdictHeadline dérive le bandeau de verdict de l'ASSESSMENT, pas des seuls findings :
// un scan où RIEN n'a été évalué (collecte vide, gardes de capacité) ne doit jamais s'afficher
// « CONFORME », et un verdict sur un plan Terraform est qualifié « périmètre déclaré » (état
// planifié, pas configuration effective). Le code de sortie reste piloté par res.Conforme.
func verdictHeadline(res scoring.Result, asmt assessment.Assessment, source string) string {
	var pass, fail, evaluated int
	for _, r := range asmt.Results {
		switch r.Status {
		case assessment.Pass:
			pass++
			evaluated++
		case assessment.Fail:
			fail++
			evaluated++
		}
	}
	scope := "périmètre évalué"
	if source == "terraform-plan" {
		scope = "périmètre déclaré (plan Terraform, état planifié)"
	}
	switch {
	case evaluated == 0:
		return "Verdict : INDÉTERMINÉ — aucun contrôle évalué sur le " + scope
	case !res.Conforme:
		return "Verdict : NON CONFORME"
	default:
		return fmt.Sprintf("Verdict : aucune non-conformité critique ou haute sur le %s (%d conformes)", scope, pass)
	}
}

// scanReportOptions assemble les options de rendu scankit pour Pépin. Le bandeau de verdict
// (SummaryHeadline) est renseigné plus tard, une fois l'assessment calculé (verdictHeadline).
func scanReportOptions(provName, path string) screport.Options {
	mode := "scan " + provName
	if scanTF {
		mode += " (terraform)"
	}
	if scanLive {
		mode += " (live)"
	}
	return screport.Options{
		ToolName: "pepin",
		Version:  version,
		Mode:     mode,
		Source:   path,
		Banner:   pepinBanner(),
		Tagline:  "scanner de posture cloud (sécurité · conformité)",
		Brand:    lipgloss.Color("#C792EA"),
		TierOf:   func(f finding.Finding) string { return f.Label("provider") },
		DocURL:   docURL,
	}
}

// pepinBanner construit le logo ASCII « PEPIN » (style ANSI Shadow).
func pepinBanner() []string {
	letters := map[rune][]string{
		'P': {"██████╗ ", "██╔══██╗", "██████╔╝", "██╔═══╝ ", "██║     ", "╚═╝     "},
		'E': {"███████╗", "██╔════╝", "█████╗  ", "██╔══╝  ", "███████╗", "╚══════╝"},
		'I': {"██╗ ", "██║ ", "██║ ", "██║ ", "██║ ", "╚═╝ "},
		'N': {"███╗   ██╗", "████╗  ██║", "██╔██╗ ██║", "██║╚██╗██║", "██║ ╚████║", "╚═╝  ╚═══╝"},
	}
	lines := make([]string, 6)
	for _, r := range "PEPIN" {
		for i := 0; i < 6; i++ {
			lines[i] += letters[r][i]
		}
	}
	return lines
}

func renderJSON(findings []finding.Finding, res scoring.Result) error {
	out := map[string]any{"findings": findings, "summary": res}
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	_, _ = fmt.Println(string(b))
	return nil
}

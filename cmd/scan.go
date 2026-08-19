package cmd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/stephrobert/pepin/internal/assess"
	"github.com/stephrobert/pepin/internal/commonrules"
	"github.com/stephrobert/pepin/internal/genprovider"
	"github.com/stephrobert/pepin/internal/i18n"
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
	scanKubeconfig string
	scanS3Endpoint string
	scanSeal       string
	scanRedact     bool   // caviarder les valeurs sensibles de l'input.json embarqué
	scanStrict     bool   // porte CI stricte : échec sur couverture nulle ou écart medium/low
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
			return fmt.Errorf(tr("provider inconnu : %q (voir `pepin providers`)",
				"unknown provider: %q (see `pepin providers`)"), name)
		}
		if !scanLive && path == "" {
			return errors.New(tr(
				"préciser un fichier (export JSON ou plan Terraform), ou utiliser --live",
				"give a file (JSON export or Terraform plan), or use --live"))
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
		// du fournisseur indépendante de l'inventaire collecté/importé. Idempotent :
		// rejouer l'input.json d'un bundle (déjà augmenté) ne duplique pas la ressource.
		input = withGovernance(p, input)
		// Horodatage d'évaluation porté PAR l'input : les règles sensibles au temps s'y
		// ancrent (au lieu de l'horloge), donc le bundle (input.json) rejoue le MÊME verdict.
		// On NE l'écrase PAS s'il est déjà présent (rejeu d'un input.json scellé) : sinon
		// l'ancrage temporel gelé serait perdu et le rejeu donnerait un verdict différent.
		if m, ok := input.(map[string]any); ok {
			if _, present := m["evaluated_at"]; !present {
				m["evaluated_at"] = scanTimestamp
			}
		}

		// Toutes les règles sont communes ; le provider ne fournit que la
		// collecte (la source). On peut ajouter des règles externes via --policy-dir.
		sources := []fs.FS{commonrules.FS()}
		for _, dir := range policyDirs {
			sources = append(sources, os.DirFS(dir))
		}

		// Borne de temps sur l'évaluation. Les règles de --policy-dir sont du code tiers :
		// une règle coûteuse (ou simplement fautive) figerait un job de CI indéfiniment,
		// sans qu'aucune deadline ne traverse le moteur. scankit refuse déjà les builtins
		// réseau ; le temps est l'autre ressource qu'une politique peut consommer.
		evalCtx, cancel := context.WithTimeout(cmd.Context(), evalTimeout)
		defer cancel()
		findings, err := engine.Evaluate(evalCtx, input, sources...)
		if err != nil {
			return err
		}
		// La traduction voyage dans les labels des règles ; on la consomme ICI, avant
		// assess.Build, pour que l'assessment, l'OSCAL et le bundle scellé parlent la
		// même langue que le terminal. Le rendu de scankit, lui, n'a rien à savoir
		// de la langue : il reçoit des findings déjà dans la bonne.
		localizeFindings(findings)
		// Build the opposable assessment from the AGNOSTIC-coded findings, BEFORE enrichment
		// rewrites Code to the SCSL id: typed statuses (fail/pass/not-evaluated), exact
		// normative references, and run provenance (tool/ruleset digests, target, timestamp).
		rtypes := resourceTypesOf(input)
		asmt := assess.Build(name, referentiel.All(), findings, rtypes, providerNAReasons(name), providerVerified(name), controlTypes(), attrsByTypeOf(input), buildRun(name, rtypes))

		// Bundle de preuve horodaté et hashé (opposabilité : intégrité + non-répudiation).
		if scanSeal != "" {
			bundleInput := input
			if scanRedact {
				// Caviarde les valeurs sensibles (user-data, documents de policy) de l'inventaire
				// embarqué : un bundle remis à un tiers ne doit pas exfiltrer les secrets détectés.
				// INCOMPATIBLE avec `verify --re-derive` (la détection ne rejoue pas sur du caviardé)
				// → un bundle caviardé s'appuie sur la SIGNATURE cosign, pas sur la re-dérivation.
				bundleInput = redactInventory(input)
			}
			inputJSON, merr := json.MarshalIndent(bundleInput, "", "  ")
			if merr != nil {
				return fmt.Errorf(tr("sérialisation de l'inventaire évalué : %w",
					"serializing the evaluated inventory: %w"), merr)
			}
			cs, err := assess.WriteBundle(scanSeal, asmt, inputJSON)
			if err != nil {
				return err
			}
			if !scanRedact {
				_, _ = fmt.Fprintln(os.Stderr, tr(
					"pepin: ⚠ input.json embarque l'inventaire BRUT (peut contenir des secrets : user-data, policies). Traiter le bundle comme SENSIBLE, ou utiliser --redact pour le partager.",
					"pepin: ⚠ input.json embeds the RAW inventory (it may contain secrets: user-data, policies). Treat the bundle as SENSITIVE, or use --redact to share it."))
			}
			_, _ = fmt.Fprintf(os.Stderr, tr(
				"pepin: bundle de preuve écrit dans %s — sceller : cosign sign-blob %s\n",
				"pepin: evidence bundle written to %s — seal it with: cosign sign-blob %s\n"), scanSeal, cs)
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
		// Portée prestataire/commanditaire : obligatoire pour l'opposabilité (un rapport pepin
		// ne prouve pas une qualification, seulement la posture d'un tenant).
		_, _ = fmt.Fprintf(os.Stderr, "\nⓘ %s\n", assess.ScopeDisclaimer())
		if !res.Conforme {
			os.Exit(exitNonConformite)
		}
		// Couverture nulle : JAMAIS 0, et sans avoir à demander --strict. « Aucun finding »
		// et « rien n'a été mesuré » produisent le même résultat vide, mais le second ne dit
		// rien de la posture : identifiants expirés, droits insuffisants, région vide ou
		// inventaire tronqué rendraient une porte de CI verte sur un périmètre jamais regardé.
		// Le bandeau annonce déjà « INDÉTERMINÉ » dans ce cas — le code de sortie doit le suivre.
		if evaluatedNonGov(asmt) == 0 {
			os.Exit(exitStrict)
		}
		// --strict : porte CI plus exigeante, qui refuse aussi les écarts medium/low
		// (que le code de sortie normal ignore).
		if scanStrict && res.Medium+res.Low > 0 {
			os.Exit(exitStrict)
		}
		return nil
	},
}

// evalTimeout borne l'évaluation des politiques. Généreux à dessein : un inventaire de
// dizaines de milliers de ressources reste sous cette borne, qui ne vise que la règle
// externe partie en boucle, pas le gros scan légitime.
const evalTimeout = 5 * time.Minute

// sensitiveAttrs : attributs dont la VALEUR peut porter un secret et qu'on caviarde dans le
// bundle partageable (l'outil détecte les secrets ; il ne doit pas les ré-exfiltrer).
//
// La liste couvre les attributs du modèle normalisé, pas seulement les documents libres :
// `access_key` est un attribut de ressource à part entière (cf. examples/scaleway/inventory.json,
// type "access_key"), et `password`/`certificate` remontent des bases managées. Un bundle
// `--seal --redact` est destiné à un auditeur EXTERNE : ce qui traverse cette carte quitte
// le périmètre du tenant scanné.
var sensitiveAttrs = map[string]bool{
	"user_data": true, "document": true, "statements": true, "policy": true,
	"access_key": true, "secret_key": true, "password": true, "token": true,
	"ssh_key": true, "public_key": true, "private_key": true,
	"certificate": true, "connection_string": true,
}

// redactInventory retourne une COPIE de l'inventaire dont les valeurs des attributs sensibles
// sont remplacées par une empreinte (le finding reste, la valeur brute disparaît). Ne mute pas
// l'input évalué (la détection a déjà eu lieu en amont).
func redactInventory(input any) any {
	m, ok := input.(map[string]any)
	if !ok {
		return input
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	redactAttrs := func(attrs map[string]any) map[string]any {
		ac := make(map[string]any, len(attrs))
		for k, v := range attrs {
			if sensitiveAttrs[k] && v != nil {
				b, _ := json.Marshal(v)
				sum := sha256.Sum256(b)
				ac[k] = "[REDACTED sha256:" + hex.EncodeToString(sum[:])[:16] + "]"
			} else {
				ac[k] = v
			}
		}
		return ac
	}
	switch rs := m["resources"].(type) {
	case []model.Resource: // source live/terraform (typée)
		nr := make([]model.Resource, len(rs))
		for i, r := range rs {
			r.Attributes = redactAttrs(r.Attributes)
			nr[i] = r
		}
		out["resources"] = nr
	case []any: // source export JSON générique
		nr := make([]any, len(rs))
		for i, r := range rs {
			rm, ok := r.(map[string]any)
			if !ok {
				nr[i] = r
				continue
			}
			rc := make(map[string]any, len(rm))
			for k, v := range rm {
				rc[k] = v
			}
			if attrs, ok := rm["attributes"].(map[string]any); ok {
				rc["attributes"] = redactAttrs(attrs)
			}
			nr[i] = rc
		}
		out["resources"] = nr
	default:
		return input
	}
	return out
}

// evaluatedNonGov compte les contrôles réellement mesurés (pass/fail) hors gouvernance.
func evaluatedNonGov(asmt assessment.Assessment) int {
	n := 0
	for _, r := range asmt.Results {
		if strings.HasPrefix(r.Control, "governance_") {
			continue
		}
		if r.Status == assessment.Pass || r.Status == assessment.Fail {
			n++
		}
	}
	return n
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
	scanCmd.Flags().StringVar(&scanKubeconfig, "kubeconfig", "", "chemin d'un kubeconfig pour auditer l'état DANS un cluster Kubernetes (utiliser un accès en LECTURE SEULE, TTL court — jamais cluster-admin)")
	scanCmd.Flags().StringVar(&scanProfile, "profile", "", "profil d'identifiants pour la collecte live (ex. ~/.osc/config.json)")
	scanCmd.Flags().StringVar(&scanS3Endpoint, "s3-endpoint", "", "endpoint S3 custom pour le stockage objet (collecte live ; ex. MinIO http://localhost:9000)")
	scanCmd.Flags().StringVar(&scanSeal, "seal", "", "écrire un bundle de preuve opposable (assessment + OSCAL + manifest + checksums) dans ce dossier")
	scanCmd.Flags().BoolVar(&scanStrict, "strict", false, "porte CI stricte : code de sortie ≠ 0 si aucun contrôle n'est mesuré (hors gouvernance) ou s'il subsiste un écart medium/low")
	scanCmd.Flags().BoolVar(&scanRedact, "redact", false, "caviarder les valeurs sensibles (user-data, policies) de l'input.json du bundle — pour partage à un tiers ; INCOMPATIBLE avec verify --re-derive")
	// --live et --terraform sont exclusifs : sinon loadInput privilégie le live et ignore le plan en silence.
	scanCmd.MarkFlagsMutuallyExclusive("live", "terraform")
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
		findings[i].Title = ctl.TitreIn(i18n.Current())
		if ctl.SCSL() != "" {
			findings[i].Code = ctl.SCSL()
		}
	}
}

// labelMessageEn / labelRemediationEn : les deux labels par lesquels une règle Rego
// transporte sa traduction anglaise. Le modèle amont `scankit/finding.Finding` porte
// `Labels map[string]string`, extensible : la traduction voyage donc DANS le finding,
// sans qu'une ligne de scankit ne bouge.
const (
	labelMessageEn     = "message_en"
	labelRemediationEn = "remediation_en"
)

// localizeFindings substitue le message et la remédiation anglais quand la langue
// résolue est l'anglais, puis RETIRE les deux labels de traduction dans les deux cas.
//
// Le retrait n'est pas cosmétique : `labels` est publié tel quel dans `--format json`,
// dans le SARIF et dans l'assessment scellé. Les y laisser ferait voyager les deux
// langues dans chaque finding : un rapport français porterait sa traduction anglaise,
// et le digest du bundle changerait pour une raison qui ne regarde pas la posture.
// Les labels sont un TRANSPORT ; ils ne sont pas une donnée du rapport.
//
// Une traduction absente laisse le français en place (dégradation propre, cf.
// i18n.Pick) : c'est le cas d'une règle externe chargée par --policy-dir, que Pépin
// ne contrôle pas. Pour les règles du dépôt, l'absence est refusée en CI par
// TestEveryFindingCarriesRemediation, jamais découverte en production.
func localizeFindings(findings []finding.Finding) {
	en := i18n.Current() == i18n.EN
	for i := range findings {
		labels := findings[i].Labels
		if labels == nil {
			continue
		}
		if en {
			findings[i].Message = i18n.Pick(findings[i].Message, labels[labelMessageEn])
			findings[i].Remediation = i18n.Pick(findings[i].Remediation, labels[labelRemediationEn])
		}
		delete(labels, labelMessageEn)
		delete(labels, labelRemediationEn)
	}
}

// cloneFindings copie en profondeur ce que localizeFindings mute (la carte des labels),
// pour qu'un même jeu de findings puisse être rendu dans les deux langues sans que le
// premier passage n'ampute le second. Utilisé par `verify --re-derive`.
func cloneFindings(in []finding.Finding) []finding.Finding {
	out := make([]finding.Finding, len(in))
	copy(out, in)
	for i := range out {
		out[i].Labels = maps.Clone(out[i].Labels)
	}
	return out
}

// scanSource décrit la provenance des données pour le rapport. En live, le
// chemin est vide : on affiche le mode, le profil et la région à la place.
func scanSource(path string) string {
	if !scanLive {
		return path
	}
	src := tr("collecte live", "live collection")
	if scanProfile != "" {
		src += tr(" · profil ", " · profile ") + scanProfile
	}
	if scanRegion != "" {
		src += tr(" · région ", " · region ") + scanRegion
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

// attrsByTypeOf collecte, PAR TYPE, l'ensemble des noms d'attributs présents sur au moins une
// ressource. assess s'en sert pour n'affirmer un « pass » que si l'ATTRIBUT précis qu'un
// contrôle lit a réellement été collecté (pas seulement le type) — sinon une garde de capacité
// silencieuse produit un faux « pass » (ex. compute_instance présent mais sans user_data).
// collected dit si la VALEUR d'un attribut porte une information exploitable.
//
// La présence de la clé ne suffit pas : `iamPolicyStatements` pose toujours
// `statements`, y compris à [] quand le document n'a pas pu être analysé. Quatre
// contrôles IAM critical/high concluaient alors « conforme » sur zéro information.
//
// `false`, `0` et `""` sont en revanche des informations parfaitement valides
// (`encrypted: false` est justement ce qu'un contrôle cherche) : seules les
// collections vides et les valeurs absentes comptent comme non collectées.
func collected(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case []any:
		return len(t) > 0
	case map[string]any:
		return len(t) > 0
	default:
		return true
	}
}

func attrsByTypeOf(input any) map[string]map[string]bool {
	out := map[string]map[string]bool{}
	add := func(typ string, attrs map[string]any) {
		if typ == "" {
			return
		}
		if out[typ] == nil {
			out[typ] = map[string]bool{}
		}
		for k, v := range attrs {
			if !collected(v) {
				continue
			}
			out[typ][k] = true
		}
	}
	// La RÉGION est un champ de la ressource, pas un attribut : on l'enregistre sous la clé
	// vide, celle des contrôles de gouvernance (dont le type visé est ""). Sans ça,
	// governance_resource_region_in_eu — qui lit r.region sur CHAQUE ressource — pouvait
	// s'afficher conforme alors qu'aucune région n'avait été collectée.
	addRegion := func(region string) {
		if region == "" {
			return
		}
		if out[""] == nil {
			out[""] = map[string]bool{}
		}
		out[""]["region"] = true
	}
	m, ok := input.(map[string]any)
	if !ok {
		return out
	}
	switch rs := m["resources"].(type) {
	case []model.Resource:
		for _, r := range rs {
			add(r.Type, r.Attributes)
			addRegion(r.Region)
		}
	case []any:
		for _, it := range rs {
			if rm, ok := it.(map[string]any); ok {
				t, _ := rm["type"].(string)
				attrs, _ := rm["attributes"].(map[string]any)
				add(t, attrs)
				reg, _ := rm["region"].(string)
				addRegion(reg)
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
// nécessaire est réellement collectée. Le verrou lui-même vit dans assess.Verified : la
// documentation de couverture s'y adosse aussi, donc il n'existe qu'une seule définition.
func providerVerified(provider string) map[string]bool {
	out := map[string]bool{}
	for code := range referentiel.All() {
		out[code] = assess.Verified(provider, code)
	}
	return out
}

// targetID identifie la cible du scan (le tenant audité) pour les sujets des résultats et
// l'OSCAL. À défaut d'un identifiant de compte exposé par la source, on compose un identifiant
// stable provider(/région) — jamais vide, pour qu'un résultat « pass » porte un sujet.
func targetID(provider string) string {
	if scanRegion != "" {
		return provider + "/" + scanRegion
	}
	return provider
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
		Tool:      assessment.Component{Name: "pepin", Version: version, Digest: binaryDigest()},
		Ruleset:   assessment.Component{Name: "pepin-config", Digest: configDigest()},
		Target:    assessment.Target{ID: targetID(provider), Provider: provider, Region: scanRegion, Platform: provider},
		Timestamp: scanTimestamp,
		Source:    source,
		Scope:     assessment.Scope{Included: included},
	}
}

// binaryDigest empreinte le BINAIRE qui a produit le résultat : la logique Go de assess/scoring
// n'entre pas dans configDigest, donc deux binaires différents pourraient afficher la même
// provenance sur des verdicts différents. On enregistre la révision VCS (+ marqueur `modified`
// si l'arbre était sale) ; à défaut, le hash du binaire courant. Une version SemVer ne suffit pas.
func binaryDigest() string {
	if bi, ok := debug.ReadBuildInfo(); ok {
		var rev, modified string
		for _, s := range bi.Settings {
			switch s.Key {
			case "vcs.revision":
				rev = s.Value
			case "vcs.modified":
				modified = s.Value
			}
		}
		if rev != "" {
			if modified == "true" {
				return "vcs:" + rev + "+modified"
			}
			return "vcs:" + rev
		}
	}
	if exe, err := os.Executable(); err == nil {
		if b, err := os.ReadFile(exe); err == nil { // #nosec G304 -- chemin du binaire courant.
			sum := sha256.Sum256(b)
			return "sha256:" + hex.EncodeToString(sum[:])
		}
	}
	return ""
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
	for _, dir := range policyDirs {                        // règles externes : le CONTENU seul (pas le chemin, qui
		h.Write([]byte(assess.RulesetDigest(os.DirFS(dir)))) // varie d'une machine à l'autre)
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

// loadInput charge l'entrée à évaluer : collecte live de l'API (--live), plan
// Terraform (--terraform, projeté par le mapper du provider), ou export
// d'inventaire JSON déjà normalisé.
func loadInput(ctx context.Context, p provider.Provider, path string) (any, error) {
	if scanLive {
		cfg := provider.Config{Profile: scanProfile, Region: scanRegion}
		// Une seule map, alimentée par toutes les options : deux affectations successives
		// faisaient perdre la première (--s3-endpoint écrasait --kubeconfig).
		opts := map[string]string{}
		if scanKubeconfig != "" {
			opts["kubeconfig"] = scanKubeconfig
		}
		if scanS3Endpoint != "" {
			opts["s3_endpoint"] = scanS3Endpoint
		}
		if len(opts) > 0 {
			cfg.Options = opts
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
			return nil, fmt.Errorf(tr("le provider %q ne supporte pas l'audit Terraform",
				"provider %q does not support Terraform auditing"), p.Name())
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
		return nil, fmt.Errorf(tr("export JSON invalide : %w", "invalid JSON export: %w"), err)
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
	if hasGovernanceResource(m["resources"]) {
		return m // idempotent : déjà augmenté (ex. rejeu d'un input.json scellé)
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

// hasGovernanceResource indique si l'inventaire contient déjà une ressource `governance_provider`
// (rend withGovernance idempotent, pour un rejeu fidèle d'un input.json déjà augmenté).
func hasGovernanceResource(resources any) bool {
	typeOf := func(r any) string {
		if mr, ok := r.(map[string]any); ok {
			s, _ := mr["type"].(string)
			return s
		}
		return ""
	}
	switch rs := resources.(type) {
	case []model.Resource:
		for _, r := range rs {
			if r.Type == "governance_provider" {
				return true
			}
		}
	case []any:
		for _, r := range rs {
			if typeOf(r) == "governance_provider" {
				return true
			}
		}
	}
	return false
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
	// `evaluated` ne compte QUE des contrôles mesurés sur des ressources collectées : les
	// contrôles de gouvernance passent sur des faits AUTO-DÉCLARÉS (souveraineté) et ne doivent
	// pas empêcher INDÉTERMINÉ sur un inventaire par ailleurs vide (sinon un tenant vidé de ses
	// ressources s'afficherait « conforme »).
	var pass, evaluated int
	for _, r := range asmt.Results {
		gov := strings.HasPrefix(r.Control, "governance_")
		switch r.Status {
		case assessment.Pass:
			pass++
			if !gov {
				evaluated++
			}
		case assessment.Fail:
			if !gov {
				evaluated++
			}
		}
	}
	scope := tr("périmètre évalué", "assessed scope")
	if source == "terraform-plan" {
		scope = tr("périmètre déclaré (plan Terraform, état planifié)",
			"declared scope (Terraform plan, planned state)")
	}
	switch {
	case evaluated == 0:
		return tr(
			"Verdict : INDÉTERMINÉ — aucun contrôle mesuré sur des ressources (le "+scope+" est vide ou non collecté)",
			"Verdict: UNDETERMINED — no control measured on any resource (the "+scope+" is empty or was not collected)")
	case !res.Conforme:
		return tr("Verdict : NON CONFORME", "Verdict: NON-COMPLIANT")
	case res.Medium+res.Low > 0:
		// Pas d'écart critique/haut, MAIS des écarts medium/low subsistent : ne pas laisser lire « conforme ».
		return fmt.Sprintf(tr(
			"Verdict : aucun écart critique/haut, mais %d écart(s) medium/low sur le %s (%d conformes)",
			"Verdict: no critical/high deviation, but %d medium/low deviation(s) on the %s (%d compliant)"),
			res.Medium+res.Low, scope, pass)
	default:
		return fmt.Sprintf(tr(
			"Verdict : conforme sur le %s (aucune non-conformité détectée, %d contrôles conformes)",
			"Verdict: compliant on the %s (no deviation detected, %d compliant controls)"),
			scope, pass)
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
		Tagline:  tr("scanner de posture cloud (sécurité · conformité)", "cloud posture scanner (security · compliance)"),
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

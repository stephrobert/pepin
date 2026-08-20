package assess

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/stephrobert/scankit/assessment"
	screport "github.com/stephrobert/scankit/report"

	"github.com/stephrobert/pepin/internal/i18n"
	"github.com/stephrobert/pepin/internal/model"
)

// BundleFormat identifie le schéma du bundle de preuve, version comprise (`/vN`).
// C'est le signal qu'un consommateur lit dans manifest.json avant de parser le
// reste : un vérificateur qui rencontre une version qu'il ne connaît pas doit
// s'arrêter plutôt que deviner. La forme du bundle (fichiers, rôles, manifest)
// est gelée par cmd/testdata/frozen/bundle.json : la changer sans incrémenter le
// suffixe `/vN` fait échouer TestASurfaceChangeDemandsItsVersionBump.
// v2 : le manifeste porte la version du schéma d'inventaire (l'inventaire est un
// contrat, cf. internal/model.InventoryFormat) et le résumé des DÉROGATIONS
// appliquées ; le bundle gagne l'artefact exemptions.json quand une politique de
// dérogations a été appliquée. Un dossier qui tait ses exemptions n'est pas
// opposable, donc elles sont scellées comme le reste.
// v3 : le manifeste porte la CONFIGURATION appliquée — l'empreinte de la
// politique résolue et, le cas échéant, les assouplissements qui ont fait tomber
// une correspondance normative ; le bundle gagne l'artefact config.json quand un
// fichier de politique a été fourni. Un dossier qui tait sous quelle exigence il
// a été rendu n'est pas opposable : c'est très exactement la porte dérobée vers
// le vert que la configurabilité ouvrirait sans lui.
const BundleFormat = "pepin-assessment-bundle/v3"

// ScopeDisclaimer : avertissement de PORTÉE, obligatoire pour l'opposabilité. pepin évalue la
// configuration d'un TENANT (périmètre commanditaire) ; les référentiels cités (SecNumCloud,
// ISO, CIS) portent, eux, sur le PRESTATAIRE ou sont des correspondances thématiques. Un rapport
// pepin ne prouve donc jamais une qualification/certification, seulement la posture du tenant.
//
// Rendu dans la langue courante : l'avertissement est imprimé à chaque scan et scellé dans le
// manifest du bundle, deux endroits où un lecteur anglophone doit pouvoir le lire.
func ScopeDisclaimer() string { return ScopeDisclaimerIn(i18n.Current()) }

// ScopeDisclaimerIn est ScopeDisclaimer pour une langue explicite : la documentation
// bilingue rend les deux versions dans une seule exécution.
func ScopeDisclaimerIn(l i18n.Lang) string {
	return i18n.TIn(l,
		"Ce rapport évalue la configuration d'un tenant (périmètre commanditaire). "+
			"Les correspondances normatives (SecNumCloud, ISO, CIS) sont indicatives : elles ne constituent "+
			"pas une preuve de qualification/certification, laquelle porte sur le prestataire de service cloud.",
		"This report assesses the configuration of a tenant (customer-side scope). The normative "+
			"mappings (SecNumCloud, ISO, CIS) are indicative: they are not a proof of "+
			"qualification or certification, which applies to the cloud service provider.")
}

// Manifest is the opposable envelope of an evidence bundle: the run provenance, a summary by
// status, and the digest of every artifact. Sealing it (a detached signature over
// checksums.txt) is left to the operator's own identity — see `pepin verify`.
type Manifest struct {
	Format string `json:"format"`
	// InventorySchema : la version du schéma de l'inventaire normalisé que porte
	// input.json. Elle VOYAGE avec le bundle pour qu'un consommateur sache quelle
	// forme il lit, au lieu de la déduire de ce qu'il y trouve.
	InventorySchema string               `json:"inventory_schema"`
	Disclaimer      string               `json:"disclaimer"` // portée prestataire/commanditaire (opposabilité)
	Generated       string               `json:"generated"`  // Run.Timestamp (RFC3339 UTC)
	Tool            assessment.Component `json:"tool"`
	Ruleset         assessment.Component `json:"ruleset"`
	Target          assessment.Target    `json:"target"`
	Source          string               `json:"source"`
	Summary         map[string]int       `json:"summary"` // status -> count
	// Exemptions : l'effet des dérogations sur CE dossier. Absent quand aucune
	// politique n'a été fournie ; jamais silencieux quand il y en a une.
	Exemptions *ExemptionSummary `json:"exemptions,omitempty"`
	// Config : la configuration des contrôles sous laquelle ce dossier a été
	// rendu. Absente quand aucun fichier de politique n'a été fourni — la
	// configuration EFFECTIVE, elle, voyage toujours dans input.json, ce qui rend
	// le dossier re-dérivable dans les deux cas.
	Config    *ConfigSummary `json:"config,omitempty"`
	Artifacts []Artifact     `json:"artifacts"`
}

// ConfigSummary résume, dans le manifeste, sous quels réglages ce dossier a été
// rendu et ce qu'ils ont fait perdre. Le détail est dans config.json, lui aussi
// empreint et signé ; ce résumé est ce qu'un vérificateur lit AVANT d'ouvrir le
// reste — et `relaxed_controls` est la ligne qu'il doit lire en premier.
type ConfigSummary struct {
	// PolicyDigest : empreinte de la configuration RÉSOLUE (défauts compris).
	PolicyDigest string `json:"policy_digest"`
	// RelaxedControls : les contrôles dont une correspondance normative est tombée.
	RelaxedControls []string `json:"relaxed_controls,omitempty"`
	// DroppedReferences : les correspondances abandonnées, `framework:id`.
	DroppedReferences []string `json:"dropped_references,omitempty"`
}

// ExemptionSummary résume, dans le manifeste, ce que les dérogations ont changé.
// Le détail est dans exemptions.json, lui aussi empreint et signé ; ce résumé est
// ce qu'un vérificateur lit AVANT d'ouvrir le reste.
type ExemptionSummary struct {
	// PolicyDigest : empreinte de la politique appliquée (contenu normalisé).
	PolicyDigest string `json:"policy_digest"`
	// Applied / Expired / Orphan : combien de dérogations ont écarté un écart,
	// combien étaient échues, combien visaient un contrôle ou une ressource absents.
	Applied int `json:"applied"`
	Expired int `json:"expired"`
	Orphan  int `json:"orphan"`
}

// BundleExtras porte ce qui s'ajoute au trio input/assessment/OSCAL. Un struct
// plutôt qu'un paramètre de plus : la liste des artefacts d'un bundle grandira
// encore, et une signature à sept positions se lit mal au point de se tromper.
type BundleExtras struct {
	// Exemptions : le document exemptions.json (nil = aucune politique fournie).
	Exemptions []byte
	// ExemptionSummary : son résumé, porté par le manifeste.
	ExemptionSummary *ExemptionSummary
	// Config : le document config.json (nil = aucun fichier de politique fourni).
	Config []byte
	// ConfigSummary : son résumé, porté par le manifeste.
	ConfigSummary *ConfigSummary
}

// Artifact is one file of the bundle with its digest, so any tampering is detectable.
type Artifact struct {
	File   string `json:"file"`
	Role   string `json:"role"`
	SHA256 string `json:"sha256"`
	Bytes  int    `json:"bytes"`
}

// Canonical exposes the deterministic ordering used for sealing, so `verify --re-derive` can
// compare a re-evaluated assessment to the sealed one on the same canonical basis.
func Canonical(a assessment.Assessment) assessment.Assessment { return canonical(a) }

// canonical returns the assessment with its results sorted deterministically, so the same run
// always serializes byte-for-byte the same (a prerequisite for a meaningful digest).
func canonical(a assessment.Assessment) assessment.Assessment {
	res := append([]assessment.Result(nil), a.Results...)
	sort.SliceStable(res, func(i, j int) bool {
		if res[i].Control != res[j].Control {
			return res[i].Control < res[j].Control
		}
		if res[i].Subject != res[j].Subject {
			return res[i].Subject < res[j].Subject
		}
		return res[i].Status < res[j].Status
	})
	a.Results = res
	return a
}

// WriteBundle writes a tamper-evident evidence bundle to dir: the exact evaluated inventory
// (input.json — so the result is RE-DERIVABLE and attributable to a specific inventory), the
// assessment (JSON), its OSCAL assessment-results, a manifest with per-artifact digests, and a
// checksums.txt an operator can sign. `inputJSON` is the inventory the rules were evaluated on
// (nil to omit). Returns the path of checksums.txt (the thing to sign).
func WriteBundle(dir string, a assessment.Assessment, inputJSON []byte, extras BundleExtras) (string, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", fmt.Errorf(i18n.T("création du dossier bundle %s : %w", "creating the bundle directory %s: %w"), dir, err)
	}
	a = canonical(a)

	asmtJSON, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return "", fmt.Errorf(i18n.T("sérialisation de l'assessment : %w", "serializing the assessment: %w"), err)
	}
	var oscalBuf []byte
	{
		f, oerr := os.CreateTemp(dir, ".oscal-*")
		if oerr != nil {
			return "", oerr
		}
		if oerr := screport.OSCAL(f, a); oerr != nil {
			_ = f.Close()
			_ = os.Remove(f.Name())
			return "", fmt.Errorf(i18n.T("rendu OSCAL : %w", "rendering the OSCAL: %w"), oerr)
		}
		_ = f.Close()
		oscalBuf, err = os.ReadFile(f.Name()) // #nosec G304 -- fichier temporaire créé juste au-dessus.
		_ = os.Remove(f.Name())
		if err != nil {
			return "", err
		}
	}

	files := []struct {
		name, role string
		data       []byte
	}{
		{"assessment.json", "assessment", asmtJSON},
		{"assessment-oscal.json", "oscal-assessment-results", oscalBuf},
	}
	if inputJSON != nil {
		// L'inventaire évalué, en tête pour qu'un tiers puisse re-dériver le résultat.
		files = append([]struct {
			name, role string
			data       []byte
		}{{"input.json", "evaluated-input", inputJSON}}, files...)
	}
	if extras.Config != nil {
		// La configuration APPLIQUÉE fait partie de la preuve, au même titre que les
		// dérogations : elle entre au bundle, donc au checksums.txt que l'opérateur
		// signe. Le digest du dossier dépend ainsi des réglages, et un dossier ne
		// peut pas taire l'assouplissement sous lequel il a été rendu.
		files = append(files, struct {
			name, role string
			data       []byte
		}{"config.json", "applied-configuration", extras.Config})
	}
	if extras.Exemptions != nil {
		// La dérogation APPLIQUÉE fait partie de la preuve : elle entre au bundle,
		// donc au checksums.txt que l'opérateur signe. Le digest du dossier dépend
		// ainsi des exemptions, et un dossier ne peut pas les taire sans se trahir.
		files = append(files, struct {
			name, role string
			data       []byte
		}{"exemptions.json", "applied-exemptions", extras.Exemptions})
	}

	manifest := Manifest{
		Format:          BundleFormat,
		InventorySchema: model.InventoryFormat,
		Disclaimer:      ScopeDisclaimer(),
		Generated:       a.Run.Timestamp,
		Tool:            a.Run.Tool,
		Ruleset:         a.Run.Ruleset,
		Target:          a.Run.Target,
		Source:          a.Run.Source,
		Summary:         summaryOf(a),
		Exemptions:      extras.ExemptionSummary,
		Config:          extras.ConfigSummary,
	}
	var checksums string
	for _, f := range files {
		if werr := os.WriteFile(filepath.Join(dir, f.name), f.data, 0o600); werr != nil {
			return "", werr
		}
		sum := sha256.Sum256(f.data)
		hexsum := hex.EncodeToString(sum[:])
		manifest.Artifacts = append(manifest.Artifacts, Artifact{File: f.name, Role: f.role, SHA256: hexsum, Bytes: len(f.data)})
		checksums += fmt.Sprintf("%s  %s\n", hexsum, f.name)
	}

	manJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "", err
	}
	if werr := os.WriteFile(filepath.Join(dir, "manifest.json"), manJSON, 0o600); werr != nil {
		return "", werr
	}
	manSum := sha256.Sum256(manJSON)
	checksums += fmt.Sprintf("%s  manifest.json\n", hex.EncodeToString(manSum[:]))

	checksumsPath := filepath.Join(dir, "checksums.txt")
	if werr := os.WriteFile(checksumsPath, []byte(checksums), 0o600); werr != nil {
		return "", werr
	}
	return checksumsPath, nil
}

// VerifyBundle re-computes the sha256 of every file listed in checksums.txt and reports any
// mismatch or missing file, AND cross-checks that checksums.txt covers EXACTLY the artifacts
// the manifest declares (plus manifest.json itself) — otherwise an artifact could be dropped
// from checksums (no longer verified) or added unlisted without detection. This is the
// third-party integrity check of an evidence bundle.
func VerifyBundle(dir string) error {
	raw, err := os.ReadFile(filepath.Join(dir, "checksums.txt")) // #nosec G304 -- dossier de bundle fourni par l'opérateur.
	if err != nil {
		return fmt.Errorf(i18n.T("lecture de checksums.txt : %w", "reading checksums.txt: %w"), err)
	}
	summed := map[string]string{} // nom -> empreinte attendue (checksums.txt)
	for _, line := range splitLines(string(raw)) {
		if want, name, ok := parseChecksum(line); ok {
			summed[name] = want
		}
	}
	if len(summed) == 0 {
		return fmt.Errorf(i18n.T("aucune empreinte à vérifier dans %s", "no digest to verify in %s"), dir)
	}
	// Recoupement manifest.json ↔ checksums.txt : bijection stricte sur les artefacts.
	manRaw, err := os.ReadFile(filepath.Join(dir, "manifest.json")) // #nosec G304 -- dossier de bundle de l'opérateur.
	if err != nil {
		return fmt.Errorf(i18n.T("lecture de manifest.json : %w", "reading manifest.json: %w"), err)
	}
	var man Manifest
	if err := json.Unmarshal(manRaw, &man); err != nil {
		return fmt.Errorf(i18n.T("manifest.json invalide : %w", "invalid manifest.json: %w"), err)
	}
	declared := map[string]bool{"manifest.json": true} // manifest lui-même listé dans checksums
	for _, a := range man.Artifacts {
		declared[a.File] = true
		if summed[a.File] == "" {
			return fmt.Errorf(i18n.T("artefact %q déclaré au manifeste mais absent de checksums.txt",
				"artifact %q declared in the manifest but missing from checksums.txt"), a.File)
		}
	}
	for name := range summed {
		if !declared[name] {
			return fmt.Errorf(i18n.T("fichier %q listé dans checksums.txt mais non déclaré au manifeste",
				"file %q listed in checksums.txt but not declared in the manifest"), name)
		}
	}
	// Re-calcul des empreintes.
	for name, want := range summed {
		// Le nom vient du manifeste, donc du bundle FOURNI par le tiers audité : c'est
		// une entrée non fiable, et le recoupement manifeste ↔ checksums ne protège pas
		// (l'auteur du bundle contrôle les deux). Sans cette garde, « ../secret » fait
		// de `verify` un oracle d'existence et de contenu sur tout fichier lisible,
		// et « ../../dev/zero » une lecture non bornée. Un artefact est un nom simple.
		if name != filepath.Base(name) || name == "." || name == ".." {
			return fmt.Errorf(i18n.T("nom d'artefact invalide dans le bundle : %q", "invalid artifact name in the bundle: %q"), name)
		}
		data, rerr := os.ReadFile(filepath.Join(dir, name)) // #nosec G304 -- nom contraint à un basename juste au-dessus.
		if rerr != nil {
			return fmt.Errorf("%s : %w", name, rerr)
		}
		got := sha256.Sum256(data)
		if hex.EncodeToString(got[:]) != want {
			return fmt.Errorf(i18n.T("empreinte invalide pour %s (fichier altéré)", "invalid digest for %s (tampered file)"), name)
		}
	}
	return nil
}

func summaryOf(a assessment.Assessment) map[string]int {
	out := map[string]int{}
	for _, r := range a.Results {
		out[string(r.Status)]++
	}
	return out
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

func parseChecksum(line string) (sum, name string, ok bool) {
	// format: "<hex>  <name>"
	i := 0
	for i < len(line) && line[i] != ' ' {
		i++
	}
	if i == 0 || i >= len(line) {
		return "", "", false
	}
	sum = line[:i]
	for i < len(line) && line[i] == ' ' {
		i++
	}
	name = line[i:]
	return sum, name, sum != "" && name != ""
}

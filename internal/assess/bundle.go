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
)

// bundleFormat identifies the tamper-evident evidence bundle schema.
const bundleFormat = "pepin-assessment-bundle/v1"

// ScopeDisclaimer : avertissement de PORTÉE, obligatoire pour l'opposabilité. pepin évalue la
// configuration d'un TENANT (périmètre commanditaire) ; les référentiels cités (SecNumCloud,
// ISO, CIS) portent, eux, sur le PRESTATAIRE ou sont des correspondances thématiques. Un rapport
// pepin ne prouve donc jamais une qualification/certification, seulement la posture du tenant.
const ScopeDisclaimer = "Ce rapport évalue la configuration d'un tenant (périmètre commanditaire). " +
	"Les correspondances normatives (SecNumCloud, ISO, CIS) sont indicatives : elles ne constituent " +
	"pas une preuve de qualification/certification, laquelle porte sur le prestataire de service cloud."

// Manifest is the opposable envelope of an evidence bundle: the run provenance, a summary by
// status, and the digest of every artifact. Sealing it (a detached signature over
// checksums.txt) is left to the operator's own identity — see `pepin verify`.
type Manifest struct {
	Format     string               `json:"format"`
	Disclaimer string               `json:"disclaimer"` // portée prestataire/commanditaire (opposabilité)
	Generated  string               `json:"generated"`  // Run.Timestamp (RFC3339 UTC)
	Tool       assessment.Component `json:"tool"`
	Ruleset    assessment.Component `json:"ruleset"`
	Target     assessment.Target    `json:"target"`
	Source     string               `json:"source"`
	Summary    map[string]int       `json:"summary"` // status -> count
	Artifacts  []Artifact           `json:"artifacts"`
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
func WriteBundle(dir string, a assessment.Assessment, inputJSON []byte) (string, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", fmt.Errorf("création du dossier bundle %s : %w", dir, err)
	}
	a = canonical(a)

	asmtJSON, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return "", fmt.Errorf("sérialisation de l'assessment : %w", err)
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
			return "", fmt.Errorf("rendu OSCAL : %w", oerr)
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

	manifest := Manifest{
		Format:     bundleFormat,
		Disclaimer: ScopeDisclaimer,
		Generated:  a.Run.Timestamp,
		Tool:       a.Run.Tool,
		Ruleset:    a.Run.Ruleset,
		Target:     a.Run.Target,
		Source:     a.Run.Source,
		Summary:    summaryOf(a),
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
		return fmt.Errorf("lecture de checksums.txt : %w", err)
	}
	summed := map[string]string{} // nom -> empreinte attendue (checksums.txt)
	for _, line := range splitLines(string(raw)) {
		if want, name, ok := parseChecksum(line); ok {
			summed[name] = want
		}
	}
	if len(summed) == 0 {
		return fmt.Errorf("aucune empreinte à vérifier dans %s", dir)
	}
	// Recoupement manifest.json ↔ checksums.txt : bijection stricte sur les artefacts.
	manRaw, err := os.ReadFile(filepath.Join(dir, "manifest.json")) // #nosec G304 -- dossier de bundle de l'opérateur.
	if err != nil {
		return fmt.Errorf("lecture de manifest.json : %w", err)
	}
	var man Manifest
	if err := json.Unmarshal(manRaw, &man); err != nil {
		return fmt.Errorf("manifest.json invalide : %w", err)
	}
	declared := map[string]bool{"manifest.json": true} // manifest lui-même listé dans checksums
	for _, a := range man.Artifacts {
		declared[a.File] = true
		if summed[a.File] == "" {
			return fmt.Errorf("artefact %q déclaré au manifeste mais absent de checksums.txt", a.File)
		}
	}
	for name := range summed {
		if !declared[name] {
			return fmt.Errorf("fichier %q listé dans checksums.txt mais non déclaré au manifeste", name)
		}
	}
	// Re-calcul des empreintes.
	for name, want := range summed {
		data, rerr := os.ReadFile(filepath.Join(dir, name)) // #nosec G304 -- nom issu du manifeste du bundle.
		if rerr != nil {
			return fmt.Errorf("%s : %w", name, rerr)
		}
		got := sha256.Sum256(data)
		if hex.EncodeToString(got[:]) != want {
			return fmt.Errorf("empreinte invalide pour %s (fichier altéré)", name)
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

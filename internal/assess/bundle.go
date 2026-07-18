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

// Manifest is the opposable envelope of an evidence bundle: the run provenance, a summary by
// status, and the digest of every artifact. Sealing it (a detached signature over
// checksums.txt) is left to the operator's own identity — see `pepin verify`.
type Manifest struct {
	Format    string               `json:"format"`
	Generated string               `json:"generated"` // Run.Timestamp (RFC3339 UTC)
	Tool      assessment.Component `json:"tool"`
	Ruleset   assessment.Component `json:"ruleset"`
	Target    assessment.Target    `json:"target"`
	Source    string               `json:"source"`
	Summary   map[string]int       `json:"summary"` // status -> count
	Artifacts []Artifact           `json:"artifacts"`
}

// Artifact is one file of the bundle with its digest, so any tampering is detectable.
type Artifact struct {
	File   string `json:"file"`
	Role   string `json:"role"`
	SHA256 string `json:"sha256"`
	Bytes  int    `json:"bytes"`
}

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

// WriteBundle writes a tamper-evident evidence bundle to dir: the assessment (JSON), its OSCAL
// assessment-results, a manifest with per-artifact digests, and a checksums.txt an operator can
// sign. Returns the path of checksums.txt (the thing to sign).
func WriteBundle(dir string, a assessment.Assessment) (string, error) {
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

	manifest := Manifest{
		Format:    bundleFormat,
		Generated: a.Run.Timestamp,
		Tool:      a.Run.Tool,
		Ruleset:   a.Run.Ruleset,
		Target:    a.Run.Target,
		Source:    a.Run.Source,
		Summary:   summaryOf(a),
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
// mismatch or missing file — the third-party integrity check of an evidence bundle.
func VerifyBundle(dir string) error {
	raw, err := os.ReadFile(filepath.Join(dir, "checksums.txt")) // #nosec G304 -- dossier de bundle fourni par l'opérateur.
	if err != nil {
		return fmt.Errorf("lecture de checksums.txt : %w", err)
	}
	var checked int
	for _, line := range splitLines(string(raw)) {
		want, name, ok := parseChecksum(line)
		if !ok {
			continue
		}
		data, rerr := os.ReadFile(filepath.Join(dir, name)) // #nosec G304 -- nom issu du manifeste du bundle.
		if rerr != nil {
			return fmt.Errorf("%s : %w", name, rerr)
		}
		got := sha256.Sum256(data)
		if hex.EncodeToString(got[:]) != want {
			return fmt.Errorf("empreinte invalide pour %s (fichier altéré)", name)
		}
		checked++
	}
	if checked == 0 {
		return fmt.Errorf("aucune empreinte à vérifier dans %s", dir)
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

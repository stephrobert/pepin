package quality

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/stephrobert/pepin/internal/canary"
)

// QualificationDir contient les relevés anonymisés de qualifications authentifiées.
const QualificationDir = "references/qualification/evidence"

// LiveObservation est un verdict effectivement comparé sur un tenant authentifié.
type LiveObservation struct {
	Control string `json:"control"`
	Verdict string `json:"verdict"`
}

// LiveProof associe un verdict à sa qualification et à sa fraîcheur effective.
type LiveProof struct {
	Qualification Qualification
	Verdict       string
	Fresh         bool
}

// Qualification est le relevé public, sans identifiant de compte ni de ressource.
type Qualification struct {
	Provider                  string            `json:"provider"`
	Source                    string            `json:"source"`
	RecordedAt                string            `json:"recorded_at"`
	ToolVersion               string            `json:"tool_version"`
	ToolDigest                string            `json:"tool_digest"`
	ReportSHA256              string            `json:"report_sha256"`
	AssessmentSHA256          string            `json:"assessment_sha256"`
	ReevaluatedExpectedSHA256 string            `json:"reevaluated_expected_sha256"`
	Observations              []LiveObservation `json:"observations"`
	RecordFile                string            `json:"record_file,omitempty"`
}

var (
	safeName       = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	safeVersion    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+-]{0,80}$`)
	safeToolDigest = regexp.MustCompile(`^(vcs:[a-f0-9]{7,64}(\+modified)?|sha256:[a-f0-9]{64})$`)
)

func validDigest(s string) bool {
	if len(s) != 64 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil && s == strings.ToLower(s)
}

func (q Qualification) validate() error {
	if !safeName.MatchString(q.Provider) || q.Source != "live" {
		return fmt.Errorf("fournisseur ou source live invalide")
	}
	if _, err := q.ObservedAt(); err != nil {
		return err
	}
	if !safeVersion.MatchString(q.ToolVersion) || !safeToolDigest.MatchString(q.ToolDigest) {
		return fmt.Errorf("version ou empreinte du binaire invalide")
	}
	if !validDigest(q.ReportSHA256) || !validDigest(q.AssessmentSHA256) || !validDigest(q.ReevaluatedExpectedSHA256) {
		return fmt.Errorf("empreinte SHA-256 de preuve invalide")
	}
	if len(q.Observations) == 0 {
		return fmt.Errorf("qualification sans verdict vérifié")
	}
	seen := map[LiveObservation]bool{}
	for _, o := range q.Observations {
		if !safeName.MatchString(o.Control) || (o.Verdict != "fail" && o.Verdict != "pass") {
			return fmt.Errorf("observation de contrôle invalide")
		}
		if seen[o] {
			return fmt.Errorf("observation dupliquée : %s %s", o.Control, o.Verdict)
		}
		seen[o] = true
	}
	return nil
}

// ObservedAt rend l'horodatage UTC exact de la collecte.
func (q Qualification) ObservedAt() (time.Time, error) {
	recorded, err := time.Parse(time.RFC3339Nano, q.RecordedAt)
	if err != nil || !strings.HasSuffix(q.RecordedAt, "Z") {
		return time.Time{}, fmt.Errorf("horodatage UTC de qualification invalide : %q", q.RecordedAt)
	}
	return recorded, nil
}

// ExpiresAt rend la limite inclusive de validité d'une qualification.
func (q Qualification) ExpiresAt() (time.Time, error) {
	recorded, err := q.ObservedAt()
	if err != nil {
		return time.Time{}, err
	}
	return recorded.Add(canary.MaxAge), nil
}

// FreshAt indique si une qualification est datée et dans la fenêtre du canari.
func (q Qualification) FreshAt(now time.Time) bool {
	expires, err := q.ExpiresAt()
	if err != nil {
		return false
	}
	return !now.Before(expires.Add(-canary.MaxAge)) && !now.After(expires)
}

// LoadQualifications lit les relevés publics, en refusant tout champ non prévu.
func LoadQualifications(root string) ([]Qualification, error) {
	dir := filepath.Join(root, QualificationDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("lecture des qualifications %s : %w", dir, err)
	}
	var records []Qualification
	recordedAt := map[string]time.Time{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			return nil, fmt.Errorf("qualification %s : seuls les relevés JSON sont admis", entry.Name())
		}
		path := filepath.Join(dir, entry.Name())
		raw, err := os.ReadFile(path) // #nosec G304 -- fichier du dossier de relevés versionnés.
		if err != nil {
			return nil, fmt.Errorf("lecture de %s : %w", path, err)
		}
		var q Qualification
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&q); err != nil {
			return nil, fmt.Errorf("qualification %s illisible : %w", path, err)
		}
		if err := dec.Decode(&struct{}{}); err != io.EOF {
			return nil, fmt.Errorf("qualification %s : contenu supplémentaire après le document", path)
		}
		if q.RecordFile != "" {
			return nil, fmt.Errorf("qualification %s : record_file est réservé au générateur", path)
		}
		if err := q.validate(); err != nil {
			return nil, fmt.Errorf("qualification %s : %w", path, err)
		}
		q.RecordFile = filepath.ToSlash(filepath.Join(QualificationDir, entry.Name()))
		recorded, err := q.ObservedAt()
		if err != nil {
			return nil, fmt.Errorf("qualification %s : %w", path, err)
		}
		recordedAt[q.RecordFile] = recorded
		records = append(records, q)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("aucune qualification authentifiée dans %s", dir)
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i].Provider != records[j].Provider {
			return records[i].Provider < records[j].Provider
		}
		ti := recordedAt[records[i].RecordFile]
		tj := recordedAt[records[j].RecordFile]
		if ti.Equal(tj) {
			return records[i].RecordFile < records[j].RecordFile
		}
		return ti.Before(tj)
	})
	return records, nil
}

// LiveProofs rend les observations qui correspondent à un verdict requis sur le
// chemin live de ce contrôle. Les preuves périmées restent visibles mais marquées.
func (s Snapshot) LiveProofs(control string, now time.Time) []LiveProof {
	var proofs []LiveProof
	for _, q := range s.Qualifications {
		for _, o := range q.Observations {
			if o.Control != control {
				continue
			}
			for _, path := range s.ByControl[control].Paths {
				if path.Source != "live" || path.Provider != q.Provider {
					continue
				}
				for _, verdict := range path.Required {
					if verdict == o.Verdict {
						proofs = append(proofs, LiveProof{Qualification: q, Verdict: verdict, Fresh: q.FreshAt(now)})
					}
				}
			}
		}
	}
	return proofs
}

// At calcule les chemins live dont au moins un verdict attendu a une preuve fraîche.
func (s Snapshot) At(now time.Time) Snapshot {
	type pathKey struct {
		control, provider string
	}
	type verdictKey struct {
		pathKey
		verdict string
	}
	paths := map[pathKey]bool{}
	verdicts := map[verdictKey]bool{}
	for code := range s.ByControl {
		for _, proof := range s.LiveProofs(code, now) {
			if proof.Fresh {
				path := pathKey{code, proof.Qualification.Provider}
				paths[path] = true
				verdicts[verdictKey{path, proof.Verdict}] = true
			}
		}
	}
	s.LiveValidated = len(paths)
	s.LiveVerdicts = len(verdicts)
	return s
}

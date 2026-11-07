package quality_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stephrobert/pepin/internal/canary"
	"github.com/stephrobert/pepin/internal/quality"
	"github.com/stephrobert/pepin/internal/veracity"
)

func qualification(provider, date string, observations ...quality.LiveObservation) quality.Qualification {
	return quality.Qualification{
		Provider: provider, Source: "live", RecordedAt: date + "T00:00:00Z",
		ToolVersion: "0.1.0-dev", ToolDigest: "vcs:abcdef123456+modified",
		ReportSHA256: strings.Repeat("a", 64), AssessmentSHA256: strings.Repeat("b", 64),
		ReevaluatedExpectedSHA256: strings.Repeat("c", 64), Observations: observations,
	}
}

func TestAuthenticatedEvidenceCountsOnlyTheObservedPathAndVerdict(t *testing.T) {
	record := qualification("scaleway", "2026-09-11",
		quality.LiveObservation{Control: "alpha", Verdict: "fail"},
		quality.LiveObservation{Control: "alpha", Verdict: "pass"},
		quality.LiveObservation{Control: "gamma", Verdict: "fail"},
		quality.LiveObservation{Control: "delta", Verdict: "fail"})
	stale := qualification("exoscale", "2026-01-01",
		quality.LiveObservation{Control: "alpha", Verdict: "pass"})
	s, err := quality.Compute(quality.Inputs{
		Cells: []veracity.Cell{
			{Control: "alpha", Provider: "scaleway", Source: "live", Status: veracity.StatusSupported},
			{Control: "beta", Provider: "scaleway", Source: "live", Status: veracity.StatusSupported},
			{Control: "alpha", Provider: "exoscale", Source: "live", Status: veracity.StatusSupported},
			{Control: "gamma", Provider: "scaleway", Source: "live", Status: veracity.StatusPartial},
			{Control: "delta", Provider: "scaleway", Source: "terraform", Status: veracity.StatusSupported},
		},
		Records:        []canary.Record{{Provider: "exoscale", Authenticated: true}},
		Qualifications: []quality.Qualification{record, record, stale},
	})
	if err != nil {
		t.Fatal(err)
	}
	if s.LivePaths != 4 || s.LiveValidated != 1 || s.LiveVerdicts != 2 {
		t.Fatalf("chemins live = %d, validés = %d, verdicts = %d ; attendus 4, 1 et 2",
			s.LivePaths, s.LiveValidated, s.LiveVerdicts)
	}
	proofs := s.LiveProofs("alpha", time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC))
	if len(proofs) != 5 || !proofs[0].Fresh || proofs[0].Verdict != "fail" || proofs[4].Fresh {
		t.Errorf("preuves fraîches et historiques inattendues : %+v", proofs)
	}
	if got := s.LiveProofs("beta", time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)); len(got) != 0 {
		t.Errorf("un canari ne constitue pas de preuve du contrôle beta : %+v", got)
	}
	afterExpiry := time.Date(2026, 12, 11, 0, 0, 0, 0, time.UTC)
	if expired := s.At(afterExpiry); expired.LiveValidated != 0 || expired.LiveVerdicts != 0 {
		t.Errorf("une preuve périmée compte encore : %d chemins, %d verdicts",
			expired.LiveValidated, expired.LiveVerdicts)
	}
}

func TestQualificationFreshnessHasExactBoundaries(t *testing.T) {
	q := qualification("scaleway", "2026-09-11",
		quality.LiveObservation{Control: "alpha", Verdict: "fail"})
	day := time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		name string
		now  time.Time
		want bool
	}{
		{"future record", day.Add(-time.Second), false},
		{"recorded", day, true},
		{"exact expiry", day.Add(canary.MaxAge), true},
		{"expired", day.Add(canary.MaxAge + time.Second), false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := q.FreshAt(tt.now); got != tt.want {
				t.Errorf("FreshAt(%s) = %t; want %t", tt.now, got, tt.want)
			}
		})
	}
	q.RecordedAt = "not-a-date"
	if q.FreshAt(day) {
		t.Error("une date illisible ne prouve pas la fraîcheur")
	}
	q.RecordedAt = "2026-09-11T12:15:00Z"
	if q.FreshAt(day.Add(12*time.Hour + 14*time.Minute)) {
		t.Error("une preuve ne doit pas compter avant l'heure effective du scan")
	}
}

func TestQualificationLoaderRejectsPrivateOrInvalidFields(t *testing.T) {
	q := qualification("scaleway", "2026-09-11",
		quality.LiveObservation{Control: "alpha", Verdict: "fail"})
	raw, err := json.Marshal(q)
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name    string
		content string
		wantErr bool
	}{
		{"valid", string(raw), false},
		{"unexpected account", strings.Replace(string(raw), `"source":"live"`, `"source":"live","account":"private"`, 1), true},
		{"unverified verdict", strings.Replace(string(raw), `"verdict":"fail"`, `"verdict":"not-evaluated"`, 1), true},
		{"duplicate document", string(raw) + string(raw), true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			dir := filepath.Join(root, quality.QualificationDir)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "record.json"), []byte(tt.content), 0o600); err != nil {
				t.Fatal(err)
			}
			got, err := quality.LoadQualifications(root)
			if (err != nil) != tt.wantErr {
				t.Fatalf("LoadQualifications() error = %v; want error %t", err, tt.wantErr)
			}
			if !tt.wantErr && (len(got) != 1 || got[0].RecordFile != quality.QualificationDir+"/record.json") {
				t.Fatalf("qualification chargée incorrectement : %+v", got)
			}
		})
	}
}

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stephrobert/pepin/internal/quality"
)

func TestQualificationPreflightRequiresFreshEvidencePerRecordedProvider(t *testing.T) {
	now := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		name    string
		records [][2]string
		wantErr bool
	}{
		{"fresh", [][2]string{{"scaleway", "2026-09-11"}}, false},
		{"stale", [][2]string{{"scaleway", "2026-01-01"}}, true},
		{"future", [][2]string{{"scaleway", "2026-11-07"}}, true},
		{"old history with fresh replacement", [][2]string{{"scaleway", "2026-01-01"}, {"scaleway", "2026-09-11"}}, false},
		{"one provider stale", [][2]string{{"scaleway", "2026-09-11"}, {"exoscale", "2026-01-01"}}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			dir := filepath.Join(root, quality.QualificationDir)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			for i, r := range tt.records {
				q := quality.Qualification{
					Provider: r[0], Source: "live", RecordedAt: r[1] + "T00:00:00Z",
					ToolVersion: "0.1.0-dev", ToolDigest: "vcs:abcdef123456+modified",
					ReportSHA256: strings.Repeat("a", 64), AssessmentSHA256: strings.Repeat("b", 64),
					ReevaluatedExpectedSHA256: strings.Repeat("c", 64),
					Observations:              []quality.LiveObservation{{Control: "alpha", Verdict: "fail"}},
				}
				raw, err := json.Marshal(q)
				if err != nil {
					t.Fatal(err)
				}
				name := filepath.Join(dir, fmt.Sprintf("%s-%d.json", r[0], i))
				if err := os.WriteFile(name, raw, 0o600); err != nil {
					t.Fatal(err)
				}
			}

			err := checkQualifications(root, now)
			if (err != nil) != tt.wantErr {
				t.Fatalf("checkQualifications() error = %v; want error %t", err, tt.wantErr)
			}
		})
	}
}

func TestQualificationPreflightRefusesMissingRecords(t *testing.T) {
	if err := checkQualifications(t.TempDir(), time.Now().UTC()); err == nil {
		t.Fatal("aucun relevé ne doit rendre la porte verte")
	}
}

func TestReleasePreflightCallsTheQualificationGate(t *testing.T) {
	raw, err := os.ReadFile("preflight.sh")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "go run ./tools/release/qualification_preflight.go") {
		t.Fatal("le préflight de release ne vérifie plus les qualifications")
	}
}

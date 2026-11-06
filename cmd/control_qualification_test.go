package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/stephrobert/pepin/internal/i18n"
	"github.com/stephrobert/pepin/internal/quality"
)

func TestExplainProofShowsOnlyTheSelectedControlAndProvider(t *testing.T) {
	old := i18n.Current()
	i18n.Set(i18n.EN)
	t.Cleanup(func() { i18n.Set(old) })

	snap := quality.Snapshot{
		ByControl: map[string]quality.ControlProof{
			"alpha": {Paths: []quality.PathProof{
				{Provider: "scaleway", Source: "live", Required: []string{"fail", "pass"}},
				{Provider: "exoscale", Source: "live", Required: []string{"fail", "pass"}},
			}},
			"beta": {Paths: []quality.PathProof{
				{Provider: "scaleway", Source: "live", Required: []string{"fail", "pass"}},
			}},
		},
		Qualifications: []quality.Qualification{
			{Provider: "scaleway", RecordedAt: "2026-09-11T12:15:00Z",
				ToolVersion: "0.1.0-dev", ToolDigest: "vcs:abcdef123456+modified",
				RecordFile:   "references/qualification/evidence/scaleway.json",
				Observations: []quality.LiveObservation{{Control: "alpha", Verdict: "fail"}}},
			{Provider: "exoscale", RecordedAt: "2026-09-12T12:15:00Z",
				ToolVersion: "0.1.0-dev", ToolDigest: "vcs:bbbbbb123456+modified",
				RecordFile:   "references/qualification/evidence/exoscale.json",
				Observations: []quality.LiveObservation{{Control: "alpha", Verdict: "pass"}}},
		},
		Canary: []quality.CanaryProvider{{Provider: "scaleway"}, {Provider: "exoscale"}},
	}
	now := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	var out strings.Builder
	if err := explainProof(&out, "alpha", []string{"scaleway"}, snap, now); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{"scaleway", "verdict fail", "2026-12-10T12:15:00Z",
		"current", "vcs:abcdef123456+modified"} {
		if !strings.Contains(got, want) {
			t.Errorf("l'explication ne contient pas %q :\n%s", want, got)
		}
	}
	if strings.Contains(got, "exoscale") {
		t.Errorf("l'explication révèle une preuve d'un autre fournisseur :\n%s", got)
	}

	out.Reset()
	if err := explainProof(&out, "beta", []string{"scaleway"}, snap, now); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "no authenticated proof") {
		t.Errorf("la preuve d'un autre contrôle ne doit pas compter :\n%s", out.String())
	}

	out.Reset()
	if err := explainProof(&out, "alpha", []string{"scaleway"}, snap, now.Add(100*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "not current") {
		t.Errorf("une preuve périmée doit être affichée comme telle :\n%s", out.String())
	}
}

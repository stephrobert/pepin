package assess

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stephrobert/pepin/internal/model"
	"github.com/stephrobert/scankit/assessment"
)

func writeDiffBundle(t *testing.T, dir, targetID, toolVersion, rulesetDigest string, results []assessment.Result, resources []model.Resource) {
	t.Helper()
	report := assessment.Assessment{
		Run: assessment.Run{
			Tool:      assessment.Component{Name: "pepin", Version: toolVersion, Digest: "sha256:tool-" + toolVersion},
			Ruleset:   assessment.Component{Name: "commonrules", Version: "2026-11-06", Digest: rulesetDigest},
			Target:    assessment.Target{ID: targetID, Provider: "outscale", Region: "eu-west-2"},
			Timestamp: "2026-11-06T12:00:00Z",
			Source:    "terraform-plan",
		},
		Results: results,
	}
	input, err := json.Marshal(model.Posture{Provider: "outscale", Resources: resources})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := WriteBundle(dir, report, input, BundleExtras{}); err != nil {
		t.Fatalf("WriteBundle: %v", err)
	}
}

func TestDiffBundlesSeparatesFixesRegressionsAndVisibility(t *testing.T) {
	before := filepath.Join(t.TempDir(), "before")
	after := filepath.Join(t.TempDir(), "after")
	writeDiffBundle(t, before, "account-1", "0.4.0", "sha256:rules-old", []assessment.Result{
		{Control: "fixed", Subject: "vm-1", Status: assessment.Fail},
		{Control: "regressed", Subject: "vm-1", Status: assessment.Pass},
		{Control: "visibility-lost", Subject: "vm-1", Status: assessment.Pass},
		{Control: "visibility-restored", Subject: "vm-1", Status: assessment.NotEvaluated},
		{Control: "status-changed", Subject: "vm-1", Status: assessment.NotApplicable},
		{Control: "unchanged", Subject: "vm-1", Status: assessment.Pass},
		{Control: "removed", Subject: "vm-1", Status: assessment.Fail},
	}, []model.Resource{
		{Provider: "outscale", Type: "compute_instance", ID: "vm-1"},
		{Provider: "outscale", Type: "compute_instance", ID: "vm-2"},
	})
	writeDiffBundle(t, after, "account-1", "0.5.0", "sha256:rules-new", []assessment.Result{
		{Control: "fixed", Subject: "vm-1", Status: assessment.Pass},
		{Control: "regressed", Subject: "vm-1", Status: assessment.Fail},
		{Control: "visibility-lost", Subject: "vm-1", Status: assessment.NotEvaluated},
		{Control: "visibility-restored", Subject: "vm-1", Status: assessment.Fail},
		{Control: "status-changed", Subject: "vm-1", Status: assessment.Fail},
		{Control: "unchanged", Subject: "vm-1", Status: assessment.Pass},
		{Control: "added", Subject: "vm-1", Status: assessment.Fail},
	}, []model.Resource{
		{Provider: "outscale", Type: "compute_instance", ID: "vm-1"},
		{Provider: "outscale", Type: "compute_instance", ID: "vm-3"},
	})

	got, err := DiffBundles(before, after)
	if err != nil {
		t.Fatalf("DiffBundles: %v", err)
	}
	want := DiffSummary{
		Fixed:              1,
		Regressed:          1,
		VisibilityLost:     1,
		VisibilityRestored: 1,
		StatusChanged:      1,
		ResultsAdded:       1,
		ResultsRemoved:     1,
		ResourcesAdded:     1,
		ResourcesRemoved:   1,
	}
	if got.Summary != want {
		t.Fatalf("summary = %+v, want %+v", got.Summary, want)
	}
	if !got.ResourcesCompared {
		t.Fatal("resource inventory comparison was not reported")
	}
	if len(got.VersionChanges) != 2 || got.VersionChanges[0].Component != "tool" ||
		got.VersionChanges[1].Component != "ruleset" {
		t.Fatalf("version changes = %+v, want tool and ruleset drift", got.VersionChanges)
	}
	wantKinds := []string{"added", "fixed", "regressed", "removed", "status-changed", "visibility-lost", "visibility-restored"}
	if len(got.ResultChanges) != len(wantKinds) {
		t.Fatalf("result changes = %+v", got.ResultChanges)
	}
	for i, wantKind := range wantKinds {
		if got.ResultChanges[i].Kind != wantKind {
			t.Fatalf("result change %d kind = %q, want %q", i, got.ResultChanges[i].Kind, wantKind)
		}
	}
	if len(got.ResourceChanges) != 2 || got.ResourceChanges[0].Kind != "removed" ||
		got.ResourceChanges[1].Kind != "added" {
		t.Fatalf("resource changes = %+v, want sorted removals and additions", got.ResourceChanges)
	}
}

func TestDiffBundlesRejectsDifferentTargets(t *testing.T) {
	before := filepath.Join(t.TempDir(), "before")
	after := filepath.Join(t.TempDir(), "after")
	writeDiffBundle(t, before, "account-1", "0.5.0", "sha256:rules", nil, nil)
	writeDiffBundle(t, after, "account-2", "0.5.0", "sha256:rules", nil, nil)
	if _, err := DiffBundles(before, after); err == nil {
		t.Fatal("DiffBundles accepted bundles for different targets")
	}
}

func TestDiffBundlesRejectsTamperedBundle(t *testing.T) {
	before := filepath.Join(t.TempDir(), "before")
	after := filepath.Join(t.TempDir(), "after")
	writeDiffBundle(t, before, "account-1", "0.5.0", "sha256:rules", nil, nil)
	writeDiffBundle(t, after, "account-1", "0.5.0", "sha256:rules", nil, nil)

	assessmentPath := filepath.Join(after, "assessment.json")
	file, err := os.OpenFile(assessmentPath, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString(" "); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := DiffBundles(before, after); err == nil {
		t.Fatal("DiffBundles accepted a bundle with a modified assessment")
	}
}

package assess

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stephrobert/scankit/assessment"
)

func bundleAssessment() assessment.Assessment {
	return assessment.Assessment{
		Run: assessment.Run{
			Tool:      assessment.Component{Name: "pepin", Version: "0.2.0"},
			Ruleset:   assessment.Component{Name: "commonrules", Digest: "sha256:abc"},
			Target:    assessment.Target{ID: "acct-1", Provider: "outscale"},
			Timestamp: "2026-07-18T20:00:00Z",
			Source:    "export",
		},
		Results: []assessment.Result{
			{Control: "network_sg_open_22", Status: assessment.Fail, Severity: "critical", Subject: "sg-1"},
			{Control: "objectstorage_bucket_public", Status: assessment.Pass, Subject: "acct-1"},
		},
	}
}

func TestWriteAndVerifyBundle(t *testing.T) {
	dir := t.TempDir()
	cs, err := WriteBundle(dir, bundleAssessment(), []byte(`{"resources":[]}`), BundleExtras{})
	if err != nil {
		t.Fatalf("WriteBundle: %v", err)
	}
	if _, err := os.Stat(cs); err != nil {
		t.Fatalf("checksums.txt missing: %v", err)
	}
	for _, f := range []string{"assessment.json", "assessment-oscal.json", "manifest.json", "checksums.txt"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("bundle artifact %s missing: %v", f, err)
		}
	}
	if err := VerifyBundle(dir); err != nil {
		t.Fatalf("a fresh bundle must verify: %v", err)
	}
}

func TestVerifyDetectsTampering(t *testing.T) {
	dir := t.TempDir()
	if _, err := WriteBundle(dir, bundleAssessment(), []byte(`{"resources":[]}`), BundleExtras{}); err != nil {
		t.Fatal(err)
	}
	// Tamper with the assessment after sealing.
	p := filepath.Join(dir, "assessment.json")
	data, _ := os.ReadFile(p) // #nosec G304 -- test fixture path.
	if err := os.WriteFile(p, append(data, '!'), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := VerifyBundle(dir); err == nil {
		t.Error("VerifyBundle must fail on a tampered artifact")
	}
}

func TestVerifyManifestChecksumBijection(t *testing.T) {
	// Retirer un artefact de checksums.txt (il ne serait plus vérifié) : rejeté via le manifeste.
	dir := t.TempDir()
	if _, err := WriteBundle(dir, bundleAssessment(), []byte(`{"resources":[]}`), BundleExtras{}); err != nil {
		t.Fatal(err)
	}
	cs := filepath.Join(dir, "checksums.txt")
	raw, _ := os.ReadFile(cs) // #nosec G304 -- test fixture.
	var kept []byte
	for _, ln := range splitLines(string(raw)) {
		if !strings.Contains(ln, "assessment-oscal.json") {
			kept = append(kept, []byte(ln+"\n")...)
		}
	}
	if err := os.WriteFile(cs, kept, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := VerifyBundle(dir); err == nil {
		t.Error("VerifyBundle must reject a manifest artifact missing from checksums.txt")
	}
}

func TestBundleDeterministic(t *testing.T) {
	d1, d2 := t.TempDir(), t.TempDir()
	if _, err := WriteBundle(d1, bundleAssessment(), []byte(`{"resources":[]}`), BundleExtras{}); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteBundle(d2, bundleAssessment(), []byte(`{"resources":[]}`), BundleExtras{}); err != nil {
		t.Fatal(err)
	}
	// Same run (same timestamp) → identical assessment digest in checksums.txt.
	c1, _ := os.ReadFile(filepath.Join(d1, "checksums.txt")) // #nosec G304 -- test fixture path.
	c2, _ := os.ReadFile(filepath.Join(d2, "checksums.txt")) // #nosec G304 -- test fixture path.
	if string(c1) != string(c2) {
		t.Error("bundle checksums must be deterministic for the same assessment")
	}
}

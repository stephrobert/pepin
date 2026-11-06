package docgen

import (
	"strings"
	"testing"
)

func TestNormalizeOSCALToolDigestDoesNotFreezeBuildArtifact(t *testing.T) {
	document := `{
  "name": "tool-digest",
  "value": "sha256:binary",
  "other-digest": "sha256:rules"
}`
	got := normalizeOSCALToolDigest(document)
	if !strings.Contains(got, `"value": "<binary-digest>"`) {
		t.Fatalf("tool digest was not normalized: %s", got)
	}
	if !strings.Contains(got, `"other-digest": "sha256:rules"`) {
		t.Fatalf("unrelated digest was modified: %s", got)
	}
}

func TestNormalizeBundleTextDoesNotFreezeToolBinaryDigest(t *testing.T) {
	for _, digest := range []string{
		"sha256:" + strings.Repeat("a", 64),
		"vcs:" + strings.Repeat("b", 40) + "+modified",
	} {
		document := `{"tool":{"name":"pepin","version":"v0.5.0","digest":"` + digest +
			`"},"ruleset":{"name":"pepin-config","digest":"sha256:rules"}}`
		got := normalizeBundleText(document, "", "v0.5.0")
		if !strings.Contains(got, `"digest":"<binary-digest>"`) {
			t.Errorf("tool digest %q was not normalized: %s", digest, got)
		}
		if !strings.Contains(got, `"digest":"sha256:rules"`) {
			t.Errorf("unrelated ruleset digest was modified: %s", got)
		}
	}
}

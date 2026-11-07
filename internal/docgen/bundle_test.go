package docgen

import (
	"strings"
	"testing"
)

func TestNormalizeVolatileReplacesOSCALBinaryDigest(t *testing.T) {
	input := `{"props":[{"name":"tool-digest","value":"sha256:abc123"},{"name":"other-digest","value":"sha256:def456"}]}`

	got := normalizeVolatile(input)
	if !strings.Contains(got, `"name":"tool-digest","value":"<provenance>"`) {
		t.Fatalf("tool digest was not normalized: %s", got)
	}
	if !strings.Contains(got, `"name":"other-digest","value":"sha256:def456"`) {
		t.Fatalf("unrelated digest was changed: %s", got)
	}
}

func TestNormalizeBundleTextReplacesBinaryDigest(t *testing.T) {
	binaryDigest := "sha256:" + strings.Repeat("a", 64)
	rulesetDigest := "sha256:" + strings.Repeat("b", 64)
	input := `{"tool":{"digest":"` + binaryDigest + `"},"ruleset":{"digest":"` + rulesetDigest + `"}}`

	got := normalizeBundleText(input, t.TempDir(), "v1.0.0")
	if !strings.Contains(got, `"tool":{"digest":"<provenance>"}`) {
		t.Fatalf("tool digest was not normalized: %s", got)
	}
	if !strings.Contains(got, `"ruleset":{"digest":"sha256:<sha256>"}`) {
		t.Fatalf("ruleset digest was changed unexpectedly: %s", got)
	}
}

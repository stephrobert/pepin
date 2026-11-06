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

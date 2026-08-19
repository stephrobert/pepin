package docgen

import (
	"os"
	"sort"
	"strings"
	"testing"
)

// TestDumpBlocks : harnais TEMPORAIRE de rédaction, à supprimer avant commit.
func TestDumpBlocks(t *testing.T) {
	out := os.Getenv("PEPIN_DUMP_BLOCKS")
	if out == "" {
		t.Skip("PEPIN_DUMP_BLOCKS non posé")
	}
	bin, err := ResolveBinary(repoRoot, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	lang := os.Getenv("PEPIN_DUMP_LANG")
	if lang == "" {
		lang = "en"
	}
	c, err := captureAll(repoRoot, bin, lang)
	if err != nil {
		t.Fatal(err)
	}
	m, err := BuildMatrix(repoRoot, lang)
	if err != nil {
		t.Fatal(err)
	}
	rem, err := RemediationCoverages(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	blocks := buildBlocks(lang, m, c, rem)
	ids := make([]string, 0, len(blocks))
	for id := range blocks {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var b strings.Builder
	for _, id := range ids {
		b.WriteString("\n===== " + id + " =====\n" + blocks[id] + "\n")
	}
	if err := os.WriteFile(out, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
}

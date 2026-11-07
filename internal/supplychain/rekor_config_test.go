package supplychain_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRekorV2ConfigPreservesTrustAndRefusesFallback(t *testing.T) {
	if _, err := exec.LookPath("jq"); err != nil {
		t.Fatal("jq requis pour tester la configuration de signature")
	}
	for _, tc := range []struct {
		name, config string
		cosignFails  bool
		wantError    bool
	}{
		{"mixed", `{"mediaType":"application/vnd.dev.sigstore.signingconfig.v0.2+json","rekorTlogUrls":[{"url":"https://v1.example","majorApiVersion":1},{"url":"https://v2.example","majorApiVersion":2}],"caUrls":[{"url":"https://ca.example"}],"oidcUrls":[{"url":"https://oidc.example"}],"tsaUrls":[{"url":"https://tsa.example"}],"tsaConfig":{"selector":"ALL"},"rekorTlogConfig":{"selector":"ANY"}}`, false, false},
		{"v2-only", `{"rekorTlogUrls":[{"url":"https://v2.example","majorApiVersion":2}]}`, false, false},
		{"v1-only", `{"rekorTlogUrls":[{"url":"https://v1.example","majorApiVersion":1}]}`, false, true},
		{"empty", `{"rekorTlogUrls":[]}`, false, true},
		{"missing-logs", `{}`, false, true},
		{"malformed", `{`, false, true},
		{"tuf-error", `{}`, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			fixture := filepath.Join(dir, "fixture.json")
			if err := os.WriteFile(fixture, []byte(tc.config), 0o600); err != nil {
				t.Fatal(err)
			}
			mock := "#!/usr/bin/env bash\nset -euo pipefail\n" +
				"[[ \"$1 $2 $3 $4\" == 'signing-config create --with-default-rekor-v2 --out' ]]\n" +
				"if [[ \"$PEPIN_TEST_COSIGN_FAIL\" == 1 ]]; then echo 'TUF unavailable' >&2; exit 1; fi\n" +
				"cp -- \"$PEPIN_TEST_SIGNING_CONFIG\" \"$5\"\n"
			if err := os.WriteFile(filepath.Join(dir, "cosign"), []byte(mock), 0o700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv("PEPIN_TEST_SIGNING_CONFIG", fixture)
			fail := "0"
			if tc.cosignFails {
				fail = "1"
			}
			t.Setenv("PEPIN_TEST_COSIGN_FAIL", fail)
			output := filepath.Join(dir, "config.json")
			out, err := exec.CommandContext(t.Context(), "bash",
				filepath.Join(repoRoot(t), "tools", "release", "rekor-v2-config.sh"), output).CombinedOutput()
			if (err != nil) != tc.wantError {
				t.Fatalf("configuration : %v\n%s", err, out)
			}
			if tc.wantError {
				if _, err := os.Stat(output); !os.IsNotExist(err) {
					t.Fatalf("une erreur ne doit pas publier de configuration : %v", err)
				}
				return
			}
			raw, err := os.ReadFile(output)
			if err != nil {
				t.Fatal(err)
			}
			var got struct {
				Logs []struct {
					URL     string `json:"url"`
					Version int    `json:"majorApiVersion"`
				} `json:"rekorTlogUrls"`
			}
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Fatal(err)
			}
			if len(got.Logs) != 1 || got.Logs[0].Version != 2 || got.Logs[0].URL != "https://v2.example" {
				t.Fatalf("repli v1 ou URL substituée : %s", raw)
			}
			var original, filtered map[string]any
			if err := json.Unmarshal([]byte(tc.config), &original); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(raw, &filtered); err != nil {
				t.Fatal(err)
			}
			delete(original, "rekorTlogUrls")
			delete(filtered, "rekorTlogUrls")
			if !reflect.DeepEqual(original, filtered) {
				t.Fatalf("configuration hors journaux modifiée : %s", raw)
			}
		})
	}
}

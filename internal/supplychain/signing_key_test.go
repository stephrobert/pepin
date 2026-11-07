package supplychain_test

import (
	"errors"
	"regexp"
	"strings"
	"testing"
)

func TestReleaseUsesDefaultKeylessCosignKeys(t *testing.T) {
	body := workflows(t)["release.yml"]
	if err := checkDefaultKeylessCosignKeys(body); err != nil {
		t.Fatal(err)
	}
}

func checkDefaultKeylessCosignKeys(body string) error {
	body = regexp.MustCompile(`\\\r?\n`).ReplaceAllString(body, " ")
	commands := regexp.MustCompile(`(?m)^\s*(?:run:\s*)?cosign\s+sign(?:-blob)?\s+([^\n]+)`).FindAllStringSubmatch(body, -1)
	if len(commands) == 0 {
		return errors.New("aucune signature de release exercée par la garde")
	}
	for _, command := range commands {
		if strings.Contains(command[1], "--key") || strings.Contains(command[1], "--signing-algorithm") {
			return errors.New("la release ne doit pas substituer une clé ou un algorithme au défaut keyless P-256 de Cosign")
		}
	}
	return nil
}

func TestKeylessGuardChecksContinuedSigningOptions(t *testing.T) {
	for _, tc := range []struct {
		name, script string
		wantError    bool
	}{
		{"inline-default", "run: cosign sign-blob --yes checksums.txt", false},
		{"continued-default", "run: |\n  cosign sign-blob --yes \\\n    --bundle checksums.bundle checksums.txt", false},
		{"inline-key", "run: cosign sign --key custom.key image", true},
		{"continued-key", "run: |\n  cosign sign-blob --yes \\\n    --key custom.key checksums.txt", true},
		{"continued-algorithm", "run: |\n  cosign sign --yes \\\n    --signing-algorithm rsa-sign-pkcs1-2048-sha256 image", true},
		{"crlf-key", "run: |\r\n  cosign sign-blob --yes \\\r\n    --key custom.key checksums.txt", true},
		{"multiple-continuations", "run: |\n  cosign sign-blob --yes \\\n    --bundle checksums.bundle \\\n    --key=custom.key checksums.txt", true},
		{"no-signature", "run: echo nothing-to-sign", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := checkDefaultKeylessCosignKeys(tc.script)
			if (err != nil) != tc.wantError {
				t.Fatalf("garde keyless incorrecte : %v (erreur attendue %v)", err, tc.wantError)
			}
		})
	}
}

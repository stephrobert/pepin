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
	commands := cosignSigningCommands(body)
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

func cosignSigningCommands(body string) [][]string {
	body = regexp.MustCompile(`\\\r?\n`).ReplaceAllString(body, " ")
	return regexp.MustCompile(`(?m)^\s*(?:run:\s*)?cosign\s+sign(?:-blob)?\s+([^\n]+)`).FindAllStringSubmatch(body, -1)
}

func TestReleaseUsesAuthenticatedRekorV2(t *testing.T) {
	body := workflows(t)["release.yml"]
	if strings.Count(body, `rekor-v2-config.sh "$RUNNER_TEMP/rekor-v2.json"`) != 2 {
		t.Fatal("les deux signatures doivent générer la configuration authentifiée Rekor v2")
	}
	if err := checkRekorV2SigningCommands(body); err != nil {
		t.Fatal(err)
	}
}

func checkRekorV2SigningCommands(body string) error {
	commands := cosignSigningCommands(body)
	if len(commands) != 2 {
		return errors.New("les signatures image et checksums doivent être exercées")
	}
	for _, command := range commands {
		if !strings.Contains(command[1], `--signing-config "$RUNNER_TEMP/rekor-v2.json"`) ||
			strings.Contains(command[1], "--use-signing-config") ||
			strings.Contains(command[1], "--tlog-upload") ||
			strings.Contains(command[1], "--insecure-ignore") ||
			strings.Contains(command[1], "--trusted-root") {
			return errors.New("signature sans configuration Rekor v2 authentifiée ou avec contournement de confiance")
		}
	}
	return nil
}

func TestRekorGuardRejectsDefaultAndDisabledTransparency(t *testing.T) {
	valid := `cosign sign --yes --signing-config "$RUNNER_TEMP/rekor-v2.json" image` + "\n" +
		`cosign sign-blob --yes --signing-config "$RUNNER_TEMP/rekor-v2.json" checksums.txt`
	for _, tc := range []struct {
		name, body string
		wantError  bool
	}{
		{"v2", valid, false},
		{"default-v1", strings.ReplaceAll(valid, `--signing-config "$RUNNER_TEMP/rekor-v2.json"`, ""), true},
		{"disabled-config", valid + " --use-signing-config=false", true},
		{"disabled-tlog", valid + " --tlog-upload=false", true},
		{"ignored-tlog", valid + " --insecure-ignore-tlog", true},
		{"custom-root", valid + " --trusted-root unchecked.json", true},
		{"missing-image", strings.Split(valid, "\n")[1], true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := checkRekorV2SigningCommands(tc.body); (err != nil) != tc.wantError {
				t.Fatalf("garde Rekor v2 incorrecte : %v", err)
			}
		})
	}
}

func TestReleaseVerifiesSignaturesBeforePublication(t *testing.T) {
	if err := checkReleaseSignatureVerification(workflows(t)["release.yml"]); err != nil {
		t.Fatal(err)
	}
}

func checkReleaseSignatureVerification(body string) error {
	release := strings.Index(body, "\n  release:\n")
	if release < 0 || !strings.Contains(body[release:], "needs: [binaries, image]") {
		return errors.New("la publication doit attendre la signature vérifiée de l'image")
	}
	if strings.Count(body, "CERTIFICATE_IDENTITY: https://github.com/${{ github.workflow_ref }}") != 2 {
		return errors.New("les deux signatures doivent vérifier l'identité exacte du workflow")
	}
	body = regexp.MustCompile(`\\\r?\n`).ReplaceAllString(body, " ")
	commands := regexp.MustCompile(`(?m)^\s*cosign\s+(verify(?:-blob)?)\s+([^\n]+)`).FindAllStringSubmatchIndex(body, -1)
	if len(commands) != 2 {
		return errors.New("les deux signatures doivent être vérifiées")
	}
	for _, command := range commands {
		kind, args := body[command[2]:command[3]], body[command[4]:command[5]]
		if !strings.Contains(args, `--certificate-identity "$CERTIFICATE_IDENTITY"`) ||
			!strings.Contains(args, "--certificate-oidc-issuer https://token.actions.githubusercontent.com") ||
			strings.Contains(args, "--insecure-ignore") ||
			strings.Contains(args, "--trusted-root") ||
			strings.Contains(args, "--key") {
			return errors.New("vérification sans identité exacte ou avec contournement de confiance")
		}
		sign := `cosign sign --yes`
		if kind == "verify-blob" {
			sign = `cosign sign-blob --yes`
			if !strings.Contains(args, "--bundle checksums.txt.cosign.bundle") ||
				!strings.HasSuffix(strings.TrimSpace(args), "checksums.txt") {
				return errors.New("la vérification doit porter sur les empreintes signées")
			}
		} else if !strings.HasSuffix(strings.TrimSpace(args), `"${IMAGE}@${DIGEST}"`) {
			return errors.New("l'image doit être vérifiée par digest")
		}
		signAt := strings.Index(body, sign)
		publishAt := strings.Index(body, "- name: Create release with assets")
		if signAt < 0 || signAt >= command[0] || publishAt <= command[0] {
			return errors.New("la vérification doit suivre la signature et précéder la publication")
		}
	}
	return nil
}

func TestReleaseVerificationGuardRejectsMissingOrUncheckedSignatures(t *testing.T) {
	body := workflows(t)["release.yml"]
	for _, tc := range []struct {
		name, from, to string
	}{
		{"missing-image-dependency", "needs: [binaries, image]", "needs: binaries"},
		{"missing-image-verification", "cosign verify \\", "echo verify \\"},
		{"missing-blob-verification", "cosign verify-blob", "echo verify-blob"},
		{"unbound-identity", `--certificate-identity "$CERTIFICATE_IDENTITY"`, `--certificate-identity-regexp ".*"`},
		{"wrong-issuer", "https://token.actions.githubusercontent.com", "https://unchecked.example"},
		{"disabled-transparency", "cosign verify-blob", "cosign verify-blob --insecure-ignore-tlog"},
		{"mutable-image", `"${IMAGE}@${DIGEST}"`, `"${IMAGE}:latest"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mutant := strings.ReplaceAll(body, tc.from, tc.to)
			if mutant == body {
				t.Fatal("la mutation doit modifier le workflow")
			}
			if err := checkReleaseSignatureVerification(mutant); err == nil {
				t.Fatal("le workflow altéré ne doit pas passer")
			}
		})
	}
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

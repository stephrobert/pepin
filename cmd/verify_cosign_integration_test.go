package cmd

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Ce témoin opt-in utilise Cosign réel et seulement du matériau synthétique.
func TestNativeCosignKeyStrengthAndSignature(t *testing.T) {
	if os.Getenv("PEPIN_COSIGN_INTEGRATION") != "1" {
		t.Skip("intégration explicite : PEPIN_COSIGN_INTEGRATION=1 et cosign requis")
	}
	cosign, err := exec.LookPath("cosign")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("COSIGN_PASSWORD", rand.Text())
	bin := os.Getenv("PEPIN_COSIGN_BINARY")
	if bin == "" {
		bin = buildPepin(t)
	} else {
		bin, err = filepath.Abs(bin)
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, bits := range []int{2048, 0, 1024} {
		t.Run(map[int]string{2048: "RSA2048", 0: "P256", 1024: "RSA1024"}[bits], func(t *testing.T) {
			dir := sceller(t, bin)
			prefix := filepath.Join(dir, "key")
			pubFile, privFile := prefix+".pub", prefix+".key"
			blob := filepath.Join(dir, "checksums.txt")
			ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
			defer cancel()
			if bits == 1024 {
				key, err := rsa.GenerateKey(rand.Reader, bits)
				if err != nil {
					t.Fatal(err)
				}
				public, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(pubFile, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: public}), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := verifyCosign(ctx, dir, pubFile, ""); err == nil || !strings.Contains(err.Error(), "2048") {
					t.Fatalf("clé faible acceptée : %v", err)
				}
				out, err := exec.CommandContext(ctx, bin, "verify", dir, "--pubkey", pubFile, "--require-signature").CombinedOutput()
				if err == nil || !strings.Contains(string(out), "2048") {
					t.Fatalf("le binaire accepte la clé faible : %v\n%s", err, out)
				}
				return
			}
			generate := []string{"generate-key-pair", "--output-key-prefix", prefix}
			if bits == 2048 {
				key, err := rsa.GenerateKey(rand.Reader, bits)
				if err != nil {
					t.Fatal(err)
				}
				rawKey := filepath.Join(dir, "import.pem")
				if err := os.WriteFile(rawKey, pem.EncodeToMemory(&pem.Block{
					Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key),
				}), 0o600); err != nil {
					t.Fatal(err)
				}
				generate = []string{"import-key-pair", "--key", rawKey, "--output-key-prefix", prefix}
			}
			if out, err := exec.CommandContext(ctx, cosign, generate...).CombinedOutput(); err != nil {
				t.Fatalf("génération synthétique : %v\n%s", err, out)
			}
			args := []string{"sign-blob", "--yes", "--key", privFile,
				"--bundle", blob + ".bundle", "--use-signing-config=false", "--oidc-disable-ambient-providers"}
			if bits == 2048 {
				args = append(args, "--signing-algorithm", "rsa-sign-pkcs1-2048-sha256")
			}
			command := exec.CommandContext(ctx, cosign, append(args, blob)...)
			if out, err := command.CombinedOutput(); err != nil {
				t.Fatalf("signature synthétique : %v\n%s", err, out)
			}
			if err := verifyCosign(ctx, dir, pubFile, ""); err != nil {
				t.Fatalf("signature conforme refusée : %v", err)
			}
			if out, err := exec.CommandContext(ctx, bin, "verify", dir, "--pubkey", pubFile, "--require-signature").CombinedOutput(); err != nil {
				t.Fatalf("le binaire refuse la signature conforme : %v\n%s", err, out)
			}
			reorderSignedChecksums(t, blob)
			if out, err := exec.CommandContext(ctx, bin, "verify", dir).CombinedOutput(); err != nil {
				t.Fatalf("le témoin altéré doit garder une intégrité valide : %v\n%s", err, out)
			}
			if err := verifyCosign(ctx, dir, pubFile, ""); err == nil ||
				(!strings.Contains(err.Error(), "signature cosign INVALIDE") && !strings.Contains(err.Error(), "INVALID cosign signature")) {
				t.Fatalf("Cosign doit refuser la signature altérée : %v", err)
			}
			out, err := exec.CommandContext(ctx, bin, "verify", dir, "--pubkey", pubFile, "--require-signature").CombinedOutput()
			if err == nil ||
				(!strings.Contains(string(out), "signature cosign INVALIDE") && !strings.Contains(string(out), "INVALID cosign signature")) {
				t.Fatalf("le binaire doit refuser la signature, pas l'intégrité : %v\n%s", err, out)
			}
		})
	}
}

func reorderSignedChecksums(t *testing.T, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(raw), "\n")
	if len(lines) < 2 || lines[0] == "" || lines[1] == "" || lines[0] == lines[1] {
		t.Fatal("deux sommes distinctes sont nécessaires au témoin")
	}
	lines[0], lines[1] = lines[1], lines[0]
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestSignatureTamperingWitnessKeepsBundleIntegrity(t *testing.T) {
	bin := buildPepin(t)
	dir := sceller(t, bin)
	path := filepath.Join(dir, "checksums.txt")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	reorderSignedChecksums(t, path)
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) == string(after) {
		t.Fatal("les octets signés n'ont pas changé")
	}
	if out, err := exec.CommandContext(t.Context(), bin, "verify", dir).CombinedOutput(); err != nil {
		t.Fatalf("la permutation doit conserver l'intégrité du bundle : %v\n%s", err, out)
	}
}

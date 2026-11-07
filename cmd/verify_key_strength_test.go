package cmd

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyRefusesWeakSignatureKeyBeforeCosign(t *testing.T) {
	dir := t.TempDir()
	tool := filepath.Join(dir, "cosign")
	if err := os.WriteFile(tool, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	public := filepath.Join(dir, "weak.pub")
	if err := os.WriteFile(public, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "checksums.txt.bundle"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifyCosign(context.Background(), dir, public, ""); err == nil || !strings.Contains(err.Error(), "2048") {
		t.Fatalf("la garde de clé doit refuser avant de déléguer : %v", err)
	}
}

func TestVerifyRejectsRemoteKeyReferencesWithoutExposingCredentials(t *testing.T) {
	for _, reference := range []string{
		"https://example.org/do-not-print/key.pub",
		"awskms://do-not-print",
		"hashivault://do-not-print",
	} {
		err := verifyCosign(context.Background(), t.TempDir(), reference, "")
		if err == nil || !strings.Contains(err.Error(), "local") || strings.Contains(err.Error(), "do-not-print") {
			t.Fatalf("refus de référence distante incorrect : %v", err)
		}
	}
}

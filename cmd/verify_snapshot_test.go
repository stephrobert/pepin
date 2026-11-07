package cmd

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCosignReadsTheValidatedKeySnapshot(t *testing.T) {
	dir := t.TempDir()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	expected := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
	original := filepath.Join(dir, "original.pub")
	captured := filepath.Join(dir, "captured.pub")
	pathWitness := filepath.Join(dir, "snapshot-path")
	for path, data := range map[string][]byte{
		original: expected, filepath.Join(dir, "checksums.txt.bundle"): []byte("{}"),
		filepath.Join(dir, "cosign"): []byte("#!/bin/sh\nprintf '%s' \"$3\" > \"$SNAPSHOT_PATH\"\ncp \"$3\" \"$CAPTURED_KEY\"\nprintf 'replaced' > \"$ORIGINAL_KEY\"\n"),
	} {
		if err := os.WriteFile(path, data, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("SNAPSHOT_PATH", pathWitness)
	t.Setenv("CAPTURED_KEY", captured)
	t.Setenv("ORIGINAL_KEY", original)
	if err := verifyCosign(context.Background(), dir, original, ""); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(captured)
	if err != nil || !bytes.Equal(got, expected) {
		t.Fatalf("la clé validée n'est pas celle lue par Cosign : %v", err)
	}
	snapshot, err := os.ReadFile(pathWitness)
	if err != nil || string(snapshot) == original {
		t.Fatalf("le chemin original a été relu : %v", err)
	}
	if _, err := os.Stat(string(snapshot)); !os.IsNotExist(err) {
		t.Fatal("l'instantané temporaire n'a pas été nettoyé")
	}
	if got, err := os.ReadFile(original); err != nil || string(got) != "replaced" {
		t.Fatal("le témoin n'a pas réellement remplacé la clé d'origine")
	}
}

func TestCancelledSignatureVerificationIsNotSuccess(t *testing.T) {
	dir := t.TempDir()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	public := filepath.Join(dir, "key.pub")
	for path, data := range map[string][]byte{
		public: pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}),
		filepath.Join(dir, "checksums.txt.bundle"): []byte("{}"),
		filepath.Join(dir, "cosign"):               []byte("#!/bin/sh\nexit 0\n"),
	} {
		if err := os.WriteFile(path, data, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := verifyCosign(ctx, dir, public, ""); !errors.Is(err, context.Canceled) {
		t.Fatalf("annulation perdue : %v", err)
	}
	if err := verifyCosign(context.Background(), dir, "https://example.org/key.pub", ""); err == nil || !strings.Contains(err.Error(), "local") {
		t.Fatal("référence distante non contrôlée acceptée")
	}
}

func TestSignatureVerificationPreservesCancellationAfterCosignStarts(t *testing.T) {
	for _, mode := range []string{"cancel", "deadline"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			if err != nil {
				t.Fatal(err)
			}
			der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
			if err != nil {
				t.Fatal(err)
			}
			public := filepath.Join(dir, "key.pub")
			started := filepath.Join(dir, "started")
			for path, data := range map[string][]byte{
				public: pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}),
				filepath.Join(dir, "checksums.txt.bundle"): []byte("{}"),
				filepath.Join(dir, "cosign"):               []byte("#!/bin/sh\nprintf '%s' \"$3\" > \"$COSIGN_STARTED\"\nexec sleep 30\n"),
			} {
				if err := os.WriteFile(path, data, 0o700); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv("COSIGN_STARTED", started)
			ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
			defer cancel()
			result := make(chan error, 1)
			go func() { result <- verifyCosign(ctx, dir, public, "") }()
			until := time.Now().Add(time.Second)
			for {
				if _, err := os.Stat(started); err == nil {
					break
				} else if !os.IsNotExist(err) {
					t.Fatal(err)
				}
				if time.Now().After(until) {
					t.Fatal("le sous-processus Cosign n'a pas démarré")
				}
				time.Sleep(5 * time.Millisecond)
			}
			want := context.DeadlineExceeded
			if mode == "cancel" {
				want = context.Canceled
				cancel()
			}
			err = <-result
			if !errors.Is(err, want) || strings.Contains(strings.ToLower(err.Error()), "invalid") {
				t.Fatalf("interruption après démarrage mal rapportée : %v (attendu %v)", err, want)
			}
		})
	}
}

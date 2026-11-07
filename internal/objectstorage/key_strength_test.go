package objectstorage

import (
	"context"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestS3RejectsShortSigningSecretBeforeSDKRequest(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	_, err := CollectBuckets(context.Background(), "witness", server.URL, "witness", t.Name(), strings.Repeat("x", 13), false, false)
	if err == nil || !strings.Contains(err.Error(), "112") || calls.Load() != 0 {
		t.Fatalf("secret S3 trop court accepté : %v (%d appels)", err, calls.Load())
	}
}

func TestS3HonorsCustomCABundles(t *testing.T) {
	for _, source := range []string{"environment", "profile"} {
		t.Run(source, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls.Add(1)
				if _, err := w.Write([]byte(`<ListAllMyBucketsResult><Buckets></Buckets></ListAllMyBucketsResult>`)); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			dir := t.TempDir()
			caPath := filepath.Join(dir, "ca.pem")
			if err := os.WriteFile(caPath, pem.EncodeToMemory(&pem.Block{
				Type: "CERTIFICATE", Bytes: server.Certificate().Raw,
			}), 0o600); err != nil {
				t.Fatal(err)
			}
			profile := "[default]\n"
			t.Setenv("AWS_CA_BUNDLE", "")
			if source == "environment" {
				t.Setenv("AWS_CA_BUNDLE", caPath)
			} else {
				profile += "ca_bundle = " + caPath + "\n"
			}
			configPath := filepath.Join(dir, "config")
			credentialsPath := filepath.Join(dir, "credentials")
			for path, data := range map[string]string{configPath: profile, credentialsPath: ""} {
				if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("AWS_CONFIG_FILE", configPath)
			t.Setenv("AWS_SHARED_CREDENTIALS_FILE", credentialsPath)
			t.Setenv("AWS_PROFILE", "default")
			t.Setenv("AWS_MAX_ATTEMPTS", "1")
			got, err := CollectBuckets(t.Context(), "witness", server.URL, "us-east-1",
				t.Name(), strings.Repeat("x", 14), false, false)
			if err != nil || len(got) != 0 || calls.Load() != 1 {
				t.Fatalf("CA personnalisée %s refusée : %v (%d appels)", source, err, calls.Load())
			}
		})
	}
}

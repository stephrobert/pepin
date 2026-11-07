package kubeauth

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func keyForTest(t *testing.T, bits int) crypto.Signer {
	t.Helper()
	if bits == 0 {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		return key
	}
	key, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func certificateForTest(t *testing.T, key crypto.Signer, parent *x509.Certificate, issuer crypto.Signer) ([]byte, []byte, *x509.Certificate) {
	t.Helper()
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "local TLS witness"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true, IsCA: parent == nil,
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
	}
	if parent == nil {
		parent, issuer = template, key
	}
	der, err := x509.CreateCertificate(rand.Reader, template, parent, key.Public(), issuer)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	private, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: private}), cert
}

func configForTest(t *testing.T, server string, ca, cert, key []byte) string {
	t.Helper()
	user := map[string]string{"token": t.Name()}
	if len(cert) > 0 {
		user = map[string]string{
			"client-certificate-data": base64.StdEncoding.EncodeToString(cert),
			"client-key-data":         base64.StdEncoding.EncodeToString(key),
		}
	}
	raw, err := json.Marshal(map[string]any{
		"clusters": []any{map[string]any{"cluster": map[string]string{
			"server": server, "certificate-authority-data": base64.StdEncoding.EncodeToString(ca),
		}}},
		"users": []any{map[string]any{"user": user}},
	})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "kubeconfig")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestClientKeyStrength(t *testing.T) {
	caKey := keyForTest(t, 0)
	caPEM, _, ca := certificateForTest(t, caKey, nil, nil)
	for _, bits := range []int{1024, 2048, 0} {
		t.Run(map[int]string{1024: "RSA1024", 2048: "RSA2048", 0: "P256"}[bits], func(t *testing.T) {
			cert, key, _ := certificateForTest(t, keyForTest(t, bits), ca, caKey)
			_, err := Load(configForTest(t, "https://127.0.0.1", caPEM, cert, key), time.Second)
			if bits == 1024 {
				if err == nil || !strings.Contains(err.Error(), "2048") {
					t.Fatalf("RSA 1024 doit être refusé explicitement : %v", err)
				}
			} else if err != nil {
				t.Fatalf("clé conforme refusée : %v", err)
			}
		})
	}
}

func TestServerKeyStrength(t *testing.T) {
	caKey := keyForTest(t, 0)
	caPEM, _, ca := certificateForTest(t, caKey, nil, nil)
	for _, bits := range []int{1024, 2048, 0} {
		t.Run(map[int]string{1024: "RSA1024", 2048: "RSA2048", 0: "P256"}[bits], func(t *testing.T) {
			certPEM, keyPEM, _ := certificateForTest(t, keyForTest(t, bits), ca, caKey)
			pair, err := tls.X509KeyPair(certPEM, keyPEM)
			if err != nil {
				t.Fatal(err)
			}
			server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}))
			server.TLS = &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{pair}}
			server.StartTLS()
			defer server.Close()
			cfg, err := Load(configForTest(t, server.URL, caPEM, nil, nil), time.Second)
			if err != nil {
				t.Fatal(err)
			}
			resp, err := cfg.Client.Get(server.URL)
			if resp != nil {
				if err := resp.Body.Close(); err != nil {
					t.Fatal(err)
				}
			}
			if bits == 1024 {
				if err == nil || !strings.Contains(err.Error(), "2048") {
					t.Fatalf("serveur RSA 1024 doit être refusé : %v", err)
				}
			} else if err != nil {
				t.Fatalf("serveur conforme refusé : %v", err)
			}
		})
	}
}

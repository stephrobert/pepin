package tlspolicy_test

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stephrobert/pepin/internal/collect"
	"github.com/stephrobert/pepin/internal/collectkit"
	"github.com/stephrobert/pepin/internal/objectstorage"
)

func TestRESTAndS3UseTheSameTLSKeyPolicy(t *testing.T) {
	t.Setenv("AWS_MAX_ATTEMPTS", "1")
	for _, bits := range []int{1024, 2048, 0} {
		t.Run(map[int]string{1024: "RSA1024", 2048: "RSA2048", 0: "P256"}[bits], func(t *testing.T) {
			var key crypto.Signer
			var err error
			if bits == 0 {
				key, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			} else {
				key, err = rsa.GenerateKey(rand.Reader, bits)
			}
			if err != nil {
				t.Fatal(err)
			}
			cert := &x509.Certificate{
				SerialNumber: big.NewInt(1), NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
				IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
				KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
				ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
				IsCA:        true, BasicConstraintsValid: true,
			}
			der, err := x509.CreateCertificate(rand.Reader, cert, cert, key.Public(), key)
			if err != nil {
				t.Fatal(err)
			}
			certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
			private, err := x509.MarshalPKCS8PrivateKey(key)
			if err != nil {
				t.Fatal(err)
			}
			pair, err := tls.X509KeyPair(certPEM, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: private}))
			if err != nil {
				t.Fatal(err)
			}
			var requests atomic.Int32
			server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				var data string
				if r.URL.Path == "/items" {
					data = `{"items":[]}`
				} else {
					data = `<ListAllMyBucketsResult><Buckets></Buckets></ListAllMyBucketsResult>`
				}
				if _, err := w.Write([]byte(data)); err != nil {
					t.Error(err)
				}
			}))
			server.TLS = &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{pair}}
			server.StartTLS()
			defer server.Close()
			base := http.DefaultTransport
			transport := base.(*http.Transport).Clone()
			pool := x509.NewCertPool()
			if !pool.AppendCertsFromPEM(certPEM) {
				t.Fatal("CA du témoin illisible")
			}
			transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: pool}
			http.DefaultTransport = transport
			t.Cleanup(func() { http.DefaultTransport = base })

			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			inv, err := collectkit.Run(ctx, "witness", collect.Spec{
				BaseURL:   server.URL,
				Resources: []collect.ResourceSpec{{Type: "witness", Path: "/items", Items: "items"}},
			}, collect.HeaderAuth{Header: "X-Witness", Value: t.Name()}, nil,
				collectkit.S3{}, collectkit.OKS{}, collectkit.EIM{}, nil)
			if err != nil {
				t.Fatal(err)
			}
			if len(inv.Collection.Units) != 1 {
				t.Fatal("le refus TLS doit rester visible dans l'état de collecte")
			}
			unit := inv.Collection.Units[0]
			if unit.Complete != (bits != 1024) || (bits == 1024 && !strings.Contains(unit.Detail, "2048")) {
				t.Fatalf("état REST incorrect : %+v", unit)
			}
			_, err = objectstorage.CollectBuckets(ctx, "witness", server.URL, "witness", t.Name(), t.Name(), false, false)
			if bits == 1024 {
				if err == nil || !strings.Contains(err.Error(), "2048") || requests.Load() != 0 {
					t.Fatalf("S3 doit refuser sans envoyer la requête : %v, %d requêtes", err, requests.Load())
				}
			} else if err != nil || requests.Load() < 2 {
				t.Fatalf("REST et S3 conformes refusés : %v, %d requêtes", err, requests.Load())
			}
		})
	}
}

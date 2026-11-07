package tlspolicy

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"math/big"
	"net/http"
	"testing"

	"github.com/stephrobert/pepin/internal/cryptopolicy"
	"github.com/stephrobert/pepin/internal/i18n"
)

func TestKeyStrengthBoundaries(t *testing.T) {
	for _, bits := range []int{1024, 2047, 2048, 3072} {
		key := &rsa.PublicKey{N: new(big.Int).Lsh(big.NewInt(1), uint(bits-1)), E: 65537}
		if err := cryptopolicy.ValidatePublicKey(key); (err == nil) != (bits >= 2048) {
			t.Fatalf("RSA %d : %v", bits, err)
		}
	}
	for _, curve := range []elliptic.Curve{elliptic.P224(), elliptic.P256(), elliptic.P384(), elliptic.P521()} {
		if err := cryptopolicy.ValidatePublicKey(&ecdsa.PublicKey{Curve: curve}); err != nil {
			t.Fatal(err)
		}
	}
	if err := cryptopolicy.ValidatePublicKey(make(ed25519.PublicKey, ed25519.PublicKeySize)); err != nil {
		t.Fatal(err)
	}
	for _, key := range []any{nil, (*rsa.PublicKey)(nil), &rsa.PublicKey{}, (*ecdsa.PublicKey)(nil), &ecdsa.PublicKey{}, ed25519.PublicKey{1}} {
		if err := cryptopolicy.ValidatePublicKey(key); err == nil {
			t.Fatalf("clé invalide acceptée : %T", key)
		}
	}
}

func TestEveryCertificateInATrustedPathMustBeStrong(t *testing.T) {
	strong := &x509.Certificate{PublicKey: &ecdsa.PublicKey{Curve: elliptic.P256()}}
	weak := &x509.Certificate{PublicKey: &rsa.PublicKey{N: new(big.Int).Lsh(big.NewInt(1), 1023), E: 65537}}
	for _, chains := range [][][]*x509.Certificate{nil, {{}}, {{weak}}, {{strong, weak}}} {
		if err := Config().VerifyConnection(tls.ConnectionState{VerifiedChains: chains}); err == nil {
			t.Fatal("chaîne absente ou faible acceptée")
		}
	}
	if err := Config().VerifyConnection(tls.ConnectionState{
		VerifiedChains: [][]*x509.Certificate{{strong, weak}, {strong, strong}},
	}); err != nil {
		t.Fatalf("le chemin alternatif fort doit rester accepté : %v", err)
	}
	if err := ValidateClientCertificate(tls.Certificate{}); err == nil {
		t.Fatal("certificat client absent accepté")
	}
	if err := ValidateClientCertificate(tls.Certificate{Certificate: [][]byte{{1}}}); err == nil {
		t.Fatal("certificat client illisible accepté")
	}
}

func TestPolicyIsIndependentAndDoesNotDisableTrust(t *testing.T) {
	base := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = base })
	custom := base.(*http.Transport).Clone()
	pool := x509.NewCertPool()
	custom.TLSClientConfig = &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}
	http.DefaultTransport = custom
	first, err := Transport()
	if err != nil {
		t.Fatal(err)
	}
	second, err := Transport()
	if err != nil {
		t.Fatal(err)
	}
	if first == second || first == custom || first.TLSClientConfig == second.TLSClientConfig {
		t.Fatal("la politique partage un transport mutable")
	}
	cfg := first.TLSClientConfig
	if cfg.MinVersion != tls.VersionTLS12 || cfg.InsecureSkipVerify || cfg.RootCAs != pool ||
		cfg.VerifyConnection == nil || first.Proxy == nil || !first.ForceAttemptHTTP2 {
		t.Fatal("politique ou comportement HTTP standard perdu")
	}
	allowed := map[uint16]bool{
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256: true, tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384: true,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256: true, tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384: true,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256: true, tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256: true,
	}
	for _, suite := range cfg.CipherSuites {
		if !allowed[suite] {
			t.Fatalf("suite TLS 1.2 non AEAD/ECDHE : %x", suite)
		}
	}
}

func TestRefusalsAreBilingual(t *testing.T) {
	before := i18n.Current()
	t.Cleanup(func() { i18n.Set(before) })
	for lang, expected := range map[i18n.Lang]string{i18n.FR: "politique cryptographique : clé RSA inférieure à 2048 bits", i18n.EN: "cryptographic policy: RSA key below 2048 bits"} {
		i18n.Set(lang)
		err := cryptopolicy.ValidatePublicKey(&rsa.PublicKey{})
		if err == nil || err.Error() != expected {
			t.Fatalf("%s : %v", lang, err)
		}
	}
}

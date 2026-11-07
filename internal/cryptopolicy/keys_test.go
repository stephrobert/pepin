package cryptopolicy

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"strings"
	"testing"

	"github.com/stephrobert/pepin/internal/i18n"
)

func TestHMACKeySizeIsMeasuredInBytesNotCharactersOrEntropy(t *testing.T) {
	for _, input := range []string{"", strings.Repeat("x", 13), strings.Repeat("é", 6)} {
		if err := ValidateHMACSecret(input); err == nil {
			t.Fatal("matériau inférieur à 14 octets accepté")
		}
	}
	for _, input := range []string{strings.Repeat("x", 14), strings.Repeat("é", 7), strings.Repeat("x", 128)} {
		if err := ValidateHMACSecret(input); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPublicPEMKeyStrength(t *testing.T) {
	for _, bits := range []int{1024, 2048, 0} {
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
		der, err := x509.MarshalPKIXPublicKey(key.Public())
		if err != nil {
			t.Fatal(err)
		}
		data := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
		_, err = ParsePublicKey(data)
		if (err == nil) != (bits != 1024) {
			t.Fatalf("clé de %d bits : %v", bits, err)
		}
		if bits != 1024 {
			if _, err := ParsePublicKey(append([]byte(" \r\n"), data...)); err != nil {
				t.Fatalf("espaces avant la clé refusés : %v", err)
			}
			for _, invalid := range [][]byte{
				append(append([]byte(nil), data...), data...),
				append(append([]byte(nil), data...), []byte("extra")...),
				append([]byte("extra\n"), data...),
				append([]byte("-----BEGIN PUBLIC KEY-----\ninvalid\n-----END PUBLIC KEY-----\n"), data...),
			} {
				if _, err := ParsePublicKey(invalid); err == nil {
					t.Fatal("contenu supplémentaire accepté")
				}
			}
			if bits == 2048 {
				rsaKey := key.Public().(*rsa.PublicKey)
				if _, err := ParsePublicKey(pem.EncodeToMemory(&pem.Block{
					Type: "RSA PUBLIC KEY", Bytes: x509.MarshalPKCS1PublicKey(rsaKey),
				})); err != nil {
					t.Fatalf("PKCS#1 conforme refusé : %v", err)
				}
			}
		}
	}
	for _, data := range [][]byte{
		nil, []byte("not a key"),
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte{1}}),
		pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: []byte{1}}),
	} {
		if _, err := ParsePublicKey(data); err == nil {
			t.Fatal("clé publique invalide acceptée")
		}
	}
}

func TestHMACRefusalsAreBilingualAndDoNotExposeSecrets(t *testing.T) {
	before := i18n.Current()
	t.Cleanup(func() { i18n.Set(before) })
	for lang, expected := range map[i18n.Lang]string{
		i18n.FR: "politique cryptographique : secret HMAC inférieur à 112 bits (14 octets)",
		i18n.EN: "cryptographic policy: HMAC secret below 112 bits (14 bytes)",
	} {
		i18n.Set(lang)
		secret := "do-not-print"
		err := ValidateHMACSecret(secret)
		if err == nil || err.Error() != expected || strings.Contains(err.Error(), secret) {
			t.Fatalf("%s : refus incorrect", lang)
		}
	}
}

func FuzzParsePublicKey(f *testing.F) {
	public := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize)).Public()
	der, err := x509.MarshalPKIXPublicKey(public)
	if err != nil {
		f.Fatal(err)
	}
	data := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
	f.Add(data)
	f.Add([]byte("not a key"))
	f.Add(append(append([]byte(nil), data...), data...))
	f.Fuzz(func(t *testing.T, data []byte) {
		key, err := ParsePublicKey(data)
		if err != nil {
			return
		}
		der, err := x509.MarshalPKIXPublicKey(key)
		if err != nil {
			t.Fatal(err)
		}
		canonical := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
		reparsed, err := ParsePublicKey(canonical)
		if err != nil {
			t.Fatal(err)
		}
		roundTrip, err := x509.MarshalPKIXPublicKey(reparsed)
		if err != nil || !bytes.Equal(der, roundTrip) {
			t.Fatalf("la clé publique change après normalisation : %v", err)
		}
	})
}

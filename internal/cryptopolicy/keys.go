// Package cryptopolicy fixe les tailles minimales des clés utilisées par Pépin.
// Ces gardes portent sur la taille du matériau, pas sur son entropie : les
// secrets doivent toujours être générés par un mécanisme cryptographique sûr.
package cryptopolicy

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"

	"github.com/stephrobert/pepin/internal/i18n"
)

// MinHMACKeyBytes impose 112 bits de matériau secret, sans compter les préfixes
// constants de la dérivation SigV4. Aucun drapeau ne désactive cette garde.
const MinHMACKeyBytes = 14

// ValidateHMACSecret contrôle les octets UTF-8 effectivement passés au HMAC.
// Il ne prétend ni reconnaître une clé aléatoire ni mesurer son entropie.
func ValidateHMACSecret(secret string) error {
	if len(secret) < MinHMACKeyBytes {
		return errors.New(i18n.T(
			"politique cryptographique : secret HMAC inférieur à 112 bits (14 octets)",
			"cryptographic policy: HMAC secret below 112 bits (14 bytes)"))
	}
	return nil
}

// ValidatePublicKey applique les mêmes seuils aux clés TLS et de signature.
func ValidatePublicKey(key any) error {
	switch pub := key.(type) {
	case *rsa.PublicKey:
		if pub != nil && pub.N != nil && pub.N.BitLen() >= 2048 {
			return nil
		}
		return errors.New(i18n.T("politique cryptographique : clé RSA inférieure à 2048 bits", "cryptographic policy: RSA key below 2048 bits"))
	case *ecdsa.PublicKey:
		if pub != nil && pub.Curve != nil && pub.Curve.Params().BitSize >= 224 {
			return nil
		}
		return errors.New(i18n.T("politique cryptographique : clé EC inférieure à 224 bits", "cryptographic policy: EC key below 224 bits"))
	case ed25519.PublicKey:
		if len(pub) == ed25519.PublicKeySize {
			return nil
		}
	}
	return errors.New(i18n.T("politique cryptographique : type de clé non pris en charge", "cryptographic policy: unsupported key type"))
}

// ParsePublicKey accepte une seule clé publique PEM PKIX ou PKCS#1 locale.
// Les références distantes et clés privées ne sont pas des clés déjà contrôlées.
func ParsePublicKey(data []byte) (crypto.PublicKey, error) {
	data = bytes.TrimSpace(data)
	block, rest := pem.Decode(data)
	if block == nil || !bytes.HasPrefix(data, []byte("-----BEGIN "+block.Type+"-----")) ||
		bytes.Count(data, []byte("-----BEGIN ")) != 1 || len(bytes.TrimSpace(rest)) != 0 {
		return nil, errors.New(i18n.T("clé publique PEM absente ou contenu supplémentaire", "missing PEM public key or additional content"))
	}
	var key crypto.PublicKey
	var err error
	switch block.Type {
	case "PUBLIC KEY":
		key, err = x509.ParsePKIXPublicKey(block.Bytes)
	case "RSA PUBLIC KEY":
		key, err = x509.ParsePKCS1PublicKey(block.Bytes)
	default:
		return nil, errors.New(i18n.T("fichier PEM : clé publique requise", "PEM file: public key required"))
	}
	if err != nil {
		return nil, fmt.Errorf(i18n.T("clé publique illisible : %w", "unreadable public key: %w"), err)
	}
	if err := ValidatePublicKey(key); err != nil {
		return nil, err
	}
	return key, nil
}

// Package tlspolicy applique la même politique cryptographique aux clients TLS
// Kubernetes, REST et S3. Il ne transforme pas une connexion HTTP en connexion TLS.
package tlspolicy

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"

	"github.com/stephrobert/pepin/internal/i18n"
)

// Config retourne une configuration indépendante : TLS 1.2+, suites AEAD avec
// ECDHE, et contrôle de robustesse après la vérification normale des certificats.
func Config() *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
		},
		VerifyConnection: verifyConnection,
	}
}

// Transport conserve les réglages HTTP standard, notamment HTTPS_PROXY, sans
// modifier le transport global ni offrir de repli vers une politique TLS faible.
func Transport() (*http.Transport, error) {
	base, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, errors.New(i18n.T(
			"politique TLS : transport HTTP standard incompatible",
			"TLS policy: incompatible standard HTTP transport"))
	}
	transport := base.Clone()
	transport.TLSClientConfig = Config()
	if base.TLSClientConfig != nil {
		transport.TLSClientConfig.RootCAs = base.TLSClientConfig.RootCAs
	}
	return transport, nil
}

// ValidateClientCertificate refuse une clé faible dans toute la chaîne client,
// avant qu'une requête ou un jeton ne soit envoyé.
func ValidateClientCertificate(pair tls.Certificate) error {
	if len(pair.Certificate) == 0 {
		return errors.New(i18n.T("politique TLS : certificat client absent", "TLS policy: missing client certificate"))
	}
	for _, der := range pair.Certificate {
		cert, err := x509.ParseCertificate(der)
		if err != nil {
			return fmt.Errorf(i18n.T("politique TLS, certificat client : %w", "TLS policy, client certificate: %w"), err)
		}
		if err := validatePublicKey(cert.PublicKey); err != nil {
			return err
		}
	}
	return nil
}

func verifyConnection(state tls.ConnectionState) error {
	// Une chaîne forte suffit, si Go a validé plusieurs chemins de confiance.
	// On ne remplace jamais la validation de confiance ou du nom du serveur.
	var refus error
	for _, chain := range state.VerifiedChains {
		if len(chain) == 0 {
			continue
		}
		var err error
		for _, cert := range chain {
			if err = validatePublicKey(cert.PublicKey); err != nil {
				break
			}
		}
		if err == nil {
			return nil
		}
		refus = err
	}
	if refus != nil {
		return refus
	}
	return errors.New(i18n.T("politique TLS : aucune chaîne de confiance vérifiée", "TLS policy: no verified trust chain"))
}

func validatePublicKey(key any) error {
	switch pub := key.(type) {
	case *rsa.PublicKey:
		if pub != nil && pub.N != nil && pub.N.BitLen() >= 2048 {
			return nil
		}
		return errors.New(i18n.T("politique TLS : clé RSA inférieure à 2048 bits", "TLS policy: RSA key below 2048 bits"))
	case *ecdsa.PublicKey:
		if pub != nil && pub.Curve != nil && pub.Curve.Params().BitSize >= 224 {
			return nil
		}
		return errors.New(i18n.T("politique TLS : clé EC inférieure à 224 bits", "TLS policy: EC key below 224 bits"))
	case ed25519.PublicKey:
		if len(pub) == ed25519.PublicKeySize {
			return nil
		}
	}
	return errors.New(i18n.T("politique TLS : type de clé non pris en charge", "TLS policy: unsupported key type"))
}

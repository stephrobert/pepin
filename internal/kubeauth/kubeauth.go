// Package kubeauth transforme un kubeconfig en client HTTP prêt à interroger l'API
// Kubernetes. L'API Kubernetes étant une simple API REST/JSON, cette brique suffit à
// réutiliser le moteur de collecte générique (internal/collect) : aucune dépendance
// client-go (arbre de dépendances considérable pour un outil dont la chaîne
// d'approvisionnement doit rester auditable).
//
// L'authentification retenue est le certificat client (mTLS), mode délivré par OKS ;
// le jeton porteur est accepté en repli. Le client est en LECTURE SEULE par nature :
// c'est le kubeconfig fourni qui porte les droits — un scanner doit être lancé avec un
// kubeconfig restreint (groupe en lecture seule, TTL court), jamais cluster-admin.
package kubeauth

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"time"

	yaml "go.yaml.in/yaml/v3"
)

// Config est le résultat utile d'un kubeconfig : l'URL du serveur d'API et un client
// HTTP configuré pour s'y authentifier.
type Config struct {
	Server string
	Client *http.Client
}

// kubeconfig ne décode que ce qui est nécessaire (contexte courant ignoré : on prend le
// premier cluster/user, ce que produit `oks-cli cluster kubeconfig`).
type kubeconfig struct {
	Clusters []struct {
		Cluster struct {
			Server string `yaml:"server"`
			CAData string `yaml:"certificate-authority-data"`
			CAFile string `yaml:"certificate-authority"`
		} `yaml:"cluster"`
	} `yaml:"clusters"`
	Users []struct {
		User struct {
			ClientCertData string `yaml:"client-certificate-data"`
			ClientKeyData  string `yaml:"client-key-data"`
			Token          string `yaml:"token"`
		} `yaml:"user"`
	} `yaml:"users"`
}

// Load lit un kubeconfig et construit le client d'accès à l'API.
func Load(path string, timeout time.Duration) (Config, error) {
	raw, err := os.ReadFile(path) // #nosec G304 -- chemin fourni explicitement par l'opérateur
	if err != nil {
		return Config{}, fmt.Errorf("kubeconfig : %w", err)
	}
	var kc kubeconfig
	if err := yaml.Unmarshal(raw, &kc); err != nil {
		return Config{}, fmt.Errorf("kubeconfig invalide : %w", err)
	}
	if len(kc.Clusters) == 0 || kc.Clusters[0].Cluster.Server == "" {
		return Config{}, fmt.Errorf("kubeconfig : aucun serveur d'API déclaré")
	}
	cl := kc.Clusters[0].Cluster

	// Autorité de certification du cluster : on ne désactive JAMAIS la vérification TLS —
	// un scanner de posture qui accepterait n'importe quel certificat serait indéfendable.
	pool := x509.NewCertPool()
	ca, err := decodeOrRead(cl.CAData, cl.CAFile)
	if err != nil {
		return Config{}, fmt.Errorf("kubeconfig, CA du cluster : %w", err)
	}
	if len(ca) > 0 && !pool.AppendCertsFromPEM(ca) {
		return Config{}, fmt.Errorf("kubeconfig : CA du cluster illisible")
	}
	tlsCfg := &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}

	var bearer string
	if len(kc.Users) > 0 {
		u := kc.Users[0].User
		bearer = u.Token
		if u.ClientCertData != "" && u.ClientKeyData != "" {
			certPEM, err := base64.StdEncoding.DecodeString(u.ClientCertData)
			if err != nil {
				return Config{}, fmt.Errorf("kubeconfig, certificat client : %w", err)
			}
			keyPEM, err := base64.StdEncoding.DecodeString(u.ClientKeyData)
			if err != nil {
				return Config{}, fmt.Errorf("kubeconfig, clé client : %w", err)
			}
			pair, err := tls.X509KeyPair(certPEM, keyPEM)
			if err != nil {
				return Config{}, fmt.Errorf("kubeconfig, paire certificat/clé : %w", err)
			}
			tlsCfg.Certificates = []tls.Certificate{pair}
		}
	}
	if len(tlsCfg.Certificates) == 0 && bearer == "" {
		return Config{}, fmt.Errorf("kubeconfig : ni certificat client ni jeton — authentification impossible")
	}

	var rt http.RoundTripper = &http.Transport{TLSClientConfig: tlsCfg}
	if bearer != "" {
		rt = bearerRoundTripper{token: bearer, next: rt}
	}
	return Config{Server: cl.Server, Client: &http.Client{Timeout: timeout, Transport: rt}}, nil
}

// decodeOrRead retourne le PEM depuis la donnée base64 inline, sinon depuis le fichier.
func decodeOrRead(data, file string) ([]byte, error) {
	if data != "" {
		return base64.StdEncoding.DecodeString(data)
	}
	if file != "" {
		return os.ReadFile(file) // #nosec G304 -- chemin issu du kubeconfig de l'opérateur
	}
	return nil, nil
}

// bearerRoundTripper ajoute le jeton porteur (repli quand le kubeconfig n'a pas de
// certificat client, ex. un jeton de ServiceAccount).
type bearerRoundTripper struct {
	token string
	next  http.RoundTripper
}

func (b bearerRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	r2 := r.Clone(r.Context())
	r2.Header.Set("Authorization", "Bearer "+b.token)
	return b.next.RoundTrip(r2)
}

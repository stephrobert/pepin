// Package collectkit mutualise la collecte live des providers : exécuter une spec
// YAML embarquée via le moteur générique (internal/collect) puis ajouter le
// stockage objet S3-compatible. Chaque provider ne fournit que ses identifiants,
// son auth, ses variables et son endpoint S3 — aucun code de collecte dupliqué.
package collectkit

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/stephrobert/pepin/internal/collect"
	"github.com/stephrobert/pepin/internal/eimpolicy"
	"github.com/stephrobert/pepin/internal/model"
	"github.com/stephrobert/pepin/internal/objectstorage"
	"github.com/stephrobert/pepin/internal/oks"
)

// httpTimeout borne chaque requête de collecte live : sans borne, un endpoint provider qui
// pend gèle le scan indéfiniment (pas de deadline sur le contexte cobra).
const httpTimeout = 30 * time.Second

// S3 décrit l'accès au stockage objet S3-compatible d'un provider (endpoint vide
// = pas de collecte de buckets).
type S3 struct {
	Endpoint, Region, Key, Secret string
	SSEKMS                        bool // le provider expose une clé client au niveau bucket (SSE-KMS, CLD-CHF-4)
}

// EIM décrit la collecte des politiques EIM *inline* (chaîne à 3 niveaux, hors moteur
// YAML). BaseURL vide = pas de collecte.
type EIM struct{ BaseURL string }

// OKS décrit l'accès à l'API Kubernetes managé (host + auth distincts de l'OAPI ;
// endpoint vide = pas de collecte de clusters).
type OKS struct {
	Endpoint, Region, Key, Secret string
}

// Run exécute la spec de collecte (auth + variables fournies par le provider),
// puis complète avec le stockage objet S3 et les clusters OKS si leurs endpoints
// sont donnés (APIs à part que le moteur YAML générique ne peut pas exprimer).
// `hc` permet à un provider d'imposer son propre client (ex. mTLS d'un kubeconfig) ;
// nil = client par défaut borné par httpTimeout.
func Run(ctx context.Context, name string, spec collect.Spec, auth collect.Auth, vars map[string]string, s3 S3, oksCfg OKS, eimCfg EIM, hc *http.Client) ([]model.Resource, error) {
	if hc == nil {
		hc = &http.Client{Timeout: httpTimeout}
	}
	out, err := collect.Collect(ctx, hc, spec, auth, vars)
	if err != nil {
		return nil, err
	}
	if s3.Endpoint != "" {
		buckets, err := objectstorage.CollectBuckets(ctx, name, s3.Endpoint, s3.Region, s3.Key, s3.Secret, s3.SSEKMS)
		if err != nil {
			return nil, err
		}
		out = append(out, buckets...)
	}
	if eimCfg.BaseURL != "" {
		inline, err := eimpolicy.CollectInlinePolicies(ctx, hc, name, eimCfg.BaseURL, auth)
		if err != nil {
			return nil, err
		}
		out = append(out, inline...)
	}
	if oksCfg.Endpoint != "" {
		clusters, err := oks.CollectClusters(ctx, hc, name, oksCfg.Endpoint, oksCfg.Region, oksCfg.Key, oksCfg.Secret)
		if err != nil {
			// Le Kubernetes managé n'est pas disponible partout (région sans OKS, quota non
			// activé, compte sans droit) : ce n'est PAS une erreur de scan, sinon un tenant
			// sans OKS ne serait plus auditable du tout. On AVERTIT (jamais en silence) et les
			// contrôles Kubernetes restent « non évalués » — jamais faussement conformes.
			fmt.Fprintf(os.Stderr, "pepin: ⚠ collecte OKS ignorée (%v) — contrôles Kubernetes non évalués\n", err)
		} else {
			out = append(out, clusters...)
		}
	}
	return out, nil
}

// FirstNonEmpty retourne la première chaîne non vide (résolution d'identifiants).
func FirstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

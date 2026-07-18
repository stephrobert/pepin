// Package collectkit mutualise la collecte live des providers : exécuter une spec
// YAML embarquée via le moteur générique (internal/collect) puis ajouter le
// stockage objet S3-compatible. Chaque provider ne fournit que ses identifiants,
// son auth, ses variables et son endpoint S3 — aucun code de collecte dupliqué.
package collectkit

import (
	"context"
	"net/http"

	"github.com/stephrobert/pepin/internal/collect"
	"github.com/stephrobert/pepin/internal/model"
	"github.com/stephrobert/pepin/internal/objectstorage"
)

// S3 décrit l'accès au stockage objet S3-compatible d'un provider (endpoint vide
// = pas de collecte de buckets).
type S3 struct {
	Endpoint, Region, Key, Secret string
	SSEKMS                        bool // le provider expose une clé client au niveau bucket (SSE-KMS, CLD-CHF-4)
}

// Run exécute la spec de collecte (auth + variables fournies par le provider),
// puis collecte les buckets S3 si un endpoint est donné.
func Run(ctx context.Context, name string, spec collect.Spec, auth collect.Auth, vars map[string]string, s3 S3) ([]model.Resource, error) {
	out, err := collect.Collect(ctx, &http.Client{}, spec, auth, vars)
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

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
	"github.com/stephrobert/pepin/internal/i18n"
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
func Run(ctx context.Context, name string, spec collect.Spec, auth collect.Auth, vars map[string]string, s3 S3, oksCfg OKS, eimCfg EIM, hc *http.Client) (model.Inventory, error) {
	if hc == nil {
		hc = &http.Client{
			Timeout: httpTimeout,
			// Go ne retire, sur redirection cross-domain, que Authorization, Cookie et
			// WWW-Authenticate. Les en-têtes d'auth des providers souverains n'en font pas
			// partie : X-Auth-Token porte la clé secrète Scaleway, AccessKey/SecretKey les
			// clés Outscale EN CLAIR. Une seule 302 vers un hôte contrôlé suffirait donc à
			// les lui livrer. Aucune API de collecte ne redirige : on ne suit pas.
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	}
	inv, err := collect.Collect(ctx, hc, spec, auth, vars)
	if err != nil {
		return model.Inventory{}, err
	}
	// Les trois collecteurs Go hors moteur YAML suivent EXACTEMENT la même règle que
	// les endpoints déclaratifs : ce qui a été lu est gardé, ce qui a échoué est
	// enregistré, et rien n'est interrompu.
	//
	// C'est la généralisation de ce qui n'existait que pour le Kubernetes managé.
	// Le comportement y était le bon — avertir, ne pas échouer, laisser les contrôles
	// Kubernetes « non évalués » — mais il était CODÉ EN DUR pour ce seul collecteur.
	// Un 403 sur les politiques EIM inline, lui, faisait échouer tout le scan, et un
	// 403 sur le stockage objet aussi. Trois collecteurs, trois comportements
	// différents devant la même situation : c'est cela que l'invariant remplace.
	collectUnit(ctx, &inv, s3.Endpoint != "", "object_storage_bucket", []string{"object_storage_bucket"},
		func() ([]model.Resource, error) {
			return objectstorage.CollectBuckets(ctx, name, s3.Endpoint, s3.Region, s3.Key, s3.Secret, s3.SSEKMS)
		})
	// Les politiques EIM inline alimentent le MÊME type normalisé que les politiques
	// managées (`iam_policy`) : leur unité est distincte — c'est une autre chaîne
	// d'appels, qui peut échouer seule — mais elle marque bien `iam_policy` incomplet.
	// C'est précisément l'incident qui a motivé cette vague : une policy inline
	// « Action:* » échappait à tous les contrôles iam_policy_*.
	collectUnit(ctx, &inv, eimCfg.BaseURL != "", "iam_policy_inline", []string{"iam_policy"},
		func() ([]model.Resource, error) {
			return eimpolicy.CollectInlinePolicies(ctx, hc, name, eimCfg.BaseURL, auth)
		})
	collectUnit(ctx, &inv, oksCfg.Endpoint != "", "kubernetes_cluster", []string{"kubernetes_cluster"},
		func() ([]model.Resource, error) {
			return oks.CollectClusters(ctx, hc, name, oksCfg.Endpoint, oksCfg.Region, oksCfg.Key, oksCfg.Secret)
		})
	return inv, nil
}

// collectUnit exécute UNE unité de collecte Go et enregistre son issue. `enabled`
// faux = le fournisseur n'expose pas ce service : l'unité n'est pas tentée, et une
// unité non tentée n'est pas une unité en échec (rien n'est promis, donc rien n'est
// dû). Elle n'est pas enregistrée du tout : « on n'a jamais regardé » se lit à
// l'absence, pas à une entrée qui prétendrait le contraire.
func collectUnit(ctx context.Context, inv *model.Inventory, enabled bool, unit string, types []string, run func() ([]model.Resource, error)) {
	if !enabled {
		return
	}
	// Un contexte déjà annulé rendrait un échec par unité restante, ce qui est vrai
	// mais bruyant ; l'appel lui-même le signalera de la même façon.
	_ = ctx
	res, err := run()
	inv.Resources = append(inv.Resources, res...)
	u := model.CollectionUnit{Unit: unit, Types: types, Attempted: true, Complete: err == nil}
	if err != nil {
		u.Error, u.Detail = collect.Classify(err)
		// L'avertissement reste : un échec de collecte doit se voir à l'écran, en plus
		// d'être enregistré. Ce qui change, c'est qu'il n'est plus la SEULE trace.
		fmt.Fprintf(os.Stderr, i18n.T(
			"pepin: ⚠ collecte %q incomplète (%s) — les contrôles qui en dépendent seront « non évalués »\n",
			"pepin: ⚠ collection %q is incomplete (%s) — the controls that depend on it will be \"not evaluated\"\n"),
			unit, u.Error)
	}
	inv.Collection.Record(u)
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

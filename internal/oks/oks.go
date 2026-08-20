// Package oks collecte les clusters Kubernetes managés OUTSCALE (OKS). OKS est une
// API DISTINCTE de l'OAPI : host « api.{region}.oks.outscale.com », version « /api/v2 »,
// authentification par DEUX en-têtes (AccessKey / SecretKey). Le moteur YAML générique
// (SigV4, host OAPI unique) ne peut pas l'exprimer : d'où ce collecteur Go dédié, sur le
// modèle d'objectstorage. Il projette chaque cluster dans le type normalisé agnostique
// `kubernetes_cluster`, audité par les mêmes règles Rego quel que soit le fournisseur.
package oks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/stephrobert/pepin/internal/collect"
	"github.com/stephrobert/pepin/internal/i18n"
	"github.com/stephrobert/pepin/internal/model"
)

// maxRespBytes borne le corps d'une reponse : le timeout borne la DUREE, pas la
// taille. Un endpoint hostile ou detourne repond sinon un flux sans fin, que le
// scanner charge entierement en memoire (deni de service du job de CI).
const maxRespBytes = 64 << 20 // 64 Mio

// CollectClusters interroge GET {endpoint}/api/v2/clusters/all (auth bi-en-têtes) et
// projette chaque cluster. `endpoint` peut contenir {region} (substitué). `endpoint` vide
// = pas de collecte OKS.
func CollectClusters(ctx context.Context, hc *http.Client, provider, endpoint, region, accessKey, secretKey string) ([]model.Resource, error) {
	base := strings.ReplaceAll(endpoint, "{region}", region)
	url := strings.TrimRight(base, "/") + "/api/v2/clusters/all"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	// Auth OKS : deux en-têtes en clair (pas de SigV4). Confirmé sur l'API réelle.
	req.Header.Set("AccessKey", accessKey)
	req.Header.Set("SecretKey", secretKey)
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	// La requête telle qu'elle a été RÉELLEMENT émise, lue après réponse : une
	// provenance ne nomme jamais un appel qui n'a pas eu lieu (cf. model.Provenance).
	called := req.Method + " " + req.URL.Scheme + "://" + req.URL.Host + req.URL.Path
	body, rerr := io.ReadAll(io.LimitReader(resp.Body, maxRespBytes))
	if rerr != nil {
		return nil, fmt.Errorf(i18n.T("lecture de la reponse OKS : %w", "reading the OKS response: %w"), rerr)
	}
	if resp.StatusCode >= 300 {
		// Erreur TYPÉE : c'est le statut qui range l'échec dans sa classe (droits
		// insuffisants, service indisponible), et c'est cette classe que l'état de
		// collecte publie.
		return nil, &collect.HTTPError{Status: resp.StatusCode, Call: called, Body: strings.TrimSpace(string(body))}
	}
	var doc struct {
		Clusters []map[string]any `json:"Clusters"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf(i18n.T("OKS réponse JSON invalide : %w", "invalid OKS JSON response: %w"), err)
	}
	out := make([]model.Resource, 0, len(doc.Clusters))
	for _, c := range doc.Clusters {
		out = append(out, mapCluster(provider, region, called, c))
	}
	return out, nil
}

// ClusterAttributes énumère les attributs communs qu'un cluster managé collecté
// par cette API peut porter. Déclaré à côté du projecteur qui les pose, et lu par
// le catalogue de l'inventaire (genprovider.InventoryCatalogue).
func ClusterAttributes() []string {
	return []string{"admin_whitelist", "auto_upgrade", "control_plane_multi_az", "deletion_protection", "name", "version"}
}

// mapCluster projette un cluster OKS (champs natifs de l'API v2) vers le modèle normalisé.
// Un attribut ABSENT n'est pas posé : les règles le traitent alors comme conforme par
// défaut (object.get(..., true)), donc pas de faux échec.
//
// Chaque attribut est ATTESTÉ : d'où il vient (`called`, la requête réellement émise),
// quel champ natif a été lu, et si la réponse le portait. L'attestation est posée même
// quand le champ est absent — « on a cherché `cp_multi_az`, la réponse ne l'avait pas »
// n'est pas « on n'a jamais regardé ». Ce collecteur était l'un des deux qui n'attestaient
// rien du tout.
func mapCluster(provider, region, called string, c map[string]any) model.Resource {
	name, _ := c["name"].(string)
	id, _ := c["id"].(string)
	attrs := map[string]any{"name": name}
	var prov model.Provenance
	attest := func(attr, path string, observed, derived bool) {
		prov.Attest(attr, model.Attestation{
			Origin: model.OriginAPI, Source: called, Path: path,
			Observed: observed, Derived: derived,
		})
	}
	attest("name", "name", true, false)
	// Les affectations restent LITTÉRALES (`attrs["nom"] = …`), une par attribut.
	// Une boucle serait plus courte, mais la documentation de couverture DÉRIVE la
	// liste des attributs de ce collecteur en lisant ce fichier
	// (internal/docgen/descriptors.go) : la factoriser rendrait la page de couverture
	// silencieusement fausse, c'est-à-dire exactement le genre de panne qui se mesure
	// elle-même au lieu de mesurer son sujet.
	//
	// admin_whitelist : []string de CIDR autorisés à joindre l'API server (K8S-1).
	v, ok := c["admin_whitelist"]
	if ok {
		attrs["admin_whitelist"] = v
	}
	attest("admin_whitelist", "admin_whitelist", ok, false)
	// cp_multi_az -> control_plane_multi_az (K8S-2, HA du plan de contrôle).
	v, ok = c["cp_multi_az"]
	if ok {
		attrs["control_plane_multi_az"] = v
	}
	attest("control_plane_multi_az", "cp_multi_az", ok, false)
	// disable_api_termination -> deletion_protection (K8S-3).
	v, ok = c["disable_api_termination"]
	if ok {
		attrs["deletion_protection"] = v
	}
	attest("deletion_protection", "disable_api_termination", ok, false)
	v, ok = c["version"]
	if ok {
		attrs["version"] = v
	}
	attest("version", "version", ok, false)
	// auto_upgrade DÉRIVÉ de auto_maintenances.minor_upgrade_maintenance.enabled (K8S-3).
	const autoPath = "auto_maintenances.minor_upgrade_maintenance.enabled"
	found := false
	if am, ok := c["auto_maintenances"].(map[string]any); ok {
		if mu, ok := am["minor_upgrade_maintenance"].(map[string]any); ok {
			if en, ok := mu["enabled"]; ok {
				attrs["auto_upgrade"] = en
				found = true
			}
		}
	}
	attest("auto_upgrade", autoPath, found, found)
	return model.Resource{
		Provider:   provider,
		Type:       "kubernetes_cluster",
		ID:         id,
		Name:       name,
		Region:     region,
		Attributes: attrs,
		Provenance: prov,
	}
}

// Package eimpolicy collecte les politiques EIM *inline* d'Outscale — celles attachées
// directement à un utilisateur OU À UN GROUPE (donc héritées par tous ses membres), par opposition aux politiques managées (ReadPolicies) et
// liées (ReadLinkedPolicies) déjà collectées par la spec YAML.
//
// Pourquoi du Go et pas une spec YAML : la chaîne est à TROIS niveaux —
// ReadUsers → ReadUserPolicies (qui ne renvoie que des NOMS) → ReadUserPolicy (qui renvoie
// le document). Le `for_each` du moteur générique est mono-niveau ; il ne peut pas exprimer
// cette jointure. Contrat officiel (osc-api) : ReadUserPoliciesResponse.PolicyNames[],
// ReadUserPolicyResponse.PolicyDocument (chaîne JSON).
//
// Enjeu : une politique inline « Action:* / Resource:* » accorde les pleins pouvoirs et
// ÉCHAPPAIT jusqu'ici à tous les contrôles iam_policy_* (faux négatif). Les ressources
// produites portent le même type normalisé `iam_policy` et le même attribut `statements`
// que les politiques managées : les règles existantes les évaluent sans modification.
package eimpolicy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/stephrobert/pepin/internal/collect"
	"github.com/stephrobert/pepin/internal/model"
)

// pageSize borne chaque page : l'OAPI plafonne de toute façon à 100 (MaxResultsLimit).
const pageSize = 100

// maxPages borne les boucles de pagination. Sans elle, un endpoint qui renvoie
// des entrees sans nom avec HasMoreItems=true fait tourner la boucle sans fin :
// l'offset n'avancait que sur les entrees RETENUES, jamais sur celles recues.
const maxPages = 1000

// maxRespBytes borne le corps d'une reponse : le timeout borne la DUREE, pas la
// taille. Un endpoint hostile ou detourne repond sinon un flux sans fin, que le
// scanner charge entierement en memoire (deni de service du job de CI).
const maxRespBytes = 64 << 20 // 64 Mio

// CollectInlinePolicies parcourt les utilisateurs EIM et projette chaque politique inline
// en ressource `iam_policy` (scope: inline). `baseURL` est l'URL de l'OAPI déjà substituée.
func CollectInlinePolicies(ctx context.Context, hc *http.Client, provider, baseURL string, auth collect.Auth) ([]model.Resource, error) {
	// L'OAPI plafonne toute page à 100 items (MaxResultsLimit) et signale la suite par
	// HasMoreItems : sans pagination, les utilisateurs 101+ — et donc leurs politiques
	// inline — échapperaient EN SILENCE, c'est-à-dire précisément le faux négatif que ce
	// collecteur existe pour fermer.
	users, err := readAllUsers(ctx, hc, auth, baseURL)
	if err != nil {
		return nil, err
	}

	out, err := collectGroupPolicies(ctx, hc, provider, baseURL, auth)
	if err != nil {
		return nil, err
	}
	for _, user := range users {
		policyNames, err := readAllUserPolicies(ctx, hc, auth, baseURL, user)
		if err != nil {
			return nil, err
		}
		for _, name := range policyNames {
			var doc struct {
				PolicyDocument string `json:"PolicyDocument"`
			}
			b := mustJSON(map[string]string{"UserName": user, "PolicyName": name})
			if err := post(ctx, hc, auth, baseURL+"/ReadUserPolicy", b, &doc); err != nil {
				return nil, fmt.Errorf("ReadUserPolicy(%s/%s) : %w", user, name, err)
			}
			id := user + "/" + name
			out = append(out, model.Resource{
				Provider: provider,
				Type:     "iam_policy",
				ID:       id,
				Name:     name,
				Attributes: map[string]any{
					"policy_name": name,
					"policy_id":   id,
					"owner_user":  user,
					"scope":       "inline",
					"statements":  collect.IAMPolicyStatements(doc.PolicyDocument),
				},
			})
		}
	}
	return out, nil
}

// collectGroupPolicies projette les politiques inline attachées aux GROUPES. Une policy
// posée sur un groupe est héritée par tous ses membres : l'ignorer laissait passer un
// `Action:*` accordé à toute une équipe. Chaîne plus courte que pour les utilisateurs :
// ReadUserGroupPolicies renvoie déjà `Policies[] = InlinePolicy{Name, Body}` (contrat
// vérifié), donc le document est là — inutile d'appeler ReadUserGroupPolicy par policy.
func collectGroupPolicies(ctx context.Context, hc *http.Client, provider, baseURL string, auth collect.Auth) ([]model.Resource, error) {
	groups, err := readAllUserGroups(ctx, hc, auth, baseURL)
	if err != nil {
		return nil, err
	}
	var out []model.Resource
	for _, g := range groups {
		policies, err := readAllGroupPolicies(ctx, hc, auth, baseURL, g)
		if err != nil {
			return nil, err
		}
		for _, pol := range policies {
			id := "group/" + g + "/" + pol.Name
			out = append(out, model.Resource{
				Provider: provider,
				Type:     "iam_policy",
				ID:       id,
				Name:     pol.Name,
				Attributes: map[string]any{
					"policy_name": pol.Name,
					"policy_id":   id,
					"owner_group": g,
					"scope":       "inline_group",
					"statements":  collect.IAMPolicyStatements(pol.Body),
				},
			})
		}
	}
	return out, nil
}

// inlinePolicy = schéma InlinePolicy du contrat.
type inlinePolicy struct {
	Name string `json:"Name"`
	Body string `json:"Body"`
}

// readAllUserGroups pagine ReadUserGroups.
func readAllUserGroups(ctx context.Context, hc *http.Client, auth collect.Auth, baseURL string) ([]string, error) {
	var out []string
	seen := 0 // items REÇUS : c'est l'offset de l'API, pas le nombre de noms retenus.
	for p := 0; p < maxPages; p++ {
		var page struct {
			UserGroups []struct {
				Name string `json:"Name"`
			} `json:"UserGroups"`
			HasMoreItems bool `json:"HasMoreItems"`
		}
		body := mustJSON(map[string]any{"FirstItem": seen, "ResultsPerPage": pageSize})
		if err := post(ctx, hc, auth, baseURL+"/ReadUserGroups", body, &page); err != nil {
			return nil, fmt.Errorf("ReadUserGroups : %w", err)
		}
		for _, g := range page.UserGroups {
			if g.Name != "" {
				out = append(out, g.Name)
			}
		}
		seen += len(page.UserGroups)
		if !page.HasMoreItems || len(page.UserGroups) == 0 {
			return out, nil
		}
	}
	return nil, fmt.Errorf("ReadUserGroups : borne de %d pages atteinte", maxPages)
}

// readAllGroupPolicies pagine ReadUserGroupPolicies pour un groupe.
func readAllGroupPolicies(ctx context.Context, hc *http.Client, auth collect.Auth, baseURL, group string) ([]inlinePolicy, error) {
	var out []inlinePolicy
	for {
		var page struct {
			Policies     []inlinePolicy `json:"Policies"`
			HasMoreItems bool           `json:"HasMoreItems"`
		}
		body := mustJSON(map[string]any{"UserGroupName": group, "FirstItem": len(out), "ResultsPerPage": pageSize})
		if err := post(ctx, hc, auth, baseURL+"/ReadUserGroupPolicies", body, &page); err != nil {
			return nil, fmt.Errorf("ReadUserGroupPolicies(%s) : %w", group, err)
		}
		out = append(out, page.Policies...)
		if !page.HasMoreItems || len(page.Policies) == 0 {
			return out, nil
		}
	}
}

// readAllUsers pagine ReadUsers (offset FirstItem 0-basé, arrêt sur HasMoreItems).
func readAllUsers(ctx context.Context, hc *http.Client, auth collect.Auth, baseURL string) ([]string, error) {
	var out []string
	seen := 0 // items REÇUS : c'est l'offset de l'API, pas le nombre de noms retenus.
	for p := 0; p < maxPages; p++ {
		var page struct {
			Users []struct {
				UserName string `json:"UserName"`
			} `json:"Users"`
			HasMoreItems bool `json:"HasMoreItems"`
		}
		body := mustJSON(map[string]any{"FirstItem": seen, "ResultsPerPage": pageSize})
		if err := post(ctx, hc, auth, baseURL+"/ReadUsers", body, &page); err != nil {
			return nil, fmt.Errorf("ReadUsers : %w", err)
		}
		for _, u := range page.Users {
			if u.UserName != "" {
				out = append(out, u.UserName)
			}
		}
		seen += len(page.Users)
		if !page.HasMoreItems || len(page.Users) == 0 {
			return out, nil
		}
	}
	return nil, fmt.Errorf("ReadUsers : borne de %d pages atteinte", maxPages)
}

// readAllUserPolicies pagine ReadUserPolicies pour un utilisateur.
func readAllUserPolicies(ctx context.Context, hc *http.Client, auth collect.Auth, baseURL, user string) ([]string, error) {
	var out []string
	for {
		var page struct {
			PolicyNames  []string `json:"PolicyNames"`
			HasMoreItems bool     `json:"HasMoreItems"`
		}
		body := mustJSON(map[string]any{"UserName": user, "FirstItem": len(out), "ResultsPerPage": pageSize})
		if err := post(ctx, hc, auth, baseURL+"/ReadUserPolicies", body, &page); err != nil {
			return nil, fmt.Errorf("ReadUserPolicies(%s) : %w", user, err)
		}
		out = append(out, page.PolicyNames...)
		if !page.HasMoreItems || len(page.PolicyNames) == 0 {
			return out, nil
		}
	}
}

// mustJSON sérialise un body de requête. json.Marshal d'une map de scalaires n'échoue pas,
// et cet encodage ÉVITE toute injection : une valeur venant de l'API (nom d'utilisateur)
// est échappée, là où un fmt.Sprintf laisserait fabriquer des champs arbitraires.
func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// post exécute un appel OAPI (POST + body JSON, signé) et décode la réponse.
func post(ctx context.Context, hc *http.Client, auth collect.Auth, url, body string, into any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if auth != nil {
		if err := auth.Apply(req); err != nil {
			return err
		}
	}
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, rerr := io.ReadAll(io.LimitReader(resp.Body, maxRespBytes))
	if rerr != nil {
		return fmt.Errorf("lecture de la reponse EIM : %w", rerr)
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d : %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return json.Unmarshal(raw, into)
}

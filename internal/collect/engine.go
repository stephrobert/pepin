// Package collect est un moteur de collecte GÉNÉRIQUE piloté par configuration
// (YAML). Au lieu d'écrire un collecteur Go par provider/ressource, on DÉCLARE
// dans une spec : l'endpoint REST, le chemin de la liste (imbrication possible),
// la projection champ_natif → attribut commun, des transforms nommés et la
// pagination. Le moteur fait l'appel HTTP, lit le JSON et produit des
// `model.Resource` normalisés.
//
// Objectif : « peu de code par provider » — seule l'AUTH reste du Go ; toute la
// cartographie des ressources est en YAML.
package collect

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/stephrobert/pepin/internal/model"
)

// Auth applique l'authentification à une requête HTTP (seule partie Go
// spécifique au provider : en-tête de jeton, signature SigV4/HMAC…).
type Auth interface{ Apply(req *http.Request) error }

// HeaderAuth : authentification par en-tête (ex. Scaleway X-Auth-Token: <secret>).
type HeaderAuth struct{ Header, Value string }

// Apply ajoute l'en-tête d'authentification.
func (h HeaderAuth) Apply(req *http.Request) error { req.Header.Set(h.Header, h.Value); return nil }

// Paging décrit la pagination par numéro de page (?<param>=N&<size_param>=<size>).
type Paging struct {
	Style     string `yaml:"style"`      // "page" | "" (aucune)
	Param     string `yaml:"param"`      // ex. page
	SizeParam string `yaml:"size_param"` // ex. page_size
	Size      int    `yaml:"size"`       // ex. 100
}

// ForEach déclare une JOINTURE parent→enfant par endpoint : on liste d'abord les
// parents, puis on appelle le chemin de la ressource une fois PAR parent (variable
// {<as>.<champ>} substituée, ex. {sg.id}). Chaque item enfant reçoit `_parent` =
// l'item parent (pour mapper p. ex. security_group_id via _parent.id).
type ForEach struct {
	Path   string  `yaml:"path"`   // endpoint listant les parents
	Items  string  `yaml:"items"`  // chemin du tableau de parents
	As     string  `yaml:"as"`     // préfixe de variable (ex. "sg" -> {sg.id}, {p.Orn})
	Method string  `yaml:"method"` // verbe HTTP de la liste parente (défaut GET)
	Body   string  `yaml:"body"`   // corps de la requête parente (POST)
	Paging *Paging `yaml:"paging"` // pagination de la liste parente (optionnel)
}

// ResourceSpec déclare la collecte d'UN type de ressource (en YAML).
type ResourceSpec struct {
	Type       string            `yaml:"type"`       // type normalisé commun
	Path       string            `yaml:"path"`       // endpoint relatif
	BaseURL    string            `yaml:"base_url"`   // override du base_url global (multi-API : OAPI vs OKS…) ; {vars} substituées
	Method     string            `yaml:"method"`     // verbe HTTP (défaut GET ; POST pour OAPI Outscale)
	Body       string            `yaml:"body"`       // corps JSON de la requête (POST), signé par l'auth
	Items      string            `yaml:"items"`      // chemin du tableau ; `[*]` = aplatir (ex. security_groups[*].rules[*])
	ID         string            `yaml:"id"`         // attribut identifiant
	Map        map[string]string `yaml:"map"`        // attribut_commun -> chemin JSON dans l'item (a.b, tableau via a.0.b, _parent.x, coalesce via "x||y")
	Transforms map[string]any    `yaml:"transforms"` // attribut -> transform (nom) | table de remplacement
	Const      map[string]any    `yaml:"const"`      // attributs littéraux ajoutés tels quels
	Aggregate  string            `yaml:"aggregate"`  // si défini : produit 1 ressource dont cet attribut = nombre d'items (ex. rule_count)
	Paging     *Paging           `yaml:"paging"`
	ForEach    *ForEach          `yaml:"for_each"` // jointure parent->enfant (optionnel)
}

// Spec est la configuration de collecte d'un provider.
type Spec struct {
	Provider  string         `yaml:"provider"`
	BaseURL   string         `yaml:"base_url"`
	Resources []ResourceSpec `yaml:"resources"`
}

// Collect exécute toute la spec et retourne les ressources normalisées. `vars`
// substitue les variables de chemin `{clef}` (ex. {zone}, {org}).
func Collect(ctx context.Context, hc *http.Client, spec Spec, auth Auth, vars map[string]string) ([]model.Resource, error) {
	spec.BaseURL = subst(spec.BaseURL, vars) // base_url peut contenir {zone}/{region}
	var out []model.Resource
	for _, r := range spec.Resources {
		rs, err := collectResource(ctx, hc, spec, auth, r, vars)
		if err != nil {
			return nil, fmt.Errorf("collecte %s : %w", r.Type, err)
		}
		out = append(out, rs...)
	}
	return out, nil
}

// subst remplace les variables de chemin {clef} par leur valeur.
func subst(s string, vars map[string]string) string {
	for k, v := range vars {
		s = strings.ReplaceAll(s, "{"+k+"}", v)
	}
	return s
}

func collectResource(ctx context.Context, hc *http.Client, spec Spec, auth Auth, r ResourceSpec, vars map[string]string) ([]model.Resource, error) {
	if r.ForEach != nil {
		return collectForEach(ctx, hc, spec, auth, r, vars)
	}
	items, err := fetchItems(ctx, hc, auth, resourceURL(spec, r, r.Path, vars), r.Items, r.Paging, r.Method, subst(r.Body, vars))
	if err != nil {
		return nil, err
	}
	if r.Aggregate != "" {
		attrs := map[string]any{r.Aggregate: int64(len(items))}
		for k, v := range r.Const {
			attrs[k] = v
		}
		id, _ := attrs[r.ID].(string)
		return []model.Resource{{Provider: spec.Provider, Type: r.Type, ID: id, Name: id, Region: vars["region"], Attributes: attrs}}, nil
	}
	return mapItems(spec, r, items, vars), nil
}

// resourceURL construit l'URL complète d'une ressource : base_url de la ressource
// (override, multi-API) sinon base_url global, + le chemin ; variables substituées.
func resourceURL(spec Spec, r ResourceSpec, path string, vars map[string]string) string {
	base := spec.BaseURL // déjà substitué dans Collect
	if r.BaseURL != "" {
		base = subst(r.BaseURL, vars)
	}
	return base + subst(path, vars)
}

// collectForEach liste les parents puis collecte la ressource une fois par parent
// (variables {<as>.<champ>} + `_parent`).
func collectForEach(ctx context.Context, hc *http.Client, spec Spec, auth Auth, r ResourceSpec, vars map[string]string) ([]model.Resource, error) {
	fe := r.ForEach
	parents, err := fetchItems(ctx, hc, auth, resourceURL(spec, r, fe.Path, vars), fe.Items, fe.Paging, fe.Method, subst(fe.Body, vars))
	if err != nil {
		return nil, fmt.Errorf("liste parente %s : %w", fe.Path, err)
	}
	var out []model.Resource
	for _, p := range parents {
		pm, _ := p.(map[string]any)
		v2 := mergeVars(vars, fe.As, pm)
		items, err := fetchItems(ctx, hc, auth, resourceURL(spec, r, r.Path, v2), r.Items, r.Paging, r.Method, subst(r.Body, v2))
		if err != nil {
			return nil, err
		}
		for i := range items {
			items[i] = withParent(items[i], pm)
		}
		out = append(out, mapItems(spec, r, items, v2)...)
	}
	return out, nil
}

// fetchItems récupère (avec pagination) le tableau d'items d'un endpoint (URL complète).
func fetchItems(ctx context.Context, hc *http.Client, auth Auth, fullURL, itemsPath string, paging *Paging, method, body string) ([]any, error) {
	var items []any
	for page := 1; ; page++ {
		doc, err := fetch(ctx, hc, fullURL, auth, paging, page, method, body)
		if err != nil {
			return nil, err
		}
		batch := extractItems(doc, itemsPath)
		items = append(items, batch...)
		if paging == nil || paging.Style != "page" || len(batch) < paging.Size || len(batch) == 0 {
			break
		}
	}
	return items, nil
}

// mapItems projette les items bruts vers des ressources normalisées (map +
// transforms). La région de collecte (vars["region"]) est propagée sur chaque
// ressource pour les contrôles de localisation (souveraineté, CLD-GVN-3).
func mapItems(spec Spec, r ResourceSpec, items []any, vars map[string]string) []model.Resource {
	region := vars["region"]
	out := make([]model.Resource, 0, len(items))
	for _, it := range items {
		attrs := Project(it, r.Map, r.Transforms)
		for k, v := range r.Const {
			attrs[k] = v
		}
		id, _ := attrs[r.ID].(string)
		out = append(out, model.Resource{Provider: spec.Provider, Type: r.Type, ID: id, Name: id, Region: region, Attributes: attrs})
	}
	return out
}

// Project applique une projection DÉCLARATIVE (map attribut_commun -> chemin dans
// l'objet source, + transforms nommés) à un objet (item JSON d'une API ou bloc
// `values` d'une ressource Terraform) et retourne les attributs normalisés. C'est
// le cœur commun à la collecte live ET au mapping Terraform (mêmes specs YAML).
func Project(item any, mapping map[string]string, transforms map[string]any) map[string]any {
	attrs := make(map[string]any, len(mapping))
	for attr, path := range mapping {
		v := lookupCoalesce(item, path)
		if t, ok := transforms[attr]; ok {
			v = applyTransform(v, t)
		}
		if v == nil {
			v = ""
		}
		attrs[attr] = v
	}
	return attrs
}

// mergeVars copie vars et y ajoute {<as>.<champ>} pour chaque champ scalaire du parent.
func mergeVars(vars map[string]string, as string, parent map[string]any) map[string]string {
	v2 := make(map[string]string, len(vars)+len(parent))
	for k, val := range vars {
		v2[k] = val
	}
	for k, val := range parent {
		if _, isMap := val.(map[string]any); isMap {
			continue
		}
		if _, isArr := val.([]any); isArr {
			continue
		}
		v2[as+"."+k] = toStr(val)
	}
	return v2
}

// withParent attache `_parent` à un item map (copie défensive).
func withParent(it any, parent map[string]any) any {
	m, ok := it.(map[string]any)
	if !ok {
		return it
	}
	m2 := make(map[string]any, len(m)+1)
	for k, v := range m {
		m2[k] = v
	}
	m2["_parent"] = parent
	return m2
}

// lookupCoalesce résout un chemin, avec coalescence "a||b" (premier non-nul).
func lookupCoalesce(it any, path string) any {
	if !strings.Contains(path, "||") {
		return lookup(it, path)
	}
	for _, p := range strings.Split(path, "||") {
		if v := lookup(it, strings.TrimSpace(p)); v != nil {
			return v
		}
	}
	return nil
}

func fetch(ctx context.Context, hc *http.Client, rawURL string, auth Auth, p *Paging, page int, method, body string) (any, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if p != nil && p.Style == "page" {
		q := u.Query()
		q.Set(p.Param, strconv.Itoa(page))
		if p.SizeParam != "" {
			q.Set(p.SizeParam, strconv.Itoa(p.Size))
		}
		u.RawQuery = q.Encode()
	}
	if method == "" {
		method = http.MethodGet
	}
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		return nil, err
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if auth != nil {
		if err := auth.Apply(req); err != nil {
			return nil, err
		}
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d : %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	var doc any
	if err := json.Unmarshal(respBody, &doc); err != nil {
		return nil, fmt.Errorf("réponse JSON invalide : %w", err)
	}
	return doc, nil
}

// ExtractItems expose extractItems pour le mapping Terraform (internal/tfmap) :
// éclater un bloc répété d'une ressource (ex. "inbound_rule[*]") en items, chaque
// item recevant `_parent` = l'objet conteneur. Réutilise le moteur de la collecte.
func ExtractItems(doc any, path string) []any { return extractItems(doc, path) }

// splitPortRange déduit (from, to) d'un port unique (number) ou d'une plage
// "22-23" (string). Retourne (0,0) si vide/illisible.
func splitPortRange(v any) (int64, int64) {
	switch x := v.(type) {
	case float64:
		return int64(x), int64(x)
	case int64:
		return x, x
	case string:
		from, to, ok := splitRange(x)
		if !ok {
			return 0, 0
		}
		return from, to
	}
	return 0, 0
}

// splitRange découpe "22" ou "22-23" en (from, to).
func splitRange(s string) (int64, int64, bool) {
	parts := strings.SplitN(strings.TrimSpace(s), "-", 2)
	from, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
	if err != nil {
		return 0, 0, false
	}
	if len(parts) == 1 {
		return from, from, true
	}
	to, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
	if err != nil {
		return from, from, true
	}
	return from, to, true
}

// awsPolicyStatements parse un document de politique au format JSON de statements
// (string {Statement:[{Effect, Action, Resource, NotAction, NotResource}]}) en
// statements normalisés [{effect, actions[], resources[], not_action[],
// not_resource[]}]. Helper COMMUN aux providers dont les politiques suivent ce
// format de statements (ex. Outscale EIM).
func awsPolicyStatements(v any) []any {
	doc, ok := v.(string)
	if !ok || doc == "" {
		return []any{}
	}
	var parsed struct {
		Statement json.RawMessage `json:"Statement"`
	}
	if json.Unmarshal([]byte(doc), &parsed) != nil || len(parsed.Statement) == 0 {
		return []any{}
	}
	// La grammaire AWS/IAM admet Statement comme TABLEAU ou comme OBJET unique : sans gérer le
	// second cas, json.Unmarshal échouait et renvoyait [] -> toutes les règles iam_policy
	// passaient au vert en silence (faux négatif). On accepte les deux formes.
	var stmts []map[string]any
	if json.Unmarshal(parsed.Statement, &stmts) != nil {
		var one map[string]any
		if json.Unmarshal(parsed.Statement, &one) != nil {
			return []any{}
		}
		stmts = []map[string]any{one}
	}
	out := make([]any, 0, len(stmts))
	for _, st := range stmts {
		out = append(out, map[string]any{
			"effect":       st["Effect"],
			"actions":      toAnyList(st["Action"]),
			"resources":    toAnyList(st["Resource"]),
			"not_action":   toAnyList(st["NotAction"]),
			"not_resource": toAnyList(st["NotResource"]),
		})
	}
	return out
}

// toAnyList normalise une valeur Action/Resource (string OU []string) en []any.
func toAnyList(v any) []any {
	switch x := v.(type) {
	case string:
		return []any{x}
	case []any:
		return x
	}
	return nil
}

// extractItems résout le chemin de liste. `[*]` aplatit un tableau ; sur un
// chemin imbriqué (a[*].b[*]), chaque feuille reçoit `_parent` = l'objet
// conteneur immédiat (pour mapper p. ex. security_group_id depuis le SG parent).
func extractItems(doc any, path string) []any {
	if path == "" {
		return nil
	}
	if path == "." { // l'objet racine de la réponse est lui-même l'item (GET …/{id})
		return []any{doc}
	}
	if !strings.Contains(path, "[*]") {
		switch v := lookup(doc, path).(type) {
		case []any:
			return v
		case map[string]any:
			return []any{v} // objet unique (ex. ReadPolicyVersion.PolicyVersion) -> 1 item
		}
		return nil
	}
	var out []any
	gather(doc, strings.Split(path, "."), nil, &out)
	return out
}

func gather(v any, segs []string, parent map[string]any, out *[]any) {
	if len(segs) == 0 {
		if m, ok := v.(map[string]any); ok && parent != nil {
			m2 := make(map[string]any, len(m)+1)
			for k, vv := range m {
				m2[k] = vv
			}
			m2["_parent"] = parent
			*out = append(*out, m2)
			return
		}
		*out = append(*out, v)
		return
	}
	m, ok := v.(map[string]any)
	if !ok {
		return
	}
	seg := segs[0]
	if name, star := strings.TrimSuffix(seg, "[*]"), strings.HasSuffix(seg, "[*]"); star {
		arr, _ := m[name].([]any)
		for _, el := range arr {
			gather(el, segs[1:], m, out)
		}
	} else {
		gather(m[name], segs[1:], parent, out)
	}
}

// lookup résout un chemin pointé (ex. "a.b", "_parent.id", "public_ips.0.address")
// dans une valeur JSON. Un segment numérique indexe un tableau.
func lookup(v any, path string) any {
	if path == "" {
		return v
	}
	for _, key := range strings.Split(path, ".") {
		switch c := v.(type) {
		case map[string]any:
			v = c[key]
		case []any:
			i, err := strconv.Atoi(key)
			if err != nil || i < 0 || i >= len(c) {
				return nil
			}
			v = c[i]
		default:
			return nil
		}
	}
	return v
}

// applyTransform applique un transform : nom prédéfini (lower|upper|first|list|
// to_int) ou table de remplacement de valeurs (map from->to).
func applyTransform(v any, spec any) any {
	switch t := spec.(type) {
	case []any:
		// chaînage : applique les transforms en séquence.
		for _, step := range t {
			v = applyTransform(v, step)
		}
		return v
	case string:
		if want, ok := strings.CutPrefix(t, "equals:"); ok {
			return toStr(v) == want // booléen : la valeur source vaut-elle `want`
		}
		if def, ok := strings.CutPrefix(t, "default:"); ok {
			if toStr(v) == "" {
				return def
			}
			return v
		}
		if field, ok := strings.CutPrefix(t, "pluck:"); ok {
			arr, _ := v.([]any)
			out := make([]any, 0, len(arr))
			for _, el := range arr {
				if m, ok := el.(map[string]any); ok {
					out = append(out, m[field])
				}
			}
			return out
		}
		if sub, ok := strings.CutPrefix(t, "contains:"); ok {
			// Booléen : la représentation de la valeur contient-elle le motif.
			// Sert à DÉRIVER un drapeau d'une structure imbriquée (ex. une policy
			// CEL contient `source_ip` ⇒ restriction par IP ; `duration(` ⇒ borne
			// de durée de vie). Heuristique d'ancrage, documentée dans le contrat.
			return strings.Contains(toStr(v), sub)
		}
		switch t {
		case "lower":
			return strings.ToLower(toStr(v))
		case "upper":
			return strings.ToUpper(toStr(v))
		case "first":
			if arr, ok := v.([]any); ok && len(arr) > 0 {
				return arr[0]
			}
			return v
		case "range_from":
			from, _ := splitPortRange(v)
			return from
		case "range_to":
			_, to := splitPortRange(v)
			return to
		case "awspolicy":
			return awsPolicyStatements(v)
		case "list":
			if arr, ok := v.([]any); ok {
				return arr // idempotent : déjà une liste
			}
			if v == nil || v == "" {
				return []any{}
			}
			return []any{v}
		case "kv":
			// tags de gouvernance -> [{key,value}]. Accepte deux formes natives :
			// []string "clé=valeur" (Scaleway) ou map[clé]valeur (labels Exoscale).
			if m, ok := v.(map[string]any); ok {
				out := make([]any, 0, len(m))
				for k, val := range m {
					out = append(out, map[string]any{"key": k, "value": toStr(val)})
				}
				return out
			}
			arr, ok := v.([]any)
			if !ok {
				return []any{}
			}
			out := make([]any, 0, len(arr))
			for _, t := range arr {
				if m, ok := t.(map[string]any); ok {
					// tag objet : {Key,Value} (Outscale, PascalCase) ou {key,value}.
					k := firstNonNil(m["key"], m["Key"])
					val := firstNonNil(m["value"], m["Value"])
					out = append(out, map[string]any{"key": toStr(k), "value": toStr(val)})
					continue
				}
				k, val, _ := strings.Cut(toStr(t), "=")
				out = append(out, map[string]any{"key": k, "value": val})
			}
			return out
		case "to_int":
			if n, err := strconv.ParseInt(toStr(v), 10, 64); err == nil {
				return n
			}
			return v
		case "nonempty":
			// Booléen : la valeur source est-elle renseignée (présence d'un endpoint,
			// d'une liste non vide…). Sert à DÉRIVER un drapeau « activé » d'un champ
			// dont la simple présence vaut activation (ex. audit.endpoint SKS).
			switch x := v.(type) {
			case nil:
				return false
			case string:
				return x != ""
			case []any:
				return len(x) > 0
			case map[string]any:
				return len(x) > 0
			default:
				return true
			}
		}
	case map[string]any:
		if mapped, ok := t[toStr(v)]; ok {
			return mapped
		}
	}
	return v
}

// firstNonNil retourne la première valeur non nil.
func firstNonNil(vals ...any) any {
	for _, v := range vals {
		if v != nil {
			return v
		}
	}
	return nil
}

func toStr(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", x)
	}
}

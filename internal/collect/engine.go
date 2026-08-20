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

	"github.com/stephrobert/pepin/internal/i18n"
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
	Style      string `yaml:"style"`       // "page" | "token" | "" (aucune)
	Param      string `yaml:"param"`       // page : ex. page
	SizeParam  string `yaml:"size_param"`  // ex. page_size
	Size       int    `yaml:"size"`        // ex. 100
	TokenParam string `yaml:"token_param"` // token : param de requête portant le jeton de page suivante
	MorePath   string `yaml:"more_path"`   // offset : chemin JSON d'un booléen « il reste des items » (ex. HasMoreItems)
	TokenPath  string `yaml:"token_path"`  // token : chemin JSON du jeton suivant dans la réponse (ex. NextPageToken)
	MaxPages   int    `yaml:"max_pages"`   // borne de sécurité anti-boucle (défaut defaultMaxPages)
}

// defaultMaxPages borne le nombre de pages d'un endpoint : un serveur qui ignore la pagination
// (renvoie toujours un lot plein) ne doit pas faire boucler la collecte à l'infini.
const defaultMaxPages = 1000

// maxRespBytes borne le corps d'une reponse : le timeout borne la DUREE, pas la
// taille. Un endpoint hostile ou detourne repond sinon un flux sans fin, que le
// scanner charge entierement en memoire (deni de service du job de CI).
const maxRespBytes = 64 << 20 // 64 Mio

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

// Collect exécute toute la spec et retourne l'inventaire normalisé AVEC son état
// de collecte. `vars` substitue les variables de chemin `{clef}` (ex. {zone}, {org}).
//
// Un échec sur UNE ressource n'interrompt plus la collecte : il est ENREGISTRÉ,
// et les autres endpoints continuent d'être lus.
//
// L'arbitrage, parce qu'il n'est pas évident. Interrompre tout le scan au
// premier 403 est « sûr » — aucun faux vert n'en sort — mais c'est un outil
// inutilisable : un compte de lecture à qui il manque UN droit ne peut alors
// plus rien auditer du tout, et la réaction rationnelle de l'utilisateur est de
// donner à ce compte des droits plus larges. Un CSPM qui pousse à élargir les
// privilèges du compte qui l'exécute travaille contre son propre objet.
// Poursuivre en enregistrant l'échec donne les deux : ce qui a pu être lu est
// mesuré, ce qui n'a pas pu l'être n'est jamais conclu.
//
// Ce qui reste une ERREUR DURE : tout ce qui précède le premier appel (auth
// impossible à construire, URL invalide). Ce n'est pas un périmètre non lu,
// c'est un scan qui n'a pas commencé.
func Collect(ctx context.Context, hc *http.Client, spec Spec, auth Auth, vars map[string]string) (model.Inventory, error) {
	spec.BaseURL = subst(spec.BaseURL, vars) // base_url peut contenir {zone}/{region}
	var inv model.Inventory
	for _, r := range spec.Resources {
		rs, err := collectResource(ctx, hc, spec, auth, r, vars)
		// Les ressources déjà lues sont GARDÉES, même quand l'appel suivant échoue.
		// Un écart observé sur une réponse partielle reste un écart observé : le
		// taire ferait disparaître une non-conformité vraie. L'inverse — conclure
		// « conforme » sur ce qui n'a pas été lu — est empêché par l'unité marquée
		// incomplète, pas par l'oubli des ressources déjà vues.
		inv.Resources = append(inv.Resources, rs...)
		unit := model.CollectionUnit{Unit: r.Type, Types: []string{r.Type}, Attempted: true, Complete: err == nil}
		if err != nil {
			unit.Error, unit.Detail = Classify(err)
		}
		inv.Collection.Record(unit)
	}
	return inv, nil
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
	items, called, err := fetchItems(ctx, hc, auth, resourceURL(spec, r, r.Path, vars), r.Items, r.Paging, r.Method, subst(r.Body, vars))
	src := Source{Origin: model.OriginAPI, Ref: called}
	if err != nil {
		// Une réponse PARTIELLE (pages 1..n lues, page n+1 refusée) rend quand même
		// ce qu'elle a lu. Sauf pour un agrégat : compter des items sur une liste
		// tronquée produirait un NOMBRE FAUX, et un nombre faux est pire qu'un
		// nombre absent — c'est exactement ce qu'une règle comparerait à un seuil.
		if r.Aggregate != "" {
			return nil, err
		}
		return mapItems(spec, r, items, vars, src), err
	}
	if r.Aggregate != "" {
		attrs := map[string]any{r.Aggregate: int64(len(items))}
		// L'agrégat est CALCULÉ sur une réponse réelle : origine `api`, valeur dérivée.
		var prov model.Provenance
		prov.Attest(r.Aggregate, model.Attestation{
			Origin: model.OriginAPI, Source: called, Observed: true, Derived: true,
		})
		for k, v := range r.Const {
			attrs[k] = v
		}
		AttestConst(&prov, r.Const, constRef)
		id, _ := attrs[r.ID].(string)
		return []model.Resource{{Provider: spec.Provider, Type: r.Type, ID: id, Name: id, Region: vars["region"], Attributes: attrs, Provenance: prov}}, nil
	}
	return mapItems(spec, r, items, vars, src), nil
}

// constRef nomme la source d'un attribut littéral : le `const:` du descripteur du
// fournisseur. Ni un appel, ni une mesure — une déclaration.
const constRef = "descriptor:const"

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
	parents, _, err := fetchItems(ctx, hc, auth, resourceURL(spec, r, fe.Path, vars), fe.Items, fe.Paging, fe.Method, subst(fe.Body, vars))
	// La liste parente peut elle aussi être partielle : on collecte les enfants des
	// parents connus, et l'erreur remonte pour marquer l'unité incomplète.
	parentErr := err
	if parentErr != nil {
		parentErr = fmt.Errorf(i18n.T("liste parente %s : %w", "parent list %s: %w"), fe.Path, parentErr)
	}
	var out []model.Resource
	for _, p := range parents {
		pm, _ := p.(map[string]any)
		v2 := mergeVars(vars, fe.As, pm)
		items, called, err := fetchItems(ctx, hc, auth, resourceURL(spec, r, r.Path, v2), r.Items, r.Paging, r.Method, subst(r.Body, v2))
		if err != nil {
			// Un enfant refusé n'annule pas les enfants déjà lus : on garde ce qui a
			// été mesuré et l'unité entière est déclarée incomplète.
			for i := range items {
				items[i] = withParent(items[i], pm)
			}
			out = append(out, mapItems(spec, r, items, v2, Source{Origin: model.OriginAPI, Ref: called})...)
			return out, err
		}
		for i := range items {
			items[i] = withParent(items[i], pm)
		}
		out = append(out, mapItems(spec, r, items, v2, Source{Origin: model.OriginAPI, Ref: called})...)
	}
	return out, parentErr
}

// fetchItems récupère (avec pagination) le tableau d'items d'un endpoint (URL complète).
// Styles : "page" (numéro de page incrémenté jusqu'à un lot incomplet) et "token" (jeton de
// page suivante lu dans la réponse jusqu'à épuisement). Une borne de pages empêche toute
// boucle infinie et, si elle est atteinte, retourne une ERREUR (jamais une troncature muette).
func fetchItems(ctx context.Context, hc *http.Client, auth Auth, fullURL, itemsPath string, paging *Paging, method, body string) ([]any, string, error) {
	max := defaultMaxPages
	if paging != nil && paging.MaxPages > 0 {
		max = paging.MaxPages
	}
	var items []any
	called := ""
	token := ""
	for page := 1; page <= max; page++ {
		// L'offset avance du nombre d'items RÉELLEMENT reçus, jamais de la taille
		// demandée : un serveur qui plafonne la page sous `size` (Outscale borne à 100
		// même si l'on demande 1000) ferait sinon sauter des items.
		doc, endpoint, err := fetch(ctx, hc, fullURL, auth, paging, page, len(items), token, method, body)
		if endpoint != "" {
			called = endpoint
		}
		if err != nil {
			// Les pages déjà lues sont rendues AVEC l'erreur : c'est le cas de la
			// « réponse partielle ». Les jeter perdrait des écarts réellement observés,
			// alors que l'erreur, elle, suffit à interdire toute conclusion positive.
			return items, called, err
		}
		batch := extractItems(doc, itemsPath)
		items = append(items, batch...)
		if paging == nil {
			return items, called, nil
		}
		switch paging.Style {
		case "page", "offset-body":
			if len(batch) == 0 {
				return items, called, nil
			}
			// Quand l'API dit elle-même s'il reste des items (HasMoreItems), on la croit :
			// c'est le seul critère fiable si elle plafonne la page sous `size`.
			if paging.MorePath != "" {
				if !boolFromDoc(doc, paging.MorePath) {
					return items, called, nil
				}
				continue
			}
			// À défaut : une page incomplète est la dernière.
			if len(batch) < paging.Size {
				return items, called, nil
			}
		case "token", "token-body":
			token = tokenFromDoc(doc, paging.TokenPath)
			if token == "" {
				return items, called, nil
			}
		default:
			return items, called, nil
		}
	}
	// Une troncature est une erreur TYPÉE : elle se distingue d'un appel refusé,
	// et l'état de collecte doit pouvoir dire laquelle des deux s'est produite.
	return items, called, &TruncatedError{Call: firstNonEmpty(called, fullURL), MaxPages: max}
}

// firstNonEmpty rend la première chaîne non vide.
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// withBodyPaging fusionne les paramètres de pagination dans le body JSON d'un POST.
// OAPI Outscale : NextPageToken (style token-body) ou FirstItem+ResultsPerPage (offset-body)
// se passent dans le body, pas en query. La taille de page (SizeParam) est toujours posée
// si configurée ; FirstItem = (page-1)*Size pour l'offset.
func withBodyPaging(body string, p *Paging, offset int, token string) string {
	m := map[string]any{}
	if strings.TrimSpace(body) != "" {
		_ = json.Unmarshal([]byte(body), &m)
	}
	if p.SizeParam != "" && p.Size > 0 {
		m[p.SizeParam] = p.Size
	}
	switch p.Style {
	case "token-body":
		if token != "" && p.TokenParam != "" {
			m[p.TokenParam] = token
		}
	case "offset-body":
		if p.Param != "" {
			m[p.Param] = offset // 0-basé, vérifié sur l'API réelle
		}
	}
	b, err := json.Marshal(m)
	if err != nil {
		return body
	}
	return string(b)
}

// boolFromDoc lit un booléen à `path` dans la réponse (ex. HasMoreItems).
func boolFromDoc(doc any, path string) bool {
	b, _ := lookup(doc, path).(bool)
	return b
}

// tokenFromDoc extrait le jeton de page suivante à `path` dans la réponse (chaîne ; "" = fin).
func tokenFromDoc(doc any, path string) string {
	if path == "" {
		return ""
	}
	v := lookup(doc, path)
	s, _ := v.(string)
	return s
}

// mapItems projette les items bruts vers des ressources normalisées (map +
// transforms). La région de collecte (vars["region"]) est propagée sur chaque
// ressource pour les contrôles de localisation (souveraineté, CLD-GVN-3).
func mapItems(spec Spec, r ResourceSpec, items []any, vars map[string]string, src Source) []model.Resource {
	region := vars["region"]
	out := make([]model.Resource, 0, len(items))
	for _, it := range items {
		attrs, prov := ProjectAttested(it, r.Map, r.Transforms, src)
		for k, v := range r.Const {
			attrs[k] = v
		}
		AttestConst(&prov, r.Const, constRef)
		id, _ := attrs[r.ID].(string)
		out = append(out, model.Resource{Provider: spec.Provider, Type: r.Type, ID: id, Name: id, Region: region, Attributes: attrs, Provenance: prov})
	}
	return out
}

// Source décrit ce qui a produit un lot d'items, tel qu'il s'est réellement passé.
// `Ref` d'une source `api` est la requête EFFECTIVEMENT émise (méthode + URL, lue
// après la réponse), jamais le chemin déclaré par la spec : une provenance qui
// nommerait un endpoint jamais appelé donnerait l'apparence de la traçabilité.
// Une Source à zéro (Origin vide) désactive l'attestation.
type Source struct {
	Origin model.Origin
	Ref    string
}

// Project applique une projection DÉCLARATIVE (map attribut_commun -> chemin dans
// l'objet source, + transforms nommés) à un objet (item JSON d'une API ou bloc
// `values` d'une ressource Terraform) et retourne les attributs normalisés. C'est
// le cœur commun à la collecte live ET au mapping Terraform (mêmes specs YAML).
func Project(item any, mapping map[string]string, transforms map[string]any) map[string]any {
	attrs, _ := ProjectAttested(item, mapping, transforms, Source{})
	return attrs
}

// ProjectAttested est Project qui rend EN PLUS l'attestation de chaque attribut du
// mapping : d'où il vient, et si la source le portait réellement.
//
// L'attestation est posée pour TOUT attribut du mapping, y compris ceux que la
// source n'expose pas et qui ne sont donc pas projetés. C'est l'information utile :
// « on a cherché `encrypted` à ce chemin, la réponse ne le portait pas » n'est pas
// « on n'a jamais regardé ». Les attributs projetés, eux, sortent exactement comme
// avant : la provenance ne déplace aucune valeur.
func ProjectAttested(item any, mapping map[string]string, transforms map[string]any, src Source) (map[string]any, model.Provenance) {
	attrs := make(map[string]any, len(mapping))
	var prov model.Provenance
	attest := func(attr, path string, observed, derived bool) {
		if src.Origin == "" {
			return
		}
		prov.Attest(attr, model.Attestation{
			Origin: src.Origin, Source: src.Ref, Path: path,
			Observed: observed, Derived: derived,
		})
	}
	for attr, path := range mapping {
		v := lookupCoalesce(item, path)
		present := sourcePresent(item, path)
		derived := false
		if t, ok := transforms[attr]; ok {
			// `kv`/`list` fabriquent une COLLECTION VIDE même sur nil, pour préserver
			// « présent mais vide » (des tags déclarés et vides sont une information).
			// Cette sémantique ne vaut que si la clé EXISTE réellement dans la source.
			// Sur un plan Terraform, un attribut encore inconnu (`unknown after apply`)
			// est simplement ABSENT de `planned_values` : fabriquer `[]` ferait croire
			// à une collecte réussie, franchirait la garde de capacité de la règle et
			// produirait un faux positif. C'est le cas d'une instance dont le groupe de
			// sécurité est créé par le même plan — la configuration la plus courante.
			if v == nil && !present {
				attest(attr, path, false, false)
				continue
			}
			v = applyTransform(v, t)
			derived = true
		}
		if v == nil {
			// Attribut absent de la source (l'API n'a rien renvoyé) : on ne le projette PAS,
			// au lieu de le forcer à "". Sinon les gardes de capacité (`"k" in object.keys`)
			// sont toujours vraies et un champ numérique absent devient une chaîne vide qui
			// casse les comparaisons — doctrine « ce que le provider n'expose pas ne se teste pas ».
			attest(attr, path, present, false)
			continue
		}
		attrs[attr] = v
		attest(attr, path, present, derived)
	}
	return attrs, prov
}

// AttestConst atteste les attributs LITTÉRAUX d'une spec (`const:`). Ils ne viennent
// d'aucun appel : leur origine est `derived`, et c'est précisément ce qu'un lecteur
// doit pouvoir constater. Deux contrôles franchissent aujourd'hui leur garde d'attribut
// grâce à un `const` (chiffrement transparent Exoscale, portée des clés Outscale) :
// l'attestation ne change pas leur verdict, elle le rend LISIBLE.
func AttestConst(prov *model.Provenance, consts map[string]any, ref string) {
	for attr := range consts {
		prov.Attest(attr, model.Attestation{
			Origin: model.OriginDerived, Source: ref, Derived: true,
		})
	}
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
// sourcePresent indique que le chemin EXISTE dans la source, indépendamment de sa
// valeur. C'est la distinction que `lookup` ne fait pas : il rend nil aussi bien
// pour une clé absente que pour une clé présente à null, alors que les deux ne
// disent pas la même chose. « Absent » signifie que la source n'expose pas
// l'information ; « présent et vide » est une information à part entière.
//
// Un chemin `a||b` est présent dès que l'une de ses alternatives l'est.
func sourcePresent(it any, path string) bool {
	if !strings.Contains(path, "||") {
		return pathExists(it, path)
	}
	for _, p := range strings.Split(path, "||") {
		if pathExists(it, strings.TrimSpace(p)) {
			return true
		}
	}
	return false
}

// pathExists parcourt le chemin comme lookup, mais rend la PRÉSENCE de la
// dernière clé plutôt que sa valeur.
func pathExists(v any, path string) bool {
	if path == "" {
		return v != nil
	}
	keys := strings.Split(path, ".")
	for i, key := range keys {
		switch c := v.(type) {
		case map[string]any:
			next, ok := c[key]
			if !ok {
				return false
			}
			if i == len(keys)-1 {
				return true
			}
			v = next
		case []any:
			idx, err := strconv.Atoi(key)
			if err != nil || idx < 0 || idx >= len(c) {
				return false
			}
			if i == len(keys)-1 {
				return true
			}
			v = c[idx]
		default:
			return false
		}
	}
	return true
}

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

// fetch émet UNE requête et rend le document ainsi que la ligne de requête
// EFFECTIVEMENT servie (« GET https://hote/chemin », sans les paramètres de
// pagination). Cette ligne est lue sur la requête après une réponse valide : elle
// atteste un appel qui a eu lieu, jamais un endpoint que la spec déclare.
func fetch(ctx context.Context, hc *http.Client, rawURL string, auth Auth, p *Paging, page, offset int, token, method, body string) (any, string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, "", err
	}
	if p != nil && p.Style == "page" {
		q := u.Query()
		q.Set(p.Param, strconv.Itoa(page))
		if p.SizeParam != "" {
			q.Set(p.SizeParam, strconv.Itoa(p.Size))
		}
		u.RawQuery = q.Encode()
	}
	if p != nil && p.Style == "token" && token != "" && p.TokenParam != "" {
		q := u.Query()
		q.Set(p.TokenParam, token)
		u.RawQuery = q.Encode()
	}
	// OAPI Outscale : le jeton de page suivante (NextPageToken) et l'offset (FirstItem +
	// ResultsPerPage) se passent DANS LE BODY JSON du POST, pas en query. On les y fusionne.
	if p != nil && (p.Style == "token-body" || p.Style == "offset-body") {
		body = withBodyPaging(body, p, offset, token)
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
		return nil, "", err
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if auth != nil {
		if err := auth.Apply(req); err != nil {
			return nil, "", err
		}
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = resp.Body.Close() }()
	// La requête telle qu'elle a été émise. Les paramètres de requête (page, jeton)
	// sont écartés : ils varient d'une page à l'autre alors que l'endpoint, lui, est
	// ce qui atteste la donnée.
	called := req.Method + " " + req.URL.Scheme + "://" + req.URL.Host + req.URL.Path
	respBody, rerr := io.ReadAll(io.LimitReader(resp.Body, maxRespBytes))
	if rerr != nil {
		return nil, called, fmt.Errorf(i18n.T("lecture de la reponse de %s : %w", "reading the response from %s: %w"), u.Host, rerr)
	}
	if resp.StatusCode >= 300 {
		// Erreur TYPÉE : le statut est ce qui range l'échec dans sa classe (droits
		// insuffisants, service indisponible…). Le relire dans un message serait une
		// correspondance de chaînes, qui casse au premier changement de formulation.
		return nil, called, &HTTPError{Status: resp.StatusCode, Call: called, Body: strings.TrimSpace(string(respBody))}
	}
	var doc any
	if err := json.Unmarshal(respBody, &doc); err != nil {
		return nil, called, fmt.Errorf(i18n.T("réponse JSON invalide : %w", "invalid JSON response: %w"), err)
	}
	return doc, called, nil
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

// IAMPolicyStatements parse un document de politique au format JSON de statements
// (string {Statement:[{Effect, Action, Resource, NotAction, NotResource}]}) en
// statements normalisés [{effect, actions[], resources[], not_action[],
// not_resource[]}]. Helper COMMUN aux providers dont les politiques suivent ce
// format de statements (ex. Outscale EIM).
// IAMPolicyStatements expose le parseur pour les collecteurs Go qui doivent normaliser
// un document de politique hors du moteur YAML (ex. policies EIM inline, chaîne à 3 niveaux).
func IAMPolicyStatements(v any) []any { return iamPolicyStatements(v) }

func iamPolicyStatements(v any) []any {
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
	// La grammaire des policies IAM admet Statement comme TABLEAU ou comme OBJET unique : sans gérer le
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

// knownBareTransforms : noms de transforms de valeur sans préfixe (liste fermée).
var knownBareTransforms = map[string]bool{
	"lower": true, "upper": true, "first": true, "range_from": true, "range_to": true,
	"iampolicy": true, "list": true, "kv": true, "to_int": true, "nonempty": true,
	"snake_keys": true,
}

// knownTransformPrefixes : préfixes de transforms paramétrés (`default:val`, `equals:val`…).
var knownTransformPrefixes = []string{"equals:", "default:", "pluck:", "contains:"}

// ValidateTransform retourne les noms de transforms INCONNUS d'une spec (chaîne, chaînage
// []any, ou table de remplacement map). Un transform inconnu (typo `lowercase`) serait un
// no-op silencieux : sans cette validation, il ferait disparaître un attribut sans erreur,
// court-circuitant la garde de capacité de la règle. Appelé au chargement du descripteur.
func ValidateTransform(spec any) []string {
	switch t := spec.(type) {
	case []any:
		var bad []string
		for _, step := range t {
			bad = append(bad, ValidateTransform(step)...)
		}
		return bad
	case map[string]any:
		return nil // table de remplacement de valeurs : toujours valide
	case string:
		if knownBareTransforms[t] {
			return nil
		}
		for _, p := range knownTransformPrefixes {
			if strings.HasPrefix(t, p) {
				return nil
			}
		}
		return []string{t}
	}
	return nil
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
		// Garde nil : un transform SCALAIRE appliqué à une source absente renvoie nil, pour NE PAS
		// fabriquer une valeur concrète (0, "") qui court-circuiterait le nil-skip de Project —
		// celui-là même qui fonde les gardes de capacité `"k" in object.keys(attrs)`. Les préfixes
		// de dérivation (default:/equals:/contains:/pluck:) sont déjà traités au-dessus ; `nonempty`
		// dérive un booléen d'absence ; `kv`/`list` produisent une COLLECTION vide (l'attribut EST
		// collecté, juste vide) — sémantique légitime à préserver (ex. tags[] présents mais vides).
		if v == nil && t != "nonempty" && t != "kv" && t != "list" {
			return nil
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
		case "iampolicy":
			return iamPolicyStatements(v)
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
		case "snake_keys":
			// Renomme RÉCURSIVEMENT les clés PascalCase -> snake_case d'un objet ou d'une
			// liste d'objets imbriqués (le map: de premier niveau ne renomme pas l'intérieur
			// d'une structure). Ne touche QUE les clés, jamais les valeurs (ex. un protocole
			// "HTTPS" reste "HTTPS"). Sert aux réponses OAPI PascalCase (Listeners[], AccessLog)
			// dont les règles agnostiques lisent les champs en snake_case.
			return snakeKeys(v)
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

// snakeKeys convertit récursivement les clés d'un objet (ou d'une liste d'objets)
// de PascalCase vers snake_case. Les valeurs sont laissées intactes.
func snakeKeys(v any) any {
	switch x := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, val := range x {
			out[toSnakeCase(k)] = snakeKeys(val)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, el := range x {
			out[i] = snakeKeys(el)
		}
		return out
	default:
		return v
	}
}

// toSnakeCase : "LoadBalancerProtocol" -> "load_balancer_protocol", "IsEnabled" ->
// "is_enabled". Insère un underscore avant une majuscule précédée d'une minuscule/chiffre,
// ou en fin d'acronyme (majuscule suivie d'une minuscule).
func toSnakeCase(s string) string {
	var b strings.Builder
	rs := []rune(s)
	for i, r := range rs {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				prev := rs[i-1]
				var next rune
				if i+1 < len(rs) {
					next = rs[i+1]
				}
				lowerPrev := (prev >= 'a' && prev <= 'z') || (prev >= '0' && prev <= '9')
				acronymEnd := prev >= 'A' && prev <= 'Z' && next >= 'a' && next <= 'z'
				if lowerPrev || acronymEnd {
					b.WriteByte('_')
				}
			}
			b.WriteRune(r - 'A' + 'a')
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
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

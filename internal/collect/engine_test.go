package collect

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	yaml "go.yaml.in/yaml/v3"
)

// Démontre la collecte PILOTÉE PAR YAML : une spec déclarative + le moteur
// générique produisent des ressources normalisées, sans code de mapping Go.
func TestCollectFromYAMLSpec(t *testing.T) {
	// API simulée façon Scaleway IAM : GET /iam/v1alpha1/api-keys.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Auth-Token") != "secret-xyz" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"api_keys":[
			{"access_key":"SCWAAAA","expires_at":null},
			{"access_key":"SCWBBBB","expires_at":"2027-01-01T00:00:00Z"}
		]}`))
	}))
	defer srv.Close()

	// Spec DÉCLARÉE en YAML (le « peu de code par provider » = ceci, pas du Go).
	specYAML := `
provider: scaleway
base_url: ` + srv.URL + `
resources:
  - type: access_key
    path: /iam/v1alpha1/api-keys
    items: api_keys
    id: access_key_id
    map:
      access_key_id: access_key
      expiration_date: expires_at
`
	var spec Spec
	if err := yaml.Unmarshal([]byte(specYAML), &spec); err != nil {
		t.Fatalf("spec invalide : %v", err)
	}

	auth := HeaderAuth{Header: "X-Auth-Token", Value: "secret-xyz"}
	res, err := Collect(context.Background(), srv.Client(), spec, auth, nil)
	if err != nil {
		t.Fatalf("Collect : %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("attendu 2 access_key, obtenu %d", len(res))
	}
	// 1re clé : sans expiration → expiration_date ABSENT (non projeté : « ce que la source
	// n'expose pas ne se teste pas »). La règle iam_accesskey_expiration_set se déclenche
	// quand même, via object.get(k, "expiration_date", "") == "".
	if res[0].Type != "access_key" || res[0].Provider != "scaleway" || res[0].ID != "SCWAAAA" {
		t.Errorf("ressource inattendue : %+v", res[0])
	}
	if _, present := res[0].Attributes["expiration_date"]; present {
		t.Errorf("clé sans expiration : expiration_date devrait être ABSENT, got %v", res[0].Attributes["expiration_date"])
	}
	// 2e clé : avec expiration.
	if res[1].Attributes["expiration_date"] != "2027-01-01T00:00:00Z" {
		t.Errorf("expiration_date mal mappé : %v", res[1].Attributes["expiration_date"])
	}
}

// Démontre items IMBRIQUÉS (security_groups[*].rules[*]) + TRANSFORMS, produisant
// le schéma SG commun, toujours sans code de mapping Go.
func TestCollectNestedWithTransforms(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"security_groups":[
			{"id":"sg-1","rules":[
				{"flow-direction":"ingress","protocol":"TCP","start-port":22,"end-port":22,"network":"0.0.0.0/0"}
			]}
		]}`))
	}))
	defer srv.Close()

	specYAML := `
provider: exoscale
base_url: ` + srv.URL + `
resources:
  - type: security_group_rule
    path: /v2/security-group
    items: security_groups[*].rules[*]
    id: security_group_id
    map:
      direction: flow-direction
      protocol: protocol
      port_from: start-port
      port_to: end-port
      cidrs: network
      security_group_id: _parent.id
    transforms:
      direction: { ingress: inbound, egress: outbound }
      protocol: lower
      cidrs: list
`
	var spec Spec
	if err := yaml.Unmarshal([]byte(specYAML), &spec); err != nil {
		t.Fatalf("spec : %v", err)
	}
	res, err := Collect(context.Background(), srv.Client(), spec, nil, nil)
	if err != nil {
		t.Fatalf("Collect : %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("attendu 1 règle, obtenu %d", len(res))
	}
	a := res[0].Attributes
	if a["direction"] != "inbound" {
		t.Errorf("direction transform KO : %v", a["direction"])
	}
	if a["protocol"] != "tcp" {
		t.Errorf("protocol lower KO : %v", a["protocol"])
	}
	if a["security_group_id"] != "sg-1" {
		t.Errorf("_parent.id KO : %v", a["security_group_id"])
	}
	cidrs, ok := a["cidrs"].([]any)
	if !ok || len(cidrs) != 1 || cidrs[0] != "0.0.0.0/0" {
		t.Errorf("cidrs list KO : %v", a["cidrs"])
	}
}

// Démontre la PAGINATION (style page) : 2 pages agrégées.
func TestCollectPaging(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("page") {
		case "1":
			_, _ = w.Write([]byte(`{"items":[{"id":"a"},{"id":"b"}]}`))
		default:
			_, _ = w.Write([]byte(`{"items":[{"id":"c"}]}`))
		}
	}))
	defer srv.Close()
	spec := Spec{Provider: "p", BaseURL: srv.URL, Resources: []ResourceSpec{{
		Type: "thing", Path: "/x", Items: "items", ID: "id", Map: map[string]string{"id": "id"},
		Paging: &Paging{Style: "page", Param: "page", SizeParam: "page_size", Size: 2},
	}}}
	res, err := Collect(context.Background(), srv.Client(), spec, nil, nil)
	if err != nil {
		t.Fatalf("Collect : %v", err)
	}
	if len(res) != 3 {
		t.Fatalf("pagination : attendu 3 (2+1), obtenu %d", len(res))
	}
}

// Démontre la JOINTURE parent→enfant (for_each) façon Scaleway : lister les SG
// puis les règles de chacun, avec coalesce de port et table de protocole.
func TestCollectForEach(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/instance/v1/zones/fr-par-1/security_groups":
			_, _ = w.Write([]byte(`{"security_groups":[{"id":"sg-1"}]}`))
		case "/instance/v1/zones/fr-par-1/security_groups/sg-1/rules":
			_, _ = w.Write([]byte(`{"rules":[
				{"direction":"inbound","action":"accept","protocol":"TCP","ip_range":"0.0.0.0/0","dest_port_from":22,"dest_port_to":null}
			]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	specYAML := `
provider: scaleway
base_url: ` + srv.URL + `
resources:
  - type: security_group_rule
    for_each:
      path: /instance/v1/zones/{zone}/security_groups
      items: security_groups
      as: sg
    path: /instance/v1/zones/{zone}/security_groups/{sg.id}/rules
    items: rules
    id: security_group_id
    map:
      security_group_id: _parent.id
      protocol: protocol
      port_from: dest_port_from
      port_to: dest_port_to||dest_port_from
      cidrs: ip_range
    transforms:
      protocol: { TCP: tcp, UDP: udp, ICMP: icmp, ANY: all }
      cidrs: list
`
	var spec Spec
	if err := yaml.Unmarshal([]byte(specYAML), &spec); err != nil {
		t.Fatalf("spec : %v", err)
	}
	res, err := Collect(context.Background(), srv.Client(), spec, nil, map[string]string{"zone": "fr-par-1"})
	if err != nil {
		t.Fatalf("Collect : %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("attendu 1 règle, obtenu %d", len(res))
	}
	a := res[0].Attributes
	if a["security_group_id"] != "sg-1" {
		t.Errorf("_parent.id KO : %v", a["security_group_id"])
	}
	if a["protocol"] != "tcp" {
		t.Errorf("table protocole KO : %v", a["protocol"])
	}
	// port_to null -> coalesce sur dest_port_from (22).
	if a["port_to"] != float64(22) {
		t.Errorf("coalesce port_to KO : %v (%T)", a["port_to"], a["port_to"])
	}
}

// Protocole IP NUMÉRIQUE (Outscale) : dans un Net, IpProtocol peut être un numéro
// (6=tcp, 17=udp, 1=icmp) au lieu du nom. Sans mapping, une règle "6" sur le port 22
// n'est reconnue ni comme tcp ni comme all → SSH ouvert invisible (faux négatif CSPM,
// contournement classique). La chaîne [lower, {"-1":all,…,"6":tcp}] doit normaliser.
func TestCollectNumericProtocol(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"SecurityGroups":[
			{"SecurityGroupId":"sg-1","InboundRules":[
				{"IpProtocol":"6","IpRanges":["0.0.0.0/0"],"FromPortRange":22,"ToPortRange":22}
			]}
		]}`))
	}))
	defer srv.Close()

	specYAML := `
provider: outscale
base_url: ` + srv.URL + `
resources:
  - type: security_group_rule
    path: /ReadSecurityGroups
    items: SecurityGroups[*].InboundRules[*]
    id: security_group_id
    map:
      security_group_id: _parent.SecurityGroupId
      protocol: IpProtocol
      cidrs: IpRanges
      port_from: FromPortRange
      port_to: ToPortRange
    transforms:
      protocol: [lower, { "-1": all, any: all, "": all, "1": icmp, "6": tcp, "17": udp }]
      cidrs: list
`
	var spec Spec
	if err := yaml.Unmarshal([]byte(specYAML), &spec); err != nil {
		t.Fatalf("spec : %v", err)
	}
	res, err := Collect(context.Background(), srv.Client(), spec, nil, nil)
	if err != nil {
		t.Fatalf("Collect : %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("attendu 1 règle, obtenu %d", len(res))
	}
	if p := res[0].Attributes["protocol"]; p != "tcp" {
		t.Errorf("protocole numérique 6 non normalisé : %v (attendu tcp)", p)
	}
}

// snake_keys : les structures imbriquées d'un LoadBalancer OAPI arrivent en PascalCase
// (Listeners[].LoadBalancerProtocol, AccessLog.IsEnabled) ; les règles agnostiques les
// lisent en snake_case. Le transform doit renommer les CLÉS (récursivement) sans toucher
// aux VALEURS (le protocole "HTTPS" reste "HTTPS").
func TestCollectSnakeKeysNested(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"LoadBalancers":[
			{"LoadBalancerName":"lb-1","LoadBalancerType":"internet-facing",
			 "Listeners":[{"LoadBalancerProtocol":"HTTPS","LoadBalancerPort":443}],
			 "AccessLog":{"IsEnabled":true,"OsuBucketName":"logs"}}
		]}`))
	}))
	defer srv.Close()

	specYAML := `
provider: outscale
base_url: ` + srv.URL + `
resources:
  - type: load_balancer
    path: /ReadLoadBalancers
    items: LoadBalancers
    id: load_balancer_name
    map:
      load_balancer_name: LoadBalancerName
      load_balancer_type: LoadBalancerType
      listeners: Listeners
      access_log: AccessLog
    transforms:
      listeners: snake_keys
      access_log: snake_keys
`
	var spec Spec
	if err := yaml.Unmarshal([]byte(specYAML), &spec); err != nil {
		t.Fatalf("spec : %v", err)
	}
	res, err := Collect(context.Background(), srv.Client(), spec, nil, nil)
	if err != nil {
		t.Fatalf("Collect : %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("attendu 1 LB, obtenu %d", len(res))
	}
	a := res[0].Attributes
	// Clé imbriquée d'objet renommée ; valeur bool intacte.
	al, _ := a["access_log"].(map[string]any)
	if al == nil || al["is_enabled"] != true {
		t.Errorf("access_log.is_enabled KO : %v", a["access_log"])
	}
	// Clé imbriquée d'une liste d'objets renommée ; valeur "HTTPS" intacte.
	ls, _ := a["listeners"].([]any)
	if len(ls) != 1 {
		t.Fatalf("listeners KO : %v", a["listeners"])
	}
	l0, _ := ls[0].(map[string]any)
	if l0["load_balancer_protocol"] != "HTTPS" {
		t.Errorf("listeners[0].load_balancer_protocol KO : %v", l0)
	}
}

// token-body : OAPI Outscale renvoie NextPageToken DANS LE BODY de réponse, et l'attend
// DANS LE BODY du POST suivant. Une spec sans paging ne lirait que la 1re page (troncature
// silencieuse, F1) ; avec paging token-body, le moteur doit suivre le jeton et tout collecter.
func TestCollectTokenBodyPaging(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		if body["NextPageToken"] == nil {
			_, _ = w.Write([]byte(`{"Items":[{"id":"a"},{"id":"b"}],"NextPageToken":"tok2"}`))
			return
		}
		_, _ = w.Write([]byte(`{"Items":[{"id":"c"}]}`)) // dernière page : pas de token
	}))
	defer srv.Close()

	specYAML := `
provider: outscale
base_url: ` + srv.URL + `
resources:
  - type: thing
    path: /ReadThings
    method: POST
    body: "{}"
    items: Items
    id: id
    paging: { style: token-body, token_param: NextPageToken, token_path: NextPageToken, size_param: ResultsPerPage, size: 2 }
    map:
      id: id
`
	var spec Spec
	if err := yaml.Unmarshal([]byte(specYAML), &spec); err != nil {
		t.Fatalf("spec : %v", err)
	}
	res, err := Collect(context.Background(), srv.Client(), spec, nil, nil)
	if err != nil {
		t.Fatalf("Collect : %v", err)
	}
	if len(res) != 3 {
		t.Fatalf("token-body : attendu 3 items (2 pages), obtenu %d — troncature ?", len(res))
	}
}

// offset-body : FirstItem + ResultsPerPage dans le body ; une page incomplète est la dernière.
func TestCollectOffsetBodyPaging(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		first := 0.0
		if v, ok := body["FirstItem"].(float64); ok {
			first = v
		}
		if first == 0 {
			_, _ = w.Write([]byte(`{"Policies":[{"id":"p1"},{"id":"p2"}]}`)) // page pleine (size=2)
			return
		}
		_, _ = w.Write([]byte(`{"Policies":[{"id":"p3"}]}`)) // page incomplète → fin
	}))
	defer srv.Close()

	specYAML := `
provider: outscale
base_url: ` + srv.URL + `
resources:
  - type: iam_policy
    path: /ReadPolicies
    method: POST
    body: "{}"
    items: Policies
    id: id
    paging: { style: offset-body, param: FirstItem, size_param: ResultsPerPage, size: 2 }
    map:
      id: id
`
	var spec Spec
	if err := yaml.Unmarshal([]byte(specYAML), &spec); err != nil {
		t.Fatalf("spec : %v", err)
	}
	res, err := Collect(context.Background(), srv.Client(), spec, nil, nil)
	if err != nil {
		t.Fatalf("Collect : %v", err)
	}
	if len(res) != 3 {
		t.Fatalf("offset-body : attendu 3 policies (2 pages), obtenu %d", len(res))
	}
}

// toSnakeCase : cas limites (acronyme, chiffres).
func TestToSnakeCase(t *testing.T) {
	cases := map[string]string{
		"LoadBalancerProtocol": "load_balancer_protocol",
		"IsEnabled":            "is_enabled",
		"OsuBucketName":        "osu_bucket_name",
		"NetId":                "net_id",
	}
	for in, want := range cases {
		if got := toSnakeCase(in); got != want {
			t.Errorf("toSnakeCase(%q) = %q, attendu %q", in, got, want)
		}
	}
}

// Démontre index de tableau (public_ips.0.address) + transform kv (tags).
func TestCollectArrayIndexAndKV(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"servers":[
			{"id":"vm-1","public_ips":[{"address":"1.2.3.4"}],"tags":["env=prod","owner=bob"]}
		]}`))
	}))
	defer srv.Close()
	spec := Spec{Provider: "scaleway", BaseURL: srv.URL, Resources: []ResourceSpec{{
		Type: "compute_instance", Path: "/servers", Items: "servers", ID: "vm_id",
		Map:        map[string]string{"vm_id": "id", "public_ip": "public_ips.0.address", "tags": "tags"},
		Transforms: map[string]any{"tags": "kv"},
	}}}
	res, err := Collect(context.Background(), srv.Client(), spec, nil, nil)
	if err != nil {
		t.Fatalf("Collect : %v", err)
	}
	a := res[0].Attributes
	if a["public_ip"] != "1.2.3.4" {
		t.Errorf("index tableau KO : %v", a["public_ip"])
	}
	tags, ok := a["tags"].([]any)
	if !ok || len(tags) != 2 {
		t.Fatalf("kv KO : %v", a["tags"])
	}
	if m, _ := tags[0].(map[string]any); m["key"] != "env" || m["value"] != "prod" {
		t.Errorf("kv[0] KO : %v", tags[0])
	}
}

func TestLookupDotted(t *testing.T) {
	doc := map[string]any{"a": map[string]any{"b": "x"}}
	if lookup(doc, "a.b") != "x" {
		t.Fatal("lookup pointé cassé")
	}
	if lookup(doc, "a.z") != nil {
		t.Fatal("champ absent devrait être nil")
	}
}

func TestAwsPolicyStatementsSingleObject(t *testing.T) {
	// Statement en OBJET unique (pas un tableau) : grammaire des policies IAM valide, doit parser.
	got := iamPolicyStatements(`{"Statement":{"Effect":"Allow","Action":"*","Resource":"*"}}`)
	if len(got) != 1 {
		t.Fatalf("Statement objet unique : %d statements, attendu 1", len(got))
	}
	st := got[0].(map[string]any)
	if st["effect"] != "Allow" {
		t.Errorf("effect = %v, attendu Allow", st["effect"])
	}
	if acts := st["actions"].([]any); len(acts) != 1 || acts[0] != "*" {
		t.Errorf("actions = %v, attendu [*]", st["actions"])
	}
}

func TestAwsPolicyStatementsArray(t *testing.T) {
	got := iamPolicyStatements(`{"Statement":[{"Effect":"Allow","Action":["s3:*"]},{"Effect":"Deny","Action":"*"}]}`)
	if len(got) != 2 {
		t.Fatalf("Statement tableau : %d, attendu 2", len(got))
	}
}

func TestAwsPolicyStatementsInvalid(t *testing.T) {
	if got := iamPolicyStatements("not json"); len(got) != 0 {
		t.Errorf("policy invalide : %d, attendu 0", len(got))
	}
	if got := iamPolicyStatements(""); len(got) != 0 {
		t.Errorf("policy vide : %d, attendu 0", len(got))
	}
}

func TestProjectOmitsAbsentAttributes(t *testing.T) {
	item := map[string]any{"name": "vol-1", "size": float64(20)}
	mapping := map[string]string{
		"id":        "name",
		"size":      "size",
		"encrypted": "encrypted", // absent de la source
	}
	attrs := Project(item, mapping, nil)
	if attrs["id"] != "vol-1" || attrs["size"] != float64(20) {
		t.Errorf("attributs présents mal projetés : %+v", attrs)
	}
	// Attribut absent de la source : NON projeté (pas forcé à "").
	if _, present := attrs["encrypted"]; present {
		t.Errorf("un attribut absent de la source ne doit pas être projeté, got %v", attrs["encrypted"])
	}
}

// Un transform de COLLECTION (`list`, `kv`) fabrique une collection vide même sur
// nil, pour préserver « présent mais vide ». Cette sémantique ne vaut que si la clé
// existe réellement : sur un plan Terraform, un attribut encore inconnu
// (`unknown after apply`) est simplement ABSENT de `planned_values`, et fabriquer
// `[]` faisait croire à une collecte réussie.
//
// Conséquence mesurée avant correction : une instance Scaleway dont le groupe de
// sécurité est créé par le même plan — la configuration la plus courante — recevait
// un finding CRITICAL « VM sans groupe de sécurité ». Un faux positif sur le cas
// nominal coûte plus cher qu'un contrôle non évalué : il apprend à ignorer l'outil.
func TestProjectDistinguishesAbsentFromEmptyCollection(t *testing.T) {
	mapping := map[string]string{"security_group_ids": "security_group_id"}
	transforms := map[string]any{"security_group_ids": "list"}

	// Clé ABSENTE (plan Terraform, id encore inconnu) : rien ne doit être projeté,
	// pour que la garde de capacité de la règle reste fermée.
	absent := Project(map[string]any{"id": "srv-1"}, mapping, transforms)
	if _, present := absent["security_group_ids"]; present {
		t.Errorf("clé absente de la source : ne doit pas être projetée, got %v", absent["security_group_ids"])
	}

	// Clé PRÉSENTE à nil (l'API dit explicitement « aucun ») : collection vide
	// projetée, l'information « aucun groupe » est réelle et doit être évaluée.
	explicitNil := Project(map[string]any{"security_group_id": nil}, mapping, transforms)
	got, present := explicitNil["security_group_ids"]
	if !present {
		t.Fatal("clé présente à nil : la collection vide doit être projetée")
	}
	if arr, ok := got.([]any); !ok || len(arr) != 0 {
		t.Errorf("attendu une collection vide, got %#v", got)
	}

	// Clé PRÉSENTE avec une valeur : comportement inchangé.
	withValue := Project(map[string]any{"security_group_id": "sg-1"}, mapping, transforms)
	arr, ok := withValue["security_group_ids"].([]any)
	if !ok || len(arr) != 1 || arr[0] != "sg-1" {
		t.Errorf("valeur scalaire mal enveloppée en liste : %#v", withValue["security_group_ids"])
	}

	// Chemin `a||b` : présent dès qu'une alternative l'est.
	coalesce := map[string]string{"tags": "labels||tags"}
	kv := map[string]any{"tags": "kv"}
	if out := Project(map[string]any{"tags": map[string]any{}}, coalesce, kv); len(out) != 1 {
		t.Errorf("chemin coalescé : la clé présente doit être projetée, got %+v", out)
	}
	if out := Project(map[string]any{"autre": 1}, coalesce, kv); len(out) != 0 {
		t.Errorf("chemin coalescé : aucune alternative présente, rien ne doit être projeté, got %+v", out)
	}
}

func TestValidateTransform(t *testing.T) {
	// Connus : bare, préfixés, chaînage, table de remplacement.
	for _, ok := range []any{
		"lower", "range_from", "list", "nonempty",
		"default:0.0.0.0/0", "contains:iam", "equals:x", "pluck:ip",
		[]any{"default:accept", "lower"},
		map[string]any{"any": "all"},
	} {
		if bad := ValidateTransform(ok); len(bad) != 0 {
			t.Errorf("transform %v jugé inconnu à tort : %v", ok, bad)
		}
	}
	// Inconnu (typo) : signalé, y compris dans un chaînage.
	if bad := ValidateTransform("lowercase"); len(bad) != 1 || bad[0] != "lowercase" {
		t.Errorf("typo `lowercase` non détectée : %v", bad)
	}
	if bad := ValidateTransform([]any{"lower", "nope"}); len(bad) != 1 || bad[0] != "nope" {
		t.Errorf("typo dans un chaînage non détectée : %v", bad)
	}
}

func TestFetchItemsTokenPagination(t *testing.T) {
	// Serveur : 3 pages liées par un jeton `next` (dernière page : next="").
	pages := map[string]string{
		"":   `{"items":[{"id":"a"},{"id":"b"}],"next":"p2"}`,
		"p2": `{"items":[{"id":"c"}],"next":"p3"}`,
		"p3": `{"items":[{"id":"d"}],"next":""}`,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(pages[r.URL.Query().Get("page_token")]))
	}))
	defer srv.Close()

	p := &Paging{Style: "token", TokenParam: "page_token", TokenPath: "next"}
	items, _, err := fetchItems(context.Background(), srv.Client(), nil, srv.URL, "items", p, "GET", "")
	if err != nil {
		t.Fatalf("fetchItems: %v", err)
	}
	if len(items) != 4 {
		t.Errorf("pagination token : %d items, attendu 4 (a,b,c,d)", len(items))
	}
}

func TestFetchItemsPageBoundGuardsInfiniteLoop(t *testing.T) {
	// Serveur pathologique : ignore la pagination et renvoie TOUJOURS un lot plein.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"items":[{"id":"x"},{"id":"y"}]}`))
	}))
	defer srv.Close()

	p := &Paging{Style: "page", Param: "page", Size: 2, MaxPages: 5}
	_, _, err := fetchItems(context.Background(), srv.Client(), nil, srv.URL, "items", p, "GET", "")
	if err == nil {
		t.Error("un serveur qui ignore la pagination doit provoquer une ERREUR de borne, pas une boucle infinie")
	}
}

func TestNilTransformSemantics(t *testing.T) {
	// Collections : nil → collection VIDE (attribut collecté mais vide) — la garde de capacité
	// d'une règle doit voir l'attribut présent (ex. tags:[] → required_tags flague les manquants).
	if got := applyTransform(nil, "kv"); got == nil {
		t.Error("kv(nil) doit produire [] (collecté, vide), pas nil")
	}
	if got := applyTransform(nil, "list"); got == nil {
		t.Error("list(nil) doit produire [] (collecté, vide), pas nil")
	}
	// Scalaires : nil → nil (attribut absent), pour préserver le nil-skip des gardes.
	if got := applyTransform(nil, "range_from"); got != nil {
		t.Errorf("range_from(nil) doit rester nil, pas %v", got)
	}
	if got := applyTransform(nil, "lower"); got != nil {
		t.Errorf("lower(nil) doit rester nil, pas %v", got)
	}
}

// RÉGRESSION : un serveur qui PLAFONNE la page sous la taille demandée (Outscale borne à
// 100 même si l'on demande 1000) faisait s'arrêter la collecte après la 1re page, car
// l'arrêt reposait sur `len(batch) < size`. On suit désormais `HasMoreItems` et l'offset
// avance du nombre d'items RÉELLEMENT reçus. Sans le correctif, ce test ne voit que 2/5.
func TestCollectOffsetBodyServerCapsPageSize(t *testing.T) {
	const cap = 2 // le serveur ne rend jamais plus de 2 items, quoi qu'on demande
	all := []string{"u1", "u2", "u3", "u4", "u5"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		first := 0
		if v, ok := body["FirstItem"].(float64); ok {
			first = int(v)
		}
		end := first + cap
		if end > len(all) {
			end = len(all)
		}
		var names []string
		if first < len(all) {
			names = all[first:end]
		}
		out := map[string]any{"Users": []map[string]string{}, "HasMoreItems": end < len(all)}
		us := make([]map[string]string, 0, len(names))
		for _, n := range names {
			us = append(us, map[string]string{"UserName": n})
		}
		out["Users"] = us
		b, _ := json.Marshal(out)
		_, _ = w.Write(b)
	}))
	defer srv.Close()

	specYAML := `
provider: outscale
base_url: ` + srv.URL + `
resources:
  - type: iam_user
    path: /ReadUsers
    method: POST
    body: "{}"
    items: Users
    id: name
    paging: { style: offset-body, param: FirstItem, size_param: ResultsPerPage, size: 100, more_path: HasMoreItems }
    map:
      name: UserName
`
	var spec Spec
	if err := yaml.Unmarshal([]byte(specYAML), &spec); err != nil {
		t.Fatalf("spec : %v", err)
	}
	res, err := Collect(context.Background(), srv.Client(), spec, nil, nil)
	if err != nil {
		t.Fatalf("Collect : %v", err)
	}
	if len(res) != len(all) {
		t.Fatalf("troncature : attendu %d items malgré le plafond serveur, obtenu %d", len(all), len(res))
	}
	seen := map[string]bool{}
	for _, r := range res {
		n, _ := r.Attributes["name"].(string)
		if seen[n] {
			t.Errorf("item dupliqué : %s (offset mal avancé)", n)
		}
		seen[n] = true
	}
}

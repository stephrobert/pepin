package collect

import (
	"context"
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
	// Statement en OBJET unique (pas un tableau) : grammaire AWS/IAM valide, doit parser.
	got := awsPolicyStatements(`{"Statement":{"Effect":"Allow","Action":"*","Resource":"*"}}`)
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
	got := awsPolicyStatements(`{"Statement":[{"Effect":"Allow","Action":["s3:*"]},{"Effect":"Deny","Action":"*"}]}`)
	if len(got) != 2 {
		t.Fatalf("Statement tableau : %d, attendu 2", len(got))
	}
}

func TestAwsPolicyStatementsInvalid(t *testing.T) {
	if got := awsPolicyStatements("not json"); len(got) != 0 {
		t.Errorf("policy invalide : %d, attendu 0", len(got))
	}
	if got := awsPolicyStatements(""); len(got) != 0 {
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

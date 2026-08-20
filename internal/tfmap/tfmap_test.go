package tfmap

import (
	"testing"

	"github.com/stephrobert/pepin/internal/tfparse"
)

// Vérifie la projection déclarative : map, transform equals, const, region.
func TestApply(t *testing.T) {
	spec, err := Parse([]byte(`
provider: exoscale
resources:
  - tf_type: exoscale_sks_cluster
    type: kubernetes_cluster
    id: name
    region: zone
    map:
      name: name
      auto_upgrade: auto_upgrade
      control_plane_multi_az: service_level
    transforms:
      control_plane_multi_az: "equals:pro"
  - tf_type: exoscale_security_group_rule
    type: security_group_rule
    id: security_group_id
    map:
      security_group_id: security_group_id
      cidrs: cidr
    transforms:
      cidrs: list
    const:
      action: accept
`))
	if err != nil {
		t.Fatalf("Parse : %v", err)
	}

	inv := Apply(spec, []tfparse.Resource{
		{Type: "exoscale_sks_cluster", Address: "exoscale_sks_cluster.k", Values: map[string]any{
			"name": "k1", "zone": "ch-gva-2", "service_level": "starter", "auto_upgrade": false,
		}},
		{Type: "exoscale_security_group_rule", Address: "r", Values: map[string]any{
			"security_group_id": "sg-1", "cidr": "0.0.0.0/0",
		}},
		{Type: "exoscale_ignored", Address: "x", Values: map[string]any{"foo": "bar"}},
	})
	res := inv.Resources

	if len(res) != 2 {
		t.Fatalf("attendu 2 ressources (type inconnu ignoré), obtenu %d", len(res))
	}
	// Le type non projeté n'est plus ignoré EN SILENCE : il est enregistré, parce
	// qu'il porte le préfixe du fournisseur — donc parce qu'un lecteur pourrait
	// croire que Pépin l'a audité.
	if len(inv.Collection.Unmapped) != 1 || inv.Collection.Unmapped[0].Type != "exoscale_ignored" {
		t.Errorf("le type non projeté doit être enregistré, obtenu %+v", inv.Collection.Unmapped)
	}

	sks := res[0]
	if sks.Type != "kubernetes_cluster" || sks.Region != "ch-gva-2" {
		t.Errorf("SKS type/region KO : %+v", sks)
	}
	// service_level "starter" != "pro" → control_plane_multi_az false (equals).
	if sks.Attributes["control_plane_multi_az"] != false {
		t.Errorf("equals KO : %v", sks.Attributes["control_plane_multi_az"])
	}

	rule := res[1]
	if rule.Attributes["action"] != "accept" {
		t.Errorf("const action KO : %v", rule.Attributes["action"])
	}
	cidrs, ok := rule.Attributes["cidrs"].([]any)
	if !ok || len(cidrs) != 1 || cidrs[0] != "0.0.0.0/0" {
		t.Errorf("transform list KO : %v", rule.Attributes["cidrs"])
	}
	if rule.ID != "sg-1" {
		t.Errorf("id KO : %v", rule.ID)
	}
}

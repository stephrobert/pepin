package tfparse

import (
	"os"
	"path/filepath"
	"testing"
)

func writePlan(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "plan.json")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatalf("écriture du plan : %v", err)
	}
	return p
}

// Sortie d'un fichier de plan : bloc planned_values, modules enfants inclus.
func TestParsePlanPlannedValues(t *testing.T) {
	path := writePlan(t, `{
	  "planned_values": { "root_module": {
	    "resources": [
	      {"address":"a.root","type":"outscale_vm","name":"root","values":{"id":"i-1"}}
	    ],
	    "child_modules": [
	      {"address":"module.net","resources":[
	        {"address":"module.net.b","type":"outscale_security_group","name":"b","values":{"id":"sg-1"}}
	      ]}
	    ]
	  }}
	}`)

	got, err := ParsePlan(path)
	if err != nil {
		t.Fatalf("ParsePlan : %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("attendu 2 ressources (racine + module enfant), obtenu %d", len(got))
	}
	// Tri déterministe par adresse : "a.root" avant "module.net.b".
	if got[0].Type != "outscale_vm" || got[0].Values["id"] != "i-1" {
		t.Errorf("ressource racine inattendue : %+v", got[0])
	}
	if got[1].Type != "outscale_security_group" {
		t.Errorf("ressource de module enfant non collectée : %+v", got[1])
	}
}

// Sortie d'un état appliqué : bloc values (et non planned_values).
func TestParsePlanAppliedState(t *testing.T) {
	path := writePlan(t, `{
	  "values": { "root_module": { "resources": [
	    {"address":"a","type":"scaleway_object_bucket_acl","name":"a","values":{"bucket":"b1","acl":"public-read"}}
	  ]}}
	}`)

	got, err := ParsePlan(path)
	if err != nil {
		t.Fatalf("ParsePlan : %v", err)
	}
	if len(got) != 1 || got[0].Values["acl"] != "public-read" {
		t.Fatalf("état appliqué mal parsé : %+v", got)
	}
}

func TestParsePlanRejectsNonPlan(t *testing.T) {
	path := writePlan(t, `{"hello":"world"}`)
	if _, err := ParsePlan(path); err == nil {
		t.Fatal("attendu une erreur pour un JSON sans planned_values ni values")
	}
}

package tfparse

import (
	"os"
	"path/filepath"
	"testing"
)

// Ce que ces tests tiennent : l'origine est MESURÉE ou ABSENTE, jamais fabriquée.
// Une ligne fausse envoie corriger le mauvais endroit, et on la croit — c'est
// strictement pire qu'une ligne manquante.

// writePlanWithSources écrit un plan et les sources HCL qui l'accompagnent, et rend le
// chemin du plan.
func writePlanWithSources(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatalf("création de %s : %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatalf("écriture de %s : %v", path, err)
		}
	}
	return filepath.Join(dir, "plan.json")
}

const rootPlan = `{
  "format_version": "1.2",
  "planned_values": {"root_module": {"resources": [
    {"address": "scaleway_object_bucket.backups", "type": "scaleway_object_bucket", "name": "backups", "values": {"name": "b"}}
  ]}}
}`

const rootHCL = `# un commentaire

resource "scaleway_instance_server" "web" {
  type = "DEV1-S"
}

resource "scaleway_object_bucket" "backups" {
  name = "backups-prod"
}
`

// TestTheOriginNamesTheFileAndTheLineOfTheBlock : le cas nominal. La ligne est
// celle de l'EN-TÊTE du bloc, l'endroit où un correcteur ouvre son éditeur.
func TestTheOriginNamesTheFileAndTheLineOfTheBlock(t *testing.T) {
	path := writePlanWithSources(t, map[string]string{"plan.json": rootPlan, "main.tf": rootHCL})
	res, err := ParsePlan(path)
	if err != nil {
		t.Fatalf("ParsePlan : %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("%d ressources, attendu 1", len(res))
	}
	if got := res[0].Origin.File; got != "main.tf" {
		t.Errorf("fichier %q, attendu main.tf", got)
	}
	if got := res[0].Origin.Line; got != 7 {
		t.Errorf("ligne %d, attendu 7 (l'en-tête du bloc)", got)
	}
	if got := res[0].Origin.Module; got != "" {
		t.Errorf("module %q, attendu vide (module racine)", got)
	}
}

// TestWithoutSourcesTheOriginIsAbsentAndNothingFails : un plan seul, sans ses
// `.tf`. C'est le cas d'un plan produit ailleurs et transmis : on ne devine pas
// un chemin, et rien n'échoue.
func TestWithoutSourcesTheOriginIsAbsentAndNothingFails(t *testing.T) {
	path := writePlanWithSources(t, map[string]string{"plan.json": rootPlan})
	res, err := ParsePlan(path)
	if err != nil {
		t.Fatalf("l'absence de sources ne doit rien faire échouer : %v", err)
	}
	if !res[0].Origin.Empty() {
		t.Errorf("aucune origine ne devait être établie, obtenu %+v", res[0].Origin)
	}
}

// TestAnAmbiguousBlockYieldsNoOrigin : deux blocs de même type et de même nom
// dans le périmètre cherché ne se départagent pas. En rendre un au hasard, c'est
// désigner le mauvais une fois sur deux.
func TestAnAmbiguousBlockYieldsNoOrigin(t *testing.T) {
	dupe := `resource "scaleway_object_bucket" "backups" {
  name = "a"
}
`
	path := writePlanWithSources(t, map[string]string{
		"plan.json": rootPlan, "main.tf": rootHCL, "duplicate.tf": dupe,
	})
	res, err := ParsePlan(path)
	if err != nil {
		t.Fatalf("ParsePlan : %v", err)
	}
	if res[0].Origin.File != "" || res[0].Origin.Line != 0 {
		t.Errorf("un bloc ambigu ne doit désigner aucun fichier, obtenu %+v", res[0].Origin)
	}
}

const modulePlan = `{
  "format_version": "1.2",
  "planned_values": {"root_module": {"child_modules": [
    {"resources": [
      {"address": "module.network.scaleway_vpc_private_network.main",
       "type": "scaleway_vpc_private_network", "name": "main", "values": {"name": "n"}}
    ]}
  ]}},
  "configuration": {"root_module": {"module_calls": {
    "network": {"source": "./modules/network", "module": {}}
  }}}
}`

// TestAModuleResourceIsLocatedInItsOwnDirectory : la correspondance jusqu'au
// fichier passe par `module_calls[].source`, le seul indice de localisation que
// le format JSON de Terraform expose.
func TestAModuleResourceIsLocatedInItsOwnDirectory(t *testing.T) {
	path := writePlanWithSources(t, map[string]string{
		"plan.json": modulePlan,
		"main.tf":   rootHCL,
		"modules/network/main.tf": `resource "scaleway_vpc_private_network" "main" {
  name = "prod"
}
`,
	})
	res, err := ParsePlan(path)
	if err != nil {
		t.Fatalf("ParsePlan : %v", err)
	}
	o := res[0].Origin
	if o.Module != "module.network" {
		t.Errorf("module %q, attendu module.network", o.Module)
	}
	if o.File != filepath.Join("modules", "network", "main.tf") {
		t.Errorf("fichier %q, attendu modules/network/main.tf", o.File)
	}
	if o.Line != 1 {
		t.Errorf("ligne %d, attendu 1", o.Line)
	}
}

// TestARemoteModuleKeepsItsModuleButHasNoFile : une source de registre ou de
// dépôt git ne désigne aucun fichier de l'arbre de travail. L'origine est alors
// PARTIELLE, et rendue partielle — le module reste utile.
func TestARemoteModuleKeepsItsModuleButHasNoFile(t *testing.T) {
	remote := `{
  "format_version": "1.2",
  "planned_values": {"root_module": {"child_modules": [
    {"resources": [
      {"address": "module.network.scaleway_vpc_private_network.main",
       "type": "scaleway_vpc_private_network", "name": "main", "values": {}}
    ]}
  ]}},
  "configuration": {"root_module": {"module_calls": {
    "network": {"source": "registry.terraform.io/acme/network/scaleway", "module": {}}
  }}}
}`
	path := writePlanWithSources(t, map[string]string{"plan.json": remote, "main.tf": rootHCL})
	res, err := ParsePlan(path)
	if err != nil {
		t.Fatalf("ParsePlan : %v", err)
	}
	o := res[0].Origin
	if o.Module != "module.network" {
		t.Errorf("le module reste connu : %q", o.Module)
	}
	if o.File != "" || o.Line != 0 {
		t.Errorf("aucun fichier ne doit être désigné pour une source distante, obtenu %+v", o)
	}
}

// TestModuleOfReadsTheAddress : la déduction du module est une lecture de
// l'adresse, y compris imbriquée et indexée.
func TestModuleOfReadsTheAddress(t *testing.T) {
	cases := map[string]string{
		"scaleway_object_bucket.b":                           "",
		"module.a.scaleway_object_bucket.b":                  "module.a",
		"module.a.module.b.scaleway_object_bucket.c":         "module.a.module.b",
		"module.production_network.scaleway_vpc.main[\"x\"]": "module.production_network",
	}
	for addr, want := range cases {
		if got := moduleOf(addr); got != want {
			t.Errorf("moduleOf(%q) = %q, attendu %q", addr, got, want)
		}
	}
}

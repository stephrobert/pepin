package tfmap

import (
	"testing"

	"github.com/stephrobert/pepin/internal/model"
	"github.com/stephrobert/pepin/internal/tfparse"
)

// specRef : un descripteur minimal qui lit deux champs, l'un scalaire, l'autre par
// un chemin composé.
func specRef(t *testing.T) Spec {
	t.Helper()
	s, err := Parse([]byte(`
provider: essai
resources:
  - tf_type: essai_vm
    type: compute_instance
    id: vm_id
    map:
      vm_id: vm_id
      security_group_ids: security_group_ids
      imbrique: bloc.champ
    transforms:
      security_group_ids: list
`))
	if err != nil {
		t.Fatalf("spec illisible : %v", err)
	}
	return s
}

func vmAvec(values map[string]any, refs map[string][]string) []tfparse.Resource {
	return []tfparse.Resource{{
		Type: "essai_vm", Name: "web", Address: "essai_vm.web",
		Values: values, References: refs,
	}}
}

func premiere(t *testing.T, inv model.Inventory) model.Resource {
	t.Helper()
	if len(inv.Resources) != 1 {
		t.Fatalf("%d ressource(s) projetée(s), 1 attendue", len(inv.Resources))
	}
	return inv.Resources[0]
}

// Une référence comble le trou que laisse un identifiant encore inconnu.
func TestAReferenceFillsTheGapLeftByAnUnknownValue(t *testing.T) {
	inv := Apply(specRef(t), vmAvec(
		map[string]any{"state": "running"},
		map[string][]string{"security_group_ids": {"essai_sg.web"}},
	))
	r := premiere(t, inv)
	got, _ := r.Attributes["security_group_ids"].([]any)
	if len(got) != 1 || got[0] != "essai_sg.web" {
		t.Fatalf("attribut = %v, attendu [essai_sg.web]", r.Attributes["security_group_ids"])
	}
}

// LE CONTRE-EXEMPLE le plus important : une valeur PRÉSENTE gagne toujours. Sans
// cela, un état appliqué (`values`, tout résolu) verrait ses vraies valeurs
// remplacées par des adresses — on écraserait l'observation par la déclaration.
func TestAPresentValueAlwaysWinsOverAReference(t *testing.T) {
	inv := Apply(specRef(t), vmAvec(
		map[string]any{"security_group_ids": []any{"sg-12345678"}},
		map[string][]string{"security_group_ids": {"essai_sg.web"}},
	))
	r := premiere(t, inv)
	got, _ := r.Attributes["security_group_ids"].([]any)
	if len(got) != 1 || got[0] != "sg-12345678" {
		t.Fatalf("attribut = %v, attendu la valeur réelle [sg-12345678]", r.Attributes["security_group_ids"])
	}
}

// Un chemin COMPOSÉ désigne une structure du bloc `values`, pas un argument de la
// configuration. Les mêler ferait projeter une adresse là où une descente est
// attendue, et ce serait un mélange de deux espaces de noms.
func TestACompoundPathIsNeverFilledFromAReference(t *testing.T) {
	inv := Apply(specRef(t), vmAvec(
		map[string]any{},
		map[string][]string{"bloc.champ": {"essai_sg.web"}, "bloc": {"essai_sg.web"}},
	))
	if _, present := premiere(t, inv).Attributes["imbrique"]; present {
		t.Fatal("un chemin composé a été comblé par une référence")
	}
}

// L'attestation doit dire d'où vient la valeur. Une traçabilité qui désigne le
// mauvais endroit est PIRE que son absence : ici, la valeur ne vient pas de
// `planned_values` mais de `configuration`, et elle porte une adresse plutôt que la
// valeur du champ.
func TestAReferenceIsAttestedAsDerivedFromTheConfiguration(t *testing.T) {
	inv := Apply(specRef(t), vmAvec(
		map[string]any{},
		map[string][]string{"security_group_ids": {"essai_sg.web"}},
	))
	a, ok := premiere(t, inv).Provenance["security_group_ids"]
	if !ok {
		t.Fatal("aucune attestation pour un attribut comblé par une référence")
	}
	if !a.Derived {
		t.Error("attestation non marquée dérivée : la valeur est une adresse, pas le champ")
	}
	if a.Source != "configuration:essai_vm" {
		t.Errorf("source = %q, attendue configuration:essai_vm", a.Source)
	}
	if a.Path != "security_group_ids.references" {
		t.Errorf("chemin = %q, attendu security_group_ids.references", a.Path)
	}
}

// Une ressource sans référence traverse le mapper exactement comme avant : la
// fonctionnalité n'ajoute rien là où il n'y a rien à résoudre.
func TestAResourceWithoutReferencesIsUntouched(t *testing.T) {
	inv := Apply(specRef(t), vmAvec(map[string]any{"vm_id": "i-1"}, nil))
	r := premiere(t, inv)
	if r.ID != "i-1" || len(r.Attributes) != 1 {
		t.Fatalf("ressource modifiée sans raison : id=%q attrs=%v", r.ID, r.Attributes)
	}
}

// ── UN SUJET, PAS DEUX (#193) ─────────────────────────────────────────────────

// specParent : un mapping en mode `items` qui lit son porteur par `_parent.<champ>`,
// et le mapping de la ressource que ce champ référence.
func specParent(t *testing.T) Spec {
	t.Helper()
	s, err := Parse([]byte(`
provider: essai
resources:
  - tf_type: essai_db_acl
    type: managed_database
    items: acl_rules[*]
    id: database_id
    map:
      database_id: _parent.instance_id
      ip_filter: ip
  - tf_type: essai_db
    type: managed_database
    id: database_id
    map:
      database_id: name
      disable_backup: disable_backup
`))
	if err != nil {
		t.Fatalf("spec illisible : %v", err)
	}
	return s
}

// Une même base obtenait DEUX sujets sur un plan : la règle dérivée de l'ACL nommait la
// ressource ACL, celles dérivées de l'instance nommaient la base. Trois écarts, deux
// noms — et une dérogation écrite sur l'un ratait l'autre, en silence.
//
// Deux choses le corrigent, et il faut les deux : combler `_parent.<champ>` depuis la
// référence, puis résoudre l'adresse obtenue vers l'IDENTITÉ que la ressource visée
// porte dans l'inventaire — ici son `name`, pas son adresse.
func TestOneDatabaseGetsOneSubject(t *testing.T) {
	inv := Apply(specParent(t), []tfparse.Resource{
		{Type: "essai_db", Name: "exposed", Address: "essai_db.exposed",
			Values: map[string]any{"name": "ma-base", "disable_backup": true}},
		{Type: "essai_db_acl", Name: "exposed", Address: "essai_db_acl.exposed",
			Values:     map[string]any{"acl_rules": []any{map[string]any{"ip": "0.0.0.0/0"}}},
			References: map[string][]string{"instance_id": {"essai_db.exposed"}}},
	})
	if len(inv.Resources) != 2 {
		t.Fatalf("%d ressource(s), 2 attendues", len(inv.Resources))
	}
	for _, r := range inv.Resources {
		if r.ID != "ma-base" {
			t.Errorf("sujet %q, attendu \"ma-base\" : la base porte deux noms", r.ID)
		}
	}
}

// LE CONTRE-EXEMPLE de la seconde passe : une valeur qui n'a PAS été comblée depuis une
// référence n'est jamais réécrite, même si elle ressemble à une adresse. Réécrire à
// l'aveugle ferait basculer une donnée du tenant sur une identité qui n'est pas la
// sienne.
func TestAValueThatLooksLikeAnAddressIsNeverRewritten(t *testing.T) {
	inv := Apply(specParent(t), []tfparse.Resource{
		{Type: "essai_db", Name: "exposed", Address: "essai_db.exposed",
			Values: map[string]any{"name": "ma-base", "disable_backup": true}},
		// `instance_id` est PRÉSENT et vaut littéralement une adresse : aucune
		// référence n'a été consultée, donc rien ne doit bouger.
		{Type: "essai_db_acl", Name: "acl", Address: "essai_db_acl.acl",
			Values: map[string]any{"instance_id": "essai_db.exposed", "acl_rules": []any{map[string]any{"ip": "10.0.0.0/8"}}}},
	})
	var vu bool
	for _, r := range inv.Resources {
		if r.ID == "essai_db.exposed" {
			vu = true
		}
	}
	if !vu {
		t.Error("une valeur littérale a été réécrite : la seconde passe déborde des attributs injectés")
	}
}

// Un chemin qui DESCEND dans une structure reste écarté : une référence n'y répond pas.
func TestACompoundPathBeyondParentIsStillNeverFilled(t *testing.T) {
	if _, ok := champArgument("audit.0.endpoint"); ok {
		t.Error("un chemin de descente a été pris pour un argument")
	}
	if c, ok := champArgument("_parent.instance_id"); !ok || c != "instance_id" {
		t.Errorf("champArgument(_parent.instance_id) = %q,%v", c, ok)
	}
	if c, ok := champArgument("vm_id"); !ok || c != "vm_id" {
		t.Errorf("champArgument(vm_id) = %q,%v", c, ok)
	}
}

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

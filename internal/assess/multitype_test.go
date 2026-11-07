package assess

import (
	"testing"

	"github.com/stephrobert/pepin/internal/model"
)

// TestAnIncompleteSecondTypeDegradesTheControl ferme le faux vert de l'ADR-0006.
//
// Six règles CORRÈLENT plusieurs types. Tant qu'un contrôle ne déclarait que celui
// dont son code porte le préfixe, l'incomplétude du second passait inaperçue :
//
//	Collecte des VMs        -> OK
//	Collecte des règles SG  -> 403
//
// et `compute_instance_public_ip_with_open_securitygroup` rendait `pass` — « aucune
// non-conformité détectée sur les ressources de type compute_instance » — alors que
// la donnée qui ÉTABLIT l'exposition n'était jamais arrivée. Mesuré avant/après :
// le verdict passe désormais à `not-evaluated`, en nommant l'unité fautive.
func TestAnIncompleteSecondTypeDegradesTheControl(t *testing.T) {
	coll := model.Collection{Units: []model.CollectionUnit{{
		Unit:      "security_group_rule",
		Types:     []string{"security_group_rule"},
		Attempted: true,
		Complete:  false,
		Error:     model.OutcomePermissionDenied,
	}}}
	scope := map[string]bool{"compute_instance_public_ip_with_open_securitygroup": true}

	got := DegradedControls(coll, nil, scope, nil)
	unit, ok := got["compute_instance_public_ip_with_open_securitygroup"]
	if !ok {
		t.Fatal("un contrôle qui corrèle security_group_rule n'est pas dégradé quand cette " +
			"collecte est incomplète : il conclura sur une donnée jamais arrivée")
	}
	if unit.Unit != "security_group_rule" {
		t.Fatalf("l'unité nommée doit être celle qui a échoué, obtenu %q", unit.Unit)
	}
}

// TestOnlyTheTypesAControlReadsDegradeIt est le contre-test. Dégrader trop large
// serait un faux `not-evaluated` : moins grave qu'un faux `pass`, et tout aussi faux.
func TestOnlyTheTypesAControlReadsDegradeIt(t *testing.T) {
	coll := model.Collection{Units: []model.CollectionUnit{{
		Unit:      "object_storage_bucket",
		Types:     []string{"object_storage_bucket"},
		Attempted: true,
		Complete:  false,
		Error:     model.OutcomePermissionDenied,
	}}}
	scope := map[string]bool{"compute_instance_public_ip_with_open_securitygroup": true}

	if got := DegradedControls(coll, nil, scope, nil); len(got) != 0 {
		t.Fatalf("un contrôle qui ne lit pas ce type ne doit pas être dégradé, obtenu %v", got)
	}
}

func TestTaggingDegradationUsesOnlyConfiguredResourceTypes(t *testing.T) {
	coll := model.Collection{Units: []model.CollectionUnit{{
		Unit: "object_storage_bucket", Types: []string{"object_storage_bucket"},
		Attempted: true, Error: model.OutcomeUnreadable,
	}}}
	scope := map[string]bool{"governance_resource_required_tags": true}
	for _, tt := range []struct {
		name  string
		types []string
		want  bool
	}{
		{"includes incomplete type", []string{"compute_instance", "object_storage_bucket"}, true},
		{"excludes incomplete type", []string{"compute_instance"}, false},
		{"explicitly empty", []string{}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := DegradedControls(coll, nil, scope, tt.types)
			unit, found := got["governance_resource_required_tags"]
			if found != tt.want || (found && unit.Unit != "object_storage_bucket") {
				t.Fatalf("dégradation = %v ; attendue = %t", got, tt.want)
			}
		})
	}
}

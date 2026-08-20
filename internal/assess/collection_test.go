package assess

import (
	"strings"
	"testing"

	"github.com/stephrobert/pepin/internal/model"
	"github.com/stephrobert/scankit/assessment"
)

// Ce que ces tests tiennent : la dégradation est DIRECTIONNELLE. Elle n'a le droit
// de produire qu'une seule transition, `pass → not-evaluated`. Tout le reste
// serait soit une perte d'information (effacer un écart vu), soit un faux vert
// (le contraire de ce que la vague cherche).

// degradedIAM : une unité de collecte incomplète qui alimente `iam_policy`, telle
// que la produit un 403 sur ReadUserPolicies — l'incident fondateur de la vague.
var degradedIAM = map[string]model.CollectionUnit{
	"iam_policy_no_wildcard_resource": {
		Unit: "iam_policy_inline", Types: []string{"iam_policy"},
		Attempted: true, Complete: false, Error: model.OutcomePermissionDenied,
	},
}

// TestAnIncompleteCollectionWithdrawsThePassAndOnlyThePass est l'invariant
// central : sur les quatre statuts possibles, un seul bouge.
func TestAnIncompleteCollectionWithdrawsThePassAndOnlyThePass(t *testing.T) {
	unit := degradedIAM["iam_policy_no_wildcard_resource"]
	degraded := map[string]model.CollectionUnit{
		"pass_control":  unit,
		"fail_control":  unit,
		"na_control":    unit,
		"ne_control":    unit,
		"other_control": unit,
	}
	in := assessment.Assessment{Results: []assessment.Result{
		{Control: "pass_control", Status: assessment.Pass},
		{Control: "fail_control", Status: assessment.Fail, Subject: "policy-admin"},
		{Control: "na_control", Status: assessment.NotApplicable},
		{Control: "ne_control", Status: assessment.NotEvaluated,
			Evidence: assessment.Evidence{Observed: "aucune ressource de type iam_policy"}},
		{Control: "untouched", Status: assessment.Pass},
	}}
	// `untouched` n'est pas dans la carte : il ne doit pas bouger.
	delete(degraded, "other_control")

	out, withdrawn := Degrade(in, degraded, map[string]string{"iam_policy_inline": "api:Read*"})
	if withdrawn != 1 {
		t.Errorf("un seul « pass » devait être retiré, %d l'ont été", withdrawn)
	}
	byControl := map[string]assessment.Result{}
	for _, r := range out.Results {
		byControl[r.Control] = r
	}
	if got := byControl["pass_control"].Status; got != assessment.NotEvaluated {
		t.Errorf("un pass sur une collecte incomplète doit devenir not-evaluated, obtenu %q", got)
	}
	if got := byControl["fail_control"].Status; got != assessment.Fail {
		t.Errorf("un écart OBSERVÉ reste observé : statut %q, attendu fail", got)
	}
	if got := byControl["na_control"].Status; got != assessment.NotApplicable {
		t.Errorf("un not-applicable vient du CONTRAT, pas de la collecte : statut %q", got)
	}
	if got := byControl["untouched"].Status; got != assessment.Pass {
		t.Errorf("un contrôle qui ne dépend d'aucune unité incomplète ne bouge pas : statut %q", got)
	}
}

// TestADegradedControlSaysWhatItBumpedInto : un « non évalué » muet n'apprend rien.
// Le cas le plus trompeur est celui du contrôle DÉJÀ non évalué : sa raison
// d'origine (« aucune ressource de ce type ») suggère un tenant vide, alors que la
// vérité est « l'API a refusé de les lister ».
func TestADegradedControlSaysWhatItBumpedInto(t *testing.T) {
	in := assessment.Assessment{Results: []assessment.Result{
		{Control: "ne_control", Status: assessment.NotEvaluated,
			Evidence: assessment.Evidence{Observed: "aucune ressource de type « iam_policy » dans l'inventaire évalué"}},
	}}
	unit := degradedIAM["iam_policy_no_wildcard_resource"]
	out, _ := Degrade(in, map[string]model.CollectionUnit{"ne_control": unit}, nil)

	reason := out.Results[0].Evidence.Observed
	if strings.Contains(reason, "aucune ressource") {
		t.Errorf("la raison trompeuse a survécu : %q", reason)
	}
	if !strings.Contains(reason, "iam_policy_inline") {
		t.Errorf("la raison doit NOMMER l'unité qui a manqué : %q", reason)
	}
}

// TestOnlyTheTypesOfAnIncompleteUnitAreDegraded : la dégradation suit le LIEN
// unité → type → contrôle. Sans ce lien, elle dégraderait tout au premier échec,
// ce qui reviendrait à rendre l'outil inutile dès qu'un droit manque.
func TestOnlyTheTypesOfAnIncompleteUnitAreDegraded(t *testing.T) {
	coll := model.Collection{}
	coll.Record(model.CollectionUnit{Unit: "iam_policy_inline", Types: []string{"iam_policy"},
		Attempted: true, Complete: false, Error: model.OutcomePermissionDenied})
	coll.Record(model.CollectionUnit{Unit: "compute_instance", Types: []string{"compute_instance"},
		Attempted: true, Complete: true})

	controlType := map[string]string{
		"iam_policy_no_wildcard_resource":      "iam_policy",
		"compute_instance_has_security_group":  "compute_instance",
		"objectstorage_bucket_public_access":   "object_storage_bucket",
		"control_not_declared_for_this_tenant": "iam_policy",
	}
	scope := map[string]bool{
		"iam_policy_no_wildcard_resource":      true,
		"compute_instance_has_security_group":  true,
		"objectstorage_bucket_public_access":   true,
		"control_not_declared_for_this_tenant": false, // hors périmètre du fournisseur
	}
	got := DegradedControls(coll, controlType, scope)

	if len(got) != 1 {
		t.Fatalf("un seul contrôle dépend de l'unité incomplète, %d dégradés : %v", len(got), SortedCodes(got))
	}
	if _, ok := got["iam_policy_no_wildcard_resource"]; !ok {
		t.Errorf("le contrôle qui lit iam_policy doit être dégradé, obtenu %v", SortedCodes(got))
	}
}

// TestACompleteCollectionDegradesNothing : la propriété de non-régression. Une
// collecte qui a tout lu ne doit RIEN déplacer — sans quoi le mécanisme, en
// devenant la règle, ferait plus de dégâts que le défaut qu'il corrige.
func TestACompleteCollectionDegradesNothing(t *testing.T) {
	coll := model.Collection{}
	coll.Record(model.CollectionUnit{Unit: "compute_instance", Types: []string{"compute_instance"},
		Attempted: true, Complete: true})
	got := DegradedControls(coll, map[string]string{"c": "compute_instance"}, map[string]bool{"c": true})
	if len(got) != 0 {
		t.Errorf("une collecte complète ne dégrade rien, obtenu %v", SortedCodes(got))
	}

	in := assessment.Assessment{Results: []assessment.Result{{Control: "c", Status: assessment.Pass}}}
	out, withdrawn := Degrade(in, got, nil)
	if withdrawn != 0 || out.Results[0].Status != assessment.Pass {
		t.Errorf("aucun statut ne devait bouger, obtenu %q (%d retraits)", out.Results[0].Status, withdrawn)
	}
}

// TestAPartiallyFailingUnitIsRecordedAsIncomplete : la fusion est PESSIMISTE. Deux
// specs peuvent alimenter le même type (règles entrantes et sortantes d'un groupe
// de sécurité) ; si l'une répond et l'autre pas, le type n'est pas entièrement lu.
func TestAPartiallyFailingUnitIsRecordedAsIncomplete(t *testing.T) {
	coll := model.Collection{}
	coll.Record(model.CollectionUnit{Unit: "security_group_rule", Types: []string{"security_group_rule"},
		Attempted: true, Complete: true})
	coll.Record(model.CollectionUnit{Unit: "security_group_rule", Types: []string{"security_group_rule"},
		Attempted: true, Complete: false, Error: model.OutcomeTruncated})

	if len(coll.Incomplete()) != 1 {
		t.Fatalf("l'unité doit être incomplète après un échec partiel : %+v", coll.Units)
	}
	if coll.Incomplete()[0].Error != model.OutcomeTruncated {
		t.Errorf("la classe d'échec doit être conservée, obtenu %q", coll.Incomplete()[0].Error)
	}
}

// TestAnUnattemptedUnitIsNotAFailure : ne pas tenter n'est pas échouer. Un
// fournisseur sans Kubernetes managé ne doit pas voir ses contrôles Kubernetes
// dégradés au motif qu'aucun cluster n'a été lu — il n'y avait rien à lire.
func TestAnUnattemptedUnitIsNotAFailure(t *testing.T) {
	coll := model.Collection{}
	coll.Record(model.CollectionUnit{Unit: "kubernetes_cluster", Types: []string{"kubernetes_cluster"},
		Attempted: false, Complete: false})
	if n := len(coll.Incomplete()); n != 0 {
		t.Errorf("une unité non tentée n'est pas une unité en échec, %d comptée(s)", n)
	}
}

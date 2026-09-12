package assess

import (
	"strings"
	"testing"

	"github.com/stephrobert/pepin/internal/i18n"
	"github.com/stephrobert/pepin/internal/model"
)

// La RAISON lue par l'opérateur, pour chacune des trois classes de refus.
//
// L'issue #91 ne portait pas sur le statut d'un contrôle — aucun `pass` ne
// sortait d'un refus, l'invariant de la vague 4 tenait. Elle portait sur la
// PHRASE : « service indisponible » là où la vérité était « la clé n'est pas
// reconnue ». Un statut juste avec une raison trompeuse coûte le temps de celui
// qui la lit, et c'est ce temps-là que ce test protège.
//
// Les trois phrases doivent envoyer vers TROIS gestes différents : corriger les
// identifiants, corriger les droits, attendre ou signaler une panne. Une phrase
// qui envoie vers le mauvais geste est un défaut, même adossée au bon statut.
func TestEachRefusalSendsTheOperatorToTheRightPlace(t *testing.T) {
	defer i18n.Set(i18n.Current())
	i18n.Set(i18n.FR)

	cas := []struct {
		classe   model.CollectionOutcome
		attendu  string   // ce que la phrase doit contenir
		interdit []string // ce qu'elle ne doit surtout pas dire
	}{
		{
			classe:   model.OutcomeUnauthenticated,
			attendu:  "identifiants",
			interdit: []string{"indisponible", "privilège"},
		},
		{
			classe:   model.OutcomeRejected,
			attendu:  "refusée",
			interdit: []string{"indisponible", "privilège"},
		},
		{
			classe:   model.OutcomePermissionDenied,
			attendu:  "privilège",
			interdit: []string{"indisponible"},
		},
		{
			classe:   model.OutcomeUnavailable,
			attendu:  "indisponible",
			interdit: []string{"privilège", "identifiants"},
		},
	}
	for _, c := range cas {
		got := OutcomeLabel(c.classe)
		if !strings.Contains(got, c.attendu) {
			t.Errorf("classe %q : libellé %q, attendu qu'il contienne %q", c.classe, got, c.attendu)
		}
		for _, mot := range c.interdit {
			if strings.Contains(got, mot) {
				t.Errorf("classe %q : le libellé %q contient %q — il envoie l'opérateur "+
					"corriger la mauvaise chose", c.classe, got, mot)
			}
		}
	}
}

// Le DROIT REQUIS ne se nomme que sur un refus de droit.
//
// C'est la moitié la plus facile à casser du correctif. Si `unauthenticated`
// venait à se ranger sous `permission_denied`, la phrase dirait « privilège
// insuffisant (droit requis : api:ReadVms) » à un opérateur dont la clé n'existe
// pas. Il irait attacher une politique à une identité absente — exactement le
// coût que l'issue #91 décrit, déplacé d'un cran.
func TestOnlyAMissingRightNamesTheRequiredGrant(t *testing.T) {
	defer i18n.Set(i18n.Current())
	i18n.Set(i18n.FR)

	const droit = "api:ReadVms"
	unite := func(o model.CollectionOutcome) model.CollectionUnit {
		return model.CollectionUnit{Unit: "security_group", Attempted: true, Error: o}
	}

	if got := IncompleteReason(unite(model.OutcomePermissionDenied), droit); !strings.Contains(got, droit) {
		t.Errorf("refus de DROIT : %q ne nomme pas le droit requis — c'est pourtant "+
			"la seule classe où le nommer aide", got)
	}
	for _, o := range []model.CollectionOutcome{
		model.OutcomeUnauthenticated,
		model.OutcomeRejected,
		model.OutcomeUnavailable,
		model.OutcomeTimeout,
	} {
		if got := IncompleteReason(unite(o), droit); strings.Contains(got, droit) {
			t.Errorf("classe %q : %q nomme le droit %q. Élargir une politique n'y "+
				"changerait rien, et le suggérer fait élargir des privilèges au hasard.",
				o, got, droit)
		}
	}
}

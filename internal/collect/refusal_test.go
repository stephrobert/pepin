package collect

import (
	"testing"

	"github.com/stephrobert/pepin/internal/model"
)

// Les TROIS refus d'Outscale, tels qu'ils ont été mesurés — et ils ne veulent
// pas dire la même chose.
//
// # Pourquoi cette mesure a coûté un compte réel
//
// L'issue #91 avait relevé, avec des identifiants SYNTHÉTIQUES, qu'Outscale
// refuse en 400 là où Scaleway refuse en 401 et Exoscale en 403. Elle s'était
// arrêtée là, et elle avait raison de s'arrêter : des identifiants synthétiques
// ne produisent qu'un seul des trois refus. Ils ne peuvent pas distinguer « la
// clé est inconnue » de « la clé est bonne mais le droit manque », et mapper
// l'un sur l'autre aurait remplacé un mensonge par le mensonge inverse.
//
// Les corps ci-dessous sont les réponses RÉELLES de l'OAPI eu-west-2, relevées
// le 2026-09-12 avec une identité EIM créée pour la mesure, ne portant que
// `api:ReadVms`, puis détruite. L'appel AUTORISÉ a été joué d'abord et a rendu
// 200 : sans ce témoin, un refus aurait pu venir de la signature, et la mesure
// n'aurait rien mesuré.
//
// # Ce que chaque ligne empêche
//
// Elles empêchent le classement par STATUT, qui se trompe ici dans les deux
// sens : l'échec d'authentification sort en 400 (rangé « service
// indisponible »), l'échec d'autorisation sort en 401 dans la table officielle
// (rangé « privilège insuffisant », ce qui est juste par accident). Le statut
// n'est pas le contrat ; le code d'erreur l'est.
func TestTheThreeMeasuredOutscaleRefusalsDoNotMeanTheSameThing(t *testing.T) {
	cas := []struct {
		nom    string
		status int
		body   string
		veut   model.CollectionOutcome
		parce  string
	}{
		{
			nom:    "clé d'accès inexistante — le cas des identifiants synthétiques",
			status: 400,
			body: `{"Errors":[{"Details":"","Type":"InvalidParameterValue","Code":"4120"}],` +
				`"ResponseContext":{"RequestId":"f7a55666-299a-4c32-8a47-61f0ce9ba5b1"}}`,
			veut:  model.OutcomeUnauthenticated,
			parce: "code 4120 = ErrorAuthenticationexception, documenté en 400",
		},
		{
			nom:    "clé existante, signature invalide",
			status: 401,
			body: `{"Errors":[{"Details":"","Type":"AccessDenied","Code":"1"}],` +
				`"ResponseContext":{"RequestId":"cd37784c-ba01-45ed-a6fd-f801d8665b09"}}`,
			veut:  model.OutcomeUnauthenticated,
			parce: "code 1 = ErrorAuthenticationFailed : la clé existe, elle signe mal",
		},
		{
			nom:    "clé valide, droit manquant — la mesure que l'issue réclamait",
			status: 403,
			body: `{"Errors":[{"Details":"You are not authorized to perform this action.",` +
				`"Type":"AccessDenied","Code":"4"}],` +
				`"ResponseContext":{"RequestId":"2d3016ea-5e06-422f-8508-a6ab22e5c9d4"}}`,
			veut: model.OutcomePermissionDenied,
			parce: "le seul des trois où élargir la politique du compte de scan " +
				"change quelque chose",
		},
	}
	for _, c := range cas {
		got, detail := Classify(&HTTPError{Status: c.status, Call: "POST /api/v1/ReadSecurityGroups", Body: c.body})
		if got != c.veut {
			t.Errorf("%s : classé %q, attendu %q (%s)", c.nom, got, c.veut, c.parce)
		}
		// Le corps voyage dans le détail : c'est lui que l'opérateur lit quand la
		// classe ne suffit pas, et il est scellé dans le bundle de preuve.
		if detail == "" {
			t.Errorf("%s : détail vide — la réponse du fournisseur doit être conservée", c.nom)
		}
	}
}

// Un 4xx n'est JAMAIS « service indisponible ».
//
// C'est le défaut nommé par l'issue #91, et il n'était pas cosmétique : le
// relevé de canari portait `classified: unavailable` sur les DIX-SEPT endpoints
// Outscale — tous répondant, aucun en panne. Un opérateur qui lit « service
// indisponible » va consulter une page d'état pendant que le plan de contrôle
// répond parfaitement.
//
// La borne haute compte autant : un 5xx, lui, reste `unavailable`, parce que là
// le service n'a vraiment pas répondu.
func TestAnAnsweringServiceIsNeverCalledUnavailable(t *testing.T) {
	refus := []int{400, 402, 405, 409, 413, 415, 418, 422, 451, 499}
	for _, s := range refus {
		got := outcomeForStatus(s)
		if got == model.OutcomeUnavailable {
			t.Errorf("HTTP %d classé %q : le service A RÉPONDU. Dire « indisponible » "+
				"envoie l'opérateur regarder une page d'état au lieu de sa requête.", s, got)
		}
		if got != model.OutcomeRejected {
			t.Errorf("HTTP %d classé %q, attendu %q", s, got, model.OutcomeRejected)
		}
	}

	// Les classes fines gardent la main sur la classe large.
	fines := map[int]model.CollectionOutcome{
		401: model.OutcomePermissionDenied,
		403: model.OutcomePermissionDenied,
		404: model.OutcomeNotFound,
		410: model.OutcomeNotFound,
		408: model.OutcomeTimeout,
		504: model.OutcomeTimeout,
		429: model.OutcomeRateLimited,
	}
	for s, veut := range fines {
		if got := outcomeForStatus(s); got != veut {
			t.Errorf("HTTP %d classé %q, attendu %q — la nouvelle classe large a "+
				"absorbé une classe fine", s, got, veut)
		}
	}

	// ET le contre-exemple : un service qui n'a vraiment pas répondu.
	for _, s := range []int{500, 502, 503} {
		if got := outcomeForStatus(s); got != model.OutcomeUnavailable {
			t.Errorf("HTTP %d classé %q, attendu %q — un 5xx est la seule chose "+
				"qu'« indisponible » doive désigner", s, got, model.OutcomeUnavailable)
		}
	}
}

// Un corps qui n'est pas l'enveloppe attendue ne change RIEN au classement.
//
// La lecture du corps est opportuniste par construction : elle raffine quand
// elle reconnaît un code documenté, et se tait autrement. Sans cette garantie,
// un proxy qui renvoie une page HTML, ou un fournisseur dont l'enveloppe
// ressemble de loin à celle d'Outscale, déplacerait des verdicts.
func TestAnUnrecognizedBodyLeavesTheStatusInCharge(t *testing.T) {
	corps := []struct{ nom, body string }{
		{"corps vide", ""},
		{"page HTML d'un proxy", "<html><body>502 Bad Gateway</body></html>"},
		{"JSON sans enveloppe d'erreur", `{"vms":[]}`},
		{"enveloppe présente mais JSON tronqué", `{"Errors":[{"Code":"4120"`},
		{"code inconnu de la table documentée", `{"Errors":[{"Type":"DefaultError","Code":"0"}]}`},
		{"bon code, mauvais type", `{"Errors":[{"Type":"DependencyProblem","Code":"4120"}]}`},
		{"Errors vide", `{"Errors":[]}`},
		{"Errors null", `{"Errors":null}`},
	}
	for _, c := range corps {
		if got := outcomeForAPIErrorCode(c.body); got != "" {
			t.Errorf("%s : le corps a rendu %q alors qu'il ne dit rien de reconnaissable.\n"+
				"  Raffiner sur un corps non reconnu, c'est deviner.", c.nom, got)
		}
		// Et de bout en bout : le statut garde la main.
		if got, _ := Classify(&HTTPError{Status: 400, Body: c.body}); got != model.OutcomeRejected {
			t.Errorf("%s : classé %q, attendu %q", c.nom, got, model.OutcomeRejected)
		}
	}
}

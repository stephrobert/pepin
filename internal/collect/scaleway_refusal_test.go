package collect

import (
	"testing"

	"github.com/stephrobert/pepin/internal/model"
)

// Le refus de SCALEWAY, classé par le type que son SDK définit.
//
// # Ce que l'issue #250 a relevé
//
// Le relevé de canari envoie des identifiants SYNTHÉTIQUES — donc, du point de
// vue du fournisseur, des identifiants qu'il ne connaît pas. Scaleway répond
// `401`, et un `401` seul se range `permission_denied` : « privilège insuffisant
// du compte de scan ». Le canari n'a pourtant AUCUNE clé. La phrase envoyait
// l'opérateur élargir une politique attachée à une identité qui n'existe pas —
// le défaut d'#91, transposé d'un fournisseur à l'autre.
//
// # Le corps ci-dessous est réel
//
// Mesuré le 2026-09-21 contre `api.scaleway.com`, sans compte ni identifiant, sur
// `GET /iam/v1alpha1/users`. Il est ici tel quel : une fixture réécrite à la main
// ne prouverait que la main qui l'a écrite.
func TestAnUnknownScalewayCredentialIsNotAMissingRight(t *testing.T) {
	const mesure = `{"message":"authentication is denied","method":"api_key",` +
		`"reason":"not_found","type":"denied_authentication"}`

	got, detail := Classify(&HTTPError{
		Status: 401,
		Call:   "GET https://api.scaleway.com/iam/v1alpha1/users",
		Body:   mesure,
	})
	if got != model.OutcomeUnauthenticated {
		t.Errorf("classé %q, attendu %q.\n"+
			"  Un 401 seul dit « privilège insuffisant » ; le type documenté dit que\n"+
			"  c'est l'IDENTIFIANT qui est refusé, et les deux mènent à des gestes\n"+
			"  opposés.", got, model.OutcomeUnauthenticated)
	}
	if detail == "" {
		t.Error("détail vide : la réponse du fournisseur doit être conservée")
	}
}

// Les quatre raisons que le SDK énumère désignent toutes l'identifiant.
//
// `DeniedAuthenticationError` porte un champ `Reason` dont le SDK sait rendre
// quatre valeurs. Aucune ne parle de droits : la clé est absente, mal formée,
// inconnue ou expirée. Le classement ne doit donc dépendre que du `type`, jamais
// de la raison — sans quoi une cinquième raison, ajoutée un jour par Scaleway,
// ferait retomber le refus sur son statut et rouvrirait le défaut en silence.
func TestEveryDocumentedAuthenticationReasonStaysUnauthenticated(t *testing.T) {
	raisons := []string{"unknown_reason", "invalid_argument", "not_found", "expired",
		"une_raison_que_scaleway_ajoutera_un_jour"}
	for _, r := range raisons {
		body := `{"type":"denied_authentication","method":"api_key","reason":"` + r + `"}`
		if got := outcomeForScalewayError(body); got != model.OutcomeUnauthenticated {
			t.Errorf("raison %q : classé %q, attendu %q — le type suffit, la raison ne "+
				"doit rien décider", r, got, model.OutcomeUnauthenticated)
		}
	}
}

// Un droit manquant reste un droit manquant, et c'est l'autre moitié de la paire.
//
// Sans ce contre-exemple, on pourrait classer TOUT refus Scaleway
// `unauthenticated` et faire passer le test précédent. La paire est ce qui prouve
// que la distinction est faite, et non qu'une seule branche a été câblée.
func TestAScalewayPermissionDenialIsNotACredentialProblem(t *testing.T) {
	const body = `{"message":"insufficient permissions","type":"permissions_denied"}`
	got, _ := Classify(&HTTPError{Status: 403, Call: "GET /iam/v1alpha1/api-keys", Body: body})
	if got != model.OutcomePermissionDenied {
		t.Errorf("classé %q, attendu %q — ici, et ici seulement, élargir la politique "+
			"du compte de scan change quelque chose", got, model.OutcomePermissionDenied)
	}
}

// Les deux dialectes coexistent sans se marcher dessus.
//
// L'enveloppe d'Outscale est une LISTE (`Errors[]`), celle de Scaleway est PLATE
// (`type` à la racine). Rien ne garantit a priori qu'un corps de l'un ne se
// décode pas partiellement comme l'autre : `encoding/json` ignore les champs
// absents, donc une enveloppe Outscale se décode en `scalewayErrorEnvelope` avec
// un `Type` VIDE. C'est précisément ce qui rend le silence obligatoire.
func TestOneProviderDialectNeverAnswersForAnother(t *testing.T) {
	outscale := `{"Errors":[{"Type":"AccessDenied","Code":"4"}],` +
		`"ResponseContext":{"RequestId":"x"}}`
	if got := outcomeForScalewayError(outscale); got != "" {
		t.Errorf("un corps Outscale a rendu %q par le chemin Scaleway", got)
	}

	scaleway := `{"type":"denied_authentication","reason":"not_found"}`
	if got := outcomeForOutscaleError(scaleway); got != "" {
		t.Errorf("un corps Scaleway a rendu %q par le chemin Outscale", got)
	}

	// Et de bout en bout, chacun reste classé par SON contrat.
	if got, _ := Classify(&HTTPError{Status: 403, Body: outscale}); got != model.OutcomePermissionDenied {
		t.Errorf("corps Outscale de bout en bout : %q", got)
	}
	if got, _ := Classify(&HTTPError{Status: 401, Body: scaleway}); got != model.OutcomeUnauthenticated {
		t.Errorf("corps Scaleway de bout en bout : %q", got)
	}
}

// EXOSCALE n'est PAS classé par son corps, et c'est une décision, pas un oubli.
//
// Mesuré le 2026-09-21 contre `api-ch-gva-2.exoscale.com/v2/instance` avec une
// clé synthétique : HTTP 403, corps `{"message":"Invalid key or request
// signature"}`. Aucun champ structuré — ni type, ni code, ni raison. Le seul
// discriminant serait le TEXTE du message.
//
// ADR-0023 l'interdit, et l'en-tête d'HTTPError dit pourquoi : « reconnaître un
// 403 en relisant le texte d'un message serait une correspondance de chaînes,
// donc une correspondance qui casse au premier changement de formulation ». Le
// refus d'Exoscale reste donc classé par son statut, ce qui est large et vrai
// plutôt que fin et fragile.
//
// Ce test garde la DÉCISION : si quelqu'un ajoute un jour la correspondance de
// chaînes, il rougit.
func TestExoscaleIsNotClassifiedByTheTextOfItsMessage(t *testing.T) {
	const mesure = `{"message":"Invalid key or request signature"}`
	if got := outcomeForAPIErrorCode(mesure); got != "" {
		t.Errorf("le corps d'Exoscale a rendu %q.\n"+
			"  Il ne porte aucun champ structuré : le classer demanderait de lire son\n"+
			"  MESSAGE, ce qu'ADR-0023 refuse. Son statut le classe, et c'est tout ce\n"+
			"  que le contrat permet d'affirmer aujourd'hui.", got)
	}
	// Le statut garde la main, et il est juste ici — 403 sur une clé refusée.
	if got, _ := Classify(&HTTPError{Status: 403, Body: mesure}); got != model.OutcomePermissionDenied {
		t.Errorf("classé %q, attendu %q par le statut seul", got, model.OutcomePermissionDenied)
	}
}

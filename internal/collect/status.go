package collect

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/stephrobert/pepin/internal/i18n"
	"github.com/stephrobert/pepin/internal/model"
)

// La CLASSIFICATION d'un échec de collecte, partagée par tous les collecteurs.
//
// Pourquoi une classe et pas le message. Le message d'erreur d'une API est de la
// donnée du fournisseur : il n'est ni stable, ni comparable d'un cloud à
// l'autre, ni traduit. Un pipeline, lui, a besoin de trancher entre « le compte
// de scan ne voit pas cette surface » (à corriger sur les droits) et « le
// service n'a pas répondu » (à réessayer). La classe est donc un identifiant
// stable ; le détail la complète, factuel, sans être interprété.
//
// Pourquoi une classification par ERREUR et pas par point d'appel. Un statut
// posé à la main à chaque appel est un statut qu'on oublie de poser au
// quarante-deuxième. La classification lit l'erreur telle qu'elle remonte, donc
// elle vaut pour tout collecteur présent et à venir, y compris ceux qui
// n'utilisent pas le moteur générique.

// HTTPError est une réponse d'API refusée, avec le STATUT qui la classe et
// l'appel qui l'a provoquée. Type dédié plutôt que fmt.Errorf : reconnaître un
// 403 en relisant le texte d'un message serait une correspondance de chaînes,
// donc une correspondance qui casse au premier changement de formulation.
type HTTPError struct {
	Status int
	Call   string // méthode + URL réellement émises
	Body   string
}

func (e *HTTPError) Error() string {
	if e.Call != "" {
		return fmt.Sprintf(i18n.T("HTTP %d sur %s : %s", "HTTP %d on %s: %s"), e.Status, e.Call, e.Body)
	}
	return fmt.Sprintf("HTTP %d : %s", e.Status, e.Body)
}

// TruncatedError signale une pagination interrompue : des items existent que le
// scan n'a pas vus. Distinct d'un échec d'appel — chaque page a répondu — et
// c'est la distinction qui compte, parce qu'une troncature rend une collection
// NON VIDE mais INCOMPLÈTE, le cas le plus trompeur de tous.
type TruncatedError struct {
	Call     string
	MaxPages int
}

func (e *TruncatedError) Error() string {
	return fmt.Sprintf(i18n.T(
		"pagination : borne de %d pages atteinte sur %s — collecte tronquée (vérifier la configuration de pagination)",
		"pagination: reached the %d-page bound on %s — truncated collection (check the pagination config)"),
		e.MaxPages, e.Call)
}

// Classify range une erreur de collecte dans sa classe et rend un DÉTAIL
// factuel, non traduit.
//
// Le détail n'est pas le message d'erreur rendu : il est reconstruit à partir
// des champs de l'erreur typée, parce qu'il est SCELLÉ dans le bundle de preuve.
// Un détail traduit ferait dire deux choses au même dossier selon la langue de
// celui qui l'a produit, et le digest du bundle changerait pour une raison qui
// ne regarde pas la posture — la même règle que pour les labels de traduction
// des règles, consommés puis retirés.
//
// L'ordre des tests n'est pas indifférent : les types propres à Pépin d'abord
// (ils portent le statut exact), puis les interfaces que les SDK tiers exposent,
// puis les causes de transport. Le repli est OutcomeUnavailable, jamais une
// classe devinée : « le service n'a pas répondu » est vrai de toute erreur non
// reconnue, alors qu'affirmer « droits insuffisants » sans le savoir enverrait
// l'utilisateur corriger une politique qui n'a rien à se reprocher.
func Classify(err error) (model.CollectionOutcome, string) {
	if err == nil {
		return "", ""
	}

	var he *HTTPError
	if errors.As(err, &he) {
		o := outcomeForStatus(he.Status)
		// Le CORPS avant le statut, quand il porte un code d'erreur documenté.
		// Chez Outscale le statut se trompe dans les deux sens (cf.
		// outcomeForAPIErrorCode), et un statut qui se trompe entraîne une
		// raison qui se trompe.
		if refined := outcomeForAPIErrorCode(he.Body); refined != "" {
			o = refined
		}
		return o, detailf("HTTP %d · %s · %s", he.Status, he.Call, he.Body)
	}
	var te *TruncatedError
	if errors.As(err, &te) {
		return model.OutcomeTruncated, detailf("pagination bound %d · %s", te.MaxPages, te.Call)
	}

	// Les SDK tiers (le client S3 de l'AWS SDK, ici) n'exposent pas leurs types
	// d'erreur sans qu'on importe leurs paquets. Leurs MÉTHODES, elles, sont un
	// contrat suffisant : on les reconnaît par interface anonyme, donc sans
	// dépendance nouvelle et sans correspondance de chaînes.
	var withStatus interface{ HTTPStatusCode() int }
	if errors.As(err, &withStatus) {
		if o := outcomeForStatus(withStatus.HTTPStatusCode()); o != "" {
			return o, detailf("HTTP %d · %s", withStatus.HTTPStatusCode(), err.Error())
		}
	}
	var withCode interface{ ErrorCode() string }
	if errors.As(err, &withCode) {
		switch withCode.ErrorCode() {
		case "InvalidAccessKeyId", "SignatureDoesNotMatch":
			// La clé n'est pas reconnue, ou elle signe mal : ce n'est pas un droit
			// qui manque. Élargir une politique n'y changerait rien.
			return model.OutcomeUnauthenticated, detailf("%s", err.Error())
		case "AccessDenied", "AccessDeniedException", "AllAccessDisabled", "Forbidden",
			"UnauthorizedOperation":
			return model.OutcomePermissionDenied, detailf("%s", err.Error())
		case "SlowDown", "RequestLimitExceeded", "TooManyRequests", "Throttling":
			return model.OutcomeRateLimited, detailf("%s", err.Error())
		}
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return model.OutcomeTimeout, detailf("%s", err.Error())
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return model.OutcomeTimeout, detailf("%s", err.Error())
	}
	var se *json.SyntaxError
	var ue *json.UnmarshalTypeError
	if errors.As(err, &se) || errors.As(err, &ue) {
		return model.OutcomeUnreadable, detailf("%s", err.Error())
	}
	return model.OutcomeUnavailable, detailf("%s", err.Error())
}

// outcomeForStatus classe un statut HTTP. 401 et 403 partagent la même classe :
// des deux côtés, le compte de scan ne voit pas la surface, et c'est la seule
// chose que le statut SEUL permette d'affirmer.
//
// Un 4xx non classé rend OutcomeRejected, jamais OutcomeUnavailable. Le service
// a répondu : c'est même la preuve qu'il fonctionne. Le ranger en « service
// indisponible » envoie l'opérateur consulter une page d'état pendant que le
// plan de contrôle répond parfaitement — le défaut mesuré par l'issue #91, qui
// touchait les dix-huit endpoints Outscale du relevé de canari.
func outcomeForStatus(status int) model.CollectionOutcome {
	switch {
	case status == 401 || status == 403:
		return model.OutcomePermissionDenied
	case status == 404 || status == 410:
		return model.OutcomeNotFound
	case status == 429:
		return model.OutcomeRateLimited
	case status == 408 || status == 504:
		return model.OutcomeTimeout
	case status >= 400 && status < 500:
		return model.OutcomeRejected
	case status >= 500:
		return model.OutcomeUnavailable
	}
	return ""
}

// apiErrorEnvelope est l'enveloppe d'erreur de l'API OUTSCALE (OAPI), telle que
// la réponse la porte. Elle est lue de façon OPPORTUNISTE : un corps qui n'a pas
// cette forme ne produit rien, et le statut garde le dernier mot.
type apiErrorEnvelope struct {
	Errors []struct {
		Type string `json:"Type"`
		Code string `json:"Code"`
	} `json:"Errors"`
}

// outcomeForAPIErrorCode raffine la classe à partir du CODE d'erreur que le
// fournisseur documente, pour les seuls cas où le statut HTTP induit en erreur.
//
// Source du contrat : table officielle des erreurs de l'API OUTSCALE,
// https://docs.outscale.com/api-errors.html (relevée le 2026-09-12).
//
//	Code 4120 · InvalidParameterValue · ErrorAuthenticationexception · HTTP 400
//	Code    1 · AccessDenied          · ErrorAuthenticationFailed    · HTTP 401
//	Code    5 · AccessDenied          · ErrorNotAuthorized           · HTTP 401
//
// Ces trois lignes disent pourquoi le statut ne suffit pas, et pourquoi il ne
// suffira jamais : l'échec d'AUTHENTIFICATION sort en 400 quand la clé est
// inconnue, et l'échec d'AUTORISATION sort en 401. Un classement par statut
// range donc le premier en « service indisponible » et le second en « privilège
// insuffisant » — les deux erreurs possibles, dans les deux sens.
//
// # Ce qui a été mesuré, et ce qui ne l'a pas été
//
// Mesuré le 2026-09-12 sur eu-west-2, avec une identité EIM créée pour
// l'occasion puis détruite, ne portant que `api:ReadVms` (issue #91) :
//
//	clé inexistante             → 400 · InvalidParameterValue · 4120
//	clé connue, secret faux     → 401 · AccessDenied          · 1
//	clé valide, droit manquant  → 403 · AccessDenied          · 4
//
// Le troisième cas — celui que l'issue réclamait et qu'aucun identifiant
// synthétique ne pouvait produire — n'est PAS dans la table publiée : le code 4
// n'y figure pas, et son voisin documenté `ErrorNotAuthorized` y est donné en
// 401. Il n'est donc pas mappé ici : le statut 403 le classe déjà
// permission_denied, ce qui est la bonne réponse, et affirmer un code absent de
// la doc serait l'inventer (CLAUDE.md §2).
//
// Tout code non listé retombe sur le statut. Une classe large et vraie vaut
// mieux qu'une classe fine et devinée.
func outcomeForAPIErrorCode(body string) model.CollectionOutcome {
	var env apiErrorEnvelope
	if err := json.Unmarshal([]byte(body), &env); err != nil {
		return ""
	}
	for _, e := range env.Errors {
		switch {
		case e.Code == "4120" && e.Type == "InvalidParameterValue":
			return model.OutcomeUnauthenticated
		case e.Code == "1" && e.Type == "AccessDenied":
			return model.OutcomeUnauthenticated
		}
	}
	return ""
}

// detailf borne le détail conservé dans l'état de collecte. Le corps d'une
// réponse d'erreur peut faire des kilo-octets (la page HTML d'un proxy) ; il est
// scellé dans le bundle de preuve, où il n'a pas à peser plus que l'inventaire.
func detailf(format string, args ...any) string {
	s := strings.Join(strings.Fields(fmt.Sprintf(format, args...)), " ")
	const limit = 400
	if len(s) <= limit {
		return s
	}
	// Découpe sur une frontière de rune : un détail tronqué au milieu d'un
	// caractère multi-octets produirait du JSON invalide dans le bundle.
	cut := limit
	for cut > 0 && !isRuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + "…"
}

// isRuneStart indique qu'un octet n'est pas une continuation UTF-8 (10xxxxxx).
func isRuneStart(b byte) bool { return b&0xC0 != 0x80 }

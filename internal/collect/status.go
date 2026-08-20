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
		return outcomeForStatus(he.Status), detailf("HTTP %d · %s · %s", he.Status, he.Call, he.Body)
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
		case "AccessDenied", "AccessDeniedException", "AllAccessDisabled", "Forbidden",
			"UnauthorizedOperation", "InvalidAccessKeyId", "SignatureDoesNotMatch":
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
// chose que l'utilisateur a à corriger.
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
	case status >= 400:
		return model.OutcomeUnavailable
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

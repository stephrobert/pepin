package assess

import (
	"fmt"
	"sort"

	"github.com/stephrobert/pepin/internal/i18n"
	"github.com/stephrobert/pepin/internal/model"
	"github.com/stephrobert/scankit/assessment"
)

// L'INCOMPLÉTUDE DE LA COLLECTE, appliquée au verdict.
//
// La doctrine « partial → not-evaluated » existait déjà, et plusieurs collecteurs
// la respectaient individuellement. Ce qui manquait, c'est d'en faire un
// INVARIANT plutôt qu'une bonne pratique appliquée au cas par cas : la règle Rego
// ne sait rien de ce qui n'a pas pu être lu, et il ne faut surtout pas le lui
// apprendre — une garde de capacité par règle est une garde qu'on oublie
// d'écrire à la cinquantième. C'est l'ASSESSMENT qui tranche, une fois, pour
// toutes les règles présentes et à venir.
//
// La dégradation est STRICTEMENT DIRECTIONNELLE, et c'est sa seule propriété
// intéressante : elle ne rend jamais un verdict plus affirmatif.
//
//	pass           → not-evaluated   (la seule transition qu'elle produit)
//	fail           → fail            (un écart observé reste observé)
//	not-applicable → not-applicable  (une déclaration de contrat, pas une mesure)
//	not-evaluated  → not-evaluated   (avec la VRAIE raison, ce qui est le gain)
//
// Pourquoi un `fail` survit à une collecte incomplète. Un écart a été VU : la
// donnée qui l'établit est arrivée. Le taire au motif que le reste manque ferait
// disparaître une non-conformité vraie — l'exact inverse de ce que cette vague
// cherche. L'incomplétude interdit d'affirmer l'absence d'écart, pas d'affirmer
// sa présence.
//
// Pourquoi un `not-applicable` y survit aussi. Il vient du CONTRAT du
// fournisseur — « ce mécanisme n'existe pas dans cette API » — et rien de ce
// qu'un scan lit ou ne lit pas ne peut le contredire. Le dégrader en
// « non évalué » remplacerait un fait justifié par une ignorance, ce qui est une
// perte d'information, pas un gain de prudence.

// DegradedControls rend, par code de contrôle, l'unité de collecte incomplète
// dont il dépend. `scope` restreint aux contrôles que ce scan pouvait conclure
// (déclarés pour le fournisseur, non déclarés non applicables) : compter les
// autres gonflerait le relevé de contrôles qui n'auraient rien conclu de toute
// façon.
//
// Fonction UNIQUE, appelée deux fois : par le relevé de capacités, imprimé AVANT
// tout verdict, et par la dégradation elle-même. Deux définitions divergeraient,
// et le relevé annoncerait alors un nombre que le rapport ne tiendrait pas.
func DegradedControls(coll model.Collection, controlType map[string]string, scope map[string]bool) map[string]model.CollectionUnit {
	out := map[string]model.CollectionUnit{}
	incomplete := coll.IncompleteTypes()
	if len(incomplete) == 0 {
		return out
	}
	for code := range scope {
		if !scope[code] {
			continue
		}
		typ := controlType[code]
		if typ == "" {
			// Un contrôle transverse (gouvernance) ne lit pas un type précis : aucune
			// unité ne le porte, donc aucune ne peut prouver qu'il est dégradé. Ne pas
			// le compter est l'énoncé prudent — l'affirmer serait inventer un lien.
			continue
		}
		if unit, ok := incomplete[typ]; ok {
			out[code] = unit
		}
	}
	return out
}

// Degrade applique l'incomplétude aux résultats. Passe POSTÉRIEURE, comme la
// provenance : elle lit un statut et n'en écrit qu'un seul, toujours dans le sens
// de la prudence. Elle rend aussi le nombre de « pass » retirés, qui est
// exactement ce qu'une collecte incomplète a coûté en certitude.
func Degrade(a assessment.Assessment, degraded map[string]model.CollectionUnit, grants map[string]string) (assessment.Assessment, int) {
	if len(degraded) == 0 {
		return a, 0
	}
	res := append([]assessment.Result(nil), a.Results...)
	withdrawn := 0
	for i := range res {
		unit, ok := degraded[res[i].Control]
		if !ok {
			continue
		}
		switch res[i].Status {
		case assessment.Pass:
			res[i].Status = assessment.NotEvaluated
			res[i].Evidence.Observed = IncompleteReason(unit, grants[unit.Unit])
			withdrawn++
		case assessment.NotEvaluated:
			// Le statut ne bouge pas ; la RAISON, si. « Aucune ressource de type
			// iam_policy dans l'inventaire évalué » est trompeur quand la vérité est
			// « l'API a refusé de les lister » : le premier suggère un tenant vide, le
			// second un compte de scan à corriger.
			res[i].Evidence.Observed = IncompleteReason(unit, grants[unit.Unit])
		default:
			// fail, not-applicable, exempted : inchangés (cf. l'en-tête du fichier).
		}
	}
	a.Results = res
	return a, withdrawn
}

// IncompleteReason explique, dans la langue de l'utilisateur, pourquoi un
// contrôle ne peut pas conclure. Elle NOMME l'unité et la classe d'échec : un
// « non évalué » qui ne dit pas sur quoi il bute n'apprend rien à qui doit le
// corriger, et c'est très exactement la critique que cette vague adresse aux
// CSPM qui rendent des rapports rassurants.
func IncompleteReason(u model.CollectionUnit, grant string) string {
	reason := fmt.Sprintf(i18n.T(
		"collecte incomplète de « %s » : %s — aucune conclusion possible sur ce périmètre",
		"incomplete collection of \"%s\": %s — no conclusion is possible on this scope"),
		u.Unit, OutcomeLabel(u.Error))
	// Le DROIT manquant, quand le descripteur le déclare et que l'échec vient bien
	// des droits. Le nommer sur un timeout enverrait corriger une politique qui n'a
	// rien à se reprocher — un CSPM qui fait élargir les privilèges au hasard fait
	// plus de mal que le contrôle qu'il n'a pas pu rendre.
	if grant != "" && u.Error == model.OutcomePermissionDenied {
		reason += fmt.Sprintf(i18n.T(" (droit requis : %s)", " (required grant: %s)"), grant)
	}
	return reason
}

// OutcomeLabel traduit une classe d'échec de collecte. La CLASSE est un
// identifiant stable, publié tel quel dans les formats analysables et scellé
// dans le bundle ; seul son libellé se traduit. Une classe inconnue rend sa
// propre valeur plutôt qu'un texte vide : un fournisseur futur ne doit pas faire
// disparaître la raison d'un « non évalué ».
func OutcomeLabel(o model.CollectionOutcome) string {
	switch o {
	case model.OutcomePermissionDenied:
		return i18n.T("privilège insuffisant du compte de scan",
			"insufficient privilege on the scanning account")
	case model.OutcomeNotFound:
		return i18n.T("service absent de ce tenant ou de cette région",
			"service absent from this tenant or region")
	case model.OutcomeRateLimited:
		return i18n.T("débit plafonné par l'API (inventaire tronqué)",
			"rate limited by the API (truncated inventory)")
	case model.OutcomeTimeout:
		return i18n.T("délai dépassé", "the request timed out")
	case model.OutcomeTruncated:
		return i18n.T("pagination interrompue (inventaire tronqué)",
			"pagination interrupted (truncated inventory)")
	case model.OutcomeUnreadable:
		return i18n.T("réponse illisible", "unreadable response")
	case model.OutcomeUnavailable:
		return i18n.T("service indisponible", "service unavailable")
	}
	return string(o)
}

// SortedCodes rend les codes d'une carte de contrôles dégradés, triés. Le
// déterminisme n'est pas cosmétique : ces codes s'impriment dans le relevé, qui
// est capturé par la documentation générée.
func SortedCodes(degraded map[string]model.CollectionUnit) []string {
	out := make([]string, 0, len(degraded))
	for code := range degraded {
		out = append(out, code)
	}
	sort.Strings(out)
	return out
}

package policy

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/stephrobert/pepin/internal/i18n"
	"github.com/stephrobert/pepin/referentiel"
)

// Relaxation est un ASSOUPLISSEMENT : un réglage qui s'écarte du défaut du
// côté où la contrainte normative ne tient plus. Elle porte de quoi le juger
// sans lire le code — le réglage, sa valeur par défaut, sa valeur effective, et
// la phrase qui dit ce qui est perdu.
type Relaxation struct {
	// Control : le contrôle dont la correspondance normative tombe.
	Control string `json:"control"`
	// Parameter : le réglage en cause (`section.champ`).
	Parameter string `json:"parameter"`
	// Constraint : le sens de contrainte qui a été rompu.
	Constraint string `json:"constraint"`
	// Default / Effective : les deux valeurs, rendues lisibles.
	Default   string `json:"default"`
	Effective string `json:"effective"`
	// DroppedReferences : les correspondances normatives ABANDONNÉES, sous leur
	// forme `framework:id`. C'est ce que l'assouplissement fait perdre.
	DroppedReferences []string `json:"dropped_references"`
}

// Sentence rend l'assouplissement en une phrase, dans la langue courante.
func (r Relaxation) Sentence() string { return r.SentenceIn(i18n.Current()) }

// SentenceIn est Sentence pour une langue explicite : la documentation bilingue
// rend les deux versions dans une seule exécution.
func (r Relaxation) SentenceIn(l i18n.Lang) string {
	return fmt.Sprintf(i18n.TIn(l,
		"%s : %s au lieu de %s (défaut) — %s",
		"%s: %s instead of %s (default) — %s"),
		r.Parameter, r.Effective, r.Default, constraintLossIn(l, r.Constraint))
}

// constraintLossIn dit CE QUI EST PERDU, par sens de contrainte. C'est la phrase
// qui empêche de lire un assouplissement comme un simple réglage.
func constraintLossIn(l i18n.Lang, kind string) string {
	switch kind {
	case referentiel.ConstraintAtMostDefault:
		return i18n.TIn(l,
			"une valeur au-delà du défaut tait ce que l'exigence demandait de voir",
			"a value beyond the default silences what the requirement asked to see")
	case referentiel.ConstraintSupersetOfDefault:
		return i18n.TIn(l,
			"retirer un membre du défaut, c'est cesser de vérifier ce que l'exigence demande",
			"dropping a member of the default means no longer checking what the requirement asks")
	case referentiel.ConstraintSubsetOfDefault:
		return i18n.TIn(l,
			"élargir au-delà du défaut, c'est accepter ce que l'exigence rejette",
			"widening beyond the default means accepting what the requirement rejects")
	case referentiel.ConstraintAtLeastAsStrict:
		return i18n.TIn(l,
			"une exigence du profil par défaut n'est plus tenue : elle a été retirée, ou ce qu'elle accepte a été élargi",
			"a requirement of the default profile is no longer held: it was dropped, or what it accepts was widened")
	default:
		return i18n.TIn(l, "contrainte non tenue", "constraint not met")
	}
}

// Relaxations rend, PAR CONTRÔLE, les assouplissements qui rompent ses
// correspondances normatives. `refs` donne, par contrôle, les correspondances
// qu'il perdrait — elles voyagent avec l'assouplissement pour que le rapport
// puisse nommer ce qui est abandonné, et pas seulement dire qu'il abandonne.
//
// Un réglage DURCI (plus d'étiquettes exigées, un délai plus court, un ensemble
// d'états acceptés plus étroit) n'est PAS un assouplissement : la correspondance
// tient, et rien n'est signalé. La contrainte ne dit pas « ne touche à rien »,
// elle dit de quel côté du défaut la promesse survit.
func Relaxations(res Resolved, constraints map[string][]referentiel.ConfigConstraint, refs map[string][]string) map[string][]Relaxation {
	def := Defaults()
	out := map[string][]Relaxation{}
	codes := make([]string, 0, len(constraints))
	for code := range constraints {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	for _, code := range codes {
		for _, c := range constraints[code] {
			if holds(res, def, c) {
				continue
			}
			out[code] = append(out[code], Relaxation{
				Control:           code,
				Parameter:         c.Parametre,
				Constraint:        c.Contrainte,
				Default:           render(def, c.Parametre),
				Effective:         render(res, c.Parametre),
				DroppedReferences: refs[code],
			})
		}
	}
	return out
}

// holds évalue une contrainte : la valeur effective est-elle du bon côté du
// défaut ? Un paramètre ou un sens de contrainte inconnu rend FAUX, donc
// signale un assouplissement : entre taire une contrainte qu'on ne sait pas
// évaluer et la signaler à tort, la seconde erreur est la seule réparable. Le
// cas est de toute façon refusé en amont par `mise run validate`.
func holds(res, def Resolved, c referentiel.ConfigConstraint) bool {
	switch c.Contrainte {
	case referentiel.ConstraintAtMostDefault:
		got, ok1 := ordinal(res, c.Parametre)
		want, ok2 := ordinal(def, c.Parametre)
		return ok1 && ok2 && got <= want
	case referentiel.ConstraintSupersetOfDefault:
		got, ok1 := members(res, c.Parametre)
		want, ok2 := members(def, c.Parametre)
		return ok1 && ok2 && contains(got, want)
	case referentiel.ConstraintSubsetOfDefault:
		got, ok1 := members(res, c.Parametre)
		want, ok2 := members(def, c.Parametre)
		return ok1 && ok2 && contains(want, got)
	case referentiel.ConstraintAtLeastAsStrict:
		got, ok1 := requirements(res, c.Parametre)
		want, ok2 := requirements(def, c.Parametre)
		return ok1 && ok2 && atLeastAsStrict(got, want)
	default:
		return false
	}
}

// atLeastAsStrict : pour CHAQUE exigence du profil par défaut, le profil effectif
// en porte une dont les écritures acceptées sont non vides et incluses dans celles
// du défaut. Tout ce qui satisfait l'exigence effective satisfait alors celle du
// défaut — c'est la définition exacte de « au moins aussi strict ».
func atLeastAsStrict(got, want []RequiredTag) bool {
	for _, w := range want {
		if !coveredBy(w, got) {
			return false
		}
	}
	return true
}

func coveredBy(want RequiredTag, got []RequiredTag) bool {
	accepted := make(map[string]bool, len(want.Keys))
	for _, k := range want.Keys {
		accepted[k] = true
	}
	for _, g := range got {
		if len(g.Keys) == 0 {
			continue
		}
		included := true
		for _, k := range g.Keys {
			if !accepted[k] {
				included = false
				break
			}
		}
		if included {
			return true
		}
	}
	return false
}

// requirements rend les EXIGENCES d'un réglage d'étiquetage (nom + écritures
// acceptées), et si le paramètre en porte.
func requirements(r Resolved, param string) ([]RequiredTag, bool) {
	switch param {
	case "tagging.required_tags":
		return r.Tagging.Required, true
	case "tagging.network_required_tags":
		return r.Tagging.NetworkRequired, true
	default:
		return nil, false
	}
}

// contains indique que `super` contient tous les membres de `sub`.
func contains(super, sub map[string]bool) bool {
	for k := range sub {
		if !super[k] {
			return false
		}
	}
	return true
}

// ordinal rend la valeur ORDONNÉE d'un réglage scalaire (un délai, un rang de
// seuil), et si le paramètre en porte une.
func ordinal(r Resolved, param string) (int, bool) {
	switch param {
	case "snapshots.max_age_days":
		return r.Snapshots.MaxAgeDays, true
	case "secrets.min_confidence":
		rank := rankOfConfidence(r.Secrets.MinConfidence)
		return rank, rank >= 0
	default:
		return 0, false
	}
}

// members rend l'ENSEMBLE des membres d'un réglage de type collection, et si le
// paramètre en porte un.
func members(r Resolved, param string) (map[string]bool, bool) {
	set := func(vals []string) map[string]bool {
		out := make(map[string]bool, len(vals))
		for _, v := range vals {
			out[v] = true
		}
		return out
	}
	switch param {
	case "tagging.resource_types":
		return set(r.Tagging.ResourceTypes), true
	case "snapshots.accepted_states":
		return set(r.Snapshots.AcceptedStates), true
	default:
		return nil, false
	}
}

// render rend la valeur d'un réglage sous une forme lisible par un humain, celle
// qui apparaît dans le rapport à côté du défaut.
func render(r Resolved, param string) string {
	if n, ok := ordinal(r, param); ok {
		switch param {
		case "snapshots.max_age_days":
			return strconv.Itoa(n) + i18n.T(" jours", " days")
		case "secrets.min_confidence":
			return r.Secrets.MinConfidence
		}
	}
	if reqs, ok := requirements(r, param); ok {
		// Nom ET écritures acceptées : deux profils qui exigent les mêmes noms mais
		// pas les mêmes écritures ne sont PAS la même exigence, et un rapport qui
		// afficherait deux fois la même chaîne ne dirait pas ce qui a changé.
		vals := make([]string, 0, len(reqs))
		for _, t := range reqs {
			vals = append(vals, t.Name+"{"+strings.Join(t.Keys, "|")+"}")
		}
		sort.Strings(vals)
		return "[" + strings.Join(vals, ", ") + "]"
	}
	if m, ok := members(r, param); ok {
		vals := make([]string, 0, len(m))
		for k := range m {
			vals = append(vals, k)
		}
		sort.Strings(vals)
		return "[" + strings.Join(vals, ", ") + "]"
	}
	return i18n.T("(réglage inconnu)", "(unknown setting)")
}

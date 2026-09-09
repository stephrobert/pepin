// Package profile porte les PROFILS DE SCAN : ce qu'une porte de CI retient, sans
// jamais retirer quoi que ce soit du rapport.
//
// # Le problème
//
// Un premier scan doit provoquer « ah oui, ça c'est intéressant », pas « oui, je sais
// que ma VM de test n'a pas de protection contre la suppression ». Une VM publique
// dont SSH est ouvert à Internet et un volume sans snapshot récente sont tous deux
// `high` : le second est un faux positif ASSUMÉ — la règle le documente elle-même —
// et lui donner le poids du premier fait douter du premier.
//
// # Ce qu'un profil NE fait PAS, et c'est le cœur
//
// Il ne cache RIEN. Le rapport reste complet dans tous les formats, et c'est la
// doctrine déjà appliquée aux dérogations et aux constats d'incertitude : « le rapport
// dit tout, seule la porte tient compte des dérogations ». Un profil qui retirerait
// des écarts du rapport transformerait le silence en faux vert — précisément ce que
// ce projet passe son temps à combattre, et ce serait pire ici, parce que ça viendrait
// d'un réglage plutôt que d'un défaut de règle, donc ce serait invisible.
//
// Ce qu'il change est le POIDS dans la porte de CI, et il le DIT :
//
//   - les écarts hors profil ne rendent pas 1 ;
//   - mais un écart critical/high mis de côté empêche de rendre 0 : le scan sort en
//     3, « le scan n'établit pas la conformité », qui est exactement ce qu'un scan
//     volontairement partiel est (ADR-0005, et aucun cinquième code) ;
//   - le nombre et les codes mis de côté sont publiés.
//
// # Le défaut ne change pas
//
// `all` reste le profil par défaut. Changer un défaut ferait passer au vert, en
// silence et sans que personne ne l'ait décidé, une chaîne qui échoue aujourd'hui.
package profile

import (
	"sort"
	"strings"

	"github.com/stephrobert/scankit/finding"
)

// Noms des profils. `all` est le défaut et ne filtre rien.
const (
	All         = "all"
	Security    = "security"
	Compliance  = "compliance"
	Sovereignty = "sovereignty"
)

// Names énumère les profils, dans l'ordre où l'aide les présente.
func Names() []string { return []string{All, Security, Compliance, Sovereignty} }

// def décrit ce qu'un profil RETIENT dans la porte.
type def struct {
	// categories : les valeurs de `labels.category` retenues. Vide = toutes.
	categories map[string]bool
	// confidences : les valeurs de `labels.confidence` retenues. Vide = toutes.
	confidences map[string]bool
}

// Les profils sont définis sur les DEUX dimensions que porte un finding depuis #111.
// Filtrer sur la seule sévérité mélangerait encore « grave » et « certain », ce qui
// est le défaut que la confiance existe pour corriger.
var defs = map[string]def{
	All: {},
	// L'exposition et les secrets, quand la règle est sûre de ce qu'elle avance. Un
	// écart `contextual` dépend d'un contexte que le scan ne voit pas : il reste au
	// rapport, il ne casse pas une chaîne.
	Security: {
		categories:  set("security"),
		confidences: set("confirmed", "probable"),
	},
	// L'audit normatif : ce qui se défend devant un référentiel, y compris l'hygiène
	// documentaire sans laquelle un auditeur ne peut rien retracer.
	Compliance: {categories: set("compliance", "hygiene")},
	// La localisation et l'extraterritorialité — la raison d'être du produit.
	Sovereignty: {categories: set("sovereignty")},
}

func set(vals ...string) map[string]bool {
	m := make(map[string]bool, len(vals))
	for _, v := range vals {
		m[v] = true
	}
	return m
}

// Valid dit si le nom est un profil connu.
func Valid(name string) bool {
	_, ok := defs[name]
	return ok
}

// Retains dit si un finding pèse dans la porte de ce profil.
//
// Un finding dont le label manque est RETENU. Le repli penche vers plus de sévérité :
// une étiquette absente ne doit pas faire disparaître un écart d'une porte de CI, ce
// qui serait exactement le silence-devenu-vert que ce paquet refuse.
func Retains(name string, f finding.Finding) bool {
	d, ok := defs[name]
	if !ok {
		return true
	}
	if len(d.categories) > 0 {
		if c := f.Label("category"); c != "" && !d.categories[c] {
			return false
		}
	}
	if len(d.confidences) > 0 {
		if c := f.Label("confidence"); c != "" && !d.confidences[c] {
			return false
		}
	}
	return true
}

// Split partage les findings entre ceux que la porte retient et ceux qu'elle met de
// côté. L'ordre d'origine est conservé dans les deux.
func Split(name string, findings []finding.Finding) (retenus, ecartes []finding.Finding) {
	for _, f := range findings {
		if Retains(name, f) {
			retenus = append(retenus, f)
			continue
		}
		ecartes = append(ecartes, f)
	}
	return retenus, ecartes
}

// SetAside résume ce qu'un profil a mis de côté : les codes concernés, triés et
// dédoublonnés, et s'il en reste un de sévérité critical ou high.
//
// Le second est ce qui empêche de rendre 0 : mettre de côté un écart grave est une
// décision de lecture, pas une absence d'écart.
func SetAside(ecartes []finding.Finding) (codes []string, grave bool) {
	vus := map[string]bool{}
	for _, f := range ecartes {
		if !vus[f.Code] {
			vus[f.Code] = true
			codes = append(codes, f.Code)
		}
		if s := strings.ToLower(f.Severity); s == "critical" || s == "high" {
			grave = true
		}
	}
	sort.Strings(codes)
	return codes, grave
}

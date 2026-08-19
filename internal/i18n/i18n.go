// Package i18n résout la langue de l'interface et sert de sélecteur entre les
// deux versions d'une même chaîne.
//
// Pépin est bilingue : le français est la langue de RÉFÉRENCE du contenu
// normatif (référentiel, messages de règles), l'anglais en est la traduction
// maintenue en parallèle. Une seule chose est résolue ici, laquelle des deux
// s'affiche, et elle l'est UNE fois, au démarrage, avant que cobra ne fige les
// chaînes d'aide.
//
// Ordre de résolution, du plus explicite au plus implicite :
//
//	--lang=fr|en  →  PEPIN_LANG  →  LC_ALL  →  LANG  →  repli : en
//
// La première source NON VIDE décide, et elle décide seule : c'est la règle
// POSIX (LC_ALL l'emporte sur LANG), et c'est aussi la seule qui ne surprenne
// pas : un LC_ALL posé volontairement ne doit pas être contourné par un LANG
// resté d'une session précédente. Une valeur que Pépin ne parle pas
// (LC_ALL=de_DE.UTF-8, LC_ALL=C.UTF-8) retombe sur l'anglais, sans erreur : un
// outil de posture ne refuse pas de tourner parce qu'il ne connaît pas une
// locale.
package i18n

import (
	"strings"
	"sync/atomic"
)

// Lang est une langue d'interface. Deux valeurs seulement : toute autre entrée
// est normalisée vers EN par Parse.
type Lang string

const (
	// FR est le français, langue de référence du contenu normatif.
	FR Lang = "fr"
	// EN est l'anglais, et le repli de toute locale inconnue.
	EN Lang = "en"
)

// EnvVars nomme, DANS L'ORDRE DE PRIORITÉ, les variables d'environnement
// consultées après le drapeau --lang. Exportée pour que l'aide de la CLI et la
// documentation citent la liste réellement appliquée, jamais une recopie.
var EnvVars = []string{"PEPIN_LANG", "LC_ALL", "LANG"}

// current porte la langue résolue du processus. atomic parce que le rendu et
// les tests la lisent depuis plusieurs goroutines (`go test -race`) ; elle
// n'est écrite qu'au démarrage, ou par un test qui restaure ensuite.
var current atomic.Value // Lang

func init() { current.Store(EN) }

// frenchTags : les sous-tags primaires qui désignent le français. Un ENSEMBLE
// EXACT, pas un préfixe : `strings.HasPrefix(tag, "fr")` ferait passer « frisian »
// (`fy` en BCP 47, mais `fri`/`frr` existent en ISO 639-2) et n'importe quelle
// valeur fantaisiste commençant par « fr » pour du français.
var frenchTags = map[string]bool{"fr": true, "fra": true, "fre": true}

// Parse normalise une valeur de locale en langue d'interface. Elle reconnaît la
// forme nue (`fr`), la forme POSIX (`fr_FR.UTF-8`, `fr_BE@euro`) et la forme
// BCP 47 (`fr-CA`) ; tout le reste, valeur vide comprise, donne EN.
func Parse(v string) Lang {
	if frenchTags[primaryTag(v)] {
		return FR
	}
	return EN
}

// primaryTag extrait le sous-tag primaire d'une valeur de locale, en minuscules :
// « fr_FR.UTF-8 » → « fr », « en-GB » → « en », « C.UTF-8 » → « c ».
func primaryTag(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	for _, sep := range []string{".", "@", "_", "-"} {
		if i := strings.Index(v, sep); i >= 0 {
			v = v[:i]
		}
	}
	return v
}

// Resolve applique l'ordre de résolution complet. `flag` est la valeur de
// --lang (vide s'il est absent) ; `getenv` lit l'environnement (os.Getenv en
// production, une carte en test). La première source non vide décide.
func Resolve(flag string, getenv func(string) string) Lang {
	if strings.TrimSpace(flag) != "" {
		return Parse(flag)
	}
	if getenv == nil {
		return EN
	}
	for _, name := range EnvVars {
		if v := getenv(name); strings.TrimSpace(v) != "" {
			return Parse(v)
		}
	}
	return EN
}

// Current retourne la langue résolue du processus (EN tant que Set n'a pas été
// appelée : un repli sûr vaut mieux qu'une valeur vide qui traverse le rendu).
func Current() Lang {
	l, _ := current.Load().(Lang)
	if l != FR {
		return EN
	}
	return FR
}

// Set fixe la langue du processus. Appelée une fois au démarrage, après la
// lecture de --lang et avant que cobra ne construise l'aide.
func Set(l Lang) {
	if l != FR {
		l = EN
	}
	current.Store(l)
}

// T choisit entre la chaîne française (référence) et sa contrepartie anglaise,
// selon la langue courante. Les deux versions se tiennent côte à côte dans le
// code : une traduction qu'on ne voit pas en modifiant l'original est une
// traduction qui dérive.
func T(fr, en string) string { return TIn(Current(), fr, en) }

// TIn est T pour une langue explicite. Utile là où une même exécution doit
// produire les deux langues (génération de la documentation bilingue).
func TIn(l Lang, fr, en string) string {
	if l == FR {
		return fr
	}
	return en
}

// Pick retourne `en` si la langue courante est l'anglais ET que la traduction
// existe ; sinon la version française. C'est la dégradation propre du contenu
// TRADUIT EN DONNÉE (référentiel, labels des règles), là où une chaîne peut
// manquer : une chaîne anglaise absente rend le français, jamais du vide.
// L'absence elle-même est refusée en CI, pas au runtime.
func Pick(fr, en string) string { return PickIn(Current(), fr, en) }

// PickIn est Pick pour une langue explicite.
func PickIn(l Lang, fr, en string) string {
	if l == EN && strings.TrimSpace(en) != "" {
		return en
	}
	return fr
}

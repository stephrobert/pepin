package cmd

import (
	"fmt"
	"io"

	"github.com/stephrobert/pepin/internal/assess"
	"github.com/stephrobert/pepin/internal/model"
)

// LE RELEVÉ DE CAPACITÉS : ce que ce scan a pu observer, et ce qu'il n'a pas pu.
//
// Le problème qu'il ferme (issue #45). Les droits du compte qui lance Pépin sont
// une SURFACE DE SÉCURITÉ. Un compte qui ne voit pas quelque chose ne doit jamais
// permettre de conclure que cette chose n'existe pas — c'est le travers classique
// du CSPM, dont le rapport devient d'autant plus rassurant qu'il a moins de
// droits.
//
// # Pourquoi le relevé est DÉRIVÉ de la collecte, et non d'une sonde préalable
//
// L'issue demande un relevé « en tête de scan ». La lecture littérale — sonder
// chaque endpoint avant de collecter — a trois défauts, et le premier est
// rédhibitoire :
//
//  1. Une sonde qui réussit puis un appel réel qui échoue produit un relevé QUI
//     MENT. C'est la même règle que pour la provenance des attributs, posée au lot
//     précédent : on ne nomme jamais un appel qui n'a pas eu lieu, et on n'annonce
//     pas une capacité qu'on n'a pas exercée.
//  2. Sonder double le nombre de requêtes, donc l'exposition au plafonnement de
//     débit — lequel est lui-même une cause de collecte incomplète.
//  3. La sonde et l'appel réel ne portent pas les mêmes paramètres (pagination,
//     jointures) : un droit peut suffire à l'un et pas à l'autre.
//
// Le relevé est donc le RENDU de l'état de collecte réel, imprimé AVANT le
// rapport et avant tout verdict — ce que l'exigence demande vraiment : que
// personne ne lise un verdict sans savoir sur quoi il porte.
//
// # Pourquoi les unités ne sont pas traduites
//
// Une unité est nommée par le type de ressource normalisé (`object_storage_bucket`),
// le vocabulaire déjà gelé par l'inventaire et utilisé par la matrice de
// couverture. Lui donner un libellé traduit créerait un second vocabulaire à
// maintenir, qui ne correspondrait à aucune autre page. Ce qui se traduit, c'est
// la RAISON de l'échec et la phrase de synthèse.

// renderCapabilities écrit le relevé. `live` force son affichage : une collecte
// live doit annoncer ce qu'elle a su lire même quand tout s'est bien passé, parce
// que c'est là que les droits du compte décident de ce qui sera conclu. Hors live,
// le relevé n'apparaît que s'il a quelque chose à dire — un bandeau qui s'affiche à
// chaque exécution est un bandeau que personne ne lit, et l'avertissement qu'il
// portera un jour se dévalue d'autant.
//
// Quand il s'affiche, il montre TOUT : les unités qui ont répondu autant que celles
// qui ont échoué. Le contraste est l'information — « trois lues sur quatre » se
// comprend, « une a échoué » laisse le lecteur deviner l'étendue de ce qui manque.
func renderCapabilities(w io.Writer, coll model.Collection, degraded map[string]model.CollectionUnit, live bool) {
	if coll.Empty() {
		return
	}
	if !live && len(coll.Incomplete()) == 0 && len(coll.Unmapped) == 0 {
		return
	}
	_, _ = fmt.Fprintf(w, "\n%s\n", tr("Relevé de capacités du collecteur",
		"Collector capability report"))
	for _, u := range coll.Units {
		if !u.Attempted {
			// Une unité non tentée n'est ni un succès ni un échec : le fournisseur
			// n'expose pas ce service. La lister la ferait passer pour l'un des deux.
			continue
		}
		if u.Complete {
			_, _ = fmt.Fprintf(w, "  ✓ %s\n", u.Unit)
			continue
		}
		_, _ = fmt.Fprintf(w, "  ✗ %s — %s\n", u.Unit, assess.OutcomeLabel(u.Error))
		// Le détail brut de l'API, sous la ligne qui le résume : la classe dit quoi
		// faire, le détail dit à qui le demander.
		if u.Detail != "" {
			_, _ = fmt.Fprintf(w, "    %s\n", u.Detail)
		}
	}
	for _, t := range coll.Unmapped {
		_, _ = fmt.Fprintf(w, "  · %s\n", fmt.Sprintf(tr(
			"%s × %d : type présent dans la source, projeté par aucune spec — non audité",
			"%s x %d: type present in the source, projected by no spec — not audited"), t.Type, t.Count))
	}
	if n := len(degraded); n > 0 {
		_, _ = fmt.Fprintf(w, "%s\n", fmt.Sprintf(tr(
			"Résultat : %d contrôle(s) ne pourront pas être évalués sur ce périmètre.",
			"Result: %d control(s) cannot be evaluated on this scope."), n))
		for _, code := range assess.SortedCodes(degraded) {
			_, _ = fmt.Fprintf(w, "  · %s\n", code)
		}
	}
}

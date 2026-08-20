package model

import "sort"

// L'ÉTAT DE COLLECTE : ce que le scan a réellement pu lire, unité par unité.
//
// Le défaut qu'il ferme. Un collecteur qui rencontre un 403 sur un endpoint et
// poursuit produit une collection VIDE là où l'API n'a rien voulu dire. Les
// règles ne se déclenchent pas sur du vide, aucun finding n'est émis, et le
// contrôle conclut « conforme » sur un périmètre que personne n'a lu. Un faux
// vert est invisible par construction : c'est le pire défaut possible pour un
// outil de posture.
//
// La correction ne peut pas être « chaque collecteur fait attention ». Elle
// existait déjà sous cette forme — le kit de collecte avalait l'erreur du
// Kubernetes managé en avertissement sur stderr, avec le bon comportement CODÉ
// EN DUR pour ce seul cas. Ce qu'il fallait, c'est un INVARIANT : tout échec de
// collecte est ENREGISTRÉ (pas seulement journalisé), il voyage avec
// l'inventaire, et c'est l'assessment qui en tire les conséquences.
//
// # Ce que l'unité désigne
//
// Une unité de collecte est un endpoint (ou une chaîne d'endpoints) qui alimente
// un ou plusieurs TYPES de ressources normalisés. Elle est nommée par un
// identifiant stable et agnostique — le même vocabulaire que l'inventaire et que
// la matrice de couverture — jamais par une prose traduite : l'état de collecte
// est scellé dans le bundle de preuve, et un dossier qui dirait deux choses
// selon la langue de son auteur ne serait pas opposable.
//
// # Ce que « complète » veut dire
//
// `Complete` est vrai quand l'unité a rendu TOUT ce que l'API avait à rendre.
// Une unité qui a rendu zéro ressource sans erreur est complète : « il n'y a
// rien » est une mesure. Une unité qui a rendu cent ressources sur mille avant
// un 403 ne l'est pas : une réponse partielle n'est pas une réponse.

// CollectionOutcome classe la RAISON d'une collecte incomplète. Une classe, pas
// un message : un pipeline doit pouvoir distinguer « droits insuffisants » (à
// corriger sur le compte de scan) de « service indisponible » (à réessayer), et
// un message d'API n'est pas comparable d'un fournisseur à l'autre.
type CollectionOutcome string

// Les classes d'échec que Pépin sait distinguer. Toute autre cause retombe sur
// OutcomeUnavailable : mieux vaut une classe large et vraie qu'une classe fine
// et devinée.
const (
	// OutcomePermissionDenied : l'API a refusé (401/403). Le compte de scan ne
	// voit pas cette surface — le cas que l'issue #45 nomme « privilège
	// insuffisant », et celui qui rend un rapport faussement rassurant.
	OutcomePermissionDenied CollectionOutcome = "permission_denied"
	// OutcomeNotFound : l'endpoint n'existe pas sur ce tenant/cette région (404).
	OutcomeNotFound CollectionOutcome = "not_found"
	// OutcomeRateLimited : l'API a plafonné le débit (429). L'inventaire est
	// tronqué, pas vide.
	OutcomeRateLimited CollectionOutcome = "rate_limited"
	// OutcomeTimeout : la requête n'a pas abouti dans le délai imparti.
	OutcomeTimeout CollectionOutcome = "timeout"
	// OutcomeTruncated : la pagination s'est interrompue (borne de pages
	// atteinte). Des items existent que le scan n'a pas vus.
	OutcomeTruncated CollectionOutcome = "truncated"
	// OutcomeUnreadable : la réponse n'était pas le document attendu (JSON
	// invalide, schéma inconnu).
	OutcomeUnreadable CollectionOutcome = "unreadable"
	// OutcomeUnavailable : le service n'a pas répondu (5xx, erreur de transport,
	// cause non classée).
	OutcomeUnavailable CollectionOutcome = "unavailable"
)

// CollectionUnit est l'état d'UNE unité de collecte.
type CollectionUnit struct {
	// Unit : identifiant stable et agnostique de l'unité (ex. "iam_policy",
	// "object_storage_bucket", "iam_policy_inline").
	Unit string `json:"unit"`
	// Types : les types de ressources normalisés que cette unité alimente. C'est
	// par ce lien que l'assessment sait quels contrôles dépendent d'elle.
	Types []string `json:"types,omitempty"`
	// Attempted : la collecte a réellement été tentée. Faux pour une unité
	// déclarée mais non exécutée (endpoint non configuré pour ce fournisseur).
	Attempted bool `json:"attempted"`
	// Complete : l'unité a rendu tout ce que l'API avait à rendre.
	Complete bool `json:"complete"`
	// Error : la classe d'échec, vide quand l'unité est complète.
	Error CollectionOutcome `json:"error,omitempty"`
	// Detail : ce que l'API a répondu, tel quel. C'est une DONNÉE du fournisseur,
	// pas de la prose de Pépin : elle ne se traduit pas.
	Detail string `json:"detail,omitempty"`
}

// UnmappedType est un type de ressource présent dans la source mais qu'aucune
// spec ne projette : Pépin le VOIT sans savoir le lire.
//
// Ce n'est PAS une collecte incomplète, et la distinction porte tout le sens.
// Un 403 empêche de lire une donnée dont un contrôle a besoin : le périmètre
// promis n'a pas été lu. Un type non projeté est une donnée dont AUCUN contrôle
// n'a besoin : Pépin n'a jamais prétendu l'auditer. Faire échouer une porte de
// CI dessus punirait un utilisateur pour avoir des ressources hors couverture,
// et une porte rouge en permanence est une porte qu'on apprend à ignorer.
// L'exigence de l'issue #43 est donc tenue à la lettre — « jamais ignoré en
// silence » — sans être transformée en verdict.
type UnmappedType struct {
	// Type : le type natif de la source (ex. "outscale_public_ip").
	Type string `json:"type"`
	// Count : combien de ressources de ce type la source portait.
	Count int `json:"count"`
}

// Collection est l'état de collecte complet d'un scan. Il VOYAGE avec
// l'inventaire (donc dans l'input.json d'un bundle scellé) : un dossier de
// preuve qui ne dirait pas ce qu'il n'a pas pu lire promettrait plus qu'il ne
// tient, et `verify --re-derive` rejouerait un verdict plus affirmatif que
// l'original.
type Collection struct {
	// Units : les unités de collecte, triées par identifiant (déterminisme :
	// l'empreinte du bundle ne doit pas dépendre de l'ordre d'itération d'une map).
	Units []CollectionUnit `json:"units,omitempty"`
	// Unmapped : les types de la source qu'aucune spec ne projette, triés.
	Unmapped []UnmappedType `json:"unmapped,omitempty"`
}

// Record enregistre (ou fusionne) l'état d'une unité. La fusion est PESSIMISTE :
// une unité alimentée par plusieurs appels dont l'un échoue est incomplète. Deux
// specs peuvent viser le même type normalisé (les règles entrantes et sortantes
// d'un groupe de sécurité en sont un cas réel) ; si l'une des deux ne répond
// pas, le type n'est pas entièrement lu, et c'est le seul énoncé vrai.
func (c *Collection) Record(u CollectionUnit) {
	for i := range c.Units {
		if c.Units[i].Unit != u.Unit {
			continue
		}
		c.Units[i].Attempted = c.Units[i].Attempted || u.Attempted
		c.Units[i].Types = mergeSorted(c.Units[i].Types, u.Types)
		if !u.Complete && c.Units[i].Complete {
			c.Units[i].Complete = false
			c.Units[i].Error = u.Error
			c.Units[i].Detail = u.Detail
		}
		return
	}
	u.Types = mergeSorted(nil, u.Types)
	c.Units = append(c.Units, u)
	sort.Slice(c.Units, func(i, j int) bool { return c.Units[i].Unit < c.Units[j].Unit })
}

// RecordUnmapped enregistre un type de la source qu'aucune spec ne projette.
func (c *Collection) RecordUnmapped(typ string, count int) {
	if typ == "" || count <= 0 {
		return
	}
	for i := range c.Unmapped {
		if c.Unmapped[i].Type == typ {
			c.Unmapped[i].Count += count
			return
		}
	}
	c.Unmapped = append(c.Unmapped, UnmappedType{Type: typ, Count: count})
	sort.Slice(c.Unmapped, func(i, j int) bool { return c.Unmapped[i].Type < c.Unmapped[j].Type })
}

// Empty indique qu'aucun état de collecte n'a été enregistré. Une source qui ne
// mesure rien (un export JSON reçu d'un tiers) n'en produit pas : Pépin n'a pas
// collecté cet inventaire, il l'a reçu, et il n'invente pas d'attestation pour
// lui — la même règle que pour la provenance des attributs.
func (c Collection) Empty() bool { return len(c.Units) == 0 && len(c.Unmapped) == 0 }

// Incomplete rend les unités dont la collecte a échoué ou s'est tronquée.
func (c Collection) Incomplete() []CollectionUnit {
	var out []CollectionUnit
	for _, u := range c.Units {
		if u.Attempted && !u.Complete {
			out = append(out, u)
		}
	}
	return out
}

// IncompleteTypes indexe, par type de ressource normalisé, l'unité incomplète
// qui aurait dû l'alimenter. C'est le lien par lequel l'assessment fait passer
// un contrôle en « non évalué » sans qu'aucune règle n'ait à le savoir.
func (c Collection) IncompleteTypes() map[string]CollectionUnit {
	out := map[string]CollectionUnit{}
	for _, u := range c.Incomplete() {
		for _, t := range u.Types {
			if _, seen := out[t]; !seen {
				out[t] = u
			}
		}
	}
	return out
}

// mergeSorted fusionne deux listes de types en une liste triée sans doublon.
func mergeSorted(a, b []string) []string {
	if len(a) == 0 && len(b) == 0 {
		return nil
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(a)+len(b))
	for _, list := range [][]string{a, b} {
		for _, v := range list {
			if v == "" || seen[v] {
				continue
			}
			seen[v] = true
			out = append(out, v)
		}
	}
	sort.Strings(out)
	return out
}

// Inventory est ce qu'une collecte rend : les ressources normalisées ET l'état
// de ce qui a pu être lu pour les produire. Les deux voyagent ENSEMBLE parce
// qu'ils ne se comprennent que l'un par l'autre — une liste de ressources sans
// son état de collecte ne permet pas de distinguer « il n'y en a pas » de « on
// n'a pas pu regarder ».
type Inventory struct {
	Resources  []Resource
	Collection Collection
}

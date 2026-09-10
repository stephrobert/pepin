package cmd

import "testing"

// Une valeur VIDE qui est arrivée est-elle une observation ?
//
// Le défaut mesuré (issue #227), sur le tenant de qualification Exoscale. Le tenant
// contient, délibérément, une instance SANS aucun groupe de sécurité — l'écart même que
// `compute_instance_has_security_group` existe pour attraper. Le scan a rendu
// `not-evaluated`, en disant :
//
//	attribute "security_group_ids" not collected on the resources of type
//	"compute_instance" (capability guard)
//
// L'inventaire scellé de ce run portait pourtant `security_group_ids: []` sur cette
// instance. `collected` comptait une liste vide pour non collectée ; l'intersection
// retirait alors l'attribut de TOUT le type, et le contrôle perdait sa capacité à
// conclure. La ressource fautive faisait taire le contrôle qui la visait — le motif
// que cette vague a corrigé quatre fois dans les règles, une cinquième fois ici.
//
// Pourquoi cette réponse n'était pas gratuite. Elle défendait un vrai risque :
// `IAMPolicyStatements` posait `[]` aussi bien pour « ce document n'a pas pu être
// analysé » que pour « cette politique n'accorde rien ». Compter ce `[]` comme une
// collecte aurait rouvert l'incident fondateur de l'ADR-0006 — une policy `Action:*`
// échappant à tous les contrôles `iam_policy_*`. Le bouchon a donc été retiré à la
// SOURCE : le parseur rend `nil` quand il n'a pas su lire, et les collecteurs OMETTENT
// alors l'attribut. Apprendre au verrou à tolérer un bouchon aurait été la mauvaise
// direction ; le supprimer est la bonne.
//
// La frontière que ce test grave : la PRÉSENCE décide, pas le remplissage. Une liste
// vide énumère complètement, et compte zéro.
func TestAnObservedEmptyValueIsCollected(t *testing.T) {
	cas := []struct {
		nom    string
		valeur any
		veut   bool
		motif  string
	}{
		{"une liste pleine", []any{"sg-1"}, true, ""},
		{
			"une liste VIDE", []any{}, true,
			"« aucun groupe de sécurité » est une information, et c'est celle que le contrôle cherche",
		},
		{
			"une table VIDE", map[string]any{}, true,
			"« aucune étiquette » est une observation, pas une lacune de collecte",
		},
		{"une table pleine", map[string]any{"env": "prod"}, true, ""},
		{"une chaîne vide", "", true, "la source a répondu, et sa réponse est vide"},
		{"un faux booléen", false, true, "`false` est une valeur, pas une absence"},
		{"un zéro", float64(0), true, "`0` est une valeur, pas une absence"},
		{
			"nil", nil, false,
			"un scalaire nul n'énumère rien : il ne permet pas de distinguer absent d'inconnu",
		},
	}
	for _, c := range cas {
		if got := collected(c.valeur); got != c.veut {
			t.Errorf("%s : collected(%#v) = %v, attendu %v.\n  %s",
				c.nom, c.valeur, got, c.veut, c.motif)
		}
	}
}

// Le bout de chaîne qui compte vraiment : l'INTERSECTION par type.
//
// `collected` seul ne dit rien de ce qu'un contrôle pourra conclure. C'est
// `attrsByTypeOf` qui décide, et c'est là que la ressource fautive faisait le dégât :
// une seule liste vide suffisait à retirer l'attribut de tout le type, donc à faire
// taire le contrôle sur les ressources VOISINES autant que sur elle.
func TestOneEmptyValueNoLongerBlindsItsWholeType(t *testing.T) {
	res := func(typ, id string, attrs map[string]any) map[string]any {
		return map[string]any{"type": typ, "id": id, "name": id, "attributes": attrs}
	}
	// Le tenant Exoscale, dans sa forme mesurée : deux instances avec un groupe, une
	// sans — celle qui est fautive.
	inv := map[string]any{"resources": []any{
		res("compute_instance", "avec-1", map[string]any{"security_group_ids": []any{"sg-1"}}),
		res("compute_instance", "avec-2", map[string]any{"security_group_ids": []any{"sg-2"}}),
		res("compute_instance", "sans", map[string]any{"security_group_ids": []any{}}),
	}}
	if !attrsByTypeOf(inv)["compute_instance"]["security_group_ids"] {
		t.Error("une instance SANS groupe retire encore l'attribut de tout son type — " +
			"le contrôle qui la vise ne peut plus conclure (#227)")
	}

	// LE CONTRE-EXEMPLE, et il vaut autant : un attribut réellement ABSENT d'une
	// ressource doit toujours fermer la porte. Sans lui, ce correctif ouvrirait la voie
	// à un `pass` que rien n'établit — le seul défaut que l'ADR-0006 interdit.
	partiel := map[string]any{"resources": []any{
		res("compute_instance", "porte", map[string]any{"deletion_protection": true}),
		res("compute_instance", "porte-pas", map[string]any{}),
	}}
	if attrsByTypeOf(partiel)["compute_instance"]["deletion_protection"] {
		t.Error("un attribut absent d'une ressource compte comme collecté pour le type — " +
			"c'est l'union que l'intersection existe pour empêcher")
	}

	// Et un `nil` explicite ne vaut pas mieux qu'une absence : il ne dit pas ce que la
	// valeur est.
	nul := map[string]any{"resources": []any{
		res("compute_instance", "a", map[string]any{"encrypted": true}),
		res("compute_instance", "b", map[string]any{"encrypted": nil}),
	}}
	if attrsByTypeOf(nul)["compute_instance"]["encrypted"] {
		t.Error("un `nil` compte comme collecté : un scalaire nul n'établit rien")
	}
}

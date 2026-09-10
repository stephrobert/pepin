package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stephrobert/pepin/internal/genprovider"
	"github.com/stephrobert/pepin/internal/i18n"
	"github.com/stephrobert/pepin/internal/model"
)

// Un refus nomme-t-il le droit qui, lui, aurait collecté ?
//
// Le défaut mesuré (issue #168). Sur Outscale, `object_storage_bucket` et
// `kubernetes_cluster` portaient un `grant` VIDE au descripteur. Le relevé de
// capacités n'imprime la ligne « droit requis » que si le descripteur la déclare,
// et le motif d'un « non évalué » de même : l'opérateur ne lisait donc que la
// réponse brute d'OOS —
//
//	The AWS access key Id you provided does not exist in our records
//
// — qui l'envoyait vérifier une clé parfaitement valide. Le fait mesuré est tout
// autre : OOS ne connaît AUCUNE clé EIM, et seules les clés du propriétaire du
// compte collectent cette unité. Un message qui envoie corriger la mauvaise chose
// coûte plus cher qu'un message absent, parce qu'on le suit.
//
// Le test tire les droits du VRAI descripteur, jamais d'une table écrite ici :
// c'est la chaîne descripteur → relevé qui est en cause, pas le rendu isolé.
func TestARefusedUnitNamesTheGrantThatWouldHaveCollected(t *testing.T) {
	ensureProvidersRegistered(t)
	// La langue est un état de PAQUET : la laisser déplacée ferait dépendre les
	// autres tests du binaire de l'ordre d'exécution.
	defer i18n.Set(i18n.Current())
	grants := genprovider.Grants("outscale")

	coll := model.Collection{Units: []model.CollectionUnit{
		{Unit: "compute_instance", Attempted: true, Complete: true},
		{
			Unit: "object_storage_bucket", Attempted: true, Complete: false,
			Error:  model.OutcomePermissionDenied,
			Detail: "HTTP 403 · ListBuckets · InvalidAccessKeyId: The AWS access key Id you provided does not exist in our records.",
		},
		{
			Unit: "kubernetes_cluster", Attempted: true, Complete: false,
			Error:  model.OutcomePermissionDenied,
			Detail: `HTTP 403 · GET /api/v2/clusters/all · {"message":"Forbidden: User type not allowed."}`,
		},
	}}

	// Les deux langues : un opérateur francophone doit lire la même action.
	for _, l := range []i18n.Lang{i18n.FR, i18n.EN} {
		i18n.Set(l)
		var b bytes.Buffer
		renderCapabilities(&b, coll, nil, grants, true)
		out := b.String()

		for _, cas := range []struct{ unit, droit string }{
			{"object_storage_bucket", "account owner access key (OOS)"},
			{"kubernetes_cluster", "account owner access key (OKS)"},
		} {
			if !strings.Contains(out, cas.droit) {
				t.Errorf("[%s] %s refusé : le relevé ne nomme pas le droit qui aurait collecté (%q attendu).\n"+
					"  L'opérateur ne lit alors que l'erreur brute de l'API, qui accuse sa clé.\n"+
					"--- relevé ---\n%s", l, cas.unit, cas.droit, out)
			}
		}
		// Le détail brut reste imprimé : la classe dit quoi faire, le détail dit à
		// qui le demander. Le remplacer priverait l'utilisateur de la réponse réelle.
		if !strings.Contains(out, "InvalidAccessKeyId") {
			t.Errorf("[%s] le détail brut de l'API a disparu du relevé — le droit nommé le complète, il ne le remplace pas.\n--- relevé ---\n%s", l, out)
		}
	}
}

// Le contre-exemple : une unité refusée dont le descripteur ne déclare AUCUN droit
// ne doit pas faire inventer de ligne. Un droit fabriqué ferait élargir des
// privilèges au hasard, ce qui est le contraire du service rendu (ADR-0014).
func TestAnUndeclaredGrantIsNeverInvented(t *testing.T) {
	ensureProvidersRegistered(t)
	defer i18n.Set(i18n.Current())
	i18n.Set(i18n.FR)
	coll := model.Collection{Units: []model.CollectionUnit{
		{Unit: "type_inconnu_du_descripteur", Attempted: true, Complete: false,
			Error: model.OutcomePermissionDenied, Detail: "HTTP 403"},
	}}
	var b bytes.Buffer
	renderCapabilities(&b, coll, nil, genprovider.Grants("outscale"), true)
	if strings.Contains(b.String(), "droit requis") {
		t.Errorf("un droit a été nommé pour une unité que le descripteur ne déclare pas :\n%s", b.String())
	}
}

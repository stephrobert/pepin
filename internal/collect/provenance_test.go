package collect

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stephrobert/pepin/internal/model"
)

// TestProvenanceNamesTheCallThatActuallyHappened est la garde centrale de la
// provenance : la source d'un attribut `api` est la requête RÉELLEMENT servie, et
// aucune source ne peut être fabriquée depuis la spec.
//
// La preuve est une MESURE, pas une lecture : un serveur de test enregistre ce
// qu'il a servi, et l'attestation est comparée à cet enregistrement. C'est la
// forme exacte de l'écart que ce dépôt a déjà connu (les policies EIM inline : la
// règle était juste, la donnée n'arrivait jamais, et aucun test ne pouvait le
// voir).
func TestProvenanceNamesTheCallThatActuallyHappened(t *testing.T) {
	var served []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		served = append(served, r.Method+" "+r.URL.Path)
		_, _ = w.Write([]byte(`{"volumes":[{"id":"vol-1","state":"attached"}]}`))
	}))
	defer srv.Close()

	spec := Spec{Provider: "essai", BaseURL: srv.URL}
	rs := ResourceSpec{
		Type:  "blockstorage_volume",
		Path:  "/block-storage",
		Items: "volumes",
		ID:    "volume_id",
		Map:   map[string]string{"volume_id": "id", "state": "state", "jamais_la": "absent_field"},
		Const: map[string]any{"encrypted": true},
	}
	got, err := collectResource(context.Background(), srv.Client(), spec, nil, rs, nil)
	if err != nil {
		t.Fatalf("collecte : %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("%d ressources, attendu 1", len(got))
	}
	prov := got[0].Provenance

	// 1. L'attribut lu porte l'appel qui l'a servi, et cet appel a bien été servi.
	att, ok := prov["state"]
	if !ok {
		t.Fatal("aucune attestation pour `state`")
	}
	if att.Origin != model.OriginAPI || !att.Observed {
		t.Errorf("`state` : origine %q observed=%v, attendu api/true", att.Origin, att.Observed)
	}
	if !strings.HasSuffix(att.Source, "/block-storage") || !strings.HasPrefix(att.Source, "GET ") {
		t.Errorf("`state` : source %q ne nomme pas la requête émise", att.Source)
	}
	if len(served) != 1 || served[0] != "GET /block-storage" {
		t.Fatalf("le serveur a servi %v — la mesure ne confirme pas l'attestation", served)
	}

	// 2. Le littéral du descripteur ne prétend PAS venir d'un appel.
	c := prov["encrypted"]
	if c.Origin != model.OriginDerived {
		t.Errorf("`encrypted` (const) : origine %q, attendu %q", c.Origin, model.OriginDerived)
	}
	if c.Observed {
		t.Error("`encrypted` (const) : observed=true — un littéral n'a jamais été observé")
	}
	if strings.Contains(c.Source, srv.URL) || strings.Contains(c.Source, "http") {
		t.Errorf("`encrypted` (const) : source %q désigne un appel qui n'a pas eu lieu", c.Source)
	}

	// 3. Un champ cherché et absent de la réponse est attesté NON observé — et il
	//    n'est pas projeté. « On a regardé, il n'y était pas » n'est pas « on n'a
	//    jamais regardé ».
	missing, ok := prov["jamais_la"]
	if !ok {
		t.Fatal("aucune attestation pour un champ absent de la réponse")
	}
	if missing.Observed {
		t.Error("un champ absent de la réponse est attesté observé")
	}
	if _, projected := got[0].Attributes["jamais_la"]; projected {
		t.Error("un champ absent de la réponse a été projeté")
	}
}

// TestProvenanceIsAbsentWithoutASource : hors collecte attestée (Project nu, appelé
// par des chemins qui n'ont pas de source à nommer), aucune attestation n'est
// fabriquée. Une provenance inventée serait pire que pas de provenance.
func TestProvenanceIsAbsentWithoutASource(t *testing.T) {
	item := map[string]any{"id": "x"}
	attrs, prov := ProjectAttested(item, map[string]string{"vm_id": "id"}, nil, Source{})
	if len(prov) != 0 {
		t.Errorf("provenance fabriquée sans source : %v", prov)
	}
	if attrs["vm_id"] != "x" {
		t.Errorf("la projection a changé : %v", attrs)
	}
}

// TestProjectStillProjectsExactlyTheSameAttributes : la refonte de Project en
// ProjectAttested ne devait déplacer AUCUNE valeur. Les cas limites de la
// projection (transform sur source absente, collection vide, coalescence) sont
// rejoués ici sur les deux chemins.
func TestProjectStillProjectsExactlyTheSameAttributes(t *testing.T) {
	item := map[string]any{
		"id":     "i-1",
		"labels": map[string]any{"env": "prod"},
		"empty":  []any{},
		"nul":    nil,
	}
	mapping := map[string]string{
		"vm_id":    "id",
		"tags":     "labels",
		"vide":     "empty",
		"absent":   "pas_la",
		"nul":      "nul",
		"coalesce": "pas_la||id",
	}
	transforms := map[string]any{"tags": "kv", "vide": "list", "absent": "list"}

	plain := Project(item, mapping, transforms)
	attested, _ := ProjectAttested(item, mapping, transforms, Source{Origin: model.OriginAPI, Ref: "GET https://exemple/x"})
	if len(plain) != len(attested) {
		t.Fatalf("projections différentes : %v vs %v", plain, attested)
	}
	for k := range plain {
		if _, ok := attested[k]; !ok {
			t.Errorf("attribut %q perdu par la projection attestée", k)
		}
	}
	if _, ok := plain["absent"]; ok {
		t.Error("un transform `list` sur une clé ABSENTE de la source ne doit rien fabriquer")
	}
	if _, ok := plain["vide"]; !ok {
		t.Error("une collection présente et VIDE reste une information collectée")
	}
}

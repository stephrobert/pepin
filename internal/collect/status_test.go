package collect

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	yaml "go.yaml.in/yaml/v3"

	"github.com/stephrobert/pepin/internal/model"
)

// Les chemins de DÉGRADATION, mesurés contre un vrai serveur plutôt que contre une
// erreur fabriquée à la main. Une classification qu'on éprouve sur une erreur qu'on
// a soi-même construite ne mesure que sa propre construction ; ici, la réponse
// traverse le client HTTP, le décodage et la pagination, comme en production.

// twoResourceSpec : deux endpoints indépendants. Le premier répond, le second non —
// c'est le cas qui décide, parce qu'il oppose « le scan s'arrête » à « le scan
// mesure ce qu'il peut et enregistre le reste ».
const twoResourceSpec = `
provider: test
resources:
  - type: compute_instance
    path: /vms
    items: Vms
    id: vm_id
    map:
      vm_id: VmId
      state: State
  - type: iam_policy
    path: /policies
    items: Policies
    id: policy_id
    map:
      policy_id: PolicyId
`

func specOf(t *testing.T, base, body string) Spec {
	t.Helper()
	var s Spec
	if err := yaml.Unmarshal([]byte(body), &s); err != nil {
		t.Fatalf("spec de test invalide : %v", err)
	}
	s.BaseURL = base
	return s
}

// unitByName rend l'unité de collecte portant ce nom.
func unitByName(t *testing.T, coll model.Collection, name string) model.CollectionUnit {
	t.Helper()
	for _, u := range coll.Units {
		if u.Unit == name {
			return u
		}
	}
	t.Fatalf("unité %q absente de l'état de collecte : %+v", name, coll.Units)
	return model.CollectionUnit{}
}

// TestA403OnOneEndpointNeitherStopsTheScanNorPassesInSilence est le cas de l'issue :
// ReadVms répond, ReadUserPolicies rend 403. Les deux issues fautives sont
// symétriques — tout arrêter rendrait l'outil inutilisable pour un compte de lecture
// à qui il manque un droit, et poursuivre en silence fabriquerait un « conforme »
// sur un périmètre jamais lu.
func TestA403OnOneEndpointNeitherStopsTheScanNorPassesInSilence(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/policies") {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"Errors":[{"Code":"AccessDenied"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"Vms":[{"VmId":"i-1","State":"running"}]}`))
	}))
	defer srv.Close()

	inv, err := Collect(context.Background(), srv.Client(), specOf(t, srv.URL, twoResourceSpec), nil, nil)
	if err != nil {
		t.Fatalf("un endpoint refusé ne doit pas faire échouer le scan entier : %v", err)
	}
	if len(inv.Resources) != 1 {
		t.Errorf("ce qui a pu être lu doit l'être : %d ressources, attendu 1", len(inv.Resources))
	}
	vms := unitByName(t, inv.Collection, "compute_instance")
	if !vms.Complete {
		t.Error("l'endpoint qui a répondu doit être marqué complet")
	}
	pol := unitByName(t, inv.Collection, "iam_policy")
	if pol.Complete {
		t.Fatal("l'endpoint refusé ne doit JAMAIS être marqué complet — c'est de là que naît le faux vert")
	}
	if pol.Error != model.OutcomePermissionDenied {
		t.Errorf("un 403 se classe permission_denied, obtenu %q", pol.Error)
	}
	if !strings.Contains(pol.Detail, "403") {
		t.Errorf("le détail doit porter la réponse de l'API, obtenu %q", pol.Detail)
	}
	if _, degraded := inv.Collection.IncompleteTypes()["iam_policy"]; !degraded {
		t.Error("le type iam_policy doit être signalé incomplet aux contrôles qui le lisent")
	}
}

// TestAnInterruptedPaginationKeepsWhatItReadAndSaysItIsTruncated : la troncature est
// le cas le plus trompeur, parce qu'elle rend une collection NON VIDE. Un inventaire
// à moitié lu ressemble à un inventaire lu.
func TestAnInterruptedPaginationKeepsWhatItReadAndSaysItIsTruncated(t *testing.T) {
	// Serveur pathologique : il ignore la pagination et rend toujours un lot plein,
	// donc la borne de pages finit par tomber.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"Vms":[{"VmId":"i-1"},{"VmId":"i-2"}]}`))
	}))
	defer srv.Close()

	spec := specOf(t, srv.URL, `
provider: test
resources:
  - type: compute_instance
    path: /vms
    items: Vms
    id: vm_id
    map:
      vm_id: VmId
    paging:
      style: page
      param: page
      size_param: per_page
      size: 2
      max_pages: 3
`)
	inv, err := Collect(context.Background(), srv.Client(), spec, nil, nil)
	if err != nil {
		t.Fatalf("une troncature s'enregistre, elle n'arrête pas le scan : %v", err)
	}
	u := unitByName(t, inv.Collection, "compute_instance")
	if u.Complete {
		t.Fatal("une pagination interrompue ne rend pas une collecte complète")
	}
	if u.Error != model.OutcomeTruncated {
		t.Errorf("classe attendue truncated, obtenu %q", u.Error)
	}
	if len(inv.Resources) == 0 {
		t.Error("les pages déjà lues doivent être conservées : un écart vu reste un écart vu")
	}
}

// TestAPartialResponseIsIncompleteButNotEmpty : la page 1 répond, la page 2 est
// refusée. C'est la « réponse partielle » de l'issue #43.
func TestAPartialResponseIsIncompleteButNotEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") == "2" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{"Vms":[{"VmId":"i-1"},{"VmId":"i-2"}]}`))
	}))
	defer srv.Close()

	spec := specOf(t, srv.URL, `
provider: test
resources:
  - type: compute_instance
    path: /vms
    items: Vms
    id: vm_id
    map:
      vm_id: VmId
    paging:
      style: page
      param: page
      size_param: per_page
      size: 2
`)
	inv, err := Collect(context.Background(), srv.Client(), spec, nil, nil)
	if err != nil {
		t.Fatalf("scan interrompu : %v", err)
	}
	if len(inv.Resources) != 2 {
		t.Errorf("la page lue doit être gardée : %d ressources, attendu 2", len(inv.Resources))
	}
	u := unitByName(t, inv.Collection, "compute_instance")
	if u.Complete {
		t.Fatal("une réponse partielle n'est pas une réponse")
	}
	if u.Error != model.OutcomeUnavailable {
		t.Errorf("un 500 se classe unavailable, obtenu %q", u.Error)
	}
}

// TestATimeoutIsClassedAsATimeout : distinguer « le service ne répond pas » de
// « le compte n'a pas le droit » n'est pas cosmétique — l'un se réessaie, l'autre
// se corrige sur la politique du compte de scan.
func TestATimeoutIsClassedAsATimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()

	inv, err := Collect(ctx, srv.Client(), specOf(t, srv.URL, twoResourceSpec), nil, nil)
	if err != nil {
		t.Fatalf("un délai dépassé s'enregistre, il n'arrête pas le scan : %v", err)
	}
	u := unitByName(t, inv.Collection, "compute_instance")
	if u.Error != model.OutcomeTimeout {
		t.Errorf("classe attendue timeout, obtenu %q (détail : %s)", u.Error, u.Detail)
	}
}

// TestAnUnreadableResponseIsNotAnEmptyOne : une réponse qui n'est pas du JSON n'est
// pas « zéro ressource ». Les confondre est la voie la plus courte vers un faux vert
// — un proxy d'entreprise qui rend une page HTML de connexion suffirait.
func TestAnUnreadableResponseIsNotAnEmptyOne(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<html><body>SSO login</body></html>"))
	}))
	defer srv.Close()

	inv, err := Collect(context.Background(), srv.Client(), specOf(t, srv.URL, twoResourceSpec), nil, nil)
	if err != nil {
		t.Fatalf("scan interrompu : %v", err)
	}
	u := unitByName(t, inv.Collection, "compute_instance")
	if u.Complete {
		t.Fatal("une réponse illisible ne rend pas une collecte complète")
	}
	if u.Error != model.OutcomeUnreadable {
		t.Errorf("classe attendue unreadable, obtenu %q", u.Error)
	}
}

// TestAnAggregateIsNotComputedOnATruncatedList : un agrégat est comparé à un seuil
// par la règle qui le lit. Un compte faux est donc pire qu'un compte absent : le
// premier produit un verdict, le second un « non évalué ».
func TestAnAggregateIsNotComputedOnATruncatedList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") == "2" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		_, _ = w.Write([]byte(`{"Rules":[{"Id":"a"},{"Id":"b"}]}`))
	}))
	defer srv.Close()

	spec := specOf(t, srv.URL, `
provider: test
resources:
  - type: api_access_summary
    path: /rules
    items: Rules
    id: id
    aggregate: rule_count
    const:
      id: summary
    paging:
      style: page
      param: page
      size_param: per_page
      size: 2
`)
	inv, err := Collect(context.Background(), srv.Client(), spec, nil, nil)
	if err != nil {
		t.Fatalf("scan interrompu : %v", err)
	}
	if len(inv.Resources) != 0 {
		t.Errorf("aucun agrégat ne doit sortir d'une liste tronquée, obtenu %+v", inv.Resources)
	}
	if unitByName(t, inv.Collection, "api_access_summary").Complete {
		t.Error("l'unité doit être incomplète")
	}
}

// TestACompleteCollectionRecordsEveryUnitAsComplete : la non-régression. Le chemin
// nominal n'a pas changé, et l'état de collecte le dit.
func TestACompleteCollectionRecordsEveryUnitAsComplete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/policies") {
			_, _ = w.Write([]byte(`{"Policies":[{"PolicyId":"p-1"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"Vms":[{"VmId":"i-1","State":"running"}]}`))
	}))
	defer srv.Close()

	inv, err := Collect(context.Background(), srv.Client(), specOf(t, srv.URL, twoResourceSpec), nil, nil)
	if err != nil {
		t.Fatalf("scan interrompu : %v", err)
	}
	if n := len(inv.Collection.Incomplete()); n != 0 {
		t.Errorf("aucune unité ne devait être incomplète, %d le sont", n)
	}
	if len(inv.Resources) != 2 {
		t.Errorf("%d ressources, attendu 2", len(inv.Resources))
	}
}

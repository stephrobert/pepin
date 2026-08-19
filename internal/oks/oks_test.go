package oks

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Mappe une réponse OKS réaliste (clés natives de l'API v2) et vérifie la projection
// vers les attributs agnostiques + la dérivation auto_upgrade (imbriquée), et que
// l'auth bi-en-têtes est bien posée.
func TestCollectClusters(t *testing.T) {
	var gotAK, gotSK, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAK = r.Header.Get("AccessKey")
		gotSK = r.Header.Get("SecretKey")
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"Clusters":[
			{"id":"c1","name":"demo","admin_whitelist":["0.0.0.0/0"],"cp_multi_az":false,
			 "disable_api_termination":false,"version":"1.31",
			 "auto_maintenances":{"minor_upgrade_maintenance":{"enabled":true}}},
			{"id":"c2","name":"prod","admin_whitelist":["10.0.0.0/8"],"cp_multi_az":true,
			 "disable_api_termination":true,"version":"1.31",
			 "auto_maintenances":{"minor_upgrade_maintenance":{"enabled":false}}}
		]}`))
	}))
	defer srv.Close()

	res, err := CollectClusters(context.Background(), srv.Client(), "outscale", srv.URL, "eu-west-2", "AK123", "SK456")
	if err != nil {
		t.Fatalf("CollectClusters : %v", err)
	}
	if gotAK != "AK123" || gotSK != "SK456" {
		t.Errorf("auth bi-en-têtes KO : AccessKey=%q SecretKey=%q", gotAK, gotSK)
	}
	if gotPath != "/api/v2/clusters/all" {
		t.Errorf("chemin KO : %q", gotPath)
	}
	if len(res) != 2 {
		t.Fatalf("attendu 2 clusters, obtenu %d", len(res))
	}
	c1 := res[0].Attributes
	if c1["control_plane_multi_az"] != false {
		t.Errorf("cp_multi_az non projeté : %v", c1["control_plane_multi_az"])
	}
	if c1["deletion_protection"] != false {
		t.Errorf("disable_api_termination non projeté : %v", c1["deletion_protection"])
	}
	if c1["auto_upgrade"] != true {
		t.Errorf("auto_upgrade non dérivé (attendu true) : %v", c1["auto_upgrade"])
	}
	wl, _ := c1["admin_whitelist"].([]any)
	if len(wl) != 1 || wl[0] != "0.0.0.0/0" {
		t.Errorf("admin_whitelist non projeté : %v", c1["admin_whitelist"])
	}
	if res[0].Type != "kubernetes_cluster" || res[0].ID != "c1" || res[0].Region != "eu-west-2" {
		t.Errorf("métadonnées ressource KO : %+v", res[0])
	}
	// Cluster 2 : auto_upgrade dérivé à false.
	if res[1].Attributes["auto_upgrade"] != false {
		t.Errorf("auto_upgrade cluster 2 attendu false : %v", res[1].Attributes["auto_upgrade"])
	}
}

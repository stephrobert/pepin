package eimpolicy

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stephrobert/pepin/internal/collect"
)

// Rejoue le contrat OAPI réel (vérifié sur l'API) : ReadUsers → ReadUserPolicies
// (PolicyNames) → ReadUserPolicy (PolicyDocument, une CHAÎNE JSON). Le document doit
// être normalisé en `statements`, l'attribut que lisent les règles iam_policy_*.
func TestCollectInlinePolicies(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		switch {
		case strings.HasSuffix(r.URL.Path, "/ReadUserGroups"):
			_, _ = w.Write([]byte(`{"UserGroups":[{"Name":"ops"}],"HasMoreItems":false}`))
		case strings.HasSuffix(r.URL.Path, "/ReadUserGroupPolicies"):
			// Le contrat renvoie directement Policies[] = InlinePolicy{Name, Body}.
			_, _ = w.Write([]byte(`{"Policies":[{"Name":"grp-admin","Body":"{\"Statement\": [ {\"Effect\": \"Allow\", \"Action\": [\"*\"], \"Resource\": [\"*\"]} ]}"}],"HasMoreItems":false}`))
		case strings.HasSuffix(r.URL.Path, "/ReadUsers"):
			_, _ = w.Write([]byte(`{"Users":[{"UserName":"alice"},{"UserName":"bob"}]}`))
		case strings.HasSuffix(r.URL.Path, "/ReadUserPolicies"):
			if strings.Contains(string(raw), "alice") {
				_, _ = w.Write([]byte(`{"PolicyNames":["inline-admin"]}`))
				return
			}
			_, _ = w.Write([]byte(`{"PolicyNames":[]}`)) // bob n'a aucune policy inline
		case strings.HasSuffix(r.URL.Path, "/ReadUserPolicy"):
			_, _ = w.Write([]byte(`{"UserName":"alice","PolicyName":"inline-admin","PolicyDocument":"{\"Statement\": [ {\"Effect\": \"Allow\", \"Action\": [\"*\"], \"Resource\": [\"*\"]} ]}"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	res, err := CollectInlinePolicies(context.Background(), srv.Client(), "outscale", srv.URL, nil)
	if err != nil {
		t.Fatalf("CollectInlinePolicies : %v", err)
	}
	// 1 policy inline de GROUPE (ops) + 1 inline utilisateur (alice).
	if len(res) != 2 {
		t.Fatalf("attendu 2 policies inline (groupe + utilisateur), obtenu %d", len(res))
	}
	var grp, usr map[string]any
	for _, r := range res {
		if r.Attributes["scope"] == "inline_group" {
			grp = r.Attributes
		} else {
			usr = r.Attributes
		}
	}
	if grp == nil || grp["owner_group"] != "ops" || grp["policy_name"] != "grp-admin" {
		t.Errorf("policy inline de groupe non collectée : %+v", grp)
	}
	if len(collect.IAMPolicyStatements("")) != 0 {
		t.Error("garde parseur")
	}
	a := usr
	if a["owner_user"] != "alice" || a["scope"] != "inline" || a["policy_name"] != "inline-admin" {
		t.Errorf("attributs KO : %+v", a)
	}
	// Le document (chaîne) doit être parsé en statements exploitables par les règles.
	st, _ := a["statements"].([]any)
	if len(st) != 1 {
		t.Fatalf("statements non normalisés : %#v", a["statements"])
	}
	// Le parseur normalise vers la forme que lisent les règles (actions/effect/resources).
	b, _ := json.Marshal(st[0])
	for _, want := range []string{`"actions":["*"]`, `"effect":"Allow"`, `"resources":["*"]`} {
		if !strings.Contains(string(b), want) {
			t.Errorf("statement normalisé attendu contenant %s, obtenu : %s", want, b)
		}
	}
}

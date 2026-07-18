package collect

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strings"
	"testing"
)

// Vérifie que ExoscaleAuth produit un en-tête EXO2-HMAC-SHA256 bien formé et une
// signature AUTO-COHÉRENTE : on rejoue le calcul à partir de l'expiration émise.
func TestExoscaleAuthSignature(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "https://api-ch-gva-2.exoscale.com/v2/instance?zone=ch-gva-2", nil)
	auth := ExoscaleAuth{Key: "EXOabc", Secret: "s3cr3t"}
	if err := auth.Apply(req); err != nil {
		t.Fatalf("Apply : %v", err)
	}
	h := req.Header.Get("Authorization")
	if !strings.HasPrefix(h, "EXO2-HMAC-SHA256 credential=EXOabc,") {
		t.Fatalf("préfixe/credential KO : %s", h)
	}
	parts := map[string]string{}
	for _, seg := range strings.Split(strings.TrimPrefix(h, "EXO2-HMAC-SHA256 "), ",") {
		if k, v, ok := strings.Cut(seg, "="); ok {
			parts[k] = v
		}
	}
	if parts["expires"] == "" || parts["signature"] == "" {
		t.Fatalf("expires/signature manquants : %v", parts)
	}
	// La requête a un seul query arg mono-valué (zone) → doit être annoncé signé.
	if parts["signed-query-args"] != "zone" {
		t.Errorf("signed-query-args KO : %q", parts["signed-query-args"])
	}
	// Rejoue la chaîne signée et compare la signature.
	sig := strings.Join([]string{
		"GET /v2/instance", // method + EscapedPath
		"",                 // corps
		"ch-gva-2",         // valeur du query arg signé (zone)
		"",                 // en-têtes signés : aucun
		parts["expires"],
	}, "\n")
	mac := hmac.New(sha256.New, []byte("s3cr3t"))
	_, _ = mac.Write([]byte(sig))
	want := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	if parts["signature"] != want {
		t.Errorf("signature incohérente :\n  obtenu %s\n  attendu %s", parts["signature"], want)
	}
}

// Vérifie que SigV4Auth pose une signature AWS V4 valide (en-têtes attendus +
// scope service/région) via le signer officiel aws-sdk-go-v2.
func TestSigV4AuthHeaders(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "https://api.eu-west-2.outscale.com/api/v1/ReadVms", nil)
	auth := SigV4Auth{Key: "AKID", Secret: "SECRET", Service: "oapi", Region: "eu-west-2"}
	if err := auth.Apply(req); err != nil {
		t.Fatalf("Apply : %v", err)
	}
	authz := req.Header.Get("Authorization")
	if !strings.HasPrefix(authz, "AWS4-HMAC-SHA256 ") {
		t.Fatalf("algo KO : %s", authz)
	}
	if !strings.Contains(authz, "Credential=AKID/") || !strings.Contains(authz, "/eu-west-2/oapi/aws4_request") {
		t.Errorf("scope credential KO : %s", authz)
	}
	if req.Header.Get("X-Amz-Date") == "" {
		t.Error("X-Amz-Date absent")
	}
	if !strings.Contains(authz, "Signature=") {
		t.Error("Signature absente")
	}
}

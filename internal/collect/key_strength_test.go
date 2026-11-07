package collect

import (
	"net/http"
	"strings"
	"testing"
)

func TestRequestMACRejectsSecretsBelow112Bits(t *testing.T) {
	for _, size := range []int{0, 13, 14, 32} {
		secret := strings.Repeat("x", size)
		for name, auth := range map[string]Auth{
			"sigv4": SigV4Auth{Key: "witness", Secret: secret, Service: "oapi", Region: "witness"},
			"exo2":  ExoscaleAuth{Key: "witness", Secret: secret},
		} {
			t.Run(name+"/"+strings.Repeat("x", size), func(t *testing.T) {
				req, err := http.NewRequest(http.MethodPost, "https://example.org/items", strings.NewReader("payload"))
				if err != nil {
					t.Fatal(err)
				}
				err = auth.Apply(req)
				if size < 14 {
					if err == nil || !strings.Contains(err.Error(), "112") || req.Header.Get("Authorization") != "" {
						t.Fatalf("secret trop court accepté ou refus non explicite : %v", err)
					}
				} else if err != nil || req.Header.Get("Authorization") == "" {
					t.Fatalf("secret conforme refusé : %v", err)
				}
			})
		}
	}
}

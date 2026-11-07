package genprovider

import (
	"strings"
	"testing"
)

func TestProviderRejectsWeakMACSecretsBeforeCollection(t *testing.T) {
	for _, kind := range []string{"sigv4", "exoscale-hmac"} {
		g := GenericProvider{}
		g.desc.Name = "witness"
		g.desc.Auth.Type = kind
		for _, size := range []int{13, 14} {
			auth, err := g.buildAuth(map[string]string{"access_key": "witness", "secret_key": strings.Repeat("x", size)})
			if size == 13 {
				if err == nil || auth != nil || !strings.Contains(err.Error(), "112") {
					t.Fatalf("%s : secret faible accepté : %v", kind, err)
				}
			} else if err != nil || auth == nil {
				t.Fatalf("%s : secret conforme refusé : %v", kind, err)
			}
		}
	}
}

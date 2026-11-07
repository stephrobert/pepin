package collectkit

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stephrobert/pepin/internal/collect"
)

func TestSigningSetupFailureStopsBeforeAnyCollection(t *testing.T) {
	for _, kind := range []string{"s3", "oks"} {
		t.Run(kind, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls.Add(1)
				if _, err := w.Write([]byte(`{"items":[]}`)); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			var s3 S3
			var oks OKS
			if kind == "s3" {
				s3 = S3{Endpoint: server.URL, Secret: strings.Repeat("x", 13)}
			} else {
				oks = OKS{Endpoint: server.URL, Secret: strings.Repeat("x", 13)}
			}
			_, err := Run(t.Context(), "witness", collect.Spec{
				BaseURL:   server.URL,
				Resources: []collect.ResourceSpec{{Type: "witness", Path: "/items", Items: "items"}},
			}, collect.HeaderAuth{Header: "X-Witness", Value: t.Name()}, nil, s3, oks, EIM{}, nil)
			if err == nil || !strings.Contains(err.Error(), "112") || calls.Load() != 0 {
				t.Fatalf("configuration faible masquée ou collecte déjà commencée : %v, %d appels", err, calls.Load())
			}
		})
	}
}

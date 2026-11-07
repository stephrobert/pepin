package objectstorage

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestS3RejectsShortSigningSecretBeforeSDKRequest(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	_, err := CollectBuckets(context.Background(), "witness", server.URL, "witness", t.Name(), strings.Repeat("x", 13), false, false)
	if err == nil || !strings.Contains(err.Error(), "112") || calls.Load() != 0 {
		t.Fatalf("secret S3 trop court accepté : %v (%d appels)", err, calls.Load())
	}
}

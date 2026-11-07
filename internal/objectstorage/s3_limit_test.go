package objectstorage

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"testing/iotest"

	"github.com/stephrobert/pepin/internal/collect"
	"github.com/stephrobert/pepin/internal/model"
)

func TestListBucketsBodyLimit(t *testing.T) {
	for _, tt := range []struct {
		name, body string
		wantError  bool
	}{
		{"at limit", "1234", false},
		{"over limit", "12345", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readListBucketsBody(strings.NewReader(tt.body), 4)
			if (err != nil) != tt.wantError {
				t.Fatalf("erreur = %v ; attendue = %t", err, tt.wantError)
			}
			if err == nil && string(got) != tt.body {
				t.Fatalf("corps = %q ; attendu %q", got, tt.body)
			}
		})
	}
	sentinel := errors.New("reader failure")
	_, err := readListBucketsBody(iotest.ErrReader(sentinel), 4)
	if !errors.Is(err, sentinel) {
		t.Fatalf("l'erreur de lecture a été perdue : %v", err)
	}
}

func TestListBucketsPageLimitReportsTruncation(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		page := calls.Add(1)
		_, _ = fmt.Fprintf(w, `<ListAllMyBucketsResult><Buckets/><ContinuationToken>page-%d</ContinuationToken></ListAllMyBucketsResult>`, page)
	}))
	defer srv.Close()

	buckets, err := CollectBuckets(context.Background(), "scaleway", srv.URL, "fr-par",
		"PEPINTESTACCESSKEY00", "pepin-test-secret-key", false, true)
	if len(buckets) != 0 || err == nil {
		t.Fatalf("buckets = %d, erreur = %v ; une pagination infinie doit être tronquée", len(buckets), err)
	}
	if got, _ := collect.Classify(err); got != model.OutcomeTruncated {
		t.Fatalf("classe = %q ; attendue = %q", got, model.OutcomeTruncated)
	}
	if got := calls.Load(); got != maxListBucketsPages {
		t.Fatalf("pages lues = %d ; borne attendue = %d", got, maxListBucketsPages)
	}
}

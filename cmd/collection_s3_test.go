package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stephrobert/pepin/internal/collect"
	"github.com/stephrobert/pepin/internal/collectkit"
	"github.com/stephrobert/pepin/internal/model"
	"github.com/stephrobert/scankit/assessment"
)

func inventoryFromUnreadableS3(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html>gateway error</html>"))
	}))
	defer srv.Close()
	inv, err := collectkit.Run(context.Background(), "scaleway", collect.Spec{}, nil, nil,
		collectkit.S3{Endpoint: srv.URL, Region: "fr-par", Key: "test-key", Secret: "test-secret", TagsPersisted: true},
		collectkit.OKS{}, collectkit.EIM{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	units := inv.Collection.Incomplete()
	if len(units) != 1 || units[0].Unit != "object_storage_bucket" || units[0].Error != model.OutcomeUnreadable {
		t.Fatalf("l'échec S3 n'est pas enregistré comme unreadable : %+v", units)
	}

	raw, err := os.ReadFile(filepath.Join(repoRoot, cleanInventory)) // #nosec G304 -- fixture du dépôt.
	if err != nil {
		t.Fatal(err)
	}
	var input map[string]any
	if err := json.Unmarshal(raw, &input); err != nil {
		t.Fatal(err)
	}
	input["collection"] = inv.Collection
	raw, err = json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "incomplete-s3.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestUnreadableS3DegradesVerdictsAndReplaysTheEffectivePolicy(t *testing.T) {
	bin := buildPepin(t)
	inventory := inventoryFromUnreadableS3(t)

	for _, tt := range []struct {
		name        string
		policy      string
		wantTagUnit bool
	}{
		{name: "default policy includes buckets", wantTagUnit: true},
		{name: "custom policy excludes buckets", policy: "controls:\n  tagging:\n    resource_types: [compute_instance]\n"},
		{name: "custom policy includes buckets", policy: "controls:\n  tagging:\n    resource_types: [object_storage_bucket]\n", wantTagUnit: true},
		{name: "explicitly empty policy", policy: "controls:\n  tagging:\n    resource_types: []\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			bundle := t.TempDir()
			args := []string{"scan", "scaleway", inventory, "--format", "assessment", "--seal", bundle}
			if tt.policy != "" {
				args = append(args, "--policy", writePolicy(t, tt.policy))
			}
			stdout, stderr, code := runScan(t, bin, args...)
			if code != exitStrict {
				t.Fatalf("collecte incomplète : code %d, attendu %d\n%s", code, exitStrict, stderr)
			}
			if !strings.Contains(stderr, "object_storage_bucket") || !strings.Contains(stderr, "unreadable") {
				t.Errorf("le relevé de capacités ne nomme pas l'échec illisible :\n%s", stderr)
			}
			var doc assessment.Assessment
			if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
				t.Fatal(err)
			}
			seenBucket, seenTags := false, false
			for _, r := range doc.Results {
				if strings.HasPrefix(r.Control, "objectstorage_bucket_") {
					seenBucket = true
					if r.Status == assessment.Pass {
						t.Errorf("%s conclut pass sur une collecte S3 illisible", r.Control)
					}
				}
				if r.Control == "governance_resource_required_tags" {
					seenTags = true
					got := strings.Contains(r.Evidence.Observed, `incomplete collection of "object_storage_bucket"`)
					if got != tt.wantTagUnit {
						t.Errorf("la raison du contrôle d'étiquetage est %q ; dépendance S3 attendue = %t",
							r.Evidence.Observed, tt.wantTagUnit)
					}
				}
			}
			if !seenBucket || !seenTags {
				t.Fatalf("contrôles témoins absents : bucket=%t tags=%t", seenBucket, seenTags)
			}
			_, verifyErr, verifyCode := runScan(t, bin, "verify", bundle, "--re-derive")
			if verifyCode != 0 {
				t.Fatalf("le bundle ne rejoue pas le même verdict : code %d\n%s", verifyCode, verifyErr)
			}
		})
	}
}

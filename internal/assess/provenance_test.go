package assess

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stephrobert/pepin/internal/model"
	"github.com/stephrobert/scankit/assessment"
)

// provenanceFixture : un inventaire attesté couvrant les trois origines et les deux
// formes (ressources typées, export JSON relu).
func provenanceFixture() map[string]any {
	return map[string]any{
		"provider": "essai",
		"resources": []model.Resource{
			{
				Type: "blockstorage_volume", ID: "vol-1",
				Attributes: map[string]any{"encrypted": true, "state": "attached"},
				Provenance: model.Provenance{
					"encrypted": {Origin: model.OriginDerived, Source: "descriptor:const", Derived: true},
					"state":     {Origin: model.OriginAPI, Source: "GET https://api.exemple/block-storage", Observed: true},
				},
			},
			{
				Type: "blockstorage_volume", ID: "vol-2",
				Attributes: map[string]any{"encrypted": true},
				Provenance: model.Provenance{
					"encrypted": {Origin: model.OriginDerived, Source: "descriptor:const", Derived: true},
					"state":     {Origin: model.OriginAPI, Source: "GET https://api.exemple/block-storage", Observed: false},
				},
			},
		},
	}
}

// TestProvenanceIndexSeparatesTheDerivedFromTheObserved : c'est l'objet même de
// l'issue. `encrypted` franchit la garde d'attribut de blockstorage_volume_encryption
// alors qu'aucun appel ne l'a porté — l'index le dit, sans rien changer au verdict.
func TestProvenanceIndexSeparatesTheDerivedFromTheObserved(t *testing.T) {
	idx := ProvenanceOf(provenanceFixture())
	enc := idx["blockstorage_volume"]["encrypted"]
	if len(enc.Sources) != 1 || enc.Sources[0] != "derived:descriptor:const" {
		t.Errorf("`encrypted` : sources %v, attendu [derived:descriptor:const]", enc.Sources)
	}
	if enc.Observed != 0 || enc.Total != 2 {
		t.Errorf("`encrypted` : observed=%d/%d, attendu 0/2 (un littéral n'est jamais observé)", enc.Observed, enc.Total)
	}
	st := idx["blockstorage_volume"]["state"]
	if st.Observed != 1 || st.Total != 2 {
		t.Errorf("`state` : observed=%d/%d, attendu 1/2 — un champ cherché et absent chez l'une des deux", st.Observed, st.Total)
	}
	if !strings.Contains(st.Label(), "observed=1/2") {
		t.Errorf("le libellé %q ne porte pas le compte d'observation", st.Label())
	}
}

// TestProvenanceSurvivesTheJSONRoundTrip : l'attestation doit traverser input.json,
// sinon `verify --re-derive` reconstruirait un assessment différent du scellé et
// crierait à la falsification sur un bundle parfaitement fidèle.
func TestProvenanceSurvivesTheJSONRoundTrip(t *testing.T) {
	raw, err := json.Marshal(provenanceFixture())
	if err != nil {
		t.Fatalf("sérialisation : %v", err)
	}
	var back any
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("relecture : %v", err)
	}
	direct := ProvenanceOf(provenanceFixture())
	roundTripped := ProvenanceOf(back)
	a, _ := json.Marshal(direct)
	b, _ := json.Marshal(roundTripped)
	if string(a) != string(b) {
		t.Errorf("l'index diffère après aller-retour JSON :\n  direct : %s\n  relu   : %s", a, b)
	}
}

// TestProvenanceNeverMovesAVerdict : la mesure de la promesse. La passe de
// provenance est appliquée sur un assessment de chaque statut, et AUCUN statut ne
// bouge — seules `evidence.attribute` et `evidence.source` changent.
func TestProvenanceNeverMovesAVerdict(t *testing.T) {
	before := assessment.Assessment{Results: []assessment.Result{
		{Control: "blockstorage_volume_encryption", Status: assessment.Pass, Subject: "essai",
			Evidence: assessment.Evidence{Observed: "aucune non-conformité", Source: "live-api"}},
		{Control: "blockstorage_volume_snapshots_exist", Status: assessment.NotEvaluated, Subject: "essai"},
		{Control: "objectstorage_bucket_public_access", Status: assessment.Fail, Subject: "bucket-1"},
		{Control: "governance_provider_sovereignty", Status: assessment.NotApplicable, Subject: "essai"},
	}}
	ct := map[string]string{
		"blockstorage_volume_encryption":      "blockstorage_volume",
		"blockstorage_volume_snapshots_exist": "blockstorage_volume",
		"objectstorage_bucket_public_access":  "object_storage_bucket",
		"governance_provider_sovereignty":     "",
	}
	after := WithProvenance(before, ProvenanceOf(provenanceFixture()), ct)

	if len(after.Results) != len(before.Results) {
		t.Fatalf("%d résultats après la passe, %d avant", len(after.Results), len(before.Results))
	}
	for i := range before.Results {
		b, a := before.Results[i], after.Results[i]
		if a.Status != b.Status {
			t.Errorf("%s : statut %q → %q — la provenance a déplacé un verdict", b.Control, b.Status, a.Status)
		}
		if a.Control != b.Control || a.Subject != b.Subject || a.Severity != b.Severity {
			t.Errorf("%s : identité du résultat modifiée", b.Control)
		}
		if a.Evidence.Observed != b.Evidence.Observed {
			t.Errorf("%s : la preuve OBSERVÉE a changé (%q → %q)", b.Control, b.Evidence.Observed, a.Evidence.Observed)
		}
	}
	// Et elle a bien fait son travail là où il y avait quelque chose à dire.
	if got := after.Results[0].Evidence.Source; !strings.Contains(got, "derived:descriptor:const") {
		t.Errorf("le `pass` de chiffrement n'expose pas la provenance de son attribut décisif : %q", got)
	}
	if got := after.Results[0].Evidence.Attribute; got != "encrypted" {
		t.Errorf("attribut décisif exposé : %q, attendu `encrypted`", got)
	}
	// Un contrôle dont l'inventaire ne porte AUCUNE attestation garde sa preuve telle
	// quelle : on n'invente pas une provenance pour combler un trou.
	if after.Results[2].Evidence.Source != before.Results[2].Evidence.Source {
		t.Error("une provenance a été fabriquée pour un type sans attestation")
	}
}

// TestProvenanceOfToleratesHostileInput : input.json vient d'un tiers audité. Une
// carte de provenance mal formée doit être ignorée, jamais provoquer une panique.
func TestProvenanceOfToleratesHostileInput(t *testing.T) {
	for _, bad := range []any{
		nil,
		"pas un objet",
		map[string]any{"resources": "pas une liste"},
		map[string]any{"resources": []any{"pas un objet"}},
		map[string]any{"resources": []any{map[string]any{"type": "x", "provenance": "pas une carte"}}},
		map[string]any{"resources": []any{map[string]any{"type": "x", "provenance": map[string]any{"a": 42}}}},
	} {
		if idx := ProvenanceOf(bad); idx == nil {
			t.Errorf("index nil sur %v", bad)
		}
	}
}

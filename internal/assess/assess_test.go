package assess

import (
	"testing"
	"testing/fstest"

	"github.com/stephrobert/pepin/referentiel"
	"github.com/stephrobert/scankit/assessment"
	"github.com/stephrobert/scankit/finding"
)

func testControls() map[string]referentiel.Control {
	return map[string]referentiel.Control{
		"objectstorage_bucket_public_access": {
			Code: "objectstorage_bucket_public_access", Titre: "Bucket public", Severite: "high",
			Scsl:         []string{"CLD-STO-1"},
			Frameworks:   map[string][]string{"secnumcloud_3_2": {"13.3"}},
			Fournisseurs: []string{"outscale", "exoscale"},
		},
		"network_sg_open_22": {
			Code: "network_sg_open_22", Titre: "SSH ouvert", Severite: "critical",
			Frameworks:   map[string][]string{"cis_controls_v8": {"4.4"}},
			Fournisseurs: []string{"outscale"},
		},
		"kubernetes_api_private": { // implemented for outscale but no k8s resource in the input
			Code: "kubernetes_api_private", Titre: "API K8s privée", Severite: "high",
			Fournisseurs: []string{"outscale"},
		},
		"compute_tags_present": { // implemented for outscale, resource present, but NOT verified
			Code: "compute_tags_present", Titre: "Étiquettes présentes", Severite: "medium",
			Fournisseurs: []string{"outscale"},
		},
		"iam_policy_no_wildcard": { // implemented for another provider only
			Code: "iam_policy_no_wildcard", Titre: "IAM sans joker", Severite: "high",
			Fournisseurs: []string{"scaleway"},
		},
	}
}

func TestReferences(t *testing.T) {
	c := testControls()["objectstorage_bucket_public_access"]
	refs := References(c)
	got := map[string]string{}
	for _, r := range refs {
		got[r.Framework] = r.ID
	}
	if got["scsl"] != "CLD-STO-1" {
		t.Errorf("missing scsl ref: %v", got)
	}
	if got["secnumcloud-3.2"] != "13.3" { // yaml key mapped to auditor slug
		t.Errorf("secnumcloud slug/id wrong: %v", got)
	}
}

func TestApplicable(t *testing.T) {
	present := map[string]bool{"object_storage_bucket": true, "security_group": true}
	if !applicable("objectstorage_bucket_public_access", present) {
		t.Error("objectstorage control must be applicable when a bucket is present")
	}
	if !applicable("network_sg_open_22", present) {
		t.Error("network control must be applicable when a security group is present")
	}
	if applicable("kubernetes_api_private", present) {
		t.Error("kubernetes control must NOT be applicable without a cluster resource")
	}
	if !applicable("governance_provider_sovereignty", present) {
		t.Error("governance controls always apply (synthetic resource)")
	}
}

func TestBuildStatuses(t *testing.T) {
	controls := testControls()
	findings := []finding.Finding{
		{Code: "network_sg_open_22", Severity: "critical", Subject: "sg-1", Message: "sg-1 : 0.0.0.0/0 on 22"},
	}
	rtypes := map[string]bool{"object_storage_bucket": true, "security_group": true, "compute_instance": true}
	run := assessment.Run{Target: assessment.Target{ID: "acct-1"}, Source: "export"}

	// Only the bucket control is contract-verified; compute_tags_present is applicable
	// (a compute_instance is present) but NOT verified — its data may not be collected.
	verified := map[string]bool{"objectstorage_bucket_public_access": true}
	a := Build("outscale", controls, findings, rtypes, nil, verified, run)
	byControl := map[string]assessment.Result{}
	for _, r := range a.Results {
		byControl[r.Control] = r
	}

	if byControl["network_sg_open_22"].Status != assessment.Fail {
		t.Errorf("failed control status = %s, want fail", byControl["network_sg_open_22"].Status)
	}
	if byControl["objectstorage_bucket_public_access"].Status != assessment.Pass {
		t.Errorf("verified, applicable, non-failing control = %s, want pass", byControl["objectstorage_bucket_public_access"].Status)
	}
	if byControl["kubernetes_api_private"].Status != assessment.NotEvaluated {
		t.Errorf("non-applicable control = %s, want not-evaluated", byControl["kubernetes_api_private"].Status)
	}
	// A6: applicable resource present, but the contract does NOT confirm the data is
	// collected → NotEvaluated, never a silent Pass.
	if byControl["compute_tags_present"].Status != assessment.NotEvaluated {
		t.Errorf("applicable but unverified control = %s, want not-evaluated", byControl["compute_tags_present"].Status)
	}
	// A control implemented only for another provider is out of THIS provider's assessment.
	if _, present := byControl["iam_policy_no_wildcard"]; present {
		t.Error("controls not implemented for the scanned provider must be excluded")
	}
	// The fail carries its exact references.
	if len(byControl["network_sg_open_22"].References) == 0 {
		t.Error("failing result must carry normative references")
	}
}

func TestBuildNotApplicableWithJustification(t *testing.T) {
	controls := testControls()
	rtypes := map[string]bool{"object_storage_bucket": true}
	run := assessment.Run{Target: assessment.Target{ID: "acct-1"}}
	reason := "SKS API endpoint is always public by construction (doc containers/overview)"
	na := map[string]string{"kubernetes_api_private": reason}

	a := Build("outscale", controls, nil, rtypes, na, nil, run)
	var got *assessment.Result
	for i := range a.Results {
		if a.Results[i].Control == "kubernetes_api_private" {
			got = &a.Results[i]
		}
	}
	if got == nil {
		t.Fatal("N/A control must appear in the assessment")
	}
	if got.Status != assessment.NotApplicable {
		t.Errorf("status = %s, want not-applicable", got.Status)
	}
	if got.Waiver == nil || got.Waiver.Justification != reason {
		t.Errorf("N/A must carry its justification, got %+v", got.Waiver)
	}
}

func TestRulesetDigestStable(t *testing.T) {
	fsys := fstest.MapFS{
		"a.rego": &fstest.MapFile{Data: []byte("package a\n")},
		"b.rego": &fstest.MapFile{Data: []byte("package b\n")},
		"c.txt":  &fstest.MapFile{Data: []byte("ignored")},
	}
	d1 := RulesetDigest(fsys)
	d2 := RulesetDigest(fsys)
	if d1 != d2 || d1 == "sha256:" {
		t.Errorf("digest must be stable and non-empty: %q %q", d1, d2)
	}
	// A rule change changes the digest.
	fsys["a.rego"] = &fstest.MapFile{Data: []byte("package a\ndeny := true\n")}
	if RulesetDigest(fsys) == d1 {
		t.Error("digest must change when a rule changes")
	}
}

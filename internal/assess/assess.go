// Package assess turns a provider scan into an opposable assessment.Assessment: a typed
// status per control (fail/pass/not-evaluated), the exact normative references from the
// common referentiel, and a run-provenance envelope. This is what makes a Pépin report
// defensible before an auditor — "no finding" is never confused with "compliant", and every
// result carries its SecNumCloud/ANSSI/CIS/ISO reference.
//
// Increment 1 covers fail (from findings), pass and not-evaluated (from applicability), the
// references and the provenance. Not-applicable justified by the provider contracts, and
// structured per-resource observed/expected evidence, come in later increments.
package assess

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"sort"
	"strings"

	"github.com/stephrobert/pepin/referentiel"
	"github.com/stephrobert/scankit/assessment"
	"github.com/stephrobert/scankit/finding"
)

// frameworkSlug maps the referentiel's YAML framework keys to stable, auditor-facing slugs.
var frameworkSlug = map[string]string{
	"secnumcloud_3_2": "secnumcloud-3.2",
	"anssi_bp_028":    "anssi-bp028",
	"cis_controls_v8": "cis-v8",
	"iso_27001_2022":  "iso-27001",
	"iso_27017":       "iso-27017",
}

// serviceHints maps a control's service prefix (the `<service>_` head of its code) to the
// resource-type substrings that indicate the service is present in the inventory. Used to tell
// "evaluated and compliant" (pass) from "nothing of this type to check" (not-evaluated).
var serviceHints = map[string][]string{
	"objectstorage": {"object_storage", "bucket"},
	"network":       {"security_group", "network", "firewall", "subnet", "vpc", "nic"},
	"compute":       {"compute", "instance", "server", "vm"},
	"iam":           {"iam", "access_key", "user", "policy", "role", "credential"},
	"kubernetes":    {"kubernetes", "sks", "k8s", "cluster", "kapsule"},
	"loadbalancer":  {"load", "_lb", "nlb", "alb"},
	"blockstorage":  {"block", "volume", "disk", "snapshot"},
}

// References converts a control's framework + SCSL mappings into exact, versioned references.
func References(c referentiel.Control) []assessment.Reference {
	var refs []assessment.Reference
	for _, id := range c.Scsl {
		refs = append(refs, assessment.Reference{Framework: "scsl", ID: id})
	}
	// Deterministic framework order.
	fws := make([]string, 0, len(c.Frameworks))
	for fw := range c.Frameworks {
		fws = append(fws, fw)
	}
	sort.Strings(fws)
	for _, fw := range fws {
		slug := frameworkSlug[fw]
		if slug == "" {
			slug = fw
		}
		for _, id := range c.Frameworks[fw] {
			refs = append(refs, assessment.Reference{Framework: slug, ID: id})
		}
	}
	return refs
}

// applicable reports whether a control's service is present in the collected resource types.
// Governance controls are evaluated via a synthetic sovereignty resource and always apply.
func applicable(code string, resourceTypes map[string]bool) bool {
	service := code
	if i := strings.IndexByte(code, '_'); i >= 0 {
		service = code[:i]
	}
	if service == "governance" {
		return true
	}
	hints := serviceHints[service]
	if len(hints) == 0 {
		return false
	}
	for rt := range resourceTypes {
		for _, h := range hints {
			if strings.Contains(rt, h) {
				return true
			}
		}
	}
	return false
}

// Build derives the assessment for one provider scan. `findings` must carry the AGNOSTIC
// control code (finding.Code before any SCSL enrichment). `resourceTypes` is the set of
// resource types present in the evaluated inventory. `naReasons` maps a control code to the
// justification of its not-applicability for this provider (from the provider contract) —
// an unjustified N/A is not opposable, so a control is only marked NotApplicable when a
// reason is present. `run` supplies the provenance.
func Build(provider string, controls map[string]referentiel.Control, findings []finding.Finding, resourceTypes map[string]bool, naReasons map[string]string, run assessment.Run) assessment.Assessment {
	// One Fail result per finding (a control may fail on several subjects).
	failedControls := map[string]bool{}
	var results []assessment.Result
	for _, f := range findings {
		failedControls[f.Code] = true
		c := controls[f.Code]
		results = append(results, assessment.Result{
			Control:     f.Code,
			Title:       first(c.Titre, f.Title),
			Status:      assessment.Fail,
			Severity:    first(f.Severity, c.Severite),
			Subject:     f.Subject,
			Evidence:    assessment.Evidence{Observed: stripSubject(f.Message), Source: run.Source},
			References:  References(c),
			Remediation: first(f.Remediation, c.Remediation),
			Labels:      f.Labels,
		})
	}

	// One pass/not-evaluated result per control implemented for this provider that did not fail.
	codes := make([]string, 0, len(controls))
	for code := range controls {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	for _, code := range codes {
		if failedControls[code] {
			continue // already emitted as Fail (possibly several subjects)
		}
		c := controls[code]
		res := assessment.Result{
			Control:    code,
			Title:      c.Titre,
			Severity:   c.Severite,
			Subject:    run.Target.ID,
			References: References(c),
		}
		switch {
		case naReasons[code] != "":
			// Contract-declared not-applicable, with its justification — the auditor gold.
			res.Status = assessment.NotApplicable
			res.Waiver = &assessment.Waiver{Justification: naReasons[code]}
		case contains(c.Fournisseurs, provider):
			// Implemented for this provider: pass if its service is present, else nothing to check.
			if applicable(code, resourceTypes) {
				res.Status = assessment.Pass
			} else {
				res.Status = assessment.NotEvaluated
			}
		default:
			continue // implemented for other providers and not N/A here — out of this scan's scope
		}
		results = append(results, res)
	}

	return assessment.Assessment{Run: run, Results: results}
}

// RulesetDigest hashes the embedded rule set so the assessment's provenance pins exactly which
// rules produced it (content hash, not just a version string).
func RulesetDigest(fsys fs.FS) string {
	h := sha256.New()
	_ = fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".rego") {
			return err
		}
		b, rerr := fs.ReadFile(fsys, p)
		if rerr != nil {
			return rerr
		}
		h.Write([]byte(p))
		h.Write(b)
		return nil
	})
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func first(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// stripSubject removes a leading "<subject> : " prefix from a finding message.
func stripSubject(msg string) string {
	if i := strings.Index(msg, " : "); i >= 0 {
		return strings.TrimSpace(msg[i+3:])
	}
	return msg
}

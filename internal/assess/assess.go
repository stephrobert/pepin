// Package assess turns a provider scan into an opposable assessment.Assessment: a typed
// status per control (fail/pass/not-evaluated), the exact normative references from the
// common referentiel, and a run-provenance envelope. This is what makes a Pépin report
// defensible before an auditor — "no finding" is never confused with "compliant", and every
// result carries its SecNumCloud/ANSSI/CIS/ISO reference.
//
// Statuses: fail (from findings), pass (ONLY when the provider contract confirms the data is
// collected AND a resource is present), not-applicable (justified by the provider contract),
// and not-evaluated (implemented but the data source is unconfirmed or absent). Each result
// carries its exact normative references and the run provenance. Structured per-resource
// observed/expected evidence comes in a later increment.
package assess

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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
	"cis_controls_v8": "cis-v8",
	"iso_27001_2022":  "iso-27001",
	"iso_27017":       "iso-27017",
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

// applicable reports whether the EXACT resource type a control evaluates is present in the
// collected inventory. `controlType[code]` is the normalized type the control reads (from
// genprovider.ControlType). Matching the exact type — not a service-family substring — is
// what prevents a false Pass: a compute_image control must not be deemed evaluated just
// because compute_instance resources are present. Governance controls read a synthetic
// resource and are gated by `verified` instead, so they always apply here.
func applicable(code string, controlType map[string]string, resourceTypes map[string]bool) bool {
	if strings.HasPrefix(code, "governance_") {
		return true
	}
	t := controlType[code]
	if t == "" {
		return false
	}
	return resourceTypes[t]
}

// Build derives the assessment for one provider scan. `findings` must carry the AGNOSTIC
// control code (finding.Code before any SCSL enrichment). `resourceTypes` is the set of
// resource types present in the evaluated inventory. `naReasons` maps a control code to the
// justification of its not-applicability for this provider (from the provider contract) —
// an unjustified N/A is not opposable, so a control is only marked NotApplicable when a
// reason is present. `verified[code]` is true only when the provider contract CONFIRMS the
// data this control needs is actually collected: a Pass is asserted ONLY for a verified
// control whose resource is present, so a capability guard that silently skipped evaluation
// (attribute not collected) surfaces as NotEvaluated, never a false "compliant". `run`
// supplies the provenance.
func Build(provider string, controls map[string]referentiel.Control, findings []finding.Finding, resourceTypes map[string]bool, naReasons map[string]string, verified map[string]bool, controlType map[string]string, run assessment.Run) assessment.Assessment {
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
		typ := controlType[code]
		switch {
		case naReasons[code] != "":
			// Contract-declared not-applicable, with its justification — the auditor gold.
			res.Status = assessment.NotApplicable
			res.Waiver = &assessment.Waiver{Justification: naReasons[code]}
		case contains(c.Fournisseurs, provider):
			// Implemented for this provider. Pass is asserted ONLY when the contract confirms
			// the needed data is collected (verified) AND a resource of that service is present.
			// Otherwise the control could not actually be evaluated (attribute not collected, or
			// nothing of this type in scope) — NotEvaluated, never a silent Pass.
			switch {
			case verified[code] && applicable(code, controlType, resourceTypes):
				// A Pass carries WHAT was checked (basis of the assertion), not just a status.
				res.Status = assessment.Pass
				res.Evidence = assessment.Evidence{
					Observed: fmt.Sprintf("aucune non-conformité détectée sur les ressources de type « %s » collectées (contrat vérifié)", typ),
					Source:   run.Source,
				}
			default:
				res.Status = assessment.NotEvaluated
				res.Evidence = assessment.Evidence{Observed: notEvaluatedReason(code, typ, verified[code], resourceTypes), Source: run.Source}
			}
		default:
			continue // implemented for other providers and not N/A here — out of this scan's scope
		}
		results = append(results, res)
	}

	return assessment.Assessment{Run: run, Results: results}
}

// notEvaluatedReason explique POURQUOI un contrôle n'a pas pu être évalué — un « non évalué »
// opposable dit sur quoi il bute (donnée non collectée, ou aucune ressource du type en scope).
func notEvaluatedReason(code, typ string, verified bool, resourceTypes map[string]bool) string {
	if !verified {
		return "collecte de la donnée nécessaire non confirmée pour ce fournisseur (contrat non « vérifié »)"
	}
	if typ != "" && !resourceTypes[typ] {
		return fmt.Sprintf("aucune ressource de type « %s » dans l'inventaire évalué", typ)
	}
	return "contrôle non évaluable sur cet inventaire"
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

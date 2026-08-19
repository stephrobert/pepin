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
	"maps"
	"sort"
	"strings"

	"github.com/stephrobert/pepin/internal/genprovider"
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

// requiredAttr maps a control to the resource attribute whose PRESENCE its evaluation needs.
// For these controls, absence of the attribute makes the rule silently emit no finding (a
// capability guard, or a compliant-by-default read), so a `verified` type would yield a FALSE
// pass. Listed here, such a control becomes NotEvaluated when the attribute was not collected on
// any resource of its type. Controls NOT listed are evaluated by the presence of a finding
// (absence of a bad config = genuinely compliant) and need no attribute gate. This map must stay
// in sync with the rules' capability guards — TestRequiredAttrGuardsExist enforces that.
// Valeur = attributs acceptés (any-of) : le gate passe si AU MOINS UN est présent. Permet
// à un contrôle d'être évaluable par plusieurs dérivations selon le fournisseur (ex.
// iam_no_root : flag explicite `root_owned` chez l'un, tag `scope` account/eim chez l'autre).
var requiredAttr = map[string][]string{
	"compute_instance_deletion_protection":               {"deletion_protection"},
	"compute_instance_no_secrets_in_user_data":           {"user_data"},
	"compute_instance_has_security_group":                {"security_group_ids"},
	"compute_instance_public_ip_with_open_securitygroup": {"public_ip"},
	"compute_image_not_public":                           {"public"},
	"iam_no_root_access_key":                             {"root_owned", "scope"},
	"iam_user_mfa_enabled":                               {"mfa_enabled"},
	"iam_account_mfa_enforced":                           {"require_trusted_env"},
	"iam_apiaccesspolicy_max_key_expiration":             {"max_access_key_expiration_seconds"},
	"blockstorage_volume_encryption":                     {"encrypted"},
	"blockstorage_snapshot_not_public":                   {"global_permission"},
	"objectstorage_bucket_object_lock_enabled":           {"object_lock_enabled"},
	"objectstorage_bucket_kms_encryption":                {"sse_kms_enabled"},
	"objectstorage_bucket_versioning_enabled":            {"versioning"},
	"objectstorage_bucket_default_encryption":            {"default_encryption_enabled"},
	// Un 403 sur GetBucketAcl ne doit pas rendre un bucket public « conforme ». La liste
	// n'accepte QUE des signaux d'ACL, volontairement : `policy_public` en faisait partie,
	// or collectBucket interroge ACL et policy SÉPARÉMENT (best effort). Un 403 sur
	// GetBucketAcl suivi d'un GetBucketPolicy réussi posait donc `policy_public: false`
	// seul, ce qui franchissait le verrou et concluait « conforme » sur une ACL jamais lue —
	// alors que l'ACL est le vecteur d'exposition le plus courant. Sans signal d'ACL, le
	// contrôle sort « non évalué », ce qui est la réponse honnête.
	"objectstorage_bucket_public_access":                {"acl", "acl_grants", "public_via_acl"},
	"database_encryption_at_rest_enabled":               {"encryption_at_rest"},
	"loadbalancer_http_redirect_to_https":               {"redirect_to_https"},
	"loadbalancer_ssl_listeners":                        {"load_balancer_type"},
	"network_subnet_no_public_ip_by_default":            {"map_public_ip_on_launch"},
	"network_securitygroup_default_restrict_traffic":    {"security_group_name"},
	"network_peering_cross_organization":                {"source_account", "accepter_account"},
	"kubernetes_cluster_not_publicly_accessible":        {"admin_whitelist"},
	"kubernetes_cluster_control_plane_highly_available": {"control_plane_multi_az"},
	"kubernetes_cluster_auto_upgrade_enabled":           {"auto_upgrade"},
	"kubernetes_cluster_deletion_protection":            {"deletion_protection"},
	"kubernetes_cluster_audit_logging_enabled":          {"audit_enabled"},
	// Ce contrôle lit la RÉGION de chaque ressource (pas la ressource synthétique du
	// fournisseur) : sans région collectée, il ne mesure rien.
	"governance_resource_region_in_eu": {"region"},

	// --- Verrous ajoutés après l'audit de livraison -------------------------
	// Ces contrôles concluaient « conforme » alors que l'attribut qui porte leur
	// décision n'avait jamais été collecté : la règle ne se déclenchait pas, donc
	// aucun finding, donc « pass ». Un faux vert est invisible par construction,
	// ce qui en fait le pire défaut possible pour un outil de posture.

	// La base est ouverte tant qu'aucune ACL n'est posée (défaut d'API Scaleway) :
	// sans ip_filter collecté, « conforme » est exactement l'inverse de la réalité.
	"database_service_not_open_to_internet": {"ip_filter"},
	"database_backup_enabled":               {"disable_backup"},

	// `statements` est TOUJOURS posé par le collecteur, au besoin à [] quand le
	// document n'a pas pu être analysé : le verrou ne vaut que parce que
	// attrsByTypeOf ne compte plus une collection vide comme collectée.
	"iam_policy_no_administrative_privileges": {"statements"},
	"iam_policy_no_notaction_notresource":     {"statements"},
	"iam_policy_no_wildcard_resource":         {"statements"},
	"iam_policy_no_privilege_escalation":      {"statements", "manages_iam"},
	"iam_role_key_lifetime_bounded":           {"max_session_ttl", "policy_has_expiration"},
	"iam_role_no_admin_privileges":            {"admin_privileges"},
	"iam_role_source_ip_restricted":           {"source_ip_restricted"},
	"iam_apiaccessrule_no_public_cidr":        {"ip_ranges"},

	"network_securitygroup_default_deny": {"inbound_default_policy"},
	"network_flow_matrix_documented":     {"description"},

	// Sur un plan Terraform, `state` arrive en after_unknown et aucun
	// blockstorage_snapshot n'existe : le contrôle ne peut PAS y être évalué,
	// et s'affichait pourtant vert sur l'exemple livré dans le dépôt.
	"blockstorage_volume_snapshots_exist": {"state"},
}

// RequiredAttrs retourne, par contrôle, les attributs dont la PRÉSENCE conditionne un « pass »
// (copie : la table interne reste immuable). Exposé pour que la documentation de couverture
// (internal/docgen) soit DÉRIVÉE de la table réellement appliquée, jamais recopiée à côté.
func RequiredAttrs() map[string][]string {
	out := make(map[string][]string, len(requiredAttr))
	for code, attrs := range requiredAttr {
		out[code] = append([]string(nil), attrs...)
	}
	return out
}

// governanceProviderReaders : contrôles de gouvernance dont la donnée EST la ressource
// synthétique `governance_provider` (toujours construite depuis le descripteur du
// fournisseur). Leur source est donc intrinsèquement collectée ; les autres contrôles de
// gouvernance (ex. étiquettes, qui dépendent de la collecte d'attributs par ressource) ne
// sont PAS présumés vérifiés.
var governanceProviderReaders = map[string]bool{
	"governance_resource_region_in_eu": true,
	"governance_provider_sovereignty":  true,
}

// Verified indique si le contrat du fournisseur CONFIRME que la donnée dont un contrôle a
// besoin est réellement collectée (état `verifie` du type visé). C'est le verrou d'un « pass »
// opposable : sans confirmation, Build reste sur NotEvaluated plutôt que d'affirmer une
// conformité qu'une garde de capacité a pu court-circuiter silencieusement. Un contrôle sans
// type de ressource visé qui ne lit pas le descripteur n'est jamais vérifié — donc jamais
// « pass ». Fonction UNIQUE : le scan et la documentation de couverture s'y adossent tous deux,
// pour qu'une page de couverture ne puisse pas décrire un verrou différent de celui appliqué.
func Verified(provider, code string) bool {
	switch {
	case governanceProviderReaders[code]:
		return true
	case genprovider.ControlType(code) != "":
		return genprovider.TypeEtat(provider, genprovider.ControlType(code)) == "verifie"
	default:
		return false
	}
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
func Build(provider string, controls map[string]referentiel.Control, findings []finding.Finding, resourceTypes map[string]bool, naReasons map[string]string, verified map[string]bool, controlType map[string]string, attrsByType map[string]map[string]bool, run assessment.Run) assessment.Assessment {
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
			Labels:      maps.Clone(f.Labels), // copie : une mutation post-Build (enrichFromReferentiel) ne doit pas altérer l'assessment scellé
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
			case verified[code] && applicable(code, controlType, resourceTypes) && attrCollected(code, typ, attrsByType):
				// A Pass carries WHAT was checked (basis of the assertion), not just a status.
				res.Status = assessment.Pass
				observed := "aucune non-conformité détectée (contrat vérifié)"
				if typ != "" {
					observed = fmt.Sprintf("aucune non-conformité détectée sur les ressources de type « %s » collectées (contrat vérifié)", typ)
				} else if strings.HasPrefix(code, "governance_provider_") {
					// Réservé aux contrôles dont la donnée EST le descripteur. Les
					// governance_resource_* sont mesurés sur le tenant : leur attribuer
					// cette preuve serait un mensonge.
					observed = "conforme selon les faits de souveraineté déclarés au descripteur du fournisseur (attestation, non mesuré sur le tenant)"
				}
				res.Evidence = assessment.Evidence{Observed: observed, Source: run.Source}
			default:
				res.Status = assessment.NotEvaluated
				res.Evidence = assessment.Evidence{Observed: notEvaluatedReason(code, typ, verified[code], resourceTypes, attrsByType), Source: run.Source}
			}
		default:
			continue // implemented for other providers and not N/A here — out of this scan's scope
		}
		results = append(results, res)
	}

	return assessment.Assessment{Run: run, Results: results}
}

// attrCollected indique que l'ATTRIBUT clé du contrôle (s'il en a un) est présent sur au moins
// une ressource de son type. Sans attribut requis déclaré, true (le contrôle s'évalue par la
// présence d'un finding). C'est le verrou par ATTRIBUT (pas seulement par type) du « pass ».
func attrCollected(code, typ string, attrsByType map[string]map[string]bool) bool {
	attrs := requiredAttr[code]
	if len(attrs) == 0 {
		return true
	}
	for _, a := range attrs {
		if attrsByType[typ][a] {
			return true
		}
	}
	return false
}

// notEvaluatedReason explique POURQUOI un contrôle n'a pas pu être évalué — un « non évalué »
// opposable dit sur quoi il bute (donnée non collectée, ou aucune ressource du type en scope).
func notEvaluatedReason(code, typ string, verified bool, resourceTypes map[string]bool, attrsByType map[string]map[string]bool) string {
	if !verified {
		return "collecte de la donnée nécessaire non confirmée pour ce fournisseur (contrat non « vérifié »)"
	}
	if typ != "" && !resourceTypes[typ] {
		return fmt.Sprintf("aucune ressource de type « %s » dans l'inventaire évalué", typ)
	}
	if attrs := requiredAttr[code]; len(attrs) > 0 && !attrCollected(code, typ, attrsByType) {
		where := fmt.Sprintf("les ressources de type « %s »", typ)
		if typ == "" { // contrôle transverse (gouvernance) : aucun type visé
			where = "les ressources collectées"
		}
		return fmt.Sprintf("attribut « %s » non collecté sur %s (garde de capacité)", strings.Join(attrs, " / "), where)
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

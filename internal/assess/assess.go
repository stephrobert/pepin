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
	"github.com/stephrobert/pepin/internal/i18n"
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
//
// La déclaration est faite PAR TYPE, d'où l'imbrication. La clé `""` désigne le type
// PRINCIPAL, celui que `genprovider.ControlType` dérive déjà du code : l'écrire serait
// redondant, et une redondance est ce qui finit par diverger. Seul un type SECONDAIRE
// se nomme.
//
// Pourquoi ce cran supplémentaire. La table portait un attribut par contrôle, appliqué
// au type principal, et six contrôles en corrèlent plusieurs. `volume_id` non collecté
// sur les SNAPSHOTS rendait alors chaque snapshot inattribuable, donc tous les volumes
// « sans sauvegarde » — un faux positif de MASSE sur un contrôle `high`, déclenché par
// une lacune de collecte que rien ne signalait. Mesuré avant correction : deux volumes
// réellement sauvegardés par des snapshots fraîches et terminées, tous deux en écart.
//
// L'AND est entre les types, le OR entre les attributs d'un même type : chaque type que
// le contrôle lit doit avoir sa donnée décisive, sans quoi il conclut sur ce qu'il n'a
// pas vu. Les types nommés ici sont confrontés à `genprovider.ControlTypes` par
// TestDecidingAttributesDeclareKnownTypes — la chaîne va du Rego aux types, puis des
// types aux attributs, et aucun maillon ne flotte.
var requiredAttr = map[string]map[string][]string{
	"compute_instance_deletion_protection":               {"": {"deletion_protection"}},
	"compute_instance_no_secrets_in_user_data":           {"": {"user_data"}},
	"compute_instance_has_security_group":                {"": {"security_group_ids"}},
	"compute_instance_public_ip_with_open_securitygroup": {"": {"public_ip"}},
	"compute_image_not_public":                           {"": {"public"}},
	"iam_no_root_access_key":                             {"": {"root_owned", "scope"}},
	// Sans date de création, l'ÂGE d'une clé ne se déduit pas : une absence n'est pas
	// une rotation récente. Le contrôle sort « non évalué » plutôt que conforme.
	"iam_accesskey_rotated":                    {"": {"creation_date"}},
	"iam_user_mfa_enabled":                     {"": {"mfa_enabled"}},
	"iam_account_mfa_enforced":                 {"": {"require_trusted_env"}},
	"iam_apiaccesspolicy_max_key_expiration":   {"": {"max_access_key_expiration_seconds"}},
	"blockstorage_volume_encryption":           {"": {"encrypted"}},
	"blockstorage_snapshot_not_public":         {"": {"global_permission"}},
	"objectstorage_bucket_object_lock_enabled": {"": {"object_lock_enabled"}},
	"objectstorage_bucket_kms_encryption":      {"": {"sse_kms_enabled"}},
	"objectstorage_bucket_versioning_enabled":  {"": {"versioning"}},
	"objectstorage_bucket_default_encryption":  {"": {"default_encryption_enabled"}},
	// Un 403 sur GetBucketAcl ne doit pas rendre un bucket public « conforme ». La liste
	// n'accepte QUE des signaux d'ACL, volontairement : `policy_public` en faisait partie,
	// or collectBucket interroge ACL et policy SÉPARÉMENT (best effort). Un 403 sur
	// GetBucketAcl suivi d'un GetBucketPolicy réussi posait donc `policy_public: false`
	// seul, ce qui franchissait le verrou et concluait « conforme » sur une ACL jamais lue —
	// alors que l'ACL est le vecteur d'exposition le plus courant. Sans signal d'ACL, le
	// contrôle sort « non évalué », ce qui est la réponse honnête.
	"objectstorage_bucket_public_access":  {"": {"acl", "acl_grants", "public_via_acl"}},
	"database_encryption_at_rest_enabled": {"": {"encryption_at_rest"}},
	// Sans cette entrée, une LBU dont `access_log` n'a pas été collecté rendait un
	// `pass` SILENCIEUX depuis que la règle a cessé de conclure sur une absence.
	// La règle se tait, le verrou dit pourquoi : « non collecté », pas « conforme ».
	"loadbalancer_logging_enabled":                      {"": {"access_log"}},
	"loadbalancer_http_redirect_to_https":               {"": {"redirect_to_https"}},
	"loadbalancer_ssl_listeners":                        {"": {"load_balancer_type"}},
	"network_subnet_no_public_ip_by_default":            {"": {"map_public_ip_on_launch"}},
	"network_securitygroup_default_restrict_traffic":    {"": {"security_group_name"}},
	"network_peering_cross_organization":                {"": {"source_account", "accepter_account"}},
	"kubernetes_cluster_not_publicly_accessible":        {"": {"admin_whitelist"}},
	"kubernetes_cluster_control_plane_highly_available": {"": {"control_plane_multi_az"}},
	"kubernetes_cluster_auto_upgrade_enabled":           {"": {"auto_upgrade"}},
	"kubernetes_cluster_deletion_protection":            {"": {"deletion_protection"}},
	"kubernetes_cluster_audit_logging_enabled":          {"": {"audit_enabled"}},
	// Ce contrôle lit la RÉGION de chaque ressource (pas la ressource synthétique du
	// fournisseur) : sans région collectée, il ne mesure rien.
	"governance_resource_region_in_eu": {"": {"region"}},

	// --- Verrous ajoutés après l'audit de livraison -------------------------
	// Ces contrôles concluaient « conforme » alors que l'attribut qui porte leur
	// décision n'avait jamais été collecté : la règle ne se déclenchait pas, donc
	// aucun finding, donc « pass ». Un faux vert est invisible par construction,
	// ce qui en fait le pire défaut possible pour un outil de posture.

	// La base est ouverte tant qu'aucune ACL n'est posée (défaut d'API Scaleway) :
	// sans ip_filter collecté, « conforme » est exactement l'inverse de la réalité.
	"database_service_not_open_to_internet": {"": {"ip_filter"}},
	"database_backup_enabled":               {"": {"disable_backup"}},

	// `statements` est TOUJOURS posé par le collecteur, au besoin à [] quand le
	// document n'a pas pu être analysé : le verrou ne vaut que parce que
	// attrsByTypeOf ne compte plus une collection vide comme collectée.
	"iam_policy_no_administrative_privileges": {"": {"statements"}},
	"iam_policy_no_notaction_notresource":     {"": {"statements"}},
	"iam_policy_no_wildcard_resource":         {"": {"statements"}},
	"iam_policy_no_privilege_escalation":      {"": {"statements", "manages_iam"}},
	"iam_role_key_lifetime_bounded":           {"": {"max_session_ttl", "policy_has_expiration"}},
	"iam_role_no_admin_privileges":            {"": {"admin_privileges"}},
	"iam_role_source_ip_restricted":           {"": {"source_ip_restricted"}},
	"iam_apiaccessrule_no_public_cidr":        {"": {"ip_ranges"}},

	"network_securitygroup_default_deny": {"": {"inbound_default_policy"}},
	"network_flow_matrix_documented":     {"": {"description"}},

	// La cartographie réseau se juge sur les ÉTIQUETTES du réseau. Sans elles, la
	// règle se tait (garde de capacité) et le contrôle s'affichait « conforme » : le
	// faux vert exact que l'issue #47 décrit, un contrôle qui valide une conformité
	// qu'il n'a pas mesurée.
	"network_documented": {"": {"tags"}},

	// Sur un plan Terraform, `state` arrive en after_unknown et aucun
	// blockstorage_snapshot n'existe : le contrôle ne peut PAS y être évalué,
	// et s'affichait pourtant vert sur l'exemple livré dans le dépôt.
	//
	// Et `volume_id` sur les SNAPSHOTS, qui est le lien vers le volume. Sans lui
	// aucune snapshot n'est attribuable, donc aucun volume n'a de sauvegarde
	// visible, donc TOUS sortent en écart `high` — alors que la lacune est dans la
	// collecte, pas dans la sauvegarde. C'est le type secondaire que la table ne
	// savait pas exprimer avant #133.
	"blockstorage_volume_snapshots_exist": {
		"":                      {"state"},
		"blockstorage_snapshot": {"volume_id"},
	},
}

// RequiredAttrs retourne, par contrôle, les attributs dont la PRÉSENCE conditionne un « pass »
// (copie : la table interne reste immuable). Exposé pour que la documentation de couverture
// (internal/docgen) soit DÉRIVÉE de la table réellement appliquée, jamais recopiée à côté.
// L'APLATISSEMENT convient à qui demande « de quelle donnée ce contrôle a-t-il
// besoin ? » — la matrice de couverture s'en sert pour savoir si un contrôle est gaté.
// Il ne convient PAS à qui AFFICHE la réponse : joindre les attributs par « ou »
// dirait « state ou volume_id » là où le contrôle exige « state sur les volumes ET
// volume_id sur les snapshots ». Pour afficher, prendre DecidingAttrsByType.
func RequiredAttrs() map[string][]string {
	out := make(map[string][]string, len(requiredAttr))
	for code, parType := range requiredAttr {
		seen := map[string]bool{}
		var attrs []string
		for _, liste := range parType {
			for _, a := range liste {
				if seen[a] {
					continue
				}
				seen[a] = true
				attrs = append(attrs, a)
			}
		}
		sort.Strings(attrs) // ordre stable : la doc générée en dépend
		out[code] = attrs
	}
	return out
}

// DecidingAttrsByType rend, par contrôle puis par TYPE, les attributs décisifs — la
// déclaration telle qu'elle est réellement appliquée, avec le type PRINCIPAL résolu en
// son nom concret pour que l'affichage n'ait pas à refaire la dérivation.
//
// C'est la vue à utiliser pour montrer l'exigence à un humain : « state sur les
// volumes, volume_id sur les snapshots » se lit, là où l'union à plat ment sur la
// conjonction.
func DecidingAttrsByType() map[string]map[string][]string {
	out := make(map[string]map[string][]string, len(requiredAttr))
	for code, parType := range requiredAttr {
		concret := map[string][]string{}
		for t, attrs := range parType {
			nom := t
			if nom == "" {
				nom = genprovider.ControlType(code)
			}
			liste := append([]string(nil), attrs...)
			sort.Strings(liste)
			concret[nom] = append(concret[nom], liste...)
		}
		out[code] = concret
	}
	return out
}

// governanceProviderReaders : contrôles de gouvernance dont la donnée EST la ressource
// synthétique `governance_provider` (toujours construite depuis le descripteur du
// fournisseur). Leur source est donc intrinsèquement collectée ; les autres contrôles de
// gouvernance (ex. étiquettes, qui dépendent de la collecte d'attributs par ressource) ne
// sont PAS présumés vérifiés.
// `governance_resource_region_in_eu` n'y figure PAS, bien qu'il y ait figuré : il lit
// la région de chaque ressource du tenant, pas le descripteur. L'y inscrire lui
// accordait le verrou sans condition, et trois mécanismes se contredisaient alors sur
// le même contrôle — `requiredAttr` le tenait par « region », `ControlType` rendait ""
// et ce marqueur le déclarait intrinsèquement vérifié. C'est ce dernier qui gagnait.
// Il passe désormais par ses types déclarés, comme n'importe quel contrôle mesuré.
var governanceProviderReaders = map[string]bool{
	"governance_provider_sovereignty": true,
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
		// Contrôle TRANSVERSE : aucun type déduit de son code, mais des types
		// déclarés — le contrôle de souveraineté, qui mesure la localisation sur les
		// sept types qui hébergent quelque chose. Le contrat doit confirmer qu'AU
		// MOINS un de ces types est collecté pour ce fournisseur, sans quoi il n'y a
		// rien à localiser. Ce que le contrat ne dit pas, en revanche, c'est si la
		// région est arrivée sur chaque ressource : c'est `attrCollected` qui le
		// mesure sur l'inventaire, et c'est lui le verrou effectif du « pass ».
		for _, t := range genprovider.ControlTypes(code) {
			if genprovider.TypeEtat(provider, t) == "verifie" {
				return true
			}
		}
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
		// Un contrôle de gouvernance qui DÉCLARE les types qu'il lit s'applique
		// seulement si l'un d'eux est là. Le contrôle de souveraineté en déclare sept
		// et n'en juge aucun autre : sans ressource localisée, il n'a rien à dire —
		// et « rien à dire » ne s'écrit pas « conforme ».
		if declared := genprovider.ControlTypes(code); len(declared) > 0 {
			for _, t := range declared {
				if resourceTypes[t] {
					return true
				}
			}
			return false
		}
		return true
	}
	t := controlType[code]
	if t == "" {
		return false
	}
	return resourceTypes[t]
}

// correlationBroken dit si un contrôle CORRÉLANT plusieurs types a perdu le lien qui
// le rend concluant, et pourquoi.
//
// Le verrou de capacité ne protège que la branche « pass » : dès qu'une règle émet un
// écart, il n'est jamais consulté. Or une donnée manquante sur un type SECONDAIRE ne
// produit pas un faux vert, elle produit son contraire — un faux ÉCART, et de masse.
// `volume_id` non collecté sur les snapshots casse la jointure pour TOUS les volumes,
// et chacun sort « sans sauvegarde » alors que rien de tel n'a été observé.
//
// La distinction avec le type PRINCIPAL est délibérée et c'est elle qui borne le
// mécanisme. Sur le type principal, la règle juge ressource par ressource : celle dont
// l'attribut manque ne déclenche pas, les autres restent des écarts RÉELS qu'il serait
// grave de taire. Sur un type secondaire, l'absence casse la corrélation pour toutes,
// et aucun des écarts n'est alors adossé à une observation.
//
// On ne requalifie donc que ce second cas, et seulement quand le type secondaire est
// PRÉSENT : l'absence totale de snapshots est une observation — rien n'est sauvegardé —
// tandis que des snapshots privées de leur lien sont une lacune de collecte.
func correlationBroken(code string, resourceTypes map[string]bool, attrsByType map[string]map[string]bool) (string, bool) {
	parType := requiredAttr[code]
	if len(parType) == 0 {
		return "", false
	}
	types := make([]string, 0, len(parType))
	for t := range parType {
		if t != "" && resourceTypes[t] {
			types = append(types, t)
		}
	}
	sort.Strings(types) // ordre stable : ce motif part dans un rapport opposable
	var rompu []string
	for _, t := range types {
		if attrsOnType(code, t, parType[t], attrsByType) {
			continue
		}
		attrs := append([]string(nil), parType[t]...)
		sort.Strings(attrs)
		rompu = append(rompu, fmt.Sprintf(i18n.T(
			"« %s » non collecté sur les ressources de type « %s »",
			"\"%s\" not collected on the resources of type \"%s\""), strings.Join(attrs, " / "), t))
	}
	if len(rompu) == 0 {
		return "", false
	}
	return fmt.Sprintf(i18n.T(
		"corrélation impossible : %s — l'écart ne peut être ni établi ni écarté (garde de capacité)",
		"correlation impossible: %s — the deviation can be neither established nor ruled out (capability guard)"),
		strings.Join(rompu, ", ")), true
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
	// Un contrôle dont la CORRÉLATION est rompue ne produit pas des écarts, il produit
	// du bruit : le motif est calculé une fois par contrôle, pas par finding.
	rompu := map[string]string{}
	for _, f := range findings {
		if _, deja := rompu[f.Code]; deja {
			continue
		}
		if motif, cassee := correlationBroken(f.Code, resourceTypes, attrsByType); cassee {
			rompu[f.Code] = motif
		}
	}
	var results []assessment.Result
	for _, f := range findings {
		// Un finding INCONCLUANT n'est pas un écart : la règle a vu la donnée et n'a
		// pas pu trancher. Il devient `not-evaluated`, avec sa raison (ADR-0015).
		// `failedControls` reste marqué : la ligne par sujet est déjà émise, la
		// branche pass/not-evaluated ne doit pas en ajouter une seconde.
		status := assessment.Fail
		evidence := stripSubject(f.Message)
		if IsInconclusive(f) {
			status = assessment.NotEvaluated
		}
		// Même conclusion, cause différente : ici ce n'est pas la règle qui constate
		// qu'elle ne sait pas, c'est l'assessment qui sait que la donnée reliant les
		// deux types n'est jamais arrivée. Affirmer l'écart serait l'inventer.
		if motif, cassee := rompu[f.Code]; cassee {
			status = assessment.NotEvaluated
			evidence = motif
		}
		failedControls[f.Code] = true
		c := controls[f.Code]
		results = append(results, assessment.Result{
			Control:     f.Code,
			Title:       first(c.TitreIn(i18n.Current()), f.Title),
			Status:      status,
			Severity:    first(f.Severity, c.Severite),
			Subject:     f.Subject,
			Evidence:    assessment.Evidence{Observed: evidence, Source: run.Source},
			References:  References(c),
			Remediation: first(f.Remediation, c.RemediationIn(i18n.Current())),
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
			Title:      c.TitreIn(i18n.Current()),
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
			// La justification est AUSSI portée par `evidence.observed`. Elle était déjà
			// là, dans `waiver`, mais `not-applicable` était le seul statut dont
			// `evidence` restait nul : un lecteur qui parcourt les preuves d'un rapport
			// trouvait une explication partout sauf ici, et devait savoir qu'un autre
			// champ existait. Rien n'est retiré, rien ne bouge : la même phrase est
			// lisible là où toutes les autres le sont.
			res.Evidence = assessment.Evidence{Observed: naReasons[code], Source: run.Source}
		case contains(c.Fournisseurs, provider):
			// Implemented for this provider. Pass is asserted ONLY when the contract confirms
			// the needed data is collected (verified) AND a resource of that service is present.
			// Otherwise the control could not actually be evaluated (attribute not collected, or
			// nothing of this type in scope) — NotEvaluated, never a silent Pass.
			switch {
			case verified[code] && applicable(code, controlType, resourceTypes) && attrCollected(code, typ, resourceTypes, attrsByType):
				// A Pass carries WHAT was checked (basis of the assertion), not just a status.
				res.Status = assessment.Pass
				observed := i18n.T(
					"aucune non-conformité détectée (contrat vérifié)",
					"no deviation detected (contract verified)")
				if typ != "" {
					observed = fmt.Sprintf(i18n.T(
						"aucune non-conformité détectée sur les ressources de type « %s » collectées (contrat vérifié)",
						"no deviation detected on the collected resources of type \"%s\" (contract verified)"), typ)
				} else if strings.HasPrefix(code, "governance_provider_") {
					// Réservé aux contrôles dont la donnée EST le descripteur. Les
					// governance_resource_* sont mesurés sur le tenant : leur attribuer
					// cette preuve serait un mensonge.
					observed = i18n.T(
						"conforme selon les faits de souveraineté déclarés au descripteur du fournisseur (attestation, non mesuré sur le tenant)",
						"compliant according to the sovereignty facts declared in the provider descriptor (an attestation, not measured on the tenant)")
				} else if scope := typesInScope(code, typ, resourceTypes); len(scope) > 0 {
					// Contrôle transverse MESURÉ sur le tenant : la preuve nomme les types
					// sur lesquels la donnée a effectivement été observée. « Contrat
					// vérifié » ne disait rien de ce qui avait été regardé, et c'est
					// précisément la phrase qui accompagnait le faux vert de la
					// souveraineté — un bucket sans région, déclaré conforme.
					sorted := append([]string(nil), scope...)
					sort.Strings(sorted)
					observed = fmt.Sprintf(i18n.T(
						"aucune non-conformité détectée sur les ressources de type « %s », dont la donnée décisive a été observée sur chacune",
						"no deviation detected on the resources of type \"%s\", whose deciding data was observed on every one of them"),
						strings.Join(sorted, " / "))
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
// requireAll énumère les contrôles dont les attributs déclarés se combinent en ET,
// et non en OU.
//
// Le défaut : `requiredAttr` portait une LISTE, et la boucle rendait vrai au premier
// attribut trouvé — un `any-of` implicite que personne n'avait décidé. Pour la
// plupart des contrôles c'est juste : plusieurs signaux alternatifs établissent le
// même fait. Pour celui-ci, non.
//
// `network_peering_cross_organization` COMPARE les deux comptes. Avec un seul, il n'y
// a rien à comparer, et le contrôle franchissait pourtant sa porte.
var requireAll = map[string]bool{
	"network_peering_cross_organization": true,
}

// typesInScope rend les types sur lesquels ce contrôle JUGE réellement dans cet
// inventaire — ceux qu'il lit et qui sont présents.
//
// Pour un contrôle ordinaire, c'est le type déduit de son code, et cette fonction ne
// change rien. Elle existe pour le contrôle TRANSVERSE : `ControlType` rend "" pour
// la gouvernance, et le contrôle de souveraineté lit pourtant sept types. Sans elle,
// sa donnée décisive était cherchée sous une clé vide alimentée par n'importe quelle
// ressource — un `iam_user` localisé certifiait une VM qui ne l'était pas.
func typesInScope(code, typ string, resourceTypes map[string]bool) []string {
	if typ != "" {
		return []string{typ}
	}
	var out []string
	for _, t := range genprovider.ControlTypes(code) {
		if resourceTypes[t] {
			out = append(out, t)
		}
	}
	return out
}

// attrCollected dit si la donnée qui DÉCIDE a été collectée pour ce contrôle.
//
// `attrsByType` est construit par INTERSECTION sur les ressources d'un type
// (cf. attrsByTypeOf) : un attribut n'y figure que si CHAQUE ressource le porte.
// C'est ce qui empêche une ressource voisine d'ouvrir la porte du `pass` à une
// ressource dont on n'a rien observé.
//
// Un contrôle transverse doit l'avoir sur CHACUN de ses types en scope : la
// souveraineté ne se conclut pas des VMs vers les buckets. Et un scope VIDE ne
// conclut rien — il n'y a alors aucune ressource à localiser, ce que
// `notEvaluatedReason` dit avec ses mots.
// Chaque type que le contrôle lit doit avoir SA donnée décisive : `volume_id` manquant
// sur les snapshots ne dit rien des volumes, mais il rend le lien illisible, et un lien
// illisible n'est pas une absence de sauvegarde.
func attrCollected(code, typ string, resourceTypes map[string]bool, attrsByType map[string]map[string]bool) bool {
	exige := decidingByType(code, typ, resourceTypes)
	if len(requiredAttr[code]) == 0 {
		return true
	}
	if len(exige) == 0 {
		return false
	}
	for t, attrs := range exige {
		if !attrsOnType(code, t, attrs, attrsByType) {
			return false
		}
	}
	return true
}

// decidingByType rend, pour cet inventaire, les attributs à exiger SUR CHAQUE TYPE.
//
// La clé `""` de la déclaration vise le type principal — ou, pour un contrôle
// transverse qui n'en a pas, tous les types qu'il lit et qui sont présents. Un type
// SECONDAIRE nommé n'est exigé que s'il est présent : l'absence totale de snapshots
// est une OBSERVATION (rien n'est sauvegardé), là où des snapshots privées de leur
// `volume_id` sont une lacune de collecte. Confondre les deux est précisément ce qui
// produisait le faux positif de masse.
func decidingByType(code, typ string, resourceTypes map[string]bool) map[string][]string {
	out := map[string][]string{}
	for t, attrs := range requiredAttr[code] {
		if t != "" {
			if resourceTypes[t] {
				out[t] = append(out[t], attrs...)
			}
			continue
		}
		for _, s := range typesInScope(code, typ, resourceTypes) {
			out[s] = append(out[s], attrs...)
		}
	}
	return out
}

// attrsOnType applique la règle de suffisance d'un contrôle à UN type : tous les
// attributs déclarés (`requireAll`), ou au moins un.
func attrsOnType(code, typ string, attrs []string, attrsByType map[string]map[string]bool) bool {
	if requireAll[code] {
		for _, a := range attrs {
			if !attrsByType[typ][a] {
				return false
			}
		}
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
		return i18n.T(
			"collecte de la donnée nécessaire non confirmée pour ce fournisseur (contrat non « vérifié »)",
			"collection of the required data is not confirmed for this provider (contract not \"verified\")")
	}
	if typ != "" && !resourceTypes[typ] {
		return fmt.Sprintf(i18n.T(
			"aucune ressource de type « %s » dans l'inventaire évalué",
			"no resource of type \"%s\" in the assessed inventory"), typ)
	}
	scope := typesInScope(code, typ, resourceTypes)
	// Contrôle transverse dont AUCUN type lu n'est présent : il n'y a rien à juger.
	// Le dire ainsi vaut mieux que « attribut non collecté », qui laisserait croire à
	// une collecte défaillante là où l'inventaire ne contient simplement rien de
	// localisable.
	if typ == "" && len(scope) == 0 && len(genprovider.ControlTypes(code)) > 0 {
		return i18n.T(
			"aucune ressource des types que ce contrôle examine dans l'inventaire évalué",
			"no resource of the types this control examines in the assessed inventory")
	}
	if len(requiredAttr[code]) > 0 && !attrCollected(code, typ, resourceTypes, attrsByType) {
		// Nommer le COUPLE (type, attribut) qui manque, pas l'union des attributs du
		// contrôle : dire « volume_id non collecté » sans dire « sur les snapshots »
		// envoie l'opérateur chercher au mauvais endroit, puisque le contrôle porte le
		// nom du volume.
		exige := decidingByType(code, typ, resourceTypes)
		types := make([]string, 0, len(exige))
		for t := range exige {
			types = append(types, t)
		}
		sort.Strings(types) // ordre stable : ce message part dans un rapport opposable
		type lacune struct{ attrs, typ string }
		var manquants []lacune
		for _, t := range types {
			if attrsOnType(code, t, exige[t], attrsByType) {
				continue
			}
			attrs := append([]string(nil), exige[t]...)
			sort.Strings(attrs)
			manquants = append(manquants, lacune{attrs: strings.Join(attrs, " / "), typ: t})
		}
		// Un seul type manquant garde la phrase d'origine, qui se lit mieux : le verbe
		// suit son sujet. Au-delà, la liste passe DEVANT, car accumuler les compléments
		// avant le participe produit « attribut « a » sur « X », « b » sur « Y » non
		// collecté », où le verbe arrive après trois virgules sans plus se rattacher à
		// rien.
		if len(manquants) == 1 {
			return fmt.Sprintf(i18n.T(
				"attribut « %s » non collecté sur les ressources de type « %s » (garde de capacité)",
				"attribute \"%s\" not collected on the resources of type \"%s\" (capability guard)"),
				manquants[0].attrs, manquants[0].typ)
		}
		clauses := make([]string, 0, len(manquants))
		for _, m := range manquants {
			clauses = append(clauses, fmt.Sprintf(i18n.T(
				"« %s » sur les ressources de type « %s »",
				"\"%s\" on the resources of type \"%s\""), m.attrs, m.typ))
		}
		return fmt.Sprintf(i18n.T(
			"attributs décisifs non collectés : %s (garde de capacité)",
			"deciding attributes not collected: %s (capability guard)"), strings.Join(clauses, ", "))
	}
	return i18n.T("contrôle non évaluable sur cet inventaire", "control not evaluable on this inventory")
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

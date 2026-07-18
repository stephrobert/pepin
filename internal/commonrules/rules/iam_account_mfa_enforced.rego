# iam_account_mfa_enforced — COMMUN à tous les providers.
#   La politique d'accès API du compte n'impose pas la MFA à l'ensemble des
#   utilisateurs : un mot de passe compromis suffit (pas d'environnement de confiance).
# SCSL : CLD-IAM-3 (MFA imposée sur les accès d'administration et la console).
#   Angle COMPTE (complémentaire de iam_user_mfa_enabled, qui est par utilisateur).
#   Contrat Outscale : ApiAccessPolicy.RequireTrustedEnv (bool) — true ⇒ tous les
#   EIM users doivent se connecter en MFA (WebAuthn/OTP). Source : doc officielle
#   docs.outscale.com (About Your API Access Policy) + osc-sdk-go/v2.
#   Ne se déclenche que si require_trusted_env vaut explicitement false.
package pepin.rules

import rego.v1

deny contains f if {
	some r in input.resources
	r.type == "api_access_policy"
	object.get(r.attributes, "require_trusted_env", true) == false
	name := object.get(r, "name", r.id)
	f := {
		"code": "iam_account_mfa_enforced",
		"severity": "high",
		"subject": name,
		"message": "La politique d'accès API n'impose pas la MFA à tous les utilisateurs (environnement de confiance désactivé).",
		"remediation": "Activer l'exigence d'environnement de confiance (RequireTrustedEnv) pour imposer la MFA à tous les comptes ; configurer WebAuthn/OTP.",
		"labels": {"provider": provider_of(r), "category": "security"},
	}
}

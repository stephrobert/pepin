# iam_user_mfa_enabled — COMMUN à tous les providers.
#   Compte/utilisateur cloud sans authentification multifacteur (MFA) : un mot de
#   passe compromis suffit à accéder à la console/API.
# SCSL : CLD-IAM-3 (MFA imposée sur les accès d'administration et la console).
#   Type normalisé `iam_user`, attribut booléen `mfa_enabled`. Contrats :
#   Scaleway api/iam User.mfa ; Exoscale User.two-factor-authentication. Ne se
#   déclenche que si mfa_enabled vaut explicitement false (absent ⇒ pas de finding).
package pepin.rules

import rego.v1

deny contains f if {
	some r in input.resources
	r.type == "iam_user"
	object.get(r.attributes, "mfa_enabled", true) == false
	name := object.get(r.attributes, "username", object.get(r, "name", r.id))
	f := {
		"code": "iam_user_mfa_enabled",
		"severity": "high",
		"subject": name,
		"message": sprintf("Compte « %s » sans authentification multifacteur (MFA) activée.", [name]),
		"remediation": "Activer la MFA sur le compte ; l'imposer pour tous les accès d'administration et la console cloud.",
		"labels": {"provider": provider_of(r), "category": "security"},
	}
}

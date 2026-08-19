# Borne de durée de vie des accès d'un rôle IAM.
#   Type normalisé agnostique `iam_role`. Attributs DÉRIVÉS par le collecteur :
#   max_session_ttl (entier, sec ; 0/absent = non borné) et policy_has_expiration
#   (bool ; la politique refuse l'usage au-delà d'une durée).
# Posture (choix « sécuriser au maximum », sans imposer un montage unique) : le rôle
#   est conforme s'il borne la durée de vie par AU MOINS un moyen — TTL de session
#   maximal OU expiration dans la politique. Sinon, les accès sont permanents.
# Ancrage Exoscale : max-session-ttl du rôle (GET /iam-role) ; expiration via
#   expression CEL timestamp(identity.created) < timestamp(now) - duration(…)
#   (references/docs/exoscale/product-iam-how-to-policy-guide.md). En Terraform,
#   max-session-ttl est absent du schéma → seule l'expiration de la politique compte.
# SCSL : CLD-IAM-2 (clés d'API à durée de vie limitée).
package pepin.rules

import rego.v1

# Rôle dont la durée de vie des accès n'est bornée par aucun moyen. Ne se déclenche
# que pour les rôles dont le collecteur expose la capacité (au moins un des deux
# attributs renseigné) — un provider sans ces dérivés ne déclenche pas la règle.
deny contains f if {
	some r in resources_of_type("iam_role")
	truthy(object.get(r.attributes, "editable", true)) # rôles prédéfinis (non éditables) hors scope
	_role_exposes_lifetime(r)
	not _lifetime_bounded(r)
	name := object.get(r.attributes, "name", r.id)
	f := {
		"code": "iam_role_key_lifetime_bounded",
		"severity": "critical",
		"subject": name,
		"message": sprintf("Rôle IAM « %s » : aucune borne de durée de vie (ni TTL de session, ni expiration) — les accès assumant ce rôle sont permanents.", [name]),
		"remediation": "Borner la durée de vie : définir un TTL de session maximal sur le rôle, ou une condition d'expiration dans sa politique.",
		"labels": {
			"provider": provider_of(r),
			"category": "security",
			"message_en": sprintf("IAM role \"%s\": no bound on credential lifetime (no session TTL, no expiry) — credentials assuming this role are permanent.", [name]),
			"remediation_en": "Bound that lifetime: set a maximum session TTL on the role, or an expiry condition in its policy.",
		},
	}
}

# Le provider expose-t-il la borne de durée de vie pour ce rôle (évite les faux
# positifs sur un provider/type qui n'a aucun de ces attributs) ?
_role_exposes_lifetime(r) if "max_session_ttl" in object.keys(r.attributes)

_role_exposes_lifetime(r) if "policy_has_expiration" in object.keys(r.attributes)

# Durée de vie bornée par TTL de session maximal…
_lifetime_bounded(r) if to_number(object.get(r.attributes, "max_session_ttl", 0)) > 0

# …ou par une expiration exprimée dans la politique.
_lifetime_bounded(r) if truthy(object.get(r.attributes, "policy_has_expiration", false))

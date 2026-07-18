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
	object.get(r.attributes, "editable", true) == true # rôles prédéfinis (non éditables) hors scope
	role_exposes_lifetime(r)
	not lifetime_bounded(r)
	name := object.get(r.attributes, "name", r.id)
	f := {
		"code": "iam_role_key_lifetime_bounded",
		"severity": "critical",
		"subject": name,
		"message": sprintf("Rôle IAM « %s » : aucune borne de durée de vie (ni TTL de session, ni expiration) — les accès assumant ce rôle sont permanents.", [name]),
		"remediation": "Borner la durée de vie : définir un TTL de session maximal sur le rôle, ou une condition d'expiration dans sa politique.",
		"labels": {"provider": provider_of(r), "category": "security"},
	}
}

# Le provider expose-t-il la borne de durée de vie pour ce rôle (évite les faux
# positifs sur un provider/type qui n'a aucun de ces attributs) ?
role_exposes_lifetime(r) if "max_session_ttl" in object.keys(r.attributes)

role_exposes_lifetime(r) if "policy_has_expiration" in object.keys(r.attributes)

# Durée de vie bornée par TTL de session maximal…
lifetime_bounded(r) if to_number(object.get(r.attributes, "max_session_ttl", 0)) > 0

# …ou par une expiration exprimée dans la politique.
lifetime_bounded(r) if object.get(r.attributes, "policy_has_expiration", false) == true

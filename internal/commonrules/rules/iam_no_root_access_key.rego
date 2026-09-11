# iam_no_root_access_key — Clé d'API rattachée au compte root (contourne l'IAM).
# Origine : Pépin (POC Scaleway). SCSL : CLD-IAM-1.
# Contrat : type normalisé agnostique `access_key`. Une clé est « root » selon deux
#   dérivations (any-of, selon le fournisseur) :
#   - `root_owned: true` : flag explicite posé par le collecteur (Scaleway : APIKey.UserID
#     == propriétaire de l'organisation).
#   - Outscale : différence d'ensembles. Les clés sont collectées en deux passes taggées :
#     `scope: account` (ReadAccessKeys de l'appelant) et `scope: eim` (ReadAccessKeys par
#     user EIM, avec owner_user). Une clé `scope: account` dont l'id N'EST PAS dans
#     l'ensemble des ids `scope: eim` appartient au root (non attribuable à un user EIM).
package pepin.rules

import rego.v1

# Ensemble des ids de clés attribuées à un utilisateur EIM (donc NON-root).
_eim_key_ids contains id if {
	some r in input.resources
	r.type == "access_key"
	object.get(r.attributes, "scope", "") == "eim"
	id := object.get(r.attributes, "access_key_id", "")
}

# root_owned explicite (Scaleway/fixtures).
_root_key(r) if truthy(object.get(r.attributes, "root_owned", false))

# root_owned dérivé (Outscale) : clé de niveau compte non attribuable à un user EIM.
_root_key(r) if {
	object.get(r.attributes, "scope", "") == "account"
	id := object.get(r.attributes, "access_key_id", "")
	not id in _eim_key_ids
}

# Ensemble des identifiants d'utilisateurs PROPRIÉTAIRES de l'organisation.
#
# Scaleway le dit littéralement : chaque utilisateur porte `type`, dont la valeur est
# `owner` ou `member` (GET /iam/v1alpha1/users, vérifié contre l'API réelle le
# 2026-09-11). C'est le seul marqueur fiable — `account_root_user_id`, porté par le même
# enregistrement, désigne autre chose et diffère de l'id du propriétaire.
_owner_user_ids contains id if {
	some r in input.resources
	r.type == "iam_user"
	lower(object.get(r.attributes, "user_type", "")) == "owner"
	id := object.get(r.attributes, "user_id", "")
	id != ""
}

# root_owned dérivé (Scaleway) : la clé appartient à l'utilisateur PROPRIÉTAIRE.
#
# Ce que cette jointure évite. Une clé d'API Scaleway porte `user_id` (un humain) ou
# `application_id` (une identité de service). S'arrêter à « c'est une clé
# d'utilisateur » signalerait la clé de CHAQUE membre qui en a une : un faux positif par
# personne, et dix de ces findings apprennent à ignorer l'outil. Ce que le contrôle vise,
# c'est la clé qui contourne l'IAM parce qu'elle appartient au compte qui possède
# l'organisation — pas celle d'un collègue.
_root_key(r) if {
	uid := object.get(r.attributes, "owner_user_id", "")
	uid != ""
	uid in _owner_user_ids
}

deny contains f if {
	some r in input.resources
	r.type == "access_key"
	_root_key(r)
	name := object.get(r, "name", object.get(r.attributes, "access_key_id", r.id))
	f := {
		"code": "iam_no_root_access_key",
		"severity": "high",
		"subject": name,
		"message": sprintf("Clé d'API « %s » rattachée au compte root — contourne les politiques IAM.", [name]),
		"remediation": "Créer une application IAM dédiée à moindre privilège puis révoquer la clé root.",
		"labels": {
			"provider": provider_of(r),
			"category": "security",
			"confidence": "confirmed",
			"message_en": sprintf("API key \"%s\" attached to the root account — it bypasses IAM policies.", [name]),
			"remediation_en": "Create a dedicated least-privilege IAM application, then revoke the root key.",
		},
	}
}

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
		"labels": {"provider": provider_of(r), "category": "security"},
	}
}

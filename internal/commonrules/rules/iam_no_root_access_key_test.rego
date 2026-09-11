package pepin.rules

import rego.v1

_sw_iam_code := "iam_no_root_access_key"

_sw_key(attrs) := {"resources": [{
	"provider": "scaleway", "type": "access_key", "id": "k1", "name": "ci-deploy",
	"attributes": attrs,
}]}

# ✗ Clé rattachée au compte root → finding.
test_sw_root_key_denied if {
	some f in deny with input as _sw_key({"root_owned": true})
	f.code == _sw_iam_code
}

# ✓ Clé rattachée à une application dédiée → aucun finding.
test_sw_app_key_ok if {
	count({f | some f in deny; f.code == _sw_iam_code}) == 0 with input as _sw_key({"root_owned": false, "application_id": "app-1"})
}

# ── Dérivation Outscale (différence d'ensembles scope account / eim) ──

# Compte avec : 1 clé de niveau compte "acct-1" (root, absente de l'ensemble EIM),
# 1 clé de niveau compte "shared-1" QUI EST aussi une clé EIM (donc NON root),
# 1 clé EIM "eim-1". Attendu : seul "acct-1" est flaggé root.
_osc_keys := {"resources": [
	{"provider": "outscale", "type": "access_key", "id": "acct-1", "attributes": {"access_key_id": "acct-1", "scope": "account"}},
	{"provider": "outscale", "type": "access_key", "id": "shared-1", "attributes": {"access_key_id": "shared-1", "scope": "account"}},
	{"provider": "outscale", "type": "access_key", "id": "eim-1", "attributes": {"access_key_id": "eim-1", "scope": "eim", "owner_user": "robert"}},
	{"provider": "outscale", "type": "access_key", "id": "shared-1b", "attributes": {"access_key_id": "shared-1", "scope": "eim", "owner_user": "robert"}},
]}

# ✗ Clé de niveau compte non attribuable à un user EIM → root → finding.
test_osc_root_account_key_denied if {
	some f in deny with input as _osc_keys
	f.code == _sw_iam_code
	f.subject == "acct-1"
}

# ✓ Une clé scope:account AUSSI présente en scope:eim (scan sous un user EIM) → NON root.
test_osc_shared_key_not_root if {
	count({f | some f in deny; f.code == _sw_iam_code; f.subject == "shared-1"}) == 0 with input as _osc_keys
}

# ✓ Une clé purement EIM → jamais root.
test_osc_eim_key_not_root if {
	count({f | some f in deny; f.code == _sw_iam_code; f.subject == "eim-1"}) == 0 with input as _osc_keys
}

# ── La dérivation Scaleway : la clé du PROPRIÉTAIRE, pas celle d'un collègue ────
#
# Le piège que ces cas gardent. Une clé d'API Scaleway porte `user_id` (un humain) ou
# `application_id` (une identité de service). S'arrêter à « c'est une clé
# d'utilisateur » signalerait la clé de CHAQUE membre : un faux positif par personne,
# et c'est ce qui fait désinstaller un outil. Le contrôle vise la clé qui contourne
# l'IAM parce qu'elle appartient au compte propriétaire de l'organisation.

_scw_user(id, type) := {
	"type": "iam_user",
	"id": id,
	"name": id,
	"attributes": {"user_id": id, "user_type": type},
}

_scw_key(id, owner) := {
	"type": "access_key",
	"id": id,
	"name": id,
	"attributes": {"access_key_id": id, "owner_user_id": owner},
}

test_scaleway_owner_key_is_root if {
	count({f | some f in deny; f.code == _sw_iam_code}) == 1 with input as {"resources": [
		_scw_user("u-owner", "owner"),
		_scw_key("SCW1", "u-owner"),
	]}
}

# LE CONTRE-EXEMPLE QUI COMPTE : un membre a sa propre clé, et ce n'est pas un écart.
test_scaleway_member_key_is_not_root if {
	count({f | some f in deny; f.code == _sw_iam_code}) == 0 with input as {"resources": [
		_scw_user("u-owner", "owner"),
		_scw_user("u-alice", "member"),
		_scw_key("SCW2", "u-alice"),
	]}
}

# Une clé d'APPLICATION ne porte aucun propriétaire humain : rien à signaler.
test_scaleway_application_key_is_not_root if {
	count({f | some f in deny; f.code == _sw_iam_code}) == 0 with input as {"resources": [
		_scw_user("u-owner", "owner"),
		{
			"type": "access_key",
			"id": "SCW3",
			"name": "SCW3",
			"attributes": {"access_key_id": "SCW3"},
		},
	]}
}

# Sans l'utilisateur, la jointure ne trouve rien : on ne conclut pas sur ce qu'on n'a
# pas vu, plutôt que de supposer qu'une clé d'utilisateur est celle du propriétaire.
test_scaleway_without_the_user_the_rule_stays_silent if {
	count({f | some f in deny; f.code == _sw_iam_code}) == 0 with input as {"resources": [_scw_key("SCW4", "u-owner")]}
}

# La casse ne décide pas : `Owner` vaut `owner`.
test_scaleway_owner_type_is_case_insensitive if {
	count({f | some f in deny; f.code == _sw_iam_code}) == 1 with input as {"resources": [
		_scw_user("u-owner", "Owner"),
		_scw_key("SCW5", "u-owner"),
	]}
}

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

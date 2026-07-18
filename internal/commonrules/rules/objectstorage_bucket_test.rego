package pepin.rules

import rego.v1

_b(attrs) := {"resources": [{"provider": "outscale", "type": "object_storage_bucket", "id": "b1", "attributes": attrs}]}

_pub_code := "objectstorage_bucket_public_access"
_ver_code := "objectstorage_bucket_versioning_enabled"
_au := "http://acs.amazonaws.com/groups/global/AllUsers"

# ---- public access (ACL canned / grants AllUsers / policy publique) ----
test_bucket_canned_public_denied if {
	some f in deny with input as _b({"name": "b1", "acl": "public-read"})
	f.code == _pub_code
}

test_bucket_grant_allusers_denied if {
	some f in deny with input as _b({"name": "b1", "acl_grants": [{"grantee": {"uri": _au}, "permission": "READ"}]})
	f.code == _pub_code
}

test_bucket_public_via_acl_denied if {
	some f in deny with input as _b({"name": "b1", "public_via_acl": true})
	f.code == _pub_code
}

test_bucket_policy_public_denied if {
	some f in deny with input as _b({"name": "b1", "policy_public": true})
	f.code == _pub_code
}

test_bucket_private_ok if {
	count({f | some f in deny; f.code == _pub_code}) == 0 with input as _b({"name": "b1", "acl": "private", "public_via_acl": false, "policy_public": false})
}

# ---- versioning ----
test_versioning_disabled_denied if {
	some f in deny with input as _b({"name": "b1", "versioning": "Suspended"})
	f.code == _ver_code
}

test_versioning_empty_denied if {
	count({f | some f in deny; f.code == _ver_code}) == 1 with input as _b({"name": "b1", "versioning": ""})
}

test_versioning_enabled_ok if {
	count({f | some f in deny; f.code == _ver_code}) == 0 with input as _b({"name": "b1", "versioning": "Enabled"})
}

# Pas de clé versioning collectée (ex. source Terraform) → pas de faux positif.
test_versioning_absent_ok if {
	count({f | some f in deny; f.code == _ver_code}) == 0 with input as _b({"name": "b1", "acl": "private"})
}

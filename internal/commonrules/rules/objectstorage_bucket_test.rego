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

# ---- objectstorage_bucket_default_encryption (CLD-CHF-2) ----

# ✗ Chiffrement par défaut désactivé → finding.
test_bucket_default_encryption_off_denied if {
	some f in deny with input as _b({"name": "b1", "default_encryption_enabled": false})
	f.code == "objectstorage_bucket_default_encryption"
}

# ✓ Chiffrement activé → conforme (même sans BYOK : la clé fournisseur suffit à CHF-2).
test_bucket_default_encryption_on_ok if {
	count({f | some f in deny; f.code == "objectstorage_bucket_default_encryption"}) == 0 with input as _b({"name": "b1", "default_encryption_enabled": true})
}

# ✓ Attribut non collecté (403 / endpoint non supporté) → aucun finding : c'est
# l'assessment qui rendra « non évalué », pas la règle qui invente un verdict.
test_bucket_default_encryption_uncollected_silent if {
	count({f | some f in deny; f.code == "objectstorage_bucket_default_encryption"}) == 0 with input as _b({"name": "b1"})
}

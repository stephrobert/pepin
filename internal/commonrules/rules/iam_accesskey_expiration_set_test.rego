package pepin.rules

import rego.v1

_key_code := "iam_accesskey_expiration_set"

_key(attrs) := {"resources": [{"provider": "outscale", "type": "access_key", "id": "ak-1", "attributes": attrs}]}

# ✗ Clé sans expiration → finding.
test_key_no_expiration_denied if {
	some f in deny with input as _key({"access_key_id": "ak-1", "state": "ACTIVE"})
	f.code == _key_code
}

# ✗ expiration_date vide → finding.
test_key_empty_expiration_denied if {
	count({f | some f in deny; f.code == _key_code}) == 1 with input as _key({"access_key_id": "ak-1", "expiration_date": ""})
}

# ✓ Clé avec expiration → aucun finding.
test_key_with_expiration_ok if {
	count({f | some f in deny; f.code == _key_code}) == 0 with input as _key({"access_key_id": "ak-1", "expiration_date": "2026-12-31T23:59:59Z"})
}

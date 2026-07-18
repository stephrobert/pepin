package pepin.rules

import rego.v1

_role(attrs) := {"resources": [{"provider": "exoscale", "type": "iam_role", "id": "r1", "attributes": attrs}]}

# ✗ ni TTL ni expiration → finding IAM-2.
test_role_no_lifetime_bound_denied if {
	some f in deny with input as _role({"name": "admin", "max_session_ttl": 0, "policy_has_expiration": false})
	f.code == "iam_role_key_lifetime_bounded"
}

# ✓ TTL de session borné → pas de finding.
test_role_ttl_bounded_ok if {
	count({f | some f in deny; f.code == "iam_role_key_lifetime_bounded"}) == 0 with input as _role({"name": "admin", "max_session_ttl": 3600, "policy_has_expiration": false})
}

# ✓ expiration dans la policy (TTL non borné) → pas de finding.
test_role_expiration_ok if {
	count({f | some f in deny; f.code == "iam_role_key_lifetime_bounded"}) == 0 with input as _role({"name": "admin", "max_session_ttl": 0, "policy_has_expiration": true})
}

# ✓ Terraform : pas de max_session_ttl, mais expiration présente → pas de finding.
test_role_tf_expiration_ok if {
	count({f | some f in deny; f.code == "iam_role_key_lifetime_bounded"}) == 0 with input as _role({"name": "admin", "policy_has_expiration": true})
}

# ✗ Terraform : pas de max_session_ttl et pas d'expiration → finding.
test_role_tf_no_bound_denied if {
	some f in deny with input as _role({"name": "admin", "source_ip_restricted": true, "policy_has_expiration": false})
	f.code == "iam_role_key_lifetime_bounded"
}

# ✓ type sans aucun attribut de durée de vie → pas de finding (pas de faux positif).
test_role_no_lifetime_attrs_ok if {
	count({f | some f in deny; f.code == "iam_role_key_lifetime_bounded"}) == 0 with input as _role({"name": "admin", "source_ip_restricted": true})
}

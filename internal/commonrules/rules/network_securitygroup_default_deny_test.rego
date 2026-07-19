package pepin.rules

import rego.v1

_sg(attrs) := {"resources": [{"provider": "scaleway", "type": "security_group", "id": "sg-1", "attributes": attrs}]}

# ✗ Politique entrante par défaut « accept » → finding.
test_default_accept_denied if {
	some f in deny with input as _sg({"security_group_id": "sg-1", "inbound_default_policy": "accept"})
	f.code == "network_securitygroup_default_deny"
}

# ✓ Politique entrante par défaut « drop » → aucun finding.
test_default_drop_ok if {
	count({f | some f in deny; f.code == "network_securitygroup_default_deny"}) == 0 with input as _sg({"security_group_id": "sg-1", "inbound_default_policy": "drop"})
}

# ✓ Attribut non collecté (garde de capacité) → pas de faux positif.
test_default_policy_uncollected_ok if {
	count({f | some f in deny; f.code == "network_securitygroup_default_deny"}) == 0 with input as _sg({"security_group_id": "sg-1"})
}

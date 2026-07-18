package pepin.rules

import rego.v1

_rule(attrs) := {"resources": [{"provider": "exoscale", "type": "security_group_rule", "id": "sg1", "attributes": attrs}]}

# ✗ flux entrant sans justification → finding NET-5.
test_flow_undocumented_denied if {
	some f in deny with input as _rule({"direction": "inbound", "port_from": 443, "description": ""})
	f.code == "network_flow_matrix_documented"
}

# ✓ flux entrant justifié → pas de finding.
test_flow_documented_ok if {
	count({f | some f in deny; f.code == "network_flow_matrix_documented"}) == 0 with input as _rule({"direction": "inbound", "port_from": 443, "description": "HTTPS public du site"})
}

# ✓ flux sortant non justifié → pas de finding (on ne vise que l'entrant).
test_flow_egress_ok if {
	count({f | some f in deny; f.code == "network_flow_matrix_documented"}) == 0 with input as _rule({"direction": "outbound", "description": ""})
}

# ✓ provider n'exposant pas la justification (pas de clé description) → pas de finding.
test_flow_no_description_field_ok if {
	count({f | some f in deny; f.code == "network_flow_matrix_documented"}) == 0 with input as _rule({"direction": "inbound", "port_from": 22})
}

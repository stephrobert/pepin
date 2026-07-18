package pepin.rules

import rego.v1

_vm_code := "compute_instance_public_ip_with_open_securitygroup"

_open_rule := {
	"provider": "outscale", "type": "security_group_rule", "id": "sg-1-in-0",
	"attributes": {"direction": "inbound", "action": "accept", "security_group_id": "sg-1", "cidrs": ["0.0.0.0/0"], "protocol": "all"},
}

_vm(attrs) := {"provider": "outscale", "type": "compute_instance", "id": "i-1", "attributes": attrs}

# ✗ VM avec IP publique + SG ouvert sur Internet → finding critical.
test_public_vm_open_sg_denied if {
	input_doc := {"resources": [_open_rule, _vm({"vm_id": "i-1", "public_ip": "203.0.113.1", "security_group_ids": ["sg-1"]})]}
	some f in deny with input as input_doc
	f.code == _vm_code
	f.severity == "critical"
}

# ✓ VM avec IP publique mais SG NON ouvert → aucun finding.
test_public_vm_closed_sg_ok if {
	closed_rule := {"provider": "outscale", "type": "security_group_rule", "id": "sg-2-in-0", "attributes": {"direction": "inbound", "action": "accept", "security_group_id": "sg-2", "cidrs": ["10.0.0.0/8"], "protocol": "tcp"}}
	input_doc := {"resources": [closed_rule, _vm({"vm_id": "i-1", "public_ip": "203.0.113.1", "security_group_ids": ["sg-2"]})]}
	count({f | some f in deny; f.code == _vm_code}) == 0 with input as input_doc
}

# ✓ VM sans IP publique (SG ouvert) → aucun finding.
test_private_vm_open_sg_ok if {
	input_doc := {"resources": [_open_rule, _vm({"vm_id": "i-1", "security_group_ids": ["sg-1"]})]}
	count({f | some f in deny; f.code == _vm_code}) == 0 with input as input_doc
}

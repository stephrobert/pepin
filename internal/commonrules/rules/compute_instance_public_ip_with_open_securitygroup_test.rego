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

# _sg — règle entrante depuis Internet, sur une plage de ports donnée.
_sg_range(id, from, to) := {
	"provider": "outscale", "type": "security_group_rule", "id": sprintf("%s-in", [id]),
	"attributes": {
		"direction": "inbound", "action": "accept", "security_group_id": id,
		"cidrs": ["0.0.0.0/0"], "protocol": "tcp", "port_from": from, "port_to": to,
	},
}

_public_vm(sg) := _vm({"vm_id": "i-1", "public_ip": "203.0.113.1", "security_group_ids": [sg]})

# ✓ LE CONTRE-EXEMPLE. Une VM publique servant 443/tcp depuis Internet est un serveur
# web, pas un incident. La signaler en `critical` était le finding le plus coûteux du
# dépôt : celui qu'une personne rencontre en cinq minutes, et qui fait douter de tous
# les autres. Ce test est ce qui empêche son retour.
test_public_vm_serving_https_only_is_not_a_deviation if {
	doc := {"resources": [_sg_range("sg-web", 443, 443), _public_vm("sg-web")]}
	count({f | some f in deny; f.code == _vm_code}) == 0 with input as doc
}

# ✓ Le même en HTTP, qui redirige en pratique vers HTTPS → pas davantage un écart.
test_public_vm_serving_http_only_is_not_a_deviation if {
	doc := {"resources": [_sg_range("sg-web", 80, 80), _public_vm("sg-web")]}
	count({f | some f in deny; f.code == _vm_code}) == 0 with input as doc
}

# ✗ SSH ouvert sur Internet → high, et le message NOMME le port.
test_public_vm_ssh_open_is_high if {
	doc := {"resources": [_sg_range("sg-adm", 22, 22), _public_vm("sg-adm")]}
	some f in deny with input as doc
	f.code == _vm_code
	f.severity == "high"
	contains(f.message, "22")
}

# ✗ RDP ouvert sur Internet → high également.
test_public_vm_rdp_open_is_high if {
	doc := {"resources": [_sg_range("sg-adm", 3389, 3389), _public_vm("sg-adm")]}
	some f in deny with input as doc
	f.code == _vm_code
	f.severity == "high"
}

# ✗ Une plage large qui ENGLOBE un port sensible reste un écart : la largeur ne dilue
# pas l'exposition, elle l'aggrave.
test_public_vm_wide_range_covering_ssh_is_high if {
	doc := {"resources": [_sg_range("sg-large", 1, 1024), _public_vm("sg-large")]}
	some f in deny with input as doc
	f.code == _vm_code
	f.severity == "high"
}

# ✗ Sentinelle « tous les ports » (-1/-1, encodage Outscale OAPI) → critical.
test_public_vm_all_ports_sentinel_is_critical if {
	doc := {"resources": [_sg_range("sg-any", -1, -1), _public_vm("sg-any")]}
	some f in deny with input as doc
	f.code == _vm_code
	f.severity == "critical"
}

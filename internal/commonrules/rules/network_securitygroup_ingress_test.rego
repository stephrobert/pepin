package pepin.rules

import rego.v1

_sgr(attrs) := {"resources": [{"provider": "outscale", "type": "security_group_rule", "id": "r1", "attributes": attrs}]}

_ssh := "network_securitygroup_allow_ingress_from_internet_to_tcp_port_22"
_rdp := "network_securitygroup_allow_ingress_from_internet_to_tcp_port_3389"
_hr := "network_securitygroup_allow_ingress_from_internet_to_high_risk_tcp_ports"
_all := "network_securitygroup_allow_ingress_from_internet_to_all_ports"
_egr := "network_securitygroup_unrestricted_egress"

# ✗ SSH ouvert (tcp 22, 0.0.0.0/0).
test_ssh_denied if {
	some f in deny with input as _sgr({"direction": "inbound", "action": "accept", "protocol": "tcp", "cidrs": ["0.0.0.0/0"], "port_from": 22, "port_to": 22, "security_group_id": "sg-1"})
	f.code == _ssh
}

# ✓ SSH source restreinte.
test_ssh_restricted_ok if {
	count({f | some f in deny; f.code == _ssh}) == 0 with input as _sgr({"direction": "inbound", "action": "accept", "protocol": "tcp", "cidrs": ["10.0.0.0/8"], "port_from": 22, "port_to": 22})
}

# ✓ Règle drop sur 0.0.0.0/0 → aucun finding (action drop).
test_ssh_drop_ok if {
	count({f | some f in deny; f.code == _ssh}) == 0 with input as _sgr({"direction": "inbound", "action": "drop", "protocol": "tcp", "cidrs": ["0.0.0.0/0"], "port_from": 22, "port_to": 22})
}

# ✗ RDP ouvert.
test_rdp_denied if {
	some f in deny with input as _sgr({"direction": "inbound", "action": "accept", "protocol": "tcp", "cidrs": ["::/0"], "port_from": 3389, "port_to": 3389})
	f.code == _rdp
}

# ✗ Port sensible (PostgreSQL 5432) ouvert → high_risk (et pas SSH/RDP).
test_high_risk_denied if {
	some f in deny with input as _sgr({"direction": "inbound", "action": "accept", "protocol": "tcp", "cidrs": ["0.0.0.0/0"], "port_from": 5432, "port_to": 5432})
	f.code == _hr
}

# ✗ Port nouvellement listé (Kubernetes API 6443) ouvert → high_risk.
test_high_risk_k8s_api_denied if {
	some f in deny with input as _sgr({"direction": "inbound", "action": "accept", "protocol": "tcp", "cidrs": ["0.0.0.0/0"], "port_from": 6443, "port_to": 6443})
	f.code == _hr
}

# ✗ Daemon Docker exposé (2375) ouvert → high_risk.
test_high_risk_docker_denied if {
	some f in deny with input as _sgr({"direction": "inbound", "action": "accept", "protocol": "tcp", "cidrs": ["0.0.0.0/0"], "port_from": 2375, "port_to": 2375})
	f.code == _hr
}

# ✗ any/any entrant depuis Internet → all_ports (critical).
test_all_ports_denied if {
	some f in deny with input as _sgr({"direction": "inbound", "action": "accept", "protocol": "all", "cidrs": ["0.0.0.0/0"], "port_from": 0, "port_to": 0})
	f.code == _all
	f.severity == "critical"
}

# ✗ egress tout-trafic vers Internet.
test_egress_denied if {
	some f in deny with input as _sgr({"direction": "outbound", "action": "accept", "protocol": "all", "cidrs": ["0.0.0.0/0"]})
	f.code == _egr
}

# ✓ provider tiré de la ressource (Scaleway).
test_provider_from_resource if {
	doc := {"resources": [{"provider": "scaleway", "type": "security_group_rule", "id": "r1", "attributes": {"direction": "inbound", "action": "accept", "protocol": "tcp", "cidrs": ["0.0.0.0/0"], "port_from": 22, "port_to": 22, "security_group_id": "sg-x"}}]}
	some f in deny with input as doc
	f.code == _ssh
	f.labels.provider == "scaleway"
}

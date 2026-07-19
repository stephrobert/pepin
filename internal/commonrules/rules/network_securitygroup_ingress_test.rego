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

# ✗ FN-1 : règle TCP SANS bornes de port (le moteur live projette port_from/to en "") = tous
# ports ouverts → doit être flaguée (auparavant : faux négatif silencieux).
test_tcp_no_port_bounds_denied if {
	some f in deny with input as _sgr({"direction": "inbound", "action": "accept", "protocol": "tcp", "cidrs": ["0.0.0.0/0"], "port_from": "", "port_to": ""})
	f.code == _hr
}

# ✗ FN-9 : /0 IPv6 écrit « ::0/0 » reconnu public (parsing du préfixe, pas littéral).
test_ipv6_zero_public_denied if {
	some f in deny with input as _sgr({"direction": "inbound", "action": "accept", "protocol": "tcp", "cidrs": ["::0/0"], "port_from": 22, "port_to": 22})
	f.code == _ssh
}

# ✗ FN-9 : contournement « moitié d'Internet » 0.0.0.0/1 reconnu public.
test_ipv4_half_internet_denied if {
	some f in deny with input as _sgr({"direction": "inbound", "action": "accept", "protocol": "tcp", "cidrs": ["0.0.0.0/1"], "port_from": 22, "port_to": 22})
	f.code == _ssh
}

# ✗ IPv6 « moitié d'Internet » ::/1 reconnu public (généralisation IPv6 du /1).
test_ipv6_half_internet_denied if {
	some f in deny with input as _sgr({"direction": "inbound", "action": "accept", "protocol": "tcp", "cidrs": ["::/1"], "port_from": 22, "port_to": 22})
	f.code == _ssh
}

# ✗ IPv6 mappé-IPv4 couvrant tout l'IPv4 (::ffff:0.0.0.0/96) → public (contournement).
test_ipv4_mapped_ipv6_denied if {
	some f in deny with input as _sgr({"direction": "inbound", "action": "accept", "protocol": "tcp", "cidrs": ["::ffff:0.0.0.0/96"], "port_from": 22, "port_to": 22})
	f.code == _ssh
}

# ✗ Littéral « toute origine » sans masque + espaces parasites → public.
test_bare_any_with_spaces_denied if {
	some f in deny with input as _sgr({"direction": "inbound", "action": "accept", "protocol": "tcp", "cidrs": [" 0.0.0.0 "], "port_from": 22, "port_to": 22})
	f.code == _ssh
}

# ✗ Sentinelle -1/-1 (Outscale OAPI = tous ports) ouverte à Internet → all/any TCP high_risk.
test_port_sentinel_minus_one_denied if {
	some f in deny with input as _sgr({"direction": "inbound", "action": "accept", "protocol": "tcp", "cidrs": ["0.0.0.0/0"], "port_from": -1, "port_to": -1})
	f.code == _hr
}

# ✓ action « reject » (ni accept ni drop) NON présumée acceptante → aucun finding.
test_action_reject_not_accepting_ok if {
	count({f | some f in deny; f.code == _ssh}) == 0 with input as _sgr({"direction": "inbound", "action": "reject", "protocol": "tcp", "cidrs": ["0.0.0.0/0"], "port_from": 22, "port_to": 22})
}

# ✗ FN-2 : NFS/UDP (2049) ouvert à Internet → high_risk UDP.
test_udp_nfs_denied if {
	some f in deny with input as _sgr({"direction": "inbound", "action": "accept", "protocol": "udp", "cidrs": ["0.0.0.0/0"], "port_from": 2049, "port_to": 2049})
	f.code == "network_securitygroup_allow_ingress_from_internet_to_high_risk_udp_ports"
}

# ✓ Source restreinte en /24 : non publique, aucun finding.
test_restricted_24_ok if {
	count({f | some f in deny; f.code == _ssh}) == 0 with input as _sgr({"direction": "inbound", "action": "accept", "protocol": "tcp", "cidrs": ["203.0.113.0/24"], "port_from": 22, "port_to": 22})
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

# Règles d'exposition des security groups — COMMUNES à tous les providers.
#   Lisent le type normalisé `security_group_rule` (schéma commun produit par les
#   collecteurs/mappers de chaque provider) : direction, action, protocol
#   (tcp|udp|icmp|all), port_from, port_to, cidrs[], security_group_id.
#   `labels.provider` est tiré de la ressource.
# SCSL : CLD-NET-1 (ports d'admin/sensibles), CLD-NET-2 (any/any).
package pepin.rules

import rego.v1

# SSH (22) ouvert à Internet → CLD-NET-1.
deny contains f if {
	some r in resources_of_type("security_group_rule")
	sg_inbound_from_internet(r.attributes)
	proto_covers(r.attributes, "tcp")
	covers_port(r.attributes, 22)
	f := _sg_finding(r, "network_securitygroup_allow_ingress_from_internet_to_tcp_port_22", "high", "SSH (port 22)")
}

# RDP (3389) ouvert à Internet → CLD-NET-1.
deny contains f if {
	some r in resources_of_type("security_group_rule")
	sg_inbound_from_internet(r.attributes)
	proto_covers(r.attributes, "tcp")
	covers_port(r.attributes, 3389)
	f := _sg_finding(r, "network_securitygroup_allow_ingress_from_internet_to_tcp_port_3389", "high", "RDP (port 3389)")
}

# Ports sensibles TCP (BD, annuaire, orchestration…) ouverts à Internet → CLD-NET-1.
# AGRÉGÉ : une règle « tous ports » ouverte ne produit qu'UN finding listant les ports, pas ~38.
deny contains f if {
	some r in resources_of_type("security_group_rule")
	sg_inbound_from_internet(r.attributes)
	proto_covers(r.attributes, "tcp")
	ports := sort([p | some p in (sensitive_ports - {22, 3389}); covers_port(r.attributes, p)])
	count(ports) > 0
	f := _sg_finding(r, "network_securitygroup_allow_ingress_from_internet_to_high_risk_tcp_ports", "high", sprintf("ports sensibles TCP %v", [ports]))
}

# Ports sensibles UDP (amplification DDoS, services non authentifiés) ouverts à Internet → CLD-NET-1.
deny contains f if {
	some r in resources_of_type("security_group_rule")
	sg_inbound_from_internet(r.attributes)
	proto_covers(r.attributes, "udp")
	ports := sort([p | some p in sensitive_udp_ports; covers_port(r.attributes, p)])
	count(ports) > 0
	f := _sg_finding(r, "network_securitygroup_allow_ingress_from_internet_to_high_risk_udp_ports", "high", sprintf("ports sensibles UDP %v", [ports]))
}

# Tout le trafic entrant depuis Internet (any/any) → CLD-NET-2.
deny contains f if {
	some r in resources_of_type("security_group_rule")
	sg_inbound_from_internet(r.attributes)
	lower(object.get(r.attributes, "protocol", "")) == "all"
	f := _sg_finding(r, "network_securitygroup_allow_ingress_from_internet_to_all_ports", "critical", "tout le trafic (any/any)")
}

# Sortie tout-trafic non restreinte vers Internet → CLD-NET-4.
deny contains f if {
	some r in resources_of_type("security_group_rule")
	lower(object.get(r.attributes, "direction", "")) == "outbound"
	sg_accepting(r.attributes)
	lower(object.get(r.attributes, "protocol", "")) == "all"
	some cidr in cidr_list(object.get(r.attributes, "cidrs", []))
	is_public_cidr(cidr)
	f := _sg_finding(r, "network_securitygroup_unrestricted_egress", "medium", "tout le trafic sortant")
}

_sg_finding(r, code, sev, what) := {
	"code": code,
	"severity": sev,
	"subject": object.get(r.attributes, "security_group_id", r.id),
	"message": sprintf("Security group « %s » : %s accepté depuis/vers Internet.", [object.get(r.attributes, "security_group_id", r.id), what]),
	"remediation": "Restreindre la règle à des sources/destinations et ports légitimes (CIDR d'administration, bastion, VPN) ; ne jamais exposer un service sensible à 0.0.0.0/0.",
	"labels": {"provider": provider_of(r), "category": "security"},
}

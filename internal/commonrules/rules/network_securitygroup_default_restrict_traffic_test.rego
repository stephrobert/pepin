package pepin.rules

import rego.v1

_default_sgr(attrs) := {"resources": [{"provider": "outscale", "type": "security_group_rule", "id": "sg-1", "attributes": attrs}]}

# ✗ Le SG « default » porte une règle entrante : elle s'applique d'office aux ressources
# créées sans groupe explicite, donc sans décision de l'exploitant.
test_default_sg_inbound_denied if {
	some f in deny with input as _default_sgr({"security_group_id": "sg-1", "security_group_name": "default", "direction": "inbound", "protocol": "tcp", "cidrs": ["10.0.0.0/8"]})
	f.code == "network_securitygroup_default_restrict_traffic"
}

# ✓ Un SG nommé autrement n'est pas concerné : il est attaché explicitement.
test_named_sg_inbound_ok if {
	count({f | some f in deny; f.code == "network_securitygroup_default_restrict_traffic"}) == 0 with input as _default_sgr({"security_group_id": "sg-2", "security_group_name": "web", "direction": "inbound", "protocol": "tcp", "cidrs": ["10.0.0.0/8"]})
}

# ✓ Une règle SORTANTE du SG default n'est pas visée ici (l'egress a son propre contrôle) :
# sinon chaque SG produirait un doublon, la règle sortante par défaut étant systématique.
test_default_sg_outbound_ok if {
	count({f | some f in deny; f.code == "network_securitygroup_default_restrict_traffic"}) == 0 with input as _default_sgr({"security_group_id": "sg-1", "security_group_name": "default", "direction": "outbound", "protocol": "all", "cidrs": ["0.0.0.0/0"]})
}

# ✓ Nom non collecté (provider ne l'exposant pas) → garde de capacité, aucun verdict inventé.
test_default_sg_name_uncollected_silent if {
	count({f | some f in deny; f.code == "network_securitygroup_default_restrict_traffic"}) == 0 with input as _default_sgr({"security_group_id": "sg-1", "direction": "inbound", "protocol": "tcp"})
}

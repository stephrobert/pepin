# Branches ajoutées pour corriger un faux négatif, et jamais exercées.
#
# Une branche non testée est une branche dont on ignore si elle marche. Les trois
# ci-dessous sont d'autant plus sensibles qu'elles existent précisément pour
# rattraper des cas que le chemin principal ratait : si elles sont muettes, le
# faux négatif qu'elles corrigeaient est simplement revenu, en silence.
package pepin.rules

import rego.v1

# --- 1. VM joignable par l'IP publique d'une NIC SECONDAIRE ------------------
# `_vm_has_public_ip` a une seconde branche (nic_public_ips) ajoutée pour les VMs
# multi-cartes, que `public_ip` seul laissait passer. Contrat : NicLight.LinkPublicIp.

_multi_nic_vm(nics) := {"resources": [
	{
		"provider": "outscale", "type": "compute_instance", "id": "i-1",
		"attributes": {"name": "vm", "nic_public_ips": nics, "security_group_ids": ["sg-1"]},
	},
	{
		"provider": "outscale", "type": "security_group_rule", "id": "sg-1",
		"attributes": {
			"security_group_id": "sg-1", "direction": "inbound", "action": "accept",
			"protocol": "tcp", "port_from": 22, "port_to": 22, "cidrs": ["0.0.0.0/0"],
		},
	},
]}

# ✗ IP publique portée par une NIC secondaire, SG ouvert → la VM doit être vue.
test_vm_public_via_secondary_nic_denied if {
	some f in deny with input as _multi_nic_vm(["203.0.113.7"])
	f.code == "compute_instance_public_ip_with_open_securitygroup"
}

# ✓ Liste de NIC vide → aucune IP publique, donc pas de finding sur ce contrôle.
test_vm_no_nic_public_ip_ok if {
	count({f |
		some f in deny
		f.code == "compute_instance_public_ip_with_open_securitygroup"
	}) == 0
		with input as _multi_nic_vm([])
}

# ✓ Entrées nulles/vides : la branche les écarte explicitement, pas de faux positif.
test_vm_null_nic_public_ip_ok if {
	count({f |
		some f in deny
		f.code == "compute_instance_public_ip_with_open_securitygroup"
	}) == 0
		with input as _multi_nic_vm([null, ""])
}

# --- 2. Ports transmis en CHAÎNE (plans Terraform) --------------------------
# `_port_bound` accepte une chaîne purement numérique. Sans cette branche, un plan
# qui rend `port_from: "22"` sortait du champ de toutes les règles de port.

_sg_rule(attrs) := {"resources": [{
	"provider": "scaleway", "type": "security_group_rule", "id": "r1",
	"attributes": object.union(
		{
			"security_group_id": "sg", "direction": "inbound", "action": "accept",
			"protocol": "tcp", "cidrs": ["0.0.0.0/0"],
		},
		attrs,
	),
}]}

# ✗ Port SSH en chaîne → doit se déclencher comme la forme numérique.
test_ssh_port_as_string_denied if {
	some f in deny with input as _sg_rule({"port_from": "22", "port_to": "22"})
	f.code == "network_securitygroup_allow_ingress_from_internet_to_tcp_port_22"
}

# ✗ Port NON numérique : aucune borne exploitable. covers_port retombe alors sur
# sa branche « toutes les bornes indéfinies ⇒ tous les ports couverts », donc la
# règle SE DÉCLENCHE. C'est délibérément fail-closed : une règle de security group
# dont les bornes sont illisibles ouvre potentiellement tout, et un outil de posture
# doit alerter plutôt que se taire. Ce test verrouille ce choix, qui serait sinon
# facile à « corriger » en fail-open lors d'un refactor.
test_non_numeric_port_is_fail_closed if {
	some f in deny with input as _sg_rule({"port_from": "abc", "port_to": "abc"})
	f.code == "network_securitygroup_allow_ingress_from_internet_to_tcp_port_22"
}

# --- 3. `port_to` ABSENT : forme port unique du collecteur live -------------
# `_to_bound` retombe sur la borne basse quand `port_to` manque. C'est exactement
# ce que produit le collecteur live Scaleway (port_to := dest_port_to || dest_port_from).

# ✗ Port unique 22, sans port_to → SSH ouvert, doit être détecté.
test_single_port_without_port_to_denied if {
	some f in deny with input as _sg_rule({"port_from": 22})
	f.code == "network_securitygroup_allow_ingress_from_internet_to_tcp_port_22"
}

# ✓ Port unique 443, sans port_to → hors des ports sensibles, pas de finding SSH.
test_single_port_without_port_to_other_port_ok if {
	count({f |
		some f in deny
		f.code == "network_securitygroup_allow_ingress_from_internet_to_tcp_port_22"
	}) == 0
		with input as _sg_rule({"port_from": 443})
}

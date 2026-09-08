# compute_instance_public_ip_with_open_securitygroup
#   VM avec une IP publique ET un security group acceptant du trafic entrant
#   depuis Internet sur un port qui le rend dangereux.
#
#   CE QUI EST VÉRIFIÉ : ce qui est RÉELLEMENT exposé, pas la seule existence d'un
#   ingress Internet. Une VM publique servant 443/tcp depuis 0.0.0.0/0 est un
#   serveur web, pas un incident — et la signaler en `critical` était le finding le
#   plus coûteux du dépôt, celui qu'une personne rencontre dans ses cinq premières
#   minutes et qui fait douter de tous les autres.
#
#   CE QUE LE CONTRÔLE NE PROUVE PAS : qu'un service exposé sur un port sensible est
#   effectivement vulnérable, ni qu'un port anodin ne l'est pas. Il mesure la
#   surface, pas l'exploitabilité.
#
# Origine : osc-policy OSC-VM-014. SCSL : CLD-NET-3.
# Contrat : type normalisé agnostique `compute_instance` (attributs natifs
#   osc-sdk-go Vm). Champs : vm_id, public_ip (string), security_group_ids
#   ([]string, DÉRIVÉ depuis Vm.SecurityGroups[].SecurityGroupId). Corrélé aux
#   `security_group_rule` entrants ouverts sur Internet.
package pepin.rules

import rego.v1

# _sg_all_ports — la règle n'a AUCUNE borne de port, ou porte la sentinelle -1 que
# plusieurs API emploient pour « toute la plage ». Tout est exposé, y compris ce que
# personne n'a voulu exposer.
_sg_all_ports(attrs) if {
	not _port_bound(attrs, "port_from")
	not _port_bound(attrs, "port_to")
}

_sg_all_ports(attrs) if _port_bound(attrs, "port_from") == -1

_sg_all_ports(attrs) if _port_bound(attrs, "port_to") == -1

# _sg_ids_any_port — security groups ouverts sur Internet SANS restriction de port.
_sg_ids_any_port contains sg_id if {
	some r in resources_of_type("security_group_rule")
	sg_inbound_from_internet(r.attributes)
	_sg_all_ports(r.attributes)
	sg_id := object.get(r.attributes, "security_group_id", "")
	sg_id != ""
}

# _sg_sensitive[sg_id] — ports sensibles qu'un security group expose à Internet.
# `sensitive_ports` est la table partagée (lib.rego), sourcée sur les benchmarks CIS.
_sg_sensitive[sg_id] contains port if {
	some r in resources_of_type("security_group_rule")
	sg_inbound_from_internet(r.attributes)
	not _sg_all_ports(r.attributes)
	some port in sensitive_ports
	covers_port(r.attributes, port)
	sg_id := object.get(r.attributes, "security_group_id", "")
	sg_id != ""
}

# ── critical : tous les ports ouverts sur Internet ────────────────────────────────
deny contains f if {
	some vm in resources_of_type("compute_instance")
	_vm_has_public_ip(vm)
	some sg_id in object.get(vm.attributes, "security_group_ids", [])
	sg_id in _sg_ids_any_port
	id := object.get(vm.attributes, "vm_id", vm.id)
	f := {
		"code": "compute_instance_public_ip_with_open_securitygroup",
		"severity": "critical",
		"subject": id,
		"message": sprintf("VM « %s » exposée publiquement : le security group %s ouvre TOUS les ports sur Internet.", [id, sg_id]),
		"remediation": "Restreindre la règle entrante aux seuls ports servis, ou détacher l'IP publique et passer par un LBU / NAT.",
		"labels": {
			"provider": provider_of(vm),
			"category": "security",
			"message_en": sprintf("VM \"%s\" publicly exposed: security group %s opens EVERY port to the internet.", [id, sg_id]),
			"remediation_en": "Restrict the inbound rule to the ports actually served, or detach the public IP and go through an LBU / NAT.",
		},
	}
}

# ── high : un port sensible ouvert sur Internet ───────────────────────────────────
deny contains f if {
	some vm in resources_of_type("compute_instance")
	_vm_has_public_ip(vm)
	some sg_id in object.get(vm.attributes, "security_group_ids", [])
	not sg_id in _sg_ids_any_port
	ports := sort([p | some p in _sg_sensitive[sg_id]])
	count(ports) > 0
	id := object.get(vm.attributes, "vm_id", vm.id)
	f := {
		"code": "compute_instance_public_ip_with_open_securitygroup",
		"severity": "high",
		"subject": id,
		"message": sprintf("VM « %s » exposée publiquement : le security group %s ouvre sur Internet le(s) port(s) sensible(s) %v.", [id, sg_id, ports]),
		"remediation": "Restreindre ces ports aux réseaux d'administration, ou passer par un bastion / VPN ; ne laisser ouverts que les ports du service rendu.",
		"labels": {
			"provider": provider_of(vm),
			"category": "security",
			"message_en": sprintf("VM \"%s\" publicly exposed: security group %s opens sensitive port(s) %v to the internet.", [id, sg_id, ports]),
			"remediation_en": "Restrict those ports to administration networks, or go through a bastion / VPN; leave open only the ports the service actually needs.",
		},
	}
}

# Une VM est joignable si son IP primaire OU l'IP publique d'une NIC secondaire existe :
# ne regarder que `public_ip` laissait passer les VMs multi-cartes (contrat NicLight.LinkPublicIp).
_vm_has_public_ip(vm) if object.get(vm.attributes, "public_ip", "") != ""

_vm_has_public_ip(vm) if {
	some ip in object.get(vm.attributes, "nic_public_ips", [])
	ip != null
	ip != ""
}

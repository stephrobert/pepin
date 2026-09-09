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
#   L'EXPOSITION EST UNE PROPRIÉTÉ DE LA CARTE, PAS DE LA MACHINE. Mesuré sur un
#   tenant réel : une VM dont la carte primaire est privée avec un groupe fermé et
#   dont la carte secondaire porte l'IP publique ET un groupe ouvert sur 22 ne
#   produisait AUCUN finding — la règle confrontait l'union des IP aux groupes de la
#   seule carte primaire. Un port d'administration joignable depuis Internet, et un
#   rapport muet.
#
#   L'aplatissement est fautif dans les DEUX sens, et c'est ce qui interdit de le
#   corriger en réunissant tout : une carte publique au groupe fermé plus une carte
#   privée au groupe ouvert donneraient « publique et ouverte », un écart que la
#   machine ne porte pas. Ce qui compte est l'APPARIEMENT (ip, groupes) par carte.
#
# Origine : osc-policy OSC-VM-014. SCSL : CLD-NET-3.
# Contrat : types normalisés agnostiques `compute_instance` et `network_interface`.
#   `compute_instance` : vm_id, public_ip, security_group_ids (carte primaire).
#   `network_interface` : nic_id, vm_id, public_ip, security_group_ids (CETTE carte).
#   Corrélés aux `security_group_rule` entrants ouverts sur Internet.
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
	some couple in _exposed
	vm := couple.vm
	sg_id := couple.sg
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
			"confidence": "confirmed",
			"message_en": sprintf("VM \"%s\" publicly exposed: security group %s opens EVERY port to the internet.", [id, sg_id]),
			"remediation_en": "Restrict the inbound rule to the ports actually served, or detach the public IP and go through an LBU / NAT.",
		},
	}
}

# ── high : un port sensible ouvert sur Internet ───────────────────────────────────
deny contains f if {
	some couple in _exposed
	vm := couple.vm
	sg_id := couple.sg
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
			"confidence": "confirmed",
			"message_en": sprintf("VM \"%s\" publicly exposed: security group %s opens sensitive port(s) %v to the internet.", [id, sg_id, ports]),
			"remediation_en": "Restrict those ports to administration networks, or go through a bastion / VPN; leave open only the ports the service actually needs.",
		},
	}
}

# _exposed — les couples (machine, groupe) réellement exposés, appariés PAR CARTE.
#
# Quand les cartes sont collectées, elles font foi : chaque carte joignable apporte SES
# groupes, et une carte privée n'en apporte aucun. C'est le seul appariement fidèle.
_exposed contains {"vm": vm, "sg": sg_id} if {
	some nic in resources_of_type("network_interface")
	_has_public_ip(nic)
	some vm in resources_of_type("compute_instance")
	object.get(vm.attributes, "vm_id", vm.id) == object.get(nic.attributes, "vm_id", "")
	some sg_id in object.get(nic.attributes, "security_group_ids", [])
}

# REPLI, pour toute source qui ne collecte pas les cartes — un plan Terraform, un
# provider dont l'API ne les expose pas. La machine y est jugée sur ses propres
# attributs, exactement comme avant. Un repli qui se tairait ferait DISPARAÎTRE des
# écarts que le dépôt détecte aujourd'hui, ce qui serait pire que le défaut corrigé.
_exposed contains {"vm": vm, "sg": sg_id} if {
	some vm in resources_of_type("compute_instance")
	not _has_nics(vm)
	_vm_has_public_ip(vm)
	some sg_id in object.get(vm.attributes, "security_group_ids", [])
}

# _has_nics — cette machine a-t-elle des cartes collectées ? La jointure porte sur
# `vm_id` des deux côtés, tous deux issus de la MÊME réponse d'API : une carte sans
# `vm_id` ne rattache rien et ne fait donc pas taire le repli.
_has_nics(vm) if {
	some nic in resources_of_type("network_interface")
	object.get(vm.attributes, "vm_id", vm.id) == object.get(nic.attributes, "vm_id", "")
}

_has_public_ip(r) if object.get(r.attributes, "public_ip", "") != ""

# Une VM est joignable si son IP primaire OU l'IP publique d'une NIC secondaire existe :
# ne regarder que `public_ip` laissait passer les VMs multi-cartes (contrat NicLight.LinkPublicIp).
_vm_has_public_ip(vm) if _has_public_ip(vm)

_vm_has_public_ip(vm) if {
	some ip in object.get(vm.attributes, "nic_public_ips", [])
	ip != null
	ip != ""
}

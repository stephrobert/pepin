# compute_instance_public_ip_with_open_securitygroup
#   VM avec une IP publique ET un security group acceptant du trafic entrant
#   depuis Internet (0.0.0.0/0 ou ::/0) — vecteur d'intrusion direct.
# Origine : osc-policy OSC-VM-014. SCSL : CLD-NET-3.
# Contrat : type normalisé agnostique `compute_instance` (attributs natifs
#   osc-sdk-go Vm). Champs : vm_id, public_ip (string), security_group_ids
#   ([]string, DÉRIVÉ depuis Vm.SecurityGroups[].SecurityGroupId). Corrélé aux
#   `security_group_rule` entrants ouverts sur Internet.
package pepin.rules

import rego.v1

# _exposed_sg_ids — security groups avec ≥ 1 règle entrante ouverte sur Internet
# (schéma SG commun : direction/action/cidrs).
_exposed_sg_ids contains sg_id if {
	some r in resources_of_type("security_group_rule")
	sg_inbound_from_internet(r.attributes)
	sg_id := object.get(r.attributes, "security_group_id", "")
	sg_id != ""
}

deny contains f if {
	some vm in resources_of_type("compute_instance")
	_vm_has_public_ip(vm)
	some sg_id in object.get(vm.attributes, "security_group_ids", [])
	sg_id in _exposed_sg_ids
	id := object.get(vm.attributes, "vm_id", vm.id)
	f := {
		"code": "compute_instance_public_ip_with_open_securitygroup",
		"severity": "critical",
		"subject": id,
		"message": sprintf("VM « %s » exposée publiquement via le security group %s (règle entrante ouverte sur Internet).", [id, sg_id]),
		"remediation": "Retirer la règle entrante 0.0.0.0/0 du security group, ou détacher l'IP publique et passer par un LBU / NAT.",
		"labels": {
			"provider": provider_of(vm),
			"category": "security",
			"message_en": sprintf("VM \"%s\" publicly exposed through security group %s (inbound rule open to the internet).", [id, sg_id]),
			"remediation_en": "Remove the 0.0.0.0/0 inbound rule from the security group, or detach the public IP and go through an LBU / NAT.",
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

# compute_instance_has_security_group
#   VM sans aucun groupe de sécurité attaché — aucun filtrage réseau ne s'applique.
# Origine : osc-policy OSC-VM-001. SCSL : CLD-CMP-1.
# Contrat : type normalisé agnostique `compute_instance` ; attribut
#   security_group_ids ([]string, DÉRIVÉ de osc-sdk-go Vm.SecurityGroups[]).
package pepin.rules

import rego.v1

deny contains f if {
	some vm in resources_of_type("compute_instance")
	count(object.get(vm.attributes, "security_group_ids", [])) == 0
	id := object.get(vm.attributes, "vm_id", vm.id)
	f := {
		"code": "compute_instance_has_security_group",
		"severity": "critical",
		"subject": id,
		"message": sprintf("VM « %s » sans groupe de sécurité : aucun filtrage réseau ne s'applique.", [id]),
		"remediation": "Attacher un groupe de sécurité restrictif (refus par défaut) à la VM.",
		"labels": {"provider": provider_of(vm), "category": "security"},
	}
}

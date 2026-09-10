# compute_instance_has_security_group
#   VM sans aucun groupe de sécurité attaché — aucun filtrage réseau ne s'applique.
# Origine : osc-policy OSC-VM-001. SCSL : CLD-CMP-1.
# Contrat : type normalisé agnostique `compute_instance` ; attribut
#   security_group_ids ([]string, DÉRIVÉ de osc-sdk-go Vm.SecurityGroups[]).
package pepin.rules

import rego.v1

deny contains f if {
	some vm in resources_of_type("compute_instance")

	# Garde de capacité : l'attribut a été COLLECTÉ. Sur un plan Terraform, security_group_ids
	# référence souvent un SG créé dans le même plan (« known after apply ») → absent des
	# planned_values ; sans cette garde, une VM pourtant rattachée serait faussement « sans SG ».
	"security_group_ids" in object.keys(vm.attributes)
	count(object.get(vm.attributes, "security_group_ids", [])) == 0

	# ...et la machine a bien quelque chose à filtrer. Chez un fournisseur dont les
	# groupes de sécurité filtrent l'INTERFACE PUBLIQUE, une instance qui n'en a pas
	# se voit attacher une liste vide PAR CONSTRUCTION : l'API le fait, l'exploitant
	# ne l'a pas choisi, et aucune remédiation ne peut y changer quoi que ce soit.
	#
	# Crier dessus était le pire des faux positifs — systématique, reproductible, et
	# impossible à faire disparaître. Dix instances privées, dix findings `critical`
	# que personne ne peut corriger, et l'outil se fait ignorer.
	#
	# Le fait est OBSERVÉ (`public_interface`, dérivé de `public-ip-assignment`), pas
	# déduit d'une absence de `public_ip` — qui ne distinguerait pas « privée » de
	# « non collectée ». Là où le fournisseur ne publie pas ce champ, rien ne change.
	not _sans_interface_publique(vm)
	id := object.get(vm.attributes, "vm_id", vm.id)
	f := {
		"code": "compute_instance_has_security_group",
		"severity": "critical",
		"subject": id,
		"message": sprintf("VM « %s » sans groupe de sécurité : aucun filtrage réseau ne s'applique.", [id]),
		"remediation": "Attacher un groupe de sécurité restrictif (refus par défaut) à la VM.",
		"labels": {
			"provider": provider_of(vm),
			"category": "security",
			"confidence": "confirmed",
			"message_en": sprintf("VM \"%s\" has no security group: no network filtering applies to it.", [id]),
			"remediation_en": "Attach a restrictive security group (deny by default) to the VM.",
		},
	}
}

# _sans_interface_publique — la machine n'a AUCUNE interface publique, et le
# fournisseur le dit explicitement.
#
# Vrai seulement si l'attribut est présent ET faux. Un attribut absent ne rend pas
# vrai : ne pas savoir n'est pas savoir que non, et le contrôle continue alors de
# parler comme avant (ADR-0014).
_sans_interface_publique(vm) if {
	"public_interface" in object.keys(vm.attributes)
	not truthy(object.get(vm.attributes, "public_interface", true))
}

# compute_instance_deletion_protection
#   Instance de calcul sans protection contre la suppression : une commande
#   accidentelle ou malveillante détruit le service et ses données locales, sans
#   qu'aucune compromission ne soit nécessaire.
# SCSL : CLD-CMP-10 (protection contre la suppression accidentelle des ressources
#   de calcul portant un service en production) — vecteur V-CLD-10 (destruction /
#   rançongiciel cloud, ATT&CK T1485.001).
# Contrat : type normalisé agnostique `compute_instance` ; attribut
#   deletion_protection (bool ; Outscale Vm.DeletionProtection). Absent ⇒ pas de
#   finding (garde de capacité), l'assessment le marque « non évalué ».
package pepin.rules

import rego.v1

deny contains f if {
	some r in resources_of_type("compute_instance")
	object.get(r.attributes, "deletion_protection", true) == false
	id := object.get(r.attributes, "vm_id", r.id)
	f := {
		"code": "compute_instance_deletion_protection",
		"severity": "medium",
		"subject": id,
		"message": sprintf("Instance « %s » sans protection contre la suppression — une action accidentelle ou malveillante la détruit.", [id]),
		"remediation": "Activer la protection contre la suppression sur les instances portant un service en production.",
		"labels": {"provider": provider_of(r), "category": "compliance"},
	}
}

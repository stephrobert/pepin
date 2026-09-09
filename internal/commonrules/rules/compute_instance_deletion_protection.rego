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

	# PRODUCTION seulement. Le commentaire de ce contrôle visait « les ressources de
	# calcul portant un service en production » ; sa condition, elle, exigeait la
	# protection de TOUTE instance. Une VM de développement, de CI ou de laboratoire
	# produisait donc le même écart qu'un serveur de production — un faux positif
	# contextuel, et le genre qui use la confiance sans rien apprendre.
	is_production(r.attributes)
	not truthy(object.get(r.attributes, "deletion_protection", true))
	id := object.get(r.attributes, "vm_id", r.id)
	f := {
		"code": "compute_instance_deletion_protection",
		"severity": "medium",
		"subject": id,
		"message": sprintf("Instance « %s » sans protection contre la suppression — une action accidentelle ou malveillante la détruit.", [id]),
		"remediation": "Activer la protection contre la suppression sur les instances portant un service en production.",
		"labels": {
			"provider": provider_of(r),
			"category": "compliance",
			"confidence": "contextual",
			"message_en": sprintf("Instance \"%s\" has no deletion protection — an accidental or malicious action destroys it.", [id]),
			"remediation_en": "Enable deletion protection on the instances carrying a production service.",
		},
	}
}

# ── indéterminé : l'environnement n'est pas lisible ───────────────────────────
#
# Sans étiquette d'environnement, on ne SAIT pas si cette instance porte un service
# de production. Se taire vaudrait « conforme » et laisserait passer un vrai serveur
# non protégé ; crier ferait le faux positif qu'on vient de retirer. La règle
# constate, l'assessment statue (ADR-0015).
deny contains f if {
	some r in resources_of_type("compute_instance")
	not environment_known(r.attributes)
	not truthy(object.get(r.attributes, "deletion_protection", true))
	id := object.get(r.attributes, "vm_id", r.id)
	f := {
		"code": "compute_instance_deletion_protection",
		"severity": "medium",
		"subject": id,
		"message": sprintf("VM « %s » sans protection contre la suppression, et sans étiquette d'environnement — impossible de dire si elle porte un service de production.", [id]),
		"remediation": "Étiqueter l'environnement (Env / environment / stage) pour que ce contrôle sache s'il s'applique ; activer la protection sur les instances de production.",
		"labels": {
			"provider": provider_of(r),
			"category": "compliance",
			"confidence": "contextual",
			"inconclusive": "true",
			"message_en": sprintf("VM \"%s\" has no deletion protection and no environment tag — whether it carries a production service cannot be told.", [id]),
			"remediation_en": "Tag the environment (Env / environment / stage) so this control knows whether it applies; enable protection on production instances.",
		},
	}
}

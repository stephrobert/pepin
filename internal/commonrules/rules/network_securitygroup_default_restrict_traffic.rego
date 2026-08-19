# network_securitygroup_default_restrict_traffic
#   Le security group « default » d'un réseau est attaché automatiquement à toute
#   ressource dont on ne précise pas le SG. Il doit donc ne RIEN autoriser : la moindre
#   règle entrante qu'il porte s'applique silencieusement à ces ressources, sans décision
#   explicite. Équivalent : Prowler ec2_securitygroup_default_restrict_traffic.
# SCSL : CLD-NET-4 (filtrage réseau au moindre privilège).
# Contrat : type normalisé agnostique `security_group_rule` ; attributs
#   security_group_name (nom natif du SG) et direction. Nom absent ⇒ pas de finding
#   (garde de capacité) : l'assessment rend « non évalué » plutôt qu'un faux vert.
package pepin.rules

import rego.v1

deny contains f if {
	some r in resources_of_type("security_group_rule")
	lower(object.get(r.attributes, "security_group_name", "")) == "default"
	object.get(r.attributes, "direction", "") == "inbound"
	sg := object.get(r.attributes, "security_group_id", r.id)
	f := {
		"code": "network_securitygroup_default_restrict_traffic",
		"severity": "high",
		"subject": sg,
		"message": sprintf("Security group « default » (%s) : porte une règle entrante — il s'applique d'office à toute ressource créée sans SG explicite.", [sg]),
		"remediation": "Vider le security group « default » de toutes ses règles ; attacher explicitement un SG dédié et restrictif à chaque ressource.",
		"labels": {"provider": provider_of(r), "category": "security"},
	}
}

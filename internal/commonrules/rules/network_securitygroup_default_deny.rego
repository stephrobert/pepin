# network_securitygroup_default_deny
#   Groupe de sécurité dont la POLITIQUE PAR DÉFAUT en entrée est « accept » : tout
#   trafic non explicitement filtré est admis (équivalent any/any implicite), quel
#   que soit le jeu de règles explicites.
# SCSL : CLD-NET-2 (aucune règle any/any depuis Internet), CLD-NET-1.
# Contrat : type normalisé agnostique `security_group` ; attribut
#   inbound_default_policy (accept|drop, DÉRIVÉ — Scaleway Instance SecurityGroup
#   inbound_default_policy). Absent ⇒ pas de finding (garde de capacité).
package pepin.rules

import rego.v1

deny contains f if {
	some r in resources_of_type("security_group")
	lower(object.get(r.attributes, "inbound_default_policy", "")) == "accept"
	id := object.get(r.attributes, "security_group_id", r.id)
	f := {
		"code": "network_securitygroup_default_deny",
		"severity": "high",
		"subject": id,
		"message": sprintf("Security group « %s » : politique entrante par défaut « accept » — tout trafic non filtré est admis.", [id]),
		"remediation": "Basculer la politique entrante par défaut sur « drop » et n'ouvrir que les flux légitimes par des règles explicites.",
		"labels": {"provider": provider_of(r), "category": "security"},
	}
}

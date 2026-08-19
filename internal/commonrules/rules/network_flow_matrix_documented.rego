# Matrice des flux documentée : chaque flux entrant autorisé (règle de security
#   group) doit porter une justification. Type normalisé `security_group_rule`,
#   attribut `description` (justification du flux). Ne se déclenche que pour les
#   providers qui exposent ce champ (la clé `description` est présente) — un
#   provider sans description par règle ne déclenche pas la règle.
# Ancrage Exoscale : Security Group rule `description` (max 255), schéma
#   reference-api-schemas-security-group-rule. SCSL : CLD-NET-5 (matrice des flux
#   autorisés — services/protocoles/ports + justification).
package pepin.rules

import rego.v1

deny contains f if {
	some r in resources_of_type("security_group_rule")
	object.get(r.attributes, "direction", "inbound") == "inbound"
	"description" in object.keys(r.attributes) # provider exposant la justification
	object.get(r.attributes, "description", "") == ""
	port := object.get(r.attributes, "port_from", "?")
	f := {
		"code": "network_flow_matrix_documented",
		"severity": "medium",
		"subject": object.get(r.attributes, "security_group_id", r.id),
		"message": sprintf("Flux entrant autorisé (port %v) sans justification : la matrice des flux exige une description par règle.", [port]),
		"remediation": "Documenter chaque règle de security group (service, raison) via sa description ; tenir la matrice des flux à jour.",
		"labels": {
			"provider": provider_of(r),
			"category": "compliance",
			"message_en": sprintf("Allowed inbound flow (port %v) with no justification: the flow matrix requires a description per rule.", [port]),
			"remediation_en": "Document every security group rule (service, reason) in its description; keep the flow matrix up to date.",
		},
	}
}

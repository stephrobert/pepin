# iam_policy_no_notaction_notresource
#   Policy EIM avec un Statement Allow employant NotAction ou NotResource —
#   pattern d'inversion qui autorise tout sauf une liste (permissions imprévues).
# Origine : osc-policy OSC-EIM-009. SCSL : CLD-IAM-1.
# Contrat : type normalisé agnostique `iam_policy` ; statements[] (DÉRIVÉ du
#   document) avec not_action[]/not_resource[] quand présents.
package pepin.rules

import rego.v1

deny contains f if {
	some p in resources_of_type("iam_policy")
	some stmt in object.get(p.attributes, "statements", [])
	lower(object.get(stmt, "effect", "")) == "allow"
	_has_inversion(stmt)
	name := object.get(p.attributes, "policy_name", p.id)
	f := {
		"code": "iam_policy_no_notaction_notresource",
		"severity": "critical",
		"subject": name,
		"message": sprintf("Politique EIM « %s » : autorisation Allow employant NotAction/NotResource (inversion dangereuse).", [name]),
		"remediation": "Réécrire en liste d'autorisations explicite (Action/Resource) ; NotAction/NotResource n'est acceptable qu'avec Effect Deny.",
		"labels": {
			"provider": provider_of(p),
			"category": "security",
			"message_en": sprintf("EIM policy \"%s\": Allow statement using NotAction/NotResource (dangerous inversion).", [name]),
			"remediation_en": "Rewrite it as an explicit allow list (Action/Resource); NotAction/NotResource is only acceptable with an Effect Deny.",
		},
	}
}

_has_inversion(stmt) if count(object.get(stmt, "not_action", [])) > 0

_has_inversion(stmt) if count(object.get(stmt, "not_resource", [])) > 0

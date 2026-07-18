# iam_policy_no_wildcard_resource
#   Policy EIM avec un Statement Allow portant une ressource joker « * ».
# Origine : osc-policy OSC-EIM-008. SCSL : CLD-IAM-1.
# Contrat : type normalisé agnostique `iam_policy` ; statements[] (DÉRIVÉ du
#   document de la policy) {effect, actions[], resources[]}.
package pepin.rules

import rego.v1

deny contains f if {
	some p in resources_of_type("iam_policy")
	some stmt in object.get(p.attributes, "statements", [])
	object.get(stmt, "effect", "") == "Allow"
	"*" in object.get(stmt, "resources", [])

	# Pas de doublon avec iam_policy_no_administrative_privileges (Action * = plus grave).
	not "*" in object.get(stmt, "actions", [])
	name := object.get(p.attributes, "policy_name", p.id)
	f := {
		"code": "iam_policy_no_wildcard_resource",
		"severity": "high",
		"subject": name,
		"message": sprintf("Politique EIM « %s » : autorisation Allow portant Resource=\"*\" (toutes les ressources du compte).", [name]),
		"remediation": "Restreindre Resource aux identifiants (ORN) que la politique doit réellement couvrir.",
		"labels": {"provider": provider_of(p), "category": "security"},
	}
}

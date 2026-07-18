# iam_policy_no_administrative_privileges
#   Policy EIM avec un Statement Allow portant une Action joker « * » : accorde
#   toutes les permissions, viole le moindre privilège.
# Origine : osc-policy OSC-EIM-007. SCSL : CLD-IAM-1.
# Contrat : type normalisé agnostique `iam_policy`. osc-sdk-go iam Policy +
#   document de version (ReadPolicyVersion) ; attribut statements[] (DÉRIVÉ : le
#   document JSON de la version par défaut est parsé en
#   {effect, actions[], resources[]}).
package pepin.rules

import rego.v1

deny contains f if {
	some p in resources_of_type("iam_policy")
	some stmt in object.get(p.attributes, "statements", [])
	object.get(stmt, "effect", "") == "Allow"
	"*" in object.get(stmt, "actions", [])
	name := object.get(p.attributes, "policy_name", p.id)
	f := {
		"code": "iam_policy_no_administrative_privileges",
		"severity": "critical",
		"subject": name,
		"message": sprintf("Politique EIM « %s » : autorisation Allow portant Action=\"*\" (toutes les actions accordées).", [name]),
		"remediation": "Remplacer Action=\"*\" par la liste exhaustive des actions réellement nécessaires (moindre privilège).",
		"labels": {"provider": provider_of(p), "category": "security"},
	}
}

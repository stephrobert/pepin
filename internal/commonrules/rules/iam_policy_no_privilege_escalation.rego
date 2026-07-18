# Élévation de privilèges : une politique ne doit pas autoriser des actions de
#   gestion d'identité permettant à un principal de s'octroyer plus de droits
#   (attacher une politique, créer/modifier une politique, créer une clé d'accès…).
#   Type normalisé `iam_policy`, attribut statements [{effect, actions[], ...}].
# Ancrage Outscale (EIM) : actions de gestion d'identité de la doc locale (eim) —
#   AttachUserPolicy/AttachGroupPolicy/LinkManagedPolicyToUserGroup, CreatePolicy/
#   CreatePolicyVersion, PutUserPolicy/PutGroupPolicy, CreateAccessKey/UpdateAccessKey.
#   Matching par NOM d'action (insensible à la casse et au préfixe de service).
# SCSL : CLD-IAM-12 (aucune politique ne permet une élévation de privilèges).
package pepin.rules

import rego.v1

# Verbes d'actions de gestion d'identité permettant l'auto-élévation (minuscules).
_escalation_actions := {
	"attachuserpolicy", "attachgrouppolicy", "linkmanagedpolicytousergroup",
	"createpolicy", "createpolicyversion", "putuserpolicy", "putgrouppolicy",
	"createaccesskey", "updateaccesskey",
}

deny contains f if {
	some p in resources_of_type("iam_policy")
	some s in object.get(p.attributes, "statements", [])
	lower(object.get(s, "effect", "")) == "allow"
	some a in object.get(s, "actions", [])
	escalation_action(a)
	name := object.get(p.attributes, "policy_name", p.id)
	f := {
		"code": "iam_policy_no_privilege_escalation",
		"severity": "high",
		"subject": name,
		"message": sprintf("Politique « %s » : autorise une action de gestion d'identité permettant une élévation de privilèges (%s).", [name, a]),
		"remediation": "Retirer les actions de gestion d'identité (attacher/créer une politique, créer une clé) des politiques d'usage ; les réserver à un rôle d'administration dédié et scopé.",
		"labels": {"provider": provider_of(p), "category": "security"},
	}
}

escalation_action(a) if {
	some esc in _escalation_actions
	contains(lower(a), esc)
}

# Modèle par rôle (ex. Exoscale, policy CEL) : un rôle ÉDITABLE dont la policy
# autorise la gestion des rôles IAM (attribut dérivé manages_iam) permet
# l'auto-élévation. Les rôles prédéfinis (non éditables) sont hors scope.
deny contains f if {
	some r in resources_of_type("iam_role")
	object.get(r.attributes, "editable", true) == true
	object.get(r.attributes, "manages_iam", false) == true
	name := object.get(r.attributes, "name", r.id)
	f := {
		"code": "iam_policy_no_privilege_escalation",
		"severity": "high",
		"subject": name,
		"message": sprintf("Rôle IAM « %s » : sa politique autorise la gestion des rôles IAM — chemin d'élévation de privilèges.", [name]),
		"remediation": "Réserver la gestion des rôles/clés IAM à un rôle d'administration dédié ; retirer ces autorisations des rôles d'usage.",
		"labels": {"provider": provider_of(r), "category": "security"},
	}
}

# Modèle par PermissionSets (ex. Scaleway) : une politique confère un PermissionSet
# de gestion d'identité (attribut dérivé manages_iam) → chemin d'élévation.
deny contains f if {
	some p in resources_of_type("iam_policy")
	object.get(p.attributes, "manages_iam", false) == true
	name := object.get(p.attributes, "policy_name", p.id)
	f := {
		"code": "iam_policy_no_privilege_escalation",
		"severity": "high",
		"subject": name,
		"message": sprintf("Politique « %s » : confère la gestion de l'IAM (PermissionSet) — chemin d'élévation de privilèges.", [name]),
		"remediation": "Réserver la gestion IAM à une politique d'administration dédiée ; retirer le PermissionSet de gestion des politiques d'usage.",
		"labels": {"provider": provider_of(p), "category": "security"},
	}
}

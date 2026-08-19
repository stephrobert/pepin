# Élévation de privilèges : une politique ne doit pas autoriser des actions de
#   gestion d'identité permettant à un principal de s'octroyer plus de droits
#   (attacher une politique, créer/modifier une politique, créer une clé d'accès…).
#   Type normalisé `iam_policy`, attribut statements [{effect, actions[], ...}].
# Ancrage Outscale (EIM) : noms d'actions RÉELS du contrat OAPI (osc-api), pas leurs
#   équivalents AWS — LinkPolicy, LinkManagedPolicyToUserGroup, PutUserPolicy,
#   PutUserGroupPolicy, AddUserToUserGroup (s'ajouter à un groupe privilégié),
#   CreatePolicy/CreatePolicyVersion, SetDefaultPolicyVersion (basculer une policy sur
#   une version plus permissive), CreateAccessKey/UpdateAccessKey, UpdateApiAccessPolicy.
#   Matching sur le nom SANS préfixe de service, par PRÉFIXE : `UnlinkPolicy` (qui retire
#   une policy) ne doit pas être confondu avec `LinkPolicy`.
# SCSL : CLD-IAM-12 (aucune politique ne permet une élévation de privilèges).
package pepin.rules

import rego.v1

# Verbes d'actions de gestion d'identité permettant l'auto-élévation (minuscules).
_escalation_actions := {
	"linkpolicy", "linkmanagedpolicytousergroup",
	"putuserpolicy", "putusergrouppolicy",
	"addusertousergroup",
	"createpolicy", "setdefaultpolicyversion",
	"createaccesskey", "updateaccesskey", "updateapiaccesspolicy",
}

deny contains f if {
	some p in resources_of_type("iam_policy")
	some s in object.get(p.attributes, "statements", [])
	lower(object.get(s, "effect", "")) == "allow"
	some a in object.get(s, "actions", [])
	_escalation_action(a)
	name := object.get(p.attributes, "policy_name", p.id)
	f := {
		"code": "iam_policy_no_privilege_escalation",
		"severity": "high",
		"subject": name,
		"message": sprintf("Politique « %s » : autorise une action de gestion d'identité permettant une élévation de privilèges (%s).", [name, a]),
		"remediation": "Retirer les actions de gestion d'identité (attacher/créer une politique, créer une clé) des politiques d'usage ; les réserver à un rôle d'administration dédié et scopé.",
		"labels": {
			"provider": provider_of(p),
			"category": "security",
			"message_en": sprintf("Policy \"%s\": allows an identity management action that enables privilege escalation (%s).", [name, a]),
			"remediation_en": "Remove identity management actions (attach/create a policy, create a key) from everyday policies; reserve them for a dedicated, scoped administration role.",
		},
	}
}

# Nom d'action sans son préfixe de service (`api:LinkPolicy` -> `linkpolicy`).
_action_name(a) := n if {
	parts := split(lower(a), ":")
	n := parts[count(parts) - 1]
}

_escalation_action(a) if {
	some esc in _escalation_actions
	startswith(_action_name(a), esc)
}

# Joker de SERVICE d'identité (`eim:*`, `iam:*`) : il confère TOUTES les actions de gestion
# d'identité (dont attacher une policy / créer une clé) → auto-élévation, même si la Resource
# est scopée (donc non attrapé par iam_policy_no_wildcard_resource).
_escalation_action(a) if {
	some svc in {"eim", "iam"}
	lower(a) == sprintf("%s:*", [svc])
}

# Modèle par rôle (ex. Exoscale, policy CEL) : un rôle ÉDITABLE dont la policy
# autorise la gestion des rôles IAM (attribut dérivé manages_iam) permet
# l'auto-élévation. Les rôles prédéfinis (non éditables) sont hors scope.
deny contains f if {
	some r in resources_of_type("iam_role")
	truthy(object.get(r.attributes, "editable", true))
	truthy(object.get(r.attributes, "manages_iam", false))
	name := object.get(r.attributes, "name", r.id)
	f := {
		"code": "iam_policy_no_privilege_escalation",
		"severity": "high",
		"subject": name,
		"message": sprintf("Rôle IAM « %s » : sa politique autorise la gestion des rôles IAM — chemin d'élévation de privilèges.", [name]),
		"remediation": "Réserver la gestion des rôles/clés IAM à un rôle d'administration dédié ; retirer ces autorisations des rôles d'usage.",
		"labels": {
			"provider": provider_of(r),
			"category": "security",
			"message_en": sprintf("IAM role \"%s\": its policy allows managing IAM roles — a privilege escalation path.", [name]),
			"remediation_en": "Reserve IAM role and key management for a dedicated administration role; remove those permissions from everyday roles.",
		},
	}
}

# Modèle par PermissionSets (ex. Scaleway) : une politique confère un PermissionSet
# de gestion d'identité (attribut dérivé manages_iam) → chemin d'élévation.
deny contains f if {
	some p in resources_of_type("iam_policy")
	truthy(object.get(p.attributes, "manages_iam", false))
	name := object.get(p.attributes, "policy_name", p.id)
	f := {
		"code": "iam_policy_no_privilege_escalation",
		"severity": "high",
		"subject": name,
		"message": sprintf("Politique « %s » : confère la gestion de l'IAM (PermissionSet) — chemin d'élévation de privilèges.", [name]),
		"remediation": "Réserver la gestion IAM à une politique d'administration dédiée ; retirer le PermissionSet de gestion des politiques d'usage.",
		"labels": {
			"provider": provider_of(p),
			"category": "security",
			"message_en": sprintf("Policy \"%s\": grants IAM management (PermissionSet) — a privilege escalation path.", [name]),
			"remediation_en": "Reserve IAM management for a dedicated administration policy; remove the management PermissionSet from everyday policies.",
		},
	}
}

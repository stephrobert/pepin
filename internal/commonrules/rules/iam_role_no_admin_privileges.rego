# Moindre privilège d'un rôle IAM.
#   Type normalisé agnostique `iam_role`. Attribut DÉRIVÉ par le collecteur :
#   admin_privileges (bool) = la politique autorise tout par défaut.
# Ancrage Exoscale : policy CEL du rôle (GET /iam-role). `default-service-strategy`
#   vaut `allow` ou `deny` ; `allow` = tout autorisé sauf deny ciblés = privilèges
#   étendus (équivalent CEL du joker admin AWS). Réf : references/docs/exoscale/
#   product-iam-how-to-policy-guide.md. Schéma TF : policy.default_service_strategy.
# SCSL : CLD-IAM-1 (interdire les identités/clés à privilèges excessifs).
package pepin.rules

import rego.v1

# Rôle dont la politique autorise tout par défaut. Ne se déclenche que si le
# collecteur a renseigné `admin_privileges` (provider exposant la capacité).
deny contains f if {
	some r in resources_of_type("iam_role")
	object.get(r.attributes, "editable", true) == true # rôles prédéfinis (non éditables) hors scope
	object.get(r.attributes, "admin_privileges", false) == true
	name := object.get(r.attributes, "name", r.id)
	f := {
		"code": "iam_role_no_admin_privileges",
		"severity": "high",
		"subject": name,
		"message": sprintf("Rôle IAM « %s » : politique en « allow » par défaut — privilèges étendus, à l'encontre du moindre privilège.", [name]),
		"remediation": "Repartir d'une stratégie de service « deny » par défaut et n'autoriser explicitement que le strict nécessaire.",
		"labels": {"provider": provider_of(r), "category": "security"},
	}
}

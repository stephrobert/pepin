# Moindre privilège d'un rôle IAM.
#   Type normalisé agnostique `iam_role`. Attribut DÉRIVÉ par le collecteur :
#   admin_privileges (bool) = la politique autorise tout par défaut.
# Ancrage Exoscale : policy CEL du rôle (GET /iam-role). `default-service-strategy`
#   vaut `allow` ou `deny` ; `allow` = tout autorisé sauf deny ciblés = privilèges
#   étendus (équivalent CEL du joker admin des policies IAM). Réf : references/docs/exoscale/
#   product-iam-how-to-policy-guide.md. Schéma TF : policy.default_service_strategy.
# SCSL : CLD-IAM-1 (interdire les identités/clés à privilèges excessifs).
package pepin.rules

import rego.v1

# Rôle dont la politique autorise tout par défaut. Ne se déclenche que si le
# collecteur a renseigné `admin_privileges` (provider exposant la capacité).
deny contains f if {
	some r in resources_of_type("iam_role")
	truthy(object.get(r.attributes, "editable", true)) # rôles prédéfinis (non éditables) hors scope
	truthy(object.get(r.attributes, "admin_privileges", false))
	name := object.get(r.attributes, "name", r.id)
	f := {
		"code": "iam_role_no_admin_privileges",
		"severity": "high",
		"subject": name,
		"message": sprintf("Rôle IAM « %s » : politique en « allow » par défaut — privilèges étendus, à l'encontre du moindre privilège.", [name]),
		"remediation": "Repartir d'une stratégie de service « deny » par défaut et n'autoriser explicitement que le strict nécessaire.",
		"labels": {
			"provider": provider_of(r),
			"category": "security",
			"message_en": sprintf("IAM role \"%s\": policy set to \"allow\" by default — broad privileges, against least privilege.", [name]),
			"remediation_en": "Start again from a default \"deny\" service strategy and explicitly allow only what is strictly needed.",
		},
	}
}

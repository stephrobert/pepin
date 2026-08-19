# Restriction d'IP source sur un rôle IAM.
#   Type normalisé agnostique `iam_role`. Attribut DÉRIVÉ par le collecteur :
#   source_ip_restricted (bool) = la politique du rôle borne les IP sources.
# Ancrage Exoscale : policy CEL du rôle (GET /iam-role) ; la restriction par IP
#   s'exprime via le binding source_ip — source_ip.inIpRange('<CIDR>'),
#   source_ip == '<ip>', source_ip in [...] (references/docs/exoscale/
#   product-iam-how-to-policy-guide.md). Le collecteur dérive le drapeau (contains).
# SCSL : CLD-IAM-4 (restriction d'accès par plage d'IP).
package pepin.rules

import rego.v1

# Rôle dont la politique ne borne aucune IP source. Ne se déclenche que si le
# collecteur a renseigné `source_ip_restricted` (provider exposant la capacité).
deny contains f if {
	some r in resources_of_type("iam_role")
	truthy(object.get(r.attributes, "editable", true)) # rôles prédéfinis (non éditables) hors scope
	not truthy(object.get(r.attributes, "source_ip_restricted", true))
	name := object.get(r.attributes, "name", r.id)
	f := {
		"code": "iam_role_source_ip_restricted",
		"severity": "high",
		"subject": name,
		"message": sprintf("Rôle IAM « %s » : aucune restriction d'IP source — une clé assumant ce rôle est utilisable depuis n'importe quelle adresse.", [name]),
		"remediation": "Ajouter à la politique du rôle une condition sur l'IP source (plages d'administration légitimes).",
		"labels": {"provider": provider_of(r), "category": "security"},
	}
}

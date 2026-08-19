# Règles d'accès à l'API du compte — COMMUNES (labels.provider tiré de la ressource).
#   Types normalisés : api_access_rule (ip_ranges[]), api_access_summary
#   (rule_count, DÉRIVÉ du collecteur), api_access_policy
#   (max_access_key_expiration_seconds).
# Origine : osc-policy OSC-ACC-001/003/004. SCSL : CLD-IAM-4, CLD-IAM-2.
package pepin.rules

import rego.v1

# Règle d'accès API ouverte à un CIDR public — SAUF si elle exige un certificat
# client (CaIds/Cns) : dans ce cas la plage IP ouverte ne suffit pas à appeler l'API,
# l'appelant doit présenter un certificat émis par une CA déclarée. Flaguer une telle
# règle serait un faux positif. Attributs absents (provider ne les exposant pas) ⇒
# comportement inchangé (on flague), donc rétro-compatible.
deny contains f if {
	some r in resources_of_type("api_access_rule")
	some cidr in cidr_list(object.get(r.attributes, "ip_ranges", []))
	is_public_cidr(cidr)
	not _requires_client_cert(r)
	f := {
		"code": "iam_apiaccessrule_no_public_cidr",
		"severity": "high",
		"subject": object.get(r.attributes, "api_access_rule_id", r.id),
		"message": sprintf("Règle d'accès API « %s » : appels autorisés depuis %s (CIDR public).", [object.get(r.attributes, "api_access_rule_id", r.id), cidr]),
		"remediation": "Restreindre les plages IP de la règle aux adresses légitimes ; supprimer toute règle ouverte à 0.0.0.0/0 ou ::/0.",
		"labels": {
			"provider": provider_of(r),
			"category": "security",
			"message_en": sprintf("API access rule \"%s\": calls allowed from %s (public CIDR).", [object.get(r.attributes, "api_access_rule_id", r.id), cidr]),
			"remediation_en": "Restrict the rule's IP ranges to legitimate addresses; remove any rule open to 0.0.0.0/0 or ::/0.",
		},
	}
}

# La règle impose un certificat client : CA déclarée(s) ou Common Name(s) exigé(s).
_requires_client_cert(r) if count(object.get(r.attributes, "ca_ids", [])) > 0

_requires_client_cert(r) if count(object.get(r.attributes, "cns", [])) > 0

# Aucune règle d'accès API définie (API ouverte depuis n'importe où).
deny contains f if {
	some r in resources_of_type("api_access_summary")
	object.get(r.attributes, "rule_count", 0) == 0
	f := {
		"code": "iam_apiaccessrule_defined",
		"severity": "high",
		"subject": object.get(r, "name", r.id),
		"message": "Aucune règle d'accès API définie : l'API du compte est joignable depuis n'importe où avec une clé valide.",
		"remediation": "Créer au moins une règle d'accès API restreignant les appels à des plages IP légitimes.",
		"labels": {
			"provider": provider_of(r),
			"category": "security",
			"message_en": "No API access rule is defined: the account's API is reachable from anywhere with a valid key.",
			"remediation_en": "Create at least one API access rule restricting calls to legitimate IP ranges.",
		},
	}
}

# Politique d'accès API sans expiration maximale des clés.
deny contains f if {
	some r in resources_of_type("api_access_policy")
	"max_access_key_expiration_seconds" in object.keys(r.attributes) # provider EXPOSANT le réglage
	object.get(r.attributes, "max_access_key_expiration_seconds", 0) <= 0 # 0 = aucune limite (sémantique OAPI)
	f := {
		"code": "iam_apiaccesspolicy_max_key_expiration",
		"severity": "medium",
		"subject": object.get(r, "name", r.id),
		"message": "Politique d'accès API : aucune expiration maximale imposée aux clés d'accès.",
		"remediation": "Configurer une expiration maximale des clés d'accès (ex. 90 jours).",
		"labels": {
			"provider": provider_of(r),
			"category": "security",
			"message_en": "API access policy: no maximum expiry is enforced on access keys.",
			"remediation_en": "Configure a maximum access key expiry (90 days, for instance).",
		},
	}
}

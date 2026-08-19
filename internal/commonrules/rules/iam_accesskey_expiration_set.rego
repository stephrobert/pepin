# iam_accesskey_expiration_set
#   Clé d'accès sans date d'expiration : une fuite reste exploitable indéfiniment.
# Origine : osc-policy OSC-KEY-001. SCSL : CLD-IAM-2.
# Contrat : type normalisé agnostique `access_key` ; attribut expiration_date
#   (string RFC3339 ; osc-sdk-go AccessKey.ExpirationDate, vide si non défini).
package pepin.rules

import rego.v1

deny contains f if {
	some k in resources_of_type("access_key")
	object.get(k.attributes, "expiration_date", "") == ""
	id := object.get(k.attributes, "access_key_id", k.id)
	f := {
		"code": "iam_accesskey_expiration_set",
		"severity": "critical",
		"subject": id,
		"message": sprintf("Clé d'accès « %s » sans date d'expiration — une fuite resterait exploitable indéfiniment.", [id]),
		"remediation": "Définir une date d'expiration sur la clé et mettre en place une rotation ; préférer une identité courte (OIDC).",
		"labels": {
			"provider": provider_of(k),
			"category": "security",
			"message_en": sprintf("Access key \"%s\" has no expiry date — a leak would stay exploitable indefinitely.", [id]),
			"remediation_en": "Set an expiry date on the key and put a rotation in place; prefer a short-lived identity (OIDC).",
		},
	}
}

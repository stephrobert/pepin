# iam_accesskey_expiration_set
#   Clé d'accès sans date d'expiration : une fuite reste exploitable indéfiniment.
# Origine : osc-policy OSC-KEY-001. SCSL : CLD-IAM-2.
# Contrat : type normalisé agnostique `access_key` ; attributs expiration_date et
#   state (strings ; osc-sdk-go v2.24.0 AccessKey.ExpirationDate / .State, vérifiés
#   dans model_access_key.go). ExpirationDate est vide si non définie.
#
# # Une date PASSÉE ne protège rien
#
# La règle ne refusait qu'une date ABSENTE. Une clé ACTIVE dont l'expiration est
# dépassée depuis deux ans passait donc pour conforme — mesuré sur un tenant réel,
# avec `ExpirationDate: 2024-09-07` et `State: ACTIVE`.
#
# Les deux issues de ce cas sont mauvaises, et c'est pourquoi le `pass` était faux :
# soit le fournisseur honore encore la clé, et l'expiration ne protège rien ; soit il
# ne l'honore plus, et une clé morte reste déclarée active dans le tenant. Le scan ne
# tranche pas laquelle — il n'a pas à le faire pour savoir que « conforme » est faux.
#
# L'instant de référence est celui de l'ÉVALUATION (`_eval_now_ns`, ancré sur
# `input.evaluated_at`), pas l'horloge : sans quoi le rejeu d'un bundle scellé
# rendrait un autre verdict que celui qui y a été scellé.
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
			"confidence": "confirmed",
			"message_en": sprintf("Access key \"%s\" has no expiry date — a leak would stay exploitable indefinitely.", [id]),
			"remediation_en": "Set an expiry date on the key and put a rotation in place; prefer a short-lived identity (OIDC).",
		},
	}
}

# Date d'expiration DÉPASSÉE sur une clé encore active : l'expiration existe, elle ne
# protège plus. Sévérité `high` plutôt que `critical` : une clé sans aucune expiration
# est indéfiniment exploitable, celle-ci est au moins bornée par une décision passée.
deny contains f if {
	some k in resources_of_type("access_key")
	date := object.get(k.attributes, "expiration_date", "")
	date != ""
	_expired(date)
	lower(object.get(k.attributes, "state", "")) in {"active", ""}
	id := object.get(k.attributes, "access_key_id", k.id)
	f := {
		"code": "iam_accesskey_expiration_set",
		"severity": "high",
		"subject": id,
		"message": sprintf("Clé d'accès « %s » encore active alors que son expiration (%s) est dépassée — l'échéance ne protège plus rien, et la clé n'a pas été renouvelée depuis.", [id, date]),
		"remediation": "Révoquer la clé et en émettre une nouvelle avec une échéance à venir ; mettre en place une rotation automatique.",
		"labels": {
			"provider": provider_of(k),
			"category": "security",
			# CONFIRMÉ : la date et l'état sont observés, et leur comparaison ne suppose
			# rien du contexte. Que le fournisseur honore ou non une clé expirée ne
			# change pas le fait qu'elle n'a pas été renouvelée.
			"confidence": "confirmed",
			"message_en": sprintf("Access key \"%s\" is still active while its expiry (%s) has passed — the deadline no longer protects anything, and the key has not been renewed since.", [id, date]),
			"remediation_en": "Revoke the key and issue a new one with a future expiry; put an automatic rotation in place.",
		},
	}
}

# _expired — la date d'expiration est derrière l'instant d'évaluation.
#
# Une date ILLISIBLE ne rend pas la règle vraie : `time.parse_rfc3339_ns` y devient
# indéfini, donc le corps ne se déclenche pas. C'est le bon sens du repli — on ne
# déduit pas un écart d'une donnée qu'on n'a pas su lire (ADR-0014).
_expired(date) if time.parse_rfc3339_ns(date) < _eval_now_ns

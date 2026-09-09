# iam_accesskey_rotated — ROTATION d'une clé d'accès longue durée.
#
# SCSL : CLD-IAM-2, qui demande « clés d'accès longue durée assorties d'une expiration
# ET d'une rotation ». `iam_accesskey_expiration_set` mesure la première moitié ;
# celui-ci mesure la seconde, qui n'était mesurée nulle part.
#
# # Pourquoi la présence d'une date ne suffit pas
#
# Mesuré sur un tenant réel : une clé dont l'expiration est fixée à 2099 satisfait le
# contrôle d'expiration sans qu'aucune rotation n'ait jamais eu lieu. Et le plafond de
# compte (`iam_apiaccesspolicy_max_key_expiration`) est le bon endroit pour borner cela
# — sauf qu'un tenant avec `MaxAccessKeyExpirationSeconds: 0` n'a aucun plafond, et les
# dates par clé sont alors le seul signal qui reste.
#
# # Ce que la règle mesure, et ce qu'elle ne mesure pas
#
# Elle mesure l'ÂGE : une clé créée il y a plus que la fenêtre configurée n'a pas été
# remplacée depuis. Elle ne mesure PAS l'usage — une clé ancienne et jamais utilisée
# reste un secret valide, donc son âge compte quand même.
#
# Contrat : type normalisé agnostique `access_key` ; attributs creation_date et state
#   (strings ; osc-sdk-go v2.24.0 AccessKey.CreationDate / .State, vérifiés dans
#   model_access_key.go). L'instant de référence est celui de l'ÉVALUATION.
package pepin.rules

import rego.v1

deny contains f if {
	some k in resources_of_type("access_key")
	creee := object.get(k.attributes, "creation_date", "")
	creee != ""
	_older_than_window(creee)
	lower(object.get(k.attributes, "state", "")) in {"active", ""}
	id := object.get(k.attributes, "access_key_id", k.id)
	f := {
		"code": "iam_accesskey_rotated",
		"severity": "high",
		"subject": id,
		"message": sprintf("Clé d'accès « %s » créée le %s et jamais remplacée depuis — au-delà de la fenêtre de rotation de %d jours.", [id, creee, key_max_age_days]),
		"remediation": "Émettre une nouvelle clé, migrer les consommateurs, puis révoquer l'ancienne ; automatiser la rotation. Préférer une identité courte (OIDC) à un secret statique.",
		"labels": {
			"provider": provider_of(k),
			"category": "security",
			# CONFIRMÉ : la date de création et l'état sont observés, et l'âge s'en
			# déduit sans rien supposer du contexte. Ce que la règle ne sait pas —
			# l'usage de la clé — ne change pas le fait qu'elle n'a pas été remplacée.
			"confidence": "confirmed",
			"message_en": sprintf("Access key \"%s\" was created on %s and never replaced since — past the %d-day rotation window.", [id, creee, key_max_age_days]),
			"remediation_en": "Issue a new key, migrate its consumers, then revoke the old one; automate the rotation. Prefer a short-lived identity (OIDC) over a static secret.",
		},
	}
}

# _older_than_window — la clé a été créée avant la fenêtre de rotation.
#
# Une date ILLISIBLE ne produit aucun écart : `time.parse_rfc3339_ns` y devient
# indéfini, donc le corps ne se déclenche pas. On ne déduit rien d'une donnée qu'on n'a
# pas su lire (ADR-0014).
_older_than_window(date) if time.parse_rfc3339_ns(date) < _eval_now_ns - key_max_age_ns

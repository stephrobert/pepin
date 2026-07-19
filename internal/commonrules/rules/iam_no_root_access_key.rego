# iam_no_root_access_key — Clé d'API rattachée au compte root (contourne l'IAM).
# Origine : Pépin (POC Scaleway). SCSL : CLD-IAM-1.
# Contrat : type normalisé agnostique `access_key` ; attributs natifs
#   scaleway-sdk-go iam/v1alpha1 APIKey (AccessKey, UserID, ApplicationID).
# Champ `root_owned` : DÉRIVÉ par le collecteur (APIKey.UserID == utilisateur
#   propriétaire de l'organisation) — pas un champ brut de l'API.
package pepin.rules

import rego.v1

deny contains f if {
	some r in input.resources
	r.type == "access_key"
	object.get(r.attributes, "root_owned", false) == true
	name := object.get(r, "name", r.id)
	f := {
		"code": "iam_no_root_access_key",
		"severity": "high",
		"subject": name,
		"message": sprintf("Clé d'API « %s » rattachée au compte root — contourne les politiques IAM.", [name]),
		"remediation": "Créer une application IAM dédiée à moindre privilège puis révoquer la clé root.",
		"labels": {"provider": provider_of(r), "category": "security"},
	}
}

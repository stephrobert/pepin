# database_encryption_at_rest_enabled
#   Base de données managée sans chiffrement au repos : en cas d'accès au support
#   de stockage, les données sont lisibles en clair.
# SCSL : CLD-CHF-2 (chiffrement au repos sur les volumes et le stockage objet).
# Contrat : type normalisé agnostique `managed_database` ; attribut encryption_at_rest
#   (bool, DÉRIVÉ — Scaleway RDB encryption_at_rest). Absent ⇒ pas de finding
#   (garde de capacité : le provider n'expose pas l'état de chiffrement).
package pepin.rules

import rego.v1

deny contains f if {
	some r in resources_of_type("managed_database")
	"encryption_at_rest" in object.keys(r.attributes)
	not truthy(object.get(r.attributes, "encryption_at_rest", false))
	id := object.get(r.attributes, "database_id", r.id)
	f := {
		"code": "database_encryption_at_rest_enabled",
		"severity": "high",
		"subject": id,
		"message": sprintf("Base de données managée « %s » sans chiffrement au repos.", [id]),
		"remediation": "Activer le chiffrement au repos de l'instance (à la création ou par mise à niveau).",
		"labels": {"provider": provider_of(r), "category": "security"},
	}
}

# database_backup_enabled
#   Base de données managée dont les sauvegardes automatiques sont DÉSACTIVÉES :
#   pas de restauration garantie après incident ou suppression.
# SCSL : CLD-STO-3 (sauvegarde régulière des données et de la configuration).
# Contrat : type normalisé agnostique `managed_database` ; attribut disable_backup
#   (bool, DÉRIVÉ — Scaleway RDB disable_backup). Absent/false ⇒ sauvegardes actives
#   par défaut, pas de finding.
package pepin.rules

import rego.v1

deny contains f if {
	some r in resources_of_type("managed_database")
	truthy(object.get(r.attributes, "disable_backup", false))
	id := object.get(r.attributes, "database_id", r.id)
	f := {
		"code": "database_backup_enabled",
		"severity": "high",
		"subject": id,
		"message": sprintf("Base de données managée « %s » : sauvegardes automatiques désactivées.", [id]),
		"remediation": "Réactiver les sauvegardes automatiques et fixer une rétention adaptée au RPO.",
		"labels": {"provider": provider_of(r), "category": "compliance"},
	}
}

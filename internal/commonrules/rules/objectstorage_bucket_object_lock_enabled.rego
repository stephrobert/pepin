# Immutabilité du stockage objet (Object Lock / WORM).
#   Un bucket sans Object Lock ne protège pas ses objets contre la suppression ou
#   l'écrasement (accidentel ou malveillant — rançongiciel). Recommandé pour les
#   buckets de sauvegarde et les objets critiques.
#   Type normalisé `object_storage_bucket`, attribut object_lock_enabled (bool).
#   Ne se déclenche que si l'attribut est renseigné (provider exposant la capacité).
# Ancrage : API S3 GetObjectLockConfiguration (SOS/OOS/Scaleway) ; schéma Terraform
#   scaleway_object_bucket.object_lock_enabled. SCSL : CLD-STO-8.
package pepin.rules

import rego.v1

deny contains f if {
	some b in resources_of_type("object_storage_bucket")
	"object_lock_enabled" in object.keys(b.attributes)
	not truthy(object.get(b.attributes, "object_lock_enabled", true))
	name := object.get(b.attributes, "name", b.id)
	f := {
		"code": "objectstorage_bucket_object_lock_enabled",
		"severity": "low",
		"subject": name,
		"message": sprintf("Bucket « %s » sans Object Lock : objets non immuables (pas de protection WORM contre suppression/écrasement).", [name]),
		"remediation": "Activer l'Object Lock (mode conformité/gouvernance) sur les buckets de sauvegarde et d'objets critiques.",
		"labels": {
			"provider": provider_of(b),
			"category": "compliance",
			"message_en": sprintf("Bucket \"%s\" has no Object Lock: objects are mutable (no WORM protection against deletion or overwrite).", [name]),
			"remediation_en": "Enable Object Lock (compliance or governance mode) on backup buckets and critical objects.",
		},
	}
}

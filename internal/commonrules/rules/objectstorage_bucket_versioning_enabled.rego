# objectstorage_bucket_versioning_enabled — COMMUN à tous les providers.
#   Bucket de stockage objet sans versioning : suppression/écrasement irréversible
#   (accidentel ou malveillant, ex. rançongiciel).
# SCSL : CLD-STO-4. Attribut versioning (string S3 : "Enabled" | "Suspended" |
#   "" si jamais activé) produit par le collecteur partagé internal/objectstorage.
package pepin.rules

import rego.v1

deny contains f if {
	some r in resources_of_type("object_storage_bucket")
	versioning := object.get(r.attributes, "versioning", "absent")
	versioning != "absent" # clé non collectée (ex. source Terraform) ⇒ pas de faux positif.
	versioning != "Enabled"
	bucket := object.get(r.attributes, "name", r.id)
	f := {
		"code": "objectstorage_bucket_versioning_enabled",
		# medium : l'absence de versioning est une lacune de résilience, pas une exposition
		# directe ; les référentiels CSPM la classent medium (high réservé à l'exposition/fuite).
		"severity": "medium",
		"subject": bucket,
		"message": sprintf("Bucket « %s » sans versioning activé (statut %q) — suppression/écrasement irréversible.", [bucket, versioning]),
		"remediation": "Activer le versioning du bucket, au moins pour les données critiques.",
		"labels": {"provider": provider_of(r), "category": "security"},
	}
}

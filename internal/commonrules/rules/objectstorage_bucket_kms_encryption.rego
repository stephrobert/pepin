# Clé de chiffrement gérée par le client (BYOK / SSE-KMS) sur un bucket sensible.
#   CLD-CHF-4 (R3, DOIT) : « clés de chiffrement gérées par le client (BYOK/HYOK)
#   pour les données sensibles, hors du contrôle exclusif du fournisseur ».
#   La règle ne se déclenche QUE si :
#     - l'attribut `sse_kms_enabled` est présent (le provider expose une clé client
#       au niveau bucket — seul Scaleway via Key Manager aujourd'hui ; SOS Exoscale
#       = SSE-C par-objet, OOS Outscale = AES256 fournisseur → attribut absent) ;
#     - le bucket porte un tag de SENSIBILITÉ (convention ci-dessous) ;
#     - le chiffrement par défaut n'utilise PAS de clé client (sse_kms_enabled=false).
#   → ciblage des seules données classées sensibles : pas de bruit sur les buckets
#   ordinaires (l'absence de BYOK n'est un écart que pour les données sensibles).
#
# Convention de sensibilité (tags {key,value} normalisés) : clé ∈ {data_classification,
#   classification, sensitivity, confidentialite} et valeur ∈ {sensitive, confidential,
#   confidentiel, restricted, secret, high} (insensible à la casse).
# Ancrage : API S3 GetBucketEncryption → règle par défaut SSEAlgorithm=aws:kms +
#   KMSMasterKeyID (KEK client via Scaleway Key Manager, doc object-storage/api-cli/
#   enable-sse-kms). SCSL : CLD-CHF-4.
package pepin.rules

import rego.v1

_sensitivity_keys := {"data_classification", "classification", "sensitivity", "confidentialite"}

_sensitivity_values := {"sensitive", "confidential", "confidentiel", "restricted", "secret", "high"}

# _bucket_sensitive — le bucket porte un tag classant ses données comme sensibles.
_bucket_sensitive(b) if {
	some t in object.get(b.attributes, "tags", [])
	lower(object.get(t, "key", "")) in _sensitivity_keys
	lower(object.get(t, "value", "")) in _sensitivity_values
}

deny contains f if {
	some b in resources_of_type("object_storage_bucket")
	"sse_kms_enabled" in object.keys(b.attributes)
	not truthy(object.get(b.attributes, "sse_kms_enabled", true))
	_bucket_sensitive(b)
	name := object.get(b.attributes, "name", b.id)
	f := {
		"code": "objectstorage_bucket_kms_encryption",
		"severity": "medium",
		"subject": name,
		"message": sprintf("Bucket « %s » classé sensible mais chiffré sans clé gérée par le client (SSE-KMS) : données sous le contrôle exclusif du fournisseur.", [name]),
		"remediation": "Créer une clé dans Key Manager et l'associer au bucket comme clé de chiffrement par défaut (SSE-KMS) pour les données sensibles.",
		"labels": {
			"provider": provider_of(b),
			"category": "compliance",
			"message_en": sprintf("Bucket \"%s\" is classified sensitive but encrypted without a customer-managed key (SSE-KMS): its data stays under the provider's exclusive control.", [name]),
			"remediation_en": "Create a key in Key Manager and attach it to the bucket as its default encryption key (SSE-KMS) for sensitive data.",
		},
	}
}

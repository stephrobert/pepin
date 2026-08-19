# objectstorage_bucket_default_encryption
#   Bucket de stockage objet sans chiffrement par défaut au repos : les objets déposés
#   sont écrits en clair côté fournisseur.
# SCSL : CLD-CHF-2 (chiffrement des données au repos).
# Contrat : type normalisé agnostique `object_storage_bucket` ; attribut
#   default_encryption_enabled (bool, DÉRIVÉ : présence d'une règle de chiffrement par
#   défaut sur le bucket). À NE PAS confondre avec sse_kms_enabled, qui porte sur la
#   gestion CLIENT de la clé (BYOK, CLD-CHF-4) : un bucket peut chiffrer avec une clé
#   fournisseur sans BYOK. Le chiffrement est opt-in par bucket chez plusieurs
#   fournisseurs souverains : son absence est un écart réel, pas une fatalité.
#   Attribut absent (403, endpoint non supporté) ⇒ pas de finding : l'assessment le rend
#   « non évalué » via requiredAttr, jamais un faux vert.
package pepin.rules

import rego.v1

deny contains f if {
	some b in resources_of_type("object_storage_bucket")
	object.get(b.attributes, "default_encryption_enabled", true) == false
	name := object.get(b.attributes, "name", b.id)
	f := {
		"code": "objectstorage_bucket_default_encryption",
		"severity": "medium",
		"subject": name,
		"message": sprintf("Bucket « %s » sans chiffrement par défaut au repos — les objets y sont écrits en clair côté fournisseur.", [name]),
		"remediation": "Activer le chiffrement par défaut du bucket (SSE) ; vérifier que les objets déjà déposés sont ré-écrits chiffrés.",
		"labels": {"provider": provider_of(b), "category": "security"},
	}
}

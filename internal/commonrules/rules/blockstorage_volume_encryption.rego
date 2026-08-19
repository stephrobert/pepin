# Chiffrement au repos des volumes block storage.
#   Un volume dont le chiffrement au repos est désactivé expose ses données en
#   clair côté plateforme. La règle ne se déclenche QUE si l'attribut `encrypted`
#   est explicitement renseigné à false : un provider qui ne l'expose pas (absent
#   du contrat) ne produit aucun faux positif.
#   Type normalisé `blockstorage_volume`, attribut encrypted (bool).
#
# Modèle de chiffrement au repos par provider (ANCRÉ — le
# chiffrement est un drapeau optionnel détectable) :
#   - Exoscale : AES-256 XTS transparent au niveau hyperviseur, TOUJOURS actif,
#     sans configuration (doc storage/block-storage) → encrypted const true,
#     conforme par construction.
#   - Outscale : chiffrement CÔTÉ INVITÉ (EncFS/LUKS dans la VM), responsabilité
#     client ; osc-sdk-go/v2 Volume n'expose AUCUN champ de chiffrement → non
#     observable via l'API, attribut absent.
#   - Scaleway : chiffrement CÔTÉ INVITÉ (LUKS/Cryptsetup), responsabilité client
#     (modèle de responsabilité partagée) → non exposé par l'API block, absent.
# Ancrage : egoscale/v3 BlockStorageVolume (Exoscale, chiffrement transparent) ;
#   doc Outscale « Encrypting Data on Your Volumes » (EncFS) ; doc Scaleway
#   « Storage shared responsibility model » (LUKS). SCSL : CLD-CHF-2.
package pepin.rules

import rego.v1

deny contains f if {
	some v in resources_of_type("blockstorage_volume")
	"encrypted" in object.keys(v.attributes)
	not truthy(object.get(v.attributes, "encrypted", true))
	name := object.get(v.attributes, "volume_id", v.id)
	f := {
		"code": "blockstorage_volume_encryption",
		"severity": "high",
		"subject": name,
		"message": sprintf("Volume block storage « %s » : chiffrement au repos désactivé (données en clair côté plateforme).", [name]),
		"remediation": "Activer le chiffrement au repos du volume (selon le fournisseur : transparent, clé gérée fournisseur, ou chiffrement client).",
		"labels": {
			"provider": provider_of(v),
			"category": "compliance",
			"message_en": sprintf("Block storage volume \"%s\": encryption at rest is disabled (data in cleartext on the platform side).", [name]),
			"remediation_en": "Enable encryption at rest on the volume (depending on the provider: transparent, provider-managed key, or client-side encryption).",
		},
	}
}

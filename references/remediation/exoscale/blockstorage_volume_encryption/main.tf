# Remédiation CONFORME : blockstorage_volume_encryption (module autonome).
# Exigence : CLD-CHF-2 (chiffrement des données au repos).
# Fournisseur : exoscale (exoscale_block_storage_volume).
# Pourquoi conforme : chez Exoscale, le chiffrement au repos du block storage est
#   TRANSPARENT et toujours actif (AES-256 XTS au niveau de l'hyperviseur), sans
#   option à poser : le descripteur le porte en const encrypted = true, et la règle
#   Pépin ne se déclenche que si l'attribut vaut explicitement false. Il n'y a donc
#   pas de « montage à corriger » chez ce fournisseur ; ce module montre la forme d'un
#   volume conforme, et rappelle que le chiffrement DANS l'invité (LUKS) reste la
#   réponse quand une exigence impose une clé maîtrisée par le client.
# Source : references/docs/exoscale/product-storage-block-storage-overview.md ;
#          references/docs/exoscale/reference-api-schemas-block-storage-volume.md ;
#          contrat providers/exoscale.yaml (mapping blockstorage_volume).
terraform {
  required_providers {
    exoscale = { source = "exoscale/exoscale" }
  }
}

provider "exoscale" {}

locals {
  # Zone de l'Union européenne (Vienne) : la localisation UE est une exigence
  # transverse (CLD-GVN-3), donc les montages conformes la portent par défaut.
  zone = "at-vie-1"
}

resource "exoscale_block_storage_volume" "donnees" {
  zone = local.zone
  name = "donnees-applicatives"
  size = 100

  labels = {
    CostCenter = "CC-4711"
    Project    = "socle-applicatif"
    Env        = "production"
    Owner      = "equipe-plateforme"
  }
}

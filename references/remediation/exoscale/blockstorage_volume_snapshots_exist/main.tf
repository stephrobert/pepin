# Remédiation CONFORME : blockstorage_volume_snapshots_exist (module autonome).
# Exigence : CLD-STO-3 (sauvegarde récente et restauration éprouvée).
# Fournisseur : exoscale (exoscale_block_storage_volume,
#   exoscale_block_storage_volume_snapshot).
# Pourquoi conforme : le volume porte un snapshot, déclaré à côté de lui. La règle
#   Pépin cherche un snapshot de MOINS DE SEPT JOURS pour chaque volume en usage : un
#   snapshot unique figé dans le temps ne suffit donc pas, et ce module n'est complet
#   qu'associé à une planification (exo CLI, ordonnanceur, ou réapplication
#   périodique). Miroir NON conforme : examples/exoscale/terraform/storage.tf (volume
#   sans aucun snapshot).
# Source : references/docs/exoscale/product-storage-block-storage-how-to-snapshot.md ;
#          references/docs/exoscale/reference-terraform-provider-resources-block-storage-volume-snapshot.md ;
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
}

resource "exoscale_block_storage_volume_snapshot" "donnees" {
  zone = local.zone
  name = "donnees-applicatives-snapshot"

  volume = {
    id = exoscale_block_storage_volume.donnees.id
  }
}

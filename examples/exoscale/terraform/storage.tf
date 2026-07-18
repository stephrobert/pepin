# Block storage Exoscale — code volontairement NON CONFORME (CLD-STO-3).
# Schéma réel : exoscale_block_storage_volume {name, size, zone, ...}.
# NB : CLD-STO-2 (snapshot partagé publiquement) est NON APPLICABLE chez Exoscale
# — les snapshots block storage ne sont ni exportables ni partageables publiquement.
# Source : community.exoscale.com/product/storage/block-storage/operation/snapshot/

# ✗ Volume de données SANS aucun snapshot/sauvegarde → restauration non garantie
#   → blockstorage_volume_snapshots_exist (CLD-STO-3).
resource "exoscale_block_storage_volume" "data" {
  zone = local.zone
  name = "pepin-test-data"
  size = 10
}

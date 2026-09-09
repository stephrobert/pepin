# Calcul — quatre instances appliquées (le QUOTA de l'organisation est de quatre
# instances, `GET /quota` : instance 4), une cinquième sur le plan seulement, deux
# volumes block storage, une snapshot.
#
#   exposed   IP publique (défaut) + SG SSH ouvert        → compute_instance_public_ip_with_open_securitygroup
#   untagged  sans étiquettes de gouvernance              → governance_resource_required_tags ;
#             et, sans IP publique (`private = true`) derrière le même SG SSH ouvert,
#             contre-exemple de CLD-NET-3 : le groupe ouvert ne suffit pas, il faut l'IP
#   secrets   cloud-init avec un mot de passe               → compute_instance_no_secrets_in_user_data ;
#             porte le volume `unsnapshotted`, dont la faute est la sienne (CLD-STO-3)
#   hardened  IP publique + SG restrictif, étiquetée, cloud-init anodin, volume
#             `snapshotted` (avec sa snapshot)             → silencieuse partout
#   swiss     en ch-gva-2 (Suisse : hors UE, espace de confiance) → governance_resource_region_in_eu
#             (low) — sur le PLAN seulement (`terraform_only_resources`) : un scan live ne
#             lit qu'une zone (de-fra-1), elle y serait invisible, et elle compterait
#             dans le quota
#
# Volumes : `unsnapshotted` attaché à `secrets` → blockstorage_volume_snapshots_exist ;
# `snapshotted` attaché à `hardened`, avec une snapshot → contre-exemple. Une
# instance qui porte un volume doit être au moins standard.small (provider #375).
# L'ordre de destruction (instance, puis volume) est celui des dépendances : le
# provider ne détache pas avant de supprimer (#453), et n'a pas à le faire ici.

data "exoscale_template" "main" {
  zone = var.zone
  name = var.template_name
}

data "exoscale_template" "trusted" {
  zone = var.trusted_zone
  name = var.template_name
}

locals {
  cloud_init_plain = <<-EOT
    #cloud-config
    package_update: false
    timezone: Europe/Paris
  EOT

  cloud_init_with_secret = <<-EOT
    #cloud-config
    users:
      - name: pepin
        plain_text_passwd: pepin-qualification-fake-password
        lock_passwd: false
  EOT
}

# ÉCART compute_instance_public_ip_with_open_securitygroup (CLD-NET-3) : IP publique
# par défaut, groupe SSH ouvert.
resource "exoscale_compute_instance" "exposed" {
  zone               = var.zone
  name               = "pepin-qual-vm-exposed"
  template_id        = data.exoscale_template.main.id
  type               = var.instance_type
  disk_size          = 10
  security_group_ids = [exoscale_security_group.ssh_open.id]
  user_data          = local.cloud_init_plain
  labels             = local.tagged
}

# ÉCART governance_resource_required_tags (CLD-GVN-1) : aucune étiquette de
# gouvernance. Et CONTRE-EXEMPLE de CLD-NET-3 : même groupe SSH ouvert qu'`exposed`,
# mais aucune IP publique.
resource "exoscale_compute_instance" "untagged" {
  zone               = var.zone
  name               = "pepin-qual-vm-untagged"
  template_id        = data.exoscale_template.main.id
  type               = var.instance_type
  disk_size          = 10
  private            = true
  security_group_ids = [exoscale_security_group.ssh_open.id]
  user_data          = local.cloud_init_plain
  labels             = local.untagged
}

# ÉCART compute_instance_no_secrets_in_user_data (CLD-CMP-9) : un mot de passe
# dans le cloud-init. Porte le volume `unsnapshotted` (la faute est celle du volume).
resource "exoscale_compute_instance" "secrets" {
  zone                     = var.zone
  name                     = "pepin-qual-vm-secrets"
  template_id              = data.exoscale_template.main.id
  type                     = var.instance_type_with_volume
  disk_size                = 10
  security_group_ids       = [exoscale_security_group.hardened.id]
  block_storage_volume_ids = [exoscale_block_storage_volume.unsnapshotted.id]
  user_data                = local.cloud_init_with_secret
  labels                   = local.tagged
}

# ÉCART governance_resource_region_in_eu (CLD-GVN-3, mineur) : ch-gva-2. Sur le plan
# seulement : hors de la zone scannée, et l'organisation n'a que quatre instances.
resource "exoscale_compute_instance" "swiss" {
  count              = var.terraform_only_resources ? 1 : 0
  zone               = var.trusted_zone
  name               = "pepin-qual-vm-swiss"
  template_id        = data.exoscale_template.trusted.id
  type               = var.instance_type
  disk_size          = 10
  security_group_ids = [exoscale_security_group.hardened.id]
  user_data          = local.cloud_init_plain
  labels             = local.tagged
}

resource "exoscale_compute_instance" "hardened" {
  zone                     = var.zone
  name                     = "pepin-qual-vm-hardened"
  template_id              = data.exoscale_template.main.id
  type                     = var.instance_type_with_volume
  disk_size                = 10
  security_group_ids       = [exoscale_security_group.hardened.id]
  block_storage_volume_ids = [exoscale_block_storage_volume.snapshotted.id]
  user_data                = local.cloud_init_plain
  labels                   = local.tagged
}

# ÉCART blockstorage_volume_snapshots_exist (CLD-STO-3) : attaché (à `secrets`),
# jamais sauvegardé. Chiffré par construction (mapping `encrypted: true`, CLD-CHF-2).
resource "exoscale_block_storage_volume" "unsnapshotted" {
  zone   = var.zone
  name   = "pepin-qual-vol-unsnapshotted"
  size   = 10
  labels = local.tagged
}

# CONTRE-EXEMPLE : attaché, avec une snapshot fraîche et terminée.
resource "exoscale_block_storage_volume" "snapshotted" {
  zone   = var.zone
  name   = "pepin-qual-vol-snapshotted"
  size   = 10
  labels = local.tagged
}

resource "exoscale_block_storage_volume_snapshot" "private" {
  zone   = var.zone
  name   = "pepin-qual-snap-private"
  labels = local.tagged
  volume = {
    id = exoscale_block_storage_volume.snapshotted.id
  }
}

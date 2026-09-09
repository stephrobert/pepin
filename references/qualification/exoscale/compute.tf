# Calcul — quatre instances (plan seul : rien n'est créé).
#
#   untagged  sans étiquettes de gouvernance              → governance_resource_required_tags
#   secrets   cloud-init avec un mot de passe               → compute_instance_no_secrets_in_user_data
#   swiss     en ch-gva-2 (Suisse : hors UE, espace de confiance) → governance_resource_region_in_eu (low)
#   hardened  étiquetée, en zone UE, cloud-init anodin       → silencieuse
#
# L'exposition (CLD-NET-3) ne se juge pas sur un plan : `public_ip_address` est
# calculée à l'apply. Elle attend la moitié live de ce tenant.

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

resource "exoscale_compute_instance" "untagged" {
  zone               = var.zone
  name               = "pepin-qual-vm-untagged"
  template_id        = var.template_id
  type               = var.instance_type
  disk_size          = 10
  security_group_ids = [exoscale_security_group.hardened.id]
  user_data          = local.cloud_init_plain
  labels             = local.untagged
}

resource "exoscale_compute_instance" "secrets" {
  zone               = var.zone
  name               = "pepin-qual-vm-secrets"
  template_id        = var.template_id
  type               = var.instance_type
  disk_size          = 10
  security_group_ids = [exoscale_security_group.hardened.id]
  user_data          = local.cloud_init_with_secret
  labels             = local.tagged
}

resource "exoscale_compute_instance" "swiss" {
  zone               = var.trusted_zone
  name               = "pepin-qual-vm-swiss"
  template_id        = var.template_id
  type               = var.instance_type
  disk_size          = 10
  security_group_ids = [exoscale_security_group.hardened.id]
  user_data          = local.cloud_init_plain
  labels             = local.tagged
}

resource "exoscale_compute_instance" "hardened" {
  zone               = var.zone
  name               = "pepin-qual-vm-hardened"
  template_id        = var.template_id
  type               = var.instance_type
  disk_size          = 10
  security_group_ids = [exoscale_security_group.hardened.id]
  user_data          = local.cloud_init_plain
  labels             = local.tagged
}

# Volume block storage (chiffré par construction chez Exoscale : le mapping le
# déclare `encrypted: true`, CLD-CHF-2 conforme). Son état d'usage n'est connu qu'à
# l'apply : CLD-STO-3 attend la moitié live.
resource "exoscale_block_storage_volume" "data" {
  zone   = var.zone
  name   = "pepin-qual-vol-data"
  size   = 10
  labels = local.tagged
}

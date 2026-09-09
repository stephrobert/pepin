# Calcul — huit VM tinav6.c2r4p2, quatre IP publiques, deux cartes secondaires,
# deux volumes de données, deux snapshots, deux images.
#
# Chaque VM ne porte qu'UNE faute, ou aucune :
#   exposed           IP publique + SG SSH ouvert         → compute_instance_public_ip_with_open_securitygroup
#   private           pas d'IP   + SG SSH ouvert         → silencieuse (pas d'exposition)
#   hardened          IP publique + SG restrictif, Env=prod, PROTÉGÉE → silencieuse (SG discriminant, protection présente)
#   untagged          pas d'IP   + SG restrictif, sans étiquettes → governance_resource_required_tags
#   secrets           cloud-init avec mot de passe       → compute_instance_no_secrets_in_user_data
#   unprotected       Env=prod, sans protection          → compute_instance_deletion_protection
#   two_nics          carte primaire privée+fermée, carte SECONDAIRE publique+SSH ouvert → CLD-NET-3 (audit #E, #187)
#   two_nics_inverse  carte primaire publique+fermée, carte secondaire privée+SSH ouvert → silencieuse (l'inverse ne doit pas parler)
#
# Volumes racine : chaque VM en a un, en usage, sans snapshot → ils portent TOUS
# l'écart blockstorage_volume_snapshots_exist. Le volume de données `snapshotted`
# est le contre-exemple ; `unsnapshotted` la faute isolée.
#
# Cartes secondaires : `outscale_nic` + `outscale_nic_link`, jamais les blocs
# `nics`/`primary_nic` d'`outscale_vm` (issues #778, #424, #50, #448 du provider :
# remplacement systématique de la VM, carte « oubliée »). Aucune
# `outscale_nic_private_ip` (#28, #137 : 400/409 au destroy, ouvertes).

locals {
  sg_ids = {
    ssh_open = outscale_security_group.ssh_open.security_group_id
    hardened = outscale_security_group.hardened.security_group_id
  }

  cloud_init_plain = <<-EOT
    #cloud-config
    package_update: false
    timezone: Europe/Paris
  EOT

  # Un mot de passe en clair dans le cloud-init : motif « mot de passe en clair »
  # (confiance heuristique). La valeur est évidemment factice.
  cloud_init_with_secret = <<-EOT
    #cloud-config
    users:
      - name: pepin
        plain_text_passwd: pepin-qualification-fake-password
        lock_passwd: false
  EOT

  vms = {
    exposed = { sg = "ssh_open", tags = local.tagged, user_data = local.cloud_init_plain, protect = false }
    private = { sg = "ssh_open", tags = local.tagged, user_data = local.cloud_init_plain, protect = false }
    # PROTÉGÉE : levée par le runner avant le destroy (pre_destroy_vars, provider #88).
    hardened    = { sg = "hardened", tags = local.production_tagged, user_data = local.cloud_init_plain, protect = var.deletion_protection }
    untagged    = { sg = "hardened", tags = local.untagged, user_data = local.cloud_init_plain, protect = false }
    secrets     = { sg = "hardened", tags = local.tagged, user_data = local.cloud_init_with_secret, protect = false }
    unprotected = { sg = "hardened", tags = local.production_tagged, user_data = local.cloud_init_plain, protect = false }
  }
}

resource "outscale_vm" "public" {
  for_each                 = local.vms
  image_id                 = var.image_id
  vm_type                  = var.instance_type
  placement_subregion_name = var.subregion
  subnet_id                = outscale_subnet.main.subnet_id
  security_group_ids       = [local.sg_ids[each.value.sg]]
  deletion_protection      = each.value.protect
  user_data                = base64encode(each.value.user_data)

  dynamic "tags" {
    for_each = merge(each.value.tags, { Name = "pepin-qual-vm-${each.key}" })
    content {
      key   = tags.key
      value = tags.value
    }
  }
}

# IP publiques (dans le Net, liées par identifiant) : `exposed` (la faute) et `hardened` (le contre-exemple).
resource "outscale_public_ip" "public" {
  for_each = toset(["exposed", "hardened"])

  dynamic "tags" {
    for_each = merge(local.untagged, { Name = "pepin-qual-ip-${each.key}" })
    content {
      key   = tags.key
      value = tags.value
    }
  }
}

# Même course possible sur les VM sans seconde carte : le lien attend le lien de
# volume de `private`/`hardened`, qui n'est accepté qu'une fois la VM créée.
resource "outscale_public_ip_link" "public" {
  for_each     = toset(["exposed", "hardened"])
  public_ip_id = outscale_public_ip.public[each.key].public_ip_id
  vm_id        = outscale_vm.public[each.key].vm_id
  depends_on   = [outscale_internet_service_link.documented, outscale_volume_link.snapshotted, outscale_volume_link.unsnapshotted]
}

# ── Volumes, snapshots ─────────────────────────────────────────────────────────

# ÉCART blockstorage_volume_snapshots_exist (CLD-STO-3) : en usage, jamais sauvegardé.
resource "outscale_volume" "unsnapshotted" {
  subregion_name = var.subregion
  size           = 10
  volume_type    = "standard"

  dynamic "tags" {
    for_each = merge(local.tagged, { Name = "pepin-qual-vol-unsnapshotted" })
    content {
      key   = tags.key
      value = tags.value
    }
  }
}

resource "outscale_volume_link" "unsnapshotted" {
  device_name = "/dev/xvdb"
  volume_id   = outscale_volume.unsnapshotted.volume_id
  vm_id       = outscale_vm.public["private"].vm_id
}

# CONTRE-EXEMPLE : en usage, avec une snapshot fraîche et terminée.
resource "outscale_volume" "snapshotted" {
  subregion_name = var.subregion
  size           = 10
  volume_type    = "standard"

  dynamic "tags" {
    for_each = merge(local.tagged, { Name = "pepin-qual-vol-snapshotted" })
    content {
      key   = tags.key
      value = tags.value
    }
  }
}

resource "outscale_volume_link" "snapshotted" {
  device_name = "/dev/xvdb"
  volume_id   = outscale_volume.snapshotted.volume_id
  vm_id       = outscale_vm.public["hardened"].vm_id
}

resource "outscale_snapshot" "private" {
  volume_id   = outscale_volume.snapshotted.volume_id
  description = "pepin-qual-snap-private"

  dynamic "tags" {
    for_each = merge(local.tagged, { Name = "pepin-qual-snap-private" })
    content {
      key   = tags.key
      value = tags.value
    }
  }
}

# ÉCART blockstorage_snapshot_not_public (CLD-STO-2) : partagée avec tout le monde.
resource "outscale_snapshot" "public" {
  volume_id   = outscale_volume.snapshotted.volume_id
  description = "pepin-qual-snap-public"

  dynamic "tags" {
    for_each = merge(local.tagged, { Name = "pepin-qual-snap-public" })
    content {
      key   = tags.key
      value = tags.value
    }
  }
}

resource "outscale_snapshot_attributes" "public" {
  snapshot_id = outscale_snapshot.public.snapshot_id

  permissions_to_create_volume_additions {
    global_permission = true
  }
}

# ── Images (OMI) ───────────────────────────────────────────────────────────────

# ÉCART compute_image_not_public (CLD-STO-2) : OMI rendue publique. Construite
# depuis la snapshot `private` (gérée par Terraform) et non depuis une VM : mesuré,
# une OMI créée depuis une VM prend une snapshot IMPLICITE que `DeleteImage` ne
# supprime pas — deux snapshots orphelines après un destroy réussi, attrapées par le
# delta de la preuve de destruction. Depuis une snapshot gérée, rien d'implicite.
resource "outscale_image" "public" {
  image_name       = "pepin-qual-omi-public"
  root_device_name = "/dev/sda1"

  block_device_mappings {
    device_name = "/dev/sda1"
    bsu {
      snapshot_id           = outscale_snapshot.private.snapshot_id
      delete_on_vm_deletion = true
    }
  }

  dynamic "tags" {
    for_each = merge(local.tagged, { Name = "pepin-qual-omi-public" })
    content {
      key   = tags.key
      value = tags.value
    }
  }
}

resource "outscale_image_launch_permission" "public" {
  image_id = outscale_image.public.image_id

  permission_additions {
    global_permission = "true"
  }
}

# CONTRE-EXEMPLE : la même image, privée.
resource "outscale_image" "private" {
  image_name       = "pepin-qual-omi-private"
  root_device_name = "/dev/sda1"

  block_device_mappings {
    device_name = "/dev/sda1"
    bsu {
      snapshot_id           = outscale_snapshot.private.snapshot_id
      delete_on_vm_deletion = true
    }
  }

  dynamic "tags" {
    for_each = merge(local.tagged, { Name = "pepin-qual-omi-private" })
    content {
      key   = tags.key
      value = tags.value
    }
  }
}

# ── Les deux VM à deux cartes, dans le Net ─────────────────────────────────────

# ÉCART CLD-NET-3 par la carte SECONDAIRE : primaire privée dans un groupe fermé,
# secondaire avec une IP publique dans le groupe SSH ouvert. C'est le cas que
# l'aplatissement des cartes rendait invisible (#170, #187).
resource "outscale_vm" "two_nics" {
  image_id                 = var.image_id
  vm_type                  = var.instance_type
  placement_subregion_name = var.subregion
  subnet_id                = outscale_subnet.main.subnet_id
  security_group_ids       = [outscale_security_group.net_closed.security_group_id]
  deletion_protection      = false
  user_data                = base64encode(local.cloud_init_plain)

  dynamic "tags" {
    for_each = merge(local.tagged, { Name = "pepin-qual-vm-two-nics" })
    content {
      key   = tags.key
      value = tags.value
    }
  }
}

resource "outscale_nic" "two_nics_secondary" {
  subnet_id          = outscale_subnet.main.subnet_id
  security_group_ids = [outscale_security_group.net_ssh_open.security_group_id]

  dynamic "tags" {
    for_each = merge(local.untagged, { Name = "pepin-qual-nic-two-nics" })
    content {
      key   = tags.key
      value = tags.value
    }
  }
}

resource "outscale_nic_link" "two_nics_secondary" {
  device_number = 1
  vm_id         = outscale_vm.two_nics.vm_id
  nic_id        = outscale_nic.two_nics_secondary.nic_id
}

resource "outscale_public_ip" "two_nics" {
  dynamic "tags" {
    for_each = merge(local.untagged, { Name = "pepin-qual-ip-two-nics" })
    content {
      key   = tags.key
      value = tags.value
    }
  }
}

resource "outscale_public_ip_link" "two_nics" {
  public_ip_id = outscale_public_ip.two_nics.public_ip_id
  nic_id       = outscale_nic.two_nics_secondary.nic_id
  depends_on   = [outscale_nic_link.two_nics_secondary, outscale_internet_service_link.documented]
}

# CONTRE-EXEMPLE (ligne 2 de la table de #187) : primaire PUBLIQUE dans un groupe
# fermé, secondaire privée dans le groupe SSH ouvert. Aplaties, ces deux cartes
# diraient « publique et ouverte » ; appariées, rien n'est joignable.
resource "outscale_vm" "two_nics_inverse" {
  image_id                 = var.image_id
  vm_type                  = var.instance_type
  placement_subregion_name = var.subregion
  subnet_id                = outscale_subnet.main.subnet_id
  security_group_ids       = [outscale_security_group.net_closed.security_group_id]
  deletion_protection      = false
  user_data                = base64encode(local.cloud_init_plain)

  dynamic "tags" {
    for_each = merge(local.tagged, { Name = "pepin-qual-vm-two-nics-inverse" })
    content {
      key   = tags.key
      value = tags.value
    }
  }
}

resource "outscale_public_ip" "two_nics_inverse" {
  dynamic "tags" {
    for_each = merge(local.untagged, { Name = "pepin-qual-ip-two-nics-inverse" })
    content {
      key   = tags.key
      value = tags.value
    }
  }
}

# Lié AVANT la carte secondaire, et c'est l'ordre qui compte : mesuré (trois runs),
# `LinkPublicIp` par `VmId` sur une VM qui porte déjà DEUX cartes répond « Unable to
# link Public IP » (400 InvalidResource) — l'API ne sait pas laquelle viser. Tant que
# la VM n'a que sa carte primaire, le lien par `VmId` la désigne sans ambiguïté ; la
# seconde carte est attachée ensuite (le lien de carte dépend de celui-ci).
resource "outscale_public_ip_link" "two_nics_inverse" {
  public_ip_id = outscale_public_ip.two_nics_inverse.public_ip_id
  vm_id        = outscale_vm.two_nics_inverse.vm_id
  depends_on   = [outscale_internet_service_link.documented]
}

resource "outscale_nic" "two_nics_inverse_secondary" {
  subnet_id          = outscale_subnet.main.subnet_id
  security_group_ids = [outscale_security_group.net_ssh_open.security_group_id]

  dynamic "tags" {
    for_each = merge(local.untagged, { Name = "pepin-qual-nic-two-nics-inverse" })
    content {
      key   = tags.key
      value = tags.value
    }
  }
}

resource "outscale_nic_link" "two_nics_inverse_secondary" {
  device_number = 1
  vm_id         = outscale_vm.two_nics_inverse.vm_id
  nic_id        = outscale_nic.two_nics_inverse_secondary.nic_id
  depends_on    = [outscale_public_ip_link.two_nics_inverse]
}

# Les volumes RACINE, créés avec les VM, naissent sans étiquette : `outscale_tag` leur
# pose la gouvernance et le tag du tenant, pour que la seule ressource sans étiquette
# soit la VM `untagged`, et que le listing de sortie les voie tous.
resource "outscale_tag" "root_volumes" {
  resource_ids = concat(
    [for vm in outscale_vm.public : tolist(vm.block_device_mappings_created[0].bsu)[0].volume_id],
    [
      tolist(outscale_vm.two_nics.block_device_mappings_created[0].bsu)[0].volume_id,
      tolist(outscale_vm.two_nics_inverse.block_device_mappings_created[0].bsu)[0].volume_id,
    ],
  )

  dynamic "tag" {
    for_each = local.tagged
    content {
      key   = tag.key
      value = tag.value
    }
  }
}

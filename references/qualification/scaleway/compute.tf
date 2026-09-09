# Calcul — cinq serveurs DEV1-S (0,009 €/h chacun) et deux IP flexibles.
#
# Chaque serveur ne porte qu'UNE faute, ou aucune :
#   exposed   IP publique + SG SSH ouvert      → compute_instance_public_ip_with_open_securitygroup
#   private   pas d'IP   + SG SSH ouvert      → silencieux (contre-exemple : pas d'exposition)
#   hardened  IP publique + SG restrictif      → silencieux (contre-exemple : SG discriminant)
#   untagged  pas d'IP   + SG restrictif, sans étiquettes → governance_resource_required_tags
#   secrets   pas d'IP   + SG restrictif, cloud-init avec mot de passe → compute_instance_no_secrets_in_user_data
#
# TOUS portent un cloud-init : l'attribut `user_data` doit être présent sur chaque
# serveur pour que le verrou de capacité (ADR-0006, intersection par type) laisse
# le contrôle des secrets conclure « pass » sur ceux qui n'en ont pas. Seul
# `secrets` porte un secret ; les autres portent un cloud-config anodin, qui est le
# contre-exemple du détecteur.
#
# Volume racine : stockage LOCAL de la gamme (l_ssd), détruit avec le serveur
# (`delete_on_termination = true`, écrit plutôt que présumé). Pas de volume Block
# additionnel : l'API Instance ne crée plus de b_ssd, et un volume Block orphelin
# est exactement le reste qu'on ne retrouve pas (issue #2853, corrigée en 2.48).
#
# Volontairement ABSENT : toute interface privée (`scaleway_instance_private_nic`),
# dont la destruction est cassée depuis le provider 2.81.0 (#4338). Le cas d'une
# machine à deux cartes (audit #E) ne peut donc pas être exercé sur Scaleway
# aujourd'hui, et le README le consigne.

locals {
  cloud_init_plain = <<-EOT
    #cloud-config
    package_update: false
    timezone: Europe/Paris
  EOT

  # Un mot de passe en clair dans le cloud-init : détecté par le motif
  # « mot de passe en clair » (confiance heuristique) de la règle commune. La
  # valeur est évidemment factice — elle n'ouvre rien, nulle part.
  cloud_init_with_secret = <<-EOT
    #cloud-config
    users:
      - name: pepin
        plain_text_passwd: pepin-qualification-fake-password
        lock_passwd: false
  EOT
}

resource "scaleway_instance_ip" "exposed" {
  tags = local.untagged
}

resource "scaleway_instance_ip" "hardened" {
  tags = local.untagged
}

# ÉCART compute_instance_public_ip_with_open_securitygroup (CLD-NET-3) : IP publique
# ET SG qui ouvre SSH à Internet → sévérité high (port sensible), sujet = id serveur.
resource "scaleway_instance_server" "exposed" {
  name              = "pepin-qual-vm-exposed"
  type              = var.instance_type
  image             = "ubuntu_jammy"
  ip_id             = scaleway_instance_ip.exposed.id
  security_group_id = scaleway_instance_security_group.ssh_open.id
  tags              = local.tagged
  user_data         = { cloud-init = local.cloud_init_plain }

  root_volume {
    delete_on_termination = true
  }
}

# CONTRE-EXEMPLE : même SG ouvert, mais AUCUNE IP publique → pas d'exposition.
resource "scaleway_instance_server" "private" {
  name              = "pepin-qual-vm-private"
  type              = var.instance_type
  image             = "ubuntu_jammy"
  security_group_id = scaleway_instance_security_group.ssh_open.id
  tags              = local.tagged
  user_data         = { cloud-init = local.cloud_init_plain }

  root_volume {
    delete_on_termination = true
  }
}

# CONTRE-EXEMPLE : IP publique, mais SG restrictif → pas d'exposition.
resource "scaleway_instance_server" "hardened" {
  name              = "pepin-qual-vm-hardened"
  type              = var.instance_type
  image             = "ubuntu_jammy"
  ip_id             = scaleway_instance_ip.hardened.id
  security_group_id = scaleway_instance_security_group.hardened.id
  tags              = local.tagged
  user_data         = { cloud-init = local.cloud_init_plain }

  root_volume {
    delete_on_termination = true
  }
}

# ÉCART governance_resource_required_tags (CLD-GVN-1) : aucune étiquette de
# gouvernance (seul le tag du tenant, qui sert à la preuve de destruction).
resource "scaleway_instance_server" "untagged" {
  name              = "pepin-qual-vm-untagged"
  type              = var.instance_type
  image             = "ubuntu_jammy"
  security_group_id = scaleway_instance_security_group.hardened.id
  tags              = local.untagged
  user_data         = { cloud-init = local.cloud_init_plain }

  root_volume {
    delete_on_termination = true
  }
}

# ÉCART compute_instance_no_secrets_in_user_data (CLD-CMP-9) : observable sur le
# PLAN (la collecte live ne lit pas GetServerUserData, contrat `a_verifier`).
resource "scaleway_instance_server" "secrets" {
  name              = "pepin-qual-vm-secrets"
  type              = var.instance_type
  image             = "ubuntu_jammy"
  security_group_id = scaleway_instance_security_group.hardened.id
  tags              = local.tagged
  user_data         = { cloud-init = local.cloud_init_with_secret }

  root_volume {
    delete_on_termination = true
  }
}

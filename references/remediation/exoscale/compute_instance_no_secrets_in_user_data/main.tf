# Remédiation CONFORME : compute_instance_no_secrets_in_user_data (module autonome).
# Exigence : CLD-CMP-9 (aucun secret en clair dans les données utilisateur).
# Fournisseur : exoscale (exoscale_compute_instance, exoscale_ssh_key).
# Pourquoi conforme : le cloud-init ne porte AUCUNE valeur secrète. L'accès se fait par
#   clé SSH déclarée comme ressource, et le secret applicatif est lu au démarrage
#   depuis le coffre, jamais écrit dans le user-data. Les données utilisateur sont
#   lisibles par quiconque atteint les métadonnées de l'instance, et un SSRF suffit à
#   les exfiltrer. Miroir NON conforme : examples/exoscale/terraform/main.tf
#   (mot de passe et clé d'accès en clair dans le cloud-init).
# Source : references/docs/exoscale/product-compute-instances-how-to-cloud-init-user-data.md ;
#          references/docs/exoscale/product-compute-instances-how-to-ssh-keypairs.md ;
#          contrat providers/exoscale.yaml (mapping compute_instance : user_data).
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

data "exoscale_template" "ubuntu" {
  zone = local.zone
  name = "Linux Ubuntu 24.04 LTS 64-bit"
}

variable "cle_publique_exploitation" {
  type        = string
  description = "Clé publique SSH de l'équipe d'exploitation (jamais la clé privée)"
}

resource "exoscale_ssh_key" "exploitation" {
  name       = "exploitation"
  public_key = var.cle_publique_exploitation
}

resource "exoscale_security_group" "applicatif" {
  name        = "applicatif-sans-secret"
  description = "Charge applicative dont le secret vient du coffre"
}

resource "exoscale_compute_instance" "applicatif" {
  name        = "srv-sans-secret"
  zone        = local.zone
  type        = "standard.medium"
  template_id = data.exoscale_template.ubuntu.id
  disk_size   = 50
  ssh_key     = exoscale_ssh_key.exploitation.name

  security_group_ids = [exoscale_security_group.applicatif.id]

  # Le user-data ne contient que du non-sensible : l'agent lit le secret au
  # démarrage auprès du coffre, en s'authentifiant par l'identité de l'instance.
  user_data = <<-EOT
    #cloud-config
    package_update: true
    packages:
      - vault
    runcmd:
      - [ systemctl, enable, --now, vault-agent ]
  EOT
}

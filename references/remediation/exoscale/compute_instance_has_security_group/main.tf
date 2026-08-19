# Remédiation CONFORME : compute_instance_has_security_group (module autonome).
# Exigence : CLD-CMP-1 (aucune instance sans filtrage réseau).
# Fournisseur : exoscale (exoscale_compute_instance, exoscale_security_group).
# Pourquoi conforme : l'instance référence explicitement un security group restrictif.
#   Chez Exoscale le refus est le défaut et une règle est une autorisation : un groupe
#   attaché sans règle entrante ouverte ne laisse rien passer. La règle Pépin se
#   déclenche sur une instance dont security_group_ids est COLLECTÉ et vide.
# Source : references/docs/exoscale/product-networking-security-group-overview.md ;
#          schéma exoscale_compute_instance (security_group_ids) ;
#          contrat providers/exoscale.yaml.
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

resource "exoscale_security_group" "applicatif" {
  name        = "applicatif"
  description = "Filtrage de la charge applicative, refus par défaut"
}

resource "exoscale_security_group_rule" "https_depuis_frontal" {
  security_group_id = exoscale_security_group.applicatif.id
  description       = "HTTPS depuis le réseau d'entreprise"
  type              = "INGRESS"
  protocol          = "TCP"
  start_port        = 443
  end_port          = 443
  cidr              = "198.51.100.0/24"
}

resource "exoscale_compute_instance" "applicatif" {
  name        = "srv-applicatif"
  zone        = local.zone
  type        = "standard.medium"
  template_id = data.exoscale_template.ubuntu.id
  disk_size   = 50

  security_group_ids = [exoscale_security_group.applicatif.id]
}

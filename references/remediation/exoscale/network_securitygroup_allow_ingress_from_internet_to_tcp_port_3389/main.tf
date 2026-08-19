# Remédiation CONFORME : network_securitygroup_allow_ingress_from_internet_to_tcp_port_3389
# (module autonome).
# Exigence : CLD-NET-1 (port d'administration jamais ouvert à Internet).
# Fournisseur : exoscale (exoscale_security_group, exoscale_security_group_rule).
# Pourquoi conforme : RDP n'est joignable que depuis le CIDR d'administration, jamais
#   depuis 0.0.0.0/0. Miroir NON conforme : examples/exoscale/terraform/main.tf
#   (règle « rdp » ouverte à 0.0.0.0/0).
# Source : references/docs/exoscale/product-networking-security-group-how-to.md ;
#          schéma exoscale_security_group_rule ; contrat providers/exoscale.yaml.
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

resource "exoscale_security_group" "windows_admin" {
  name        = "windows-admin"
  description = "Serveurs Windows d'administration"
}

resource "exoscale_security_group_rule" "rdp_depuis_admin" {
  security_group_id = exoscale_security_group.windows_admin.id
  description       = "RDP d'administration depuis le réseau d'entreprise"
  type              = "INGRESS"
  protocol          = "TCP"
  start_port        = 3389
  end_port          = 3389
  cidr              = "198.51.100.0/24"
}

# Remédiation CONFORME : network_securitygroup_allow_ingress_from_internet_to_tcp_port_22
# (module autonome).
# Exigence : CLD-NET-1 (port d'administration jamais ouvert à Internet).
# Fournisseur : exoscale (exoscale_security_group, exoscale_security_group_rule).
# Pourquoi conforme : la règle SSH n'admet que le CIDR d'administration du bastion.
#   La règle Pépin se déclenche sur une source qui COUVRE Internet (préfixe <= 1) ;
#   un /24 documenté n'en est pas une. La description satisfait au passage la matrice
#   des flux (CLD-NET-5). Miroir NON conforme :
#   examples/exoscale/terraform/main.tf (cidr = "0.0.0.0/0" sur le port 22).
# Source : references/docs/exoscale/product-networking-security-group-how-to-allow-ssh.md ;
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

resource "exoscale_security_group" "bastion" {
  name        = "bastion"
  description = "Point d'entrée d'administration, seul exposé au réseau d'entreprise"
}

resource "exoscale_security_group_rule" "ssh_depuis_admin" {
  security_group_id = exoscale_security_group.bastion.id
  description       = "SSH d'administration depuis le réseau d'entreprise"
  type              = "INGRESS"
  protocol          = "TCP"
  start_port        = 22
  end_port          = 22
  cidr              = "198.51.100.0/24"
}

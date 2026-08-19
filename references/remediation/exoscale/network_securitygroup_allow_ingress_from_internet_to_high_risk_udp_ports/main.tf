# Remédiation CONFORME : network_securitygroup_allow_ingress_from_internet_to_high_risk_udp_ports
# (module autonome).
# Exigence : CLD-NET-1 (service UDP amplificateur jamais exposé à Internet).
# Fournisseur : exoscale (exoscale_security_group, exoscale_security_group_rule).
# Pourquoi conforme : le résolveur DNS (53/UDP) n'accepte que le plan d'adressage
#   privé du réseau interne. Un résolveur ouvert à 0.0.0.0/0 est un relais
#   d'amplification DDoS, ce que la règle Pépin cherche précisément.
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

resource "exoscale_security_group" "resolveur_dns" {
  name        = "resolveur-dns"
  description = "Résolveur DNS interne, jamais exposé au réseau public"
}

resource "exoscale_security_group_rule" "dns_depuis_reseau_prive" {
  security_group_id = exoscale_security_group.resolveur_dns.id
  description       = "DNS interne pour le plan d'adressage privé"
  type              = "INGRESS"
  protocol          = "UDP"
  start_port        = 53
  end_port          = 53
  cidr              = "10.0.0.0/16"
}

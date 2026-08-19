# Remédiation CONFORME : network_securitygroup_unrestricted_egress (module autonome).
# Exigence : CLD-NET-4 (filtrage sortant restreint aux destinations légitimes).
# Fournisseur : exoscale (exoscale_security_group, exoscale_security_group_rule).
# Pourquoi conforme : la sortie n'est pas ouverte en « tout protocole vers 0.0.0.0/0 ».
#   Elle est déclarée destination par destination : HTTPS vers le proxy sortant, DNS
#   vers les résolveurs internes. La règle Pépin ne se déclenche que sur une sortie
#   protocol = ALL vers un CIDR public.
# Source : references/docs/exoscale/product-networking-security-group-how-to-restrict-outbound-traffic.md ;
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

resource "exoscale_security_group" "sortie_maitrisee" {
  name        = "sortie-maitrisee"
  description = "Charge de travail dont la sortie passe par le proxy d'entreprise"
}

resource "exoscale_security_group_rule" "https_vers_proxy" {
  security_group_id = exoscale_security_group.sortie_maitrisee.id
  description       = "HTTPS vers le proxy sortant d'entreprise"
  type              = "EGRESS"
  protocol          = "TCP"
  start_port        = 443
  end_port          = 443
  cidr              = "198.51.100.0/24"
}

resource "exoscale_security_group_rule" "dns_vers_resolveurs" {
  security_group_id = exoscale_security_group.sortie_maitrisee.id
  description       = "DNS vers les résolveurs internes"
  type              = "EGRESS"
  protocol          = "UDP"
  start_port        = 53
  end_port          = 53
  cidr              = "10.0.0.0/16"
}

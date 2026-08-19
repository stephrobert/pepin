# Remédiation CONFORME : network_securitygroup_allow_ingress_from_internet_to_all_ports
# (module autonome).
# Exigence : CLD-NET-2 (pas de any/any entrant depuis Internet).
# Fournisseur : exoscale (exoscale_security_group, exoscale_security_group_rule).
# Pourquoi conforme : un service web PUBLIC reste légitime, mais il s'expose port par
#   port. Ici deux règles explicites (80 et 443/TCP) remplacent la règle « tout
#   protocole, tous ports » : la règle Pépin ne se déclenche que sur protocol = ALL
#   depuis un CIDR couvrant Internet, et 80/443 ne sont pas des ports sensibles.
# Source : references/docs/exoscale/product-networking-security-group-how-to-allow-http-https.md ;
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

resource "exoscale_security_group" "frontal_web" {
  name        = "frontal-web"
  description = "Frontal HTTP/HTTPS public, exposé port par port"
}

resource "exoscale_security_group_rule" "https_public" {
  security_group_id = exoscale_security_group.frontal_web.id
  description       = "HTTPS public du site"
  type              = "INGRESS"
  protocol          = "TCP"
  start_port        = 443
  end_port          = 443
  cidr              = "0.0.0.0/0"
}

resource "exoscale_security_group_rule" "http_public_redirection" {
  security_group_id = exoscale_security_group.frontal_web.id
  description       = "HTTP public, redirigé vers HTTPS par le frontal"
  type              = "INGRESS"
  protocol          = "TCP"
  start_port        = 80
  end_port          = 80
  cidr              = "0.0.0.0/0"
}

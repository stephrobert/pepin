# Remédiation CONFORME : network_securitygroup_allow_ingress_from_internet_to_high_risk_tcp_ports
# (module autonome).
# Exigence : CLD-NET-1 (service sensible jamais joignable depuis Internet).
# Fournisseur : exoscale (exoscale_security_group, exoscale_security_group_rule).
# Pourquoi conforme : le port PostgreSQL (5432) n'est ouvert qu'AU GROUPE applicatif,
#   via user_security_group_id : la règle ne porte aucun CIDR, donc aucune source
#   publique. C'est la forme la plus sûre chez Exoscale, où une règle peut désigner un
#   autre security group plutôt qu'une plage d'adresses.
# Source : references/docs/exoscale/product-networking-security-group-how-to-organizing-security-groups.md ;
#          schéma exoscale_security_group_rule (user_security_group_id) ;
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

resource "exoscale_security_group" "application" {
  name        = "application"
  description = "Serveurs applicatifs autorisés à joindre la base de données"
}

resource "exoscale_security_group" "base_de_donnees" {
  name        = "base-de-donnees"
  description = "Base PostgreSQL, joignable des seuls serveurs applicatifs"
}

resource "exoscale_security_group_rule" "postgres_depuis_application" {
  security_group_id      = exoscale_security_group.base_de_donnees.id
  description            = "PostgreSQL ouvert au seul groupe applicatif"
  type                   = "INGRESS"
  protocol               = "TCP"
  start_port             = 5432
  end_port               = 5432
  user_security_group_id = exoscale_security_group.application.id
}

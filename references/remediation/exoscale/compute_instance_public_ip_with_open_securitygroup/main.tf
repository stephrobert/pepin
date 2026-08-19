# Remédiation CONFORME : compute_instance_public_ip_with_open_securitygroup
# (module autonome).
# Exigence : CLD-NET-3 (pas d'instance publiquement joignable sans filtrage restrictif).
# Fournisseur : exoscale (exoscale_compute_instance, exoscale_nlb, exoscale_security_group).
# Pourquoi conforme : l'instance est PRIVÉE (private = true : aucune adresse publique)
#   et son groupe n'ouvre rien depuis Internet. Le trafic public entre par le
#   répartiteur de charge, seul point exposé. La règle Pépin corrèle une adresse
#   publique avec un groupe dont une règle entrante couvre Internet : ici, ni l'une ni
#   l'autre. Miroir NON conforme : examples/exoscale/terraform/main.tf (instance à IP
#   publique attachée au groupe « open »).
# Source : references/docs/exoscale/product-compute-instances-how-to-private-instances.md ;
#          references/docs/exoscale/product-networking-nlb-how-to.md ;
#          contrat providers/exoscale.yaml (mapping compute_instance : public_ip).
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

resource "exoscale_security_group" "derriere_le_repartiteur" {
  name        = "derriere-le-repartiteur"
  description = "Instances privées, joignables du seul répartiteur de charge"
}

resource "exoscale_security_group_rule" "https_depuis_repartiteur" {
  security_group_id = exoscale_security_group.derriere_le_repartiteur.id
  description       = "HTTPS depuis les adresses du répartiteur de charge"
  type              = "INGRESS"
  protocol          = "TCP"
  start_port        = 443
  end_port          = 443
  cidr              = "10.0.0.0/16"
}

resource "exoscale_compute_instance" "applicatif" {
  name        = "srv-prive"
  zone        = local.zone
  type        = "standard.medium"
  template_id = data.exoscale_template.ubuntu.id
  disk_size   = 50

  # Aucune adresse publique : l'instance n'est pas joignable depuis Internet.
  private = true

  security_group_ids = [exoscale_security_group.derriere_le_repartiteur.id]
}

resource "exoscale_nlb" "public" {
  zone        = local.zone
  name        = "frontal-public"
  description = "Seul point d'entrée public de l'application"
}

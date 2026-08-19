# Remédiation CONFORME : governance_resource_region_in_eu (module autonome).
# Exigence : CLD-GVN-3 (stockage, traitement et administration localisés dans l'UE).
# Fournisseur : exoscale (exoscale_compute_instance, exoscale_block_storage_volume).
# Pourquoi conforme : les ressources sont créées dans une zone de l'Union européenne
#   (at-vie-1, Vienne). Les zones suisses (ch-gva-2, ch-dk-2) restent dans l'espace
#   européen de confiance mais HORS UE : la règle Pépin y rend un écart mineur, ce
#   qu'un référentiel exigeant une localisation strictement UE ne tolère pas. Zones UE
#   reconnues par le référentiel : de-fra-1, de-muc-1, at-vie-1, at-vie-2, bg-sof-1,
#   hr-zag-1. Miroir NON conforme : examples/exoscale/terraform/main.tf (zone
#   ch-gva-2).
# Source : references/docs/exoscale/platform-dc-zones.md ;
#          internal/commonrules/rules/lib.rego (table des zones UE) ;
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
  name        = "applicatif-ue"
  description = "Charge applicative hébergée dans l'Union européenne"
}

resource "exoscale_compute_instance" "applicatif" {
  name        = "srv-applicatif-ue"
  zone        = local.zone
  type        = "standard.medium"
  template_id = data.exoscale_template.ubuntu.id
  disk_size   = 50

  security_group_ids = [exoscale_security_group.applicatif.id]
}

resource "exoscale_block_storage_volume" "donnees" {
  zone = local.zone
  name = "donnees-ue"
  size = 100
}

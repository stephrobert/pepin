# Remédiation CONFORME : kubernetes_cluster_control_plane_highly_available
# (module autonome).
# Exigence : CLD-K8S-2 (plan de contrôle hautement disponible).
# Fournisseur : exoscale (exoscale_sks_cluster).
# Pourquoi conforme : le cluster est créé au niveau de service « pro », le seul qui
#   porte un plan de contrôle hautement disponible chez Exoscale ; « starter » ne
#   garantit ni la haute disponibilité ni le SLA. Le descripteur dérive
#   control_plane_multi_az de service_level = "pro". Miroir NON conforme :
#   examples/exoscale/terraform/sks.tf (service_level = "starter").
# Source : references/docs/exoscale/documentation-sks-overview.md ;
#          schéma exoscale_sks_cluster (service_level) ;
#          contrat providers/exoscale.yaml (mapping kubernetes_cluster).
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

resource "exoscale_sks_cluster" "production" {
  zone          = local.zone
  name          = "sks-production"
  service_level = "pro"
  auto_upgrade  = true

  labels = {
    Owner   = "equipe-plateforme"
    Project = "socle-applicatif"
    Env     = "production"
  }
}

# Remédiation CONFORME : kubernetes_cluster_auto_upgrade_enabled (module autonome).
# Exigence : CLD-K8S-3 (mises à jour et maintenance appliquées).
# Fournisseur : exoscale (exoscale_sks_cluster, exoscale_sks_nodepool).
# Pourquoi conforme : auto_upgrade = true sur le cluster : les correctifs mineurs de
#   Kubernetes sont appliqués par la plateforme, sans intervention. Le pool de nœuds
#   est déclaré à côté pour que le montage soit déployable tel quel. Miroir NON
#   conforme : examples/exoscale/terraform/sks.tf (auto_upgrade = false).
# Source : references/docs/exoscale/product-compute-containers-how-to-lifecycle-management.md ;
#          schéma exoscale_sks_cluster (auto_upgrade) ;
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
}

resource "exoscale_sks_nodepool" "principal" {
  zone          = local.zone
  cluster_id    = exoscale_sks_cluster.production.id
  name          = "principal"
  instance_type = "standard.medium"
  size          = 3
}

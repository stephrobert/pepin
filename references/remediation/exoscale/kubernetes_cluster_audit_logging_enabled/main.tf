# Remédiation CONFORME — kubernetes_cluster_audit_logging_enabled  (module autonome).
# Exigence : CLD-LOG-1 — politique de journalisation (événements de sécurité collectés).
# Provider : exoscale (schéma exoscale_sks_cluster ; bloc audit).
# Pourquoi conforme : le cluster SKS définit un bloc `audit` avec un endpoint de
#   collecte des journaux d'audit Kubernetes. Miroir NON conforme :
#   examples/exoscale/terraform/sks.tf (aucun bloc audit).
# Source : references/docs/exoscale/product-compute-containers-how-to-kubernetes-audit.md
#          contrat providers/exoscale.yaml (kubernetes_cluster, audit_enabled).

terraform {
  required_providers {
    exoscale = { source = "exoscale/exoscale" }
  }
}

provider "exoscale" {}

variable "sks_audit_token" {
  type        = string
  sensitive   = true
  description = "Bearer token du collecteur d'audit (injecté hors du code)."
}

resource "exoscale_sks_cluster" "prod" {
  zone          = "ch-gva-2"
  name          = "prod"
  service_level = "pro"  # control plane HA (CLD-K8S-2)
  auto_upgrade  = true   # hygiène de cycle de vie (CLD-K8S-3)
  cni           = "calico"

  # ✔ Journalisation d'audit Kubernetes vers un collecteur externe.
  audit {
    endpoint        = "https://audit-collector.example.eu/sks"
    bearer_token    = var.sks_audit_token
    initial_backoff = "5s"
  }
}

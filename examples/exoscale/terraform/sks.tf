# Kubernetes managé SKS — code volontairement NON CONFORME (CLD-K8S-2/3).
# Schéma réel : exoscale_sks_cluster {service_level, auto_upgrade, version, ...}.
# NB : CLD-K8S-1 (API non exposée) est NON APPLICABLE chez Exoscale — l'endpoint
# de l'API SKS est toujours public, sans restriction d'IP (roadmap VPC). Source :
# exoscale.com/blog/exoscale-sks + openapi-v2.exoscale.com (endpoint-sks).

# ✗ Plan de contrôle NON hautement disponible (service_level "starter" : pas de
#   HA control plane ni SLA, vs "pro" = HA control plane). Source officielle :
#   community.exoscale.com/product/compute/containers/overview/ (Starter vs Pro).
#   → kubernetes_cluster_control_plane_highly_available (CLD-K8S-2).
#   Et mises à jour automatiques désactivées (auto_upgrade=false)
#   → kubernetes_cluster_auto_upgrade_enabled (CLD-K8S-3).
resource "exoscale_sks_cluster" "test" {
  zone          = local.zone
  name          = "pepin-test-sks"
  service_level = "starter"
  auto_upgrade  = false
  cni           = "calico"
}

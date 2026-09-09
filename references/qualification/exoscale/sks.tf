# SKS — deux clusters (plan seul).
#
#   weak    service_level starter (plan de contrôle non redondant), auto_upgrade
#           false, audit désactivé → trois écarts sur un sujet
#   strong  service_level pro, auto_upgrade true, audit vers un endpoint → silencieux
#
# Les trois contrôles Kubernetes actifs pour Exoscale lisent ces trois champs
# (mapping_terraform de providers/exoscale.yaml) : un cluster « faible » les porte
# tous, parce qu'un cluster par écart n'apprendrait rien de plus sur un plan.

resource "exoscale_sks_cluster" "weak" {
  zone          = var.zone
  name          = "pepin-qual-sks-weak"
  service_level = "starter"
  auto_upgrade  = false
  labels        = local.tagged

  audit {
    enabled  = false
    endpoint = ""
  }
}

resource "exoscale_sks_cluster" "strong" {
  zone          = var.zone
  name          = "pepin-qual-sks-strong"
  service_level = "pro"
  auto_upgrade  = true
  labels        = local.tagged

  audit {
    enabled      = true
    endpoint     = "https://audit.qualification.invalid/sks"
    bearer_token = "pepin-qualification-fake-token"
  }
}

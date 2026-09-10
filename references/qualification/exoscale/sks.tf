# SKS — deux clusters, sans nodepool (le plan de contrôle suffit à ce que les
# trois contrôles lisent : level, auto-upgrade, audit.endpoint).
#
#   weak    service_level starter (plan de contrôle non redondant, gratuit),
#           auto_upgrade false, audit désactivé → trois écarts sur un sujet
#   strong  service_level pro (0,055 EUR/h), auto_upgrade true, audit vers un
#           endpoint → silencieux
#
# Un cluster « faible » porte les trois écarts, parce qu'un cluster par écart
# n'apprendrait rien de plus et coûterait un plan de contrôle Pro de plus.
#
# `create_default_security_group = false` : sans nodepool, le groupe de sécurité
# ad hoc que le provider créerait ne servirait à rien, et c'est une ressource de
# plus à prouver détruite. Le provider a laissé des restes derrière des clusters
# (#210 : un NLB créé par le CCM survivait au destroy) : rien ici n'en crée.

resource "exoscale_sks_cluster" "weak" {
  zone                          = var.zone
  name                          = "pepin-qual-sks-weak"
  create_default_security_group = false
  service_level                 = "starter"
  auto_upgrade                  = false
  labels                        = local.tagged

  audit {
    enabled  = false
    endpoint = ""
  }
}

resource "exoscale_sks_cluster" "strong" {
  zone                          = var.zone
  name                          = "pepin-qual-sks-strong"
  create_default_security_group = false
  service_level                 = "pro"
  auto_upgrade                  = true
  labels                        = local.tagged

  audit {
    enabled      = true
    endpoint     = "https://audit.qualification.invalid/sks"
    bearer_token = "pepin-qualification-fake-token"
  }
}

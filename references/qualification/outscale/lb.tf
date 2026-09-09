# Répartiteurs de charge (LBU, 0,03 EUR/h pièce), dans le cloud public.
#
# `http` : un écouteur HTTP seul, sans journal d'accès → deux écarts sur un sujet
# (loadbalancer_ssl_listeners, loadbalancer_logging_enabled). `tcp443` : un écouteur
# TCP 443 — un candidat au passthrough TLS, que la règle refuse de trancher (statut
# non concluant, ADR-0015). Le contre-exemple du journal (bucket OOS cible +
# outscale_load_balancer_attributes) n'est pas construit : tenant.yaml le dit.
#
# Pas de politique de stickiness : leur destruction échouait (issue #663).

resource "outscale_load_balancer" "http" {
  load_balancer_name = "pepin-qual-lb-http"
  load_balancer_type = "internet-facing"
  subregion_names    = [var.subregion]

  listeners {
    backend_port           = 80
    backend_protocol       = "HTTP"
    load_balancer_port     = 80
    load_balancer_protocol = "HTTP"
  }

  dynamic "tags" {
    for_each = merge(local.tagged, { Name = "pepin-qual-lb-http" })
    content {
      key   = tags.key
      value = tags.value
    }
  }
}

resource "outscale_load_balancer" "tcp443" {
  load_balancer_name = "pepin-qual-lb-tcp443"
  load_balancer_type = "internet-facing"
  subregion_names    = [var.subregion]

  listeners {
    backend_port           = 443
    backend_protocol       = "TCP"
    load_balancer_port     = 443
    load_balancer_protocol = "TCP"
  }

  dynamic "tags" {
    for_each = merge(local.tagged, { Name = "pepin-qual-lb-tcp443" })
    content {
      key   = tags.key
      value = tags.value
    }
  }
}

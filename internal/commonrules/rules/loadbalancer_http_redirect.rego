# loadbalancer_http_redirect_to_https
#   Répartiteur de charge avec un listener HTTP (port 80) — sans redirection
#   vers HTTPS, le trafic peut transiter en clair.
# Origine : osc-policy OSC-LBU-002. SCSL : CLD-CHF-1.
# Contrat : type normalisé agnostique `load_balancer` ; listeners[]
#   (load_balancer_protocol, load_balancer_port), osc-sdk-go LoadBalancer.
package pepin.rules

import rego.v1

deny contains f if {
	some lb in resources_of_type("load_balancer")
	some l in object.get(lb.attributes, "listeners", [])
	object.get(l, "load_balancer_protocol", "") == "HTTP"
	object.get(l, "load_balancer_port", 0) == 80
	name := object.get(lb.attributes, "load_balancer_name", lb.id)
	f := {
		"code": "loadbalancer_http_redirect_to_https",
		"severity": "medium",
		"subject": name,
		"message": sprintf("LBU « %s » : listener HTTP:80 — vérifier la redirection 301 vers HTTPS.", [name]),
		"remediation": "Mettre en place une redirection 301 du listener HTTP:80 vers HTTPS.",
		"labels": {"provider": provider_of(lb), "category": "security"},
	}
}

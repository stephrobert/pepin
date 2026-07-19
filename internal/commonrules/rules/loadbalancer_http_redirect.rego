# loadbalancer_http_redirect_to_https
#   Répartiteur de charge avec un listener HTTP (port 80) dont la redirection vers
#   HTTPS est OBSERVÉE comme absente : le trafic peut transiter en clair.
# SCSL : CLD-CHF-1. Contrat : type normalisé agnostique `load_balancer` ; listeners[]
#   (load_balancer_protocol, load_balancer_port, redirect_to_https).
#
# Config effective, pas de contrôle manuel déguisé : la règle exige une garde de
# capacité `redirect_to_https` (l'API expose l'état de redirection). Sans cet
# attribut collecté, elle NE se déclenche PAS — un listener correctement redirigé
# ne peut donc pas échouer à vie (le point audité serait alors NotEvaluated).
package pepin.rules

import rego.v1

deny contains f if {
	some lb in resources_of_type("load_balancer")
	some l in object.get(lb.attributes, "listeners", [])
	object.get(l, "load_balancer_protocol", "") == "HTTP"
	object.get(l, "load_balancer_port", 0) == 80
	"redirect_to_https" in object.keys(l) # garde de capacité : état de redirection réellement collecté
	object.get(l, "redirect_to_https", true) == false
	name := object.get(lb.attributes, "load_balancer_name", lb.id)
	f := {
		"code": "loadbalancer_http_redirect_to_https",
		"severity": "medium",
		"subject": name,
		"message": sprintf("LBU « %s » : listener HTTP:80 sans redirection vers HTTPS — trafic en clair possible.", [name]),
		"remediation": "Mettre en place une redirection 301 du listener HTTP:80 vers HTTPS.",
		"labels": {"provider": provider_of(lb), "category": "security"},
	}
}

# loadbalancer_ssl_listeners
#   Répartiteur de charge (LBU) internet-facing sans listener HTTPS/SSL : le
#   trafic transite en clair.
# Origine : osc-policy OSC-LBU-001. SCSL : CLD-CHF-1.
# Contrat : type normalisé agnostique `load_balancer` ; attributs
#   load_balancer_type ("internet-facing"|"internal") et listeners[]
#   (load_balancer_protocol ∈ HTTP|HTTPS|SSL|TCP), osc-sdk-go LoadBalancer.
package pepin.rules

import rego.v1

deny contains f if {
	some lb in resources_of_type("load_balancer")
	object.get(lb.attributes, "load_balancer_type", "") == "internet-facing"
	not _has_secure_listener(lb.attributes)
	name := object.get(lb.attributes, "load_balancer_name", lb.id)
	f := {
		"code": "loadbalancer_ssl_listeners",
		"severity": "high",
		"subject": name,
		"message": sprintf("LBU « %s » internet-facing sans listener HTTPS/SSL — trafic en clair.", [name]),
		"remediation": "Ajouter un listener HTTPS/SSL (TLS ≥ 1.2) avec certificat ; rediriger le trafic en clair vers HTTPS.",
		"labels": {
			"provider": provider_of(lb),
			"category": "security",
			"message_en": sprintf("Internet-facing LBU \"%s\" has no HTTPS/SSL listener — traffic in cleartext.", [name]),
			"remediation_en": "Add an HTTPS/SSL listener (TLS 1.2 or above) with a certificate; redirect cleartext traffic to HTTPS.",
		},
	}
}

_has_secure_listener(attrs) if {
	some l in object.get(attrs, "listeners", [])
	object.get(l, "load_balancer_protocol", "") in {"HTTPS", "SSL"}
}

# TLS passthrough : un listener TCP sur un port TLS standard transporte du chiffré terminé
# au backend (configuration légitime et chiffrée de bout en bout). Le LBU ne voit pas le
# contenu : affirmer « trafic en clair » serait un faux positif — on ne conclut pas.
_has_secure_listener(attrs) if {
	some l in object.get(attrs, "listeners", [])
	object.get(l, "load_balancer_protocol", "") == "TCP"
	object.get(l, "load_balancer_port", 0) in {443, 8443}
}

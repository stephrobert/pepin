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
	not _tls_passthrough_candidate(lb.attributes)
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

# _tls_passthrough_candidate — un listener TCP sur un port TLS standard.
#
# Le commentaire d'origine disait juste et concluait faux : « affirmer trafic en
# clair serait un faux positif — on ne conclut pas », puis comptait ce listener
# comme SÉCURISÉ. Un numéro de port n'est pas un protocole : on sert ce qu'on veut
# en clair sur 443, et c'est même une pratique de contournement de pare-feu
# répandue. Compter cela comme du TLS produisait un FAUX VERT — le seul du lot, et
# le plus grave, parce qu'un faux vert ne se voit pas.
#
# La règle constate donc son incapacité à conclure, et l'assessment en fait un
# `not-evaluated` (ADR-0015). Ni « sécurisé », ni « en clair » : indéterminé.
_tls_passthrough_candidate(attrs) if {
	some l in object.get(attrs, "listeners", [])
	object.get(l, "load_balancer_protocol", "") == "TCP"
	object.get(l, "load_balancer_port", 0) in {443, 8443}
}

# ── indéterminé : un TCP sur un port TLS, dont rien ne dit s'il porte du chiffré ──
deny contains f if {
	some lb in resources_of_type("load_balancer")
	object.get(lb.attributes, "load_balancer_type", "") == "internet-facing"
	not _has_secure_listener(lb.attributes)
	_tls_passthrough_candidate(lb.attributes)
	name := object.get(lb.attributes, "load_balancer_name", lb.id)
	f := {
		"code": "loadbalancer_ssl_listeners",
		"severity": "high",
		"subject": name,
		"message": sprintf("LBU « %s » : seul un listener TCP sur port TLS standard — passthrough chiffré ou trafic en clair, l'API ne permet pas de trancher.", [name]),
		"remediation": "Confirmer la terminaison TLS côté backend, ou déclarer un listener HTTPS/SSL pour que la protection soit attestée.",
		"labels": {
			"provider": provider_of(lb),
			"category": "security",
			# La règle CONSTATE son incapacité à conclure ; l'assessment en fait un
			# `not-evaluated`. Compter ce listener comme sécurisé était un FAUX VERT
			# (ADR-0015) : un port n'est pas un protocole.
			"inconclusive": "true",
			"message_en": sprintf("LBU \"%s\": only a TCP listener on a standard TLS port — encrypted passthrough or cleartext, the API cannot tell.", [name]),
			"remediation_en": "Confirm TLS termination on the backend, or declare an HTTPS/SSL listener so the protection is attested.",
		},
	}
}

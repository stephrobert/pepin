# loadbalancer_logging_enabled
#   Répartiteur de charge (LBU) sans journal d'accès activé — pas d'investigation
#   possible après incident.
# Origine : osc-policy OSC-LBU-004. SCSL : CLD-LOG-1.
# Contrat : type normalisé agnostique `load_balancer` ; attribut
#   access_log.is_enabled (bool), osc-sdk-go LoadBalancer.AccessLog.IsEnabled.
package pepin.rules

import rego.v1

deny contains f if {
	some lb in resources_of_type("load_balancer")
	not _access_log_enabled(lb.attributes)
	name := object.get(lb.attributes, "load_balancer_name", lb.id)
	f := {
		"code": "loadbalancer_logging_enabled",
		"severity": "medium",
		"subject": name,
		"message": sprintf("LBU « %s » sans journal d'accès activé — aucune investigation possible.", [name]),
		"remediation": "Activer access_log (bucket OOS dédié, rétention configurée).",
		"labels": {"provider": provider_of(lb), "category": "compliance"},
	}
}

_access_log_enabled(attrs) if {
	object.get(object.get(attrs, "access_log", {}), "is_enabled", false) == true
}

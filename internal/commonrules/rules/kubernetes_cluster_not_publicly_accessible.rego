# kubernetes_cluster_not_publicly_accessible
#   API server du cluster managé (OKS) joignable depuis Internet : la liste
#   d'autorisation contient 0.0.0.0/0 ou ::/0 — plan de contrôle exposé.
# SCSL : CLD-K8S-1 (serveur d'API du cluster non exposé publiquement).
# Contrat : type normalisé agnostique `kubernetes_cluster` ; attribut
#   admin_whitelist ([]string) — liste des CIDR autorisés à appeler l'API.
package pepin.rules

import rego.v1

deny contains f if {
	some c in resources_of_type("kubernetes_cluster")
	some cidr in cidr_list(object.get(c.attributes, "admin_whitelist", []))
	is_public_cidr(cidr)
	name := object.get(c.attributes, "name", c.id)
	f := {
		"code": "kubernetes_cluster_not_publicly_accessible",
		"severity": "critical",
		"subject": name,
		"message": sprintf("Cluster OKS « %s » : admin_whitelist contient %s — API Kubernetes exposée à Internet.", [name, cidr]),
		"remediation": "Restreindre admin_whitelist aux CIDR d'administration (bastion, runners CI, VPN) ; supprimer 0.0.0.0/0 et ::/0.",
		"labels": {"provider": provider_of(c), "category": "security"},
	}
}

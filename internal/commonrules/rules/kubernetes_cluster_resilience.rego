# Règles de résilience / cycle de vie du cluster Kubernetes managé (OKS).
#   Type normalisé agnostique `kubernetes_cluster`. Attributs (DÉRIVÉS du
#   collecteur OKS) : control_plane_multi_az (bool), auto_upgrade (bool),
#   deletion_protection (bool).
# Origine : osc-policy OSC-OKS-002/003 (HA), OSC-OKS-005 (maintenance),
#   OSC-OKS-004 (protection suppression). SCSL : CLD-CMP-4, CLD-CMP-5.
package pepin.rules

import rego.v1

# Plan de contrôle non hautement disponible (mono-AZ).
deny contains f if {
	some c in resources_of_type("kubernetes_cluster")
	object.get(c.attributes, "control_plane_multi_az", true) == false
	name := object.get(c.attributes, "name", c.id)
	f := {
		"code": "kubernetes_cluster_control_plane_highly_available",
		"severity": "high",
		"subject": name,
		"message": sprintf("Cluster OKS « %s » : plan de contrôle non multi-AZ — perte d'une zone = interruption.", [name]),
		"remediation": "Activer un plan de contrôle multi-AZ / multi-master sur le cluster.",
		"labels": {"provider": provider_of(c), "category": "compliance"},
	}
}

# Mises à jour automatiques désactivées.
deny contains f if {
	some c in resources_of_type("kubernetes_cluster")
	object.get(c.attributes, "auto_upgrade", true) == false
	name := object.get(c.attributes, "name", c.id)
	f := {
		"code": "kubernetes_cluster_auto_upgrade_enabled",
		"severity": "medium",
		"subject": name,
		"message": sprintf("Cluster OKS « %s » : mises à jour automatiques désactivées — correctifs non appliqués.", [name]),
		"remediation": "Activer la maintenance / mise à jour automatique du cluster.",
		"labels": {"provider": provider_of(c), "category": "compliance"},
	}
}

# Protection contre la suppression désactivée.
deny contains f if {
	some c in resources_of_type("kubernetes_cluster")
	object.get(c.attributes, "deletion_protection", true) == false
	name := object.get(c.attributes, "name", c.id)
	f := {
		"code": "kubernetes_cluster_deletion_protection",
		"severity": "medium",
		"subject": name,
		"message": sprintf("Cluster OKS « %s » sans protection contre la suppression.", [name]),
		"remediation": "Activer la protection contre la suppression sur le cluster.",
		"labels": {"provider": provider_of(c), "category": "compliance"},
	}
}

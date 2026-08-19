# Journalisation d'audit du cluster Kubernetes managé.
#   Type normalisé agnostique `kubernetes_cluster`. Attribut DÉRIVÉ par le
#   collecteur : audit_enabled (bool) = présence d'un endpoint d'audit configuré.
# Ancrage Exoscale SKS : objet `audit` { enabled, endpoint, bearer-token,
#   initial-backoff } exposé en création ET en lecture (GET, bearer-token exclu).
#   Sources : openapi-v2.exoscale.com (operation-create/get-sks-cluster),
#   community.exoscale.com/product/compute/containers/how-to/kubernetes-audit/.
#   Schéma Terraform exoscale_sks_cluster : bloc `audit { enabled, endpoint }`.
# SCSL : CLD-LOG-1 (politique de journalisation : événements de sécurité collectés).
package pepin.rules

import rego.v1

# Audit Kubernetes non configuré (aucun endpoint d'audit) sur un cluster managé.
# Ne se déclenche que si le collecteur a renseigné `audit_enabled` (provider qui
# expose la capacité) ; un provider sans cet attribut ne déclenche pas la règle.
deny contains f if {
	some c in resources_of_type("kubernetes_cluster")
	not truthy(object.get(c.attributes, "audit_enabled", true))
	name := object.get(c.attributes, "name", c.id)
	f := {
		"code": "kubernetes_cluster_audit_logging_enabled",
		"severity": "medium",
		"subject": name,
		"message": sprintf("Cluster Kubernetes « %s » : journalisation d'audit désactivée — aucun endpoint d'audit configuré, un incident ne pourrait pas être investigué.", [name]),
		"remediation": "Configurer l'audit Kubernetes du cluster (endpoint de collecte) et centraliser les journaux selon la politique de rétention.",
		"labels": {"provider": provider_of(c), "category": "compliance"},
	}
}

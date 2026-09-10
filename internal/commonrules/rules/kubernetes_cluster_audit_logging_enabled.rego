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
#
# # Le faux vert que ce contrôle a produit
#
# Le défaut par ressource était `true` : attribut absent ⇒ on suppose l'audit activé.
# L'intention était de ne pas crier chez un fournisseur qui n'expose pas la capacité —
# mais l'API SKS OMET l'objet `audit` quand l'audit est coupé, et `audit_enabled` en est
# DÉRIVÉ. Un cluster sans audit n'avait donc pas l'attribut, la règle supposait
# « activé », et le rapport concluait `pass` sur le cas le plus courant : celui qu'il
# existe pour attraper.
#
# Le silence devenait un vert parce que le verrou de capacité, lui, voyait l'attribut
# collecté sur le type — un cluster voisin, audité, suffisait à le poser.
#
# La question « ce fournisseur expose-t-il la capacité » se pose donc à l'échelle de
# l'INVENTAIRE, pas de la ressource. Si un cluster porte l'attribut, le fournisseur
# l'expose, et un cluster qui ne le porte pas n'a pas d'audit. Là où aucun cluster ne le
# porte, la règle se tait toujours et c'est au verrou de dire « non évalué ».
deny contains f if {
	some c in resources_of_type("kubernetes_cluster")
	_audit_expose
	not truthy(object.get(c.attributes, "audit_enabled", false))
	name := object.get(c.attributes, "name", c.id)
	f := {
		"code": "kubernetes_cluster_audit_logging_enabled",
		"severity": "medium",
		"subject": name,
		"message": sprintf("Cluster Kubernetes « %s » : journalisation d'audit désactivée — aucun endpoint d'audit configuré, un incident ne pourrait pas être investigué.", [name]),
		"remediation": "Configurer l'audit Kubernetes du cluster (endpoint de collecte) et centraliser les journaux selon la politique de rétention.",
		"labels": {
			"provider": provider_of(c),
			"category": "compliance",
			"confidence": "confirmed",
			"message_en": sprintf("Kubernetes cluster \"%s\": audit logging disabled — no audit endpoint is configured, an incident could not be investigated.", [name]),
			"remediation_en": "Configure the cluster's Kubernetes audit (collection endpoint) and centralise the logs according to the retention policy.",
		},
	}
}

# _audit_expose — ce fournisseur expose-t-il la capacité d'audit ?
#
# Vrai dès qu'UN cluster de l'inventaire porte l'attribut, quelle que soit sa valeur :
# `audit_enabled: false` prouve à lui seul que la capacité est lue. Aucune provenance
# n'est consultée (ADR-0017).
_audit_expose if {
	some c in resources_of_type("kubernetes_cluster")
	"audit_enabled" in object.keys(c.attributes)
}

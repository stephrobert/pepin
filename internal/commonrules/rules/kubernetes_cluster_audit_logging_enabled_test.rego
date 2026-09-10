package pepin.rules

import rego.v1

_k8saudit(attrs) := {"resources": [{"provider": "exoscale", "type": "kubernetes_cluster", "id": "c1", "attributes": attrs}]}

# ✗ audit désactivé → finding LOG-1.
test_k8s_audit_disabled_denied if {
	some f in deny with input as _k8saudit({"name": "prod", "audit_enabled": false})
	f.code == "kubernetes_cluster_audit_logging_enabled"
}

# ✓ audit activé → pas de finding.
test_k8s_audit_enabled_ok if {
	count({f | some f in deny; f.code == "kubernetes_cluster_audit_logging_enabled"}) == 0 with input as _k8saudit({"name": "prod", "audit_enabled": true})
}

# ✓ attribut absent (provider n'exposant pas l'audit) → pas de finding (pas de faux positif).
test_k8s_audit_absent_ok if {
	count({f | some f in deny; f.code == "kubernetes_cluster_audit_logging_enabled"}) == 0 with input as _k8saudit({"name": "prod", "control_plane_multi_az": true})
}

# ── LE FAUX VERT (#209) ────────────────────────────────────────────────────────

_clusters(cs) := {"resources": [{"provider": "exoscale", "type": "kubernetes_cluster", "id": c.name, "attributes": c} | some c in cs]}

_audit := "kubernetes_cluster_audit_logging_enabled"

_audit_findings(cs) := {f | some f in deny with input as _clusters(cs); f.code == _audit}

# ✗ Un cluster SANS l'attribut, à côté d'un cluster qui le porte. C'est le cas réel :
# l'API SKS OMET l'objet `audit` quand l'audit est coupé, et `audit_enabled` en est
# dérivé. Le défaut par ressource étant `true`, la règle supposait « activé » et le
# rapport concluait `pass` sur le cas même qu'elle vise.
test_a_cluster_without_the_attribute_is_denied_when_the_provider_exposes_it if {
	fs := _audit_findings([
		{"name": "audite", "audit_enabled": true},
		{"name": "audit-coupe"},
	])
	count(fs) == 1
	some f in fs
	f.subject == "audit-coupe"
}

# LE CONTRE-EXEMPLE que la garde d'origine protégeait : un fournisseur qui n'expose la
# capacité NULLE PART ne déclenche rien. Sans lui, la correction échangerait un faux
# vert contre une pluie de faux positifs sur tout provider sans audit managé.
test_a_provider_without_the_capability_stays_silent if {
	count(_audit_findings([{"name": "c1"}, {"name": "c2"}])) == 0
}

# `false` explicite prouve à lui seul que la capacité est lue : un inventaire d'un seul
# cluster sans audit doit rougir.
test_an_explicit_false_alone_proves_the_capability if {
	count(_audit_findings([{"name": "seul", "audit_enabled": false}])) == 1
}

# Un cluster audité ne rougit pas.
test_an_audited_cluster_stays_silent if {
	count(_audit_findings([{"name": "ok", "audit_enabled": true}])) == 0
}

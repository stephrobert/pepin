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

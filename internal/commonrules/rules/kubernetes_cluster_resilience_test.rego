package pepin.rules

import rego.v1

_k8s(attrs) := {"resources": [{"provider": "outscale", "type": "kubernetes_cluster", "id": "c1", "attributes": attrs}]}

# ✗ mono-AZ → HA finding.
test_k8s_mono_az_denied if {
	some f in deny with input as _k8s({"name": "prod", "control_plane_multi_az": false})
	f.code == "kubernetes_cluster_control_plane_highly_available"
}

# ✓ multi-AZ → pas de finding HA.
test_k8s_multi_az_ok if {
	count({f | some f in deny; f.code == "kubernetes_cluster_control_plane_highly_available"}) == 0 with input as _k8s({"name": "prod", "control_plane_multi_az": true})
}

# ✗ auto-upgrade off → finding.
test_k8s_no_autoupgrade_denied if {
	some f in deny with input as _k8s({"name": "prod", "auto_upgrade": false})
	f.code == "kubernetes_cluster_auto_upgrade_enabled"
}

# ✗ pas de protection suppression → finding.
test_k8s_no_deletion_protection_denied if {
	some f in deny with input as _k8s({"name": "prod", "deletion_protection": false})
	f.code == "kubernetes_cluster_deletion_protection"
}

# ✓ cluster bien configuré → aucun de ces findings.
test_k8s_compliant_ok if {
	doc := _k8s({"name": "prod", "control_plane_multi_az": true, "auto_upgrade": true, "deletion_protection": true})
	count({f | some f in deny; startswith(f.code, "kubernetes_cluster_")}) == 0 with input as doc
}

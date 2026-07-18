package pepin.rules

import rego.v1

_lb(attrs) := {"resources": [{"provider": "outscale", "type": "load_balancer", "id": "lb1", "attributes": attrs}]}

# ---- kubernetes_cluster_not_publicly_accessible ----
test_oks_public_denied if {
	some f in deny with input as {"resources": [{"provider": "outscale", "type": "kubernetes_cluster", "id": "c1", "attributes": {"name": "prod", "admin_whitelist": ["0.0.0.0/0"]}}]}
	f.code == "kubernetes_cluster_not_publicly_accessible"
}

test_oks_restricted_ok if {
	count({f | some f in deny; f.code == "kubernetes_cluster_not_publicly_accessible"}) == 0 with input as {"resources": [{"provider": "outscale", "type": "kubernetes_cluster", "id": "c1", "attributes": {"name": "prod", "admin_whitelist": ["10.0.0.5/32"]}}]}
}

# ---- loadbalancer_ssl_listeners ----
test_lbu_no_https_denied if {
	some f in deny with input as _lb({"load_balancer_name": "web", "load_balancer_type": "internet-facing", "listeners": [{"load_balancer_protocol": "HTTP"}], "access_log": {"is_enabled": true}})
	f.code == "loadbalancer_ssl_listeners"
}

test_lbu_https_ok if {
	count({f | some f in deny; f.code == "loadbalancer_ssl_listeners"}) == 0 with input as _lb({"load_balancer_name": "web", "load_balancer_type": "internet-facing", "listeners": [{"load_balancer_protocol": "HTTPS"}], "access_log": {"is_enabled": true}})
}

test_lbu_internal_ok if {
	count({f | some f in deny; f.code == "loadbalancer_ssl_listeners"}) == 0 with input as _lb({"load_balancer_name": "web", "load_balancer_type": "internal", "listeners": [{"load_balancer_protocol": "HTTP"}], "access_log": {"is_enabled": true}})
}

# ---- loadbalancer_logging_enabled ----
test_lbu_no_log_denied if {
	some f in deny with input as _lb({"load_balancer_name": "web", "load_balancer_type": "internal", "listeners": [{"load_balancer_protocol": "HTTPS"}]})
	f.code == "loadbalancer_logging_enabled"
}

test_lbu_log_ok if {
	count({f | some f in deny; f.code == "loadbalancer_logging_enabled"}) == 0 with input as _lb({"load_balancer_name": "web", "load_balancer_type": "internal", "listeners": [{"load_balancer_protocol": "HTTPS"}], "access_log": {"is_enabled": true}})
}

# ---- governance_resource_required_tags ----
test_tags_missing_denied if {
	some f in deny with input as {"resources": [{"provider": "outscale", "type": "compute_instance", "id": "i-1", "attributes": {"vm_id": "i-1", "security_group_ids": ["sg-1"], "tags": [{"key": "Name", "value": "web"}]}}]}
	f.code == "governance_resource_required_tags"
}

test_tags_complete_ok if {
	doc := {"resources": [{"provider": "outscale", "type": "compute_instance", "id": "i-1", "attributes": {"vm_id": "i-1", "security_group_ids": ["sg-1"], "tags": [{"key": "CostCenter", "value": "42"}, {"key": "Project", "value": "p"}, {"key": "Env", "value": "prod"}, {"key": "Owner", "value": "sre"}]}}]}
	count({f | some f in deny; f.code == "governance_resource_required_tags"}) == 0 with input as doc
}

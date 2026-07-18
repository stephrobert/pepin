package pepin.rules

import rego.v1

_lbr(listeners) := {"resources": [{"provider": "outscale", "type": "load_balancer", "id": "lb1", "attributes": {"load_balancer_name": "web", "listeners": listeners}}]}

# ✗ Listener HTTP:80 → finding.
test_http_listener_denied if {
	some f in deny with input as _lbr([{"load_balancer_protocol": "HTTP", "load_balancer_port": 80}])
	f.code == "loadbalancer_http_redirect_to_https"
}

# ✓ Listener HTTPS uniquement → aucun finding.
test_https_only_ok if {
	count({f | some f in deny; f.code == "loadbalancer_http_redirect_to_https"}) == 0 with input as _lbr([{"load_balancer_protocol": "HTTPS", "load_balancer_port": 443}])
}

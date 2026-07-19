package pepin.rules

import rego.v1

_lbr(listeners) := {"resources": [{"provider": "outscale", "type": "load_balancer", "id": "lb1", "attributes": {"load_balancer_name": "web", "listeners": listeners}}]}

# ✗ Listener HTTP:80 dont la redirection est OBSERVÉE absente → finding.
test_http_no_redirect_denied if {
	some f in deny with input as _lbr([{"load_balancer_protocol": "HTTP", "load_balancer_port": 80, "redirect_to_https": false}])
	f.code == "loadbalancer_http_redirect_to_https"
}

# ✓ Listener HTTP:80 avec redirection observée → aucun finding.
test_http_with_redirect_ok if {
	count({f | some f in deny; f.code == "loadbalancer_http_redirect_to_https"}) == 0 with input as _lbr([{"load_balancer_protocol": "HTTP", "load_balancer_port": 80, "redirect_to_https": true}])
}

# ✓ Redirection NON collectée (garde de capacité) → pas de FAIL permanent.
test_http_redirect_uncollected_ok if {
	count({f | some f in deny; f.code == "loadbalancer_http_redirect_to_https"}) == 0 with input as _lbr([{"load_balancer_protocol": "HTTP", "load_balancer_port": 80}])
}

# ✓ Listener HTTPS uniquement → aucun finding.
test_https_only_ok if {
	count({f | some f in deny; f.code == "loadbalancer_http_redirect_to_https"}) == 0 with input as _lbr([{"load_balancer_protocol": "HTTPS", "load_balancer_port": 443}])
}

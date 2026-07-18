package pepin.rules

import rego.v1

_net(attrs) := {"resources": [{"provider": "outscale", "type": "network", "id": "n1", "attributes": attrs}]}

# ✗ réseau sans étiquette → finding NET-5 (cartographie non tenue).
test_network_untagged_denied if {
	some f in deny with input as _net({"name": "prod-net", "tags": []})
	f.code == "network_documented"
}

# ✓ réseau étiqueté (étiquettes présentes, nom porté par un tag) → pas de finding.
test_network_tagged_ok if {
	count({f | some f in deny; f.code == "network_documented"}) == 0 with input as _net({"tags": [{"key": "Name", "value": "prod-net"}, {"key": "env", "value": "prod"}]})
}

# ✓ réseau nommé ET étiqueté → pas de finding.
test_network_documented_ok if {
	count({f | some f in deny; f.code == "network_documented"}) == 0 with input as _net({"name": "prod-net", "tags": [{"key": "owner", "value": "team-x"}]})
}

# ✓ autre type (pas de network) → pas de finding.
test_network_absent_ok if {
	count({f | some f in deny; f.code == "network_documented"}) == 0 with input as {"resources": [{"provider": "outscale", "type": "compute_instance", "id": "v1", "attributes": {}}]}
}

package pepin.rules

import rego.v1

_sub(attrs) := {"resources": [{"provider": "outscale", "type": "subnet", "id": "subnet-1", "attributes": attrs}]}

# ✗ Attribution auto d'IP publique → finding.
test_subnet_auto_public_ip_denied if {
	some f in deny with input as _sub({"subnet_id": "subnet-1", "map_public_ip_on_launch": true})
	f.code == "network_subnet_no_public_ip_by_default"
}

# ✓ Pas d'attribution auto → aucun finding.
test_subnet_no_auto_public_ip_ok if {
	count({f | some f in deny; f.code == "network_subnet_no_public_ip_by_default"}) == 0 with input as _sub({"subnet_id": "subnet-1", "map_public_ip_on_launch": false})
}

# ✓ Attribut non collecté (garde de capacité) → pas de faux positif.
test_subnet_uncollected_ok if {
	count({f | some f in deny; f.code == "network_subnet_no_public_ip_by_default"}) == 0 with input as _sub({"subnet_id": "subnet-1"})
}

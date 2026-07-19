package pepin.rules

import rego.v1

_db(attrs) := {"resources": [{"provider": "scaleway", "type": "managed_database", "id": "db-1", "attributes": attrs}]}

# ✗ ACL avec 0.0.0.0/0 → finding.
test_db_open_to_all_denied if {
	some f in deny with input as _db({"database_id": "db-1", "ip_filter": ["0.0.0.0/0"]})
	f.code == "database_service_not_open_to_internet"
}

# ✗ ACL avec un /1 (moitié d'Internet) → finding (réutilise is_public_cidr).
test_db_half_internet_denied if {
	some f in deny with input as _db({"database_id": "db-1", "ip_filter": ["128.0.0.0/1"]})
	f.code == "database_service_not_open_to_internet"
}

# ✓ ACL restreinte à un CIDR privé → aucun finding.
test_db_restricted_ok if {
	count({f | some f in deny; f.code == "database_service_not_open_to_internet"}) == 0 with input as _db({"database_id": "db-1", "ip_filter": ["10.0.0.0/16"]})
}

# ✓ ip_filter non collecté (garde de capacité) → pas de faux positif.
test_db_uncollected_ok if {
	count({f | some f in deny; f.code == "database_service_not_open_to_internet"}) == 0 with input as _db({"database_id": "db-1"})
}

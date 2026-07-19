package pepin.rules

import rego.v1

_peer(attrs) := {"resources": [{"provider": "outscale", "type": "network_peering", "id": "pcx-1", "attributes": attrs}]}

# ✗ peering actif entre deux comptes différents → finding NET-7.
test_peering_cross_account_denied if {
	some f in deny with input as _peer({"peering_id": "pcx-1", "source_account": "111", "accepter_account": "222", "state": "active"})
	f.code == "network_peering_cross_organization"
}

# ✓ peering intra-compte (même SI) → pas de finding.
test_peering_same_account_ok if {
	count({f | some f in deny; f.code == "network_peering_cross_organization"}) == 0 with input as _peer({"peering_id": "pcx-1", "source_account": "111", "accepter_account": "111", "state": "active"})
}

# ✓ peering cross-compte mais expiré (pas de flux) → pas de finding.
test_peering_expired_ok if {
	count({f | some f in deny; f.code == "network_peering_cross_organization"}) == 0 with input as _peer({"peering_id": "pcx-1", "source_account": "111", "accepter_account": "222", "state": "expired"})
}

# ✓ comptes inconnus (non collectés) → pas de finding (pas de faux positif).
test_peering_unknown_accounts_ok if {
	count({f | some f in deny; f.code == "network_peering_cross_organization"}) == 0 with input as _peer({"peering_id": "pcx-1", "state": "active"})
}

# ✓ FP-4 : peering « pending-acceptance » (proposé, non accepté → aucun flux) → aucun finding.
test_peering_pending_ok if {
	count({f | some f in deny; f.code == "network_peering_cross_organization"}) == 0 with input as _peer({"peering_id": "pcx-2", "source_account": "111", "accepter_account": "222", "state": "pending-acceptance"})
}

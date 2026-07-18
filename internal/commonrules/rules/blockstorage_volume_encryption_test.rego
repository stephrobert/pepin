package pepin.rules

import rego.v1

_vol(provider, attrs) := {"resources": [{"provider": provider, "type": "blockstorage_volume", "id": "v1", "attributes": attrs}]}

# ✗ volume avec encrypted explicitement false → finding CHF-2.
test_volume_unencrypted_denied if {
	some f in deny with input as _vol("outscale", {"volume_id": "vol-1", "encrypted": false})
	f.code == "blockstorage_volume_encryption"
}

# ✓ Exoscale : encrypted true (chiffrement transparent, conforme par construction) → pas de finding.
test_volume_encrypted_ok if {
	count({f | some f in deny; f.code == "blockstorage_volume_encryption"}) == 0 with input as _vol("exoscale", {"volume_id": "vol-1", "encrypted": true})
}

# ✓ attribut absent (provider n'expose pas le chiffrement — Outscale/Scaleway) → pas de finding.
test_volume_encrypted_absent_ok if {
	count({f | some f in deny; f.code == "blockstorage_volume_encryption"}) == 0 with input as _vol("scaleway", {"volume_id": "vol-1", "state": "in-use"})
}

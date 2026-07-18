package pepin.rules

import rego.v1

_bkt(attrs) := {"resources": [{"provider": "scaleway", "type": "object_storage_bucket", "id": "b1", "attributes": attrs}]}

# ✗ bucket sans object lock → finding STO-8.
test_bucket_no_object_lock_denied if {
	some f in deny with input as _bkt({"name": "backups", "object_lock_enabled": false})
	f.code == "objectstorage_bucket_object_lock_enabled"
}

# ✓ bucket avec object lock → pas de finding.
test_bucket_object_lock_ok if {
	count({f | some f in deny; f.code == "objectstorage_bucket_object_lock_enabled"}) == 0 with input as _bkt({"name": "backups", "object_lock_enabled": true})
}

# ✓ attribut absent (source ne l'expose pas) → pas de finding.
test_bucket_object_lock_absent_ok if {
	count({f | some f in deny; f.code == "objectstorage_bucket_object_lock_enabled"}) == 0 with input as _bkt({"name": "data"})
}

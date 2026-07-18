package pepin.rules

import rego.v1

_kbkt(attrs) := {"resources": [{"provider": "scaleway", "type": "object_storage_bucket", "id": "b1", "attributes": attrs}]}

_sensitive_tags := [{"key": "data_classification", "value": "confidential"}]

# ✗ bucket sensible sans clé client (SSE-KMS) → finding CHF-4.
test_sensitive_bucket_without_kms_denied if {
	some f in deny with input as _kbkt({"name": "rh-data", "sse_kms_enabled": false, "tags": _sensitive_tags})
	f.code == "objectstorage_bucket_kms_encryption"
}

# ✓ bucket sensible AVEC clé client (SSE-KMS) → pas de finding.
test_sensitive_bucket_with_kms_ok if {
	count({f | some f in deny; f.code == "objectstorage_bucket_kms_encryption"}) == 0 with input as _kbkt({"name": "rh-data", "sse_kms_enabled": true, "tags": _sensitive_tags})
}

# ✓ bucket NON classé sensible sans SSE-KMS → pas de finding (ciblage données sensibles).
test_non_sensitive_bucket_ok if {
	count({f | some f in deny; f.code == "objectstorage_bucket_kms_encryption"}) == 0 with input as _kbkt({"name": "public-assets", "sse_kms_enabled": false, "tags": [{"key": "Env", "value": "prod"}]})
}

# ✓ provider sans capacité SSE-KMS (attribut absent — Exoscale/Outscale) → pas de finding.
test_no_kms_capability_ok if {
	count({f | some f in deny; f.code == "objectstorage_bucket_kms_encryption"}) == 0 with input as _kbkt({"name": "rh-data", "tags": _sensitive_tags})
}

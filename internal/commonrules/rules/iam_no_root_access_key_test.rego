package pepin.rules

import rego.v1

_sw_iam_code := "iam_no_root_access_key"

_sw_key(attrs) := {"resources": [{
	"provider": "scaleway", "type": "access_key", "id": "k1", "name": "ci-deploy",
	"attributes": attrs,
}]}

# ✗ Clé rattachée au compte root → finding.
test_sw_root_key_denied if {
	some f in deny with input as _sw_key({"root_owned": true})
	f.code == _sw_iam_code
}

# ✓ Clé rattachée à une application dédiée → aucun finding.
test_sw_app_key_ok if {
	count({f | some f in deny; f.code == _sw_iam_code}) == 0 with input as _sw_key({"root_owned": false, "application_id": "app-1"})
}

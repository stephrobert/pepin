package pepin.rules

import rego.v1

_role(attrs) := {"resources": [{"provider": "exoscale", "type": "iam_role", "id": "r1", "attributes": attrs}]}

# ✗ allow par défaut → finding IAM-1.
test_role_admin_privileges_denied if {
	some f in deny with input as _role({"name": "admin", "admin_privileges": true})
	f.code == "iam_role_no_admin_privileges"
}

# ✓ deny par défaut → pas de finding.
test_role_least_privilege_ok if {
	count({f | some f in deny; f.code == "iam_role_no_admin_privileges"}) == 0 with input as _role({"name": "ci", "admin_privileges": false})
}

# ✓ attribut absent → pas de finding (pas de faux positif).
test_role_admin_absent_ok if {
	count({f | some f in deny; f.code == "iam_role_no_admin_privileges"}) == 0 with input as _role({"name": "ci"})
}

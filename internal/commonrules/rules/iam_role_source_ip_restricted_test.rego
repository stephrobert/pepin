package pepin.rules

import rego.v1

_role(attrs) := {"resources": [{"provider": "exoscale", "type": "iam_role", "id": "r1", "attributes": attrs}]}

# ✗ pas de restriction IP → finding IAM-4.
test_role_no_source_ip_denied if {
	some f in deny with input as _role({"name": "admin", "source_ip_restricted": false})
	f.code == "iam_role_source_ip_restricted"
}

# ✓ restriction IP présente → pas de finding.
test_role_source_ip_ok if {
	count({f | some f in deny; f.code == "iam_role_source_ip_restricted"}) == 0 with input as _role({"name": "admin", "source_ip_restricted": true})
}

# ✓ attribut absent (provider n'exposant pas la capacité) → pas de finding.
test_role_source_ip_absent_ok if {
	count({f | some f in deny; f.code == "iam_role_source_ip_restricted"}) == 0 with input as _role({"name": "admin"})
}

# ✓ rôle prédéfini NON éditable (ex. « Billing ») → hors scope, pas de finding.
test_role_non_editable_ok if {
	count({f | some f in deny; f.code == "iam_role_source_ip_restricted"}) == 0 with input as _role({"name": "Billing", "editable": false, "source_ip_restricted": false})
}

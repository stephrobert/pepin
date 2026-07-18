package pepin.rules

import rego.v1

_wildres_code := "iam_policy_no_wildcard_resource"
_notact_code := "iam_policy_no_notaction_notresource"

_pol(stmts) := {"resources": [{"provider": "outscale", "type": "iam_policy", "id": "p1", "attributes": {"policy_name": "p1", "statements": stmts}}]}

# ✗ Resource "*" (actions ciblées) → wildcard_resource.
test_wildcard_resource_denied if {
	some f in deny with input as _pol([{"effect": "Allow", "actions": ["ReadVms"], "resources": ["*"]}])
	f.code == _wildres_code
	f.severity == "high"
}

# ✓ Resource "*" AVEC Action "*" → pas de doublon (admin rule s'en charge).
test_wildcard_resource_skipped_when_admin if {
	count({f | some f in deny; f.code == _wildres_code}) == 0 with input as _pol([{"effect": "Allow", "actions": ["*"], "resources": ["*"]}])
}

# ✗ NotAction sur Allow → notaction_notresource (critical).
test_notaction_denied if {
	some f in deny with input as _pol([{"effect": "Allow", "not_action": ["DeleteVms"], "resources": ["*"]}])
	f.code == _notact_code
	f.severity == "critical"
}

# ✓ NotAction sur Deny → aucun finding (acceptable avec Effect Deny).
test_notaction_deny_ok if {
	count({f | some f in deny; f.code == _notact_code}) == 0 with input as _pol([{"effect": "Deny", "not_action": ["DeleteVms"], "resources": ["*"]}])
}

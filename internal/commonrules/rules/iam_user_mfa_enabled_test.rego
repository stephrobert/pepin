package pepin.rules

import rego.v1

# ✗ Compte sans MFA → finding.
test_user_without_mfa_denied if {
	some f in deny with input as {"resources": [{"provider": "scaleway", "type": "iam_user", "id": "u-1", "name": "alice@example.org", "attributes": {"mfa_enabled": false}}]}
	f.code == "iam_user_mfa_enabled"
	f.severity == "high"
}

# ✓ Compte avec MFA → aucun finding.
test_user_with_mfa_ok if {
	count([f | some f in deny; f.code == "iam_user_mfa_enabled"]) == 0 with input as {"resources": [{"provider": "exoscale", "type": "iam_user", "id": "u-2", "name": "bob@example.org", "attributes": {"mfa_enabled": true}}]}
}

# ✓ Attribut absent (provider qui n'expose pas le MFA) → aucun finding (pas de faux positif).
test_user_mfa_unknown_skipped if {
	count([f | some f in deny; f.code == "iam_user_mfa_enabled"]) == 0 with input as {"resources": [{"provider": "outscale", "type": "iam_user", "id": "u-3", "name": "carol", "attributes": {}}]}
}

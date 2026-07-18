package pepin.rules

import rego.v1

# ✗ Environnement de confiance désactivé → MFA non imposée → finding.
test_account_mfa_not_enforced_denied if {
	some f in deny with input as {"resources": [{"provider": "outscale", "type": "api_access_policy", "id": "api-access-policy", "name": "api-access-policy", "attributes": {"require_trusted_env": false}}]}
	f.code == "iam_account_mfa_enforced"
	f.severity == "high"
}

# ✓ Environnement de confiance activé → aucun finding.
test_account_mfa_enforced_ok if {
	count([f | some f in deny; f.code == "iam_account_mfa_enforced"]) == 0 with input as {"resources": [{"provider": "outscale", "type": "api_access_policy", "id": "api-access-policy", "name": "api-access-policy", "attributes": {"require_trusted_env": true}}]}
}

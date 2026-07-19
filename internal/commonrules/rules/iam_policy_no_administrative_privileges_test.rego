package pepin.rules

import rego.v1

_eim_code := "iam_policy_no_administrative_privileges"

_eim(statements) := {"resources": [{
	"provider": "outscale",
	"type": "iam_policy",
	"id": "p1",
	"attributes": {"policy_name": "p1", "statements": statements},
}]}

# ✗ Statement Allow avec Action "*" → finding critical.
test_wildcard_action_denied if {
	some f in deny with input as _eim([{"effect": "Allow", "actions": ["*"], "resources": ["*"]}])
	f.code == _eim_code
	f.severity == "critical"
}

# ✗ FN-5 : joker de service « api:* » (API unifiée Outscale) accorde toutes les actions.
test_api_wildcard_denied if {
	some f in deny with input as _eim([{"effect": "Allow", "actions": ["api:*"], "resources": ["*"]}])
	f.code == _eim_code
}

# ✗ FN-5 : forme « *:* » accorde toutes les actions.
test_star_star_denied if {
	some f in deny with input as _eim([{"effect": "Allow", "actions": ["*:*"], "resources": ["*"]}])
	f.code == _eim_code
}

# ✗ Casse : `Api:*` / `API:*` (les noms d'action EIM sont insensibles à la casse) doivent
#   être attrapés — sinon contournement trivial, resource scopée (les autres règles ne voient rien).
test_api_wildcard_uppercase_denied if {
	some f in deny with input as _eim([{"effect": "Allow", "actions": ["Api:*"], "resources": ["orn:scoped"]}])
	f.code == _eim_code
}

# ✗ Effet en minuscules (« allow ») accepté (comparaison insensible à la casse).
test_lowercase_effect_denied if {
	some f in deny with input as _eim([{"effect": "allow", "actions": ["*"], "resources": ["*"]}])
	f.code == _eim_code
}

# ✓ Actions explicites → aucun finding.
test_scoped_actions_ok if {
	count({f | some f in deny; f.code == _eim_code}) == 0 with input as _eim([{"effect": "Allow", "actions": ["ReadVms", "ReadVolumes"], "resources": ["*"]}])
}

# ✓ Deny avec "*" → aucun finding (l'effet n'est pas Allow).
test_deny_wildcard_ok if {
	count({f | some f in deny; f.code == _eim_code}) == 0 with input as _eim([{"effect": "Deny", "actions": ["*"], "resources": ["*"]}])
}

# ✓ Aucun statement → aucun finding (accès défensif).
test_no_statements_ok if {
	count({f | some f in deny; f.code == _eim_code}) == 0 with input as _eim([])
}

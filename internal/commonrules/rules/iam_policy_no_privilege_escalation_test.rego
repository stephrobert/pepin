package pepin.rules

import rego.v1

_pol(stmts) := {"resources": [{"provider": "outscale", "type": "iam_policy", "id": "p1", "attributes": {"policy_name": "p1", "statements": stmts}}]}

# ✗ allow d'une action d'escalade (préfixe de service ignoré) → finding IAM-12.
# NB : on teste des actions RÉELLES du contrat OAPI. L'ancien test utilisait
# « AttachUserPolicy », un nom AWS qui n'existe pas chez Outscale : il validait une
# détection factice pendant que les vraies actions passaient au travers.
test_escalation_allowed_denied if {
	some f in deny with input as _pol([{"effect": "Allow", "actions": ["api:LinkPolicy"], "resources": ["*"]}])
	f.code == "iam_policy_no_privilege_escalation"
}

# ✗ Les autres chemins d'auto-élévation réels du contrat OAPI.
test_escalation_real_oapi_actions_denied if {
	every act in ["api:PutUserGroupPolicy", "eim:AddUserToUserGroup", "api:SetDefaultPolicyVersion", "api:UpdateApiAccessPolicy"] {
		count({f | some f in deny with input as _pol([{"effect": "Allow", "actions": [act], "resources": ["*"]}]); f.code == "iam_policy_no_privilege_escalation"}) > 0
	}
}

# ✓ UnlinkPolicy RETIRE une policy : ce n'est pas une élévation. Le matching par préfixe
# (et non par sous-chaîne) évite de le confondre avec LinkPolicy.
test_unlink_policy_not_escalation if {
	count({f | some f in deny; f.code == "iam_policy_no_privilege_escalation"}) == 0 with input as _pol([{"effect": "Allow", "actions": ["api:UnlinkPolicy"], "resources": ["x"]}])
}

# ✗ CreateAccessKey autorisé → finding.
test_create_access_key_denied if {
	some f in deny with input as _pol([{"effect": "allow", "actions": ["CreateAccessKey"], "resources": ["*"]}])
	f.code == "iam_policy_no_privilege_escalation"
}

# ✓ action non sensible (lecture S3) → pas de finding.
test_benign_action_ok if {
	count({f | some f in deny; f.code == "iam_policy_no_privilege_escalation"}) == 0 with input as _pol([{"effect": "Allow", "actions": ["s3:GetObject"], "resources": ["*"]}])
}

# ✗ M5 : joker de service d'identité `eim:*` (resource scopée) → escalade détectée.
test_eim_service_wildcard_denied if {
	some f in deny with input as _pol([{"effect": "Allow", "actions": ["eim:*"], "resources": ["orn:scoped"]}])
	f.code == "iam_policy_no_privilege_escalation"
}

test_iam_service_wildcard_denied if {
	some f in deny with input as _pol([{"effect": "Allow", "actions": ["iam:*"], "resources": ["orn:scoped"]}])
	f.code == "iam_policy_no_privilege_escalation"
}

# ✓ joker d'un service NON-identité (s3:*) → pas d'escalade (couvert ailleurs si Resource=*).
test_non_identity_service_wildcard_ok if {
	count({f | some f in deny; f.code == "iam_policy_no_privilege_escalation"}) == 0 with input as _pol([{"effect": "Allow", "actions": ["s3:*"], "resources": ["orn:scoped"]}])
}

# ✓ action d'escalade en Deny → pas de finding.
test_escalation_denied_effect_ok if {
	count({f | some f in deny; f.code == "iam_policy_no_privilege_escalation"}) == 0 with input as _pol([{"effect": "Deny", "actions": ["AttachUserPolicy"], "resources": ["*"]}])
}

# ✓ pas de statements → pas de finding.
test_no_statements_ok if {
	count({f | some f in deny; f.code == "iam_policy_no_privilege_escalation"}) == 0 with input as _pol([])
}

_role(attrs) := {"resources": [{"provider": "exoscale", "type": "iam_role", "id": "r1", "attributes": attrs}]}

# ✗ rôle éditable dont la policy autorise la gestion des rôles IAM → finding IAM-12.
test_role_iam_management_denied if {
	some f in deny with input as _role({"name": "deployer", "editable": true, "manages_iam": true})
	f.code == "iam_policy_no_privilege_escalation"
}

# ✓ rôle prédéfini (non éditable) → hors scope.
test_role_predefined_ok if {
	count({f | some f in deny; f.code == "iam_policy_no_privilege_escalation"}) == 0 with input as _role({"name": "Billing", "editable": false, "manages_iam": true})
}

# ✓ rôle sans gestion IAM → pas de finding.
test_role_no_iam_management_ok if {
	count({f | some f in deny; f.code == "iam_policy_no_privilege_escalation"}) == 0 with input as _role({"name": "ci", "editable": true, "manages_iam": false})
}

# ✗ politique conférant la gestion IAM (PermissionSet, ex. Scaleway) → finding.
test_policy_manages_iam_denied if {
	some f in deny with input as {"resources": [{"provider": "scaleway", "type": "iam_policy", "id": "p", "attributes": {"policy_name": "p", "manages_iam": true}}]}
	f.code == "iam_policy_no_privilege_escalation"
}

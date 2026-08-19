package pepin.rules

import rego.v1

# ✗ règle d'accès API ouverte à 0.0.0.0/0.
test_apiaccessrule_public_denied if {
	some f in deny with input as {"resources": [{"provider": "outscale", "type": "api_access_rule", "id": "aar-1", "attributes": {"api_access_rule_id": "aar-1", "ip_ranges": ["0.0.0.0/0"]}}]}
	f.code == "iam_apiaccessrule_no_public_cidr"
}

# ✓ règle restreinte → aucun finding.
test_apiaccessrule_restricted_ok if {
	count({f | some f in deny; f.code == "iam_apiaccessrule_no_public_cidr"}) == 0 with input as {"resources": [{"provider": "outscale", "type": "api_access_rule", "id": "aar-1", "attributes": {"ip_ranges": ["10.0.0.0/8"]}}]}
}

# ✗ aucune règle d'accès API (rule_count=0).
test_apiaccess_none_denied if {
	some f in deny with input as {"resources": [{"provider": "outscale", "type": "api_access_summary", "id": "acct", "attributes": {"rule_count": 0}}]}
	f.code == "iam_apiaccessrule_defined"
}

# ✓ au moins une règle → aucun finding « non défini ».
test_apiaccess_present_ok if {
	count({f | some f in deny; f.code == "iam_apiaccessrule_defined"}) == 0 with input as {"resources": [{"provider": "outscale", "type": "api_access_summary", "id": "acct", "attributes": {"rule_count": 2}}]}
}

# ✗ policy sans expiration max.
test_apiaccesspolicy_no_expiration_denied if {
	some f in deny with input as {"resources": [{"provider": "outscale", "type": "api_access_policy", "id": "pol", "attributes": {"max_access_key_expiration_seconds": 0}}]}
	f.code == "iam_apiaccesspolicy_max_key_expiration"
}

# ✓ policy avec expiration max → aucun finding.
test_apiaccesspolicy_expiration_ok if {
	count({f | some f in deny; f.code == "iam_apiaccesspolicy_max_key_expiration"}) == 0 with input as {"resources": [{"provider": "outscale", "type": "api_access_policy", "id": "pol", "attributes": {"max_access_key_expiration_seconds": 7776000}}]}
}

# ✓ FP-6 : politique d'accès API n'exposant PAS le réglage d'expiration (clé absente) → aucun
# finding (garde de capacité ; 0 présent = « aucune limite » reste flaguée).
test_apiaccesspolicy_expiration_absent_ok if {
	count({f | some f in deny; f.code == "iam_apiaccesspolicy_max_key_expiration"}) == 0 with input as {"resources": [{"provider": "outscale", "type": "api_access_policy", "id": "ap-1", "attributes": {"name": "default"}}]}
}

# ✓ Règle 0.0.0.0/0 MAIS exigeant un certificat client (CaIds) → pas un accès ouvert.
test_apiaccessrule_public_cidr_with_client_cert_ok if {
	count({f | some f in deny; f.code == "iam_apiaccessrule_no_public_cidr"}) == 0 with input as {"resources": [{"provider": "outscale", "type": "api_access_rule", "id": "aar-1", "attributes": {"api_access_rule_id": "aar-1", "ip_ranges": ["0.0.0.0/0"], "ca_ids": ["ca-1"], "cns": []}}]}
}

# ✗ Règle 0.0.0.0/0 sans certificat client → finding (comportement inchangé).
test_apiaccessrule_public_cidr_without_cert_denied if {
	some f in deny with input as {"resources": [{"provider": "outscale", "type": "api_access_rule", "id": "aar-2", "attributes": {"api_access_rule_id": "aar-2", "ip_ranges": ["0.0.0.0/0"], "ca_ids": [], "cns": []}}]}
	f.code == "iam_apiaccessrule_no_public_cidr"
}

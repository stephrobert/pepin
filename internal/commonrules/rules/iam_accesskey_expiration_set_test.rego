package pepin.rules

import rego.v1

_key_code := "iam_accesskey_expiration_set"

# L'instant d'évaluation est FIXÉ : sans lui, `_eval_now_ns` retombe sur l'horloge, et
# un test dont le verdict change avec la date du jour ne mesure plus rien.
_cle_datee(attrs) := {
	"resources": [{"provider": "outscale", "type": "access_key", "id": "ak-1", "attributes": attrs}],
	"evaluated_at": "2026-09-09T00:00:00Z",
}

# ✗ Clé sans expiration → finding.
test_key_no_expiration_denied if {
	some f in deny with input as _cle_datee({"access_key_id": "ak-1", "state": "ACTIVE"})
	f.code == _key_code
}

# ✗ expiration_date vide → finding.
test_key_empty_expiration_denied if {
	count({f | some f in deny; f.code == _key_code}) == 1 with input as _cle_datee({"access_key_id": "ak-1", "expiration_date": ""})
}

# ✓ Clé avec expiration → aucun finding.
test_key_with_expiration_ok if {
	count({f | some f in deny; f.code == _key_code}) == 0 with input as _cle_datee({"access_key_id": "ak-1", "expiration_date": "2026-12-31T23:59:59Z"})
}

# ── Une date PASSÉE ne protège rien ────────────────────────────────────────────
#
# Mesuré sur un tenant réel : `ExpirationDate: 2024-09-07`, `State: ACTIVE`, deux ans
# après l'échéance — et le contrôle rendait `pass`.

test_an_expired_key_still_active_is_denied if {
	f := deny with input as _cle_datee({
		"access_key_id": "AK1",
		"expiration_date": "2024-09-07T23:59:59Z",
		"state": "ACTIVE",
	})
	count(f) == 1
	some x in f
	x.severity == "high"
}

# Le CONTRE-EXEMPLE : une échéance à venir est exactement ce que le contrôle demande.
test_a_future_expiry_is_silent if {
	f := deny with input as _cle_datee({
		"access_key_id": "AK1",
		"expiration_date": "2027-01-01T00:00:00Z",
		"state": "ACTIVE",
	})
	count(f) == 0
}

# Une clé DÉSACTIVÉE dont l'échéance est passée ne dit plus rien du tenant : elle ne
# peut pas servir. Crier dessus serait le faux positif qui fait désactiver l'outil.
test_an_expired_but_inactive_key_is_silent if {
	f := deny with input as _cle_datee({
		"access_key_id": "AK1",
		"expiration_date": "2024-09-07T23:59:59Z",
		"state": "DELETED",
	})
	count(f) == 0
}

# Une date ILLISIBLE ne produit pas d'écart : on ne déduit rien d'une donnée qu'on n'a
# pas su lire (ADR-0014).
test_an_unreadable_date_produces_no_finding if {
	f := deny with input as _cle_datee({
		"access_key_id": "AK1",
		"expiration_date": "jamais",
		"state": "ACTIVE",
	})
	count(f) == 0
}

# L'instant de référence est celui de l'ÉVALUATION, pas l'horloge : le rejeu d'un
# bundle scellé doit rendre le MÊME verdict.
test_the_verdict_follows_the_evaluation_instant if {
	attrs := {"access_key_id": "AK1", "expiration_date": "2025-06-01T00:00:00Z", "state": "ACTIVE"}
	avant := deny with input as {"resources": [{"provider": "outscale", "type": "access_key", "id": "K", "attributes": attrs}], "evaluated_at": "2025-01-01T00:00:00Z"}
	apres := deny with input as {"resources": [{"provider": "outscale", "type": "access_key", "id": "K", "attributes": attrs}], "evaluated_at": "2026-01-01T00:00:00Z"}
	count(avant) == 0
	count(apres) == 1
}

package pepin.rules

import rego.v1

# L'instant d'évaluation est FIXÉ : un test dont le verdict change avec la date du jour
# ne mesure plus rien.
_cle_rot(attrs) := {
	"resources": [{"provider": "outscale", "type": "access_key", "id": "ak-1", "attributes": attrs}],
	"evaluated_at": "2026-09-09T00:00:00Z",
}

_rot := {f | some f in deny; f.code == "iam_accesskey_rotated"}

# ── Ce que le contrôle doit attraper ───────────────────────────────────────────

# Le cas mesuré : une clé de deux ans, active, avec une expiration lointaine qui
# satisfait l'autre contrôle sans qu'aucune rotation n'ait eu lieu.
test_a_two_year_old_key_is_denied if {
	f := _rot with input as _cle_rot({
		"access_key_id": "AK1",
		"creation_date": "2024-09-01T00:00:00Z",
		"expiration_date": "2099-01-01T00:00:00Z",
		"state": "ACTIVE",
	})
	count(f) == 1
}

# ── LES CONTRE-EXEMPLES : ce qui doit rester muet ──────────────────────────────

# Une clé récente est exactement ce que le contrôle demande. C'est le contre-exemple
# que ce contrôle ne doit jamais perdre.
test_a_recently_rotated_key_is_silent if {
	f := _rot with input as _cle_rot({
		"access_key_id": "AK1",
		"creation_date": "2026-08-01T00:00:00Z",
		"state": "ACTIVE",
	})
	count(f) == 0
}

# Juste dans la fenêtre : la borne se teste, sinon on ne sait pas où elle est.
test_a_key_just_inside_the_window_is_silent if {
	f := _rot with input as _cle_rot({
		"access_key_id": "AK1",
		"creation_date": "2026-07-01T00:00:00Z", # 70 jours
		"state": "ACTIVE",
	})
	count(f) == 0
}

# Une clé DÉSACTIVÉE ne peut plus servir : crier dessus est le faux positif qui fait
# désactiver l'outil.
test_an_inactive_old_key_is_silent if {
	f := _rot with input as _cle_rot({
		"access_key_id": "AK1",
		"creation_date": "2020-01-01T00:00:00Z",
		"state": "DELETED",
	})
	count(f) == 0
}

# Sans date de création, on ne déduit RIEN : une absence n'est pas un âge (ADR-0014).
test_a_key_without_a_creation_date_produces_no_finding if {
	f := _rot with input as _cle_rot({"access_key_id": "AK1", "state": "ACTIVE"})
	count(f) == 0
}

test_an_unreadable_creation_date_produces_no_finding if {
	f := _rot with input as _cle_rot({
		"access_key_id": "AK1",
		"creation_date": "hier",
		"state": "ACTIVE",
	})
	count(f) == 0
}

# ── La fenêtre est RÉGLABLE, et l'allonger tait des écarts ─────────────────────

test_the_window_is_configurable if {
	attrs := {"access_key_id": "AK1", "creation_date": "2026-05-01T00:00:00Z", "state": "ACTIVE"}
	base := {
		"resources": [{"provider": "outscale", "type": "access_key", "id": "k", "attributes": attrs}],
		"evaluated_at": "2026-09-09T00:00:00Z",
	}

	# 131 jours : au-delà du défaut de 90, donc signalé.
	count(_rot) == 1 with input as base

	# Fenêtre portée à 365 : la même clé se tait. C'est bien un ASSOUPLISSEMENT, et le
	# référentiel l'adosse à CLD-IAM-2 par `au_plus_le_defaut`.
	count(_rot) == 0 with input as object.union(base, {"config": {"iam": {"key_max_age_days": 365}}})
}

# L'instant de référence est celui de l'ÉVALUATION : le rejeu d'un bundle scellé doit
# rendre le MÊME verdict.
test_the_verdict_follows_the_evaluation_instant if {
	attrs := {"access_key_id": "AK1", "creation_date": "2026-01-01T00:00:00Z", "state": "ACTIVE"}
	res := [{"provider": "outscale", "type": "access_key", "id": "k", "attributes": attrs}]
	count(_rot) == 0 with input as {"resources": res, "evaluated_at": "2026-02-01T00:00:00Z"}
	count(_rot) == 1 with input as {"resources": res, "evaluated_at": "2026-09-09T00:00:00Z"}
}

package pepin.rules

import rego.v1

_ud_vm(userdata) := {"resources": [{
	"provider": "outscale",
	"type": "compute_instance",
	"id": "i-1",
	"attributes": {"vm_id": "i-1", "user_data": userdata},
}]}

_ud_vm_with(userdata, cfg) := object.union(_ud_vm(userdata), {"config": {"secrets": cfg}})

_secret_findings := {f | some f in deny; f.code == "compute_instance_no_secrets_in_user_data"}

# La VALEUR d'un secret, telle qu'elle ne doit JAMAIS ressortir. Les trois
# niveaux sont représentés : la propriété ne dépend pas de la confiance.
_pem := "-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAKCAQEAxTOP53CRET53CRET53CRET\n-----END RSA PRIVATE KEY-----"

_token := "ghp_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

_weak := "password=hunter2000plus"

# ── Le niveau voyage avec la détection ───────────────────────────────────────

# ✗ bloc PEM → confiance HIGH (secret confirmé par sa forme).
test_private_key_is_high_confidence if {
	some f in deny with input as _ud_vm(_pem)
	f.code == "compute_instance_no_secrets_in_user_data"
	f.labels.confidence == "high"
}

# ✗ jeton de forge à préfixe reconnu → confiance MEDIUM.
test_prefixed_token_is_medium_confidence if {
	some f in deny with input as _ud_vm(_token)
	f.code == "compute_instance_no_secrets_in_user_data"
	f.labels.confidence == "medium"
}

# ✗ heuristique générique → confiance LOW.
test_generic_password_is_low_confidence if {
	some f in deny with input as _ud_vm(_weak)
	f.code == "compute_instance_no_secrets_in_user_data"
	f.labels.confidence == "low"
}

# ── LA VALEUR NE SORT JAMAIS, quel que soit le niveau ────────────────────────
#
# C'est la propriété la plus coûteuse à perdre : un rapport voyage en SARIF, dans
# les artefacts de CI et dans un bundle scellé. Elle est donc testée aux trois
# niveaux, sur le message ET sur la remédiation, français et anglais.

test_secret_value_never_leaks_high if {
	every f in _secret_findings {
		not contains(f.message, "MIIEowIBAAKCAQEA")
		not contains(f.remediation, "MIIEowIBAAKCAQEA")
		not contains(f.labels.message_en, "MIIEowIBAAKCAQEA")
		not contains(f.labels.remediation_en, "MIIEowIBAAKCAQEA")
	}
		with input as _ud_vm(_pem)
}

test_secret_value_never_leaks_medium if {
	every f in _secret_findings {
		not contains(f.message, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
		not contains(f.labels.message_en, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	}
		with input as _ud_vm(_token)
}

test_secret_value_never_leaks_low if {
	every f in _secret_findings {
		not contains(f.message, "hunter2000plus")
		not contains(f.labels.message_en, "hunter2000plus")
	}
		with input as _ud_vm(_weak)
}

# ── Le SEUIL est réglable, et il ne l'est que dans un sens visible ───────────

# ✓ seuil `medium` : l'heuristique générique se tait. C'est l'assouplissement que
# le référentiel adosse à `secrets.min_confidence: au_plus_le_defaut` — le
# rapport dira que la correspondance CLD-CMP-9 n'est plus tenue.
test_threshold_medium_silences_low if {
	count(_secret_findings) == 0 with input as _ud_vm_with(_weak, {"min_confidence": "medium"})
}

# ✗ … mais il ne tait PAS ce qui l'atteint : un jeton reconnu passe toujours.
test_threshold_medium_keeps_medium if {
	some f in deny with input as _ud_vm_with(_token, {"min_confidence": "medium"})
	f.code == "compute_instance_no_secrets_in_user_data"
}

# ✓ seuil `high` : seul un secret confirmé subsiste.
test_threshold_high_silences_medium if {
	count(_secret_findings) == 0 with input as _ud_vm_with(_token, {"min_confidence": "high"})
}

test_threshold_high_keeps_high if {
	some f in deny with input as _ud_vm_with(_pem, {"min_confidence": "high"})
	f.code == "compute_instance_no_secrets_in_user_data"
}

# ✗ DÉFAUT INCHANGÉ : sans configuration, `low` — donc tout est signalé, comme
# avant ce lot. Un réglage qui change le comportement par défaut n'est pas un
# réglage, c'est un changement de contrôle déguisé.
test_default_threshold_reports_everything if {
	some f in deny with input as _ud_vm(_weak)
	f.code == "compute_instance_no_secrets_in_user_data"
}

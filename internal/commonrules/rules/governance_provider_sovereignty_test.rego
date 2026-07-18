package pepin.rules

import rego.v1

_gov(attrs) := {"resources": [{"provider": attrs.provider, "type": "governance_provider", "id": attrs.provider, "attributes": attrs}]}

# ✗ siège hors UE (Exoscale/CH) → finding GVN-4.
test_gov_non_eu_denied if {
	some f in deny with input as _gov({"provider": "exoscale", "eu_established": false, "jurisdiction": "CH"})
	f.code == "governance_provider_sovereignty"
}

# ✓ fournisseur UE, contrôle UE, pas d'exposition (Outscale/Scaleway) → aucun finding.
test_gov_eu_ok if {
	count({f | some f in deny; f.code == "governance_provider_sovereignty"}) == 0 with input as _gov({"provider": "outscale", "eu_established": true, "jurisdiction": "FR", "capital_control": "FR", "secnumcloud": "qualifie", "extraterritorial_exposure": false})
}

# ✗ contrôle capitalistique extra-UE → finding.
test_gov_extra_eu_capital_denied if {
	some f in deny with input as _gov({"provider": "x", "eu_established": true, "capital_control": "extra_ue"})
	f.code == "governance_provider_sovereignty"
}

# ✗ exposition extraterritoriale sans SecNumCloud → finding.
test_gov_extraterritorial_denied if {
	some f in deny with input as _gov({"provider": "x", "eu_established": true, "extraterritorial_exposure": true, "secnumcloud": "non"})
	f.code == "governance_provider_sovereignty"
}

# ✓ exposition mais qualifié SecNumCloud (immunité reconnue) → pas de finding.
test_gov_secnumcloud_immunity_ok if {
	count({f | some f in deny; f.code == "governance_provider_sovereignty"}) == 0 with input as _gov({"provider": "x", "eu_established": true, "capital_control": "FR", "extraterritorial_exposure": true, "secnumcloud": "qualifie"})
}

# ✓ contrôle « a_verifier » (incertain) → pas de faux positif.
test_gov_capital_unverified_ok if {
	count({f | some f in deny; f.code == "governance_provider_sovereignty"}) == 0 with input as _gov({"provider": "exoscale", "eu_established": true, "capital_control": "a_verifier", "secnumcloud": "non"})
}

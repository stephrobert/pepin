# Souveraineté du fournisseur cloud.
#   Type synthétique agnostique `governance_provider` (un par scan), porté par le
#   descripteur du provider (providers/<nom>.yaml, section `souverainete`), pas par
#   la collecte. Attributs : eu_established (bool), jurisdiction, capital_control
#   (FR|UE|extra_ue|a_verifier), secnumcloud (qualifie|en_cours|non),
#   extraterritorial_exposure (bool). Faits ANCRÉS sur sources officielles.
# SCSL : CLD-GVN-4 (fournisseur établi dans l'UE, soustrait à un contrôle
#   capitalistique extra-UE déterminant ; exposition extraterritoriale évaluée).
#   Profil souverain (R3). Une qualification SecNumCloud vaut immunité reconnue.
package pepin.rules

import rego.v1

# Siège du fournisseur hors Union européenne.
deny contains f if {
	some r in resources_of_type("governance_provider")
	object.get(r.attributes, "eu_established", true) == false
	f := {
		"code": "governance_provider_sovereignty",
		"severity": "high",
		"subject": r.id,
		"message": sprintf("Fournisseur « %s » : siège hors Union européenne (juridiction %s) — exigence d'établissement UE non satisfaite.", [r.id, object.get(r.attributes, "jurisdiction", "?")]),
		"remediation": "Pour une exigence souveraine, retenir un fournisseur établi dans l'UE (idéalement qualifié SecNumCloud).",
		"labels": {"provider": provider_of(r), "category": "compliance"},
	}
}

# Contrôle capitalistique extra-UE déterminant.
deny contains f if {
	some r in resources_of_type("governance_provider")
	object.get(r.attributes, "capital_control", "") == "extra_ue"
	f := {
		"code": "governance_provider_sovereignty",
		"severity": "high",
		"subject": r.id,
		"message": sprintf("Fournisseur « %s » : contrôle capitalistique extra-UE déterminant — risque de soumission à une juridiction étrangère.", [r.id]),
		"remediation": "Privilégier un fournisseur dont le contrôle capitalistique reste dans l'UE ; documenter l'exposition.",
		"labels": {"provider": provider_of(r), "category": "compliance"},
	}
}

# Exposition à une loi extraterritoriale (ex. Cloud Act) sans immunité reconnue
# (la qualification SecNumCloud vaut immunité).
deny contains f if {
	some r in resources_of_type("governance_provider")
	object.get(r.attributes, "extraterritorial_exposure", false) == true
	object.get(r.attributes, "secnumcloud", "non") != "qualifie"
	f := {
		"code": "governance_provider_sovereignty",
		"severity": "high",
		"subject": r.id,
		"message": sprintf("Fournisseur « %s » : exposition à une loi extraterritoriale, sans qualification SecNumCloud établissant l'immunité.", [r.id]),
		"remediation": "Évaluer l'exposition extraterritoriale ; retenir une offre qualifiée SecNumCloud (immunité reconnue par l'ANSSI).",
		"labels": {"provider": provider_of(r), "category": "compliance"},
	}
}

# compute_image_not_public
#   Image machine (OMI/template) partagée publiquement : quiconque peut la lancer,
#   exposant données ou secrets embarqués dans l'image.
# SCSL : CLD-STO-2 (instantanés ET images disque non partagés publiquement).
# Contrat : type normalisé agnostique `compute_image` ; attribut public (bool,
#   DÉRIVÉ — Outscale ReadImages PermissionsToLaunch.GlobalPermission=true).
package pepin.rules

import rego.v1

deny contains f if {
	some r in resources_of_type("compute_image")
	object.get(r.attributes, "public", false) == true
	id := object.get(r.attributes, "image_id", r.id)
	f := {
		"code": "compute_image_not_public",
		"severity": "high",
		"subject": id,
		"message": sprintf("Image machine « %s » partagée publiquement (lancement autorisé à tous).", [id]),
		"remediation": "Retirer le partage public de l'image ; la réserver aux comptes légitimes.",
		"labels": {"provider": provider_of(r), "category": "security"},
	}
}

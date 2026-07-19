# blockstorage_snapshot_not_public
#   Instantané (snapshot) BSU partagé publiquement (création de volume autorisée
#   à tous) — risque de fuite de données.
# Origine : osc-policy OSC-SNAP-001. SCSL : CLD-STO-2.
# Contrat : type normalisé agnostique `blockstorage_snapshot` ; attribut
#   global_permission (bool, DÉRIVÉ de osc-sdk-go
#   Snapshot.PermissionsToCreateVolume.GlobalPermission).
package pepin.rules

import rego.v1

deny contains f if {
	some r in resources_of_type("blockstorage_snapshot")
	truthy(object.get(r.attributes, "global_permission", false))
	id := object.get(r.attributes, "snapshot_id", r.id)
	f := {
		"code": "blockstorage_snapshot_not_public",
		"severity": "high",
		"subject": id,
		"message": sprintf("Snapshot « %s » partagé publiquement (création de volume autorisée à tous).", [id]),
		"remediation": "Retirer la permission globale ; partager le snapshot aux seuls comptes légitimes.",
		"labels": {"provider": provider_of(r), "category": "security"},
	}
}

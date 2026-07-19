# blockstorage_volume_snapshots_exist
#   Volume block storage en usage sans snapshot récent (< 7 jours) — restauration
#   après incident non garantie.
# Origine : osc-policy OSC-VOL-006. SCSL : CLD-STO-3.
# Contrat : types normalisés agnostiques `blockstorage_volume` (volume_id, state)
#   et `blockstorage_snapshot` (volume_id, creation_date RFC3339/ISO 8601).
#   État « en usage » normalisé via volume_in_use (Outscale `in-use`, Exoscale
#   `attached`, cf. lib.rego). Sources : OAPI Volume.State ; egoscale/v3
#   block-storage-volume.state + block-storage-snapshot (block-storage-volume ref, created-at).
package pepin.rules

import rego.v1

_recent_window_ns := ((7 * 24) * 3600) * 1000000000

deny contains f if {
	some v in resources_of_type("blockstorage_volume")
	volume_in_use(v)
	vid := object.get(v.attributes, "volume_id", v.id)
	not _has_recent_snapshot(vid)
	f := {
		"code": "blockstorage_volume_snapshots_exist",
		"severity": "high",
		"subject": vid,
		"message": sprintf("Volume « %s » (en usage) sans snapshot dans les 7 derniers jours.", [vid]),
		"remediation": "Mettre en place un snapshot régulier automatisé ; tester la restauration périodiquement.",
		"labels": {"provider": provider_of(v), "category": "compliance"},
	}
}

_has_recent_snapshot(vid) if {
	some s in resources_of_type("blockstorage_snapshot")
	object.get(s.attributes, "volume_id", "") == vid
	date := object.get(s.attributes, "creation_date", "")
	is_string(date)
	date != ""
	_eval_now_ns - time.parse_rfc3339_ns(date) <= _recent_window_ns
}

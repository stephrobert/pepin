package pepin.rules

import rego.v1

# Un instant d'évaluation FIGÉ dans tous les cas : une fenêtre de fraîcheur testée
# contre l'horloge de la machine est un test qui change de verdict à minuit.
_now := "2026-07-19T12:00:00Z"

_vol_and_snap(snapAttrs) := {
	"evaluated_at": _now,
	"resources": [
		{"provider": "outscale", "type": "blockstorage_volume", "id": "vol-1", "attributes": {"volume_id": "vol-1", "state": "in-use"}},
		{"provider": "outscale", "type": "blockstorage_snapshot", "id": "snap-1", "attributes": snapAttrs},
	],
}

_snapshot_findings := {f | some f in deny; f.code == "blockstorage_volume_snapshots_exist"}

# ✗ ÉTAT VÉRIFIÉ, PAS SEULEMENT LA DATE : une snapshot d'hier, mais en `error`,
# ne restaure rien. La compter reviendrait à rendre vert sur une sauvegarde qui
# n'existe pas (issue #49). État natif Outscale : Snapshot.State.
test_snapshot_in_error_does_not_count if {
	some f in deny with input as _vol_and_snap({"volume_id": "vol-1", "creation_date": "2026-07-18T12:00:00Z", "state": "error"})
	f.code == "blockstorage_volume_snapshots_exist"
}

# ✗ … de même pour une snapshot ENCORE EN COURS (`pending`) : elle n'est pas une
# sauvegarde tant qu'elle n'est pas terminée.
test_snapshot_pending_does_not_count if {
	some f in deny with input as _vol_and_snap({"volume_id": "vol-1", "creation_date": "2026-07-18T12:00:00Z", "state": "pending"})
	f.code == "blockstorage_volume_snapshots_exist"
}

# ✓ état `completed` (Outscale) dans la fenêtre → la snapshot compte.
test_snapshot_completed_counts if {
	count(_snapshot_findings) == 0 with input as _vol_and_snap({"volume_id": "vol-1", "creation_date": "2026-07-18T12:00:00Z", "state": "completed"})
}

# ✓ état `created` (Exoscale, block-storage-snapshot.state) dans la fenêtre.
test_snapshot_created_counts if {
	count(_snapshot_findings) == 0 with input as _vol_and_snap({"volume_id": "vol-1", "creation_date": "2026-07-18T12:00:00Z", "state": "created"})
}

# ✓ GARDE DE CAPACITÉ : `state` non collecté → la snapshot compte quand même.
# Inventer « inexploitable » ferait un faux positif sur une sauvegarde réelle ;
# c'est le verrou d'attribut de l'assessment qui dit ce qui n'a pas été lu.
test_snapshot_state_not_collected_still_counts if {
	count(_snapshot_findings) == 0 with input as _vol_and_snap({"volume_id": "vol-1", "creation_date": "2026-07-18T12:00:00Z"})
}

# ✗ DÉLAI CONFIGURABLE, sens du DURCISSEMENT : une fenêtre d'un jour refuse une
# snapshot vieille de trois jours que le défaut acceptait.
test_configured_shorter_window_denies if {
	inv := object.union(
		_vol_and_snap({"volume_id": "vol-1", "creation_date": "2026-07-16T12:00:00Z", "state": "completed"}),
		{"config": {"snapshots": {"max_age_days": 1, "accepted_states": ["completed", "created"]}}},
	)
	some f in deny with input as inv
	f.code == "blockstorage_volume_snapshots_exist"
	contains(f.message, "1 jours")
}

# ✓ DÉLAI CONFIGURABLE, sens de l'ASSOUPLISSEMENT : une fenêtre de trente jours
# accepte une snapshot vieille de vingt jours. Le verdict change, et c'est
# précisément pour cela que le référentiel adosse la correspondance CLD-STO-3 à
# `snapshots.max_age_days: au_plus_le_defaut` — le rapport dira que la
# correspondance n'est plus tenue.
test_configured_longer_window_passes if {
	inv := object.union(
		_vol_and_snap({"volume_id": "vol-1", "creation_date": "2026-06-29T12:00:00Z", "state": "completed"}),
		{"config": {"snapshots": {"max_age_days": 30, "accepted_states": ["completed", "created"]}}},
	)
	count(_snapshot_findings) == 0 with input as inv
}

# ✓ ÉTATS ACCEPTÉS CONFIGURABLES : une politique qui accepte `promoting` compte
# une snapshot dans cet état. Élargir l'ensemble est un assouplissement
# (`accepted_states: sous_ensemble_du_defaut`), et il est signalé comme tel.
test_configured_accepted_states if {
	inv := object.union(
		_vol_and_snap({"volume_id": "vol-1", "creation_date": "2026-07-18T12:00:00Z", "state": "promoting"}),
		{"config": {"snapshots": {"max_age_days": 7, "accepted_states": ["completed", "created", "promoting"]}}},
	)
	count(_snapshot_findings) == 0 with input as inv
}

# ✗ une snapshot d'un AUTRE volume ne sauve pas celui-ci.
test_snapshot_of_another_volume_does_not_count if {
	some f in deny with input as _vol_and_snap({"volume_id": "vol-9", "creation_date": "2026-07-18T12:00:00Z", "state": "completed"})
	f.code == "blockstorage_volume_snapshots_exist"
}

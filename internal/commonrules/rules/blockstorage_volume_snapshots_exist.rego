# blockstorage_volume_snapshots_exist — FRAÎCHEUR d'une snapshot de volume.
#
# Ce que le contrôle mesure, exactement : un volume en usage a-t-il, dans la
# fenêtre configurée (par défaut 7 jours), au moins une snapshot dont l'état natif
# la dit TERMINÉE ?
#
# Ce que le contrôle ne prouve PAS, et il faut le lire avant de s'en servir :
#   · que la snapshot soit RESTAURABLE — aucune restauration n'est tentée ;
#   · qu'elle soit COMPLÈTE au sens applicatif (bases à chaud, volumes multiples) ;
#   · qu'une RÉTENTION soit respectée — une seule snapshot suffit à le satisfaire ;
#   · qu'une POLITIQUE DE SAUVEGARDE existe. Un volume peut être parfaitement
#     sauvegardé autrement (sauvegarde applicative, réplication, service de backup
#     du fournisseur, outil externe) et être signalé ici : c'est un faux positif
#     assumé, qui se traite par une dérogation datée et justifiée, jamais en
#     désactivant le contrôle.
# Autrement dit : ce contrôle ne participe pas à une affirmation « sauvegarde
# conforme ». Il mesure une fraîcheur, et il ne dit que ça.
#
# Le CODE n'est PAS renommé en `blockstorage_recent_snapshot_exists` : il voyage
# dans les `ruleId` SARIF, dans les assessments archivés et dans les fichiers de
# dérogations, où un renommage transformerait du jour au lendemain une dérogation
# valide en dérogation orpheline, et ferait réapparaître l'écart qu'elle couvrait.
# C'est le TITRE, la description et le message qui disent désormais la vérité.
#
# Non évaluable sur un plan Terraform : `state` y arrive en `after_unknown` et
# aucun `blockstorage_snapshot` n'y figure — d'où le verrou `requiredAttr`.
#
# Origine : osc-policy OSC-VOL-006. SCSL : CLD-STO-3.
# Contrat : types normalisés `blockstorage_volume` (volume_id, state) et
#   `blockstorage_snapshot` (volume_id, creation_date RFC3339/ISO 8601, state).
#   « En usage » normalisé via volume_in_use (Outscale `in-use`, Exoscale
#   `attached`, cf. lib.rego). Sources : OAPI Volume.State / Snapshot.State
#   (in-queue|pending|completed|error|deleting) ; egoscale/v3
#   block-storage-volume.state + block-storage-snapshot (block-storage-volume
#   ref, created-at, state ∈ …|creating|created|promoting|error|…).
package pepin.rules

import rego.v1

deny contains f if {
	some v in resources_of_type("blockstorage_volume")
	volume_in_use(v)
	vid := object.get(v.attributes, "volume_id", v.id)
	not _has_recent_snapshot(vid)
	f := {
		"code": "blockstorage_volume_snapshots_exist",
		"severity": "high",
		"subject": vid,
		"message": sprintf("Volume « %s » (en usage) sans snapshot terminée depuis moins de %d jours.", [vid, snapshot_max_age_days]),
		"remediation": "Mettre en place un snapshot régulier automatisé ; tester la restauration périodiquement. Si ce volume est sauvegardé autrement, poser une dérogation datée et justifiée plutôt que de désactiver le contrôle.",
		"labels": {
			"provider": provider_of(v),
			"category": "compliance",
			"message_en": sprintf("Volume \"%s\" (in use) has no completed snapshot younger than %d days.", [vid, snapshot_max_age_days]),
			"remediation_en": "Set up an automated, regular snapshot schedule; test the restore periodically. If this volume is backed up by other means, file a dated, justified exemption rather than disabling the control.",
		},
	}
}

# _has_recent_snapshot — une snapshot du volume, dans la fenêtre ET dans un état
# exploitable. L'état est vérifié, pas seulement la date : une snapshot `error`,
# `creating` ou `deleting` ne restaure rien, et la compter reviendrait à rendre
# vert sur une sauvegarde qui n'existe pas.
_has_recent_snapshot(vid) if {
	some s in resources_of_type("blockstorage_snapshot")
	object.get(s.attributes, "volume_id", "") == vid
	_snapshot_usable(s)
	date := object.get(s.attributes, "creation_date", "")
	is_string(date)
	date != ""
	_eval_now_ns - time.parse_rfc3339_ns(date) <= snapshot_max_age_ns
}

# _snapshot_usable — l'état de la snapshot est ACCEPTÉ. Deux cas, et le second
# est une garde de capacité, pas une complaisance : quand le collecteur n'a pas
# projeté `state` (source qui ne l'expose pas), on ne SAIT pas si la snapshot est
# terminée. Inventer « inexploitable » ferait un faux positif sur une sauvegarde
# réelle ; ce qui manque se dit ailleurs, par le verrou d'attribut de l'assessment.
_snapshot_usable(s) if object.get(s.attributes, "state", "") in snapshot_accepted_states

_snapshot_usable(s) if not _snapshot_state_collected(s)

_snapshot_state_collected(s) if {
	"state" in object.keys(s.attributes)
	st := object.get(s.attributes, "state", "")
	is_string(st)
	st != ""
}

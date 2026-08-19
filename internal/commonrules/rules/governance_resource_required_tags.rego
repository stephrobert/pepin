# governance_resource_required_tags — COMMUN à tous les providers.
#   Ressource facturable sans les étiquettes de gouvernance obligatoires
#   (CostCenter, Project, Env, Owner) — inventaire et responsabilité non tenus.
# SCSL : CLD-GVN-1. S'applique aux types normalisés facturables ; attribut tags[]
#   ({key,value}). required_tags défini dans lib.rego.
package pepin.rules

import rego.v1

_billable_types := {"compute_instance", "blockstorage_volume", "load_balancer", "object_storage_bucket"}

deny contains f if {
	some r in input.resources
	r.type in _billable_types
	"tags" in object.keys(r.attributes) # garde de capacité : le provider EXPOSE les étiquettes

	# (sinon, un provider qui ne collecte pas les tags — ex. Exoscale live — déclencherait un
	# finding « 4 étiquettes manquantes » sur CHAQUE ressource : tempête de faux positifs).
	tags := object.get(r.attributes, "tags", [])
	missing := [t | some t in required_tags; not has_tag(tags, t)]
	count(missing) > 0
	name := object.get(r, "name", r.id)
	f := {
		"code": "governance_resource_required_tags",
		"severity": "medium",
		"subject": name,
		"message": sprintf("Ressource « %s » : étiquettes de gouvernance manquantes (%s).", [name, concat(", ", missing)]),
		"remediation": "Ajouter les étiquettes obligatoires (CostCenter, Project, Env, Owner) sur la ressource.",
		"labels": {
			"provider": provider_of(r),
			"category": "compliance",
			"message_en": sprintf("Resource \"%s\": governance tags missing (%s).", [name, concat(", ", missing)]),
			"remediation_en": "Add the mandatory tags (CostCenter, Project, Env, Owner) to the resource.",
		},
	}
}

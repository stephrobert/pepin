package pepin.rules

import rego.v1

_tagged(typ, tags) := {"resources": [{
	"provider": "scaleway",
	"type": typ,
	"id": "r1",
	"name": "web",
	"attributes": {"tags": tags},
}]}

_gvn_count := {f | some f in deny; f.code == "governance_resource_required_tags"}

_full := [
	{"key": "CostCenter", "value": "cc-1"},
	{"key": "Project", "value": "billing"},
	{"key": "Env", "value": "prod"},
	{"key": "Owner", "value": "sre"},
]

# ✓ NON-RÉGRESSION : les quatre étiquettes historiques (CostCenter, Project, Env,
# Owner) satisfont toujours le profil par défaut. C'est la propriété qui garantit
# qu'ouvrir la politique à la configuration n'a déplacé aucun verdict.
test_historic_tag_names_still_pass if {
	count(_gvn_count) == 0 with input as _tagged("compute_instance", _full)
}

# ✓ FAUX POSITIF CORRIGÉ : une organisation qui écrit `cost-center, application,
# environment, team` est gouvernée, et ne doit plus récolter un FAIL sur sa
# convention d'écriture (issue #61).
test_other_writing_conventions_pass if {
	count(_gvn_count) == 0 with input as _tagged("compute_instance", [
		{"key": "cost-center", "value": "cc-1"},
		{"key": "application", "value": "billing"},
		{"key": "environment", "value": "prod"},
		{"key": "team", "value": "sre"},
	])
}

# ✗ aucune étiquette → les quatre manquent, et elles sont nommées.
test_no_tag_denied if {
	some f in deny with input as _tagged("compute_instance", [])
	f.code == "governance_resource_required_tags"
	contains(f.message, "Owner")
	contains(f.message, "CostCenter")
}

# ✗ étiquetage partiel : seul le propriétaire est posé.
test_partial_tags_denied if {
	some f in deny with input as _tagged("compute_instance", [{"key": "Owner", "value": "sre"}])
	f.code == "governance_resource_required_tags"
	not contains(f.message, "Owner")
	contains(f.message, "Project")
}

# ✗ FAUX NÉGATIF CORRIGÉ : une base managée est facturée et étiquetable ; elle
# entre dans le périmètre au même titre qu'une instance (issue #61).
test_managed_database_in_scope if {
	some f in deny with input as _tagged("managed_database", [])
	f.code == "governance_resource_required_tags"
}

# ✓ un type HORS périmètre (règle de security group : ni facturée, ni étiquetable)
# ne déclenche rien : le périmètre est explicite, pas « tout ce qui a des tags ».
test_out_of_scope_type_silent if {
	count(_gvn_count) == 0 with input as _tagged("security_group_rule", [])
}

# ✓ GARDE DE CAPACITÉ : `tags` non collecté → la règle se tait.
test_tags_not_collected_silent if {
	count(_gvn_count) == 0 with input as {"resources": [{
		"provider": "exoscale",
		"type": "compute_instance",
		"id": "r1",
		"attributes": {"state": "running"},
	}]}
}

# ✓ la CONFIGURATION est lue : une politique qui n'exige que le propriétaire
# laisse passer une ressource qui ne porte que lui.
test_configured_required_tags if {
	inv := object.union(
		_tagged("compute_instance", [{"key": "Owner", "value": "sre"}]),
		{"config": {"tagging": {
			"resource_types": ["compute_instance"],
			"required": [{"name": "owner", "keys": ["owner"]}],
		}}},
	)
	count(_gvn_count) == 0 with input as inv
}

# ✗ … et une politique qui RESTREINT le périmètre ne désarme pas le contrôle sur
# ce qu'elle y garde.
test_configured_scope_still_denies if {
	inv := object.union(
		_tagged("compute_instance", []),
		{"config": {"tagging": {
			"resource_types": ["compute_instance"],
			"required": [{"name": "owner", "keys": ["owner"]}],
		}}},
	)
	some f in deny with input as inv
	f.code == "governance_resource_required_tags"
}

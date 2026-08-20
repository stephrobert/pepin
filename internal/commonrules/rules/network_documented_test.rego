package pepin.rules

import rego.v1

_net(attrs) := {"resources": [{"provider": "outscale", "type": "network", "id": "n1", "attributes": attrs}]}

# ✗ réseau sans étiquette → finding NET-5 (cartographie non tenue).
test_network_untagged_denied if {
	some f in deny with input as _net({"name": "prod-net", "tags": []})
	f.code == "network_documented"
}

# ✗ TAG UNIQUE ET HORS SUJET : `foo=bar` ne documente rien. C'est le cas que la
# règle validait autrefois (`count(tags) > 0`) et qui faisait annoncer au contrôle
# plus qu'il ne vérifiait. Il doit désormais échouer, et nommer ce qui manque.
test_network_single_irrelevant_tag_denied if {
	some f in deny with input as _net({"name": "prod-net", "tags": [{"key": "foo", "value": "bar"}]})
	f.code == "network_documented"
	contains(f.message, "Owner")
	contains(f.message, "Project")
	contains(f.message, "Env")
}

# ✗ documentation PARTIELLE : propriétaire posé, projet et environnement absents.
test_network_partially_tagged_denied if {
	some f in deny with input as _net({"tags": [{"key": "owner", "value": "team-x"}]})
	f.code == "network_documented"
	contains(f.message, "Project")
	not contains(f.message, "Owner")
}

# ✗ étiquette présente mais VIDE : elle ne documente pas davantage qu'une absence.
test_network_empty_tag_value_denied if {
	some f in deny with input as _net({"tags": [
		{"key": "owner", "value": ""},
		{"key": "project", "value": "p"},
		{"key": "environment", "value": "prod"},
	]})
	f.code == "network_documented"
	contains(f.message, "Owner")
}

# ✓ les trois étiquettes de cartographie, écrites autrement : `Team` pour owner,
# `Env` pour environment — casse et séparateurs indifférents.
test_network_documented_by_aliases_ok if {
	count({f | some f in deny; f.code == "network_documented"}) == 0 with input as _net({"tags": [
		{"key": "Team", "value": "sre"},
		{"key": "Project", "value": "billing"},
		{"key": "Env", "value": "prod"},
	]})
}

# ✓ les trois étiquettes exactes → pas de finding.
test_network_documented_ok if {
	count({f | some f in deny; f.code == "network_documented"}) == 0 with input as _net({"name": "prod-net", "tags": [
		{"key": "owner", "value": "team-x"},
		{"key": "project", "value": "billing"},
		{"key": "environment", "value": "prod"},
	]})
}

# ✓ GARDE DE CAPACITÉ : `tags` non collecté → la règle se tait. « Je ne sais pas »
# ne se dit pas « non conforme » ; l'assessment rendra `not-evaluated`.
test_network_tags_not_collected_silent if {
	count({f | some f in deny; f.code == "network_documented"}) == 0 with input as _net({"name": "prod-net"})
}

# ✓ autre type (pas de network) → pas de finding.
test_network_absent_ok if {
	count({f | some f in deny; f.code == "network_documented"}) == 0 with input as {"resources": [{"provider": "outscale", "type": "compute_instance", "id": "v1", "attributes": {}}]}
}

# ✓ la CONFIGURATION est lue : une politique qui n'exige que le propriétaire
# laisse passer un réseau qui ne porte que lui.
test_network_configured_required_tags if {
	inv := object.union(
		_net({"tags": [{"key": "owner", "value": "team-x"}]}),
		{"config": {"tagging": {"network_required": [{"name": "owner", "keys": ["owner"]}]}}},
	)
	count({f | some f in deny; f.code == "network_documented"}) == 0 with input as inv
}

# ✗ … et la même politique refuse toujours un réseau qui ne le porte PAS : une
# configuration lue n'est pas une configuration qui désarme.
test_network_configured_still_denies if {
	inv := object.union(
		_net({"tags": [{"key": "foo", "value": "bar"}]}),
		{"config": {"tagging": {"network_required": [{"name": "owner", "keys": ["owner"]}]}}},
	)
	some f in deny with input as inv
	f.code == "network_documented"
}

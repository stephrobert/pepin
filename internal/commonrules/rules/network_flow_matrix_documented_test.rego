package pepin.rules

import rego.v1

_rule(attrs) := {"resources": [{"provider": "exoscale", "type": "security_group_rule", "id": "sg1", "attributes": attrs}]}

# ✗ flux entrant sans justification → finding NET-5.
test_flow_undocumented_denied if {
	some f in deny with input as _rule({"direction": "inbound", "port_from": 443, "description": ""})
	f.code == "network_flow_matrix_documented"
}

# ✓ flux entrant justifié → pas de finding.
test_flow_documented_ok if {
	count({f | some f in deny; f.code == "network_flow_matrix_documented"}) == 0 with input as _rule({"direction": "inbound", "port_from": 443, "description": "HTTPS public du site"})
}

# ✓ flux sortant non justifié → pas de finding (on ne vise que l'entrant).
test_flow_egress_ok if {
	count({f | some f in deny; f.code == "network_flow_matrix_documented"}) == 0 with input as _rule({"direction": "outbound", "description": ""})
}

# ✓ provider n'exposant pas la justification (pas de clé description) → pas de finding.
test_flow_no_description_field_ok if {
	count({f | some f in deny; f.code == "network_flow_matrix_documented"}) == 0 with input as _rule({"direction": "inbound", "port_from": 22})
}

# ── LE FAUX VERT (#201), et ce qui le tenait ────────────────────────────────────

_deux_regles(a, b) := {"resources": [
	{"provider": "exoscale", "type": "security_group_rule", "id": "sg-doc", "attributes": a},
	{"provider": "exoscale", "type": "security_group_rule", "id": "sg-nodoc", "attributes": b},
]}

_net5 := "network_flow_matrix_documented"

# ✗ Une règle SANS clé `description`, à côté d'une règle qui en porte une. C'est le
# cas d'un plan Terraform : l'exploitant qui n'écrit pas de description ne produit
# tout simplement pas le champ.
#
# La garde d'origine testait la clé SUR LA RESSOURCE, donc elle se taisait — et le
# silence devenait un `pass`, le verrou de capacité voyant l'attribut collecté sur le
# type. Un `pass` que rien n'établissait, sur l'écart même que ce contrôle vise.
test_a_rule_without_the_key_is_denied_when_the_provider_exposes_it if {
	fs := {f |
		some f in deny with input as _deux_regles(
			{"direction": "inbound", "port_from": 443, "description": "HTTPS public"},
			{"direction": "inbound", "port_from": 22},
		)
		f.code == _net5
	}
	count(fs) == 1
	some f in fs
	f.subject == "sg-nodoc"
}

# LE CONTRE-EXEMPLE que la garde d'origine protégeait, et qui doit tenir : un
# fournisseur qui n'expose ce champ NULLE PART ne déclenche rien. Sans lui, la
# correction échangerait un faux vert contre une pluie de faux positifs sur les deux
# fournisseurs qui n'ont pas de description par règle.
test_a_provider_without_the_field_anywhere_stays_silent if {
	count({f |
		some f in deny with input as _deux_regles(
			{"direction": "inbound", "port_from": 443},
			{"direction": "inbound", "port_from": 22},
		)
		f.code == _net5
	}) == 0
}

# La clé PRÉSENTE ET VIDE prouve à elle seule que le champ existe : un inventaire
# d'une seule règle non documentée doit rougir. Ne compter que les descriptions non
# vides rendait la règle muette sur exactement le cas qu'elle vise — c'est le test
# `test_flow_undocumented_denied` qui l'a attrapé.
test_an_empty_description_alone_proves_the_field_exists if {
	some f in deny with input as {"resources": [{"provider": "exoscale", "type": "security_group_rule", "id": "sg-1", "attributes": {"direction": "inbound", "port_from": 443, "description": ""}}]}
	f.code == _net5
}

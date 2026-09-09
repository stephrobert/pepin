package pepin.rules

import rego.v1

_default_sgr(attrs) := {"resources": [{"provider": "outscale", "type": "security_group_rule", "id": "sg-1", "attributes": attrs}]}

_default_findings(attrs) := {f | some f in deny with input as _default_sgr(attrs); f.code == "network_securitygroup_default_restrict_traffic"}

# ✗ Le SG « default » porte une règle entrante : elle s'applique d'office aux ressources
# créées sans groupe explicite, donc sans décision de l'exploitant.
test_default_sg_inbound_denied if {
	some f in deny with input as _default_sgr({"security_group_id": "sg-1", "security_group_name": "default", "direction": "inbound", "protocol": "tcp", "cidrs": ["10.0.0.0/8"]})
	f.code == "network_securitygroup_default_restrict_traffic"
}

# ✓ Un SG nommé autrement n'est pas concerné : il est attaché explicitement.
test_named_sg_inbound_ok if {
	count(_default_findings({"security_group_id": "sg-2", "security_group_name": "web", "direction": "inbound", "protocol": "tcp", "cidrs": ["10.0.0.0/8"]})) == 0
}

# ✓ Une règle SORTANTE du SG default n'est pas visée ici (l'egress a son propre contrôle) :
# sinon chaque SG produirait un doublon, la règle sortante par défaut étant systématique.
test_default_sg_outbound_ok if {
	count(_default_findings({"security_group_id": "sg-1", "security_group_name": "default", "direction": "outbound", "protocol": "all", "cidrs": ["0.0.0.0/0"]})) == 0
}

# ✓ Nom non collecté (provider ne l'exposant pas) → garde de capacité, aucun verdict inventé.
test_default_sg_name_uncollected_silent if {
	count(_default_findings({"security_group_id": "sg-1", "direction": "inbound", "protocol": "tcp"})) == 0
}

# ── CE QUE LA RÈGLE NOMME : l'usine contre l'exploitant ────────────────────────

# La forme MESURÉE sur un réseau neuf : la seule source admise est le groupe lui-même.
# Le finding subsiste — c'est bien l'écart que CLD-NET-4 vise —, mais il dit de QUOI il
# parle, et sa confiance cesse d'affirmer plus que ce que le scan établit.
test_the_factory_self_reference_is_named_and_contextual if {
	fs := _default_findings({"security_group_id": "sg-1", "security_group_name": "default", "direction": "inbound", "protocol": "all", "peer_security_group_ids": ["sg-1"]})
	count(fs) == 1
	some f in fs
	f.labels.confidence == "contextual"
	contains(f.message, "groupe lui-même")
	contains(f.labels.message_en, "the group itself")
}

# LE CONTRE-EXEMPLE qui empêche la caractérisation de tout absoudre : une règle du SG
# « default » ouverte à un CIDR est une décision d'exploitant, et elle reste `confirmed`.
test_a_cidr_source_stays_confirmed if {
	fs := _default_findings({"security_group_id": "sg-1", "security_group_name": "default", "direction": "inbound", "protocol": "tcp", "cidrs": ["0.0.0.0/0"]})
	count(fs) == 1
	some f in fs
	f.labels.confidence == "confirmed"
	not contains(f.message, "groupe lui-même")
}

# Un AUTRE groupe admis n'est pas la règle d'usine : quelqu'un l'a écrite.
test_a_peer_group_that_is_not_itself_stays_confirmed if {
	fs := _default_findings({"security_group_id": "sg-1", "security_group_name": "default", "direction": "inbound", "protocol": "all", "peer_security_group_ids": ["sg-99"]})
	some f in fs
	f.labels.confidence == "confirmed"
}

# Le groupe lui-même ET un autre : il ne suffit pas qu'un membre soit l'auto-référence.
test_itself_plus_another_group_stays_confirmed if {
	fs := _default_findings({"security_group_id": "sg-1", "security_group_name": "default", "direction": "inbound", "protocol": "all", "peer_security_group_ids": ["sg-1", "sg-99"]})
	some f in fs
	f.labels.confidence == "confirmed"
}

# Une auto-référence assortie d'un CIDR n'est plus l'usine : le CIDR est le fait qui compte.
test_self_reference_with_a_cidr_stays_confirmed if {
	fs := _default_findings({"security_group_id": "sg-1", "security_group_name": "default", "direction": "inbound", "protocol": "all", "peer_security_group_ids": ["sg-1"], "cidrs": ["0.0.0.0/0"]})
	some f in fs
	f.labels.confidence == "confirmed"
}

# LE REPLI. Sans le champ des membres — un provider qui ne l'expose pas, une source qui
# ne le porte pas —, la règle ne caractérise pas : elle retombe sur la formulation
# générale et sur `confirmed`. Ne pas savoir ne doit pas faire sortir l'écart de la porte.
test_an_uncollected_peer_field_falls_back_to_confirmed if {
	fs := _default_findings({"security_group_id": "sg-1", "security_group_name": "default", "direction": "inbound", "protocol": "all"})
	count(fs) == 1
	some f in fs
	f.labels.confidence == "confirmed"
}

# Un champ collecté mais VIDE dit « aucune source par groupe », pas « auto-référence ».
test_an_empty_peer_list_falls_back_to_confirmed if {
	fs := _default_findings({"security_group_id": "sg-1", "security_group_name": "default", "direction": "inbound", "protocol": "all", "peer_security_group_ids": []})
	some f in fs
	f.labels.confidence == "confirmed"
}

# UNE seule forme s'applique, jamais les deux : deux corps concurrents rendraient la
# fonction indéfinie, donc la règle MUETTE — l'inverse exact de ce qu'on cherche.
test_exactly_one_finding_per_rule_whatever_the_shape if {
	count(_default_findings({"security_group_id": "sg-1", "security_group_name": "default", "direction": "inbound", "protocol": "all", "peer_security_group_ids": ["sg-1"]})) == 1
	count(_default_findings({"security_group_id": "sg-1", "security_group_name": "default", "direction": "inbound", "protocol": "all", "peer_security_group_ids": ["sg-9"]})) == 1
	count(_default_findings({"security_group_id": "sg-1", "security_group_name": "default", "direction": "inbound", "protocol": "all"})) == 1
}

# La sévérité ne bouge PAS. Nommer la forme précise le message et la confiance ; cela ne
# rend pas l'écart moins grave, et le rétrograder ferait passer au vert des chaînes qui
# rougissent aujourd'hui sans que personne ne l'ait décidé.
test_the_severity_never_moves if {
	every attrs in [
		{"security_group_id": "sg-1", "security_group_name": "default", "direction": "inbound", "protocol": "all", "peer_security_group_ids": ["sg-1"]},
		{"security_group_id": "sg-1", "security_group_name": "default", "direction": "inbound", "protocol": "tcp", "cidrs": ["0.0.0.0/0"]},
	] {
		some f in _default_findings(attrs)
		f.severity == "high"
	}
}

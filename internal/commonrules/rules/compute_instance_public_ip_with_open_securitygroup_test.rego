package pepin.rules

import rego.v1

_vm_code := "compute_instance_public_ip_with_open_securitygroup"

_open_rule := {
	"provider": "outscale", "type": "security_group_rule", "id": "sg-1-in-0",
	"attributes": {"direction": "inbound", "action": "accept", "security_group_id": "sg-1", "cidrs": ["0.0.0.0/0"], "protocol": "all"},
}

_vm(attrs) := {"provider": "outscale", "type": "compute_instance", "id": "i-1", "attributes": attrs}

# ✗ VM avec IP publique + SG ouvert sur Internet → finding critical.
test_public_vm_open_sg_denied if {
	input_doc := {"resources": [_open_rule, _vm({"vm_id": "i-1", "public_ip": "203.0.113.1", "security_group_ids": ["sg-1"]})]}
	some f in deny with input as input_doc
	f.code == _vm_code
	f.severity == "critical"
}

# ✓ VM avec IP publique mais SG NON ouvert → aucun finding.
test_public_vm_closed_sg_ok if {
	closed_rule := {"provider": "outscale", "type": "security_group_rule", "id": "sg-2-in-0", "attributes": {"direction": "inbound", "action": "accept", "security_group_id": "sg-2", "cidrs": ["10.0.0.0/8"], "protocol": "tcp"}}
	input_doc := {"resources": [closed_rule, _vm({"vm_id": "i-1", "public_ip": "203.0.113.1", "security_group_ids": ["sg-2"]})]}
	count({f | some f in deny; f.code == _vm_code}) == 0 with input as input_doc
}

# ✓ VM sans IP publique (SG ouvert) → aucun finding.
test_private_vm_open_sg_ok if {
	input_doc := {"resources": [_open_rule, _vm({"vm_id": "i-1", "security_group_ids": ["sg-1"]})]}
	count({f | some f in deny; f.code == _vm_code}) == 0 with input as input_doc
}

# _sg — règle entrante depuis Internet, sur une plage de ports donnée.
_sg_range(id, from, to) := {
	"provider": "outscale", "type": "security_group_rule", "id": sprintf("%s-in", [id]),
	"attributes": {
		"direction": "inbound", "action": "accept", "security_group_id": id,
		"cidrs": ["0.0.0.0/0"], "protocol": "tcp", "port_from": from, "port_to": to,
	},
}

_public_vm(sg) := _vm({"vm_id": "i-1", "public_ip": "203.0.113.1", "security_group_ids": [sg]})

# ✓ LE CONTRE-EXEMPLE. Une VM publique servant 443/tcp depuis Internet est un serveur
# web, pas un incident. La signaler en `critical` était le finding le plus coûteux du
# dépôt : celui qu'une personne rencontre en cinq minutes, et qui fait douter de tous
# les autres. Ce test est ce qui empêche son retour.
test_public_vm_serving_https_only_is_not_a_deviation if {
	doc := {"resources": [_sg_range("sg-web", 443, 443), _public_vm("sg-web")]}
	count({f | some f in deny; f.code == _vm_code}) == 0 with input as doc
}

# ✓ Le même en HTTP, qui redirige en pratique vers HTTPS → pas davantage un écart.
test_public_vm_serving_http_only_is_not_a_deviation if {
	doc := {"resources": [_sg_range("sg-web", 80, 80), _public_vm("sg-web")]}
	count({f | some f in deny; f.code == _vm_code}) == 0 with input as doc
}

# ✗ SSH ouvert sur Internet → high, et le message NOMME le port.
test_public_vm_ssh_open_is_high if {
	doc := {"resources": [_sg_range("sg-adm", 22, 22), _public_vm("sg-adm")]}
	some f in deny with input as doc
	f.code == _vm_code
	f.severity == "high"
	contains(f.message, "22")
}

# ✗ RDP ouvert sur Internet → high également.
test_public_vm_rdp_open_is_high if {
	doc := {"resources": [_sg_range("sg-adm", 3389, 3389), _public_vm("sg-adm")]}
	some f in deny with input as doc
	f.code == _vm_code
	f.severity == "high"
}

# ✗ Une plage large qui ENGLOBE un port sensible reste un écart : la largeur ne dilue
# pas l'exposition, elle l'aggrave.
test_public_vm_wide_range_covering_ssh_is_high if {
	doc := {"resources": [_sg_range("sg-large", 1, 1024), _public_vm("sg-large")]}
	some f in deny with input as doc
	f.code == _vm_code
	f.severity == "high"
}

# ✗ Sentinelle « tous les ports » (-1/-1, encodage Outscale OAPI) → critical.
test_public_vm_all_ports_sentinel_is_critical if {
	doc := {"resources": [_sg_range("sg-any", -1, -1), _public_vm("sg-any")]}
	some f in deny with input as doc
	f.code == _vm_code
	f.severity == "critical"
}

# ── L'EXPOSITION EST UNE PROPRIÉTÉ DE LA CARTE ────────────────────────────────

# Le cas MESURÉ sur le tenant : carte primaire privée avec un groupe fermé, carte
# secondaire portant l'IP publique ET un groupe ouvert sur 22. Aucun finding n'était
# produit, alors que SSH répondait sur l'IP publique.
_deux_cartes(attrs_vm, cartes, regles) := {"resources": array.concat(
	array.concat(
		[{"provider": "outscale", "type": "compute_instance", "id": "i-1", "attributes": attrs_vm}],
		[{"provider": "outscale", "type": "network_interface", "id": c.nic_id, "attributes": c} | some c in cartes],
	),
	[{"provider": "outscale", "type": "security_group_rule", "id": r.security_group_id, "attributes": r} | some r in regles],
)}

_ssh_ouvert(sg) := {"security_group_id": sg, "direction": "inbound", "action": "accept", "protocol": "tcp", "port_from": 22, "port_to": 22, "cidrs": ["0.0.0.0/0"]}

_cld_net_3 := "compute_instance_public_ip_with_open_securitygroup"

_findings_net3(input_doc) := {f | some f in deny with input as input_doc; f.code == _cld_net_3}

# ✗ La carte SECONDAIRE porte l'IP publique et le groupe ouvert : c'est un écart.
test_a_public_secondary_nic_with_an_open_group_is_denied if {
	fs := _findings_net3(_deux_cartes(
		{"vm_id": "i-1", "security_group_ids": ["sg-web"]},
		[
			{"nic_id": "eni-1", "vm_id": "i-1", "security_group_ids": ["sg-web"]},
			{"nic_id": "eni-2", "vm_id": "i-1", "public_ip": "198.51.100.7", "security_group_ids": ["sg-ssh"]},
		],
		[_ssh_ouvert("sg-ssh")],
	))
	count(fs) == 1
	some f in fs
	f.severity == "high"
	f.subject == "i-1"
}

# LE CONTRE-EXEMPLE que l'appariement existe pour tenir : une carte PUBLIQUE au
# groupe fermé, plus une carte PRIVÉE au groupe ouvert. Réunir les deux dirait
# « publique et ouverte » — un écart que cette machine ne porte pas. Rien n'est
# joignable depuis Internet, et la règle doit se taire.
test_a_public_nic_and_a_separate_open_nic_stay_silent if {
	count(_findings_net3(_deux_cartes(
		{"vm_id": "i-1", "security_group_ids": ["sg-web"]},
		[
			{"nic_id": "eni-1", "vm_id": "i-1", "public_ip": "198.51.100.7", "security_group_ids": ["sg-web"]},
			{"nic_id": "eni-2", "vm_id": "i-1", "security_group_ids": ["sg-ssh"]},
		],
		[_ssh_ouvert("sg-ssh")],
	))) == 0
}

# UN seul finding quand la carte primaire est déjà celle qui expose : le repli ne doit
# pas s'ajouter au chemin des cartes, sinon un même fait produirait deux écarts.
test_one_finding_when_the_primary_nic_is_the_exposed_one if {
	count(_findings_net3(_deux_cartes(
		{"vm_id": "i-1", "public_ip": "198.51.100.7", "security_group_ids": ["sg-ssh"]},
		[{"nic_id": "eni-1", "vm_id": "i-1", "public_ip": "198.51.100.7", "security_group_ids": ["sg-ssh"]}],
		[_ssh_ouvert("sg-ssh")],
	))) == 1
}

# LE REPLI. Une source qui ne collecte pas les cartes — un plan Terraform — juge la
# machine sur ses propres attributs, exactement comme avant. Un repli muet ferait
# DISPARAÎTRE des écarts que le dépôt détecte aujourd'hui.
test_without_nics_the_vm_is_judged_on_its_own_attributes if {
	count(_findings_net3({"resources": [
		{"provider": "outscale", "type": "compute_instance", "id": "i-1", "attributes": {"vm_id": "i-1", "public_ip": "198.51.100.7", "security_group_ids": ["sg-ssh"]}},
		{"provider": "outscale", "type": "security_group_rule", "id": "sg-ssh", "attributes": _ssh_ouvert("sg-ssh")},
	]})) == 1
}

# Une carte d'une AUTRE machine n'expose pas celle-ci : la jointure porte sur `vm_id`,
# et une jointure trop large est le faux positif le plus coûteux du dépôt.
test_a_nic_of_another_vm_never_exposes_this_one if {
	count(_findings_net3({"resources": [
		{"provider": "outscale", "type": "compute_instance", "id": "i-1", "attributes": {"vm_id": "i-1", "security_group_ids": ["sg-web"]}},
		{"provider": "outscale", "type": "network_interface", "id": "eni-9", "attributes": {"nic_id": "eni-9", "vm_id": "i-2", "public_ip": "198.51.100.9", "security_group_ids": ["sg-ssh"]}},
		{"provider": "outscale", "type": "security_group_rule", "id": "sg-ssh", "attributes": _ssh_ouvert("sg-ssh")},
	]})) == 0
}

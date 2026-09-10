package pepin.rules

import rego.v1

# ── UNE MACHINE PRIVÉE N'A RIEN À FILTRER (#212) ──────────────────────────────

_vms(vs) := {"resources": [{"provider": "exoscale", "type": "compute_instance", "id": v.vm_id, "attributes": v} | some v in vs]}

_sg_code := "compute_instance_has_security_group"

_sg_findings(vs) := {f | some f in deny with input as _vms(vs); f.code == _sg_code}

# ✓ Une instance sans interface publique se voit attacher une liste vide PAR
# CONSTRUCTION : l'API le fait, l'exploitant ne l'a pas choisi, et aucune remédiation
# n'y change rien. Le pire des faux positifs — systématique et impossible à corriger.
test_a_private_instance_has_nothing_to_filter if {
	count(_sg_findings([{"vm_id": "i-privee", "security_group_ids": [], "public_interface": false}])) == 0
}

# ✗ LE CONTRE-EXEMPLE, et c'est lui qui garde le contrôle utile : une instance AVEC une
# interface publique et sans groupe reste un écart critique. Sans lui, la correction
# éteindrait le contrôle au lieu de le préciser.
test_a_public_instance_without_a_group_is_still_denied if {
	fs := _sg_findings([{"vm_id": "i-publique", "security_group_ids": [], "public_interface": true}])
	count(fs) == 1
	some f in fs
	f.severity == "critical"
}

# L'attribut ABSENT ne vaut pas « privée » : ne pas savoir n'est pas savoir que non, et
# le contrôle continue de parler comme avant chez les fournisseurs qui ne publient pas
# ce champ.
test_an_absent_attribute_does_not_silence_the_control if {
	count(_sg_findings([{"vm_id": "i-inconnue", "security_group_ids": []}])) == 1
}

# Une instance privée QUI PORTE un groupe n'est pas concernée non plus : le contrôle ne
# vise que l'absence de filtrage.
test_a_private_instance_with_a_group_stays_silent if {
	count(_sg_findings([{"vm_id": "i-p", "security_group_ids": ["sg-1"], "public_interface": false}])) == 0
}

package pepin.rules

import rego.v1

# ✗ Région hors espace souverain (Outscale us-east-2) → écart MAJEUR (high).
test_region_outside_eu_denied_high if {
	some f in deny with input as {"resources": [{"provider": "outscale", "type": "compute_instance", "id": "i-1", "name": "vm-us", "region": "us-east-2", "attributes": {}}]}
	f.code == "governance_resource_region_in_eu"
	f.severity == "high"
	f.labels.provider == "outscale"
}

# ~ Suisse (espace européen de confiance) → écart MINEUR (low), pas high.
test_exoscale_switzerland_denied_low if {
	some f in deny with input as {"resources": [{"provider": "exoscale", "type": "compute_instance", "id": "i-2", "name": "vm-ch", "region": "ch-gva-2", "attributes": {}}]}
	f.code == "governance_resource_region_in_eu"
	f.severity == "low"
}

# ✓ Région UE → aucun finding (Scaleway fr-par, Outscale eu-west-2).
test_region_in_eu_ok if {
	count([f | some f in deny; f.code == "governance_resource_region_in_eu"]) == 0 with input as {"resources": [
		{"provider": "scaleway", "type": "object_storage_bucket", "id": "b1", "name": "buck", "region": "fr-par", "attributes": {}},
		{"provider": "outscale", "type": "compute_instance", "id": "i-3", "name": "vm-fr", "region": "eu-west-2", "attributes": {}},
	]}
}

# ✗ Région RENSEIGNÉE mais non cataloguée → écart MOYEN. Les tables sont des
# listes blanches : leur silence valait « conforme », donc une région hors UE
# ouverte demain aurait été certifiée UE jusqu'à ce qu'on pense à éditer lib.rego.
test_unknown_region_denied_medium if {
	some f in deny with input as {"resources": [{"provider": "scaleway", "type": "compute_instance", "id": "i-4", "name": "vm-x", "region": "zz-unknown-1", "attributes": {}}]}
	f.code == "governance_resource_region_in_eu"
	f.severity == "medium"
}

# ✓ Région ABSENTE → toujours aucun finding : la source n'expose pas l'attribut,
# il n'y a rien à qualifier. C'est le verrou de couverture (requiredAttr) qui
# traite ce cas, pas une règle de posture qui inventerait un écart.
test_empty_region_skipped if {
	count([f | some f in deny; f.code == "governance_resource_region_in_eu"]) == 0 with input as {"resources": [{"provider": "scaleway", "type": "compute_instance", "id": "i-5", "name": "vm-y", "region": "", "attributes": {}}]}
}

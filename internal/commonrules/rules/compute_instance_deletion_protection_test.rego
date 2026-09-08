package pepin.rules

import rego.v1

_dp_vm(attrs) := {"resources": [{"provider": "outscale", "type": "compute_instance", "id": "i-9", "attributes": attrs}]}

_dp_code := "compute_instance_deletion_protection"

# ✗ Production sans protection → écart, et il est CONCLUANT.
test_production_without_protection_denied if {
	some f in deny with input as _dp_vm({"vm_id": "i-9", "deletion_protection": false, "tags": [{"key": "Env", "value": "production"}]})
	f.code == _dp_code
	not f.labels.inconclusive
}

# ✓ LE CONTRE-EXEMPLE. Une VM de développement, éphémère par construction, ne doit
# pas produire le même écart qu'un serveur de production. Ce faux positif contextuel
# use la confiance sans rien apprendre.
test_development_without_protection_is_not_a_deviation if {
	count({f | some f in deny; f.code == _dp_code}) == 0 with input as _dp_vm({"vm_id": "i-9", "deletion_protection": false, "tags": [{"key": "environment", "value": "dev"}]})
}

# ✓ Production AVEC protection → rien.
test_production_with_protection_ok if {
	count({f | some f in deny; f.code == _dp_code}) == 0 with input as _dp_vm({"vm_id": "i-9", "deletion_protection": true, "tags": [{"key": "Env", "value": "prod"}]})
}

# ✗→? Sans étiquette d'environnement, on ne SAIT pas. Se taire vaudrait « conforme »
# et laisserait passer un vrai serveur non protégé (ADR-0015).
test_unknown_environment_is_inconclusive if {
	some f in deny with input as _dp_vm({"vm_id": "i-9", "deletion_protection": false})
	f.code == _dp_code
	f.labels.inconclusive == "true"
}

# ✓ L'écriture de l'environnement est insensible à la casse et aux séparateurs,
# comme celle des noms d'étiquettes : « PROD » vaut « prod ».
test_production_value_is_case_insensitive if {
	some f in deny with input as _dp_vm({"vm_id": "i-9", "deletion_protection": false, "tags": [{"key": "stage", "value": "PROD"}]})
	f.code == _dp_code
	not f.labels.inconclusive
}

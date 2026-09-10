package pepin.rules

import rego.v1

# ── UNE IDENTITÉ QUE LA PLATEFORME GÈRE (#213) ────────────────────────────────
#
# SKS crée un rôle `sks-ccm-<cluster>` pour son cloud controller manager, s'en sert et
# le supprime avec le cluster. Il est `editable: true`, donc le filtre des rôles
# prédéfinis ne le couvrait pas — et chaque cluster apportait DEUX écarts IAM que
# personne ne pouvait faire disparaître. Dix clusters, vingt findings irréparables.

_role_gere := {"resources": [{"provider": "exoscale", "type": "iam_role", "id": "sks-ccm-abc", "attributes": {
	"name": "sks-ccm-abc",
	"role_id": "sks-ccm-abc",
	"editable": true,
	"provider_managed": true,
	"source_ip_restricted": false,
	"policy_has_expiration": false,
	"max_session_ttl": 0,
	"admin_privileges": true,
}}]}

_role_exploitant := {"resources": [{"provider": "exoscale", "type": "iam_role", "id": "ci-deployer", "attributes": {
	"name": "ci-deployer",
	"role_id": "ci-deployer",
	"editable": true,
	"source_ip_restricted": false,
	"policy_has_expiration": false,
	"max_session_ttl": 0,
	"admin_privileges": true,
}}]}

_codes_role := {"iam_role_source_ip_restricted", "iam_role_key_lifetime_bounded", "iam_role_no_admin_privileges"}

# ✓ Le rôle géré par la plateforme ne produit AUCUN des trois écarts : sa remédiation
# n'aboutirait pas — borner l'IP source exigerait de deviner les adresses du plan de
# contrôle, et une clause de durée casserait le composant.
test_a_provider_managed_role_raises_nothing if {
	count({f | some f in deny with input as _role_gere; f.code in _codes_role}) == 0
}

# ✗ LE CONTRE-EXEMPLE, et c'est lui qui garde les trois contrôles utiles : le MÊME rôle,
# aux mêmes attributs, sans la marque. Sans ce test, la correction éteindrait trois
# contrôles au lieu d'en préciser un.
test_the_same_role_without_the_mark_still_raises_all_three if {
	fs := {f.code | some f in deny with input as _role_exploitant; f.code in _codes_role}
	count(fs) == 3
}

# L'attribut ABSENT ne vaut pas « géré par la plateforme » : un fournisseur qui ne
# déclare rien ne voit aucun changement. C'est ce que `_role_exploitant` montre déjà,
# et l'énoncer séparément évite qu'un jour un défaut de collecte fasse taire les trois.
test_an_absent_mark_never_silences_a_control if {
	not gere_par_le_fournisseur({"attributes": {"name": "sks-ccm-abc"}})
	gere_par_le_fournisseur({"attributes": {"provider_managed": true}})
}

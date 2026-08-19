# Booléens transmis en CHAÎNE — régression transversale.
#
# Un plan Terraform rend certains attributs de schéma sous forme de chaîne
# (« true » / « false ») là où l'API live rend un bool. Les règles qui
# comparaient en dur (`… == false`) étaient donc muettes sur « false » : la
# configuration dangereuse existait, la règle ne se déclenchait pas, et le scan
# concluait « conforme ». Fail-open silencieux, le pire défaut d'un CSPM.
#
# Ces tests fixent le comportement attendu des DEUX formes. Ils échouent sur le
# code d'avant la généralisation de truthy() et passent après : c'est ce qui en
# fait une mesure et non une paraphrase.
package pepin.rules

import rego.v1

_res(type, attrs) := {"resources": [{"provider": "scaleway", "type": type, "id": "r1", "attributes": attrs}]}

_denied(code, type, attrs) if {
	some f in deny with input as _res(type, attrs)
	f.code == code
}

# --- stockage objet ---------------------------------------------------------

test_bucket_object_lock_disabled_as_string_denied if {
	_denied("objectstorage_bucket_object_lock_enabled", "object_storage_bucket", {"name": "backups", "object_lock_enabled": "false"})
}

test_bucket_default_encryption_disabled_as_string_denied if {
	_denied("objectstorage_bucket_default_encryption", "object_storage_bucket", {"name": "backups", "default_encryption_enabled": "false"})
}

# Un bucket rendu public par une policy, le drapeau arrivant en chaîne : c'est un
# contrôle `critical`, celui dont le silence coûte le plus cher.
test_bucket_policy_public_as_string_denied if {
	_denied("objectstorage_bucket_public_access", "object_storage_bucket", {"name": "backups", "policy_public": "true"})
}

# --- stockage bloc ----------------------------------------------------------

test_volume_unencrypted_as_string_denied if {
	_denied("blockstorage_volume_encryption", "blockstorage_volume", {"name": "data", "encrypted": "false"})
}

# --- calcul -----------------------------------------------------------------

test_instance_deletion_protection_off_as_string_denied if {
	_denied("compute_instance_deletion_protection", "compute_instance", {"name": "web", "deletion_protection": "false"})
}

# --- kubernetes managé ------------------------------------------------------

test_cluster_audit_logging_off_as_string_denied if {
	_denied("kubernetes_cluster_audit_logging_enabled", "kubernetes_cluster", {"name": "k1", "audit_enabled": "false"})
}

test_cluster_auto_upgrade_off_as_string_denied if {
	_denied("kubernetes_cluster_auto_upgrade_enabled", "kubernetes_cluster", {"name": "k1", "auto_upgrade": "false"})
}

# --- la forme booléenne native reste évidemment détectée --------------------

test_native_bool_still_denied if {
	_denied("objectstorage_bucket_object_lock_enabled", "object_storage_bucket", {"name": "backups", "object_lock_enabled": false})
}

# --- et un attribut réellement conforme ne déclenche pas, dans les deux formes

test_compliant_bool_not_denied if {
	count({f | some f in deny; f.code == "objectstorage_bucket_object_lock_enabled"}) == 0 with input as _res("object_storage_bucket", {"name": "backups", "object_lock_enabled": true})
}

test_compliant_string_not_denied if {
	count({f | some f in deny; f.code == "objectstorage_bucket_object_lock_enabled"}) == 0 with input as _res("object_storage_bucket", {"name": "backups", "object_lock_enabled": "true"})
}

package pepin.rules

import rego.v1

_mdb(attrs) := {"resources": [{"provider": "scaleway", "type": "managed_database", "id": "db-1", "attributes": attrs}]}

# ── database_encryption_at_rest_enabled ──
test_db_no_encryption_denied if {
	some f in deny with input as _mdb({"database_id": "db-1", "encryption_at_rest": false})
	f.code == "database_encryption_at_rest_enabled"
}

test_db_encryption_string_false_denied if {
	some f in deny with input as _mdb({"database_id": "db-1", "encryption_at_rest": "false"})
	f.code == "database_encryption_at_rest_enabled"
}

test_db_encryption_on_ok if {
	count({f | some f in deny; f.code == "database_encryption_at_rest_enabled"}) == 0 with input as _mdb({"database_id": "db-1", "encryption_at_rest": true})
}

# ✓ Attribut non collecté (garde de capacité) → pas de faux positif.
test_db_encryption_uncollected_ok if {
	count({f | some f in deny; f.code == "database_encryption_at_rest_enabled"}) == 0 with input as _mdb({"database_id": "db-1"})
}

# ── database_backup_enabled ──
test_db_backup_disabled_denied if {
	some f in deny with input as _mdb({"database_id": "db-1", "disable_backup": true})
	f.code == "database_backup_enabled"
}

test_db_backup_enabled_ok if {
	count({f | some f in deny; f.code == "database_backup_enabled"}) == 0 with input as _mdb({"database_id": "db-1", "disable_backup": false})
}

# ✓ Absent ⇒ sauvegardes actives par défaut → pas de finding.
test_db_backup_default_ok if {
	count({f | some f in deny; f.code == "database_backup_enabled"}) == 0 with input as _mdb({"database_id": "db-1"})
}

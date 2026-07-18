package pepin.rules

import rego.v1

# ---- blockstorage_snapshot_not_public ----
test_snapshot_public_denied if {
	some f in deny with input as {"resources": [{"provider": "outscale", "type": "blockstorage_snapshot", "id": "snap-1", "attributes": {"snapshot_id": "snap-1", "global_permission": true}}]}
	f.code == "blockstorage_snapshot_not_public"
}

test_snapshot_private_ok if {
	count({f | some f in deny; f.code == "blockstorage_snapshot_not_public"}) == 0 with input as {"resources": [{"provider": "outscale", "type": "blockstorage_snapshot", "id": "snap-1", "attributes": {"snapshot_id": "snap-1", "global_permission": false}}]}
}

# ---- blockstorage_volume_snapshots_exist ----
test_volume_no_snapshot_denied if {
	some f in deny with input as {"resources": [{"provider": "outscale", "type": "blockstorage_volume", "id": "vol-1", "attributes": {"volume_id": "vol-1", "state": "in-use"}}]}
	f.code == "blockstorage_volume_snapshots_exist"
}

test_volume_recent_snapshot_ok if {
	doc := {"resources": [
		{"provider": "outscale", "type": "blockstorage_volume", "id": "vol-1", "attributes": {"volume_id": "vol-1", "state": "in-use"}},
		{"provider": "outscale", "type": "blockstorage_snapshot", "id": "snap-1", "attributes": {"volume_id": "vol-1", "creation_date": "2099-01-01T00:00:00Z"}},
	]}
	count({f | some f in deny; f.code == "blockstorage_volume_snapshots_exist"}) == 0 with input as doc
}

test_volume_available_ok if {
	count({f | some f in deny; f.code == "blockstorage_volume_snapshots_exist"}) == 0 with input as {"resources": [{"provider": "outscale", "type": "blockstorage_volume", "id": "vol-1", "attributes": {"volume_id": "vol-1", "state": "available"}}]}
}

# Exoscale : état natif « attached » = en usage → finding (sans snapshot récent).
test_volume_attached_exoscale_denied if {
	some f in deny with input as {"resources": [{"provider": "exoscale", "type": "blockstorage_volume", "id": "v-1", "attributes": {"volume_id": "v-1", "state": "attached"}}]}
	f.code == "blockstorage_volume_snapshots_exist"
}

# Exoscale : volume « detached » = non rattaché → pas de finding.
test_volume_detached_exoscale_ok if {
	count({f | some f in deny; f.code == "blockstorage_volume_snapshots_exist"}) == 0 with input as {"resources": [{"provider": "exoscale", "type": "blockstorage_volume", "id": "v-1", "attributes": {"volume_id": "v-1", "state": "detached"}}]}
}

# ---- compute_instance_has_security_group ----
test_vm_no_sg_denied if {
	some f in deny with input as {"resources": [{"provider": "outscale", "type": "compute_instance", "id": "i-1", "attributes": {"vm_id": "i-1", "security_group_ids": []}}]}
	f.code == "compute_instance_has_security_group"
}

test_vm_with_sg_ok if {
	count({f | some f in deny; f.code == "compute_instance_has_security_group"}) == 0 with input as {"resources": [{"provider": "outscale", "type": "compute_instance", "id": "i-1", "attributes": {"vm_id": "i-1", "security_group_ids": ["sg-1"]}}]}
}

# ---- compute_instance_no_secrets_in_user_data ----
test_userdata_secret_denied if {
	some f in deny with input as {"resources": [{"provider": "outscale", "type": "compute_instance", "id": "i-1", "attributes": {"vm_id": "i-1", "security_group_ids": ["sg-1"], "user_data": "export KEY=AKIAABCDEFGHIJKLMNOP"}}]}
	f.code == "compute_instance_no_secrets_in_user_data"
}

test_userdata_clean_ok if {
	count({f | some f in deny; f.code == "compute_instance_no_secrets_in_user_data"}) == 0 with input as {"resources": [{"provider": "outscale", "type": "compute_instance", "id": "i-1", "attributes": {"vm_id": "i-1", "security_group_ids": ["sg-1"], "user_data": "echo bonjour le monde"}}]}
}

test_userdata_scaleway_key_denied if {
	some f in deny with input as {"resources": [{"provider": "scaleway", "type": "compute_instance", "id": "i-2", "attributes": {"vm_id": "i-2", "security_group_ids": ["sg-1"], "user_data": "export SCW_ACCESS_KEY=SCWABCDEFGHIJKLMNOPQ"}}]}
	f.code == "compute_instance_no_secrets_in_user_data"
}

test_userdata_github_token_denied if {
	some f in deny with input as {"resources": [{"provider": "exoscale", "type": "compute_instance", "id": "i-3", "attributes": {"vm_id": "i-3", "security_group_ids": ["sg-1"], "user_data": "git clone https://ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789@github.com/x/y"}}]}
	f.code == "compute_instance_no_secrets_in_user_data"
}

test_userdata_jwt_denied if {
	some f in deny with input as {"resources": [{"provider": "outscale", "type": "compute_instance", "id": "i-4", "attributes": {"vm_id": "i-4", "security_group_ids": ["sg-1"], "user_data": "TOKEN=eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3In0.abcdefghij"}}]}
	f.code == "compute_instance_no_secrets_in_user_data"
}

test_userdata_generic_apikey_denied if {
	some f in deny with input as {"resources": [{"provider": "scaleway", "type": "compute_instance", "id": "i-5", "attributes": {"vm_id": "i-5", "security_group_ids": ["sg-1"], "user_data": "api_key: 0123456789abcdef0123"}}]}
	f.code == "compute_instance_no_secrets_in_user_data"
}

test_userdata_password_inline_denied if {
	some f in deny with input as {"resources": [{"provider": "outscale", "type": "compute_instance", "id": "i-6", "attributes": {"vm_id": "i-6", "security_group_ids": ["sg-1"], "user_data": "DB_PASSWORD=Sup3rSecret!"}}]}
	f.code == "compute_instance_no_secrets_in_user_data"
}

# ✓ FP-3 : bloc cloud-init `chpasswd` (valeur en bloc YAML, pas de secret en clair sur la
# ligne) ne doit PAS déclencher — le motif exigeait auparavant seulement `\S` après le séparateur.
test_userdata_cloudinit_chpasswd_ok if {
	count({f | some f in deny; f.code == "compute_instance_no_secrets_in_user_data"}) == 0 with input as {"resources": [{"provider": "scaleway", "type": "compute_instance", "id": "i-7", "attributes": {"vm_id": "i-7", "security_group_ids": ["sg-1"], "user_data": "#cloud-config\nchpasswd:\n  expire: true\nssh_pwauth: false\n"}}]}
}

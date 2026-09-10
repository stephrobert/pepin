# Sorties : les identifiants que la collecte LIVE emploie comme sujets, connus après
# apply. expected.yaml les cite sous la forme `${output.<nom>}`. Aucune sortie sensible.

output "zone" { value = var.zone }
output "sg_ssh_open_id" { value = exoscale_security_group.ssh_open.id }
output "sg_rdp_open_id" { value = exoscale_security_group.rdp_open.id }
output "sg_db_open_id" { value = exoscale_security_group.db_open.id }
output "sg_snmp_open_id" { value = exoscale_security_group.snmp_open.id }
output "sg_any_open_id" { value = exoscale_security_group.any_open.id }
output "sg_quartet_id" { value = exoscale_security_group.quartet.id }
output "sg_egress_any_id" { value = exoscale_security_group.egress_any.id }
output "sg_undocumented_flow_id" { value = exoscale_security_group.undocumented_flow.id }
output "sg_hardened_id" { value = exoscale_security_group.hardened.id }
output "vm_exposed_id" { value = exoscale_compute_instance.exposed.id }
output "vm_untagged_id" { value = exoscale_compute_instance.untagged.id }
output "vm_secrets_id" { value = exoscale_compute_instance.secrets.id }
output "vm_hardened_id" { value = exoscale_compute_instance.hardened.id }
output "volume_unsnapshotted_id" { value = exoscale_block_storage_volume.unsnapshotted.id }
output "volume_snapshotted_id" { value = exoscale_block_storage_volume.snapshotted.id }
output "snapshot_private_id" { value = exoscale_block_storage_volume_snapshot.private.id }
output "pn_undocumented_id" { value = exoscale_private_network.undocumented.id }
output "pn_documented_id" { value = exoscale_private_network.documented.id }
output "sks_weak_name" { value = exoscale_sks_cluster.weak.name }
output "sks_strong_name" { value = exoscale_sks_cluster.strong.name }
output "role_admin_id" { value = exoscale_iam_role.admin.id }
output "role_no_source_ip_id" { value = exoscale_iam_role.no_source_ip.id }
output "role_unbounded_id" { value = exoscale_iam_role.unbounded.id }
output "role_restricted_id" { value = exoscale_iam_role.restricted.id }

# Suffixe des noms de buckets SOS (créés par le crochet `extra`, hors Terraform).
output "bucket_suffix" { value = local.bucket_suffix }
output "bucket_public" { value = "pepin-qual-${local.bucket_suffix}-public" }
output "bucket_unversioned" { value = "pepin-qual-${local.bucket_suffix}-unversioned" }
output "bucket_hardened" { value = "pepin-qual-${local.bucket_suffix}-hardened" }

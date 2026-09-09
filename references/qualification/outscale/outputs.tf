# Sorties : les identifiants que la collecte LIVE emploie comme sujets, connus
# après apply. expected.yaml les cite sous la forme `${output.<nom>}`. À plat, une
# sortie par sujet. Aucune sortie sensible.

output "sg_ssh_open_id" { value = outscale_security_group.ssh_open.security_group_id }
output "sg_rdp_open_id" { value = outscale_security_group.rdp_open.security_group_id }
output "sg_db_open_id" { value = outscale_security_group.db_open.security_group_id }
output "sg_snmp_open_id" { value = outscale_security_group.snmp_open.security_group_id }
output "sg_any_open_id" { value = outscale_security_group.any_open.security_group_id }
output "sg_quartet_id" { value = outscale_security_group.quartet.security_group_id }
output "sg_egress_any_id" { value = outscale_security_group.egress_any.security_group_id }
output "sg_hardened_id" { value = outscale_security_group.hardened.security_group_id }
output "sg_net_ssh_open_id" { value = outscale_security_group.net_ssh_open.security_group_id }
output "sg_net_closed_id" { value = outscale_security_group.net_closed.security_group_id }
output "sg_net_default_id" { value = data.outscale_security_group.net_default.security_group_id }

output "vm_exposed_id" { value = outscale_vm.public["exposed"].vm_id }
output "vm_private_id" { value = outscale_vm.public["private"].vm_id }
output "vm_hardened_id" { value = outscale_vm.public["hardened"].vm_id }
output "vm_untagged_id" { value = outscale_vm.public["untagged"].vm_id }
output "vm_secrets_id" { value = outscale_vm.public["secrets"].vm_id }
output "vm_unprotected_id" { value = outscale_vm.public["unprotected"].vm_id }
output "vm_two_nics_id" { value = outscale_vm.two_nics.vm_id }
output "vm_two_nics_inverse_id" { value = outscale_vm.two_nics_inverse.vm_id }

# Volumes racine, un par VM : tous en usage, aucun sauvegardé.
output "root_volume_exposed_id" { value = tolist(outscale_vm.public["exposed"].block_device_mappings_created[0].bsu)[0].volume_id }
output "root_volume_private_id" { value = tolist(outscale_vm.public["private"].block_device_mappings_created[0].bsu)[0].volume_id }
output "root_volume_hardened_id" { value = tolist(outscale_vm.public["hardened"].block_device_mappings_created[0].bsu)[0].volume_id }
output "root_volume_untagged_id" { value = tolist(outscale_vm.public["untagged"].block_device_mappings_created[0].bsu)[0].volume_id }
output "root_volume_secrets_id" { value = tolist(outscale_vm.public["secrets"].block_device_mappings_created[0].bsu)[0].volume_id }
output "root_volume_unprotected_id" { value = tolist(outscale_vm.public["unprotected"].block_device_mappings_created[0].bsu)[0].volume_id }
output "root_volume_two_nics_id" { value = tolist(outscale_vm.two_nics.block_device_mappings_created[0].bsu)[0].volume_id }
output "root_volume_two_nics_inverse_id" { value = tolist(outscale_vm.two_nics_inverse.block_device_mappings_created[0].bsu)[0].volume_id }

output "volume_unsnapshotted_id" { value = outscale_volume.unsnapshotted.volume_id }
output "volume_snapshotted_id" { value = outscale_volume.snapshotted.volume_id }
output "snapshot_public_id" { value = outscale_snapshot.public.snapshot_id }
output "snapshot_private_id" { value = outscale_snapshot.private.snapshot_id }
output "image_public_id" { value = outscale_image.public.image_id }
output "image_private_id" { value = outscale_image.private.image_id }

output "net_undocumented_id" { value = outscale_net.undocumented.net_id }
output "net_documented_id" { value = outscale_net.documented.net_id }
output "subnet_autoip_id" { value = outscale_subnet.autoip.subnet_id }
output "subnet_main_id" { value = outscale_subnet.main.subnet_id }
output "peering_id" { value = outscale_net_peering.same_account.net_peering_id }
output "nic_two_nics_id" { value = outscale_nic.two_nics_secondary.nic_id }
output "nic_two_nics_inverse_id" { value = outscale_nic.two_nics_inverse_secondary.nic_id }

output "api_key_no_expiry" { value = outscale_access_key.no_expiry.access_key_id }
output "api_key_expiring" { value = outscale_access_key.expiring.access_key_id }
output "api_key_root" { value = outscale_access_key.root.access_key_id }
output "api_rule_public_id" { value = outscale_api_access_rule.public.api_access_rule_id }
output "api_rule_private_id" { value = outscale_api_access_rule.private.api_access_rule_id }
output "user_name" { value = outscale_user.qual.user_name }

# Suffixe des noms de buckets OOS (créés par le crochet `extra`, hors Terraform).
output "bucket_suffix" { value = local.bucket_suffix }
output "bucket_public" { value = "pepin-qual-${local.bucket_suffix}-public" }
output "bucket_policy" { value = "pepin-qual-${local.bucket_suffix}-policy" }
output "bucket_unversioned" { value = "pepin-qual-${local.bucket_suffix}-unversioned" }
output "bucket_unlocked" { value = "pepin-qual-${local.bucket_suffix}-unlocked" }
output "bucket_unencrypted" { value = "pepin-qual-${local.bucket_suffix}-unencrypted" }
output "bucket_hardened" { value = "pepin-qual-${local.bucket_suffix}-hardened" }

output "lb_http_name" { value = outscale_load_balancer.http.load_balancer_name }
output "lb_tcp443_name" { value = outscale_load_balancer.tcp443.load_balancer_name }

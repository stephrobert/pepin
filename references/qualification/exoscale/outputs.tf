# Sorties : en plan seul, les identifiants ne sont pas connus ; expected.yaml
# désigne les ressources par leur adresse ou leur nom. Ces sorties serviront à la
# moitié live, quand un compte existera.

output "sg_ssh_open_id" { value = exoscale_security_group.ssh_open.id }
output "sg_hardened_id" { value = exoscale_security_group.hardened.id }
output "vm_untagged_id" { value = exoscale_compute_instance.untagged.id }
output "vm_secrets_id" { value = exoscale_compute_instance.secrets.id }
output "vm_swiss_id" { value = exoscale_compute_instance.swiss.id }
output "vm_hardened_id" { value = exoscale_compute_instance.hardened.id }
output "volume_data_id" { value = exoscale_block_storage_volume.data.id }
output "sks_weak_name" { value = exoscale_sks_cluster.weak.name }
output "sks_strong_name" { value = exoscale_sks_cluster.strong.name }
output "role_admin_id" { value = exoscale_iam_role.admin.id }

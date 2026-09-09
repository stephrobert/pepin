# Sorties : les identifiants que la collecte LIVE emploie comme sujets de finding
# et qui ne sont connus qu'après apply. expected.yaml les cite sous la forme
# `${output.<nom>}`, et tools/qualification/qualify.py les résout depuis
# `terraform output -json` AVANT le destroy.
#
# Les identifiants zonés (zone/uuid) sont rendus SANS leur zone : c'est la forme
# que le collecteur live emploie comme sujet (SecurityGroup.ID, Server.ID).
#
# Aucune sortie sensible : les clés secrètes ne sortent jamais de l'état, et
# l'état lui-même est local, non versionné, détruit avec le tenant.

output "sg_ssh_open_id" {
  value = split("/", scaleway_instance_security_group.ssh_open.id)[1]
}

output "sg_rdp_open_id" {
  value = split("/", scaleway_instance_security_group.rdp_open.id)[1]
}

output "sg_db_open_id" {
  value = split("/", scaleway_instance_security_group.db_open.id)[1]
}

output "sg_snmp_open_id" {
  value = split("/", scaleway_instance_security_group.snmp_open.id)[1]
}

output "sg_any_open_id" {
  value = split("/", scaleway_instance_security_group.any_open.id)[1]
}

output "sg_quartet_id" {
  value = split("/", scaleway_instance_security_group.quartet.id)[1]
}

output "sg_egress_any_id" {
  value = split("/", scaleway_instance_security_group.egress_any.id)[1]
}

output "sg_default_accept_id" {
  value = split("/", scaleway_instance_security_group.default_accept.id)[1]
}

output "sg_hardened_id" {
  value = split("/", scaleway_instance_security_group.hardened.id)[1]
}

output "vm_exposed_id" {
  value = split("/", scaleway_instance_server.exposed.id)[1]
}

output "vm_private_id" {
  value = split("/", scaleway_instance_server.private.id)[1]
}

output "vm_hardened_id" {
  value = split("/", scaleway_instance_server.hardened.id)[1]
}

output "vm_untagged_id" {
  value = split("/", scaleway_instance_server.untagged.id)[1]
}

output "vm_secrets_id" {
  value = split("/", scaleway_instance_server.secrets.id)[1]
}

output "api_key_expired" {
  value = scaleway_iam_api_key.expired.access_key
}

output "api_key_expiring" {
  value = scaleway_iam_api_key.expiring.access_key
}

output "bucket_public" {
  value = scaleway_object_bucket.public.name
}

output "bucket_policy" {
  value = scaleway_object_bucket.policy.name
}

output "bucket_unversioned" {
  value = scaleway_object_bucket.unversioned.name
}

output "bucket_unlocked" {
  value = scaleway_object_bucket.unlocked.name
}

output "bucket_sensitive" {
  value = scaleway_object_bucket.sensitive.name
}

output "bucket_hardened" {
  value = scaleway_object_bucket.hardened.name
}

output "bucket_encrypted" {
  value = scaleway_object_bucket.encrypted.name
}

# Ce que le tenant crée, par famille — APPLIQUÉ (vu en live) et PLAN SEULEMENT.
# Sert au README et au rapport de la porte.
output "resource_counts" {
  value = {
    applied = {
      iam_applications = 1
      iam_api_keys     = 2
      security_groups  = 9
      flexible_ips     = 2
      servers          = 5
      buckets          = 7
    }
    terraform_only = {
      iam_applications = 1
      iam_api_keys     = 1
      iam_policies     = 2
      vpcs             = 1
      private_networks = 2
      rdb_instances    = 2
    }
  }
}

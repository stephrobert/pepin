# Exemple Terraform Scaleway — CONTREPARTIE CORRIGÉE de examples/scaleway/terraform/main.tf.
# Même montage, mêmes ressources, mais chaque écart relevé par Pépin sur la version
# non conforme y est corrigé. Sert le parcours de docs/getting-started/quickstart.md :
# scanner les DEUX plans montre que la correction change réellement le verdict.
#
# Rejouer le plan (aucune ressource créée, aucun compte requis) :
#   terraform init && terraform plan -out tfplan && terraform show -json tfplan > plan.json

terraform {
  required_providers {
    scaleway = {
      source = "scaleway/scaleway"
    }
    random = {
      source = "hashicorp/random"
    }
  }
}

# ✓ Chiffrement au repos activé + sauvegardes automatiques conservées
#   → database_encryption_at_rest_enabled (CLD-CHF-2), database_backup_enabled (CLD-STO-3)
resource "random_password" "db" {
  length = 20
}

resource "scaleway_rdb_instance" "secure" {
  name               = "pepin-test-rdb"
  node_type          = "DB-DEV-S"
  engine             = "PostgreSQL-15"
  user_name          = "admin"
  password           = random_password.db.result
  encryption_at_rest = true
  disable_backup     = false
}

# ✓ SSH (22) restreint au réseau d'administration, plus d'origine 0.0.0.0/0
#   → network_securitygroup_allow_ingress_from_internet_to_tcp_port_22
resource "scaleway_instance_security_group" "web" {
  name                   = "sg-front"
  inbound_default_policy = "drop"

  inbound_rule {
    action   = "accept"
    port     = 22
    ip_range = "10.0.0.0/8"
  }

  inbound_rule {
    action   = "accept"
    port     = 443
    ip_range = "10.0.0.0/8"
  }
}

# ✓ Politique entrante par défaut « drop » : seuls les flux explicites sont admis
#   → network_securitygroup_default_deny (CLD-NET-2)
resource "scaleway_instance_security_group" "closed_default" {
  name                    = "sg-closed-default"
  inbound_default_policy  = "drop"
  outbound_default_policy = "accept"
}

# ✓ ACL de base managée restreinte au réseau applicatif : plus de 0.0.0.0/0
#   → database_service_not_open_to_internet (CLD-NET-1)
resource "scaleway_rdb_acl" "private_db" {
  instance_id = "fr-par/11111111-1111-1111-1111-111111111111"

  acl_rules {
    ip          = "10.0.0.0/8"
    description = "réseau applicatif interne"
  }
}

# ✓ Bucket privé et immuable (Object Lock/WORM activé)
#   → objectstorage_bucket_public_access, objectstorage_bucket_object_lock_enabled
resource "scaleway_object_bucket" "backups" {
  name                = "backups-prod"
  object_lock_enabled = true
}

resource "scaleway_object_bucket_acl" "backups" {
  bucket = scaleway_object_bucket.backups.id
  acl    = "private"
}

# ✓ Politique IAM sans PermissionSet de gestion de l'IAM : plus de chemin
#   d'élévation de privilèges → iam_policy_no_privilege_escalation (CLD-IAM-12)
resource "scaleway_iam_policy" "deployer" {
  name         = "ci-deployer"
  no_principal = true
  rule {
    organization_id      = "11111111-1111-1111-1111-111111111111"
    permission_set_names = ["ObjectStorageObjectsRead"]
  }
}

# ✓ Aucun secret dans le user-data, groupe de sécurité attaché, étiquettes de
#   gouvernance posées → compute_instance_no_secrets_in_user_data (CLD-CMP-2),
#   compute_instance_has_security_group (CLD-CMP-1),
#   governance_resource_required_tags (CLD-GVN-1)
#   security_group_id littéral (et non une référence au SG du même plan) : une
#   référence sortirait « known after apply », donc non observable au stade plan.
resource "scaleway_instance_server" "web" {
  type              = "DEV1-S"
  image             = "ubuntu_jammy"
  security_group_id = "22222222-2222-2222-2222-222222222222"
  tags              = ["CostCenter=RD", "Project=pepin", "Env=demo", "Owner=platform"]
  user_data = {
    "cloud-init" = "#cloud-config\nruncmd:\n  - systemctl enable --now nginx"
  }
}

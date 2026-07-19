# Exemple Terraform Scaleway — code volontairement NON CONFORME, destiné à être
# audité par Pépin (`pepin scan scaleway --terraform`).
# Adapté des exemples officiels du provider :
#   registry.terraform.io/providers/scaleway/scaleway/latest/docs/resources/instance_security_group
#   registry.terraform.io/providers/scaleway/scaleway/latest/docs/resources/object_bucket_acl

terraform {
  required_providers {
    scaleway = {
      source = "scaleway/scaleway"
    }
  }
}

# ✗ SSH (22) accepté depuis 0.0.0.0/0 (ip_range omis ⇒ toute origine)
#   → network_securitygroup_allow_ingress_from_internet_to_tcp_port_22
resource "scaleway_instance_security_group" "web" {
  name                   = "sg-front"
  inbound_default_policy = "drop"

  inbound_rule {
    action = "accept"
    port   = 22
  }

  inbound_rule {
    action   = "accept"
    port     = 443
    ip_range = "10.0.0.0/8"
  }
}

# ✗ Politique entrante par défaut « accept » : tout trafic non filtré est admis
#   → network_securitygroup_default_deny (CLD-NET-2)
#   Forme reprise de jpetazzo/container.training (Kapsule custom SG, accept par défaut).
resource "scaleway_instance_security_group" "open_default" {
  name                    = "sg-open-default"
  inbound_default_policy  = "accept"
  outbound_default_policy = "accept"
}

# ✗ ACL de base de données managée autorisant 0.0.0.0/0 : base joignable depuis Internet
#   → database_service_not_open_to_internet (CLD-NET-1)
#   Forme reprise de scaleway/dagster-scaleway et Qovery/engine (acl_rules ip=0.0.0.0/0).
#   instance_id littéral : aucune instance (ni mot de passe) provisionnée pour le plan.
resource "scaleway_rdb_acl" "public_db" {
  instance_id = "fr-par/11111111-1111-1111-1111-111111111111"

  acl_rules {
    ip          = "0.0.0.0/0"
    description = "ouvert à Internet"
  }
}

# ✗ Bucket exposé publiquement (ACL canned public-read)
#   → objectstorage_bucket_public_access
resource "scaleway_object_bucket" "backups" {
  name = "backups-prod"
}

resource "scaleway_object_bucket_acl" "backups" {
  bucket = scaleway_object_bucket.backups.id
  acl    = "public-read"
}

# ✗ Politique IAM conférant la gestion de l'IAM (PermissionSet « IAMManager ») :
#   un porteur peut s'octroyer des droits → iam_policy_no_privilege_escalation (CLD-IAM-12)
resource "scaleway_iam_policy" "deployer" {
  name         = "ci-deployer"
  no_principal = true
  rule {
    organization_id      = "11111111-1111-1111-1111-111111111111"
    permission_set_names = ["IAMManager"]
  }
}

# ✗ Serveur avec un secret en clair dans le user-data (cloud-init)
#   → compute_instance_no_secrets_in_user_data (CLD-CMP-2)
resource "scaleway_instance_server" "web" {
  type  = "DEV1-S"
  image = "ubuntu_jammy"
  user_data = {
    "cloud-init" = "#cloud-config\nruncmd:\n  - export DB_PASSWORD=Sup3rSecretP@ss"
  }
}

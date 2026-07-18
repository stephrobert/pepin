# Exemple Terraform Exoscale — code volontairement NON CONFORME, destiné à être
# audité par Pépin (`pepin scan exoscale --terraform`) et à piloter des ressources
# de test jetables (`terraform apply` / `terraform destroy`).
# Ancré sur le schéma réel du provider exoscale/exoscale (v0.69.x) :
#   registry.terraform.io/providers/exoscale/exoscale/latest/docs
# Identifiants : variables d'environnement EXOSCALE_API_KEY / EXOSCALE_API_SECRET.
# Zone ch-gva-2 (Suisse) : hors UE → déclenche aussi governance_resource_region_in_eu (low).

terraform {
  required_providers {
    exoscale = {
      source = "exoscale/exoscale"
    }
  }
}

provider "exoscale" {}

locals {
  zone = "ch-gva-2"
}

data "exoscale_template" "ubuntu" {
  zone = local.zone
  name = "Linux Ubuntu 24.04 LTS 64-bit"
}

# ✗ Security group avec SSH (22) et RDP (3389) ouverts à 0.0.0.0/0
#   → network_securitygroup_allow_ingress_from_internet_to_tcp_port_22 (CLD-NET-1)
resource "exoscale_security_group" "open" {
  name = "pepin-test-open"
}

resource "exoscale_security_group_rule" "ssh" {
  security_group_id = exoscale_security_group.open.id
  type              = "INGRESS"
  protocol          = "TCP"
  start_port        = 22
  end_port          = 22
  cidr              = "0.0.0.0/0"
}

resource "exoscale_security_group_rule" "rdp" {
  security_group_id = exoscale_security_group.open.id
  type              = "INGRESS"
  protocol          = "TCP"
  start_port        = 3389
  end_port          = 3389
  cidr              = "0.0.0.0/0"
}

# ✗ Instance Micro exposée (IP publique par défaut) avec le groupe ouvert, sans
#   étiquettes de gouvernance → compute_instance_public_ip_with_open_securitygroup
#   (CLD-NET-3, en live), governance_resource_required_tags (CLD-GOV-1).
resource "exoscale_compute_instance" "vm" {
  name        = "pepin-test-vm"
  zone        = local.zone
  type        = "standard.micro"
  template_id = data.exoscale_template.ubuntu.id
  disk_size   = 10

  security_group_ids = [exoscale_security_group.open.id]

  # ✗ Secret en clair dans les données utilisateur (cloud-init)
  #   → compute_instance_no_secrets_in_user_data (CLD-CMP-2)
  user_data = <<-EOT
    #cloud-config
    write_files:
      - path: /etc/app/config.env
        content: |
          DB_PASSWORD=Sup3rSecretP@ss
          AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMIabcdefGH1234567890EXAMPLEKEY
  EOT
}

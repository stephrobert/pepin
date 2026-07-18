# Exemple Terraform Outscale — code volontairement NON CONFORME, destiné à être
# audité par Pépin (`pepin scan outscale --terraform`).
# Adapté des exemples officiels du provider :
#   registry.terraform.io/providers/outscale/outscale/latest/docs/resources/security_group_rule
#   registry.terraform.io/providers/outscale/outscale/latest/docs/resources/vm
#   registry.terraform.io/providers/outscale/outscale/latest/docs/resources/policy

terraform {
  required_providers {
    outscale = {
      source = "outscale/outscale"
    }
  }
}

resource "outscale_security_group" "web" {
  description         = "Front web"
  security_group_name = "web-front"
  net_id              = outscale_net.net.net_id
}

# ✗ SSH (22) accepté depuis Internet
#   → network_securitygroup_allow_ingress_from_internet_to_tcp_port_22
resource "outscale_security_group_rule" "ssh_world" {
  flow              = "Inbound"
  security_group_id = outscale_security_group.web.security_group_id
  from_port_range   = "22"
  to_port_range     = "22"
  ip_protocol       = "tcp"
  ip_range          = "0.0.0.0/0"
}

# ✗ VM avec IP publique + SG ouvert sur Internet
#   → compute_instance_public_ip_with_open_securitygroup
resource "outscale_vm" "web" {
  image_id           = "ami-12345678"
  vm_type            = "tinav5.c1r1p2"
  security_group_ids = [outscale_security_group.web.security_group_id]
  subnet_id          = outscale_subnet.subnet.subnet_id
}

resource "outscale_public_ip_link" "web" {
  vm_id     = outscale_vm.web.vm_id
  public_ip = outscale_public_ip.web.public_ip
}

resource "outscale_public_ip" "web" {}

# ✗ Policy EIM avec Action "*" (privilèges administratifs)
#   → iam_policy_no_administrative_privileges
resource "outscale_policy" "ci_admin" {
  policy_name = "ci-admin"
  path        = "/"
  document = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = "*"
      Resource = "*"
    }]
  })
}

# ✗ Policy EIM d'usage qui autorise des actions de gestion d'identité → un porteur
#   peut s'octroyer plus de droits (attacher une policy, créer une clé d'accès)
#   → iam_policy_no_privilege_escalation (CLD-IAM-12)
resource "outscale_policy" "ci_deployer" {
  policy_name = "ci-deployer"
  path        = "/"
  document = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = ["AttachUserPolicy", "CreateAccessKey"]
      Resource = "*"
    }]
  })
}

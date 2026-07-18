# Lab Exoscale — VM jetable pour tester le service de métadonnées (IMDS) en réel.
# Déployable tel quel : fournir sa clé SSH publique, puis `terraform apply`.
#   - SSH entrant ouvert (restreindre allowed_ssh_cidr à votre IP) ;
#   - sortie libre (apt/curl) ;
#   - une fois connecté : tester l'IMDS link-local 169.254.169.254.
# Ancré sur le schéma réel du provider exoscale/exoscale.
# Nettoyage : `terraform destroy`.

terraform {
  required_providers {
    exoscale = { source = "exoscale/exoscale" }
  }
}

provider "exoscale" {}

variable "zone" {
  type        = string
  default     = "ch-gva-2"
  description = "Zone Exoscale."
}

variable "ssh_public_key" {
  type        = string
  description = "Votre clé SSH PUBLIQUE (contenu de ~/.ssh/id_ed25519.pub)."
}

variable "allowed_ssh_cidr" {
  type        = string
  default     = "0.0.0.0/0"
  description = "CIDR autorisé pour le SSH. À RESTREINDRE à votre IP (ex. 203.0.113.4/32)."
}

# Démo CMP-2 : un user-data PIÉGÉ (secret en clair) pour montrer (1) l'exfiltration
# via l'IMDS (curl .../user-data) et (2) la détection par Pépin. Vide par défaut.
variable "demo_leaky_user_data" {
  type        = bool
  default     = false
  description = "Si true, injecte un user-data contenant un faux secret (démo SSRF/CMP-2)."
}

data "exoscale_template" "ubuntu" {
  zone = var.zone
  name = "Linux Ubuntu 24.04 LTS 64-bit"
}

locals {
  # Démo CMP-2 : secrets FACTICES (à ne jamais faire en vrai).
  leaky_user_data = <<-EOT
    #cloud-config
    write_files:
      - path: /etc/app/config.env
        content: |
          DB_PASSWORD=Sup3rSecretP@ss
          API_TOKEN=AKIA0123456789ABCDEF
  EOT
}

resource "exoscale_ssh_key" "lab" {
  name       = "pepin-imds-lab"
  public_key = var.ssh_public_key
}

resource "exoscale_security_group" "lab" {
  name = "pepin-imds-lab"
}

# SSH entrant (pour se connecter et lancer les tests IMDS).
resource "exoscale_security_group_rule" "ssh" {
  security_group_id = exoscale_security_group.lab.id
  type              = "INGRESS"
  protocol          = "TCP"
  start_port        = 22
  end_port          = 22
  cidr              = var.allowed_ssh_cidr
  description       = "SSH (lab IMDS)"
}

# Sortie libre (apt, curl…). L'IMDS est link-local, indépendant de cette règle.
resource "exoscale_security_group_rule" "egress" {
  security_group_id = exoscale_security_group.lab.id
  type              = "EGRESS"
  protocol          = "TCP"
  start_port        = 1
  end_port          = 65535
  cidr              = "0.0.0.0/0"
  description       = "Sortie libre (lab)"
}

resource "exoscale_compute_instance" "lab" {
  zone               = var.zone
  name               = "pepin-imds-lab"
  type               = "standard.micro"
  template_id        = data.exoscale_template.ubuntu.id
  disk_size          = 10
  ssh_key            = exoscale_ssh_key.lab.name
  security_group_ids = [exoscale_security_group.lab.id]

  # Démo uniquement : secret FACTICE en clair, exfiltrable via l'IMDS user-data.
  user_data = var.demo_leaky_user_data ? local.leaky_user_data : null
}

output "instance_ip" {
  value       = exoscale_compute_instance.lab.public_ip_address
  description = "IP publique de la VM (pour scripter)."
}

output "ssh_user" {
  value       = data.exoscale_template.ubuntu.default_user
  description = "Utilisateur SSH (default_user du template)."
}

output "ssh_command" {
  value       = "ssh ${data.exoscale_template.ubuntu.default_user}@${exoscale_compute_instance.lab.public_ip_address}"
  description = "Commande de connexion SSH."
}

output "imds_tests" {
  description = "À lancer DEPUIS la VM pour tester le service de métadonnées."
  value = join("\n", [
    "curl -s http://169.254.169.254/latest/meta-data/",      # liste des catégories
    "curl -s http://169.254.169.254/latest/meta-data/public-ipv4",
    "curl -s http://169.254.169.254/latest/user-data",       # données utilisateur (cloud-init)
  ])
}

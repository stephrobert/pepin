# Remédiation CONFORME : network_flow_matrix_documented (module autonome).
# Exigence : CLD-NET-5 (matrice des flux autorisés, avec justification par règle).
# Fournisseur : exoscale (exoscale_security_group, exoscale_security_group_rule).
# Pourquoi conforme : CHAQUE règle entrante porte une description non vide, qui dit le
#   service et la raison du flux. La règle Pépin lit l'attribut normalisé description
#   des règles entrantes et signale celles qui sont vides ; la matrice des flux se lit
#   alors dans l'infrastructure elle-même, et non dans un tableur à côté.
# Source : references/docs/exoscale/reference-api-schemas-networking.md
#          (description d'une règle de security group) ;
#          schéma exoscale_security_group_rule ; contrat providers/exoscale.yaml.
terraform {
  required_providers {
    exoscale = { source = "exoscale/exoscale" }
  }
}

provider "exoscale" {}

locals {
  # Zone de l'Union européenne (Vienne) : la localisation UE est une exigence
  # transverse (CLD-GVN-3), donc les montages conformes la portent par défaut.
  zone = "at-vie-1"
}

resource "exoscale_security_group" "frontal_documente" {
  name        = "frontal-documente"
  description = "Frontal applicatif ; chaque flux entrant porte sa justification"
}

resource "exoscale_security_group_rule" "https_public" {
  security_group_id = exoscale_security_group.frontal_documente.id
  description       = "HTTPS public : accès des clients au portail"
  type              = "INGRESS"
  protocol          = "TCP"
  start_port        = 443
  end_port          = 443
  cidr              = "0.0.0.0/0"
}

resource "exoscale_security_group_rule" "ssh_administration" {
  security_group_id = exoscale_security_group.frontal_documente.id
  description       = "SSH : exploitation depuis le bastion d'administration"
  type              = "INGRESS"
  protocol          = "TCP"
  start_port        = 22
  end_port          = 22
  cidr              = "198.51.100.0/24"
}

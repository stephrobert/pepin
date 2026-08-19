# Remédiation CONFORME : governance_resource_required_tags (module autonome).
# Exigence : CLD-GVN-1 (inventaire et responsabilité tenus par l'étiquetage).
# Fournisseur : exoscale (exoscale_compute_instance, exoscale_block_storage_volume).
# Pourquoi conforme : chaque ressource facturable porte les QUATRE étiquettes exigées
#   par le référentiel, avec une valeur non vide : CostCenter, Project, Env, Owner
#   (internal/commonrules/rules/lib.rego, required_tags). Chez Exoscale, ces étiquettes
#   sont les labels, une carte clé/valeur portée par la ressource. Miroir NON conforme :
#   examples/exoscale/terraform/main.tf (instance sans aucun label).
# Source : references/docs/exoscale/product-compute-instances-how-to-labels.md ;
#          internal/commonrules/rules/lib.rego (required_tags) ;
#          contrat providers/exoscale.yaml (mapping tags : labels).
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

data "exoscale_template" "ubuntu" {
  zone = local.zone
  name = "Linux Ubuntu 24.04 LTS 64-bit"
}

locals {
  # Une seule définition, appliquée à toutes les ressources facturables : c'est la
  # divergence entre ressources qui fait dériver un inventaire.
  etiquettes = {
    CostCenter = "CC-4711"
    Project    = "socle-applicatif"
    Env        = "production"
    Owner      = "equipe-plateforme"
  }
}

resource "exoscale_security_group" "applicatif" {
  name        = "applicatif-etiquete"
  description = "Charge applicative inventoriée"
}

resource "exoscale_compute_instance" "applicatif" {
  name        = "srv-applicatif"
  zone        = local.zone
  type        = "standard.medium"
  template_id = data.exoscale_template.ubuntu.id
  disk_size   = 50
  labels      = local.etiquettes

  security_group_ids = [exoscale_security_group.applicatif.id]
}

resource "exoscale_block_storage_volume" "donnees" {
  zone   = local.zone
  name   = "donnees-applicatives"
  size   = 100
  labels = local.etiquettes
}

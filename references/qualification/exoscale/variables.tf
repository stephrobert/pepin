# Variables du tenant de qualification Exoscale (plan seul).

variable "zone" {
  description = "Zone UE des ressources : de-fra-1 (Allemagne). Chez Exoscale, la « région » est la zone"
  type        = string
  default     = "de-fra-1"
}

variable "trusted_zone" {
  description = "Zone hors UE mais en espace européen de confiance (Suisse) : ch-gva-2 — l'écart MINEUR de CLD-GVN-3, constructible ici et nulle part ailleurs"
  type        = string
  default     = "ch-gva-2"
}

variable "template_id" {
  description = "Modèle (image) des instances. Un plan ne le valide pas ; un apply réel exigera l'identifiant d'un modèle Exoscale de la zone (data source exoscale_template), donc un compte"
  type        = string
  default     = "00000000-0000-0000-0000-000000000000"
}

variable "instance_type" {
  description = "Gamme des instances (plan seul : aucune facturation)"
  type        = string
  default     = "standard.micro"
}

variable "tenant_tag" {
  description = "Étiquette (label) posée sur toute ressource étiquetable du tenant"
  type        = string
  default     = "pepin-qual"
}

variable "terraform_only_resources" {
  description = "Uniformité du runner : rien n'est « plan seulement » ici, puisque tout l'est"
  type        = bool
  default     = true
}

locals {
  governance = {
    CostCenter = "qualification"
    Project    = "pepin"
    Env        = "qualification"
    Owner      = "pepin-maintainer"
  }
  tagged   = merge({ (var.tenant_tag) = "tenant" }, local.governance)
  untagged = { (var.tenant_tag) = "tenant" }
}

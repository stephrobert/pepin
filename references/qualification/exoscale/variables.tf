# Variables du tenant de qualification Exoscale (plan seul).

variable "organization_id" {
  description = "Organisation Exoscale (GET /organization), confirmée par l'API avant tout apply ; ses huit premiers caractères suffixent les buckets SOS, globaux"
  type        = string
}

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

variable "template_name" {
  description = "Modèle (image) des instances, résolu par une source de données dans chaque zone"
  type        = string
  default     = "Linux Ubuntu 22.04 LTS 64-bit"
}

variable "instance_type" {
  description = "Gamme des instances sans volume attaché"
  type        = string
  default     = "standard.micro"
}

variable "instance_type_with_volume" {
  description = "Gamme des instances qui portent un volume block storage : l'API exige au moins standard.small (provider #375, « Instance size must be at least small »)"
  type        = string
  default     = "standard.small"
}

variable "tenant_tag" {
  description = "Étiquette (label) posée sur toute ressource étiquetable du tenant"
  type        = string
  default     = "pepin-qual"
}

variable "terraform_only_resources" {
  description = "Vrai (défaut) : le plan porte aussi ce que seul le chemin Terraform mesure — l'instance de ch-gva-2, hors de la zone scannée et hors quota (l'organisation a droit à quatre instances). Le runner applique avec false"
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

  bucket_suffix = substr(var.organization_id, 0, 8)
}

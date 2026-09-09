# Variables du tenant de qualification Outscale.
#
# Celles qui identifient le compte ou portent une échéance sont injectées par
# tools/qualification/qualify.py (TF_VAR_*), APRÈS que la porte a vérifié que le
# compte observé par l'API (ReadAccounts) est celui que l'environnement attend.

variable "account_id" {
  description = "Compte Outscale (AccountId) sur lequel le tenant est créé — confirmé par l'API avant tout apply ; sert aussi de suffixe aux noms de buckets, globaux chez OOS"
  type        = string

  validation {
    condition     = can(regex("^[0-9]{12}$", var.account_id))
    error_message = "account_id doit être un identifiant de compte Outscale (12 chiffres)."
  }
}

variable "region" {
  description = "Région du tenant. Un compte Outscale n'existe que dans SA région : aucun contre-exemple hors UE (CLD-GVN-3) n'est constructible avec ce compte"
  type        = string
  default     = "eu-west-2"
}

variable "subregion" {
  description = "Sous-région (AZ) des ressources zonales"
  type        = string
  default     = "eu-west-2a"
}

variable "osc_profile" {
  description = "Profil de ~/.osc/config.json lu par le provider — le même que celui des crochets (OSC_PROFILE) et de `pepin scan --live --profile`"
  type        = string
  default     = "default"
}

variable "tenant_tag" {
  description = "Étiquette posée sur TOUTE ressource étiquetable du tenant : c'est sur elle que la preuve de destruction filtre"
  type        = string
  default     = "pepin-qual"
}

variable "instance_type" {
  description = "Gamme des VM, donnée par le mainteneur pour son compte : tinav6.c2r4p2 (2 cœurs v6 perf 2, 4 Go) — 2 × 0,035 + 4 × 0,005 = 0,09 EUR/h au catalogue (ReadCatalog, 2026-09-09)"
  type        = string
  default     = "tinav6.c2r4p2"
}

variable "image_id" {
  description = "OMI de base des VM — Ubuntu-22.04-2026-01-12 publiée par Outscale en eu-west-2 (ReadImages, 2026-09-09). Une région autre exige son propre identifiant"
  type        = string
  default     = "ami-9afa3d5c"
}

variable "key_expires_at" {
  description = "Échéance RFC 3339 de la clé d'accès CONFORME (contre-exemple de CLD-IAM-2) — calculée par le runner à J+2, jamais écrite en dur"
  type        = string
}

# La protection contre la suppression du contre-exemple `hardened`. Le provider
# refuse de supprimer une VM protégée (issue #88, ouverte) : le runner applique
# `deletion_protection = false` AVANT le destroy (tenant.yaml `pre_destroy_vars`).
variable "deletion_protection" {
  description = "Protection contre la suppression des VM qui la portent (contre-exemple CLD-CMP-10) ; passée à false par le runner juste avant le destroy"
  type        = bool
  default     = true
}

# Uniformité avec les autres tenants : chez Outscale, la collecte live lit TOUT ce que
# la stack crée (le contrat de providers/outscale.yaml couvre les 16 types), donc
# rien n'est « plan seulement » — la variable existe pour que le runner soit
# identique d'un fournisseur à l'autre, et elle n'est lue nulle part.
variable "terraform_only_resources" {
  description = "Inclure les ressources que seul le chemin Terraform sait lire : aucune chez Outscale"
  type        = bool
  default     = true
}

locals {
  # Étiquettes de gouvernance CONFORMES au profil par défaut de Pépin (CostCenter,
  # Project, Env, Owner). Chez Outscale, une étiquette est un bloc `tags { key value }`.
  governance = {
    CostCenter = "qualification"
    Project    = "pepin"
    Env        = "qualification"
    Owner      = "pepin-maintainer"
  }
  production = merge(local.governance, { Env = "prod" })

  # Toute ressource étiquetable porte le tag du tenant : c'est lui que le listing de
  # sortie filtre. `tagged` = gouvernance complète ; `untagged` = le tag du tenant seul.
  tagged            = merge({ (var.tenant_tag) = "tenant" }, local.governance)
  production_tagged = merge({ (var.tenant_tag) = "tenant" }, local.production)
  untagged          = { (var.tenant_tag) = "tenant" }

  bucket_suffix = substr(var.account_id, 0, 8)
}

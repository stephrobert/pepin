# Variables du tenant de qualification Scaleway.
#
# Aucune n'a de valeur par défaut quand elle identifie un compte ou porte un
# secret : elles sont injectées par tools/qualification/qualify.py (TF_VAR_*),
# APRÈS que la porte a vérifié que le compte observé est bien celui épinglé.
# Un `terraform apply` lancé à la main sans ces variables s'arrête sur une
# question, jamais sur un compte inattendu.

variable "project_id" {
  description = "Projet Scaleway dans lequel le tenant est créé — celui qu'expected.yaml épingle, vérifié auprès de l'API avant tout apply"
  type        = string

  validation {
    condition     = can(regex("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$", var.project_id))
    error_message = "project_id doit être un UUID."
  }
}

variable "organization_id" {
  description = "Organisation Scaleway (portée des politiques IAM du tenant), vérifiée auprès de l'API avant tout apply"
  type        = string

  validation {
    condition     = can(regex("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$", var.organization_id))
    error_message = "organization_id doit être un UUID."
  }
}

variable "region" {
  description = "Région du tenant (UE : toutes les régions Scaleway le sont, le contrôle CLD-GVN-3 n'a donc pas de contre-exemple possible ici)"
  type        = string
  default     = "fr-par"
}

variable "zone" {
  description = "Zone des ressources zonales (serveurs, groupes de sécurité, IP)"
  type        = string
  default     = "fr-par-1"
}

variable "tenant_tag" {
  description = "Étiquette posée sur TOUTE ressource étiquetable du tenant : c'est sur elle que la preuve de destruction filtre"
  type        = string
  default     = "pepin-qual"
}

variable "instance_type" {
  description = "Gamme des serveurs — la moins chère qui soit disponible sans rupture de stock (DEV1-S : 0,009 €/h, stockage local, `scw instance server-type list`)"
  type        = string
  default     = "DEV1-S"
}

variable "rdb_node_type" {
  description = "Gamme des bases managées — la plus petite disponible (`scw rdb node-type list`)"
  type        = string
  default     = "db-dev-s"
}

variable "key_expires_at" {
  description = "Échéance RFC 3339 de la clé d'API CONFORME (contre-exemple de CLD-IAM-2) — calculée par le runner à J+2, jamais écrite en dur pour ne pas périmer"
  type        = string
}

variable "rdb_password" {
  description = "Mot de passe de l'utilisateur des bases managées — engendré par le runner à chaque exécution, jamais écrit ni affiché ; il ne sert qu'à satisfaire l'API"
  type        = string
  sensitive   = true
}

# Étiquettes de gouvernance CONFORMES au profil par défaut de Pépin
# (internal/policy : CostCenter, Project, Env, Owner). Sur les ressources dont les
# étiquettes sont une liste de chaînes (Instance, RDB, VPC), la forme est
# « clé=valeur », que le collecteur projette en {key, value} (transform kv).
locals {
  governance_tags = [
    "CostCenter=qualification",
    "Project=pepin",
    "Env=qualification",
    "Owner=pepin-maintainer",
  ]
  # Toute ressource étiquetable porte le tag du tenant EN PREMIER, et c'est sur lui
  # que le listing de sortie filtre. Une ressource sans lui échapperait à la preuve.
  tagged   = concat([var.tenant_tag], local.governance_tags)
  untagged = [var.tenant_tag]

  # Même profil, en carte, pour les buckets (tags S3 = TagSet clé/valeur).
  bucket_governance_tags = {
    CostCenter = "qualification"
    Project    = "pepin"
    Env        = "qualification"
    Owner      = "pepin-maintainer"
  }

  # Les noms de buckets sont GLOBAUX chez Scaleway : le suffixe dérivé du projet
  # évite la collision avec un autre compte qui rejouerait ce tenant.
  bucket_suffix = substr(var.project_id, 0, 8)
}

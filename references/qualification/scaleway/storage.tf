# Stockage objet — sept buckets VIDES dans fr-par (aucun objet : rien à facturer,
# rien qui bloque la suppression). Chaque bucket ne porte qu'UNE faute, sauf
# `unversioned` : sans versioning, l'Object Lock est impossible, donc ce bucket
# porte structurellement deux écarts (versioning + object lock), et expected.yaml
# attend les deux.
#
# Destruction : `force_destroy = true` partout, par ceinture et bretelles — les
# buckets restent vides, mais un bucket versionné ou verrouillé qui aurait reçu un
# objet ne se détruirait pas sans (issue #2869 du provider, corrigée en 2.49.0 ;
# `force_destroy` supprime aussi les objets sous rétention).
#
# Les étiquettes de gouvernance sont posées sur TOUS les buckets : object_storage_bucket
# est un type étiquetable du profil par défaut, et un bucket sans elles porterait
# l'écart governance_resource_required_tags en plus du sien. Le sujet désigné de
# cet écart est la VM `untagged` (compute.tf), pas un bucket.

# ÉCART objectstorage_bucket_public_access (CLD-STO-1, critical) : ACL public-read.
# Versionné et verrouillé pour ne porter QUE l'exposition.
resource "scaleway_object_bucket" "public" {
  name                = "pepin-qual-${local.bucket_suffix}-public"
  object_lock_enabled = true
  force_destroy       = true
  tags                = merge(local.bucket_governance_tags, { pepin-qual = "tenant" })

  versioning {
    enabled = true
  }
}

resource "scaleway_object_bucket_acl" "public" {
  bucket = scaleway_object_bucket.public.name
  acl    = "public-read"
}

# ÉCART objectstorage_bucket_public_access par POLITIQUE : Principal « * » en
# lecture. Observable en live seulement (GetBucketPolicy) ; le plan ne mappe pas
# la ressource de politique, donc ce bucket y est silencieux.
#
# La première déclaration garde à l'IDENTITÉ QUI QUALIFIE ses droits complets :
# en version 2023-04-17, « only actions explicitly allowed by the bucket policy
# are permitted » et « you will lose access to your bucket if you are not the
# owner of the Organization, and if you are not explicitly allowed » (doc Scaleway
# « Bucket policies overview »). Le principal `project_id:` de l'ancienne version
# est REFUSÉ par l'API en 2023-04-17 (mesuré le 2026-09-09 : « project_id Principal
# is deprecated ») ; le runner fournit donc `user_id:…` ou `application_id:…`,
# celui de la clé qui exécute, résolu auprès de l'API et jamais committé.
resource "scaleway_object_bucket" "policy" {
  name                = "pepin-qual-${local.bucket_suffix}-policy"
  object_lock_enabled = true
  force_destroy       = true
  tags                = merge(local.bucket_governance_tags, { pepin-qual = "tenant" })

  versioning {
    enabled = true
  }
}

resource "scaleway_object_bucket_policy" "public" {
  bucket = scaleway_object_bucket.policy.name
  policy = jsonencode({
    Version = "2023-04-17"
    Id      = "pepin-qual-public-read"
    Statement = [
      {
        Sid       = "QualifierKeepsFullAccess"
        Effect    = "Allow"
        Principal = { SCW = var.bucket_policy_principal }
        Action    = ["s3:*"]
        Resource = [
          scaleway_object_bucket.policy.name,
          "${scaleway_object_bucket.policy.name}/*",
        ]
      },
      {
        Sid       = "AnyoneReads"
        Effect    = "Allow"
        Principal = "*"
        Action    = ["s3:GetObject"]
        Resource  = ["${scaleway_object_bucket.policy.name}/*"]
      },
    ]
  })
}

# ÉCARTS objectstorage_bucket_versioning_enabled (medium) ET, par construction,
# objectstorage_bucket_object_lock_enabled (low) : ni versioning ni verrou.
resource "scaleway_object_bucket" "unversioned" {
  name          = "pepin-qual-${local.bucket_suffix}-unversioned"
  force_destroy = true
  tags          = merge(local.bucket_governance_tags, { pepin-qual = "tenant" })
}

# ÉCART objectstorage_bucket_object_lock_enabled (CLD-STO-8, low) seul : versionné
# mais sans Object Lock.
resource "scaleway_object_bucket" "unlocked" {
  name          = "pepin-qual-${local.bucket_suffix}-unlocked"
  force_destroy = true
  tags          = merge(local.bucket_governance_tags, { pepin-qual = "tenant" })

  versioning {
    enabled = true
  }
}

# ÉCART objectstorage_bucket_kms_encryption (CLD-CHF-4, medium) : classé sensible
# (étiquette classification=confidential) sans clé gérée par le client (SSE-KMS).
# Observable en live seulement (GetBucketEncryption).
resource "scaleway_object_bucket" "sensitive" {
  name                = "pepin-qual-${local.bucket_suffix}-sensitive"
  object_lock_enabled = true
  force_destroy       = true
  tags = merge(local.bucket_governance_tags, {
    pepin-qual     = "tenant"
    classification = "confidential"
  })

  versioning {
    enabled = true
  }
}

# CONTRE-EXEMPLE de TOUS les contrôles de stockage objet : privé, versionné,
# verrouillé, étiqueté, et classé PUBLIC — donc pas d'exigence SSE-KMS. Le
# contre-exemple du contrôle KMS est la classification, pas la clé : c'est elle
# qui discrimine.
resource "scaleway_object_bucket" "hardened" {
  name                = "pepin-qual-${local.bucket_suffix}-hardened"
  object_lock_enabled = true
  force_destroy       = true
  tags = merge(local.bucket_governance_tags, {
    pepin-qual     = "tenant"
    classification = "public"
  })

  versioning {
    enabled = true
  }
}

# CONTRE-EXEMPLE de objectstorage_bucket_default_encryption : chiffrement au repos
# ACTIVÉ (SSE-ONE, clé gérée par Scaleway). Chez Scaleway, le chiffrement au repos
# d'un bucket est opt-in — un bucket sans configuration SSE écrit en clair — et ce
# bucket est le seul du tenant qui le configure. Classé public : pas d'exigence de
# clé client (SSE-KMS), donc muet sur CLD-CHF-4 aussi.
resource "scaleway_object_bucket" "encrypted" {
  name                = "pepin-qual-${local.bucket_suffix}-encrypted"
  object_lock_enabled = true
  force_destroy       = true
  tags = merge(local.bucket_governance_tags, {
    pepin-qual     = "tenant"
    classification = "public"
  })

  versioning {
    enabled = true
  }
}

resource "scaleway_object_bucket_server_side_encryption_configuration" "encrypted" {
  bucket = scaleway_object_bucket.encrypted.name

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

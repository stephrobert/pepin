# Remédiation CONFORME : objectstorage_bucket_object_lock_enabled (module autonome).
# Exigence : CLD-STO-8 (immutabilité WORM des objets critiques).
# Fournisseur : exoscale (SOS, piloté par le provider Terraform AWS, comme Exoscale
#   le documente).
# Pourquoi conforme : le bucket est créé AVEC le verrouillage d'objet
#   (object_lock_enabled = true), ce qui implique le versioning, puis une rétention
#   par défaut est posée. Point non négociable, documenté par Exoscale : le
#   verrouillage ne s'active qu'À LA CRÉATION du bucket, jamais après ; remédier un
#   bucket existant impose donc d'en créer un nouveau et d'y recopier les objets.
#   La règle Pépin lit GetObjectLockConfiguration et ne se déclenche que si la
#   capacité est exposée et désactivée.
# Source : references/docs/exoscale/product-storage-object-storage-how-to-versioning.md
#          (verrouillage à la création, rétention GOVERNANCE) ;
#          references/docs/exoscale/product-storage-object-storage-how-to-terraform.md.
terraform {
  required_providers {
    aws = { source = "hashicorp/aws" }
  }
}

locals {
  # Zone de l'Union européenne (Vienne). L'endpoint SOS suit la zone.
  zone = "at-vie-1"
}

# Configuration documentée par Exoscale : SOS est compatible S3, les validations
# propres à AWS sont désactivées. Identifiants par AWS_ACCESS_KEY_ID /
# AWS_SECRET_ACCESS_KEY (clé IAM Exoscale).
provider "aws" {
  region = local.zone

  endpoints {
    s3 = "https://sos-${local.zone}.exo.io"
  }

  skip_credentials_validation = true
  skip_region_validation      = true
  skip_requesting_account_id  = true
}

resource "aws_s3_bucket" "sauvegardes" {
  bucket = "socle-applicatif-sauvegardes"

  # Non modifiable après création (comportement S3, rappelé par la doc Exoscale).
  object_lock_enabled = true
}

resource "aws_s3_bucket_acl" "sauvegardes" {
  bucket = aws_s3_bucket.sauvegardes.id
  acl    = "private"
}

# Le verrouillage d'objet exige le versioning ; on le déclare explicitement plutôt
# que de dépendre de son activation implicite.
resource "aws_s3_bucket_versioning" "sauvegardes" {
  bucket = aws_s3_bucket.sauvegardes.id

  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_object_lock_configuration" "sauvegardes" {
  bucket = aws_s3_bucket.sauvegardes.id

  rule {
    default_retention {
      mode = "GOVERNANCE"
      days = 30
    }
  }

  depends_on = [aws_s3_bucket_versioning.sauvegardes]
}

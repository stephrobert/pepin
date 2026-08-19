# Remédiation CONFORME : objectstorage_bucket_versioning_enabled (module autonome).
# Exigence : CLD-STO-4 (versioning du stockage objet).
# Fournisseur : exoscale (SOS, piloté par le provider Terraform AWS, comme Exoscale
#   le documente).
# Pourquoi conforme : le versioning du bucket est explicitement « Enabled ». Chez
#   Exoscale, les buckets sont NON VERSIONNÉS par défaut : sans cette ressource, un
#   objet écrasé ou supprimé l'est définitivement. Une fois le versioning actif, une
#   suppression pose un marqueur et l'objet reste restaurable. La règle Pépin lit le
#   statut rendu par GetBucketVersioning et n'accepte que « Enabled ».
# Source : references/docs/exoscale/product-storage-object-storage-how-to-versioning.md ;
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

resource "aws_s3_bucket" "versionne" {
  bucket = "socle-applicatif-versionne"
}

resource "aws_s3_bucket_acl" "versionne" {
  bucket = aws_s3_bucket.versionne.id
  acl    = "private"
}

resource "aws_s3_bucket_versioning" "versionne" {
  bucket = aws_s3_bucket.versionne.id

  versioning_configuration {
    status = "Enabled"
  }
}

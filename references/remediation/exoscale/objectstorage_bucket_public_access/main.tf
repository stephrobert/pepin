# Remédiation CONFORME : objectstorage_bucket_public_access (module autonome).
# Exigence : CLD-STO-1 (stockage objet jamais exposé publiquement).
# Fournisseur : exoscale (SOS, piloté par le provider Terraform AWS, comme Exoscale
#   le documente : SOS est compatible S3).
# Pourquoi conforme : l'ACL du bucket est « private » et AUCUNE politique de bucket
#   n'accorde de droit à un principal anonyme. La règle Pépin signale une ACL publique
#   (public-read, public-read-write, authenticated-read), un grant au groupe AllUsers
#   ou AuthenticatedUsers, ou une politique dont le Principal vaut « * ». À noter :
#   authenticated-read ouvre à tout compte de la plateforme, donc hors du tenant ; ce
#   n'est pas une nuance de configuration, c'est une exposition inter-tenant.
# Source : references/docs/exoscale/product-storage-object-storage-how-to-terraform.md ;
#          references/docs/exoscale/product-storage-object-storage-how-to-acl.md ;
#          references/docs/exoscale/product-storage-object-storage-how-to-bucketpolicy.md.
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

resource "aws_s3_bucket" "prive" {
  bucket = "socle-applicatif-prive"
}

resource "aws_s3_bucket_acl" "prive" {
  bucket = aws_s3_bucket.prive.id
  acl    = "private"
}

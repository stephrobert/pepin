# Remédiation CONFORME — iam_role_key_lifetime_bounded  (module autonome).
# Exigence : CLD-IAM-2 — accès à durée de vie limitée (pas de secret permanent).
# Provider : exoscale (schéma exoscale_iam_role ; policy CEL).
# Pourquoi conforme : la durée de vie est bornée. Deux montages valides (la règle
#   accepte l'un OU l'autre) :
#     (a) expiration dans la policy — refuse l'usage au-delà d'une durée depuis la
#         création de la clé (identity.created + extension duration()) ; montré ici ;
#     (b) max-session-ttl du rôle — non exposé par le schéma Terraform (à poser via
#         API/CLI), donc en Terraform on s'appuie sur (a).
#   Miroir NON conforme : examples/exoscale/terraform/iam.tf (ni l'un ni l'autre).
# Source : references/docs/exoscale/product-iam-how-to-policy-guide.md
#          (exemple officiel « time-expiring IAM Key ») ; contrat providers/exoscale.yaml.

terraform {
  required_providers {
    exoscale = { source = "exoscale/exoscale" }
  }
}

provider "exoscale" {}

resource "exoscale_iam_role" "short_lived" {
  name        = "short-lived-ci"
  description = "Accès CI à durée de vie bornée (expiration dans la policy)"

  policy = {
    default_service_strategy = "deny"
    services = {
      compute = {
        type = "rules"
        rules = [
          # (a) au-delà de 1h après création de la clé, on refuse tout usage.
          {
            action     = "deny"
            expression = "timestamp(identity.created) < timestamp(now) - duration('1h')"
          },
          {
            action     = "allow"
            expression = "operation.startsWith('list-')"
          },
        ]
      }
    }
  }
}

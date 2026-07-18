# Remédiation CONFORME — iam_role_source_ip_restricted  (module autonome).
# Exigence : CLD-IAM-4 — restreindre l'accès API par plage d'IP source.
# Provider : exoscale (schéma exoscale_iam_role ; policy CEL, binding source_ip).
# Pourquoi conforme : chaque règle d'autorisation est gardée par source_ip.inIpRange ;
#   hors de ces plages, la stratégie « deny » par défaut s'applique. Miroir NON
#   conforme : examples/exoscale/terraform/iam.tf (aucune condition source_ip).
# Source : references/docs/exoscale/product-iam-how-to-policy-guide.md
#          (binding source_ip ; extension inIpRange) ; contrat providers/exoscale.yaml.

terraform {
  required_providers {
    exoscale = { source = "exoscale/exoscale" }
  }
}

provider "exoscale" {}

resource "exoscale_iam_role" "ops_from_office" {
  name        = "ops-from-office"
  description = "Opérations compute limitées aux IP d'administration"

  policy = {
    default_service_strategy = "deny"
    services = {
      compute = {
        type = "rules"
        rules = [
          {
            action     = "allow"
            expression = "source_ip.inIpRange('203.0.113.0/24') && operation.startsWith('list-')"
          },
        ]
      }
    }
  }
}

# Remédiation CONFORME — iam_role_no_admin_privileges  (module autonome).
# Exigence : CLD-IAM-1 — interdire les rôles/clés à privilèges excessifs.
# Provider : exoscale (schéma exoscale_iam_role ; policy CEL).
# Pourquoi conforme : stratégie de service « deny » par défaut + autorisations
#   explicites minimales. Miroir NON conforme : examples/exoscale/terraform/iam.tf
#   (default_service_strategy = "allow", services = {}).
# Source : references/docs/exoscale/product-iam-how-to-policy-guide.md ;
#          contrat providers/exoscale.yaml (type iam_role, admin_privileges).

terraform {
  required_providers {
    exoscale = { source = "exoscale/exoscale" }
  }
}

provider "exoscale" {}

resource "exoscale_iam_role" "ci_readonly" {
  name        = "ci-readonly"
  description = "Lecture seule compute pour la CI — moindre privilège"

  policy = {
    default_service_strategy = "deny" # ✔ rien n'est autorisé par défaut
    services = {
      compute = {
        type = "rules"
        rules = [
          {
            action     = "allow"
            expression = "operation in ['list-instances', 'get-instance']"
          },
        ]
      }
    }
  }
}

# Tenant de qualification Exoscale — épinglage du provider.
#
# PLAN SEUL, pour l'instant : aucun compte Exoscale n'est disponible (tenant.yaml
# `live: unavailable`). Ce tenant épingle donc la SOURCE TERRAFORM — ce que
# `pepin scan exoscale --terraform` rend sur ce plan — et rien d'autre. La moitié
# live (apply, scan --live, bundle, destroy, preuve de destruction) attend un
# compte, et le README le dit plutôt que de laisser croire à une couverture live.
#
# Version ÉPINGLÉE À L'EXACT, lockfile versionné : même doctrine que les deux autres
# tenants. Les issues de destruction du provider exoscale/exoscale sont à lire AVANT
# le premier apply, pas avant le premier plan : cette lecture reste due.
terraform {
  required_version = ">= 1.11"

  backend "local" {}

  required_providers {
    exoscale = {
      source  = "exoscale/exoscale"
      version = "0.71.0"
    }
  }
}

# Aucun identifiant ici (ADR-0012) : le provider lit EXOSCALE_API_KEY et
# EXOSCALE_API_SECRET. En plan seul, le crochet `variables` fournit des valeurs
# FACTICES de la bonne forme (celles que scripts/tenant-plan.py emploie déjà) : un
# plan sans source de données n'appelle pas l'API.
provider "exoscale" {}

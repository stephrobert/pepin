# Tenant de qualification Outscale — épinglage du provider.
#
# Version ÉPINGLÉE À L'EXACT, jamais en `~>` : une contrainte flottante laisse une
# publication amont changer ce que ce tenant mesure (c'est ainsi que la régression
# de destruction du provider Scaleway 2.81.0 est arrivée). Le lockfile à côté est
# versionné : il garantit que la qualification installe le binaire validé.
#
# Ce que ce tenant ÉVITE, parce que les issues du provider disent que ça ne se
# détruit pas (détail et liens dans README.md) :
#   - `outscale_nic_private_ip` (#28, #137 : 400/409 au destroy, ouvertes) ;
#   - les blocs `nics` / `primary_nic` mêlés dans `outscale_vm` (#778, #424, #50,
#     #448 : remplacement de la VM, NIC « oubliée ») → `outscale_nic` + `outscale_nic_link` ;
#   - une VM protégée contre la suppression au moment du destroy (#88 : « Error
#     deleting the VM ») → la protection est LEVÉE par un apply avant le destroy
#     (`pre_destroy_vars` de tenant.yaml), et le nettoyage de secours fait UpdateVm ;
#   - les politiques de stickiness sur le LBU (#663).
terraform {
  required_version = ">= 1.11"

  backend "local" {}

  required_providers {
    outscale = {
      source  = "outscale/outscale"
      version = "1.8.0"
    }
  }
}

# Aucun identifiant ici (ADR-0012) : le provider lit OSC_ACCESS_KEY/OSC_SECRET_KEY,
# ou le profil de ~/.osc/config.json, exactement comme `pepin scan --live`, les
# crochets et `octl`. Le compte est confirmé par l'API avant tout apply (runner).
provider "outscale" {
  profile = var.osc_profile
  region  = var.region
}

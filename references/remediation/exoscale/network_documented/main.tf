# Remédiation CONFORME : network_documented (module autonome).
# Exigence : CLD-NET-5 (cartographie réseau tenue à jour).
# Fournisseur : exoscale (exoscale_private_network).
# Pourquoi conforme : le réseau privé porte un nom, une description et des labels de
#   gouvernance (propriétaire, projet, environnement). La règle Pépin se déclenche sur
#   un réseau SANS aucune étiquette : la cartographie est alors portée par le réseau
#   lui-même, et non par un document qui dérive.
# Source : references/docs/exoscale/reference-terraform-provider-resources-private-network.md ;
#          references/docs/exoscale/product-networking-private-network-how-to-managed-private-network.md ;
#          contrat providers/exoscale.yaml (mapping network : name, description, labels).
terraform {
  required_providers {
    exoscale = { source = "exoscale/exoscale" }
  }
}

provider "exoscale" {}

locals {
  # Zone de l'Union européenne (Vienne) : la localisation UE est une exigence
  # transverse (CLD-GVN-3), donc les montages conformes la portent par défaut.
  zone = "at-vie-1"
}

resource "exoscale_private_network" "applicatif" {
  zone        = local.zone
  name        = "reseau-applicatif"
  description = "Réseau privé du socle applicatif, palier de production"

  # Plage gérée : le service distribue les baux DHCP dans cet intervalle.
  netmask  = "255.255.255.0"
  start_ip = "10.0.0.20"
  end_ip   = "10.0.0.200"

  labels = {
    Owner   = "equipe-plateforme"
    Project = "socle-applicatif"
    Env     = "production"
  }
}

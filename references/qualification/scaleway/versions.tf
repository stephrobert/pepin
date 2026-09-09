# Tenant de qualification Scaleway — épinglage du provider.
#
# La version est ÉPINGLÉE À L'EXACT, jamais en `~>` : une contrainte flottante
# laisse une publication amont changer ce que ce tenant mesure, sans commit de
# notre côté. C'est exactement la mécanique qui a introduit la régression de
# destruction du provider 2.81.0 (issue scaleway/terraform-provider-scaleway#4338 :
# une `scaleway_instance_private_nic` attachée ne se détruit plus, mesuré par le
# mainteneur le 2026-09-09 sur 2.81.0 ET 2.82.0, 4 ressources laissées sur 4).
#
# Ce tenant reste en 2.82.0 parce qu'il ne porte AUCUNE interface privée : c'est le
# choix de conception qui rend la régression inopérante ici. Si une interface
# privée entrait un jour dans cette stack, il faudrait soit redescendre en 2.80.0,
# soit automatiser `scw instance private-nic delete` avant le destroy — et le
# README de ce dossier le dit avant qu'on ne l'apprenne sur la facture.
#
# Le fichier .terraform.lock.hcl à côté est VERSIONNÉ (exception explicite dans
# .gitignore) : c'est lui qui garantit que la qualification installe le binaire
# de provider qui a été validé, et non celui publié depuis.
terraform {
  required_version = ">= 1.11"

  # Backend local DÉCLARÉ : sans ce bloc, le `-backend-config=path=…` que le runner
  # passe à `terraform init` est ignoré, et l'état — qui porte les clés secrètes des
  # clés d'API du tenant — s'écrirait ici, dans le dépôt. Le runner le place dans le
  # dossier du run, et le purge après un destroy dont l'état est vide.
  backend "local" {}

  required_providers {
    scaleway = {
      source  = "scaleway/scaleway"
      version = "2.82.0"
    }
  }
}

# Aucun identifiant ici (ADR-0012) : le provider lit les variables natives
# SCW_ACCESS_KEY / SCW_SECRET_KEY, puis ~/.config/scw/config.yaml, exactement
# comme `pepin scan --live` et comme la CLI scw. Le projet est passé EXPLICITEMENT
# par variable : la porte refuse de démarrer si le compte observé par l'API n'est
# pas celui qu'`expected.yaml` épingle, et Terraform reçoit ce même identifiant —
# aucun des deux ne dépend du profil scw actif au moment du lancement.
provider "scaleway" {
  region     = var.region
  zone       = var.zone
  project_id = var.project_id
}

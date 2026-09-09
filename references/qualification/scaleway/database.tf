# Bases managées — deux instances db-dev-s, PLAN SEULEMENT.
#
# Les trois contrôles de base de données ne sont observables que sur le plan
# (`scaleway_rdb_instance`, `scaleway_rdb_acl` ; la collecte live RDB n'existe
# pas, contrat de providers/scaleway.yaml). Provisionner une base sur le compte
# réel coûterait de l'argent (0,035 €/h pièce) et deux à cinq minutes de création
# puis de destruction — sans qu'aucun scan live ne la voie. Elles n'entrent donc
# que dans le plan complet (`terraform_only_resources = true`), que le runner
# scanne en `--terraform` ; la stack appliquée ne les contient pas.
#
# `exposed` porte les trois écarts sur une seule instance — c'est l'exception
# assumée à « une faute par ressource » : trois sujets distincts d'un même type
# n'apprendraient rien de plus. `hardened` est le contre-exemple des trois.
#
# Le mot de passe vient du runner (variable sensible, engendrée à chaque run).

# ÉCARTS database_service_not_open_to_internet (ACL 0.0.0.0/0),
# database_encryption_at_rest_enabled (chiffrement absent),
# database_backup_enabled (sauvegardes désactivées).
resource "scaleway_rdb_instance" "exposed" {
  count              = local.tf_only
  name               = "pepin-qual-rdb-exposed"
  node_type          = var.rdb_node_type
  engine             = "PostgreSQL-16"
  is_ha_cluster      = false
  disable_backup     = true
  encryption_at_rest = false
  user_name          = "pepin"
  password           = var.rdb_password
  tags               = local.tagged
}

resource "scaleway_rdb_acl" "exposed" {
  count       = local.tf_only
  instance_id = scaleway_rdb_instance.exposed[0].id

  acl_rules {
    ip          = "0.0.0.0/0"
    description = "ouvert a Internet"
  }
}

# CONTRE-EXEMPLE : chiffrée au repos, sauvegardée, ACL restreinte à un /8 privé.
#
# Le chiffrement au repos (LUKS) porte sur un volume BLOCK : la documentation
# « Setting up encryption at rest » de Scaleway l'illustre avec `volume_type: sbs_5k`.
# Le volume est donc déclaré Block, à la taille minimale de la gamme (5 Go).
resource "scaleway_rdb_instance" "hardened" {
  count                     = local.tf_only
  name                      = "pepin-qual-rdb-hardened"
  node_type                 = var.rdb_node_type
  engine                    = "PostgreSQL-16"
  is_ha_cluster             = false
  disable_backup            = false
  backup_schedule_frequency = 24
  backup_schedule_retention = 7
  encryption_at_rest        = true
  volume_type               = "sbs_5k"
  volume_size_in_gb         = 5
  user_name                 = "pepin"
  password                  = var.rdb_password
  tags                      = local.tagged
}

resource "scaleway_rdb_acl" "hardened" {
  count       = local.tf_only
  instance_id = scaleway_rdb_instance.hardened[0].id

  acl_rules {
    ip          = "10.0.0.0/8"
    description = "reseau applicatif prive"
  }
}

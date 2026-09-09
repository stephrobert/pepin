# Bases managées — deux instances db-dev-s (la plus petite gamme disponible).
#
# Les trois contrôles de base de données ne sont observables que sur le PLAN
# (`scaleway_rdb_instance`, `scaleway_rdb_acl` ; la collecte live RDB est à
# câbler). Les instances sont tout de même APPLIQUÉES : le plan scanné doit être
# celui qui a été appliqué, sans quoi la qualification comparerait deux tenants.
#
# `exposed` porte les trois écarts sur une seule instance — c'est l'exception
# assumée à « une faute par ressource » : trois bases coûteraient trois fois le
# temps de création/destruction (2 à 5 minutes chacune) pour trois sujets
# distincts d'un même type. `hardened` est le contre-exemple des trois.
#
# Le mot de passe vient du runner (variable sensible, engendrée à chaque run).

# ÉCARTS database_service_not_open_to_internet (ACL 0.0.0.0/0),
# database_encryption_at_rest_enabled (chiffrement absent),
# database_backup_enabled (sauvegardes désactivées).
resource "scaleway_rdb_instance" "exposed" {
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
  instance_id = scaleway_rdb_instance.exposed.id

  acl_rules {
    ip          = "0.0.0.0/0"
    description = "ouvert a Internet"
  }
}

# CONTRE-EXEMPLE : chiffrée au repos, sauvegardée, ACL restreinte à un /8 privé.
resource "scaleway_rdb_instance" "hardened" {
  name                      = "pepin-qual-rdb-hardened"
  node_type                 = var.rdb_node_type
  engine                    = "PostgreSQL-16"
  is_ha_cluster             = false
  disable_backup            = false
  backup_schedule_frequency = 24
  backup_schedule_retention = 7
  encryption_at_rest        = true
  user_name                 = "pepin"
  password                  = var.rdb_password
  tags                      = local.tagged
}

resource "scaleway_rdb_acl" "hardened" {
  instance_id = scaleway_rdb_instance.hardened.id

  acl_rules {
    ip          = "10.0.0.0/8"
    description = "reseau applicatif prive"
  }
}

# IAM — clés d'API et politiques (portée : ORGANISATION, ressources gratuites).
#
# Deux applications, et la séparation est délibérée : l'application qui PORTE
# les clés n'a aucune politique (des clés sans droit), et l'application qui porte
# la politique fautive n'a aucune clé. Le tenant reproduit ainsi l'écart CLD-IAM-12
# sans jamais mettre en circulation, même trente minutes, un secret capable de
# gérer l'IAM de l'organisation.
#
# Les utilisateurs IAM ne sont PAS créés : `scaleway_iam_user` invite une personne
# réelle dans l'organisation, ce qui sort du périmètre d'un tenant jetable. Le
# contrôle CLD-IAM-3 (MFA) s'observe donc sur les utilisateurs existants de
# l'organisation, et expected.yaml le dit (statut « evaluated », hors périmètre).

resource "scaleway_iam_application" "keys" {
  name        = "pepin-qual-app-keys"
  description = "Tenant de qualification Pépin : porte les clés d'API, aucune politique"
  tags        = local.untagged
}

resource "scaleway_iam_application" "admin" {
  name        = "pepin-qual-app-admin"
  description = "Tenant de qualification Pépin : porte les politiques, aucune clé"
  tags        = local.untagged
}

# ÉCART iam_accesskey_expiration_set (CLD-IAM-2, critical) : clé sans échéance.
# Sujet du finding = l'access key, connu seulement après apply → output.
resource "scaleway_iam_api_key" "no_expiry" {
  application_id = scaleway_iam_application.keys.id
  description    = "pepin-qual-key-no-expiry"
}

# CONTRE-EXEMPLE : la même clé, avec une échéance à venir (J+2, posée par le runner).
resource "scaleway_iam_api_key" "expiring" {
  application_id = scaleway_iam_application.keys.id
  description    = "pepin-qual-key-expiring"
  expires_at     = var.key_expires_at
}

# ÉCART iam_policy_no_privilege_escalation (CLD-IAM-12, high) : le PermissionSet
# IAMManager confère la gestion de l'IAM (ancré : mapping_terraform de
# providers/scaleway.yaml, `manages_iam: contains:IAMManager`).
resource "scaleway_iam_policy" "iam_manager" {
  name           = "pepin-qual-policy-iam-manager"
  description    = "Tenant de qualification Pépin : gestion IAM = chemin d'élévation"
  application_id = scaleway_iam_application.admin.id
  tags           = local.untagged

  rule {
    organization_id      = var.organization_id
    permission_set_names = ["IAMManager"]
  }
}

# CONTRE-EXEMPLE : lecture seule sur les instances, portée projet — aucune gestion IAM.
resource "scaleway_iam_policy" "read_only" {
  name           = "pepin-qual-policy-read-only"
  description    = "Tenant de qualification Pépin : lecture seule, portée projet"
  application_id = scaleway_iam_application.admin.id
  tags           = local.untagged

  rule {
    project_ids          = [var.project_id]
    permission_set_names = ["InstancesReadOnly"]
  }
}

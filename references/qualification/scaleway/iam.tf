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

# ÉCART iam_accesskey_expiration_set (CLD-IAM-2, critical) : clé SANS échéance.
#
# PLAN SEULEMENT. Mesuré le 2026-09-09 : l'organisation de qualification refuse
# de créer une telle clé — « organization security settings require an expiration
# date for API keys ». C'est un réglage préventif de l'organisation, et il fait
# exactement ce que le contrôle demande ; la faute n'est donc pas constructible en
# live sur ce compte, et elle reste exercée sur le plan.
resource "scaleway_iam_api_key" "no_expiry" {
  count          = local.tf_only
  application_id = scaleway_iam_application.keys.id
  description    = "pepin-qual-key-no-expiry"
}

# ÉCART iam_accesskey_expiration_set (CLD-IAM-2, high) : clé dont l'échéance est
# DÉPASSÉE et qui reste listée — la branche « l'échéance ne protège plus rien ».
# Le runner pose une échéance quelques minutes après l'apply, puis ATTEND qu'elle
# soit passée avant de scanner. Mesuré le 2026-09-09 : une clé expirée reste dans
# GET /iam/v1alpha1/api-keys, avec son expires_at dans le passé.
resource "scaleway_iam_api_key" "expired" {
  application_id = scaleway_iam_application.keys.id
  description    = "pepin-qual-key-expired"
  expires_at     = var.key_expired_at
}

# CONTRE-EXEMPLE : la même clé, avec une échéance à venir (J+2, posée par le runner).
resource "scaleway_iam_api_key" "expiring" {
  application_id = scaleway_iam_application.keys.id
  description    = "pepin-qual-key-expiring"
  expires_at     = var.key_expires_at
}

# ── Plan seulement : les politiques IAM ne sont pas collectées en live ──────────

resource "scaleway_iam_application" "admin" {
  count       = local.tf_only
  name        = "pepin-qual-app-admin"
  description = "Tenant de qualification Pépin : porte les politiques, aucune clé"
  tags        = local.untagged
}

# ÉCART iam_policy_no_privilege_escalation (CLD-IAM-12, high) : le PermissionSet
# IAMManager confère la gestion de l'IAM (ancré : mapping_terraform de
# providers/scaleway.yaml, `manages_iam: contains:IAMManager`).
resource "scaleway_iam_policy" "iam_manager" {
  count          = local.tf_only
  name           = "pepin-qual-policy-iam-manager"
  description    = "Tenant de qualification Pépin : gestion IAM = chemin d'élévation"
  application_id = scaleway_iam_application.admin[0].id
  tags           = local.untagged

  rule {
    organization_id      = var.organization_id
    permission_set_names = ["IAMManager"]
  }
}

# CONTRE-EXEMPLE : lecture seule sur les instances, portée projet — aucune gestion IAM.
resource "scaleway_iam_policy" "read_only" {
  count          = local.tf_only
  name           = "pepin-qual-policy-read-only"
  description    = "Tenant de qualification Pépin : lecture seule, portée projet"
  application_id = scaleway_iam_application.admin[0].id
  tags           = local.untagged

  rule {
    project_ids          = [var.project_id]
    permission_set_names = ["InstancesReadOnly"]
  }
}

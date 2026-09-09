# EIM — un utilisateur, trois clés d'accès, cinq politiques managées, deux règles
# d'accès API (portée : COMPTE, ressources gratuites).
#
# Les clés sont créées INACTIVES : la règle CLD-IAM-2 « sans échéance » et la
# dérivation root_owned ne lisent pas l'état, donc la faute se mesure sans mettre
# un secret utilisable en circulation. La clé ROOT (sans `user_name`, c'est celle du
# compte qui exécute) est l'écart CLD-IAM-1 ; elle porte une échéance pour ne porter
# QUE cet écart.
#
# Les politiques managées ne sont attachées à personne : la collecte lit
# ReadPolicies (Scope LOCAL), pas les liaisons, et une politique orpheline n'ouvre
# rien. La politique INLINE `Action: *` — l'incident fondateur de l'ADR-0006 — n'a
# pas de ressource Terraform : le crochet `extra` la pose sur l'utilisateur après
# l'apply (PutUserPolicy) et la retire avant le destroy.

resource "outscale_user" "qual" {
  user_name = "pepin-qual-user"
  path      = "/pepin_qual/"
}

# ÉCART iam_accesskey_expiration_set (CLD-IAM-2, critical) : clé sans échéance.
resource "outscale_access_key" "no_expiry" {
  user_name = outscale_user.qual.user_name
  state     = "INACTIVE"
  tag       = "pepin-qual-key-no-expiry"
}

# CONTRE-EXEMPLE : la même clé, avec une échéance à venir (J+2, posée par le runner).
resource "outscale_access_key" "expiring" {
  user_name       = outscale_user.qual.user_name
  expiration_date = var.key_expires_at
  state           = "INACTIVE"
  tag             = "pepin-qual-key-expiring"
}

# ÉCART iam_no_root_access_key (CLD-IAM-1, high) : clé du compte root (l'appelant).
resource "outscale_access_key" "root" {
  expiration_date = var.key_expires_at
  state           = "INACTIVE"
  tag             = "pepin-qual-key-root"
}

# ÉCART iam_policy_no_administrative_privileges (critical) : tout sur tout.
resource "outscale_policy" "admin" {
  policy_name = "pepin-qual-policy-admin"
  description = "Tenant de qualification Pepin : administration totale"
  path        = "/pepin_qual/"
  document = jsonencode({
    Statement = [{ Effect = "Allow", Action = ["*"], Resource = ["*"] }]
  })
}

# ÉCART iam_policy_no_wildcard_resource (high) : actions bornées, ressource « * ».
resource "outscale_policy" "wildcard" {
  policy_name = "pepin-qual-policy-wildcard"
  description = "Tenant de qualification Pepin : ressource joker"
  path        = "/pepin_qual/"
  document = jsonencode({
    Statement = [{ Effect = "Allow", Action = ["api:ReadVms", "api:ReadVolumes"], Resource = ["*"] }]
  })
}

# ÉCART iam_policy_no_notaction_notresource (critical) : inversion NotAction.
resource "outscale_policy" "notaction" {
  policy_name = "pepin-qual-policy-notaction"
  description = "Tenant de qualification Pepin : tout sauf"
  path        = "/pepin_qual/"
  document = jsonencode({
    Statement = [{ Effect = "Allow", NotAction = ["api:DeleteVms"], Resource = ["*"] }]
  })
}

# ÉCART iam_policy_no_privilege_escalation (high) : créer une clé, lier une
# politique — le chemin d'élévation (porte aussi la ressource « * »).
resource "outscale_policy" "escalation" {
  policy_name = "pepin-qual-policy-escalation"
  description = "Tenant de qualification Pepin : elevation de privileges"
  path        = "/pepin_qual/"
  document = jsonencode({
    Statement = [{ Effect = "Allow", Action = ["api:CreateAccessKey", "api:LinkPolicy"], Resource = ["*"] }]
  })
}

# CONTRE-EXEMPLE des quatre contrôles iam_policy_* : actions bornées, ressource
# NOMMÉE (un bucket OOS), aucune inversion, aucune action d'identité.
resource "outscale_policy" "scoped" {
  policy_name = "pepin-qual-policy-scoped"
  description = "Tenant de qualification Pepin : lecture d un bucket nomme"
  path        = "/pepin_qual/"
  document = jsonencode({
    Statement = [{ Effect = "Allow", Action = ["oos:GetObject"], Resource = ["arn:aws:s3:::pepin-qual-${local.bucket_suffix}-hardened/*"] }]
  })
}

# ÉCART iam_apiaccessrule_no_public_cidr (CLD-IAM-4, high) : règle d'accès API
# ouverte à tout Internet. Les règles s'ajoutent (OU logique) : celle-ci n'enferme
# personne dehors, et elle est retirée avec le tenant.
resource "outscale_api_access_rule" "public" {
  ip_ranges   = ["0.0.0.0/0"]
  description = "pepin-qual-rule-public"
}

# CONTRE-EXEMPLE : une plage privée.
resource "outscale_api_access_rule" "private" {
  ip_ranges   = ["10.0.0.0/8"]
  description = "pepin-qual-rule-private"
}

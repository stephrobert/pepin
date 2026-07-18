# IAM Exoscale — code volontairement NON CONFORME (CLD-IAM-1/2/4).
# Le modèle IAM Exoscale exprime privilèges, expiration et restriction d'IP DANS
# LA POLICY (default_service_strategy + services{rules} avec expressions).
# Doc : community.exoscale.com/product/iam/how-to/policy-guide/

# ✗ Rôle à privilèges administratifs : default_service_strategy = "allow" est la
#   décision par défaut quand un service n'est pas listé ; avec services vides,
#   TOUTES les opérations de tous les services sont autorisées (= joker admin).
#   Source officielle : community.exoscale.com/product/iam/operation/roles-policies/
#   → iam_policy_no_administrative_privileges (CLD-IAM-1)
#   Aucune condition source_ip → accès non restreint par IP (CLD-IAM-4).
#   Aucune expiration dans la policy → clé sans rotation forcée (CLD-IAM-2).
resource "exoscale_iam_role" "admin" {
  name        = "pepin-test-admin"
  description = "Role de test a privileges excessifs (a supprimer)"

  policy = {
    default_service_strategy = "allow"
    services                 = {}
  }
}

# ✗ Clé d'API rattachée au rôle permissif, sans expiration ni restriction d'IP
#   (l'expiration/IP s'exprimeraient dans la policy du rôle, ici absentes).
resource "exoscale_iam_api_key" "admin" {
  name    = "pepin-test-admin-key"
  role_id = exoscale_iam_role.admin.id
}

# ✗ Rôle d'usage (deny par défaut, donc PAS admin) mais qui autorise la gestion des
#   rôles IAM : un porteur peut créer/modifier des rôles et s'octroyer plus de droits
#   → iam_policy_no_privilege_escalation (CLD-IAM-12).
resource "exoscale_iam_role" "deployer" {
  name        = "pepin-test-deployer"
  description = "Role d'usage autorisant la gestion IAM (a corriger)"

  policy = {
    default_service_strategy = "deny"
    services = {
      iam = {
        type = "rules"
        rules = [
          {
            action     = "allow"
            expression = "operation in ['create-iam-role', 'update-iam-role', 'create-api-key']"
          },
        ]
      }
    }
  }
}

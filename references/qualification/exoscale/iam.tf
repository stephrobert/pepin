# IAM — quatre rôles (plan seul). Le mapping lit la politique du rôle : stratégie
# par défaut (allow = administration), présence de `source_ip` (restriction
# d'origine) et de `duration(` (borne de durée de vie) dans les règles.
#
#   admin         allow par défaut, origine et durée bornées   → iam_role_no_admin_privileges
#   no_source_ip  deny par défaut, durée bornée, sans source_ip → iam_role_source_ip_restricted
#   unbounded     deny par défaut, source_ip, sans durée        → iam_role_key_lifetime_bounded
#   restricted    deny par défaut, source_ip et durée           → silencieux

resource "exoscale_iam_role" "admin" {
  name        = "pepin-qual-role-admin"
  description = "Tenant de qualification Pepin : administration"
  editable    = true
  labels      = local.untagged

  policy = {
    default_service_strategy = "allow"
    services = {
      compute = {
        type = "rules"
        rules = [{
          action     = "allow"
          expression = "source_ip in ['10.0.0.0/8'] && now - api_key.created_at < duration('720h')"
        }]
      }
    }
  }
}

resource "exoscale_iam_role" "no_source_ip" {
  name        = "pepin-qual-role-no-source-ip"
  description = "Tenant de qualification Pepin : sans restriction d origine"
  editable    = true
  labels      = local.untagged

  policy = {
    default_service_strategy = "deny"
    services = {
      compute = {
        type = "rules"
        rules = [{
          action     = "allow"
          expression = "now - api_key.created_at < duration('720h')"
        }]
      }
    }
  }
}

resource "exoscale_iam_role" "unbounded" {
  name        = "pepin-qual-role-unbounded"
  description = "Tenant de qualification Pepin : sans borne de duree"
  editable    = true
  labels      = local.untagged

  policy = {
    default_service_strategy = "deny"
    services = {
      compute = {
        type = "rules"
        rules = [{
          action     = "allow"
          expression = "source_ip in ['10.0.0.0/8']"
        }]
      }
    }
  }
}

resource "exoscale_iam_role" "restricted" {
  name        = "pepin-qual-role-restricted"
  description = "Tenant de qualification Pepin : origine et duree bornees"
  editable    = true
  labels      = local.untagged

  policy = {
    default_service_strategy = "deny"
    services = {
      compute = {
        type = "rules"
        rules = [{
          action     = "allow"
          expression = "source_ip in ['10.0.0.0/8'] && now - api_key.created_at < duration('720h')"
        }]
      }
    }
  }
}

# Réseau — groupes de sécurité, VPC et réseaux privés (ressources gratuites).
#
# UN groupe de sécurité PAR écart, pour que chaque finding ait un sujet qui ne
# porte que sa faute. La seule exception est voulue : une règle any/any ouvre par
# définition SSH, RDP et tous les ports sensibles, donc `sg_any_open` porte cinq
# écarts, et expected.yaml les attend tous les cinq.
#
# Tous les groupes sont EXPLICITES et attachés explicitement : le groupe créé par
# le produit (« Default security group ») bloque la destruction quand un serveur
# y atterrit sans que Terraform le connaisse (issue #2821 du provider).
#
# Aucune `scaleway_instance_private_nic` : voir versions.tf (issue #4338).

# ÉCART network_securitygroup_allow_ingress_from_internet_to_tcp_port_22 (high).
resource "scaleway_instance_security_group" "ssh_open" {
  name                    = "pepin-qual-sg-ssh-open"
  description             = "SSH ouvert a Internet"
  inbound_default_policy  = "drop"
  outbound_default_policy = "accept"
  tags                    = local.untagged

  inbound_rule {
    action   = "accept"
    protocol = "TCP"
    port     = 22
    ip_range = "0.0.0.0/0"
  }
}

# ÉCART …_tcp_port_3389 (high).
resource "scaleway_instance_security_group" "rdp_open" {
  name                    = "pepin-qual-sg-rdp-open"
  description             = "RDP ouvert a Internet"
  inbound_default_policy  = "drop"
  outbound_default_policy = "accept"
  tags                    = local.untagged

  inbound_rule {
    action   = "accept"
    protocol = "TCP"
    port     = 3389
    ip_range = "0.0.0.0/0"
  }
}

# ÉCART …_high_risk_tcp_ports (high) : PostgreSQL exposé.
resource "scaleway_instance_security_group" "db_open" {
  name                    = "pepin-qual-sg-db-open"
  description             = "PostgreSQL ouvert a Internet"
  inbound_default_policy  = "drop"
  outbound_default_policy = "accept"
  tags                    = local.untagged

  inbound_rule {
    action   = "accept"
    protocol = "TCP"
    port     = 5432
    ip_range = "0.0.0.0/0"
  }
}

# ÉCART …_high_risk_udp_ports (high) : SNMP exposé (vecteur d'amplification).
resource "scaleway_instance_security_group" "snmp_open" {
  name                    = "pepin-qual-sg-snmp-open"
  description             = "SNMP ouvert a Internet"
  inbound_default_policy  = "drop"
  outbound_default_policy = "accept"
  tags                    = local.untagged

  inbound_rule {
    action   = "accept"
    protocol = "UDP"
    port     = 161
    ip_range = "0.0.0.0/0"
  }
}

# ÉCART …_all_ports (critical) — et, par construction, les quatre autres contrôles
# d'entrée : une règle any/any couvre tout port et tout protocole.
resource "scaleway_instance_security_group" "any_open" {
  name                    = "pepin-qual-sg-any-open"
  description             = "Tout le trafic entrant accepte depuis Internet"
  inbound_default_policy  = "drop"
  outbound_default_policy = "accept"
  tags                    = local.untagged

  inbound_rule {
    action   = "accept"
    protocol = "ANY"
    ip_range = "0.0.0.0/0"
  }
}

# ÉCART …_tcp_port_22 par ÉVASION : quatre /2 couvrent tout l'espace IPv4 sans
# qu'aucun ne soit 0.0.0.0/0. C'est le cas #D de l'audit externe (issue #178), que
# lib.rego ferme par un seuil de largeur (/8) et par la fusion des plages.
resource "scaleway_instance_security_group" "quartet" {
  name                    = "pepin-qual-sg-quartet"
  description             = "SSH ouvert a Internet par quatre /2"
  inbound_default_policy  = "drop"
  outbound_default_policy = "accept"
  tags                    = local.untagged

  dynamic "inbound_rule" {
    for_each = ["0.0.0.0/2", "64.0.0.0/2", "128.0.0.0/2", "192.0.0.0/2"]
    content {
      action   = "accept"
      protocol = "TCP"
      port     = 22
      ip_range = inbound_rule.value
    }
  }
}

# ÉCART network_securitygroup_unrestricted_egress (medium) : une règle SORTANTE
# explicite any → 0.0.0.0/0. La politique sortante par défaut n'est pas une règle,
# donc elle ne compte pas ; seule la règle écrite est mesurée.
resource "scaleway_instance_security_group" "egress_any" {
  name                    = "pepin-qual-sg-egress-any"
  description             = "Tout le trafic sortant accepte vers Internet"
  inbound_default_policy  = "drop"
  outbound_default_policy = "drop"
  tags                    = local.untagged

  outbound_rule {
    action   = "accept"
    protocol = "ANY"
    ip_range = "0.0.0.0/0"
  }
}

# ÉCART network_securitygroup_default_deny (high) : politique entrante par défaut
# « accept » — observable sur les DEUX sources depuis #195. La collecte live projette
# désormais `inbound_default_policy`, lu dans la réponse de ListSecurityGroups que le
# collecteur paginait déjà pour joindre les règles à leur groupe.
resource "scaleway_instance_security_group" "default_accept" {
  name                    = "pepin-qual-sg-default-accept"
  description             = "Politique entrante par defaut accept"
  inbound_default_policy  = "accept"
  outbound_default_policy = "accept"
  tags                    = local.untagged
}

# CONTRE-EXEMPLE de TOUS les contrôles de groupe de sécurité : refus par défaut,
# SSH depuis un /8 PRIVÉ (la plage que le seuil de largeur ne doit jamais prendre
# pour Internet), HTTPS depuis Internet (un port servi, pas un port sensible), et
# une sortie TCP 443 seulement. Aucune règle ne doit parler.
resource "scaleway_instance_security_group" "hardened" {
  name                    = "pepin-qual-sg-hardened"
  description             = "Refus par defaut, SSH prive, HTTPS public"
  inbound_default_policy  = "drop"
  outbound_default_policy = "drop"
  tags                    = local.untagged

  inbound_rule {
    action   = "accept"
    protocol = "TCP"
    port     = 22
    ip_range = "10.0.0.0/8"
  }

  inbound_rule {
    action   = "accept"
    protocol = "TCP"
    port     = 443
    ip_range = "0.0.0.0/0"
  }

  outbound_rule {
    action   = "accept"
    protocol = "TCP"
    port     = 443
    ip_range = "0.0.0.0/0"
  }
}

# ── Plan seulement : la collecte live des réseaux privés est en attente ──────
#
# VPC explicite (gratuit) : le tenant ne dépend pas du VPC par défaut du projet,
# que Scaleway crée à la première utilisation et qu'on ne détruit pas.
resource "scaleway_vpc" "qual" {
  count = local.tf_only
  name  = "pepin-qual-vpc"
  tags  = local.untagged
}

# ÉCART network_documented (CLD-NET-5, low) : réseau sans étiquettes de cartographie
# (Owner, Project, Env). Observable sur le plan seulement (collecte live du VPC en
# attente, cf. questions-providers SCW-Q1). Aucun serveur n'y est attaché : une
# interface privée rendrait le destroy impossible (#4338).
resource "scaleway_vpc_private_network" "undocumented" {
  count  = local.tf_only
  name   = "pepin-qual-pn-undocumented"
  vpc_id = scaleway_vpc.qual[0].id
  tags   = local.untagged

  ipv4_subnet {
    subnet = "172.16.10.0/24"
  }
}

# CONTRE-EXEMPLE : le même réseau, cartographié.
resource "scaleway_vpc_private_network" "documented" {
  count  = local.tf_only
  name   = "pepin-qual-pn-documented"
  vpc_id = scaleway_vpc.qual[0].id
  tags   = concat([var.tenant_tag], ["Owner=pepin-maintainer", "Project=pepin", "Env=qualification"])

  ipv4_subnet {
    subnet = "172.16.20.0/24"
  }
}

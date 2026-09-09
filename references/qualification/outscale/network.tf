# Réseau — groupes de sécurité (tous dans le Net), Nets, sous-réseaux, appairage.
#
# UN groupe par écart. Chez Outscale, TOUT groupe naît avec une règle sortante par
# défaut « tout vers 0.0.0.0/0 » : elle est le contrôle CLD-NET-4 en personne, donc
# chaque groupe la RETIRE (`remove_default_outbound_rule`) sauf `egress_any`, qui
# porte la faute explicitement — visible sur le plan comme en live. Le groupe
# « default » du Net, créé par le produit, la garde et porte en plus la règle
# d'usine (source = lui-même) : deux écarts attendus sur un sujet que Terraform ne
# gère pas (sa lecture passe par une source de données, son identifiant par une
# sortie).

# Huit groupes NOMMÉS, pas un for_each : sur le plan, une référence vers une instance
# de for_each est ambiguë (ADR-0022) et le sujet retombait sur l'adresse de la RÈGLE ;
# nommés, le sujet est le groupe, comme en live. Dans le Net, et pas dans le cloud
# public : mesuré, un groupe du cloud public ne peut PAS retirer sa règle sortante
# par défaut (`remove_default_outbound_rule` exige `net_id`) — tout groupe y porte
# donc l'écart CLD-NET-4 par construction.
resource "outscale_security_group" "ssh_open" {
  security_group_name          = "pepin-qual-sg-ssh-open"
  description                  = "SSH ouvert a Internet"
  net_id                       = outscale_net.documented.net_id
  remove_default_outbound_rule = true

  dynamic "tags" {
    for_each = local.untagged
    content {
      key   = tags.key
      value = tags.value
    }
  }
}

resource "outscale_security_group" "rdp_open" {
  security_group_name          = "pepin-qual-sg-rdp-open"
  description                  = "RDP ouvert a Internet"
  net_id                       = outscale_net.documented.net_id
  remove_default_outbound_rule = true

  dynamic "tags" {
    for_each = local.untagged
    content {
      key   = tags.key
      value = tags.value
    }
  }
}

resource "outscale_security_group" "db_open" {
  security_group_name          = "pepin-qual-sg-db-open"
  description                  = "PostgreSQL ouvert a Internet"
  net_id                       = outscale_net.documented.net_id
  remove_default_outbound_rule = true

  dynamic "tags" {
    for_each = local.untagged
    content {
      key   = tags.key
      value = tags.value
    }
  }
}

resource "outscale_security_group" "snmp_open" {
  security_group_name          = "pepin-qual-sg-snmp-open"
  description                  = "SNMP ouvert a Internet"
  net_id                       = outscale_net.documented.net_id
  remove_default_outbound_rule = true

  dynamic "tags" {
    for_each = local.untagged
    content {
      key   = tags.key
      value = tags.value
    }
  }
}

resource "outscale_security_group" "any_open" {
  security_group_name          = "pepin-qual-sg-any-open"
  description                  = "Tout le trafic entrant accepte depuis Internet"
  net_id                       = outscale_net.documented.net_id
  remove_default_outbound_rule = true

  dynamic "tags" {
    for_each = local.untagged
    content {
      key   = tags.key
      value = tags.value
    }
  }
}

resource "outscale_security_group" "quartet" {
  security_group_name          = "pepin-qual-sg-quartet"
  description                  = "SSH ouvert a Internet par quatre /2"
  net_id                       = outscale_net.documented.net_id
  remove_default_outbound_rule = true

  dynamic "tags" {
    for_each = local.untagged
    content {
      key   = tags.key
      value = tags.value
    }
  }
}

resource "outscale_security_group" "egress_any" {
  security_group_name          = "pepin-qual-sg-egress-any"
  description                  = "Tout le trafic sortant accepte vers Internet"
  net_id                       = outscale_net.documented.net_id
  remove_default_outbound_rule = true

  dynamic "tags" {
    for_each = local.untagged
    content {
      key   = tags.key
      value = tags.value
    }
  }
}

resource "outscale_security_group" "hardened" {
  security_group_name          = "pepin-qual-sg-hardened"
  description                  = "Refus par defaut, SSH prive, HTTPS public"
  net_id                       = outscale_net.documented.net_id
  remove_default_outbound_rule = true

  dynamic "tags" {
    for_each = local.untagged
    content {
      key   = tags.key
      value = tags.value
    }
  }
}

# ÉCART …_tcp_port_22 (high).
resource "outscale_security_group_rule" "ssh_open" {
  flow              = "Inbound"
  security_group_id = outscale_security_group.ssh_open.security_group_id
  ip_protocol       = "tcp"
  from_port_range   = 22
  to_port_range     = 22
  ip_range          = "0.0.0.0/0"
}

# ÉCART …_tcp_port_3389 (high).
resource "outscale_security_group_rule" "rdp_open" {
  flow              = "Inbound"
  security_group_id = outscale_security_group.rdp_open.security_group_id
  ip_protocol       = "tcp"
  from_port_range   = 3389
  to_port_range     = 3389
  ip_range          = "0.0.0.0/0"
}

# ÉCART …_high_risk_tcp_ports (high) : PostgreSQL exposé.
resource "outscale_security_group_rule" "db_open" {
  flow              = "Inbound"
  security_group_id = outscale_security_group.db_open.security_group_id
  ip_protocol       = "tcp"
  from_port_range   = 5432
  to_port_range     = 5432
  ip_range          = "0.0.0.0/0"
}

# ÉCART …_high_risk_udp_ports (high) : SNMP exposé.
resource "outscale_security_group_rule" "snmp_open" {
  flow              = "Inbound"
  security_group_id = outscale_security_group.snmp_open.security_group_id
  ip_protocol       = "udp"
  from_port_range   = 161
  to_port_range     = 161
  ip_range          = "0.0.0.0/0"
}

# ÉCART …_all_ports (critical) — et, par construction, les quatre autres contrôles
# d'entrée : une règle any/any couvre tout port et tout protocole.
resource "outscale_security_group_rule" "any_open" {
  flow              = "Inbound"
  security_group_id = outscale_security_group.any_open.security_group_id
  ip_protocol       = "-1"
  ip_range          = "0.0.0.0/0"
}

# ÉCART …_tcp_port_22 par ÉVASION : quatre /2 couvrent tout l'espace IPv4 sans
# qu'aucun ne soit 0.0.0.0/0 (cas #D de l'audit externe).
resource "outscale_security_group_rule" "quartet" {
  for_each          = toset(["0.0.0.0/2", "64.0.0.0/2", "128.0.0.0/2", "192.0.0.0/2"])
  flow              = "Inbound"
  security_group_id = outscale_security_group.quartet.security_group_id
  ip_protocol       = "tcp"
  from_port_range   = 22
  to_port_range     = 22
  ip_range          = each.value
}

# ÉCART network_securitygroup_unrestricted_egress (medium) : la règle sortante
# « tout vers Internet », écrite explicitement.
resource "outscale_security_group_rule" "egress_any" {
  flow              = "Outbound"
  security_group_id = outscale_security_group.egress_any.security_group_id
  ip_protocol       = "-1"
  ip_range          = "0.0.0.0/0"
}

# CONTRE-EXEMPLE de TOUS les contrôles de groupe : SSH depuis un /8 PRIVÉ, HTTPS
# depuis Internet (un port servi, pas un port sensible), sortie TCP 443 seulement.
resource "outscale_security_group_rule" "hardened_ssh_private" {
  flow              = "Inbound"
  security_group_id = outscale_security_group.hardened.security_group_id
  ip_protocol       = "tcp"
  from_port_range   = 22
  to_port_range     = 22
  ip_range          = "10.0.0.0/8"
}

resource "outscale_security_group_rule" "hardened_https" {
  flow              = "Inbound"
  security_group_id = outscale_security_group.hardened.security_group_id
  ip_protocol       = "tcp"
  from_port_range   = 443
  to_port_range     = 443
  ip_range          = "0.0.0.0/0"
}

resource "outscale_security_group_rule" "hardened_egress_https" {
  flow              = "Outbound"
  security_group_id = outscale_security_group.hardened.security_group_id
  ip_protocol       = "tcp"
  from_port_range   = 443
  to_port_range     = 443
  ip_range          = "0.0.0.0/0"
}

# ── Nets ───────────────────────────────────────────────────────────────────────

# ÉCART network_documented (CLD-NET-5, low) : Net sans étiquettes de cartographie.
resource "outscale_net" "undocumented" {
  ip_range = "10.10.0.0/16"

  dynamic "tags" {
    for_each = merge(local.untagged, { Name = "pepin-qual-net-undocumented" })
    content {
      key   = tags.key
      value = tags.value
    }
  }
}

# CONTRE-EXEMPLE : cartographié (Owner, Project, Env). C'est lui qui porte les
# sous-réseaux, les VM à deux cartes et le SG « default » observé.
resource "outscale_net" "documented" {
  ip_range = "10.20.0.0/16"

  dynamic "tags" {
    for_each = merge(local.tagged, { Name = "pepin-qual-net-documented" })
    content {
      key   = tags.key
      value = tags.value
    }
  }
}

# ÉCART network_subnet_no_public_ip_by_default (CLD-NET-3, medium) : IP publique
# attribuée d'office à toute VM lancée ici. Aucune VM n'y est lancée : la faute est
# le réglage, pas une exposition.
resource "outscale_subnet" "autoip" {
  net_id                  = outscale_net.documented.net_id
  ip_range                = "10.20.1.0/24"
  subregion_name          = var.subregion
  map_public_ip_on_launch = true

  dynamic "tags" {
    for_each = merge(local.untagged, { Name = "pepin-qual-subnet-autoip" })
    content {
      key   = tags.key
      value = tags.value
    }
  }
}

# CONTRE-EXEMPLE, et le sous-réseau de travail des VM à deux cartes.
resource "outscale_subnet" "main" {
  net_id                  = outscale_net.documented.net_id
  ip_range                = "10.20.2.0/24"
  subregion_name          = var.subregion
  map_public_ip_on_launch = false

  dynamic "tags" {
    for_each = merge(local.untagged, { Name = "pepin-qual-subnet-main" })
    content {
      key   = tags.key
      value = tags.value
    }
  }
}

# Sortie Internet du Net : pour qu'une IP publique liée à une carte soit réelle.
resource "outscale_internet_service" "documented" {
  dynamic "tags" {
    for_each = merge(local.untagged, { Name = "pepin-qual-igw" })
    content {
      key   = tags.key
      value = tags.value
    }
  }
}

resource "outscale_internet_service_link" "documented" {
  internet_service_id = outscale_internet_service.documented.internet_service_id
  net_id              = outscale_net.documented.net_id
}

resource "outscale_route_table" "main" {
  net_id = outscale_net.documented.net_id

  dynamic "tags" {
    for_each = merge(local.untagged, { Name = "pepin-qual-rt-main" })
    content {
      key   = tags.key
      value = tags.value
    }
  }
}

resource "outscale_route" "default" {
  destination_ip_range = "0.0.0.0/0"
  gateway_id           = outscale_internet_service.documented.internet_service_id
  route_table_id       = outscale_route_table.main.route_table_id
  depends_on           = [outscale_internet_service_link.documented]
}

resource "outscale_route_table_link" "main" {
  route_table_id = outscale_route_table.main.route_table_id
  subnet_id      = outscale_subnet.main.subnet_id
}

# Groupes du Net : `net_ssh_open` porte SSH ouvert (la faute que la carte secondaire
# rend joignable), `net_closed` n'admet rien. Défaut sortant retiré sur les deux.
resource "outscale_security_group" "net_ssh_open" {
  security_group_name          = "pepin-qual-sg-net-ssh-open"
  description                  = "SSH ouvert a Internet, dans le Net"
  net_id                       = outscale_net.documented.net_id
  remove_default_outbound_rule = true

  dynamic "tags" {
    for_each = local.untagged
    content {
      key   = tags.key
      value = tags.value
    }
  }
}

resource "outscale_security_group_rule" "net_ssh_open" {
  flow              = "Inbound"
  security_group_id = outscale_security_group.net_ssh_open.security_group_id
  ip_protocol       = "tcp"
  from_port_range   = 22
  to_port_range     = 22
  ip_range          = "0.0.0.0/0"
}

resource "outscale_security_group" "net_closed" {
  security_group_name          = "pepin-qual-sg-net-closed"
  description                  = "Aucune regle entrante, dans le Net"
  net_id                       = outscale_net.documented.net_id
  remove_default_outbound_rule = true

  dynamic "tags" {
    for_each = local.untagged
    content {
      key   = tags.key
      value = tags.value
    }
  }
}

# Le SG « default » que le produit crée avec le Net : lu, jamais géré. Il porte la
# règle d'usine (ÉCART network_securitygroup_default_restrict_traffic, contextuel)
# et la règle sortante par défaut (ÉCART unrestricted_egress).
data "outscale_security_group" "net_default" {
  filter {
    name   = "net_ids"
    values = [outscale_net.documented.net_id]
  }
  filter {
    name   = "security_group_names"
    values = ["default"]
  }
}

# CONTRE-EXEMPLE de network_peering_cross_organization : un appairage entre deux
# Nets du MÊME compte (source_account = accepter_account). L'écart exigerait un
# second compte.
resource "outscale_net_peering" "same_account" {
  source_net_id   = outscale_net.documented.net_id
  accepter_net_id = outscale_net.undocumented.net_id

  dynamic "tags" {
    for_each = merge(local.untagged, { Name = "pepin-qual-peering" })
    content {
      key   = tags.key
      value = tags.value
    }
  }
}

resource "outscale_net_peering_acceptation" "same_account" {
  net_peering_id = outscale_net_peering.same_account.net_peering_id
}

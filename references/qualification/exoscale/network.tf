# Réseau — groupes de sécurité (une ressource par règle chez Exoscale) et réseaux
# privés. Un groupe par écart, un contre-exemple (`hardened`).
#
# Chez Exoscale, chaque règle porte une DESCRIPTION : c'est la matrice des flux
# (network_flow_matrix_documented, CLD-NET-5). Toutes les règles fautives en ont
# une, pour ne porter que leur faute ; `undocumented_flow` en est privée, sur un
# port servi (443), pour ne porter que celle-là.

locals {
  sgs = {
    ssh_open          = "SSH ouvert a Internet"
    rdp_open          = "RDP ouvert a Internet"
    db_open           = "PostgreSQL ouvert a Internet"
    snmp_open         = "SNMP ouvert a Internet"
    any_open          = "Tout le trafic entrant accepte depuis Internet"
    quartet           = "SSH ouvert a Internet par quatre /2"
    egress_any        = "Tout le trafic sortant accepte vers Internet"
    undocumented_flow = "HTTPS sans justification de flux"
    hardened          = "SSH prive, HTTPS public, sortie 443"
  }
}

resource "exoscale_security_group" "ssh_open" {
  name        = "pepin-qual-sg-ssh-open"
  description = local.sgs.ssh_open
}

resource "exoscale_security_group" "rdp_open" {
  name        = "pepin-qual-sg-rdp-open"
  description = local.sgs.rdp_open
}

resource "exoscale_security_group" "db_open" {
  name        = "pepin-qual-sg-db-open"
  description = local.sgs.db_open
}

resource "exoscale_security_group" "snmp_open" {
  name        = "pepin-qual-sg-snmp-open"
  description = local.sgs.snmp_open
}

resource "exoscale_security_group" "any_open" {
  name        = "pepin-qual-sg-any-open"
  description = local.sgs.any_open
}

resource "exoscale_security_group" "quartet" {
  name        = "pepin-qual-sg-quartet"
  description = local.sgs.quartet
}

resource "exoscale_security_group" "egress_any" {
  name        = "pepin-qual-sg-egress-any"
  description = local.sgs.egress_any
}

resource "exoscale_security_group" "undocumented_flow" {
  name        = "pepin-qual-sg-undocumented-flow"
  description = local.sgs.undocumented_flow
}

resource "exoscale_security_group" "hardened" {
  name        = "pepin-qual-sg-hardened"
  description = local.sgs.hardened
}

# ÉCART …_tcp_port_22 (high).
resource "exoscale_security_group_rule" "ssh_open" {
  security_group_id = exoscale_security_group.ssh_open.id
  type              = "INGRESS"
  protocol          = "TCP"
  cidr              = "0.0.0.0/0"
  start_port        = 22
  end_port          = 22
  description       = "SSH depuis Internet (faute voulue)"
}

# ÉCART …_tcp_port_3389 (high).
resource "exoscale_security_group_rule" "rdp_open" {
  security_group_id = exoscale_security_group.rdp_open.id
  type              = "INGRESS"
  protocol          = "TCP"
  cidr              = "0.0.0.0/0"
  start_port        = 3389
  end_port          = 3389
  description       = "RDP depuis Internet (faute voulue)"
}

# ÉCART …_high_risk_tcp_ports (high) : PostgreSQL.
resource "exoscale_security_group_rule" "db_open" {
  security_group_id = exoscale_security_group.db_open.id
  type              = "INGRESS"
  protocol          = "TCP"
  cidr              = "0.0.0.0/0"
  start_port        = 5432
  end_port          = 5432
  description       = "PostgreSQL depuis Internet (faute voulue)"
}

# ÉCART …_high_risk_udp_ports (high) : SNMP.
resource "exoscale_security_group_rule" "snmp_open" {
  security_group_id = exoscale_security_group.snmp_open.id
  type              = "INGRESS"
  protocol          = "UDP"
  cidr              = "0.0.0.0/0"
  start_port        = 161
  end_port          = 161
  description       = "SNMP depuis Internet (faute voulue)"
}

# TOUS les ports TCP et UDP depuis Internet. Chez Exoscale, une règle n'a PAS de
# protocole « tout » (valeurs : tcp, udp, icmp, icmpv6, ah, esp, gre, ipip) : le
# contrôle …_all_ports (critical, protocole `all`) n'est donc PAS constructible sur ce
# fournisseur — sur aucune source —, et tenant.yaml le consigne. Ces deux règles
# portent les quatre autres contrôles d'entrée.
resource "exoscale_security_group_rule" "any_open_tcp" {
  security_group_id = exoscale_security_group.any_open.id
  type              = "INGRESS"
  protocol          = "TCP"
  cidr              = "0.0.0.0/0"
  start_port        = 1
  end_port          = 65535
  description       = "Tout TCP depuis Internet (faute voulue)"
}

resource "exoscale_security_group_rule" "any_open_udp" {
  security_group_id = exoscale_security_group.any_open.id
  type              = "INGRESS"
  protocol          = "UDP"
  cidr              = "0.0.0.0/0"
  start_port        = 1
  end_port          = 65535
  description       = "Tout UDP depuis Internet (faute voulue)"
}

# ÉCART …_tcp_port_22 par ÉVASION : quatre /2.
resource "exoscale_security_group_rule" "quartet" {
  for_each          = toset(["0.0.0.0/2", "64.0.0.0/2", "128.0.0.0/2", "192.0.0.0/2"])
  security_group_id = exoscale_security_group.quartet.id
  type              = "INGRESS"
  protocol          = "TCP"
  cidr              = each.value
  start_port        = 22
  end_port          = 22
  description       = "SSH depuis un quart d'Internet (faute voulue)"
}

# Tout TCP vers Internet. Même limite : sans protocole « tout », la règle
# network_securitygroup_unrestricted_egress (protocole `all`) ne peut PAS se
# déclencher chez Exoscale. Consigné dans tenant.yaml ; cette règle ne porte donc
# aucun écart attendu.
resource "exoscale_security_group_rule" "egress_any" {
  security_group_id = exoscale_security_group.egress_any.id
  type              = "EGRESS"
  protocol          = "TCP"
  cidr              = "0.0.0.0/0"
  start_port        = 1
  end_port          = 65535
  description       = "Tout TCP vers Internet (faute voulue)"
}

# ÉCART network_flow_matrix_documented (medium) : une règle entrante SANS
# justification, sur un port servi.
resource "exoscale_security_group_rule" "undocumented_flow" {
  security_group_id = exoscale_security_group.undocumented_flow.id
  type              = "INGRESS"
  protocol          = "TCP"
  cidr              = "0.0.0.0/0"
  start_port        = 443
  end_port          = 443
}

# CONTRE-EXEMPLE : SSH depuis un /8 privé, HTTPS depuis Internet, sortie 443 —
# chaque flux justifié.
resource "exoscale_security_group_rule" "hardened_ssh_private" {
  security_group_id = exoscale_security_group.hardened.id
  type              = "INGRESS"
  protocol          = "TCP"
  cidr              = "10.0.0.0/8"
  start_port        = 22
  end_port          = 22
  description       = "Administration depuis le reseau interne"
}

resource "exoscale_security_group_rule" "hardened_https" {
  security_group_id = exoscale_security_group.hardened.id
  type              = "INGRESS"
  protocol          = "TCP"
  cidr              = "0.0.0.0/0"
  start_port        = 443
  end_port          = 443
  description       = "Service web public"
}

resource "exoscale_security_group_rule" "hardened_egress_https" {
  security_group_id = exoscale_security_group.hardened.id
  type              = "EGRESS"
  protocol          = "TCP"
  cidr              = "0.0.0.0/0"
  start_port        = 443
  end_port          = 443
  description       = "Mises a jour et API sortantes"
}

# ÉCART network_documented (CLD-NET-5, low) : réseau privé sans étiquettes de
# cartographie ni description.
resource "exoscale_private_network" "undocumented" {
  zone   = var.zone
  name   = "pepin-qual-pn-undocumented"
  labels = local.untagged
}

# CONTRE-EXEMPLE : cartographié.
resource "exoscale_private_network" "documented" {
  zone        = var.zone
  name        = "pepin-qual-pn-documented"
  description = "Reseau applicatif du tenant de qualification"
  labels      = merge(local.untagged, { Owner = "pepin-maintainer", Project = "pepin", Env = "qualification" })
}

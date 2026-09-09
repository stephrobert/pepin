# Règles d'exposition des security groups — COMMUNES à tous les providers.
#   Lisent le type normalisé `security_group_rule` (schéma commun produit par les
#   collecteurs/mappers de chaque provider) : direction, action, protocol
#   (tcp|udp|icmp|all), port_from, port_to, cidrs[], security_group_id.
#   `labels.provider` est tiré de la ressource.
# SCSL : CLD-NET-1 (ports d'admin/sensibles), CLD-NET-2 (any/any).
package pepin.rules

import rego.v1

# SSH (22) ouvert à Internet → CLD-NET-1.
deny contains f if {
	some r in resources_of_type("security_group_rule")
	sg_inbound_from_internet(r.attributes)
	proto_covers(r.attributes, "tcp")
	covers_port(r.attributes, 22)
	f := _sg_finding(r, "network_securitygroup_allow_ingress_from_internet_to_tcp_port_22", "high", "SSH (port 22)", "SSH (port 22)")
}

# RDP (3389) ouvert à Internet → CLD-NET-1.
deny contains f if {
	some r in resources_of_type("security_group_rule")
	sg_inbound_from_internet(r.attributes)
	proto_covers(r.attributes, "tcp")
	covers_port(r.attributes, 3389)
	f := _sg_finding(r, "network_securitygroup_allow_ingress_from_internet_to_tcp_port_3389", "high", "RDP (port 3389)", "RDP (port 3389)")
}

# Ports sensibles TCP (BD, annuaire, orchestration…) ouverts à Internet → CLD-NET-1.
# AGRÉGÉ : une règle « tous ports » ouverte ne produit qu'UN finding listant les ports, pas ~38.
deny contains f if {
	some r in resources_of_type("security_group_rule")
	sg_inbound_from_internet(r.attributes)
	proto_covers(r.attributes, "tcp")
	ports := sort([p | some p in (sensitive_ports - {22, 3389}); covers_port(r.attributes, p)])
	count(ports) > 0
	f := _sg_finding(r, "network_securitygroup_allow_ingress_from_internet_to_high_risk_tcp_ports", "high", sprintf("ports sensibles TCP %v", [ports]), sprintf("high-risk TCP ports %v", [ports]))
}

# Ports sensibles UDP (amplification DDoS, services non authentifiés) ouverts à Internet → CLD-NET-1.
deny contains f if {
	some r in resources_of_type("security_group_rule")
	sg_inbound_from_internet(r.attributes)
	proto_covers(r.attributes, "udp")
	ports := sort([p | some p in sensitive_udp_ports; covers_port(r.attributes, p)])
	count(ports) > 0
	f := _sg_finding(r, "network_securitygroup_allow_ingress_from_internet_to_high_risk_udp_ports", "high", sprintf("ports sensibles UDP %v", [ports]), sprintf("high-risk UDP ports %v", [ports]))
}

# Tout le trafic entrant depuis Internet (any/any) → CLD-NET-2.
deny contains f if {
	some r in resources_of_type("security_group_rule")
	sg_inbound_from_internet(r.attributes)
	lower(object.get(r.attributes, "protocol", "")) == "all"
	f := _sg_finding(r, "network_securitygroup_allow_ingress_from_internet_to_all_ports", "critical", "tout le trafic (any/any)", "all traffic (any/any)")
}

# Sortie tout-trafic non restreinte vers Internet → CLD-NET-4.
deny contains f if {
	some r in resources_of_type("security_group_rule")
	lower(object.get(r.attributes, "direction", "")) == "outbound"
	sg_accepting(r.attributes)
	lower(object.get(r.attributes, "protocol", "")) == "all"
	some cidr in cidr_list(object.get(r.attributes, "cidrs", []))
	is_public_cidr(cidr)
	f := _avec_confiance(
		_sg_finding(r, "network_securitygroup_unrestricted_egress", "medium", "tout le trafic sortant", "all outbound traffic"),
		"contextual",
	)
}

# _sg_finding : finding commun des règles d'exposition. `what` nomme CE QUI est
# accepté (le fragment interpolé dans le message) ; `what_en` en est la
# contrepartie anglaise, passée par l'appelant pour que la phrase anglaise reste
# entière plutôt que mi-traduite.
_sg_finding(r, code, sev, what, what_en) := {
	"code": code,
	"severity": sev,
	"subject": object.get(r.attributes, "security_group_id", r.id),
	"message": sprintf("Security group « %s » : %s accepté %s Internet.", [object.get(r.attributes, "security_group_id", r.id), what, _sens_fr(r.attributes)]),
	"remediation": "Restreindre la règle à des sources/destinations et ports légitimes (CIDR d'administration, bastion, VPN) ; ne jamais exposer un service sensible à 0.0.0.0/0.",
	"labels": {
		"provider": provider_of(r),
		"category": "security",
		# CONFIRMÉ pour les règles d'ENTRÉE que ce constructeur sert : une origine
		# `0.0.0.0/0` sur un port sensible est OBSERVÉE dans la configuration, pas
		# inférée. Le contexte n'y change rien — il n'existe pas de raison légitime
		# d'ouvrir SSH ou RDP à tout Internet, seulement des raisons temporaires, qui
		# se posent en dérogation datée.
		#
		# Cet argument ne vaut PAS pour la sortie, et l'egress se réétiquette donc
		# `contextual` chez l'appelant. Une sortie ouverte est un chemin
		# d'exfiltration réel, mais des architectures parfaitement défendables la
		# laissent ouverte et filtrent en aval — passerelle, mandataire, pare-feu
		# périmétrique — sur un plan que le scan ne voit pas. C'est la définition même
		# de `contextual` ici, et la porte `--gate security` la met de côté sans que
		# l'écart quitte le rapport (ADR-0019).
		"confidence": "confirmed",
		"message_en": sprintf("Security group \"%s\": %s accepted %s the internet.", [object.get(r.attributes, "security_group_id", r.id), what_en, _sens_en(r.attributes)]),
		"remediation_en": "Restrict the rule to legitimate sources/destinations and ports (administration CIDR, bastion, VPN); never expose a sensitive service to 0.0.0.0/0.",
	},
}

# _avec_confiance : remplace la seule confiance d'un finding, en laissant le reste des
# étiquettes intact. Écrit plutôt que recopié dans le constructeur, parce qu'un
# constructeur qui prendrait la confiance en paramètre obligerait les six appelants à la
# répéter — et une valeur répétée six fois est une valeur qui finit par diverger.
_avec_confiance(f, c) := object.union(f, {"labels": object.union(f.labels, {"confidence": c})})

# _sens_fr, _sens_en : la préposition suit la DIRECTION de la règle.
#
# Le constructeur écrivait « depuis/vers Internet » pour les six règles, ce qui rendait
# la phrase fausse d'un côté à chaque fois : une règle sortante n'accepte rien « depuis »
# Internet. Un lecteur qui corrige ce que la phrase décrit va chercher au mauvais
# endroit, et c'est exactement le défaut de précision que ce lot traite.
#
# Le repli garde les deux prépositions : sans direction lue, on ne CHOISIT pas.
_sens_fr(attrs) := "depuis" if lower(object.get(attrs, "direction", "")) == "inbound"

_sens_fr(attrs) := "vers" if lower(object.get(attrs, "direction", "")) == "outbound"

_sens_fr(attrs) := "depuis/vers" if not lower(object.get(attrs, "direction", "")) in {"inbound", "outbound"}

_sens_en(attrs) := "from" if lower(object.get(attrs, "direction", "")) == "inbound"

_sens_en(attrs) := "to" if lower(object.get(attrs, "direction", "")) == "outbound"

_sens_en(attrs) := "from/to" if not lower(object.get(attrs, "direction", "")) in {"inbound", "outbound"}

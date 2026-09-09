# network_securitygroup_default_restrict_traffic
#   Le security group « default » d'un réseau est attaché automatiquement à toute
#   ressource dont on ne précise pas le SG. Il doit donc ne RIEN autoriser : la moindre
#   règle entrante qu'il porte s'applique silencieusement à ces ressources, sans décision
#   explicite. Équivalent : Prowler ec2_securitygroup_default_restrict_traffic.
# SCSL : CLD-NET-4 (filtrage réseau au moindre privilège).
# Contrat : type normalisé agnostique `security_group_rule` ; attributs
#   security_group_name (nom natif du SG) et direction. Nom absent ⇒ pas de finding
#   (garde de capacité) : l'assessment rend « non évalué » plutôt qu'un faux vert.
#
# # Ce que la règle NOMME, et pourquoi
#
# Sur un réseau tout juste créé, le provider fabrique lui-même le SG « default » et lui
# donne une règle entrante auto-référencée : la seule source admise est le groupe
# lui-même. Le constat reste juste — deux ressources démarrées sans SG explicite y
# atterrissent et se parlent librement, ce qui est exactement ce que CLD-NET-4 refuse —
# mais le message, lui, ne l'était pas : il annonçait « porte une règle entrante » à un
# exploitant qui allait chercher une règle qu'il aurait écrite, et n'en trouvait aucune.
#
# La règle distingue donc les deux formes, et elle les distingue en les OBSERVANT. La
# source par groupe est un champ collecté (`peer_security_group_ids`), pas une déduction
# tirée de l'absence de CIDR : sans ce champ, « pas de CIDR » ne dirait pas si la règle
# n'admet personne ou si elle admet un groupe qu'on n'a pas lu (ADR-0014).
#
# Quand le champ manque — un provider qui ne l'expose pas, une source qui ne le porte
# pas —, la règle retombe sur la formulation générale et sur `confirmed`. Le repli penche
# vers la SÉVÉRITÉ : une caractérisation impossible ne doit pas faire sortir l'écart
# d'une porte de CI (même raisonnement qu'à l'ADR-0019 pour un label absent).
package pepin.rules

import rego.v1

deny contains f if {
	some r in resources_of_type("security_group_rule")
	lower(object.get(r.attributes, "security_group_name", "")) == "default"
	lower(object.get(r.attributes, "direction", "")) == "inbound"
	sg := object.get(r.attributes, "security_group_id", r.id)
	forme := _forme_default(r.attributes, sg)
	f := {
		"code": "network_securitygroup_default_restrict_traffic",
		"severity": "high",
		"subject": sg,
		"message": forme.message,
		"remediation": forme.remediation,
		"labels": {
			"provider": provider_of(r),
			"category": "security",
			"confidence": forme.confidence,
			"message_en": forme.message_en,
			"remediation_en": forme.remediation_en,
		},
	}
}

# _forme_default : la règle d'usine auto-référencée, OBSERVÉE.
#
# `contextual` parce que ce que cette forme fait courir dépend d'un contexte que le scan
# ne regarde pas : si rien ne démarre jamais sans SG explicite, le groupe reste vide et
# la règle ne s'applique à personne. L'écart reste au rapport et compte dans la porte par
# défaut ; il cesse seulement de casser une chaîne `--gate security`, ce qui est
# précisément ce que cette dimension existe pour permettre.
_forme_default(attrs, sg) := forme if {
	_regle_dusine(attrs, sg)
	forme := {
		"confidence": "contextual",
		"message": sprintf("Security group « default » (%s) : sa seule source admise est le groupe lui-même — c'est la règle d'usine, créée avec le réseau, non une règle écrite par l'exploitant. Elle reste un écart : deux ressources démarrées sans SG explicite y atterrissent et communiquent librement.", [sg]),
		"remediation": "Ne rien laisser atterrir dans le groupe « default » : attacher explicitement un SG dédié et restrictif à chaque ressource. Vider ensuite ses règles, la règle d'usine comprise.",
		"message_en": sprintf("Security group \"default\" (%s): its only accepted source is the group itself — this is the factory rule, created with the network, not a rule an operator wrote. It remains a deviation: two resources started without an explicit SG land in it and talk to each other freely.", [sg]),
		"remediation_en": "Let nothing land in the \"default\" group: explicitly attach a dedicated, restrictive SG to every resource. Then empty its rules, the factory rule included.",
	}
}

# _forme_default : tout le reste — une source qu'il a fallu écrire, ou une forme que la
# collecte ne permet pas de caractériser. Formulation générale, `confirmed`.
_forme_default(attrs, sg) := forme if {
	not _regle_dusine(attrs, sg)
	forme := {
		"confidence": "confirmed",
		"message": sprintf("Security group « default » (%s) : porte une règle entrante — il s'applique d'office à toute ressource créée sans SG explicite.", [sg]),
		"remediation": "Vider le security group « default » de toutes ses règles ; attacher explicitement un SG dédié et restrictif à chaque ressource.",
		"message_en": sprintf("Security group \"default\" (%s) carries an inbound rule — it applies automatically to every resource created without an explicit SG.", [sg]),
		"remediation_en": "Empty the \"default\" security group of all its rules; explicitly attach a dedicated, restrictive SG to every resource.",
	}
}

# _regle_dusine : vrai quand la seule source admise est le groupe lui-même. Écrit une
# fois, employé par les deux clauses, pour qu'elles ne puissent pas diverger — deux
# corps qui se contrediraient rendraient la règle indéfinie, et donc muette.
_regle_dusine(attrs, sg) if {
	count(cidr_list(object.get(attrs, "cidrs", []))) == 0
	pairs := [p | some p in cidr_list(object.get(attrs, "peer_security_group_ids", []))]
	count(pairs) > 0
	every p in pairs {
		p == sg
	}
}

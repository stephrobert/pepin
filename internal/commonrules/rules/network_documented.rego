# network_documented — cartographie réseau tenue.
#
# Ce que la règle vérifie, exactement : chaque réseau (VPC / Net / réseau privé)
# porte les étiquettes de gouvernance qui le DOCUMENTENT — par défaut
# propriétaire, projet et environnement (profil `tagging.network_required_tags`).
#
# Pourquoi ce n'est plus `count(tags) > 0` : la règle annonçait « propriétaire,
# projet, environnement » et se contentait d'un tag quelconque. Un unique
# `foo=bar` suffisait à déclarer un réseau documenté — une conformité affirmée
# sans avoir été mesurée, c'est-à-dire le `PASS` non prouvé que cette vague
# combat. Le CODE du contrôle, lui, n'est PAS renommé : il voyage dans les
# `ruleId` SARIF, dans les assessments archivés et dans les fichiers de
# dérogations, où un renommage transformerait du jour au lendemain une dérogation
# valide en dérogation orpheline. On corrige la promesse, pas l'identifiant.
#
# Garde de capacité : la règle ne se déclenche que si l'attribut `tags` a été
# COLLECTÉ. Sans lui, on ne sait pas si le réseau est documenté — et « je ne sais
# pas » ne se dit pas « non conforme ».
#
# Ancrage : Exoscale private-network (labels), Scaleway vpc_private_network
# (tags), Outscale Net (Tags). SCSL : CLD-NET-5 (cartographie réseau tenue).
package pepin.rules

import rego.v1

deny contains f if {
	some n in resources_of_type("network")
	"tags" in object.keys(n.attributes) # garde de capacité : le provider EXPOSE les étiquettes
	missing := missing_required_tags(object.get(n.attributes, "tags", []), required_tags_network)
	count(missing) > 0
	name := object.get(n.attributes, "name", n.id)
	f := {
		"code": "network_documented",
		"severity": "low",
		"subject": name,
		"message": sprintf("Réseau « %s » : étiquettes de cartographie manquantes (%s) — la cartographie réseau n'est pas tenue.", [name, concat(", ", missing)]),
		"remediation": sprintf("Étiqueter chaque réseau (%s) ; tenir la cartographie réseau à jour.", [required_tags_label(required_tags_network)]),
		"labels": {
			"provider": provider_of(n),
			"category": "compliance",
			"message_en": sprintf("Network \"%s\": mapping tags missing (%s) — the network mapping is not maintained.", [name, concat(", ", missing)]),
			"remediation_en": sprintf("Tag every network (%s); keep the network mapping up to date.", [required_tags_label(required_tags_network)]),
		},
	}
}

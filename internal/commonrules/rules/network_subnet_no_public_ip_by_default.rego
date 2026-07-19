# network_subnet_no_public_ip_by_default
#   Sous-réseau qui attribue automatiquement une IP publique à toute interface
#   créée : les machines sont exposées à Internet par défaut, sans décision
#   explicite.
# SCSL : CLD-NET-3 (pas d'exposition directe machine + filtrage ouvert).
# Contrat : type normalisé agnostique `subnet` ; attribut map_public_ip_on_launch
#   (bool, DÉRIVÉ — Outscale ReadSubnets MapPublicIpOnLaunch). Absent ⇒ pas de
#   finding (garde de capacité).
package pepin.rules

import rego.v1

deny contains f if {
	some r in resources_of_type("subnet")
	object.get(r.attributes, "map_public_ip_on_launch", false) == true
	id := object.get(r.attributes, "subnet_id", r.id)
	f := {
		"code": "network_subnet_no_public_ip_by_default",
		"severity": "medium",
		"subject": id,
		"message": sprintf("Sous-réseau « %s » : attribution automatique d'IP publique à la création (exposition Internet par défaut).", [id]),
		"remediation": "Désactiver l'attribution automatique d'IP publique du sous-réseau ; n'attribuer une IP publique qu'aux interfaces qui le nécessitent.",
		"labels": {"provider": provider_of(r), "category": "security"},
	}
}

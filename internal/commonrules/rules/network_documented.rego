# Cartographie réseau tenue : chaque réseau (VPC / Net / private network) doit être
#   identifié (nom) et étiqueté (propriétaire/projet/environnement) pour figurer dans
#   une cartographie exploitable. Type normalisé `network`, attributs name + tags
#   ([{key,value}]). Ne se déclenche que pour les réseaux collectés (le type existe).
# Ancrage : Exoscale private-network (labels), Scaleway vpc_private_network (tags),
#   Outscale Net (Tags). SCSL : CLD-NET-5 (cartographie réseau tenue à jour).
package pepin.rules

import rego.v1

deny contains f if {
	some n in resources_of_type("network")
	not _network_documented(n)
	name := object.get(n.attributes, "name", n.id)
	f := {
		"code": "network_documented",
		"severity": "low",
		"subject": name,
		"message": sprintf("Réseau « %s » sans étiquettes de gouvernance — cartographie réseau non tenue (propriétaire/projet/environnement).", [name]),
		"remediation": "Étiqueter chaque réseau (propriétaire, projet, environnement) ; tenir la cartographie réseau à jour.",
		"labels": {"provider": provider_of(n), "category": "compliance"},
	}
}

# Réseau documenté : porte au moins une étiquette de gouvernance. (Le nom est, selon
# le provider, un champ natif ou un tag « Name » — on s'appuie donc sur les tags.)
_network_documented(n) if count(object.get(n.attributes, "tags", [])) > 0

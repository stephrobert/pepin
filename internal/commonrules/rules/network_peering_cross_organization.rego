# Cloisonnement inter-SI : un appairage réseau (peering) reliant deux comptes /
#   organisations DIFFÉRENTS ouvre les flux internes vers un autre système
#   d'information. Type normalisé `network_peering`, attributs source_account /
#   accepter_account (+ state). Ne se déclenche que si les deux comptes sont connus.
# Ancrage Outscale : ReadNetPeerings → NetPeerings[] {SourceNet.AccountId,
#   AccepterNet.AccountId, State.Name} ; un peering peut relier des comptes
#   différents (doc about-net-peerings). SCSL : CLD-NET-7 (cloisonnement des flux
#   internes vis-à-vis des autres SI).
package pepin.rules

import rego.v1

deny contains f if {
	some p in resources_of_type("network_peering")
	src := object.get(p.attributes, "source_account", "")
	acc := object.get(p.attributes, "accepter_account", "")
	src != ""
	acc != ""
	src != acc
	not peering_inactive(p)
	f := {
		"code": "network_peering_cross_organization",
		"severity": "high",
		"subject": object.get(p.attributes, "peering_id", p.id),
		"message": sprintf("Appairage réseau « %s » entre deux comptes distincts (%s ↔ %s) : flux internes ouverts vers un autre SI — cloisonnement à justifier.", [object.get(p.attributes, "peering_id", p.id), src, acc]),
		"remediation": "Limiter l'appairage aux réseaux du même SI ; pour un partenaire, justifier et restreindre les routes/flux échangés (matrice de flux).",
		"labels": {"provider": provider_of(p), "category": "compliance"},
	}
}

# Un appairage non actif (rejeté/expiré/supprimé) ne crée pas de flux.
peering_inactive(p) if object.get(p.attributes, "state", "active") in {"rejected", "failed", "expired", "deleted"}

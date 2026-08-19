# database_service_not_open_to_internet
#   Base de données managée dont la liste d'IP autorisées (ACL) admet un CIDR
#   public : le service de base de données est joignable depuis Internet.
# SCSL : CLD-NET-1 (source restreinte à un CIDR maîtrisé, jamais 0.0.0.0/0).
# Contrat : type normalisé agnostique `managed_database` ; attribut ip_filter
#   (liste de CIDR autorisés, DÉRIVÉ — Scaleway RDB ACLs / Exoscale DBaaS
#   ip-filter). Absent ⇒ pas de finding (garde de capacité).
# LIMITE (source Terraform) : la règle ne voit que les ACL DÉCLARÉES. Le défaut d'API
#   Scaleway est « ACL initiale 0.0.0.0/0 » (base ouverte tant qu'aucune ACL n'est posée) ;
#   détecter une instance SANS ACL exige de corréler instance et ACL par leur id RÉEL, ce
#   que le plan (ids « known after apply ») ne permet pas → à traiter en COLLECTE LIVE
#   (rdb/v1 instances + acls), pas au niveau du plan.
package pepin.rules

import rego.v1

deny contains f if {
	some r in resources_of_type("managed_database")
	"ip_filter" in object.keys(r.attributes)
	some cidr in cidr_list(object.get(r.attributes, "ip_filter", []))
	is_public_cidr(cidr)
	id := object.get(r.attributes, "database_id", r.id)
	f := {
		"code": "database_service_not_open_to_internet",
		"severity": "high",
		"subject": id,
		"message": sprintf("Base de données managée « %s » : ACL autorisant un CIDR public (%s) — service exposé à Internet.", [id, cidr]),
		"remediation": "Restreindre l'ACL de la base aux seuls CIDR applicatifs (réseau privé quand disponible) ; retirer 0.0.0.0/0.",
		"labels": {"provider": provider_of(r), "category": "security"},
	}
}

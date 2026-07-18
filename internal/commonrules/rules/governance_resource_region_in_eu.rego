# governance_resource_region_in_eu — COMMUN à tous les providers.
#   Ressource hébergée hors Union européenne : la localisation des données et du
#   plan de contrôle doit rester dans l'UE (souveraineté).
# SCSL : CLD-GVN-3 (localisation UE du stockage/traitement/administration).
#   La région est portée par le modèle commun (model.Resource.Region) ; la
#   classification vient des tables de lib.rego (region_in_eu / region_trusted /
#   region_known), ancrées sur la géographie + l'exposition extraterritoriale.
#   Deux niveaux : espace de confiance (EEE + Suisse) = écart MINEUR (low) ;
#   hors espace souverain (US, Asie…) = écart MAJEUR (high). Région inconnue ⇒
#   aucun finding (pas de faux positif). Voir CLD-GVN-4 pour l'extraterritorialité.
package pepin.rules

import rego.v1

# Types LOCALISÉS : ressources qui hébergent des données ou un traitement (la
# localisation y est pertinente pour la souveraineté). Les identités (iam_user,
# access_key, iam_policy) et le filtrage réseau n'y figurent pas.
_located_types := {
	"compute_instance",
	"object_storage_bucket",
	"blockstorage_volume",
	"blockstorage_snapshot",
	"kubernetes_cluster",
	"load_balancer",
}

# Espace européen de confiance (EEE + Suisse) : hors UE stricte mais adéquat et
# sans loi extraterritoriale → écart mineur.
deny contains f if {
	some r in input.resources
	r.type in _located_types
	reg := object.get(r, "region", "")
	reg != ""
	p := provider_of(r)
	region_trusted(p, reg)
	name := object.get(r, "name", r.id)
	f := {
		"code": "governance_resource_region_in_eu",
		"severity": "low",
		"subject": name,
		"message": sprintf("Ressource « %s » dans la région « %s » : hors Union européenne mais en espace européen de confiance (EEE/Suisse), niveau de protection adéquat et sans loi extraterritoriale.", [name, reg]),
		"remediation": "Si une localisation strictement UE est exigée, migrer vers une région de l'Union européenne ; sinon documenter l'acceptation du risque (zone adéquate).",
		"labels": {"provider": p, "category": "compliance"},
	}
}

# Hors espace souverain européen → écart majeur.
deny contains f if {
	some r in input.resources
	r.type in _located_types
	reg := object.get(r, "region", "")
	reg != ""
	p := provider_of(r)
	region_known(p, reg)
	not region_in_eu(p, reg)
	not region_trusted(p, reg)
	name := object.get(r, "name", r.id)
	f := {
		"code": "governance_resource_region_in_eu",
		"severity": "high",
		"subject": name,
		"message": sprintf("Ressource « %s » hébergée dans la région « %s », hors espace souverain européen — exigence de localisation UE non respectée.", [name, reg]),
		"remediation": "Recréer ou migrer la ressource dans une région de l'Union européenne ; restreindre les régions autorisées au niveau de l'organisation.",
		"labels": {"provider": p, "category": "compliance"},
	}
}

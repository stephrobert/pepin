# governance_resource_region_in_eu — COMMUN à tous les providers.
#   Ressource hébergée hors Union européenne : la localisation des données et du
#   plan de contrôle doit rester dans l'UE (souveraineté).
# SCSL : CLD-GVN-3 (localisation UE du stockage/traitement/administration).
#   La région est portée par le modèle commun (model.Resource.Region) ; la
#   classification vient des tables de lib.rego (region_in_eu / region_trusted /
#   region_known), ancrées sur la géographie + l'exposition extraterritoriale.
#   Trois niveaux : espace de confiance (EEE + Suisse) = écart MINEUR (low) ;
#   hors espace souverain (US, Asie…) = écart MAJEUR (high) ; région PRÉSENTE mais
#   absente des tables = écart MOYEN (medium), car la localisation n'est alors pas
#   vérifiable — se taire reviendrait à certifier « UE » une région jamais classée,
#   et à le faire en silence le jour où un fournisseur en ouvre une hors UE.
#   Voir CLD-GVN-4 pour l'extraterritorialité.
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
	"managed_database",
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
		"labels": {
			"provider": p,
			"category": "compliance",
			"message_en": sprintf("Resource \"%s\" in region \"%s\": outside the European Union but within the European trusted area (EEA/Switzerland), an adequate level of protection with no extraterritorial law.", [name, reg]),
			"remediation_en": "If a strictly EU location is required, migrate to a European Union region; otherwise document the accepted risk (adequate area).",
		},
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
		"labels": {
			"provider": p,
			"category": "compliance",
			"message_en": sprintf("Resource \"%s\" hosted in region \"%s\", outside the European sovereign area — the EU location requirement is not met.", [name, reg]),
			"remediation_en": "Recreate or migrate the resource in a European Union region; restrict the allowed regions at the organisation level.",
		},
	}
}

# Région renseignée mais absente des tables de classification : la localisation
# n'est PAS vérifiable. Les tables sont des listes blanches, donc leur silence
# valait « conforme » — un fail-open structurel pour un outil dont c'est le cœur
# de métier. On rend l'incertitude visible plutôt que de la résoudre en faveur
# de la conformité.
deny contains f if {
	some r in input.resources
	r.type in _located_types
	reg := object.get(r, "region", "")
	reg != ""
	p := provider_of(r)
	not region_known(p, reg)
	name := object.get(r, "name", r.id)
	f := {
		"code": "governance_resource_region_in_eu",
		"severity": "medium",
		"subject": name,
		"message": sprintf("Ressource « %s » : région « %s » non cataloguée pour le fournisseur « %s » — localisation non vérifiable, conformité UE ni établie ni infirmée.", [name, reg, p]),
		"remediation": "Vérifier la localisation réelle de cette région auprès du fournisseur, puis l'ajouter aux tables de classification (lib.rego) ; migrer la ressource si elle est hors UE.",
		"labels": {
			"provider": p,
			"category": "compliance",
			"message_en": sprintf("Resource \"%s\": region \"%s\" is not catalogued for provider \"%s\" — the location cannot be verified, EU compliance is neither established nor ruled out.", [name, reg, p]),
			"remediation_en": "Check the real location of this region with the provider, then add it to the classification tables (lib.rego); migrate the resource if it sits outside the EU.",
		},
	}
}

# governance_resource_required_tags — COMMUN à tous les providers.
#
# Ce que la règle vérifie : une ressource FACTURABLE porte les étiquettes de
# gouvernance exigées par la politique d'étiquetage (`tagging.required_tags`),
# par défaut centre de coût, projet, environnement et propriétaire.
#
# La politique est CONFIGURABLE, et elle ne l'est pas par confort. Le profil par
# défaut est une RECOMMANDATION, pas une norme : une organisation parfaitement
# gouvernée peut écrire `cost-center, application, environment, team` là où le
# profil dit `cost-center, project, environment, owner`, et récolter un FAIL sur
# une convention d'écriture. La comparaison est donc insensible à la casse et aux
# séparateurs (`cost-center` ≡ `CostCenter`), et les alias élargissent chaque nom
# logique. Ce qui est exigé, ce sont les QUESTIONS auxquelles un inventaire doit
# répondre — qui paye, pour quoi, à quel stade, qui répond —, jamais les mots.
#
# Les TYPES visés sont eux aussi explicites (`tagging.resource_types`) : le
# critère est « facturable et étiquetable », et le détail de ce qui est inclus,
# de ce qui est exclu et pourquoi vit dans internal/policy (defaultTaggedTypes).
#
# SCSL : CLD-GVN-1. Attribut lu : tags[] ({key,value}).
package pepin.rules

import rego.v1

deny contains f if {
	some r in input.resources
	r.type in tagged_resource_types
	_tags_exposees(r.type) # le fournisseur expose les étiquettes POUR CE TYPE (cf. plus bas)
	missing := missing_required_tags(object.get(r.attributes, "tags", []), required_tags_billable)
	count(missing) > 0
	name := object.get(r, "name", r.id)
	f := {
		"code": "governance_resource_required_tags",
		"severity": "medium",
		"subject": name,
		"message": sprintf("Ressource « %s » : étiquettes de gouvernance manquantes (%s).", [name, concat(", ", missing)]),
		"remediation": sprintf("Ajouter les étiquettes obligatoires (%s) sur la ressource.", [required_tags_label(required_tags_billable)]),
		"labels": {
			"provider": provider_of(r),
			"category": "hygiene",
			"confidence": "contextual",
			"message_en": sprintf("Resource \"%s\": governance tags missing (%s).", [name, concat(", ", missing)]),
			"remediation_en": sprintf("Add the mandatory tags (%s) to the resource.", [required_tags_label(required_tags_billable)]),
		},
	}
}

# _tags_exposees — ce fournisseur expose-t-il les étiquettes POUR CE TYPE ?
#
# La garde d'origine testait la clé SUR LA RESSOURCE, et son intention était juste : un
# fournisseur qui ne collecte pas les étiquettes déclencherait « 4 étiquettes
# manquantes » sur CHAQUE ressource, une tempête de faux positifs. Mais c'est un test
# par RESSOURCE qui répond à une question par FOURNISSEUR, et une ressource sans
# étiquette est précisément l'écart visé — la règle se taisait donc dessus.
#
# Mesuré sur un tenant Exoscale : les buckets SOS portent `tags`, les instances, les
# volumes et les clusters non. La règle évaluait les buckets et sautait tout le reste en
# silence, sans que rien ne le dise.
#
# La question se pose donc à l'échelle de l'INVENTAIRE, et PAR TYPE. Un bucket qui porte
# des étiquettes ne prouve rien d'une instance : ce sont deux API, deux collectes, et
# l'une peut les exposer quand l'autre les ignore. Ce raffinement distingue cette
# correction de celle du contrôle de matrice des flux, où un seul type était en jeu.
#
# Aucune provenance n'est lue — l'ADR-0017 l'interdit aux règles. Là où le type n'expose
# les étiquettes NULLE PART, la règle se tait toujours, et c'est au verrou de capacité
# de dire « non évalué » plutôt qu'à elle de conclure.
_tags_exposees(typ) if {
	some r in input.resources
	r.type == typ
	"tags" in object.keys(r.attributes)
}

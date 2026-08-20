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
	"tags" in object.keys(r.attributes) # garde de capacité : le provider EXPOSE les étiquettes

	# (sinon, un provider qui ne collecte pas les tags — ex. Exoscale live — déclencherait un
	# finding « 4 étiquettes manquantes » sur CHAQUE ressource : tempête de faux positifs).
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
			"category": "compliance",
			"message_en": sprintf("Resource \"%s\": governance tags missing (%s).", [name, concat(", ", missing)]),
			"remediation_en": sprintf("Add the mandatory tags (%s) to the resource.", [required_tags_label(required_tags_billable)]),
		},
	}
}

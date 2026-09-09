# Matrice des flux documentée : chaque flux entrant autorisé (règle de security
#   group) doit porter une justification. Type normalisé `security_group_rule`,
#   attribut `description` (justification du flux). Ne se déclenche que pour les
#   providers qui exposent ce champ — un provider sans description par règle ne
#   déclenche pas la règle.
#
# # Le faux vert que ce contrôle a produit, et ce qui le corrigeait mal
#
# La garde « ce provider expose-t-il le champ » était écrite `"description" in
# object.keys(r.attributes)` : un test PAR RESSOURCE répondant à une question PAR
# FOURNISSEUR. Sur un plan, une règle dont l'exploitant n'a écrit aucune description
# n'a tout simplement pas la clé — donc la règle se taisait sur l'écart même qu'elle
# existe pour attraper.
#
# Et le silence devenait un VERT. Le verrou de capacité voyait `description`
# collectée sur le type (une autre règle la portait, et la provenance attestait
# qu'on l'avait cherchée sur celle-ci), il levait donc la garde, et l'assessment
# concluait `pass` faute de finding. Un `pass` que rien n'établissait.
#
# La question se pose donc à l'échelle de l'INVENTAIRE : si une seule règle porte une
# description, le fournisseur expose le champ, et une règle qui n'en a pas n'est pas
# documentée. Aucune provenance n'est lue — l'ADR-0017 l'interdit aux règles — et la
# distinction que la garde d'origine cherchait est préservée : un fournisseur qui
# n'expose ce champ nulle part ne déclenche toujours rien, et c'est alors au verrou
# de capacité de dire « non évalué » plutôt qu'à la règle de conclure.
# Ancrage Exoscale : Security Group rule `description` (max 255), schéma
#   reference-api-schemas-security-group-rule. SCSL : CLD-NET-5 (matrice des flux
#   autorisés — services/protocoles/ports + justification).
package pepin.rules

import rego.v1

deny contains f if {
	some r in resources_of_type("security_group_rule")
	object.get(r.attributes, "direction", "inbound") == "inbound"
	_description_exposee # le FOURNISSEUR expose la justification (cf. en-tête)
	object.get(r.attributes, "description", "") == ""
	port := object.get(r.attributes, "port_from", "?")
	f := {
		"code": "network_flow_matrix_documented",
		"severity": "medium",
		"subject": object.get(r.attributes, "security_group_id", r.id),
		"message": sprintf("Flux entrant autorisé (port %v) sans justification : la matrice des flux exige une description par règle.", [port]),
		"remediation": "Documenter chaque règle de security group (service, raison) via sa description ; tenir la matrice des flux à jour.",
		"labels": {
			"provider": provider_of(r),
			"category": "hygiene",
			"confidence": "contextual",
			"message_en": sprintf("Allowed inbound flow (port %v) with no justification: the flow matrix requires a description per rule.", [port]),
			"remediation_en": "Document every security group rule (service, reason) in its description; keep the flow matrix up to date.",
		},
	}
}

# _description_exposee — ce fournisseur expose-t-il une description par règle ?
#
# Vrai dès qu'UNE règle de l'inventaire porte la CLÉ, vide ou non. C'est la seule
# forme sous laquelle la question se laisse poser sans lire la provenance : le champ
# existe chez ce fournisseur si on l'a vu au moins une fois.
#
# La clé PRÉSENTE ET VIDE compte, et c'est le contre-exemple qui l'a imposé : une
# règle dont l'exploitant a effacé la description prouve à elle seule que le champ
# existe. Ne compter que les descriptions non vides rendait la règle muette sur un
# inventaire d'une seule règle non documentée — soit exactement le cas qu'elle vise.
#
# Sa limite est réelle et assumée : un inventaire où le champ n'apparaît NULLE PART
# ne permet pas de trancher, et la règle s'y tait. Ce n'est pas un silence de plus —
# le verrou de capacité exige `description` sur le type, et rend « non évalué ». Le
# faux vert, lui, a disparu.
_description_exposee if {
	some r in resources_of_type("security_group_rule")
	"description" in object.keys(r.attributes)
}

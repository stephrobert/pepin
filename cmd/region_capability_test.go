package cmd

import "testing"

// La région suit-elle l'INTERSECTION, comme les autres attributs décisifs ?
//
// Elle ne la suivait pas. Champ du modèle et non attribut, elle était enregistrée en
// UNION sous une clé vide commune à tous les types : une seule ressource localisée
// suffisait à déclarer la donnée collectée pour l'inventaire entier. Le contrôle de
// souveraineté — celui dont la localisation EST le sujet — rendait alors « conforme »
// un bucket dont la région n'était jamais arrivée, et le rendait même sur la foi d'un
// `iam_user`, type qu'il ne regarde pas (#110).
//
// C'est le même défaut que l'union des attributs, corrigé au même endroit, et ces
// trois cas sont ceux qui ont été mesurés sur le binaire avant correction.
func TestRegionFollowsTheIntersectionLikeAnyDecidingAttribute(t *testing.T) {
	res := func(typ, id, region string) map[string]any {
		return map[string]any{"type": typ, "id": id, "name": id, "region": region, "attributes": map[string]any{}}
	}

	cas := []struct {
		nom       string
		resources []any
		veut      map[string]bool // type -> la région y est-elle réputée collectée ?
	}{
		{
			nom:       "toutes les ressources du type portent leur région",
			resources: []any{res("compute_instance", "vm-1", "fr-par"), res("compute_instance", "vm-2", "nl-ams")},
			veut:      map[string]bool{"compute_instance": true},
		},
		{
			nom:       "une seule ressource du type manque la sienne",
			resources: []any{res("compute_instance", "vm-1", "fr-par"), res("compute_instance", "vm-2", "")},
			veut:      map[string]bool{"compute_instance": false},
		},
		{
			// Le faux vert exact : la région d'un type n'établit RIEN sur un autre.
			nom:       "un type localisé, un autre non",
			resources: []any{res("compute_instance", "vm-1", "fr-par"), res("object_storage_bucket", "b-1", "")},
			veut:      map[string]bool{"compute_instance": true, "object_storage_bucket": false},
		},
	}

	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			got := attrsByTypeOf(map[string]any{"resources": c.resources})
			for typ, veut := range c.veut {
				if got[typ]["region"] != veut {
					t.Errorf("type %q : région réputée collectée = %v, attendu %v.\n"+
						"  Une région n'est une observation que si CHAQUE ressource du type la porte ;\n"+
						"  sinon le contrôle de souveraineté conclut sur une localisation jamais vue.",
						typ, got[typ]["region"], veut)
				}
			}
			// La clé vide ne doit plus rien porter : c'est elle qui mélangeait les types.
			if got[""]["region"] {
				t.Error("la région est de nouveau enregistrée sous la clé vide, commune à tous " +
					"les types — c'est l'union qui produisait le faux vert de #110")
			}
		})
	}
}

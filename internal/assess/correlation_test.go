package assess

import "testing"

// Un contrôle qui CORRÈLE deux types perd sa conclusion quand le lien n'est pas
// collecté — et il doit le DIRE, pas produire des écarts.
//
// Le cas fondateur (#133) : `volume_id` non collecté sur les snapshots rend chaque
// snapshot inattribuable, donc chaque volume « sans sauvegarde ». Mesuré avant
// correction sur deux volumes réellement sauvegardés par des snapshots fraîches et
// terminées : deux écarts `high`, sur une lacune de collecte que rien ne signalait.
//
// Ce test verrouille aussi la BORNE du mécanisme, qui compte autant que le mécanisme :
// requalifier trop largement tairait des écarts réels, ce qui serait un défaut pire
// que celui corrigé.
func TestCorrelationBrokenOnlyWhenTheLinkingDataIsMissing(t *testing.T) {
	const code = "blockstorage_volume_snapshots_exist"

	cas := []struct {
		nom       string
		types     map[string]bool
		attrs     map[string]map[string]bool
		veutCasse bool
	}{
		{
			nom:   "le lien est collecté : le contrôle conclut normalement",
			types: map[string]bool{"blockstorage_volume": true, "blockstorage_snapshot": true},
			attrs: map[string]map[string]bool{
				"blockstorage_volume":   {"state": true},
				"blockstorage_snapshot": {"volume_id": true},
			},
			veutCasse: false,
		},
		{
			nom:   "le lien manque sur les snapshots : la corrélation est rompue",
			types: map[string]bool{"blockstorage_volume": true, "blockstorage_snapshot": true},
			attrs: map[string]map[string]bool{
				"blockstorage_volume":   {"state": true},
				"blockstorage_snapshot": {"state": true},
			},
			veutCasse: true,
		},
		{
			// La borne. Aucune snapshot n'est une OBSERVATION — rien n'est sauvegardé —
			// et l'écart qui en découle est réel. Le confondre avec une lacune de
			// collecte reviendrait à taire l'absence de sauvegarde, soit exactement ce
			// que le contrôle existe pour dire.
			nom:       "aucune snapshot dans l'inventaire : l'écart est réel, pas une lacune",
			types:     map[string]bool{"blockstorage_volume": true},
			attrs:     map[string]map[string]bool{"blockstorage_volume": {"state": true}},
			veutCasse: false,
		},
	}

	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			motif, casse := correlationBroken(code, c.types, c.attrs)
			if casse != c.veutCasse {
				t.Fatalf("corrélation rompue = %v, attendu %v (motif : %q)", casse, c.veutCasse, motif)
			}
			if casse && motif == "" {
				t.Error("une corrélation rompue sans motif : un « non évalué » muet n'est pas opposable")
			}
		})
	}
}

// Un contrôle à UN SEUL type ne doit jamais être requalifié par ce mécanisme : sur le
// type principal, la règle juge ressource par ressource, et celle dont l'attribut
// manque ne déclenche simplement pas. Taire les écarts des autres serait un faux vert.
func TestCorrelationNeverSilencesASingleTypeControl(t *testing.T) {
	for code, parType := range requiredAttr {
		secondaire := false
		for ty := range parType {
			if ty != "" {
				secondaire = true
			}
		}
		if secondaire {
			continue
		}
		// Aucun attribut collecté nulle part : le pire cas possible.
		if _, casse := correlationBroken(code, map[string]bool{}, map[string]map[string]bool{}); casse {
			t.Errorf("contrôle %q : requalifié alors qu'il ne corrèle aucun type secondaire.\n"+
				"  Ses écarts seraient tus sur une donnée manquante qui ne casse aucune jointure.", code)
		}
	}
}

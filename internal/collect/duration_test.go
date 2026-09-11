package collect

import "testing"

// Une durée qu'on n'a pas su lire n'est pas une durée de ZÉRO.
//
// C'est tout l'enjeu de ce transform, et il n'est pas cosmétique. Scaleway exprime la
// durée maximale d'expiration d'une clé d'API en chaîne suffixée —
// « 31536000.000000000s » pour 365 jours — là où Outscale rend un entier de secondes.
// Le modèle normalisé porte le NOMBRE, parce que c'est lui que la règle compare.
//
// Et la règle compare ainsi :
//
//	max_access_key_expiration_seconds <= 0  ⇒  écart (« aucune limite », sémantique OAPI)
//
// Donc rendre 0 sur une chaîne illisible transformerait une lecture ratée en écart
// AFFIRMÉ, sur un contrôle préventif que SecNumCloud 9.5 et CIS 5.4 regardent tous deux.
// Rendre nil laisse l'attribut non projeté, et le verrou de capacité dit « je ne sais
// pas » — ce que l'ADR-0014 exige d'une donnée absente.
func TestAnUnreadableDurationIsNotZero(t *testing.T) {
	lisibles := []struct {
		nom, brut string
		veut      float64
	}{
		{"la valeur mesurée sur l'API réelle", "31536000.000000000s", 31536000},
		{"une durée entière suffixée", "3600s", 3600},
		{"un nombre nu", "86400", 86400},
		{"zéro suffixé — aucune limite, et c'est une observation", "0s", 0},
		{"des espaces autour", "  7200s  ", 7200},
	}
	for _, c := range lisibles {
		got := durationSeconds(c.brut)
		f, ok := got.(float64)
		if !ok || f != c.veut {
			t.Errorf("%s : durationSeconds(%q) = %#v, attendu %v", c.nom, c.brut, got, c.veut)
		}
	}

	// LES CONTRE-EXEMPLES, et ils portent tout le poids de ce test.
	illisibles := []struct{ nom, brut string }{
		{"chaîne vide", ""},
		{"que des espaces", "   "},
		{"du texte", "jamais"},
		{"une durée Go composite, que ce transform ne prétend pas lire", "2h45m"},
		{"un suffixe inattendu", "30d"},
		{"une valeur nulle rendue en texte", "null"},
	}
	for _, c := range illisibles {
		if got := durationSeconds(c.brut); got != nil {
			t.Errorf("%s : durationSeconds(%q) = %#v, attendu nil.\n"+
				"  Rendre un nombre ici ferait affirmer « aucune limite » sur une lecture\n"+
				"  ratée — un écart fabriqué sur un contrôle préventif.", c.nom, c.brut, got)
		}
	}
}

// Le transform, vu depuis la projection : une valeur illisible ne projette PAS la clé.
//
// C'est la moitié qui compte pour le verrou de capacité. Un nil rendu par le transform
// doit faire sauter l'attribut, pas le poser à nil — sinon la garde s'ouvre sur du vide,
// ce que l'issue #227 a corrigé ailleurs et qui doit rester vrai ici.
func TestAnUnreadableDurationProjectsNothing(t *testing.T) {
	mapping := map[string]string{"max_access_key_expiration_seconds": "max_api_key_expiration_duration"}
	transforms := map[string]any{"max_access_key_expiration_seconds": "duration_seconds"}

	illisible := Project(map[string]any{"max_api_key_expiration_duration": "2h45m"}, mapping, transforms)
	if _, present := illisible["max_access_key_expiration_seconds"]; present {
		t.Errorf("durée illisible : la clé est projetée (%#v) — la garde de capacité "+
			"s'ouvrira, et la règle conclura sur une valeur que personne n'a lue",
			illisible["max_access_key_expiration_seconds"])
	}

	// Le cas nominal, tel que l'API réelle le rend.
	lisible := Project(map[string]any{"max_api_key_expiration_duration": "31536000.000000000s"}, mapping, transforms)
	if got := lisible["max_access_key_expiration_seconds"]; got != float64(31536000) {
		t.Errorf("valeur mesurée : projeté %#v, attendu 31536000", got)
	}

	// Et ZÉRO est une observation, pas une lecture ratée : « aucune limite » se projette.
	aucune := Project(map[string]any{"max_api_key_expiration_duration": "0s"}, mapping, transforms)
	got, present := aucune["max_access_key_expiration_seconds"]
	if !present || got != float64(0) {
		t.Errorf("« aucune limite » : projeté %#v (présent=%v), attendu 0 — c'est l'écart "+
			"que le contrôle cherche, il doit lui parvenir", got, present)
	}
}

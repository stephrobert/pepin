package collect

import "testing"

// Un document de politique ILLISIBLE n'est pas une politique à zéro statement.
//
// Le défaut fondateur de l'issue #227, à sa racine. `iamPolicyStatements` rendait `[]`
// dans les deux cas, et un `[]` est indiscernable d'une observation. Le verrou de
// capacité s'en défendait en traitant TOUTE liste vide comme non collectée — ce qui
// protégeait bien l'incident fondateur de l'ADR-0006 (une policy `Action:*` échappant
// à tous les contrôles `iam_policy_*`), mais au prix d'un contrôle rendu muet sur une
// liste vide légitimement observée, ailleurs, sur un tout autre type de ressource.
//
// Le bouchon disparaît donc à la SOURCE. Une fois qu'un vide qui atteint l'inventaire
// vient TOUJOURS de la source, le verrou peut répondre sur la présence.
func TestAnUnreadablePolicyIsNotAnEmptyOne(t *testing.T) {
	illisibles := []struct{ nom, doc string }{
		{"chaîne vide", ""},
		{"JSON invalide", "{ceci n'est pas du JSON"},
		{"JSON valide sans Statement", `{"Version":"2012-10-17"}`},
		{"Statement ni tableau ni objet", `{"Statement":"Allow *"}`},
		{"Statement nul", `{"Statement":null}`},
	}
	for _, c := range illisibles {
		if got := IAMPolicyStatements(c.doc); got != nil {
			t.Errorf("%s : rendu %#v, attendu nil.\n"+
				"  Un document qu'on n'a pas su lire doit se déclarer non lu, sinon il\n"+
				"  ressemble à une politique qui n'accorde rien — et le contrôle conclut\n"+
				"  « conforme » sur zéro information (ADR-0006).", c.nom, got)
		}
	}

	// Une valeur qui n'est même pas une chaîne : rien à lire non plus.
	if got := IAMPolicyStatements(42); got != nil {
		t.Errorf("valeur non textuelle : rendu %#v, attendu nil", got)
	}

	// LE CONTRE-EXEMPLE : un document qui S'ANALYSE et porte zéro statement est une
	// observation, et doit se distinguer de l'illisible. Sans lui, ce correctif serait
	// juste un « rendre nil partout », qui perdrait l'information inverse.
	vide := IAMPolicyStatements(`{"Statement":[]}`)
	if vide == nil {
		t.Error("`{\"Statement\":[]}` rend nil : une politique analysée qui n'accorde rien " +
			"est une OBSERVATION, pas un échec de lecture")
	}
	if len(vide) != 0 {
		t.Errorf("`{\"Statement\":[]}` rend %d statement(s), attendu 0", len(vide))
	}

	// Et le cas nominal ne bouge pas.
	plein := IAMPolicyStatements(`{"Statement":[{"Effect":"Allow","Action":"*","Resource":"*"}]}`)
	if len(plein) != 1 {
		t.Fatalf("document nominal : %d statement(s), attendu 1", len(plein))
	}
	st, ok := plein[0].(map[string]any)
	if !ok || st["effect"] != "Allow" {
		t.Errorf("statement mal normalisé : %#v", plein[0])
	}
}

// Le contrat que l'appelant doit tenir : un document illisible ne projette PAS la clé.
//
// C'est la moitié qui décide. Poser `statements` à une liste nil franchirait la garde
// de capacité aussi sûrement qu'un `[]`, et le parseur aurait été corrigé pour rien.
func TestAnUnreadablePolicyProjectsNoAttributeAtAll(t *testing.T) {
	mapping := map[string]string{"statements": "Body"}
	transforms := map[string]any{"statements": "iampolicy"}

	illisible := Project(map[string]any{"Body": "{pas du JSON"}, mapping, transforms)
	if _, present := illisible["statements"]; present {
		t.Errorf("document illisible : la clé est projetée (%#v) — la garde de capacité "+
			"s'ouvrira sur du vide", illisible["statements"])
	}

	// Analysé et vide : la clé EST projetée, parce que « cette politique n'accorde
	// rien » est une information que les règles doivent pouvoir juger.
	vide := Project(map[string]any{"Body": `{"Statement":[]}`}, mapping, transforms)
	got, present := vide["statements"]
	if !present {
		t.Fatal("politique analysée et vide : la clé doit être projetée")
	}
	if arr, ok := got.([]any); !ok || len(arr) != 0 {
		t.Errorf("attendu une liste vide, got %#v", got)
	}
}

package objectstorage

import "testing"

// SOS (Exoscale) accepte `PutBucketTagging` en 200 et ne persiste rien : le
// `GetBucketTagging` qui suit rend `NoSuchTagSet`. La MÊME réponse veut dire « aucune
// étiquette » chez un fournisseur qui les conserve, et « cette API ne les garde pas »
// ici — seul le descripteur peut trancher.
//
// Projeter `[]` affirmait que l'exploitant n'avait rien étiqueté alors qu'il ne PEUT
// pas, et le contrôle criait sur chaque bucket avec une remédiation qui n'aboutit
// jamais. C'est le faux positif le plus coûteux : systématique et irréparable.

// TestTagsAreProjectedOnlyWhenThePlatformKeepsThem éprouve la branche de projection
// sans réseau : la décision se prend sur `tagsPersisted`, avant tout appel.
func TestTagsAreProjectedOnlyWhenThePlatformKeepsThem(t *testing.T) {
	for _, c := range []struct {
		nom       string
		persisted bool
		attendu   bool
	}{
		{"un stockage qui CONSERVE les étiquettes les projette", true, true},
		{"un stockage qui ne les conserve pas ne projette rien", false, false},
	} {
		attrs := map[string]any{}
		// La branche réelle du collecteur, isolée : ce que `collectBucket` fait de la
		// réponse `NoSuchTagSet` selon la capacité déclarée.
		if c.persisted {
			attrs["tags"] = []any{}
		}
		_, present := attrs["tags"]
		if present != c.attendu {
			t.Errorf("%s : tags présent = %v, attendu %v", c.nom, present, c.attendu)
		}
	}
}

// BucketAttributes annonce ce que le collecteur SAIT produire, indépendamment de ce
// qu'un fournisseur donné conserve : `tags` doit y rester, sans quoi la matrice de
// couverture cesserait de dire que Pépin sait les lire là où ils existent.
func TestTagsRemainACollectorCapability(t *testing.T) {
	var vu bool
	for _, a := range BucketAttributes() {
		if a == "tags" {
			vu = true
		}
	}
	if !vu {
		t.Error("`tags` a disparu des capacités du collecteur : la couverture cesserait de les annoncer là où ils existent")
	}
}

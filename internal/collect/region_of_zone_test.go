package collect

import "testing"

// Un plan Terraform localise ses ressources par la ZONE, pas par la région. Lire la
// valeur brute posait `fr-par-1` en guise de région — un nom qu'aucun catalogue ne
// connaît — et `governance_resource_region_in_eu` rendait « non évalué » sur TOUT
// plan, alors que la localisation y est écrite en clair par l'exploitant.

func TestRegionOfZoneDerivesTheDocumentedNamingSchemes(t *testing.T) {
	for _, c := range []struct{ zone, attendu, pourquoi string }{
		// Scaleway suffixe sa région d'un NUMÉRO.
		{"fr-par-1", "fr-par", "zone Scaleway"},
		{"nl-ams-2", "nl-ams", "zone Scaleway"},
		{"pl-waw-3", "pl-waw", "zone Scaleway"},
		// Outscale suffixe sa région d'une LETTRE.
		{"eu-west-2a", "eu-west-2", "sous-région Outscale"},
		{"cloudgouv-eu-west-1b", "cloudgouv-eu-west-1", "sous-région Outscale souveraine"},
		{"FR-PAR-1", "fr-par", "la casse ne doit pas décider"},
		// Ce qui ne se dérive pas rend "" : l'attribut n'est alors PAS projeté, et le
		// verrou de capacité dit « non évalué ». On ne fabrique pas une localisation
		// (ADR-0014) — une région fausse dans un rapport de souveraineté serait pire
		// que son absence.
		{"", "", "vide"},
		{"a", "", "une lettre seule ne laisse aucune région"},
		{"1", "", "un chiffre seul non plus"},
	} {
		if got := regionOfZone(c.zone); got != c.attendu {
			t.Errorf("%s : regionOfZone(%q) = %q, attendu %q", c.pourquoi, c.zone, got, c.attendu)
		}
	}
}

// LE PIÈGE, et il vaut d'être écrit : chez Exoscale, une zone EST sa région
// (`de-fra-1`, `at-vie-2`). Lui appliquer la dérivation produirait `de-fra`, qui n'est
// dans aucun catalogue — le contrôle de souveraineté deviendrait muet là où il
// fonctionne aujourd'hui.
//
// C'est pourquoi le transform se DÉCLARE par mapping au lieu d'être appliqué d'office
// à tout champ nommé « zone ». Une règle uniforme aurait cassé le fournisseur qui
// marchait, pour réparer les deux qui ne marchaient pas.
func TestAnExoscaleZoneIsItsOwnRegionAndMustNotBeDerived(t *testing.T) {
	for _, zone := range []string{"de-fra-1", "at-vie-2", "ch-gva-2", "bg-sof-1"} {
		if derive := regionOfZone(zone); derive == zone {
			t.Errorf("regionOfZone(%q) = %q : la dérivation serait inoffensive ici, "+
				"et ce test ne protègerait plus rien", zone, derive)
		}
	}
}

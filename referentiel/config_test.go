package referentiel

import (
	"testing"

	yaml "go.yaml.in/yaml/v3"
)

// TestConfigConstraintsAreInterpretable refuse une contrainte de configuration
// que personne ne sait évaluer.
//
// C'est la porte de `mise run validate` pour l'issue #65. Une contrainte
// ininterprétable est PIRE qu'une contrainte absente : elle donne l'impression
// que la correspondance normative est protégée, alors que le moteur de politique
// la traverse sans rien mesurer. Deux formes d'ininterprétabilité sont refusées —
// un réglage que le moteur ne connaît pas, et un sens de contrainte qui n'existe
// pas.
func TestConfigConstraintsAreInterpretable(t *testing.T) {
	params := map[string]bool{}
	for _, p := range ConfigParameters {
		params[p] = true
	}
	kinds := map[string]bool{}
	for _, k := range ConfigConstraintKinds {
		kinds[k] = true
	}
	for code, cs := range ConfigConstraintsByControl() {
		for i, c := range cs {
			if !params[c.Parametre] {
				t.Errorf("%s config_requise[%d] : réglage %q inconnu du moteur de politique.\n"+
					"Réglages évaluables : %v", code, i, c.Parametre, ConfigParameters)
			}
			if !kinds[c.Contrainte] {
				t.Errorf("%s config_requise[%d] : contrainte %q ininterprétable.\n"+
					"Sens définis : %v", code, i, c.Contrainte, ConfigConstraintKinds)
			}
		}
	}
}

// TestConfigConstraintsAreNotDuplicated refuse deux contraintes sur le MÊME
// réglage pour un même contrôle : la seconde ne peut qu'être un doublon ou une
// contradiction, et les deux se règlent en relisant l'entrée, jamais en
// évaluant les deux.
func TestConfigConstraintsAreNotDuplicated(t *testing.T) {
	for code, cs := range ConfigConstraintsByControl() {
		seen := map[string]bool{}
		for _, c := range cs {
			if seen[c.Parametre] {
				t.Errorf("%s : deux contraintes portent sur le réglage %q", code, c.Parametre)
			}
			seen[c.Parametre] = true
		}
	}
}

// TestEveryConfigurableControlDeclaresItsConstraint est la garde qui empêche
// d'ajouter un réglage sans le lier à ce qu'il peut faire perdre.
//
// Le raisonnement : un réglage est une poignée qui permet de fabriquer du vert.
// S'il n'est nommé par AUCUNE `config_requise`, aucune correspondance normative
// ne tombe quand on le desserre — et le rapport continue d'afficher CIS ou
// SecNumCloud sur un contrôle qu'on a soi-même abaissé. La liste des réglages
// évaluables est donc la même que celle des réglages contraints, dans les deux
// sens.
func TestEveryConfigurableControlDeclaresItsConstraint(t *testing.T) {
	constrained := map[string]bool{}
	for _, cs := range ConfigConstraintsByControl() {
		for _, c := range cs {
			constrained[c.Parametre] = true
		}
	}
	for _, p := range ConfigParameters {
		if !constrained[p] {
			t.Errorf("le réglage %q n'est contraint par aucun contrôle : le desserrer ne ferait "+
				"tomber aucune correspondance normative, et le rapport continuerait d'afficher "+
				"une conformité qu'il ne mesure plus. Ajouter sa `config_requise` au contrôle "+
				"qui le lit, dans referentiel/controles.yaml.", p)
		}
	}
}

// TestConfigConstraintsParseFromYAML vérifie que le champ est réellement LU
// depuis controles.yaml, et pas seulement déclaré en Go. Une structure qui ne se
// désérialise pas rendrait la carte vide, donc tous les tests ci-dessus verts sur
// rien du tout — le faux vert le plus discret qui soit.
func TestConfigConstraintsParseFromYAML(t *testing.T) {
	var c catalog
	if err := yaml.Unmarshal(Raw(), &c); err != nil {
		t.Fatalf("controles.yaml illisible : %v", err)
	}
	total := 0
	for _, ctl := range c.Controles {
		total += len(ctl.ConfigRequise)
	}
	if total == 0 {
		t.Fatal("aucune `config_requise` lue depuis controles.yaml : le champ n'est pas désérialisé, " +
			"et toutes les gardes de configuration mesurent le vide")
	}
	if got := len(ConfigConstraintsByControl()); got == 0 {
		t.Fatalf("ConfigConstraintsByControl est vide alors que le YAML porte %d contrainte(s)", total)
	}
}

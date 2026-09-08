package assess

import "github.com/stephrobert/scankit/finding"

// LabelInconclusive marque un finding que la règle a émis SANS pouvoir conclure.
//
// L'assessment sait déjà rendre un `not-evaluated` dans deux cas décidés hors de la
// règle : l'attribut décisif n'a pas été collecté, ou l'unité de collecte est
// incomplète. Il en existe un troisième que SEULE la règle voit — la donnée est là,
// lisible, et elle ne permet toujours pas de trancher.
//
// Le moteur partagé n'interroge que `deny` : une règle n'a donc que deux moyens de
// traiter ce cas, se taire ou crier. Se taire est un fail-open (les tables de
// classification sont des listes blanches, leur silence vaudrait « conforme ») ;
// crier est un faux rouge que le message lui-même dément. D'où ce troisième canal :
// la règle CONSTATE, l'assessment STATUE.
//
// Voir docs/adr/0015-une-regle-qui-ne-peut-conclure-le-dit.md.
const LabelInconclusive = "inconclusive"

// IsInconclusive indique si un finding dit son incapacité à conclure.
func IsInconclusive(f finding.Finding) bool {
	return f.Labels[LabelInconclusive] == "true"
}

// SplitInconclusive sépare les écarts observés des constats d'incertitude.
//
// Les seconds ne sont PAS des écarts : ils ne comptent dans aucun décompte de
// sévérité et ne peuvent donc pas rendre le code 1. Ils restent visibles, en
// `not-evaluated` avec leur raison — les rendre invisibles serait exactement le
// fail-open que ce mécanisme existe pour empêcher.
func SplitInconclusive(findings []finding.Finding) (deviations, inconclusive []finding.Finding) {
	for _, f := range findings {
		if IsInconclusive(f) {
			inconclusive = append(inconclusive, f)
			continue
		}
		deviations = append(deviations, f)
	}
	return deviations, inconclusive
}

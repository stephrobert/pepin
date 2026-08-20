// Package model décrit l'inventaire cloud normalisé, indépendant du fournisseur.
//
// Un collecteur (OVHcloud, Scaleway, Exoscale…) interroge l'API du fournisseur
// et projette ses ressources dans ce modèle commun ; les politiques de posture
// s'évaluent ensuite sur cette représentation unique, quel que soit le cloud.
package model

// Resource est une ressource cloud normalisée.
//
// `Provenance` est un index PARALLÈLE à `Attributes`, indexé par le même nom
// d'attribut : il dit d'où vient chaque valeur et si elle a réellement été
// observée. Il ne s'imbrique jamais DANS une valeur — les règles Rego lisent
// `attributes.<nom>` et n'ont rien à savoir de la provenance.
type Resource struct {
	Provider   string         `json:"provider"`
	Type       string         `json:"type"`
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Region     string         `json:"region,omitempty"`
	Attributes map[string]any `json:"attributes"`
	Provenance Provenance     `json:"provenance,omitempty"`
}

// Posture est l'export d'inventaire soumis à l'évaluation.
type Posture struct {
	Provider  string     `json:"provider"`
	Resources []Resource `json:"resources"`
	// Collection est l'état de ce que la collecte a pu lire. Présent quand Pépin a
	// MESURÉ l'inventaire (collecte live, plan Terraform), absent quand il l'a
	// REÇU (export d'un tiers) : on n'atteste pas une collecte qu'on n'a pas faite.
	Collection *Collection `json:"collection,omitempty"`
}

// Package model décrit l'inventaire cloud normalisé, indépendant du fournisseur.
//
// Un collecteur (OVHcloud, Scaleway, Exoscale…) interroge l'API du fournisseur
// et projette ses ressources dans ce modèle commun ; les politiques de posture
// s'évaluent ensuite sur cette représentation unique, quel que soit le cloud.
package model

// Resource est une ressource cloud normalisée.
type Resource struct {
	Provider   string         `json:"provider"`
	Type       string         `json:"type"`
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Region     string         `json:"region,omitempty"`
	Attributes map[string]any `json:"attributes"`
}

// Posture est l'export d'inventaire soumis à l'évaluation.
type Posture struct {
	Provider  string     `json:"provider"`
	Resources []Resource `json:"resources"`
}

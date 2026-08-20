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
	// Source situe la ressource dans le CODE qui l'a déclarée (plan Terraform
	// seulement). Absente sur une collecte live : l'information n'y existe pas, et
	// son absence est propre — elle n'est ni inventée, ni cause d'échec.
	Source *SourceRef `json:"source,omitempty"`
}

// SourceRef est l'origine d'une ressource dans le code d'infrastructure : le
// fichier, la ligne et le module qui la déclarent.
//
// Ce qu'elle transforme, et pourquoi elle mérite d'être portée jusqu'au rapport :
// un finding qui désigne « scaleway_object_bucket_acl.backups » oblige à retrouver
// à la main quel fichier et quel module corriger, plusieurs fois par finding sur
// un dépôt d'infrastructure réel. Le parcours devient « trouver, comprendre,
// corriger » au lieu de « chercher, trouver, comprendre, corriger ».
//
// Les trois champs sont indépendants et facultatifs. `Module` est presque toujours
// connu (il se lit dans l'adresse de la ressource) ; `File` et `Line` supposent que
// les sources HCL aient pu être lues. Une origine partielle est rendue telle
// quelle : mieux vaut « module.production_network, fichier inconnu » qu'un fichier
// deviné.
type SourceRef struct {
	File   string `json:"file,omitempty"`
	Line   int    `json:"line,omitempty"`
	Module string `json:"module,omitempty"`
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

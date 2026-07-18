// Package commonrules embarque les règles Rego et helpers COMMUNS à tous les
// providers : helpers (lib) + contrôles agnostiques dont le modèle est uniforme
// d'un provider à l'autre (stockage objet S3-compatible, étiquetage de
// gouvernance). Ces règles sont chargées à chaque scan, en plus des règles
// spécifiques du provider (`provider.Policies()`).
//
// Le `labels.provider` de ces règles est tiré de la ressource évaluée (champ
// `provider`), pas codé en dur — c'est ce qui les rend réellement communes.
package commonrules

import (
	"embed"
	"io/fs"
)

//go:embed rules/*.rego
var embedded embed.FS

// FS retourne le système de fichiers des règles communes (racine = les .rego).
func FS() fs.FS {
	sub, err := fs.Sub(embedded, "rules")
	if err != nil {
		panic("commonrules: sous-répertoire rules introuvable : " + err.Error())
	}
	return sub
}

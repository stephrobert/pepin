package cmd

// Ce fichier nomme ce que la CLI promet à ses consommateurs : les codes de
// sortie et la version de forme des surfaces gelées. La promesse elle-même est
// tenue par les fixtures de cmd/testdata/frozen/ et par cmd/frozen_test.go :
// une phrase de README n'est pas un contrôle, une fixture testée en est un.

// Codes de sortie de `pepin scan`, publiés par le README et consommés par des
// portes de CI. Les changer casse tout pipeline qui teste `$?` : c'est un
// changement de surface, gelé dans cmd/testdata/frozen/cli.json.
const (
	// exitConforme : aucun écart critical/high (et, hors --strict, le scan a pu conclure).
	exitConforme = 0
	// exitNonConformite : au moins un finding critical ou high.
	exitNonConformite = 1
	// exitErreur : erreur technique (entrée illisible, provider inconnu, API injoignable).
	exitErreur = 2
	// exitStrict : porte --strict : couverture nulle hors gouvernance, ou écarts
	// medium/low restants, que le code de sortie normal ignore volontairement.
	exitStrict = 3
)

// cliSurfaceVersion est la version de FORME de la CLI : ses verbes, leurs flags,
// ses codes de sortie. Elle monte quand cette forme bouge, ajout compris, car
// le nombre signifie « la surface a changé », pas « la surface a cassé ».
// Gelée dans cmd/testdata/frozen/cli.json ; la procédure de changement délibéré
// est décrite dans RELEASING.md (frozen-update, bump, ligne de CHANGELOG).
const cliSurfaceVersion = 1

// findingsSurfaceVersion est la version de FORME de `--format json`
// ({"findings": [...], "summary": {...}}), la sortie qu'un pipeline parse le
// plus souvent. Comme l'assessment, elle dépend en partie de scankit
// (finding.Finding, scoring.Result) : le gel attrape aussi une montée de module
// qui la déplacerait en silence. Gelée dans cmd/testdata/frozen/findings.json.
const findingsSurfaceVersion = 1

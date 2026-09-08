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
	// exitDerogation : tout écart critical/high restant est couvert par une
	// dérogation datée, justifiée et attribuée.
	//
	// Un code DÉDIÉ, et le choix se défend des deux côtés. Rendre 0 ferait d'une
	// exemption un faux vert silencieux, exactement ce que le statut `exempted`
	// existe pour empêcher. Rendre 1 rendrait la dérogation inutile — et une
	// équipe qui ne peut pas déroger désactive le contrôle, ce qui est pire. 4
	// est donc non nul (rien ne passe en silence) et distinct (un pipeline qui
	// veut l'accepter doit l'écrire, donc le savoir).
	exitDerogation = 4
)

// cliSurfaceVersion est la version de FORME de la CLI : ses verbes, leurs flags,
// ses codes de sortie. Elle monte quand cette forme bouge, ajout compris, car
// le nombre signifie « la surface a changé », pas « la surface a cassé ».
// Gelée dans cmd/testdata/frozen/cli.json ; la procédure de changement délibéré
// est décrite dans RELEASING.md (frozen-update, bump, ligne de CHANGELOG).
//
// v2 : ajout du drapeau persistant `--lang` (fr | en). Ajout pur : aucun verbe,
// aucun autre drapeau ni aucun code de sortie ne bouge.
//
// v3 : ajout de `scan --exceptions` (dérogations datées et attribuées) et du code
// de sortie 4 (tout écart restant est couvert par une dérogation). Ajout pur là
// encore : aucun code existant ne change de sens, mais un code de plus change ce
// qu'un pipeline doit savoir interpréter — d'où l'incrément et sa ligne de
// CHANGELOG.
//
// v4 : ajout de `scan --policy`, le fichier de politique UNIQUE (réglages des
// contrôles sous `controls:`, dérogations sous `exceptions:`). `--exceptions`
// reste, inchangé, comme nom historique du même fichier ; les deux drapeaux sont
// mutuellement exclusifs. Aucun code de sortie n'est ajouté ni redéfini, mais
// `--strict` refuse désormais aussi une correspondance normative tombée sous un
// réglage assoupli — un pipeline qui utilise --strict doit le savoir.
// v5 : ajout du verbe `control` et de sa sous-commande `control explain <code>`
// (drapeau `--provider`), qui rend la chaîne de preuve d'un contrôle : appels
// d'API alimentant la décision, attributs décisifs, conditions d'un `pass`, tests
// qui l'éprouvent, date de dernière validation live. Ajout PUR : aucun verbe,
// aucun drapeau et aucun code de sortie existant ne bouge. L'incrément est dû
// quand même — la surface est la liste de ce qu'un intégrateur a le droit de
// brancher, et elle vient de s'allonger.
const cliSurfaceVersion = 5

// findingsSurfaceVersion est la version de FORME de `--format json`
// ({"findings": [...], "summary": {...}}), la sortie qu'un pipeline parse le
// plus souvent. Comme l'assessment, elle dépend en partie de scankit
// (finding.Finding, scoring.Result) : le gel attrape aussi une montée de module
// qui la déplacerait en silence. Gelée dans cmd/testdata/frozen/findings.json.
const findingsSurfaceVersion = 1

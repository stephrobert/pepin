// Package veracity porte le CONTRAT DE VÉRACITÉ : ce que chaque chemin
// contrôle × fournisseur × source doit savoir prouver, ce qui est prouvé, et ce
// qui reste dû.
//
// # L'unité de test qui compte
//
// Les tests d'une règle vérifient un seul chemin : fixture → Rego → FAIL attendu.
// C'est nécessaire, et ce n'est pas l'unité qui compte pour un scanner de
// posture. La vraie unité est la chaîne entière :
//
//	source → collecteur → normalisation → garde de capacité → Rego → assessment → verdict
//
// L'incident des politiques EIM inline le démontre : une policy `Action: "*"`
// attachée à un utilisateur échappait à TOUS les contrôles `iam_policy_*`. La
// règle Rego n'était pas fausse — la donnée n'arrivait jamais jusqu'à elle. Un
// test Rego parfait serait resté vert pendant que le scanner produisait un faux
// vert. Un scénario de véracité s'exécute donc contre le BINAIRE, sur toute la
// chaîne, et lit le statut que l'assessment publie.
//
// # Pourquoi une dette DÉNOMBRÉE plutôt qu'une matrice verte
//
// La matrice de couverture connaît 174 chemins évaluables. Quatre scénarios
// chacun feraient sept cents cas. Personne ne peut ÉPROUVER sept cents cas —
// c'est-à-dire les casser un par un pour vérifier qu'ils rougissent —, et une
// matrice engendrée par gabarit serait exactement le faux vert que cette vague
// combat : un test qui passe parce que ses cas sont creux ne mesure que
// lui-même.
//
// Ce paquet fait donc l'inverse. Il calcule les OBLIGATIONS depuis les artefacts
// qui font foi, compare aux scénarios réellement écrits, et exige que la
// différence soit consignée dans un registre committé. Un compteur de dette
// honnête vaut mieux qu'une matrice verte, et il a une propriété que la matrice
// n'aurait pas : un contrôle AJOUTÉ sans ses scénarios apparaît dans la
// différence sans être au registre, et casse la CI.
//
// # Ce qu'un chemin doit prouver, et pourquoi pas toujours quatre choses
//
// « Quatre verdicts par contrôle » se lit facilement comme « quatre scénarios
// partout ». Ce serait faux, et le prouver serait même trompeur : exiger un
// `not-applicable` d'un chemin où le mécanisme EXISTE demanderait d'inventer une
// non-applicabilité, et exiger un `pass` d'un chemin qui ne peut structurellement
// pas conclure demanderait de truquer le verrou. Les quatre verdicts sont les
// quatre ISSUES POSSIBLES ; chaque chemin doit prouver celles qu'il peut
// réellement atteindre, telles que la matrice de couverture les décrit.
package veracity

import (
	"fmt"
	"sort"
	"strings"
)

// Verdict est l'un des quatre statuts qu'un scan publie.
type Verdict string

// Les quatre verdicts, tels qu'ils apparaissent dans `--format assessment`.
const (
	Fail          Verdict = "fail"
	Pass          Verdict = "pass"
	NotEvaluated  Verdict = "not-evaluated"
	NotApplicable Verdict = "not-applicable"
)

// Path est un chemin contrôle × fournisseur × source : l'unité dont la véracité
// se prouve.
type Path struct {
	Control  string
	Provider string
	Source   string
}

// Cell est une case de la matrice de couverture, réduite à ce dont le contrat a
// besoin. Ce paquet ne connaît PAS la matrice : elle est calculée ailleurs, avec
// les descripteurs et le référentiel, et lui passer des cases plutôt qu'une
// matrice le laisse être une feuille de l'arbre de dépendances — la documentation
// générée peut alors publier le compteur de dette sans que rien ne boucle.
type Cell struct {
	Control  string
	Provider string
	Source   string
	// Status : supported | partial | not-applicable | unsupported, le vocabulaire
	// de la matrice de couverture.
	Status string
}

// Les quatre statuts de couverture, tels que la matrice les nomme.
const (
	StatusSupported     = "supported"
	StatusPartial       = "partial"
	StatusNotApplicable = "not-applicable"
	StatusUnsupported   = "unsupported"
)

// String rend le chemin sous sa forme de registre : trois champs séparés par une
// espace, stables et triables.
func (p Path) String() string {
	return p.Control + " " + p.Provider + " " + p.Source
}

// Obligations rend, par chemin, les verdicts qu'il doit savoir prouver.
//
// La règle suit ce que la matrice décrit, cellule par cellule :
//
//   - `supported` : le chemin peut conclure. Il doit prouver `fail` (une
//     configuration vulnérable est détectée), `pass` (une configuration
//     réellement correcte est confirmée) et `not-evaluated` (l'attribut décisif
//     manque). Trois obligations.
//   - `partial` : le verrou du « pass » ne peut pas être levé sur cette source.
//     Le seul verdict atteignable est `not-evaluated`, et c'est précisément
//     l'énoncé qu'il faut protéger d'une régression vers un faux `pass`.
//   - `not-applicable` : le contrat du fournisseur déclare le mécanisme
//     inexistant. Le chemin doit prouver qu'il rend bien `not-applicable`, avec
//     sa justification — un N/A non prouvé est un contrôle qu'on croit couvert.
//   - `unsupported` : Pépin ne conclut rien. Il n'y a pas de verdict à prouver ;
//     ce que la matrice affirme là est vérifié par ses propres portes.
func Obligations(cells []Cell) map[Path][]Verdict {
	out := map[Path][]Verdict{}
	for _, c := range cells {
		p := Path{Control: c.Control, Provider: c.Provider, Source: c.Source}
		switch c.Status {
		case StatusSupported:
			out[p] = []Verdict{Fail, Pass, NotEvaluated}
		case StatusPartial:
			out[p] = []Verdict{NotEvaluated}
		case StatusNotApplicable:
			out[p] = []Verdict{NotApplicable}
		case StatusUnsupported:
			// Rien à prouver : le scan ne conclut pas sur ce chemin.
		}
	}
	return out
}

// Debt rend le registre de dette : une ligne par chemin dont il reste des
// verdicts à prouver, `<contrôle> <fournisseur> <source> <verdicts manquants>`,
// triée. `covered` porte, par chemin, les verdicts effectivement prouvés par un
// scénario committé.
//
// La forme est délibérément pauvre — du texte trié, une ligne par chemin — pour
// qu'un diff se lise : une ligne qui disparaît est une dette payée, une ligne qui
// apparaît est une dette contractée, et les deux se voient en relecture.
func Debt(obligations map[Path][]Verdict, covered map[Path][]Verdict) []string {
	var lines []string
	for path, want := range obligations {
		have := map[Verdict]bool{}
		for _, v := range covered[path] {
			have[v] = true
		}
		var missing []string
		for _, v := range want {
			if !have[v] {
				missing = append(missing, string(v))
			}
		}
		if len(missing) == 0 {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s %s", path, strings.Join(missing, ",")))
	}
	sort.Strings(lines)
	return lines
}

// Counts résume l'état du contrat : chemins ayant au moins une obligation,
// chemins entièrement prouvés, obligations totales et obligations restantes.
// Publié par la documentation générée — une dette qu'on ne compte pas est une
// dette qu'on n'a pas.
type Counts struct {
	Paths       int
	PathsProven int
	Obligations int
	Remaining   int
}

// Count calcule le résumé.
func Count(obligations map[Path][]Verdict, covered map[Path][]Verdict) Counts {
	c := Counts{Paths: len(obligations)}
	for path, want := range obligations {
		have := map[Verdict]bool{}
		for _, v := range covered[path] {
			have[v] = true
		}
		proven := 0
		for _, v := range want {
			c.Obligations++
			if have[v] {
				proven++
			}
		}
		c.Remaining += len(want) - proven
		if proven == len(want) {
			c.PathsProven++
		}
	}
	return c
}

// Merge fusionne plusieurs jeux de verdicts prouvés, indexés par chemin.
//
// Deux artefacts prouvent aujourd'hui la véracité d'un chemin, et ils ne se
// remplacent pas : les SCÉNARIOS, écrits pour mettre en scène un cas précis
// (notamment une réponse d'API canned, seul point d'entrée qui éprouve le
// collecteur), et les TENANTS DE RÉFÉRENCE, qui rejouent des configurations
// tierces que personne n'a écrites pour Pépin. Les compter ensemble est ce qui
// évite d'avoir deux chiffres de couverture — et celui qui diverge est toujours
// celui qu'on lit.
func Merge(sets ...map[Path][]Verdict) map[Path][]Verdict {
	out := map[Path][]Verdict{}
	for _, set := range sets {
		for p, verdicts := range set {
			out[p] = append(out[p], verdicts...)
		}
	}
	return out
}

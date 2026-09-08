package assess

import (
	"fmt"
	"sort"
	"strings"

	"github.com/stephrobert/pepin/internal/genprovider"
	"github.com/stephrobert/pepin/internal/i18n"
	"github.com/stephrobert/scankit/assessment"
)

// Distinguer « cherché et non exposé » de « jamais cherché » (ADR-0017, issue #121).
//
// Certaines règles concluent depuis une ABSENCE, et elles ont raison de le faire :
// chez Scaleway, `ExpiresAt` est un `*time.Time` et un pointeur nul EST la façon dont
// l'API dit « aucune expiration ». Exiger la présence du champ rendrait
// `iam_accesskey_expiration_set` aveugle au cas même qu'il existe pour voir.
//
// Mais pour la règle, deux situations sont indiscernables :
//
//	le pointeur était nul        -> attribut absent -> une OBSERVATION, l'écart est réel
//	le champ n'est pas mappé     -> attribut absent -> une LACUNE, l'écart est inventé
//
// La provenance porte cette distinction, et elle seule : une clé qui y figure sans
// figurer dans `attributes` est un champ CHERCHÉ et non exposé par la source
// (ADR-0007). Mesuré avant correction : les deux cas sortaient `fail`, dont un
// `critical` fabriqué depuis un champ que personne n'avait demandé — exactement ce
// que l'ADR-0014 interdit.
//
// La lecture se fait ICI, dans l'assessment, et jamais dans une règle : c'était la
// raison d'être de l'index parallèle, et elle tient toujours.

// absenceMustBeAttested — les contrôles dont l'ÉCART NAÎT D'UNE ABSENCE, avec le
// type qui porte le champ et le champ lui-même.
//
// La table est courte à dessein. Un contrôle n'y entre que si sa règle conclut
// VRAIMENT depuis l'absence d'un attribut : c'est une propriété du Rego, pas une
// préférence, et TestAbsenceDecidersReallyReadAnAbsence la confronte aux règles.
//
// Elle est distincte de `requiredAttr`, et ne peut pas fusionner avec elle : cette
// dernière exige la PRÉSENCE d'un attribut pour conclure, celle-ci décrit un contrôle
// qui conclut de son ABSENCE. Inscrire un contrôle dans les deux le rendrait muet.
var absenceMustBeAttested = map[string]map[string][]string{
	"iam_accesskey_expiration_set": {"access_key": {"expiration_date"}},
}

// AbsenceDeciders expose la table pour les gardes et la documentation générée.
func AbsenceDeciders() map[string]map[string][]string {
	out := make(map[string]map[string][]string, len(absenceMustBeAttested))
	for code, parType := range absenceMustBeAttested {
		m := map[string][]string{}
		for t, attrs := range parType {
			m[t] = append([]string(nil), attrs...)
		}
		out[code] = m
	}
	return out
}

// WithAttestedAbsence requalifie en `not-evaluated` tout écart né d'une absence que
// la provenance n'atteste pas.
//
// C'est la passe qui DÉCIDE, par opposition à `WithProvenance` qui ANNOTE. Les deux
// sont séparées et nommées pour ce qu'elles font : une fonction dont le nom dit
// qu'elle déplace un verdict est plus honnête qu'un invariant qui affirme le
// contraire de ce que fait le programme (ADR-0017).
//
// Trois bornes, et ce sont elles qui rendent la lecture sûre :
//
//   - elle ne CRÉE jamais un écart, elle ne peut qu'en retirer un ;
//   - une ressource sans AUCUNE provenance n'est pas touchée — un inventaire reçu
//     d'un tiers n'a pas été collecté par Pépin, qui n'a donc rien à en attester,
//     ni dans un sens ni dans l'autre ;
//   - seuls les contrôles déclarés dans `absenceMustBeAttested` sont concernés.
func WithAttestedAbsence(a assessment.Assessment, idx ProvenanceIndex) assessment.Assessment {
	if len(idx) == 0 {
		return a
	}
	res := append([]assessment.Result(nil), a.Results...)
	for i := range res {
		if res[i].Status != assessment.Fail {
			continue // on ne retire que des écarts, jamais on n'en ajoute
		}
		parType, decide := absenceMustBeAttested[res[i].Control]
		if !decide {
			continue
		}
		motif, nonAtteste := unattestedAbsence(parType, idx)
		if !nonAtteste {
			continue
		}
		res[i].Status = assessment.NotEvaluated
		res[i].Evidence.Observed = motif
	}
	a.Results = res
	return a
}

// unattestedAbsence dit si le champ dont dépend l'écart n'a JAMAIS été cherché, et
// pourquoi on peut l'affirmer.
//
// La condition « le type porte une provenance pour d'autres attributs » est le cœur
// du mécanisme : c'est elle qui prouve que Pépin a collecté ce type et savait quoi y
// chercher. Sans elle, un export sans provenance ferait disparaître tous ces écarts,
// ce qui remplacerait un faux positif par un faux vert.
func unattestedAbsence(parType map[string][]string, idx ProvenanceIndex) (string, bool) {
	types := make([]string, 0, len(parType))
	for t := range parType {
		types = append(types, t)
	}
	sort.Strings(types) // ordre stable : ce motif part dans un rapport opposable

	var jamaisCherches []string
	for _, t := range types {
		byAttr := idx[t]
		if len(byAttr) == 0 {
			continue // ce type n'a aucune attestation : on ne conclut rien
		}
		attrs := append([]string(nil), parType[t]...)
		sort.Strings(attrs)
		for _, attr := range attrs {
			if _, cherche := byAttr[attr]; cherche {
				continue // le champ a été cherché : l'absence EST l'observation
			}
			jamaisCherches = append(jamaisCherches, fmt.Sprintf(i18n.T(
				"« %s » sur les ressources de type « %s »",
				"\"%s\" on the resources of type \"%s\""), attr, t))
		}
	}
	if len(jamaisCherches) == 0 {
		return "", false
	}
	return fmt.Sprintf(i18n.T(
		"écart déduit d'une absence NON attestée : %s n'a jamais été cherché par le collecteur — "+
			"l'absence ne vaut donc pas observation, et l'écart ne peut être affirmé",
		"deviation inferred from an UNATTESTED absence: %s was never sought by the collector — "+
			"the absence is therefore not an observation, and the deviation cannot be asserted"),
		strings.Join(jamaisCherches, ", ")), true
}

// absenceDeciderTypesAreKnown : les types déclarés doivent être ceux que le contrôle
// lit réellement. Exposé pour la garde, qui vit dans le paquet de test.
func absenceDeciderTypesAreKnown(code string, parType map[string][]string) []string {
	lus := map[string]bool{}
	for _, t := range genprovider.ControlTypes(code) {
		lus[t] = true
	}
	var inconnus []string
	for t := range parType {
		if !lus[t] {
			inconnus = append(inconnus, t)
		}
	}
	sort.Strings(inconnus)
	return inconnus
}

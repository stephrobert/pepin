package referentiel

// La CONFIGURATION d'un contrôle est liée à ses correspondances normatives.
//
// Une fois un contrôle réglable, rien n'empêche de le desserrer tout en
// continuant d'afficher la même correspondance CIS ou SecNumCloud. Un contrôle
// assoupli qui prétend couvrir la même exigence rend le rapport trompeur, et pour
// un outil qui vend l'opposabilité, c'est le défaut le plus coûteux.
//
// Le référentiel porte donc, à côté de `scsl:` et `frameworks:`, la CONTRAINTE
// sous laquelle ces correspondances valent :
//
//	config_requise:
//	  - parametre: snapshots.max_age_days
//	    contrainte: au_plus_le_defaut
//
// Elle se lit : « la correspondance CLD-STO-3 / CIS 11.2 / ISO A.8.13 ne vaut que
// tant que le délai de fraîcheur ne dépasse pas le défaut ». Si la configuration
// effective s'en écarte, la correspondance est ABANDONNÉE — le résultat reste
// publié, avec ce qu'il mesure vraiment, mais sans prétendre couvrir l'exigence.
//
// Deux invariants, tenus par TestConfigConstraintsAreInterpretable :
//   - le `parametre` est l'un de ceux que le moteur de politique sait évaluer ;
//   - la `contrainte` est l'un des quatre sens définis ci-dessous.
//
// Une contrainte ininterprétable est refusée par `mise run validate` : une
// contrainte que personne n'évalue est pire qu'une contrainte absente, parce
// qu'elle donne l'impression d'être appliquée.

// ConfigConstraint est une contrainte de configuration sous laquelle les
// correspondances normatives d'un contrôle valent.
type ConfigConstraint struct {
	// Parametre : le chemin du réglage (`section.champ`), tel que le moteur de
	// politique le nomme.
	Parametre string `yaml:"parametre"`
	// Contrainte : le SENS dans lequel le réglage peut s'écarter du défaut sans
	// rompre la correspondance.
	Contrainte string `yaml:"contrainte"`
}

// Les quatre sens de contrainte. Chacun désigne le côté du défaut où la
// correspondance TIENT ENCORE ; s'en écarter de l'autre côté est un
// assouplissement, et la correspondance tombe.
const (
	// ConstraintAtMostDefault : la valeur ne doit pas DÉPASSER le défaut. Pour un
	// délai de fraîcheur ou un seuil de confiance : l'allonger ou le monter tait
	// ce que l'exigence demandait de voir.
	ConstraintAtMostDefault = "au_plus_le_defaut"
	// ConstraintSupersetOfDefault : la valeur doit CONTENIR au moins le défaut.
	// Pour un ensemble d'exigences (étiquettes obligatoires, types couverts) :
	// en retirer un membre, c'est cesser de vérifier ce que l'exigence demande.
	ConstraintSupersetOfDefault = "superset_du_defaut"
	// ConstraintSubsetOfDefault : la valeur doit rester CONTENUE dans le défaut.
	// Pour un ensemble de tolérances (états jugés exploitables) : l'élargir, c'est
	// accepter ce que l'exigence rejette.
	ConstraintSubsetOfDefault = "sous_ensemble_du_defaut"
	// ConstraintAtLeastAsStrict : pour chaque EXIGENCE du profil par défaut, le
	// profil effectif en porte une AU MOINS AUSSI STRICTE.
	//
	// Une comparaison par nom ne convient pas ici, et l'exemple qui l'a montré vaut
	// d'être gardé : le profil par défaut exige « la question de l'environnement est
	// répondue par l'une des écritures env | environment | stage ». Une organisation
	// qui écrit `environment` n'assouplit rien — elle exige MOINS d'écritures, donc
	// davantage. Comparer les noms l'aurait signalée comme assouplie, ce qui aurait
	// fait perdre sa correspondance normative à quelqu'un qui a resserré sa
	// convention : le faux positif le plus coûteux qui soit, puisqu'il punit le bon
	// comportement.
	//
	// La règle exacte : une exigence par défaut D reste tenue s'il existe une
	// exigence effective E dont les écritures acceptées sont NON VIDES et INCLUSES
	// dans celles de D. Tout ce qui satisfait E satisfait alors D. Élargir les
	// écritures d'un nom (un alias de plus) sort de l'inclusion, donc assouplit ;
	// retirer une exigence la laisse sans candidat, donc assouplit ; en ajouter une
	// nouvelle ne touche à aucune exigence par défaut, donc durcit.
	ConstraintAtLeastAsStrict = "au_moins_aussi_strict_que_le_defaut"
)

// ConfigConstraintKinds énumère les sens de contrainte interprétables.
var ConfigConstraintKinds = []string{
	ConstraintAtMostDefault,
	ConstraintSupersetOfDefault,
	ConstraintSubsetOfDefault,
	ConstraintAtLeastAsStrict,
}

// ConfigParameters énumère les réglages qu'une contrainte peut nommer. La liste
// vit ICI, dans le référentiel, parce que c'est le référentiel qui déclare les
// contraintes ; le moteur de politique (internal/policy) sait évaluer exactement
// ces noms, et TestEveryDeclaredParameterIsEvaluated le vérifie dans les deux
// sens — un nom que le moteur ignore ne protégerait rien, un réglage que le
// référentiel ne nomme pas ne pourrait jamais rompre une correspondance.
var ConfigParameters = []string{
	"tagging.required_tags",
	"tagging.network_required_tags",
	"tagging.resource_types",
	"snapshots.max_age_days",
	"snapshots.accepted_states",
	"secrets.min_confidence",
}

// ConfigConstraintsByControl rend, par code de contrôle, les contraintes de
// configuration sous lesquelles ses correspondances normatives valent. Les
// contrôles sans contrainte (la grande majorité : ils ne sont pas réglables) sont
// absents de la carte.
func ConfigConstraintsByControl() map[string][]ConfigConstraint {
	out := map[string][]ConfigConstraint{}
	for code, c := range byCode {
		if len(c.ConfigRequise) > 0 {
			out[code] = c.ConfigRequise
		}
	}
	return out
}

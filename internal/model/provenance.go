package model

// Provenance : l'état d'observation de chaque attribut d'une ressource.
//
// Pour un outil de posture, ces deux situations ne disent PAS la même chose :
//
//	`false`                       →  observé, et la valeur est faux
//	absent (donc zéro-value Go)   →  jamais observé
//
// La projection fait déjà la différence (collect.Project ne pose pas un attribut
// que la source n'expose pas), mais elle la PERD aussitôt : une fois l'attribut
// dans `attributes`, plus rien ne dit d'où il vient. Un attribut posé par le
// `const:` d'un descripteur y est indiscernable d'un attribut lu dans une réponse
// d'API — or l'un est une attestation du descripteur, l'autre une mesure.
//
// L'attestation vit À CÔTÉ de la valeur, jamais à sa place : `attributes` garde
// exactement la forme que les règles Rego lisent, et `provenance` est un index
// PARALLÈLE, indexé par le même nom d'attribut. Aucune règle n'a à changer, et
// aucune ne peut changer de verdict du fait de la provenance.
//
// Règle de conception, non négociable : **la provenance ne désigne jamais un
// appel d'API qui n'a pas eu lieu.** Le libellé d'une source `api` est construit
// à partir de la requête RÉELLEMENT émise (méthode + URL, après réponse), pas de
// ce que la spec YAML déclare appeler. Nommer un endpoint pour un attribut en
// réalité dérivé donnerait l'APPARENCE de la traçabilité, ce qui est pire que son
// absence.

// Origin nomme la nature de la source d'un attribut. Trois natures, parce que ce
// sont les trois que Pépin sait produire et distinguer.
type Origin string

const (
	// OriginAPI : la valeur vient d'une réponse d'API live. `Source` porte alors
	// la requête effectivement émise (« GET https://api.exemple/… »).
	OriginAPI Origin = "api"
	// OriginTerraform : la valeur vient d'un plan Terraform (état PLANIFIÉ, pas
	// la configuration effective). `Source` porte le type de ressource du plan.
	OriginTerraform Origin = "terraform-plan"
	// OriginDerived : la valeur est produite LOCALEMENT — littéral `const:` d'un
	// descripteur, fait déclaré (souveraineté), agrégat calculé. Aucun appel
	// d'API ne l'a portée, et l'attestation le dit.
	OriginDerived Origin = "derived"
)

// Attestation dit d'où vient UN attribut et s'il a réellement été observé.
type Attestation struct {
	// Origin : api | terraform-plan | derived.
	Origin Origin `json:"origin"`
	// Source : l'appel effectif (origine api), le type de ressource du plan
	// (terraform-plan), ou l'élément de descripteur (derived).
	Source string `json:"source,omitempty"`
	// Path : le chemin natif LU dans le document source (champ de la réponse
	// d'API, attribut du plan). Vide pour une valeur dérivée : il n'y a rien à
	// désigner.
	Path string `json:"path,omitempty"`
	// Observed : le document source portait réellement `Path`. Faux quand la
	// source ne l'expose pas — c'est la distinction « absent » / « présent et
	// vide » rendue durable jusqu'au modèle.
	Observed bool `json:"observed"`
	// Derived : la valeur a été CALCULÉE (transform, agrégat, littéral) plutôt
	// que recopiée telle quelle. Orthogonal à Origin : un agrégat calculé sur une
	// réponse réelle est `api` + `derived`.
	Derived bool `json:"derived,omitempty"`
}

// Provenance indexe les attestations par nom d'attribut. Une clé peut exister
// SANS que l'attribut correspondant soit dans `attributes` : c'est le cas d'un
// champ cherché dans la source et qu'elle n'exposait pas (Observed faux). C'est
// la forme la plus utile de l'information — « on a regardé, il n'y était pas »
// n'est pas « on n'a jamais regardé ».
type Provenance map[string]Attestation

// Attest enregistre l'attestation d'un attribut (crée la carte au besoin).
func (p *Provenance) Attest(attr string, a Attestation) {
	if *p == nil {
		*p = Provenance{}
	}
	(*p)[attr] = a
}

// AttestPresent atteste les attributs PRÉSENTS de `attrs` à partir d'une table
// attribut → appel. Réservé aux collecteurs Go dont chaque attribut a son propre
// appel (stockage objet) : l'attribut n'est posé que si son appel a réussi, donc
// l'attester est une lecture fidèle de ce qui s'est passé. Un attribut absent de
// la table n'est PAS attesté — plutôt aucune attestation qu'une fausse.
func AttestPresent(attrs map[string]any, calls map[string]string, derived map[string]bool) Provenance {
	var p Provenance
	for attr := range attrs {
		call, ok := calls[attr]
		if !ok {
			continue
		}
		p.Attest(attr, Attestation{
			Origin:   OriginAPI,
			Source:   call,
			Observed: true,
			Derived:  derived[attr],
		})
	}
	return p
}

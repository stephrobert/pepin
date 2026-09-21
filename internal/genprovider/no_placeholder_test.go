package genprovider

import (
	"testing"

	"github.com/stephrobert/pepin/internal/collect"
	"github.com/stephrobert/pepin/internal/tfmap"
)

// AUCUNE SPEC NE FABRIQUE UN ATTRIBUT À PARTIR DE RIEN.
//
// C'est la seconde moitié de l'issue #227, et la forme qu'elle devait prendre.
//
// # Ce que la première moitié a établi
//
// Un attribut ABSENT n'a pas été observé ; un attribut PRÉSENT ET VIDE l'a été, et sa
// valeur est « aucun ». Le verrou de capacité répond désormais sur la présence, ce qui
// rend au contrôle sa capacité à conclure sur une ressource qui n'a rien — une instance
// sans groupe de sécurité, un stockage sans étiquette.
//
// Cette réponse n'est JUSTE que si un vide qui atteint l'inventaire vient toujours de la
// source. C'était faux : `IAMPolicyStatements` posait `[]` pour un document illisible,
// et ce bouchon obligeait le verrou à se méfier de tous les vides. Il a été retiré.
//
// # Pourquoi une porte, et pas cent trente-huit annotations
//
// L'exigence est que la sémantique cesse d'être implicite. On pourrait l'écrire attribut
// par attribut — 138 aujourd'hui, sur quatre fournisseurs — mais une annotation qu'aucune
// machine ne lit se périme au premier collecteur ajouté, et personne ne relit deux cents
// lignes de commentaire. La règle est donc UNE, et elle se mesure :
//
//	une source qui ne porte rien ne projette rien
//
// Ce que cette porte interdit mécaniquement, c'est le bouchon — la seule façon dont un
// vide peut atteindre l'inventaire sans avoir été observé. Elle vaut pour les 138
// attributs d'aujourd'hui, et pour ceux qu'un fournisseur futur ajoutera sans avoir lu
// cette histoire.
// specProjetante réduit les deux formes de spec — collecte live et mapping
// Terraform — à ce que cette porte a besoin d'en savoir. Les deux projettent par
// le même `collect.Project` ; seuls leurs types Go diffèrent.
type specProjetante struct {
	Type       string
	Map        map[string]string
	Transforms map[string]any
}

func projetantes(rs []collect.ResourceSpec) []specProjetante {
	out := make([]specProjetante, 0, len(rs))
	for _, r := range rs {
		out = append(out, specProjetante{Type: r.Type, Map: r.Map, Transforms: r.Transforms})
	}
	return out
}

func projetantesTF(rs []tfmap.ResourceSpec) []specProjetante {
	out := make([]specProjetante, 0, len(rs))
	for _, r := range rs {
		out = append(out, specProjetante{Type: r.Type, Map: r.Map, Transforms: r.Transforms})
	}
	return out
}

func TestNoSpecFabricatesAnAttributeFromNothing(t *testing.T) {
	vide := map[string]any{}
	for name, d := range loadAllDescriptors(t) {
		// LES DEUX SOURCES, et la seconde manquait.
		//
		// La porte n'itérait que `Collecte` — la collecte live. Le mapping Terraform
		// projette pourtant par le MÊME `collect.Project`, avec les mêmes transforms,
		// et c'est lui que l'issue #243 vise. Une spec de mapping pouvait donc
		// fabriquer un attribut depuis rien sans que rien ne le dise, ce qui est
		// exactement ce que cette porte existe pour interdire.
		//
		// La distinction entre les deux chemins n'est pas dans ce qu'ils ont le droit
		// de projeter : elle est dans ce qu'un vide y SIGNIFIE (ADR-0024). Une clé
		// absente ne projette rien des deux côtés ; ce qu'un `null` ÉCRIT vaut est une
		// déclaration sourcée, et elle se garde ailleurs.
		specs := append(append([]specProjetante{}, projetantes(d.Collecte.Resources)...),
			projetantesTF(d.MappingTerraform.Resources)...)
		for _, r := range specs {
			attrs := collect.Project(vide, r.Map, r.Transforms)
			for attr, v := range attrs {
				t.Errorf("%s / type %q : l'attribut %q est projeté (%#v) depuis un item VIDE.\n"+
					"  Une source qui ne porte rien ne doit rien projeter : un attribut fabriqué\n"+
					"  franchit le verrou de capacité et fait conclure un contrôle sur zéro\n"+
					"  information — c'est l'incident fondateur de l'ADR-0006, et c'est le\n"+
					"  bouchon que l'issue #227 a retiré de `IAMPolicyStatements`.\n"+
					"  Si la source peut légitimement rendre « aucun », c'est à ELLE de le dire :\n"+
					"  la clé doit être présente dans la réponse, pas inventée par la spec.",
					name, r.Type, attr, v)
			}
		}
	}
}

// LE CONTRE-EXEMPLE, et il vaut autant que la porte.
//
// Interdire de fabriquer ne doit pas faire perdre l'information inverse : quand la
// source porte la clé et répond « aucun », la spec DOIT projeter le vide. Sans ce cas,
// « ne rien projeter » se satisferait en ne projetant jamais rien, et on aurait échangé
// un faux vert contre un aveuglement.
func TestASourceThatSaysNoneStillProjectsIt(t *testing.T) {
	mapping := map[string]string{"security_group_ids": "sg"}
	transforms := map[string]any{"security_group_ids": "list"}

	// La clé est là, sa valeur est nulle : l'API dit « aucun groupe ».
	dit := collect.Project(map[string]any{"sg": nil}, mapping, transforms)
	got, present := dit["security_group_ids"]
	if !present {
		t.Fatal("la source dit « aucun » et rien n'est projeté : le contrôle qui cherche " +
			"une ressource sans groupe redevient aveugle (#227)")
	}
	if arr, ok := got.([]any); !ok || len(arr) != 0 {
		t.Errorf("attendu une liste vide observée, got %#v", got)
	}

	// La clé n'est PAS là : rien n'est projeté, et c'est la porte ci-dessus.
	tait := collect.Project(map[string]any{"autre": 1}, mapping, transforms)
	if _, present := tait["security_group_ids"]; present {
		t.Error("clé absente de la source : la spec a fabriqué une valeur")
	}
}

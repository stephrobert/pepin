package genprovider

import (
	"os"
	"strings"
	"testing"
	"time"
)

// La porte des DROITS MINIMAUX : chaque unité de collecte qu'un descripteur sait
// produire doit dire de quel droit elle a besoin, et si ce droit est confirmé.
//
// L'oubli qu'elle empêche est mécanique : on ajoute un endpoint à la spec
// `collecte`, la couverture progresse, et personne ne se souvient d'aller écrire
// le droit qu'il faudra pour l'appeler. L'utilisateur le découvre alors en
// production, sous la forme d'un `403` dont Pépin ne sait pas dire ce qui manque.
//
// La porte n'exige PAS que le droit soit vérifié. Elle exige qu'il soit DÉCLARÉ
// et que son état soit dit. `a_verifier` est une réponse parfaitement recevable —
// c'est la seule honnête tant qu'aucun scan à rôle réduit n'a porté sur l'unité.
// Ce qui est refusé, c'est le silence.
//
// Quand un tel scan a bien eu lieu, il se CONSIGNE (`mesure:`), et les deux portes
// du bas disent ce que ce consignement engage.

// collectionUnitsOf énumère les unités de collecte qu'un descripteur produit :
// les types de la spec `collecte`, plus les collecteurs Go que le descripteur
// active. C'est exactement la liste que `internal/collectkit` enregistre à
// l'exécution ; les deux dériveraient si l'une était recopiée à la main.
func collectionUnitsOf(d Descriptor) []string {
	seen := map[string]bool{}
	var out []string
	add := func(u string) {
		if u == "" || seen[u] {
			return
		}
		seen[u] = true
		out = append(out, u)
	}
	for _, r := range d.Collecte.Resources {
		add(r.Type)
	}
	if d.S3.Endpoint != "" {
		add("object_storage_bucket")
	}
	if d.EIM.InlinePolicies {
		add("iam_policy_inline")
	}
	if d.OKS.Endpoint != "" {
		add("kubernetes_cluster")
	}
	return out
}

func loadAllDescriptors(t *testing.T) map[string]Descriptor {
	t.Helper()
	entries, err := os.ReadDir(providersDir)
	if err != nil {
		t.Fatalf("dossier providers illisible : %v", err)
	}
	out := map[string]Descriptor{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		d, lerr := Load(os.DirFS(providersDir), e.Name())
		if lerr != nil {
			t.Fatalf("chargement de %s : %v", e.Name(), lerr)
		}
		out[strings.TrimSuffix(e.Name(), ".yaml")] = d
	}
	if len(out) == 0 {
		t.Fatal("aucun descripteur trouvé : la porte ne mesure plus rien")
	}
	return out
}

// TestEveryCollectionUnitDeclaresItsMinimumGrant : aucune unité de collecte ne
// reste muette sur les droits qu'elle exige.
func TestEveryCollectionUnitDeclaresItsMinimumGrant(t *testing.T) {
	for name, d := range loadAllDescriptors(t) {
		declared := map[string]bool{}
		for _, p := range d.Permissions {
			declared[p.Unit] = true
		}
		for _, unit := range collectionUnitsOf(d) {
			if !declared[unit] {
				t.Errorf("%s : l'unité de collecte %q ne déclare aucun droit minimal.\n"+
					"Ajouter une entrée `permissions:` dans providers/%s.yaml — `etat: a_verifier` est "+
					"une réponse recevable, le silence n'en est pas une.", name, unit, name)
			}
		}
	}
}

// TestNoPermissionEntryDescribesAUnitThatDoesNotExist : l'autre sens. Une entrée
// orpheline est le symptôme d'un endpoint retiré dont le droit est resté, et une
// page de documentation qui réclamerait un droit devenu inutile ferait donner
// plus que le nécessaire — le contraire de l'objet de cette table.
func TestNoPermissionEntryDescribesAUnitThatDoesNotExist(t *testing.T) {
	for name, d := range loadAllDescriptors(t) {
		units := map[string]bool{}
		for _, u := range collectionUnitsOf(d) {
			units[u] = true
		}
		for _, p := range d.Permissions {
			if !units[p.Unit] {
				t.Errorf("%s : le droit déclaré pour %q ne correspond à aucune unité de collecte de ce descripteur",
					name, p.Unit)
			}
		}
	}
}

// TestEveryPermissionStatesItsStateAndItsSource : un droit sans état ni source
// n'est pas une information, c'est une affirmation. `verifie` engage la source
// citée ; `a_verifier` engage à dire pourquoi.
func TestEveryPermissionStatesItsStateAndItsSource(t *testing.T) {
	for name, d := range loadAllDescriptors(t) {
		for _, p := range d.Permissions {
			switch p.Etat {
			case "verifie", "a_verifier":
			default:
				t.Errorf("%s / %s : etat %q inconnu (verifie | a_verifier)", name, p.Unit, p.Etat)
			}
			if strings.TrimSpace(p.Source) == "" {
				t.Errorf("%s / %s : aucune source citée — un droit sans source ne se vérifie pas", name, p.Unit)
			}
			// Une réserve est lue par quelqu'un qui doit décider quels droits accorder :
			// elle est bilingue comme tout ce que Pépin imprime.
			if (p.Note == "") != (p.NoteEn == "") {
				t.Errorf("%s / %s : réserve renseignée dans une seule langue (note=%q, note_en=%q)",
					name, p.Unit, p.Note, p.NoteEn)
			}
		}
	}
}

// TestAnUnverifiedPermissionSaysWhy : `a_verifier` sans explication laisse le
// lecteur devant un « non » qu'il ne peut pas lever. La réserve est donc exigée —
// c'est elle qui distingue « nous n'avons pas cherché » de « nous avons cherché
// et la documentation ne le dit pas ».
func TestAnUnverifiedPermissionSaysWhy(t *testing.T) {
	for name, d := range loadAllDescriptors(t) {
		for _, p := range d.Permissions {
			if p.Etat != "a_verifier" {
				continue
			}
			if strings.TrimSpace(p.Note) == "" {
				t.Errorf("%s / %s : droit non vérifié sans réserve écrite — dire ce qui manque, pas seulement qu'il manque",
					name, p.Unit)
			}
		}
	}
}

// TestGrantsAndSourcesCarryNoProse : `grant` et `source` sont des IDENTIFIANTS —
// vocabulaire natif du fournisseur, référence d'un document — et ils sont rendus
// TELS QUELS dans les deux langues. Une prose française y apparaîtrait au milieu
// d'une page anglaise ; c'est ce que la porte de langue interdit ailleurs, et il
// n'y a pas de raison que cette table y échappe. Ce qui est prose va dans `note`.
func TestGrantsAndSourcesCarryNoProse(t *testing.T) {
	// Le marqueur mécanique : une lettre accentuée. Il ne prétend pas détecter
	// toute prose française, mais il attrape le cas réel — celui d'une phrase
	// écrite au fil de la plume dans un champ qui devait rester un identifiant.
	hasAccent := func(s string) bool {
		for _, r := range s {
			if r > 127 && (r < 0x2000 || r > 0x2BFF) { // hors ponctuation et symboles
				return true
			}
		}
		return false
	}
	for name, d := range loadAllDescriptors(t) {
		for _, p := range d.Permissions {
			for _, f := range []struct{ label, value string }{
				{"grant", p.Grant}, {"source", p.Source},
			} {
				if hasAccent(f.value) {
					t.Errorf("%s / %s : le champ %s porte de la prose accentuée (%q) ; les identifiants restent neutres, la prose va dans `note`/`note_en`",
						name, p.Unit, f.label, f.value)
				}
			}
		}
	}
}

// TestAMeasuredPermissionIsDatedAndSituated : un scan à rôle réduit ne vaut que
// daté et situé.
//
// Sans date, rien ne permet de juger le relevé périmé — et un droit se retire, un
// service change de modèle d'identité. Sans région, on ne sait pas sur quel plan
// de contrôle il a porté, alors qu'un même fournisseur n'expose pas les mêmes
// services partout (OKS n'existe pas en cloudgouv). Une mesure vague se lirait
// comme une preuve tout en n'en étant pas une, ce qui est pire que son absence.
func TestAMeasuredPermissionIsDatedAndSituated(t *testing.T) {
	for name, d := range loadAllDescriptors(t) {
		for _, p := range d.Permissions {
			m := p.Mesure
			if m == (MesurePermission{}) {
				continue // aucun scan à rôle réduit n'a porté sur cette unité
			}
			if _, err := time.Parse("2006-01-02", m.Date); err != nil {
				t.Errorf("%s / %s : mesure sans date lisible (%q) — un relevé qu'on ne peut pas juger périmé n'est pas un relevé",
					name, p.Unit, m.Date)
			}
			if strings.TrimSpace(m.Region) == "" {
				t.Errorf("%s / %s : mesure sans région — un droit se mesure sur un plan de contrôle, pas dans l'absolu",
					name, p.Unit)
			}
			switch m.RoleReduit {
			case RoleReduitSuffisant, RoleReduitRefuse:
			default:
				t.Errorf("%s / %s : role_reduit %q inconnu (%s | %s)",
					name, p.Unit, m.RoleReduit, RoleReduitSuffisant, RoleReduitRefuse)
			}
		}
	}
}

// TestAMeasuredRefusalNamesTheGrantThatWorks : le défaut fondateur de l'issue #168,
// tenu par un test.
//
// `object_storage_bucket` et `kubernetes_cluster` portaient un `grant` VIDE. Le
// relevé de capacités n'imprime la ligne « droit requis » que si le descripteur la
// déclare, et le motif d'un « non évalué » de même : l'opérateur ne lisait donc que
// l'erreur brute d'OOS — « The AWS access key Id you provided does not exist in our
// records » — qui l'envoyait vérifier une clé parfaitement valide, alors que le fait
// mesuré est qu'OOS ne connaît AUCUNE clé EIM.
//
// Un refus mesuré sans droit nommé est donc pire qu'un refus non mesuré : on sait ce
// qu'il faut, et on ne le dit pas. La réserve est exigée dans la foulée, parce que
// c'est elle qui porte ce que le fournisseur a réellement répondu.
func TestAMeasuredRefusalNamesTheGrantThatWorks(t *testing.T) {
	for name, d := range loadAllDescriptors(t) {
		for _, p := range d.Permissions {
			if p.Mesure.RoleReduit != RoleReduitRefuse {
				continue
			}
			if strings.TrimSpace(p.Grant) == "" {
				t.Errorf("%s / %s : rôle réduit mesuré REFUSÉ sans nommer le droit qui, lui, collecte — le relevé de capacités et le motif du « non évalué » resteront muets sous l'erreur brute de l'API",
					name, p.Unit)
			}
			if strings.TrimSpace(p.Note) == "" || strings.TrimSpace(p.NoteEn) == "" {
				t.Errorf("%s / %s : refus mesuré sans réserve écrite — dire CE QUE le fournisseur a répondu, pas seulement qu'il a dit non",
					name, p.Unit)
			}
		}
	}
}

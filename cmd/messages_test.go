package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Un message d'erreur qui ne nomme pas ce que l'appelant a tapé lui coûte une
// première hypothèse fausse. Aucun de ces défauts ne déplaçait un verdict — et c'est
// exactement pour cela qu'ils avaient survécu : rien ne rougissait.
//
// Les tests mesurent le BINAIRE, parce que ces messages sont ce que le binaire
// imprime : un harnais qui appellerait la fonction interne éprouverait un chemin que
// personne ne parcourt.

// TestAnErrorNamesWhatTheCallerTyped : chaque refus contient l'argument reçu.
func TestAnErrorNamesWhatTheCallerTyped(t *testing.T) {
	bin := buildPepin(t)
	for _, c := range []struct {
		nom      string
		args     []string
		attendu  []string
		interdit string
	}{
		{
			// `--policy-dir` construisait un `os.DirFS` sans rien vérifier ; la
			// première erreur venait du parcours, qui ne connaît plus que sa racine.
			// Le lecteur cherchait donc dans son répertoire courant une faute qu'il
			// avait faite dans un argument.
			nom:      "policy-dir absent",
			args:     []string{"scan", "outscale", "--terraform", "examples/outscale/terraform/plan.json", "--policy-dir", "/ce-chemin-nexiste-pas"},
			attendu:  []string{"/ce-chemin-nexiste-pas"},
			interdit: "stat .:",
		},
		{
			nom:     "policy-dir qui est un fichier",
			args:    []string{"scan", "outscale", "--terraform", "examples/outscale/terraform/plan.json", "--policy-dir", "go.mod"},
			attendu: []string{"go.mod", "dossier"},
		},
		{
			// Le verbe prend un dossier — son aide le dit — et un fichier n'était pas
			// refusé comme tel : « lecture de . : open . : not a directory ».
			nom:      "provider validate sur un fichier",
			args:     []string{"provider", "validate", "providers/outscale.yaml"},
			attendu:  []string{"providers/outscale.yaml", "dossier"},
			interdit: "lecture de . ",
		},
		{
			nom:     "provider validate sur un dossier absent",
			args:    []string{"provider", "validate", "/ce-chemin-nexiste-pas"},
			attendu: []string{"/ce-chemin-nexiste-pas"},
		},
	} {
		t.Run(c.nom, func(t *testing.T) {
			_, stderr := runPepin(t, bin, []string{"LANG=fr_FR.UTF-8"}, c.args...)
			for _, mot := range c.attendu {
				if !strings.Contains(stderr, mot) {
					t.Errorf("le message ne contient pas %q :\n%s", mot, stderr)
				}
			}
			if c.interdit != "" && strings.Contains(stderr, c.interdit) {
				t.Errorf("le message contient encore %q :\n%s", c.interdit, stderr)
			}
		})
	}
}

// TestKubeconfigSaysItNeedsLive : `--kubeconfig` seul proposait deux sources qui ne
// sont pas celle que l'appelant vient de nommer.
func TestKubeconfigSaysItNeedsLive(t *testing.T) {
	bin := buildPepin(t)
	_, stderr := runPepin(t, bin, []string{"LANG=fr_FR.UTF-8"}, "scan", "kubernetes", "--kubeconfig", "/tmp/kubeconfig")
	if !strings.Contains(stderr, "--live") || !strings.Contains(stderr, "--kubeconfig") {
		t.Errorf("le refus ne relie pas --kubeconfig à --live :\n%s", stderr)
	}
	// L'aide du drapeau doit le dire aussi : lire l'erreur ne devrait pas être le seul
	// moyen d'apprendre la contrainte.
	aide, _ := runPepin(t, bin, []string{"LANG=fr_FR.UTF-8"}, "scan", "--help")
	if !strings.Contains(aide, "exige --live") {
		t.Error("l'aide de --kubeconfig ne dit pas qu'il exige --live")
	}
}

// TestExplainAcceptsTheCodeTheReportPrints : le rapport n'imprime que l'exigence
// SCSL ; la commande n'acceptait que l'identifiant de check. Les deux doivent parler
// les mêmes identifiants, sans quoi `control explain` — la commande qui transforme un
// finding en argument — est inatteignable depuis ce qu'on vient de lire.
func TestExplainAcceptsTheCodeTheReportPrints(t *testing.T) {
	bin := buildPepin(t)
	stdout, stderr := runPepin(t, bin, []string{"LANG=fr_FR.UTF-8"}, "control", "explain", "CLD-STO-1")
	if !strings.Contains(stdout, "objectstorage_bucket_public_access") {
		t.Errorf("CLD-STO-1 n'a pas mené au contrôle qu'elle couvre :\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	// La casse ne doit pas être un obstacle : le rapport imprime en majuscules.
	minuscules, _ := runPepin(t, bin, []string{"LANG=fr_FR.UTF-8"}, "control", "explain", "cld-sto-1")
	if !strings.Contains(minuscules, "objectstorage_bucket_public_access") {
		t.Error("l'exigence en minuscules n'est pas reconnue")
	}
	// L'identifiant de check continue de fonctionner : on n'échange pas une entrée
	// contre une autre.
	check, _ := runPepin(t, bin, []string{"LANG=fr_FR.UTF-8"}, "control", "explain", "objectstorage_bucket_public_access")
	if !strings.Contains(check, "objectstorage_bucket_public_access") {
		t.Error("l'identifiant de check ne fonctionne plus")
	}
}

// TestExplainNamesEveryControlOfASharedRequirement : une exigence couvre parfois
// plusieurs contrôles — c'est ce partage qui fait que le rapport les regroupe. En
// choisir un à la place du lecteur reviendrait à lui en cacher un.
func TestExplainNamesEveryControlOfASharedRequirement(t *testing.T) {
	bin := buildPepin(t)
	stdout, _ := runPepin(t, bin, []string{"LANG=fr_FR.UTF-8"}, "control", "explain", "CLD-NET-4")
	for _, code := range []string{
		"network_securitygroup_default_restrict_traffic",
		"network_securitygroup_unrestricted_egress",
	} {
		if !strings.Contains(stdout, code) {
			t.Errorf("CLD-NET-4 ne nomme pas %s :\n%s", code, stdout)
		}
	}
}

// TestAnUnknownControlSaysWhichFormsAreAccepted : un refus sans forme attendue oblige
// à deviner laquelle des deux on n'a pas utilisée.
func TestAnUnknownControlSaysWhichFormsAreAccepted(t *testing.T) {
	bin := buildPepin(t)
	_, stderr := runPepin(t, bin, []string{"LANG=fr_FR.UTF-8"}, "control", "explain", "PAS-UN-CODE")
	for _, mot := range []string{"identifiant de check", "SCSL"} {
		if !strings.Contains(stderr, mot) {
			t.Errorf("le refus ne mentionne pas %q :\n%s", mot, stderr)
		}
	}
}

// TestAnUnknownRegionIsFlaggedBeforeTheCollectFails : une région mal tapée se payait
// d'une minute de résolutions DNS et de dix-sept unités « service indisponible » à
// lire avant d'en deviner la cause.
//
// C'est un AVERTISSEMENT, pas un refus : la liste appartient au fournisseur, et une
// région ajoutée demain doit rester scannable le jour même. La distinction avec un
// `--gate` inconnu — refusé, lui — est que ce vocabulaire-là est celui de Pépin.
func TestAnUnknownRegionIsFlaggedBeforeTheCollectFails(t *testing.T) {
	bin := buildPepin(t)
	// Sans --live, aucun avertissement : la région ne sert alors à rien.
	_, horsLive := runPepin(t, bin, []string{"LANG=fr_FR.UTF-8"},
		"scan", "outscale", "--terraform", "examples/outscale/terraform/plan.json", "--region", "eu-nowhere-9")
	if strings.Contains(horsLive, "n'est pas au catalogue") {
		t.Errorf("avertissement émis hors --live :\n%s", horsLive)
	}
}

// TestEachInputShapeIsRefusedInTheWrongPosition : les trois formes qu'un appelant
// peut confondre, dans les deux positions.
//
// N'importe quel objet JSON était accepté comme un inventaire VIDE : un plan
// Terraform passé sans `--terraform`, ou un `{}`, rendaient « périmètre vide » et le
// code 3. C'est honnête sur ce qui a été mesuré, et faux sur la CAUSE — l'appelant
// n'a pas un périmètre vide, il a donné le mauvais fichier. L'échange inverse était
// déjà refusé ; les deux sens marchent maintenant du même pas, et la table les tient
// ensemble pour qu'ils ne puissent plus diverger.
func TestEachInputShapeIsRefusedInTheWrongPosition(t *testing.T) {
	bin := buildPepin(t)
	vide := filepath.Join(t.TempDir(), "vide.json")
	if err := os.WriteFile(vide, []byte("{}\n"), 0o600); err != nil {
		t.Fatalf("écriture de la fixture : %v", err)
	}
	for _, c := range []struct {
		nom      string
		args     []string
		refuse   bool
		attendus []string
	}{
		{
			nom:      "un plan en position d'inventaire",
			args:     []string{"scan", "outscale", "examples/outscale/terraform/plan.json"},
			refuse:   true,
			attendus: []string{"plan Terraform", "--terraform"},
		},
		{
			nom:      "un objet vide en position d'inventaire",
			args:     []string{"scan", "outscale", vide},
			refuse:   true,
			attendus: []string{"resources"},
		},
		{
			// L'autre sens, qui refusait déjà : la table le garde en phase.
			nom:      "un inventaire en position de plan",
			args:     []string{"scan", "outscale", "--terraform", "examples/outscale/inventory.json"},
			refuse:   true,
			attendus: []string{"planned_values"},
		},
		{
			// LE CONTRE-EXEMPLE : un vrai inventaire passe. Un refus qui refuserait
			// tout serait vert pour la pire des raisons.
			nom:    "un inventaire en position d'inventaire",
			args:   []string{"scan", "outscale", "examples/outscale/inventory.json"},
			refuse: false,
		},
	} {
		t.Run(c.nom, func(t *testing.T) {
			stdout, stderr := runPepin(t, bin, []string{"LANG=fr_FR.UTF-8"}, c.args...)
			if !c.refuse {
				if strings.Contains(stderr, "erreur :") {
					t.Fatalf("un inventaire valide a été refusé :\n%s", stderr)
				}
				if !strings.Contains(stdout, "Synthèse") {
					t.Errorf("le scan n'a pas produit de rapport :\n%s", stdout)
				}
				return
			}
			if !strings.Contains(stderr, "erreur :") {
				t.Fatalf("aucun refus :\nstdout=%s\nstderr=%s", stdout, stderr)
			}
			for _, mot := range c.attendus {
				if !strings.Contains(stderr, mot) {
					t.Errorf("le refus ne mentionne pas %q :\n%s", mot, stderr)
				}
			}
		})
	}
}

package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Un bundle de preuve est fait pour être remis à un TIERS — un auditeur, un client.
// Le sujet d'un finding MFA était l'adresse e-mail de l'utilisateur, parce que
// `iam_user.username` est mappé depuis `email` : chaque bundle scellé emportait donc
// une donnée personnelle que l'outil y avait mise sans que personne ne le décide.
//
// On ne la supprime pas : un exploitant qui corrige un MFA doit savoir QUI. Seul le
// bundle destiné à sortir du périmètre porte l'identifiant stable à sa place.

const inventaireMFA = `{"format":"pepin-inventory/v9","provider":"scaleway","evaluated_at":"2026-09-10T00:00:00Z","resources":[
 {"provider":"scaleway","type":"iam_user","id":"user-42","name":"jean.dupont@exemple.fr","region":"fr-par",
  "attributes":{"user_id":"user-42","username":"jean.dupont@exemple.fr","mfa_enabled":false}}
]}`

func inventairePersonnel(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "inv.json")
	if err := os.WriteFile(p, []byte(inventaireMFA), 0o600); err != nil {
		t.Fatalf("écriture de la fixture : %v", err)
	}
	return p
}

// TestASharedBundleNamesNobody : rien de ce que le bundle emporte ne nomme la personne.
// Le test cherche l'adresse dans TOUS les fichiers, pas dans les champs qu'on croit
// concernés — c'est ainsi qu'on trouve celui qu'on avait oublié, et il y en avait un :
// le message du finding, dans `evidence.observed`.
func TestASharedBundleNamesNobody(t *testing.T) {
	bin := buildPepin(t)
	dir := filepath.Join(t.TempDir(), "bundle")
	inv := inventairePersonnel(t)
	runPepin(t, bin, nil, "scan", "scaleway", inv, "--seal", dir, "--redact")

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("bundle absent : %v", err)
	}
	var vus int
	for _, e := range entries {
		b, rerr := os.ReadFile(filepath.Join(dir, e.Name())) // #nosec G304 -- répertoire du test.
		if rerr != nil {
			t.Fatalf("lecture de %s : %v", e.Name(), rerr)
		}
		vus++
		if strings.Contains(string(b), "jean.dupont@exemple.fr") {
			t.Errorf("%s emporte l'adresse e-mail : un bundle remis à un tiers ne doit pas nommer une personne", e.Name())
		}
	}
	if vus == 0 {
		t.Fatal("bundle vide : la garde ne mesure rien")
	}
	// Et le finding reste ACTIONNABLE : l'identifiant stable désigne le même
	// utilisateur. Effacer le sujet aurait rendu l'écart inexploitable.
	b, _ := os.ReadFile(filepath.Join(dir, "assessment.json")) // #nosec G304
	if !strings.Contains(string(b), "user-42") {
		t.Error("l'identifiant stable ne remplace pas l'adresse : le finding devient inactionnable")
	}
}

// LE CONTRE-EXEMPLE : sans `--redact`, le rapport local nomme la personne. C'est ce
// qu'un exploitant a besoin de lire, et rien ne quitte sa machine.
func TestTheLocalReportStillNamesThePerson(t *testing.T) {
	bin := buildPepin(t)
	stdout, _ := runPepin(t, bin, []string{"LANG=fr_FR.UTF-8"}, "scan", "scaleway", inventairePersonnel(t), "--format", "json")
	if !strings.Contains(stdout, "jean.dupont@exemple.fr") {
		t.Error("le rapport local ne nomme plus la personne : un exploitant ne saurait plus qui corriger")
	}
}

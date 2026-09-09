package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Le bundle de preuve est la promesse centrale de cet outil : un dossier qu'un tiers
// peut rouvrir et opposer. `verify` déclarait « cohérent en interne » un bundle qui se
// contredisait lui-même — quatre `fail` réécrits en `pass`, avec un manifeste annonçant
// encore `"fail": 4`.
//
// Ce que ces gardes NE prétendent pas : fermer la chaîne. Rien dans le bundle ne peut
// ancrer `checksums.txt`, parce que qui réécrit un fichier réécrit aussi l'ancre — seule
// la signature détachée le peut, et l'outil le dit déjà. Ce qui est corrigé ici, c'est
// que le bundle portait DÉJÀ l'information qui le contredisait, et que personne ne la
// regardait.
//
// Le test est table-driven par MOTIF d'altération : chacun doit rendre non nul.

// sceller produit un bundle neuf dans un dossier jetable.
func sceller(t *testing.T, bin string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "bundle")
	runPepin(t, bin, nil, "scan", "scaleway", "examples/scaleway/inventory.json", "--seal", dir)
	if _, err := os.Stat(filepath.Join(dir, "manifest.json")); err != nil {
		t.Fatalf("scellement raté : %v", err)
	}
	return dir
}

// redigest recalcule l'empreinte d'un fichier dans checksums.txt — ce qu'un
// falsificateur fait naturellement après avoir modifié un artefact.
func redigest(t *testing.T, dir, nom string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, nom)) // #nosec G304 -- dossier jetable du test.
	if err != nil {
		t.Fatalf("lecture de %s : %v", nom, err)
	}
	somme := sha256.Sum256(data)
	cs := filepath.Join(dir, "checksums.txt")
	raw, err := os.ReadFile(cs) // #nosec G304 -- dossier jetable du test.
	if err != nil {
		t.Fatalf("lecture de checksums.txt : %v", err)
	}
	re := regexp.MustCompile(`(?m)^[0-9a-f]{64}(\s+\*?` + regexp.QuoteMeta(nom) + `)$`)
	out := re.ReplaceAllString(string(raw), hex.EncodeToString(somme[:])+"$1")
	if out == string(raw) {
		t.Fatalf("l'empreinte de %s n'a pas été remplacée : le test ne mesurerait rien", nom)
	}
	if err := os.WriteFile(cs, []byte(out), 0o600); err != nil {
		t.Fatalf("écriture de checksums.txt : %v", err)
	}
}

func TestVerifyRefusesEveryTamperingPattern(t *testing.T) {
	bin := buildPepin(t)

	// Référence : un bundle intact se vérifie. Sans ce point, une correction qui
	// refuserait TOUT passerait ce test entier.
	if code := exitCodeOfArgs(t, bin, "verify", sceller(t, bin)); code != 0 {
		t.Fatalf("un bundle intact rend %d : la correction refuse tout", code)
	}

	motifs := []struct {
		nom      string
		pourquoi string
		altere   func(t *testing.T, dir string)
	}{
		{
			nom: "verdicts réécrits, empreinte recalculée",
			pourquoi: "l'altération la plus tentante : c'est celle qui change le verdict. " +
				"Le manifeste continue d'annoncer les `fail` que l'assessment n'a plus.",
			altere: func(t *testing.T, dir string) {
				p := filepath.Join(dir, "assessment.json")
				raw, err := os.ReadFile(p) // #nosec G304 -- dossier jetable du test.
				if err != nil {
					t.Fatalf("lecture : %v", err)
				}
				var doc map[string]any
				if err := json.Unmarshal(raw, &doc); err != nil {
					t.Fatalf("assessment illisible : %v", err)
				}
				res, _ := doc["results"].([]any)
				var mues int
				for _, it := range res {
					r, _ := it.(map[string]any)
					if r != nil && r["status"] == "fail" {
						r["status"] = "pass"
						mues++
					}
				}
				if mues == 0 {
					t.Fatal("aucun `fail` à réécrire : le motif ne mesurerait rien")
				}
				out, _ := json.MarshalIndent(doc, "", "  ")
				if err := os.WriteFile(p, out, 0o600); err != nil {
					t.Fatalf("écriture : %v", err)
				}
				redigest(t, dir, "assessment.json")
			},
		},
		{
			nom:      "un octet ajouté à checksums.txt",
			pourquoi: "corruption pure : rien ne couvre checksums.txt, mais elle se VOIT à la lecture.",
			altere: func(t *testing.T, dir string) {
				p := filepath.Join(dir, "checksums.txt")
				raw, err := os.ReadFile(p) // #nosec G304 -- dossier jetable du test.
				if err != nil {
					t.Fatalf("lecture : %v", err)
				}
				if err := os.WriteFile(p, append(raw, '\n'), 0o600); err != nil {
					t.Fatalf("écriture : %v", err)
				}
			},
		},
		{
			nom:      "résumé du manifeste maquillé",
			pourquoi: "l'attaque symétrique : maquiller le manifeste plutôt que l'assessment.",
			altere: func(t *testing.T, dir string) {
				p := filepath.Join(dir, "manifest.json")
				raw, err := os.ReadFile(p) // #nosec G304 -- dossier jetable du test.
				if err != nil {
					t.Fatalf("lecture : %v", err)
				}
				var man map[string]any
				if err := json.Unmarshal(raw, &man); err != nil {
					t.Fatalf("manifeste illisible : %v", err)
				}
				sum, _ := man["summary"].(map[string]any)
				if sum == nil {
					t.Fatal("manifeste sans résumé : le motif ne mesurerait rien")
				}
				sum["fail"] = 0
				out, _ := json.MarshalIndent(man, "", "  ")
				if err := os.WriteFile(p, out, 0o600); err != nil {
					t.Fatalf("écriture : %v", err)
				}
				redigest(t, dir, "manifest.json")
			},
		},
		{
			nom:      "artefact tronqué, empreinte recalculée mais taille laissée",
			pourquoi: "la taille du manifeste est un second témoin : on pense à l'empreinte, moins à elle.",
			altere: func(t *testing.T, dir string) {
				p := filepath.Join(dir, "input.json")
				raw, err := os.ReadFile(p) // #nosec G304 -- dossier jetable du test.
				if err != nil {
					t.Fatalf("lecture : %v", err)
				}
				if err := os.WriteFile(p, raw[:len(raw)/2], 0o600); err != nil {
					t.Fatalf("écriture : %v", err)
				}
				redigest(t, dir, "input.json")
			},
		},
	}

	for _, m := range motifs {
		t.Run(m.nom, func(t *testing.T) {
			dir := sceller(t, bin)
			m.altere(t, dir)
			if code := exitCodeOfArgs(t, bin, "verify", dir); code == 0 {
				t.Errorf("verify accepte un bundle altéré (%s).\n  %s", m.nom, m.pourquoi)
			}
		})
	}
}

// TestRequireSignatureRefusesAnUnsignedBundle : un appelant qui script `verify`
// recevait 0 pour un bundle que l'outil qualifie lui-même de NON opposable —
// l'avertissement était sur stdout, le code disait « réussi », et l'automatisation lit
// le code.
//
// Le drapeau est OPT-IN, et le test le vérifie DANS LES DEUX SENS : changer le défaut
// ferait échouer, en silence, des chaînes qui passent aujourd'hui.
func TestRequireSignatureRefusesAnUnsignedBundle(t *testing.T) {
	bin := buildPepin(t)
	dir := sceller(t, bin)

	if code := exitCodeOfArgs(t, bin, "verify", dir); code != 0 {
		t.Errorf("sans le drapeau, un bundle non signé rend %d : le défaut a changé", code)
	}
	if code := exitCodeOfArgs(t, bin, "verify", dir, "--require-signature"); code == 0 {
		t.Error("--require-signature accepte un bundle dont aucune signature n'a été vérifiée")
	}
	out, _ := runPepin(t, bin, nil, "verify", dir)
	if !strings.Contains(out, "NON opposable") && !strings.Contains(out, "NOT defensible") {
		t.Errorf("le rendu ne dit pas que le bundle n'est pas opposable :\n%s", out)
	}
}

// exitCodeOfArgs lance le binaire et rend son code : ici, le code EST la mesure, donc
// une sortie non nulle ne doit pas faire échouer le test.
func exitCodeOfArgs(t *testing.T, bin string, args ...string) int {
	t.Helper()
	c := exec.Command(bin, args...)
	c.Dir = repoRoot
	if err := c.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return ee.ExitCode()
		}
		t.Fatalf("exécution de %v : %v", args, err)
	}
	return 0
}

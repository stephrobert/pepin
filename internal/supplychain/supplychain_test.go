package supplychain_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Les deux portes de l'ADR-0016. Elles lisent les workflows comme du texte : c'est
// grossier, et c'est exactement ce qui convient. Un épinglage ou une somme sont des
// propriétés SYNTAXIQUES, et les vérifier sur le texte publié évite d'avoir à faire
// confiance à une abstraction qui pourrait, elle, mentir.

func repoRoot(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..")
}

func workflows(t *testing.T) map[string]string {
	t.Helper()
	dir := filepath.Join(repoRoot(t), ".github", "workflows")
	names, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("lecture des workflows : %v", err)
	}
	out := map[string]string{}
	for _, n := range names {
		if n.IsDir() || !strings.HasSuffix(n.Name(), ".yml") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, n.Name())) //nolint:gosec // chemin du dépôt
		if err != nil {
			t.Fatalf("lecture de %s : %v", n.Name(), err)
		}
		out[n.Name()] = string(b)
	}
	if len(out) == 0 {
		t.Fatal("aucun workflow : la porte ne mesurerait rien")
	}
	return out
}

var usesRe = regexp.MustCompile(`uses:\s*(\S+)`)
var fullSHA = regexp.MustCompile(`@[0-9a-f]{40}$`)

// TestEveryActionIsPinnedToAFullSHA : une étiquette de version se déplace, donc elle
// n'épingle rien. C'est le vecteur le plus documenté de l'écosystème, et le dépôt
// tenait déjà la propriété — 79 sur 79 — sans que rien ne l'empêche de la perdre.
func TestEveryActionIsPinnedToAFullSHA(t *testing.T) {
	var checked int
	for name, body := range workflows(t) {
		for _, m := range usesRe.FindAllStringSubmatch(body, -1) {
			ref := m[1]
			if strings.HasPrefix(ref, "./") {
				continue // action locale du dépôt : rien à épingler
			}
			checked++
			if !fullSHA.MatchString(ref) {
				t.Errorf("%s : %q n'est pas épinglé par un SHA de 40 caractères.\n"+
					"  Une étiquette se déplace : elle ne garantit pas ce qui sera exécuté.", name, ref)
			}
		}
	}
	if checked == 0 {
		t.Fatal("aucune action tierce trouvée : la porte ne mesure rien")
	}
	t.Logf("%d référence(s) d'action vérifiée(s)", checked)
}

// TestEveryPublishedArtefactIsChecksummed est la porte que le SBOM a coûtée.
//
// `checksums.txt` est signé, et une signature sur un condensé de condensés ne couvre
// que ce qui y figure. Un artefact publié hors de ce fichier n'est couvert par
// AUCUN des chemins de vérification documentés — quand bien même il serait attesté
// par ailleurs, ce que l'attestation SBOM fait pour son contenu mais pas pour le
// fichier publié.
//
// La mesure porte sur le texte du workflow : ce qui est téléversé, contre ce qui est
// haché. Toute divergence est une omission, pas une subtilité.
func TestEveryPublishedArtefactIsChecksummed(t *testing.T) {
	body := workflows(t)["release.yml"]
	if body == "" {
		t.Fatal("release.yml introuvable")
	}

	// Les fichiers que la release téléverse, tels que le workflow les nomme.
	published := map[string]bool{}
	for _, m := range regexp.MustCompile(`(?m)^\s+(?:dist/)?([a-z0-9][a-z0-9._-]*\.(?:json|jsonl|txt|bundle))\s*$`).
		FindAllStringSubmatch(body, -1) {
		published[m[1]] = true
	}
	// checksums.txt ne peut pas se contenir lui-même, et le bundle signe checksums.txt.
	delete(published, "checksums.txt")
	delete(published, "checksums.txt.cosign.bundle")
	// La provenance SLSA est une enveloppe DSSE SIGNÉE : elle porte sa propre
	// vérification, et `gh attestation verify` la contrôle sans rien d'autre.
	// L'exiger dans checksums.txt serait un cercle — elle est produite APRÈS les
	// sommes, puisqu'elle atteste les binaires que ces sommes décrivent.
	delete(published, "provenance.intoto.jsonl")

	line := regexp.MustCompile(`sha256sum ([^>]+)> checksums\.txt`).FindStringSubmatch(body)
	if line == nil {
		t.Fatal("aucune commande sha256sum vers checksums.txt : la couverture n'est plus vérifiable")
	}
	hashed := line[1]

	for f := range published {
		if !strings.Contains(hashed, f) {
			t.Errorf("l'artefact %q est publié mais n'entre dans aucune somme de checksums.txt.\n"+
				"  La signature cosign porte sur checksums.txt : ce qui n'y figure pas n'est couvert\n"+
				"  par aucun chemin de vérification documenté (ADR-0016).\n"+
				"  Commande de hachage actuelle : sha256sum %s> checksums.txt", f, hashed)
		}
	}
	t.Logf("artefacts publiés contrôlés : %d", len(published))
}

// installerActions énumère les actions dont le RÔLE est d'installer un outil, avec
// le nom de l'entrée qui en fige la version.
//
// La liste est explicite plutôt que devinée : « cette action installe-t-elle quelque
// chose » n'est pas une question qu'un test peut poser au texte d'un workflow. Une
// action ajoutée ici est une décision ; une action installatrice absente d'ici est
// un trou, et c'est le prix de l'honnêteté de cette porte.
var installerActions = map[string]string{
	"jdx/mise-action":           "version",
	"actions/setup-go":          "go-version-file",
	"actions/setup-python":      "python-version",
	"actions/setup-node":        "node-version",
	"sigstore/cosign-installer": "cosign-release",
}

// TestEveryInstallerActionPinsItsTool ferme le trou que l'ADR-0016 n'avait pas nommé.
//
// Épingler une action par SHA ne fige QUE l'action. Ce qu'elle télécharge ensuite
// reste choisi par l'amont si personne ne le dit — et le 2026-09-08, `mise-action`
// sans `version:` a résolu vers une étiquette dont les binaires n'étaient pas
// publiés. 404, cinq fois, et tout le dépôt à l'arrêt.
//
// Une chaîne ne vaut que son maillon le moins figé.
func TestEveryInstallerActionPinsItsTool(t *testing.T) {
	var checked int
	for name, body := range workflows(t) {
		lines := strings.Split(body, "\n")
		for i, l := range lines {
			m := usesRe.FindStringSubmatch(l)
			if m == nil {
				continue
			}
			action := strings.SplitN(strings.TrimPrefix(m[1], "./"), "@", 2)[0]
			key, isInstaller := installerActions[action]
			if !isInstaller {
				continue
			}
			checked++
			// La version se déclare dans le bloc `with:` de CETTE étape : on lit
			// jusqu'à la prochaine étape, marquée par un tiret de liste.
			var block []string
			for j := i + 1; j < len(lines); j++ {
				if strings.HasPrefix(strings.TrimSpace(lines[j]), "- ") {
					break
				}
				block = append(block, lines[j])
			}
			if !strings.Contains(strings.Join(block, "\n"), key+":") {
				t.Errorf("%s ligne %d : %q installe un outil sans figer sa version (%q manquant).\n"+
					"  Épingler l'action ne fige que l'action ; ce qu'elle télécharge reste choisi\n"+
					"  par l'amont. Voir ADR-0016 et l'incident mise v2026.9.3.",
					name, i+1, action, key)
			}
		}
	}
	if checked == 0 {
		t.Fatal("aucune action installatrice trouvée : la porte ne mesure rien")
	}
	t.Logf("%d action(s) installatrice(s) vérifiée(s)", checked)
}

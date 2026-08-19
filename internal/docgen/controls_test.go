package docgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stephrobert/pepin/referentiel"
)

// TestEveryControlHasItsTwoPages : le catalogue n'a pas de trou. Un contrôle sans page serait
// invisible dans un dossier de 116 fichiers, et c'est exactement l'absence qui fait mentir une
// documentation de couverture.
func TestEveryControlHasItsTwoPages(t *testing.T) {
	pages := generatedControlPages(t)
	for code := range referentiel.All() {
		for _, want := range []string{
			controlsDir + "/" + code + ".md",
			controlsDir + "/" + code + ".fr.md",
		} {
			if _, ok := pages[want]; !ok {
				t.Errorf("%s : page absente du catalogue généré", want)
			}
		}
	}
	for _, want := range []string{controlsDir + "/index.md", controlsDir + "/index.fr.md"} {
		if _, ok := pages[want]; !ok {
			t.Errorf("%s : index absent", want)
		}
	}
}

// TestNoStaleControlPage : le dossier du catalogue est ENTIÈREMENT généré, donc ce qui s'y
// trouve sans y être produit décrit un contrôle que le produit n'a plus. Le test de fraîcheur
// compare le CONTENU de ce qu'il génère : sans ce contrôle-ci, une page orpheline survivrait
// indéfiniment à la suppression de son contrôle.
func TestNoStaleControlPage(t *testing.T) {
	pages := generatedControlPages(t)
	stale, err := stalePages(repoRoot, controlsDir, pages)
	if err != nil {
		t.Fatalf("lecture de %s : %v", controlsDir, err)
	}
	for _, p := range stale {
		t.Errorf("%s : page orpheline (le générateur ne la produit plus) — lancer `mise run gen-docs`", p)
	}
}

// TestTheStalePageCheckWouldCatchAnOrphan : le contrôle ci-dessus ne vaut que s'il SAIT
// échouer. Une page inventée doit être signalée, sans quoi le test mesurerait sa propre panne.
func TestTheStalePageCheckWouldCatchAnOrphan(t *testing.T) {
	orphan := controlsDir + "/controle-qui-nexiste-plus.md"
	full := filepath.Join(repoRoot, orphan)
	if err := os.WriteFile(full, []byte("témoin\n"), 0o600); err != nil {
		t.Fatalf("écriture du témoin : %v", err)
	}
	defer func() { _ = os.Remove(full) }()

	stale, err := stalePages(repoRoot, controlsDir, generatedControlPages(t))
	if err != nil {
		t.Fatalf("lecture de %s : %v", controlsDir, err)
	}
	found := false
	for _, p := range stale {
		if p == orphan {
			found = true
		}
	}
	if !found {
		t.Error("la page témoin n'a pas été signalée comme orpheline : le contrôle ne discrimine plus rien")
	}
}

// TestEveryLinkedRemediationProofExists : une page qui LIE une preuve affirme qu'elle existe.
// La table des preuves est calculée par la même lecture que `mise run check-remediation` ; ce
// test vérifie que le chemin rendu est bien celui d'un fichier ou d'un dossier du dépôt, et
// non une construction plausible.
func TestEveryLinkedRemediationProofExists(t *testing.T) {
	proofs, err := RemediationProofs(repoRoot)
	if err != nil {
		t.Fatalf("preuves de remédiation : %v", err)
	}
	seen := 0
	for provider, byCode := range proofs {
		for code, rel := range byCode {
			if _, serr := os.Stat(filepath.Join(repoRoot, rel)); serr != nil {
				t.Errorf("%s / %s : preuve liée vers %q, introuvable (%v)", provider, code, rel, serr)
			}
			seen++
		}
	}
	if seen == 0 {
		t.Fatal("aucune preuve de remédiation lue : le test ne mesurerait plus rien")
	}
}

// TestExoscaleRemediationCoverageStaysComplete : exoscale est le PREMIER fournisseur dont
// chaque contrôle actif porte sa preuve de remédiation déployable. C'est la condition que
// `mise.toml` posait pour rebrancher une porte sur la couverture des preuves : une porte
// rouge en permanence s'apprend à ignorer, une porte verte qui protège un acquis se défend.
// Ce test est cette porte : ajouter un contrôle exoscale sans déposer sa preuve la casse.
// Les autres fournisseurs restent hors garde tant que leur couverture est partielle.
func TestExoscaleRemediationCoverageStaysComplete(t *testing.T) {
	coverages, err := RemediationCoverages(repoRoot)
	if err != nil {
		t.Fatalf("couverture des remédiations : %v", err)
	}
	var exoscale *RemediationCoverage
	for i := range coverages {
		if coverages[i].Provider == "exoscale" {
			exoscale = &coverages[i]
		}
	}
	if exoscale == nil {
		t.Fatal("aucune couverture lue pour exoscale : le test ne mesurerait plus rien")
	}
	if exoscale.Total == 0 {
		t.Fatal("exoscale ne déclare aucun contrôle actif : le test ne mesurerait plus rien")
	}
	if len(exoscale.Missing) > 0 {
		t.Errorf("exoscale : %d/%d preuves de remédiation, manquantes : %s\n"+
			"déposer un module Terraform autonome sous references/remediation/exoscale/<code>/, "+
			"ou une note <code>.md ancrée sur la documentation officielle",
			exoscale.Covered, exoscale.Total, strings.Join(exoscale.Missing, ", "))
	}
}

// TestDormantControlsAreNamedAsSuch : un contrôle déclaré pour aucun fournisseur ne doit pas
// se lire comme un contrôle couvert. La page le dit, et l'index le range à part.
func TestDormantControlsAreNamedAsSuch(t *testing.T) {
	pages := generatedControlPages(t)
	dormant := 0
	for code, ctl := range referentiel.All() {
		if len(ctl.Fournisseurs) > 0 {
			continue
		}
		dormant++
		page := pages[controlsDir+"/"+code+".md"]
		if !strings.Contains(page, "dormant") {
			t.Errorf("%s : contrôle déclaré pour aucun fournisseur, mais sa page ne le dit pas", code)
		}
		if !strings.Contains(pages[controlsDir+"/index.md"], "["+"`"+code+"`") {
			t.Errorf("%s : contrôle dormant absent de l'index", code)
		}
	}
	if dormant == 0 {
		t.Skip("aucun contrôle dormant au référentiel : rien à vérifier")
	}
}

// generatedControlPages rend les pages du catalogue telles que le générateur les produit
// aujourd'hui. Le binaire n'est pas requis : le catalogue est dérivé du référentiel et des
// descripteurs, jamais d'une capture.
func generatedControlPages(t *testing.T) map[string]string {
	t.Helper()
	proofs, err := RemediationProofs(repoRoot)
	if err != nil {
		t.Fatalf("preuves de remédiation : %v", err)
	}
	families, err := familyOrder()
	if err != nil {
		t.Fatalf("familles du référentiel : %v", err)
	}
	out := map[string]string{}
	for _, lang := range []string{"fr", "en"} {
		m, merr := BuildMatrix(repoRoot, lang)
		if merr != nil {
			t.Fatalf("matrice (%s) : %v", lang, merr)
		}
		for page, body := range controlPages(lang, m, proofs, families) {
			out[page] = body
		}
	}
	return out
}

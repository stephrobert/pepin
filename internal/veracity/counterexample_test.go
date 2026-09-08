package veracity_test

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stephrobert/pepin/internal/veracity"
	"github.com/stephrobert/pepin/referentiel"
)

// Le contrat de véracité prouve « Pépin sait-il produire le verdict attendu ? ».
// Il ne prouvait pas « REFUSE-t-il de le produire sur un cas presque identique mais
// LÉGITIME ? » — et c'est cette moitié-là qui fait la précision.
//
// Un contrôle `high`/`critical` qui n'a jamais été éprouvé sur une configuration
// correcte n'a pas de précision mesurée : il a une sensibilité mesurée, ce qui n'est
// pas la même chose. Une règle qui se déclenche sur tout est parfaitement sensible.
//
// Un CONTRE-EXEMPLE est donc un couple, dans un même fichier de scénario : un cas
// `fail` et un cas `pass` sur le MÊME chemin contrôle × fournisseur × source. Le
// second doit être *proche* du premier — même type de ressource, configuration
// légitime — sans quoi il ne prouve rien de la frontière que la règle trace.
//
// La porte ne peut pas mesurer cette proximité, et elle ne prétend pas le faire :
// c'est la revue qui la juge. Ce qu'elle mesure, c'est qu'aucune règle `high` ou
// `critical` ne reste sans couple, et que le compte des manquantes soit ÉCRIT.
//
// contreExempleLedger — le registre des contrôles `high`/`critical` encore sans
// contre-exemple. Le fichier est régénéré (`mise run counterexamples-update`), jamais
// édité à la main, et la porte est exacte dans les DEUX sens : une ligne de trop est
// une dette inventée, une ligne manquante un contrôle ajouté sans sa preuve.
//
// Pourquoi un registre plutôt qu'une porte stricte tout de suite : 39 des 42
// contrôles concernés n'ont pas encore leur couple. Les écrire par gabarit
// produirait exactement le faux vert que l'ADR-0010 refuse — « une matrice engendrée
// serait verte parce que creuse ». Le chiffre laid est l'information, et il descend
// à chaque contre-exemple réellement écrit.
const contreExempleLedger = "testdata/counterexamples-debt.txt"

// couples rend les contrôles portant un couple `fail`+`pass` sur un même chemin.
//
// La porte et la CARTE DE QUALITÉ lisent la même fonction (veracity.CounterexamplePairs) :
// deux calculs de couverture divergent toujours, et celui qui diverge est celui qu'on
// publie. C'est pourquoi ce calcul a quitté ce fichier de test pour le paquet.
func couples(t *testing.T) map[string]bool {
	t.Helper()
	files, err := veracity.LoadScenarios(scenarioDir)
	if err != nil {
		t.Fatalf("chargement des scénarios : %v", err)
	}
	return veracity.CounterexamplePairs(files)
}

// highSeverityControls rend les contrôles ACTIFS de sévérité `high` ou `critical`.
// Un contrôle dormant (déclaré pour aucun fournisseur) n'est jamais évalué : exiger
// sa preuve reviendrait à mesurer une intention.
func highSeverityControls() []string {
	var out []string
	for code, c := range referentiel.All() {
		if len(c.Fournisseurs) == 0 {
			continue
		}
		if c.Severite == "high" || c.Severite == "critical" {
			out = append(out, code)
		}
	}
	sort.Strings(out)
	return out
}

// sansContreExemple rend les contrôles `high`/`critical` qui n'ont pas de couple
// fail+pass, dans l'ordre.
func sansContreExemple(t *testing.T) []string {
	t.Helper()
	return veracity.WithoutCounterexample(highSeverityControls(), couples(t))
}

// TestEveryHighSeverityControlHasALegitimateCounterexample est la porte de l'issue
// #115. Elle est exacte dans les deux sens, comme celle de la dette de véracité.
func TestEveryHighSeverityControlHasALegitimateCounterexample(t *testing.T) {
	tous := highSeverityControls()
	if len(tous) == 0 {
		t.Fatal("aucun contrôle high/critical actif : la porte ne mesure rien")
	}
	manquants := sansContreExemple(t)

	// La régénération est EXPLICITE, jamais automatique : un registre que le test
	// réécrirait seul cesserait de dire quoi que ce soit, et la dette grandirait
	// sans que personne ne la voie grandir.
	if os.Getenv("PEPIN_UPDATE_COUNTEREXAMPLES") != "" {
		ecrireRegistre(t, tous, manquants)
	}

	consigne, err := chargerRegistre(contreExempleLedger)
	if err != nil {
		t.Fatalf("chargement du registre : %v", err)
	}
	inConsigne := map[string]bool{}
	for _, c := range consigne {
		inConsigne[c] = true
	}
	inManquants := map[string]bool{}
	for _, c := range manquants {
		inManquants[c] = true
		if !inConsigne[c] {
			t.Errorf("contrôle %q (high/critical) : aucun contre-exemple légitime.\n"+
				"  Il n'a jamais été éprouvé sur une configuration CORRECTE : sa précision\n"+
				"  n'est pas mesurée. Écrire dans %s un scénario portant un cas `fail` ET un\n"+
				"  cas `pass` proche, ou consigner la dette (mise run counterexamples-update).",
				c, scenarioDir)
		}
	}
	for _, c := range consigne {
		if !inManquants[c] {
			t.Errorf("contrôle %q : consigné comme sans contre-exemple, alors qu'il en a un.\n"+
				"  Une dette payée qui reste écrite masque la suivante. Régénérer le registre.", c)
		}
	}
	t.Logf("%d contrôle(s) high/critical, %d avec contre-exemple, %d en dette",
		len(tous), len(tous)-len(manquants), len(manquants))
}

// TestNoCounterexampleIsASelfConfirmation : un couple doit porter ses deux cas sur le
// MÊME chemin (contrôle × fournisseur × source).
//
// Sans cette garde, un `fail` sur un plan Terraform et un `pass` sur une collecte
// live compteraient comme un couple, alors qu'ils n'éprouvent pas la même chaîne :
// la règle pourrait se taire en live pour une raison entièrement étrangère à la
// configuration légitime qu'on croit avoir prouvée.
func TestNoCounterexampleIsASelfConfirmation(t *testing.T) {
	files, err := veracity.LoadScenarios(scenarioDir)
	if err != nil {
		t.Fatalf("chargement des scénarios : %v", err)
	}
	surLeChemin := map[string]map[veracity.Verdict]bool{}
	for _, f := range files {
		cle := f.Control + "|" + f.Provider + "|" + f.Source
		if surLeChemin[cle] == nil {
			surLeChemin[cle] = map[veracity.Verdict]bool{}
		}
		for _, s := range f.Scenarios {
			surLeChemin[cle][s.Expect] = true
		}
	}
	horsRegistre := map[string]bool{}
	for _, c := range sansContreExemple(t) {
		horsRegistre[c] = true
	}
	var couples int
	for _, code := range highSeverityControls() {
		if horsRegistre[code] {
			continue // en dette : rien à vérifier
		}
		ok := false
		for cle, v := range surLeChemin {
			if strings.HasPrefix(cle, code+"|") &&
				v[veracity.Verdict("fail")] && v[veracity.Verdict("pass")] {
				ok = true
				break
			}
		}
		if !ok {
			t.Errorf("contrôle %q : ses cas `fail` et `pass` sont sur des chemins DIFFÉRENTS.\n"+
				"  Un couple réparti sur deux sources n'éprouve pas la même chaîne : la règle\n"+
				"  peut se taire dans l'une pour une raison étrangère à la configuration\n"+
				"  légitime. Porter les deux cas sur le même fournisseur et la même source.", code)
			continue
		}
		couples++
	}
	t.Logf("%d couple(s) fail+pass vérifié(s) sur un chemin unique", couples)
}

func chargerRegistre(path string) ([]string, error) {
	b, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	var out []string
	for _, l := range strings.Split(string(b), "\n") {
		l = strings.TrimSpace(l)
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		out = append(out, l)
	}
	sort.Strings(out)
	return out, nil
}

func ecrireRegistre(t *testing.T, tous, manquants []string) {
	t.Helper()
	var b strings.Builder
	b.WriteString("# Registre des contrôles high/critical SANS contre-exemple légitime.\n")
	b.WriteString("#   RÉGÉNÉRÉ par `mise run counterexamples-update`, jamais édité à la main.\n")
	b.WriteString("#\n")
	b.WriteString("# Un contre-exemple est un COUPLE : sur un même chemin contrôle × fournisseur\n")
	b.WriteString("# × source, un cas `fail` et un cas `pass` proche. Sans lui, la règle a une\n")
	b.WriteString("# sensibilité mesurée et AUCUNE précision mesurée — une règle qui se déclenche\n")
	b.WriteString("# sur tout est parfaitement sensible.\n")
	b.WriteString("#\n")
	b.WriteString("# Une ligne qui disparaît est une dette payée. Une ligne qui apparaît sans\n")
	b.WriteString("# contre-exemple est une règle high/critical ajoutée sans sa preuve de\n")
	b.WriteString("# précision, et la CI la refuse.\n")
	b.WriteString("#\n")
	_, _ = fmt.Fprintf(&b, "# high/critical actifs : %d · avec contre-exemple : %d · restants : %d\n\n",
		len(tous), len(tous)-len(manquants), len(manquants))
	for _, c := range manquants {
		b.WriteString(c + "\n")
	}
	if err := os.WriteFile(contreExempleLedger, []byte(b.String()), 0o600); err != nil {
		t.Fatalf("écriture du registre : %v", err)
	}
	t.Logf("registre régénéré : %s", contreExempleLedger)
}

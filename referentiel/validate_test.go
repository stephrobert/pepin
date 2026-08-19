package referentiel

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"unicode"

	yaml "go.yaml.in/yaml/v3"
)

// rulesDir : règles communes (relatif au package referentiel).
const rulesDir = "../internal/commonrules/rules"

// frameworkExigences : index SCSL (dépôt frère, comme ../scankit). Absent =
// vérification SCSL ignorée (environnement sans le framework).
const frameworkExigences = "../../framework-scsl/api/v1/exigences.json"

// readRules concatène le texte de toutes les règles .rego (hors tests).
func readRules(t *testing.T) string {
	t.Helper()
	entries, err := os.ReadDir(rulesDir)
	if err != nil {
		t.Fatalf("lecture de %s : %v", rulesDir, err)
	}
	var sb strings.Builder
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".rego") || strings.HasSuffix(e.Name(), "_test.rego") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(rulesDir, e.Name()))
		if err != nil {
			t.Fatalf("lecture de %s : %v", e.Name(), err)
		}
		sb.Write(b)
		sb.WriteByte('\n')
	}
	return sb.String()
}

// TestActiveControlsHaveRule : tout contrôle actif (au moins un fournisseur) doit
// être émis par une règle (le code apparaît en littéral dans un .rego commun).
func TestActiveControlsHaveRule(t *testing.T) {
	rules := readRules(t)
	for code, ctl := range byCode {
		if len(ctl.Fournisseurs) == 0 {
			continue
		}
		if !strings.Contains(rules, `"`+code+`"`) {
			t.Errorf("contrôle actif %q (fournisseurs %v) : aucune règle ne l'émet", code, ctl.Fournisseurs)
		}
	}
}

// TestRuleCodesAreControlled : tout code émis par une règle doit exister dans
// controles.yaml (pas de finding orphelin sans contrôle/SCSL).
func TestRuleCodesAreControlled(t *testing.T) {
	rules := readRules(t)
	// `code` peut être passé via "code": "X" OU en argument littéral d'un
	// constructeur de finding : on extrait tout littéral ressemblant à un code.
	re := regexp.MustCompile(`"([a-z][a-z0-9]+(?:_[a-z0-9]+){2,})"`)
	for _, m := range re.FindAllStringSubmatch(rules, -1) {
		cand := m[1]
		// Filtrer : un code de finding a un préfixe de service connu.
		if !hasServicePrefix(cand) {
			continue
		}
		if _, ok := byCode[cand]; !ok {
			t.Errorf("règle émet le code %q absent de controles.yaml", cand)
		}
	}
}

func hasServicePrefix(s string) bool {
	for _, p := range []string{"network_", "compute_", "objectstorage_", "blockstorage_", "iam_", "kubernetes_", "loadbalancer_", "governance_"} {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

// TestSCSLReferencesExist : toute exigence scsl (CLD-*) d'un contrôle existe dans
// l'index SCSL du framework (anti-invention, §2 / discipline ancrage-scsl-r).
func TestSCSLReferencesExist(t *testing.T) {
	raw, err := os.ReadFile(frameworkExigences)
	if err != nil {
		t.Skipf("index SCSL indisponible (%s) — vérification ignorée", frameworkExigences)
	}
	known := map[string]bool{}
	for _, m := range regexp.MustCompile(`CLD-[A-Z0-9]+-[0-9]+`).FindAllString(string(raw), -1) {
		known[m] = true
	}
	for code, ctl := range byCode {
		for _, s := range ctl.Scsl {
			if !known[s] {
				t.Errorf("contrôle %q référence une exigence SCSL inconnue : %s", code, s)
			}
		}
	}
}

// TestCatalogueConsistency : tout code marqué `implemente` au catalogue existe
// bien dans controles.yaml.
func TestCatalogueConsistency(t *testing.T) {
	raw, err := os.ReadFile("catalogue.yaml")
	if err != nil {
		t.Fatalf("lecture catalogue.yaml : %v", err)
	}
	var cat struct {
		Catalogue []struct {
			Code     string `yaml:"code"`
			Statut   string `yaml:"statut"`
			Severite string `yaml:"severite"`
		} `yaml:"catalogue"`
	}
	if err := yaml.Unmarshal(raw, &cat); err != nil {
		t.Fatalf("catalogue.yaml invalide : %v", err)
	}
	for _, e := range cat.Catalogue {
		if e.Statut != "implemente" {
			continue
		}
		ctl, ok := byCode[e.Code]
		if !ok {
			t.Errorf("catalogue : %q marqué implemente mais absent de controles.yaml", e.Code)
			continue
		}
		if e.Severite != "" && e.Severite != ctl.Severite {
			t.Errorf("catalogue : sévérité de %q = %s, diverge de controles.yaml (%s)", e.Code, e.Severite, ctl.Severite)
		}
	}
}

// TestRegoSeverityMatchesReferentiel : la sévérité émise par une règle (.rego) doit
// être IDENTIQUE à la sévérité du contrôle dans controles.yaml. Les deux existent
// (le .rego alimente scankit/engine, le référentiel alimente scoring/assess) : une
// divergence ment au lecteur d'un des deux rapports. Ce test fige l'unicité de fait.
func TestRegoSeverityMatchesReferentiel(t *testing.T) {
	entries, err := os.ReadDir(rulesDir)
	if err != nil {
		t.Fatalf("lecture de %s : %v", rulesDir, err)
	}
	reCode := regexp.MustCompile(`"code":\s*"([a-z0-9_]+)"`)
	reSev := regexp.MustCompile(`"severity":\s*"(critical|high|medium|low)"`)
	// Forme helper positionnelle : _xxx(r, "code", "severity", …).
	reHelper := regexp.MustCompile(`\(r,\s*"([a-z0-9_]+)",\s*"(critical|high|medium|low)"`)

	sevOf := map[string]string{}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".rego") || strings.HasSuffix(e.Name(), "_test.rego") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(rulesDir, e.Name()))
		if err != nil {
			t.Fatalf("lecture de %s : %v", e.Name(), err)
		}
		text := string(b)
		for _, m := range reHelper.FindAllStringSubmatch(text, -1) {
			sevOf[m[1]] = m[2]
		}
		// Forme objet : apparier chaque "code" avec la "severity" suivante (même
		// objet finding ; le code précède toujours la sévérité dans un objet).
		codeLocs := reCode.FindAllStringSubmatchIndex(text, -1)
		sevLocs := reSev.FindAllStringSubmatchIndex(text, -1)
		for _, c := range codeLocs {
			code := text[c[2]:c[3]]
			for _, s := range sevLocs {
				if s[0] > c[1] {
					sev := text[s[2]:s[3]]
					// Une règle peut émettre PLUSIEURS sévérités pour un même code
					// (governance_resource_region_in_eu : low en zone de confiance,
					// medium si la région n'est pas cataloguée, high hors UE). Le
					// référentiel déclare la sévérité MAXIMALE du contrôle : c'est
					// elle qui pilote la porte de CI, donc c'est elle qu'on compare.
					if rank(sev) > rank(sevOf[code]) {
						sevOf[code] = sev
					}
					break
				}
			}
		}
	}
	for code, sev := range sevOf {
		ctl, ok := byCode[code]
		if !ok {
			continue // couvert par TestRuleCodesAreControlled
		}
		if ctl.Severite != sev {
			t.Errorf("sévérité divergente %q : rego=%s, controles.yaml=%s", code, sev, ctl.Severite)
		}
	}
}

// rank ordonne les sévérités pour retenir la plus forte qu'une règle émet.
func rank(sev string) int {
	switch sev {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

// frameworksDir : catalogues de normes (une vue par norme), relatif au package.
const frameworksDir = "frameworks"

// loadFrameworkIDs charge frameworks/*.yaml et retourne, par code de norme, l'ensemble
// des identifiants d'exigence définis (repris VERBATIM du texte officiel).
func loadFrameworkIDs(t *testing.T) map[string]map[string]bool {
	t.Helper()
	entries, err := os.ReadDir(frameworksDir)
	if err != nil {
		t.Fatalf("lecture de %s : %v", frameworksDir, err)
	}
	out := map[string]map[string]bool{}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(frameworksDir, e.Name()))
		if err != nil {
			t.Fatalf("lecture de %s : %v", e.Name(), err)
		}
		var fw struct {
			Code     string `yaml:"code"`
			Controls []struct {
				ID string `yaml:"id"`
			} `yaml:"controls"`
		}
		if err := yaml.Unmarshal(b, &fw); err != nil {
			t.Fatalf("%s invalide : %v", e.Name(), err)
		}
		ids := map[string]bool{}
		for _, c := range fw.Controls {
			ids[c.ID] = true
		}
		out[fw.Code] = ids
	}
	return out
}

// TestFrameworkReferencesExist : tout identifiant d'exigence mappé par un contrôle
// (frameworks: {norme: [ids]}) doit être défini dans le catalogue de cette norme
// (frameworks/<norme>.yaml). Anti-invention symétrique de TestSCSLReferencesExist :
// interdit qu'un mapping cite un numéro d'exigence qui n'existe pas dans le texte.
// TestEveryFindingCarriesRemediation : tout finding émis porte une remédiation,
// ET les deux langues de Pépin.
//
// CLAUDE.md §3 impose la remédiation, et le rendu l'affiche (terminal, CSV,
// JUnit) : un écart sans remédiation dit à l'utilisateur que quelque chose ne va
// pas sans lui dire quoi faire, ce qui est la moitié du travail d'un outil de
// posture. Rien ne l'imposait jusqu'ici : le modèle amont
// `scankit/finding.Finding` déclare `Remediation` en `omitempty` : optionnel pour
// la bibliothèque, obligatoire pour Pépin.
//
// Le test porte AUSSI les deux labels de traduction (`message_en`,
// `remediation_en`) : Pépin est bilingue, et une règle qui les omettrait ferait
// basculer une sortie anglaise au français au milieu d'un rapport. Un seul
// parcours, un seul appariement : l'invariant de langue est de même nature que
// celui de remédiation, il n'appelle pas une deuxième mécanique.
//
// L'ancrage se fait sur "message" et NON sur "code" : deux findings communs
// (_sg_finding, _bucket_public_finding) reçoivent leur code en PARAMÈTRE, donc un
// ancrage sur un littéral `"code": "…"` les manquait entièrement : six règles
// d'exposition réseau et le contrôle de bucket public échappaient au contrôle.
// Tout finding porte un message ; c'est lui la borne fiable.
func TestEveryFindingCarriesRemediation(t *testing.T) {
	entries, err := os.ReadDir(rulesDir)
	if err != nil {
		t.Fatalf("lecture de %s : %v", rulesDir, err)
	}
	// `"message":` ne matche pas `"message_en":` (le deux-points diffère de place) :
	// les bornes d'objet restent les messages français.
	reMsg := regexp.MustCompile(`"message":\s*(?:"([^"]*)"|sprintf\()`)
	// Champs exigés dans le MÊME objet finding, avec leur longueur minimale utile.
	champs := []struct {
		nom string
		re  *regexp.Regexp
		min int
	}{
		{"remediation", regexp.MustCompile(`"remediation":\s*(?:"([^"]*)"|sprintf\()`), 10},
		{"message_en", regexp.MustCompile(`"message_en":\s*(?:"([^"]*)"|sprintf\()`), 10},
		{"remediation_en", regexp.MustCompile(`"remediation_en":\s*(?:"([^"]*)"|sprintf\()`), 10},
	}

	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".rego") || strings.HasSuffix(e.Name(), "_test.rego") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(rulesDir, e.Name()))
		if err != nil {
			t.Fatalf("lecture de %s : %v", e.Name(), err)
		}
		text := string(b)
		msgLocs := reMsg.FindAllStringSubmatchIndex(text, -1)
		if len(msgLocs) == 0 {
			continue // fichier sans finding (lib.rego, helpers)
		}
		for _, c := range champs {
			locs := c.re.FindAllStringSubmatchIndex(text, -1)
			for i, m := range msgLocs {
				// Fin de l'objet courant : le début du finding suivant, ou la fin du fichier.
				end := len(text)
				if i+1 < len(msgLocs) {
					end = msgLocs[i+1][0]
				}
				where := findingLabel(text, m[0], e.Name())
				found := false
				for _, r := range locs {
					if r[0] <= m[1] || r[0] >= end {
						continue
					}
					// Groupe 1 renseigné = littéral ; absent = forme sprintf(...), acceptée.
					if r[2] != -1 {
						val := strings.TrimSpace(text[r[2]:r[3]])
						if len(val) < c.min {
							t.Errorf("%s : finding %s porte un %s vide ou trop court pour dire quoi que ce soit : %q",
								e.Name(), where, c.nom, val)
						}
						assertNoAccentedLetter(t, e.Name(), where, c.nom, val)
					}
					found = true
					break
				}
				if !found {
					t.Errorf("%s : finding %s n'a pas de %s — %s", e.Name(), where, c.nom, whyItMatters(c.nom))
				}
			}
		}
	}
}

// whyItMatters dit ce que coûte l'absence du champ, plutôt que de la constater.
func whyItMatters(champ string) string {
	if champ == "remediation" {
		return "l'écart est signalé sans dire quoi faire"
	}
	return "une sortie anglaise basculerait au français en plein rapport"
}

// findingLabel nomme le finding fautif dans un message d'échec : son code quand la
// règle le pose en littéral, sinon la ligne. Les findings communs reçoivent leur
// code en paramètre et n'ont donc pas de littéral à citer.
func findingLabel(text string, pos int, file string) string {
	head := text[:pos]
	if i := strings.LastIndex(head, `"code": "`); i >= 0 {
		rest := head[i+len(`"code": "`):]
		if j := strings.Index(rest, `"`); j >= 0 && !strings.Contains(rest[:j], "\n") {
			return strconv.Quote(rest[:j])
		}
	}
	return fmt.Sprintf("%s:%d", file, strings.Count(head, "\n")+1)
}

// assertNoAccentedLetter refuse une lettre non ASCII dans un littéral ANGLAIS.
//
// C'est le critère d'acceptation « LANG=en ne produit aucun caractère accenté »,
// vérifié à la SOURCE. Seules les LETTRES comptent : « — », « … » et « · » sont de
// la ponctuation légitime en anglais, et les noms de ressources n'apparaissent
// jamais dans un littéral (ils arrivent par les arguments de sprintf).
func assertNoAccentedLetter(t *testing.T, file, where, champ, val string) {
	t.Helper()
	if !strings.HasSuffix(champ, "_en") {
		return
	}
	for _, r := range val {
		if r > unicode.MaxASCII && unicode.IsLetter(r) {
			t.Errorf("%s : finding %s, %s : lettre non ASCII %q dans %q", file, where, champ, r, val)
			return
		}
	}
}

func TestFrameworkReferencesExist(t *testing.T) {
	defined := loadFrameworkIDs(t)
	for code, ctl := range byCode {
		for fw, ids := range ctl.Frameworks {
			known, ok := defined[fw]
			if !ok {
				t.Errorf("contrôle %q mappe une norme inconnue (pas de frameworks/%s.yaml) : %s", code, fw, fw)
				continue
			}
			for _, id := range ids {
				if !known[id] {
					t.Errorf("contrôle %q référence %s %q, absent du catalogue frameworks/%s.yaml", code, fw, id, fw)
				}
			}
		}
	}
}

// TestEveryControlIsBilingual : tout contrôle porte ses trois champs anglais.
//
// Pépin est bilingue et la langue est DÉTECTÉE : un lecteur anglophone reçoit
// `titre_en`, `description_en`, `remediation_en`. Le rendu dégrade proprement
// vers le français quand la traduction manque (referentiel.Control.TitreIn), et
// c'est précisément ce qu'il ne faut PAS découvrir en production : une sortie
// anglaise qui bascule au français au milieu d'un tableau est le défaut que
// l'internationalisation vient corriger. La porte est donc ici, en CI.
//
// Le seuil de longueur n'est pas cosmétique : « N/A », « TODO » ou une chaîne
// vide passeraient une simple vérification de présence tout en ne disant rien.
func TestEveryControlIsBilingual(t *testing.T) {
	const minProse = 20 // description/remédiation : une phrase actionnable, pas un marqueur
	for code, ctl := range byCode {
		champs := []struct {
			nom, fr, en string
			min         int
		}{
			{"titre_en", ctl.Titre, ctl.TitreEn, 5},
			{"description_en", ctl.Description, ctl.DescriptionEn, minProse},
			{"remediation_en", ctl.Remediation, ctl.RemediationEn, minProse},
		}
		for _, c := range champs {
			if strings.TrimSpace(c.fr) == "" {
				t.Errorf("contrôle %q : champ français de %s vide — le français est la langue de référence", code, c.nom)
			}
			en := strings.TrimSpace(c.en)
			if en == "" {
				t.Errorf("contrôle %q : %s absent — un contrôle sans traduction ferait basculer la sortie anglaise au français", code, c.nom)
				continue
			}
			if len(en) < c.min {
				t.Errorf("contrôle %q : %s trop court (%d octets) pour dire quoi que ce soit d'actionnable : %q", code, c.nom, len(en), en)
			}
			if en == strings.TrimSpace(c.fr) {
				t.Errorf("contrôle %q : %s est identique au français — la traduction n'a pas été faite", code, c.nom)
			}
		}
	}
}

// TestEnglishControlFieldsCarryNoAccent : les champs anglais ne portent aucune
// lettre accentuée.
//
// C'est le critère d'acceptation « LANG=en ne produit aucun caractère accenté »,
// vérifié à la SOURCE plutôt que sur une sortie : le référentiel alimente le
// titre du tableau, l'evidence de l'assessment et l'OSCAL. Un « é » oublié dans
// un titre traduit se retrouverait dans les quatre formats à la fois.
//
// Seules les LETTRES sont testées : « — », « … » et « · » sont de la ponctuation,
// légitime en anglais, et les noms de ressources ne transitent pas par ce fichier.
func TestEnglishControlFieldsCarryNoAccent(t *testing.T) {
	for code, ctl := range byCode {
		for nom, v := range map[string]string{
			"titre_en":       ctl.TitreEn,
			"description_en": ctl.DescriptionEn,
			"remediation_en": ctl.RemediationEn,
		} {
			for _, r := range v {
				if r > unicode.MaxASCII && unicode.IsLetter(r) {
					t.Errorf("contrôle %q, %s : lettre non ASCII %q dans %q", code, nom, r, v)
					break
				}
			}
		}
	}
}

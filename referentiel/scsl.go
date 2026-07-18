package referentiel

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

// CLDExigence est une exigence du module posture cloud SCSL, lue dans l'API
// statique du framework (`api/v1/exigences.json`, source de vérité). Le champ
// Outillage liste les outils capables de la vérifier ; Pépin n'en retient que ses
// propres références. L'ID est normalisé sans le préfixe `SCSL-` (convention CLD-*
// historique de Pépin) : l'exigence `SCSL-CLD-CHF-4` devient `CLD-CHF-4`.
type CLDExigence struct {
	ID          string   `json:"id"`
	Domaine     string   `json:"domaine"`
	Famille     string   `json:"famille"`
	Niveau      string   `json:"niveau"`
	Devoir      string   `json:"devoir"`
	Texte       string   `json:"texte"`
	Essentielle bool     `json:"essentielle"` // exigence du socle minimal
	Outillage   []string `json:"outillage"`
}

// rePepin extrait le code Pépin d'une entrée d'outillage « Pépin: <code> ».
var rePepin = regexp.MustCompile(`Pépin:\s*([a-z0-9_]+)`)

// ParseCLDExigences lit l'API statique du framework (`api/v1/exigences.json`,
// liste JSON enrichie de toutes les exigences) et retourne celles du module
// posture cloud (domaine CLD). L'ID est dépréfixé jusqu'à `CLD-` pour conserver la
// convention `CLD-*` du référentiel Pépin, quel que soit le préfixe amont du
// framework (`SCSL-` historique, `SOCLE-` actuel) : on coupe à partir de `CLD-`
// plutôt que de retirer un préfixe fixe, sinon un renommage du framework casse
// silencieusement tous les mappings (l'index est passé de `SCSL-CLD-*` à `SOCLE-CLD-*`).
func ParseCLDExigences(path string) ([]CLDExigence, error) {
	raw, err := os.ReadFile(path) // #nosec G304 -- chemin de l'API SCSL fourni par l'appelant, lecture seule.
	if err != nil {
		return nil, fmt.Errorf("lecture de l'index SCSL %s : %w", path, err)
	}
	var items []CLDExigence
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("index SCSL %s invalide : %w", path, err)
	}
	out := make([]CLDExigence, 0, len(items))
	for _, e := range items {
		if e.Domaine != "CLD" || e.Famille == "" {
			continue
		}
		if i := strings.Index(e.ID, "CLD-"); i >= 0 {
			e.ID = e.ID[i:] // "SOCLE-CLD-CHF-1" / "SCSL-CLD-CHF-1" -> "CLD-CHF-1"
		}
		out = append(out, e)
	}
	return out, nil
}

// PepinCodes retourne les codes de contrôle Pépin explicitement cités dans
// l'outillage de l'exigence (« Pépin: <code> »).
func (e CLDExigence) PepinCodes() []string {
	var codes []string
	for _, o := range e.Outillage {
		if m := rePepin.FindStringSubmatch(o); m != nil {
			codes = append(codes, m[1])
		}
	}
	return codes
}

// OutilléeParPepin indique que l'exigence mentionne Pépin dans son outillage,
// donc qu'elle relève (au moins en partie) du périmètre CSPM de Pépin.
func (e CLDExigence) OutilléeParPepin() bool {
	for _, o := range e.Outillage {
		if strings.HasPrefix(o, "Pépin") {
			return true
		}
	}
	return false
}

// Ecart est un point de divergence entre les contrôles Pépin et l'index SCSL.
type Ecart struct {
	Type   string // "desalignement" | "a_implementer" | "non_couverte"
	Code   string // code de contrôle Pépin concerné (si applicable)
	Scsl   string // exigence SCSL concernée
	Detail string
}

// AnalyseCoherence croise les contrôles Pépin (controles.yaml) et les exigences
// CLD pour PILOTER LA ROADMAP. Trois natures d'écart :
//   - desalignement : une exigence cite un code Pépin qui mappe une AUTRE
//     exigence → mapping `scsl` à corriger (incohérence DURE) ;
//   - a_implementer : une exigence cite un code Pépin absent du référentiel ;
//   - non_couverte  : une exigence outillable par Pépin qu'aucun contrôle ne
//     mappe encore (candidate de roadmap).
func AnalyseCoherence(exs []CLDExigence) (desalignements, roadmap []Ecart) {
	ctrls := All()
	mapped := map[string]map[string]bool{} // code -> ensemble des scsl mappés
	covered := map[string]bool{}           // exigence couverte par ≥ 1 contrôle
	for code, c := range ctrls {
		mapped[code] = map[string]bool{}
		for _, s := range c.Scsl {
			mapped[code][s] = true
			covered[s] = true
		}
	}
	for _, e := range exs {
		codes := e.PepinCodes()
		for _, code := range codes {
			if _, ok := ctrls[code]; !ok {
				roadmap = append(roadmap, Ecart{"a_implementer", code, e.ID,
					"exigence cite ce code Pépin, absent de controles.yaml"})
				continue
			}
			if !mapped[code][e.ID] {
				desalignements = append(desalignements, Ecart{"desalignement", code, e.ID,
					fmt.Sprintf("l'exigence cite « %s » mais le contrôle mappe %v", code, sortedKeys(mapped[code]))})
			}
		}
		if len(codes) == 0 && e.OutilléeParPepin() && !covered[e.ID] {
			roadmap = append(roadmap, Ecart{"non_couverte", "", e.ID, e.Texte})
		}
	}
	sortEcarts(desalignements)
	sortEcarts(roadmap)
	return
}

func sortedKeys(m map[string]bool) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

func sortEcarts(e []Ecart) {
	sort.Slice(e, func(i, j int) bool {
		if e[i].Scsl != e[j].Scsl {
			return e[i].Scsl < e[j].Scsl
		}
		return e[i].Code < e[j].Code
	})
}

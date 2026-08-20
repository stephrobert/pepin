// Package referentiel charge le référentiel commun de contrôles de posture
// (controles.yaml, embarqué) et l'expose par code. Source de vérité unique
// (CLAUDE.md §0) : chaque code agnostique y est relié à l'index SCSL (CLD-*) et
// aux frameworks. Le scan s'en sert pour enrichir les findings (id SCSL, lien
// doc) sans coupler les règles Rego à ces mappings.
package referentiel

import (
	_ "embed"
	"fmt"

	yaml "go.yaml.in/yaml/v3"

	"github.com/stephrobert/pepin/internal/i18n"
)

//go:embed controles.yaml
var raw []byte

// Control est un contrôle agnostique du référentiel.
//
// Les champs de PROSE sont bilingues : le français est la langue de référence du
// contenu normatif, `*En` en est la traduction anglaise. Les deux vivent dans le
// MÊME fichier versionné, une source de vérité unique dont `mise run validate`
// exige la complétude (TestEveryControlIsBilingual). Ne pas les lire directement
// pour un affichage : passer par TitreIn/DescriptionIn/RemediationIn, qui portent
// la dégradation vers le français.
type Control struct {
	Code          string              `yaml:"code"`
	Famille       string              `yaml:"famille"`
	Titre         string              `yaml:"titre"`
	TitreEn       string              `yaml:"titre_en"`
	Severite      string              `yaml:"severite"`
	Description   string              `yaml:"description"`
	DescriptionEn string              `yaml:"description_en"`
	Remediation   string              `yaml:"remediation"`
	RemediationEn string              `yaml:"remediation_en"`
	Scsl          []string            `yaml:"scsl"`
	Frameworks    map[string][]string `yaml:"frameworks"`
	Fournisseurs  []string            `yaml:"fournisseurs"`
	// ConfigRequise porte les contraintes de configuration sous lesquelles les
	// correspondances ci-dessus (`Scsl`, `Frameworks`) VALENT. Vide pour un
	// contrôle non réglable, ce qui est le cas de la grande majorité. Voir
	// config.go : une configuration qui s'en écarte fait tomber la correspondance,
	// et le rapport le dit plutôt que de continuer à l'afficher.
	ConfigRequise []ConfigConstraint `yaml:"config_requise"`
}

type catalog struct {
	Controles []Control `yaml:"controles"`
}

var byCode map[string]Control

func init() {
	var c catalog
	if err := yaml.Unmarshal(raw, &c); err != nil {
		panic(fmt.Sprintf("référentiel controles.yaml illisible : %v", err))
	}
	byCode = make(map[string]Control, len(c.Controles))
	for _, ctl := range c.Controles {
		byCode[ctl.Code] = ctl
	}
}

// Lookup retourne le contrôle pour un code agnostique.
func Lookup(code string) (Control, bool) {
	ctl, ok := byCode[code]
	return ctl, ok
}

// All retourne tous les contrôles, indexés par code agnostique (lecture seule).
func All() map[string]Control { return byCode }

// Raw retourne le référentiel embarqué (controles.yaml) tel quel : sévérités, références
// normatives et fournisseurs déterminent le résultat, donc entrent dans la provenance.
func Raw() []byte { return raw }

// TitreIn retourne le titre du contrôle dans la langue demandée. Une traduction
// absente rend le français : le rendu montre toujours quelque chose, et c'est la
// CI qui refuse l'absence, pas l'utilisateur qui découvre un titre vide.
func (c Control) TitreIn(l i18n.Lang) string { return i18n.PickIn(l, c.Titre, c.TitreEn) }

// DescriptionIn retourne la description du contrôle dans la langue demandée.
func (c Control) DescriptionIn(l i18n.Lang) string {
	return i18n.PickIn(l, c.Description, c.DescriptionEn)
}

// RemediationIn retourne la remédiation du contrôle dans la langue demandée.
func (c Control) RemediationIn(l i18n.Lang) string {
	return i18n.PickIn(l, c.Remediation, c.RemediationEn)
}

// SCSL retourne le premier id SCSL (CLD-*) du contrôle, ou "" s'il n'y en a pas.
func (c Control) SCSL() string {
	if len(c.Scsl) == 0 {
		return ""
	}
	return c.Scsl[0]
}

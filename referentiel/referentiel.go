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
)

//go:embed controles.yaml
var raw []byte

// Control est un contrôle agnostique du référentiel.
type Control struct {
	Code         string              `yaml:"code"`
	Famille      string              `yaml:"famille"`
	Titre        string              `yaml:"titre"`
	Severite     string              `yaml:"severite"`
	Description  string              `yaml:"description"`
	Remediation  string              `yaml:"remediation"`
	Scsl         []string            `yaml:"scsl"`
	Frameworks   map[string][]string `yaml:"frameworks"`
	Fournisseurs []string            `yaml:"fournisseurs"`
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

// SCSL retourne le premier id SCSL (CLD-*) du contrôle, ou "" s'il n'y en a pas.
func (c Control) SCSL() string {
	if len(c.Scsl) == 0 {
		return ""
	}
	return c.Scsl[0]
}

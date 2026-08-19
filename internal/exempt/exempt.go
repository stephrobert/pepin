// Package exempt charge, valide et applique les DÉROGATIONS : les écarts qu'une
// organisation assume sciemment, avec une date, une justification et un
// responsable.
//
// Un CSPM utilisé en production doit permettre des exceptions. Sans elles, une
// équipe finit par désactiver le contrôle, ou par ignorer l'outil — deux issues
// pires que l'écart lui-même. Mais une dérogation muette (`ignore: true`) est un
// faux vert sous une autre étiquette : elle rend le rapport moins fiable, pas plus.
//
// D'où quatre propriétés obligatoires, chacune pour une raison :
//
//   - `expires_at`   : sans date, une dérogation devient permanente par oubli ;
//   - `justification`: ce que lira l'auditeur, et la personne qui reprendra le
//     sujet dans un an ;
//   - `owner` et `approved_by` : une dérogation sans responsable n'engage personne.
//
// Ces champs sont validés AU CHARGEMENT, pas au moment de l'appliquer : un fichier
// incomplet arrête le scan, il ne produit pas un rapport qui tait la moitié de ses
// exceptions.
//
// Et la règle qui prime sur tout : **une dérogation écarte, elle ne déclare jamais
// conforme.** Le statut produit est `exempted`, jamais `pass` ; le décompte de
// conformité et les formats analysables gardent l'écart ; seul le code de sortie
// change, et il change vers un code DÉDIÉ qu'un pipeline doit explicitement
// accepter — jamais vers 0.
package exempt

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	yaml "go.yaml.in/yaml/v3"

	"github.com/stephrobert/pepin/internal/i18n"
)

// Date est une date de fin de validité. Type nommé sur `string` pour deux raisons :
// yaml.v3 résout `2026-12-31` en horodatage et refuserait un champ `string` nu, et
// la forme sérialisée doit rester la chaîne écrite par l'auteur (ce que relira un
// humain dans le bundle), pas une représentation reformatée.
type Date string

// UnmarshalYAML lit la valeur scalaire telle qu'elle est écrite, quelle que soit
// la balise que yaml lui aurait résolue.
func (d *Date) UnmarshalYAML(n *yaml.Node) error {
	*d = Date(n.Value)
	return nil
}

// dateLayouts : les formes acceptées pour `expires_at`. La date seule est la forme
// naturelle d'une revue (« ce trimestre »), RFC3339 celle d'un outil.
var dateLayouts = []string{"2006-01-02", time.RFC3339}

// Time convertit la date en instant. Une date seule expire à la FIN du jour : une
// dérogation « jusqu'au 31 décembre » couvre le 31 décembre.
func (d Date) Time() (time.Time, error) {
	s := strings.TrimSpace(string(d))
	for _, layout := range dateLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			if layout == "2006-01-02" {
				return t.Add(24*time.Hour - time.Nanosecond).UTC(), nil
			}
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf(i18n.T(
		"date %q illisible (formes acceptées : 2006-01-02, RFC3339)",
		"unreadable date %q (accepted forms: 2006-01-02, RFC3339)"), s)
}

// Exemption est une dérogation : un contrôle (éventuellement une ressource précise)
// écarté sciemment, jusqu'à une date, par quelqu'un, pour une raison écrite.
type Exemption struct {
	// Control : le code de contrôle agnostique (celui du référentiel commun).
	Control string `yaml:"control" json:"control"`
	// Resource : l'identifiant ou le nom de la ressource visée. Vide = tous les
	// sujets de ce contrôle sur cette cible — une portée large, donc à justifier.
	Resource string `yaml:"resource" json:"resource,omitempty"`
	// Justification : ce que lira l'auditeur.
	Justification string `yaml:"justification" json:"justification"`
	// ExpiresAt : la date au-delà de laquelle la dérogation ne s'applique plus.
	ExpiresAt Date `yaml:"expires_at" json:"expires_at"`
	// Owner : qui porte le risque.
	Owner string `yaml:"owner" json:"owner"`
	// ApprovedBy : qui l'a accepté.
	ApprovedBy string `yaml:"approved_by" json:"approved_by"`
}

// Policy est le contenu d'un fichier de dérogations, versionné et revu comme du code.
type Policy struct {
	// Exemptions porte la clé YAML `exceptions` : c'est le mot du fichier que les
	// équipes écrivent, et le renommer casserait tous les fichiers existants.
	Exemptions []Exemption `yaml:"exceptions" json:"exceptions"`
}

// Digest empreinte la politique NORMALISÉE (entrées triées, forme canonique). Sert
// à prouver quelle politique a été appliquée à un scan, indépendamment de la mise
// en forme du fichier YAML.
func (p Policy) Digest() string {
	keys := make([]string, 0, len(p.Exemptions))
	for _, e := range p.Exemptions {
		keys = append(keys, strings.Join([]string{
			e.Control, e.Resource, string(e.ExpiresAt), e.Owner, e.ApprovedBy, e.Justification,
		}, "\x00"))
	}
	sort.Strings(keys)
	h := sha256.New()
	for _, k := range keys {
		h.Write([]byte(k))
		h.Write([]byte("\n"))
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

// Load lit un fichier de dérogations et VALIDE chaque entrée. Toutes les erreurs
// sont rendues d'un coup : corriger un fichier une ligne par exécution est le genre
// de friction qui fait abandonner le fichier — et revenir au contrôle désactivé.
func Load(path string) (Policy, error) {
	raw, err := os.ReadFile(path) // #nosec G304 -- fichier de dérogations choisi par l'opérateur en option CLI.
	if err != nil {
		return Policy{}, fmt.Errorf(i18n.T("lecture des dérogations %s : %w", "reading the exemptions %s: %w"), path, err)
	}
	var p Policy
	if err := yaml.Unmarshal(raw, &p); err != nil {
		return Policy{}, fmt.Errorf(i18n.T("dérogations %s invalides : %w", "invalid exemptions %s: %w"), path, err)
	}
	if len(p.Exemptions) == 0 {
		return Policy{}, fmt.Errorf(i18n.T(
			"dérogations %s : aucune entrée sous la clé `exceptions`",
			"exemptions %s: no entry under the `exceptions` key"), path)
	}
	var problems []string
	for i, e := range p.Exemptions {
		for _, pb := range e.validate() {
			problems = append(problems, fmt.Sprintf("  exceptions[%d] : %s", i, pb))
		}
	}
	if len(problems) > 0 {
		return Policy{}, fmt.Errorf(i18n.T(
			"dérogations %s : %d champ(s) obligatoire(s) manquant ou invalide —\n%s",
			"exemptions %s: %d mandatory field(s) missing or invalid —\n%s"),
			path, len(problems), strings.Join(problems, "\n"))
	}
	return p, nil
}

// validate rend les problèmes d'une entrée. Une dérogation incomplète n'engage
// personne : le chargement la refuse plutôt que de l'appliquer à moitié.
func (e Exemption) validate() []string {
	var out []string
	for _, f := range []struct{ name, value string }{
		{"control", e.Control},
		{"justification", e.Justification},
		{"owner", e.Owner},
		{"approved_by", e.ApprovedBy},
	} {
		if strings.TrimSpace(f.value) == "" {
			out = append(out, fmt.Sprintf(i18n.T("champ `%s` obligatoire et vide", "mandatory field `%s` is empty"), f.name))
		}
	}
	if strings.TrimSpace(string(e.ExpiresAt)) == "" {
		out = append(out, i18n.T(
			"champ `expires_at` obligatoire et vide (sans date, une dérogation devient permanente par oubli)",
			"mandatory field `expires_at` is empty (without a date an exemption becomes permanent by oversight)"))
	} else if _, err := e.ExpiresAt.Time(); err != nil {
		out = append(out, err.Error())
	}
	return out
}

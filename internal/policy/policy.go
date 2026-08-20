// Package policy charge la POLITIQUE d'un scan : les réglages des contrôles et
// les dérogations, dans UN SEUL fichier.
//
// # Pourquoi un seul fichier
//
// Les dérogations (`--exceptions`, vague 4 lot 2) et les réglages de contrôles
// (ce lot) répondent à la même question — « ce que cette organisation assume » —
// et se relisent au même moment, par les mêmes personnes. Deux fichiers auraient
// deux cycles de revue, et un utilisateur qui doit tenir trois fichiers de
// politique en fera diverger deux. Le fichier porte donc deux sections :
//
//	controls:     # les réglages (étiquetage, fraîcheur des snapshots, secrets)
//	exceptions:   # les dérogations, format INCHANGÉ
//
// `--exceptions` reste accepté et lit exactement le même schéma : un fichier
// existant continue de fonctionner, et peut gagner une section `controls:` sans
// qu'une ligne de ligne de commande ne bouge. Les deux drapeaux sont
// MUTUELLEMENT EXCLUSIFS : deux fichiers de politique, c'est très exactement la
// divergence que cette conception refuse.
//
// # Pourquoi un réglage ne peut pas fabriquer du vert en silence
//
// Chaque réglage est une poignée qui permet d'abaisser une exigence : desserrer
// un seuil, retirer une étiquette exigée, allonger un délai — et continuer
// d'afficher la même correspondance CIS ou SecNumCloud. Un `pass` obtenu en
// abaissant l'exigence est le `PASS` non prouvé que cette vague combat.
//
// D'où deux propriétés, tenues par construction :
//
//  1. La configuration par DÉFAUT reproduit exactement le comportement figé
//     d'avant ce lot. Un réglage qui change le comportement par défaut n'est pas
//     un réglage, c'est un changement de contrôle déguisé.
//  2. Tout écart au défaut qui ROMPT une contrainte normative (referentiel,
//     `config_requise`) est un ASSOUPLISSEMENT : la correspondance normative du
//     contrôle est abandonnée, et l'assouplissement apparaît dans le rendu
//     terminal, dans l'assessment, dans les formats analysables et dans le
//     bundle scellé. Une configuration qui n'apparaît pas dans la preuve est une
//     porte dérobée vers le vert.
//
// # Comment la configuration atteint les règles
//
// Le moteur partagé (scankit/engine) n'expose PAS de document `data` : il ne
// passe que l'`input`. La configuration résolue voyage donc DANS l'input, sous
// la clé `config`, exactement comme `evaluated_at`. Ce n'est pas un pis-aller,
// c'est la bonne place : la configuration est alors scellée dans l'input.json du
// bundle, donc `verify --re-derive` rejoue le même verdict sous la même
// politique, sans qu'on ait à la lui redonner.
package policy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	yaml "go.yaml.in/yaml/v3"

	"github.com/stephrobert/pepin/internal/exempt"
	"github.com/stephrobert/pepin/internal/i18n"
)

// File est le contenu d'un fichier de politique, versionné et revu comme du code.
//
// Les sections sont des POINTEURS : « absente » et « présente mais vide » ne se
// confondent pas. Une section absente laisse le profil par défaut intact ; une
// section présente déclare que son auteur a pris une décision, et cette décision
// est comparée au défaut.
type File struct {
	Controls *Controls `yaml:"controls" json:"controls,omitempty"`
	// Exceptions porte les dérogations, dans le format INCHANGÉ du lot 2 : un
	// fichier `--exceptions` existant est un fichier de politique valide.
	Exceptions []exempt.Exemption `yaml:"exceptions" json:"exceptions,omitempty"`
}

// Controls porte les réglages des contrôles configurables.
type Controls struct {
	Tagging   *Tagging   `yaml:"tagging" json:"tagging,omitempty"`
	Snapshots *Snapshots `yaml:"snapshots" json:"snapshots,omitempty"`
	Secrets   *Secrets   `yaml:"secrets" json:"secrets,omitempty"`
}

// Tagging est la politique d'ÉTIQUETAGE, commune aux deux contrôles qui la lisent
// (`governance_resource_required_tags` et `network_documented`). Une seule notion,
// lue au même endroit : deux implémentations parallèles divergeraient.
//
// Les noms sont LOGIQUES (« owner »), pas littéraux : la comparaison est
// insensible à la casse et aux séparateurs (`cost-center` ≡ `CostCenter`), et les
// alias élargissent ce qu'un nom logique accepte (`owner` ≡ `team`).
type Tagging struct {
	// RequiredTags : les noms logiques exigés sur les ressources FACTURABLES.
	RequiredTags []string `yaml:"required_tags" json:"required_tags,omitempty"`
	// NetworkRequiredTags : les noms logiques exigés sur les RÉSEAUX (cartographie).
	NetworkRequiredTags []string `yaml:"network_required_tags" json:"network_required_tags,omitempty"`
	// Aliases : par nom logique, les autres écritures acceptées.
	Aliases map[string][]string `yaml:"aliases" json:"aliases,omitempty"`
	// ResourceTypes : les types de ressources normalisés sur lesquels l'étiquetage
	// est exigé.
	ResourceTypes []string `yaml:"resource_types" json:"resource_types,omitempty"`
}

// Snapshots règle le contrôle de FRAÎCHEUR des snapshots de volume.
type Snapshots struct {
	// MaxAgeDays : l'âge maximal d'une snapshot pour qu'elle compte. Pointeur :
	// « non écrit » et « écrit à 7 » ne sont pas la même déclaration.
	MaxAgeDays *int `yaml:"max_age_days" json:"max_age_days,omitempty"`
	// AcceptedStates : les états NATIFS d'une snapshot réellement exploitable.
	AcceptedStates []string `yaml:"accepted_states" json:"accepted_states,omitempty"`
}

// Secrets règle la DÉTECTION de secrets dans les données utilisateur.
type Secrets struct {
	// MinConfidence : le niveau de confiance minimal qu'une détection doit porter
	// pour être signalée (`low` | `medium` | `high`). Monter ce seuil TAIT des
	// détections : c'est un assouplissement.
	MinConfidence string `yaml:"min_confidence" json:"min_confidence,omitempty"`
}

// Load lit un fichier de politique et VALIDE ses deux sections. Toutes les erreurs
// sont rendues d'un coup : corriger un fichier une ligne par exécution est le genre
// de friction qui fait abandonner le fichier — et revenir au contrôle désactivé.
func Load(path string) (File, error) {
	raw, err := os.ReadFile(path) // #nosec G304 -- fichier de politique choisi par l'opérateur en option CLI.
	if err != nil {
		return File{}, fmt.Errorf(i18n.T("lecture de la politique %s : %w", "reading the policy %s: %w"), path, err)
	}
	var f File
	if uerr := yaml.Unmarshal(raw, &f); uerr != nil {
		return File{}, fmt.Errorf(i18n.T("politique %s invalide : %w", "invalid policy %s: %w"), path, uerr)
	}
	if f.Controls == nil && len(f.Exceptions) == 0 {
		return File{}, fmt.Errorf(i18n.T(
			"politique %s : aucune entrée sous `controls:` ni sous `exceptions:`",
			"policy %s: no entry under `controls:` nor under `exceptions:`"), path)
	}
	var problems []string
	problems = append(problems, exempt.Problems(f.Exceptions)...)
	problems = append(problems, f.Controls.problems()...)
	if len(problems) > 0 {
		return File{}, fmt.Errorf(i18n.T(
			"politique %s : %d entrée(s) invalide(s) :\n%s",
			"policy %s: %d invalid entry/entries —\n%s"),
			path, len(problems), strings.Join(problems, "\n"))
	}
	return f, nil
}

// Exemptions rend la politique de dérogations portée par le fichier.
func (f File) Exemptions() exempt.Policy { return exempt.Policy{Exemptions: f.Exceptions} }

// problems valide la section `controls:`. Une valeur ininterprétable arrête le
// scan ici : une politique à moitié comprise produirait un rapport dont personne
// ne saurait dire sous quelle exigence il a été rendu.
func (c *Controls) problems() []string {
	if c == nil {
		return nil
	}
	var out []string
	if c.Tagging != nil {
		for _, name := range c.Tagging.RequiredTags {
			if strings.TrimSpace(name) == "" {
				out = append(out, i18n.T("  controls.tagging.required_tags : nom d'étiquette vide",
					"  controls.tagging.required_tags: empty tag name"))
			}
		}
		for _, name := range c.Tagging.NetworkRequiredTags {
			if strings.TrimSpace(name) == "" {
				out = append(out, i18n.T("  controls.tagging.network_required_tags : nom d'étiquette vide",
					"  controls.tagging.network_required_tags: empty tag name"))
			}
		}
		for _, typ := range c.Tagging.ResourceTypes {
			if strings.TrimSpace(typ) == "" {
				out = append(out, i18n.T("  controls.tagging.resource_types : type de ressource vide",
					"  controls.tagging.resource_types: empty resource type"))
			}
		}
	}
	if c.Snapshots != nil {
		if c.Snapshots.MaxAgeDays != nil && *c.Snapshots.MaxAgeDays < 1 {
			out = append(out, fmt.Sprintf(i18n.T(
				"  controls.snapshots.max_age_days : %d — un délai de fraîcheur est un nombre de jours ≥ 1",
				"  controls.snapshots.max_age_days: %d — a freshness window is a number of days ≥ 1"), *c.Snapshots.MaxAgeDays))
		}
		for _, st := range c.Snapshots.AcceptedStates {
			if strings.TrimSpace(st) == "" {
				out = append(out, i18n.T("  controls.snapshots.accepted_states : état vide",
					"  controls.snapshots.accepted_states: empty state"))
			}
		}
	}
	if c.Secrets != nil && c.Secrets.MinConfidence != "" {
		if rankOfConfidence(c.Secrets.MinConfidence) < 0 {
			out = append(out, fmt.Sprintf(i18n.T(
				"  controls.secrets.min_confidence : %q inconnu (valeurs : %s)",
				"  controls.secrets.min_confidence: unknown %q (values: %s)"),
				c.Secrets.MinConfidence, strings.Join(ConfidenceLevels, ", ")))
		}
	}
	return out
}

// Digest empreinte la configuration RÉSOLUE (défauts compris), sous sa forme
// canonique. Sert à prouver sous quelle politique un scan a été rendu,
// indépendamment de la mise en forme du fichier YAML — et à distinguer deux
// scans que seul un réglage sépare.
func (r Resolved) Digest() string {
	b, err := json.Marshal(r)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// normalizeTagKey rend la forme COMPARABLE d'un nom d'étiquette : minuscules, sans
// séparateur. `CostCenter`, `cost-center`, `cost_center` et `Cost Center` s'y
// réduisent tous à `costcenter`.
//
// Une convention d'écriture n'est pas une norme : imposer `CostCenter` à une
// organisation qui écrit `cost-center` produit un FAUX POSITIF, et un outil qui
// crie au loup sur une convention d'écriture finit désactivé.
func normalizeTagKey(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch r {
		case '-', '_', '.', ' ', '/', ':':
			continue
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// sortedUnique rend la liste triée et dédoublonnée : la forme canonique dont
// dépend l'empreinte de la configuration.
func sortedUnique(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, v := range in {
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

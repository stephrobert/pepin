// Package canary porte les RELEVÉS DE CANARI : la seule mesure du dépôt qui
// interroge le vrai plan de contrôle d'un fournisseur souverain.
//
// # Ce que le canari mesure, et pourquoi il n'a pas besoin d'un compte
//
// La colonne « live » de la matrice de couverture est DÉRIVÉE des descripteurs :
// elle dit ce que Pépin croit savoir collecter, jamais ce qu'il a observé. Une
// release qui promeut une capacité live validée sur des fixtures et un émulateur
// promeut une croyance.
//
// Le canari envoie des valeurs SYNTHÉTIQUES, que le fournisseur refuse. Ce qui se
// mesure est le REFUS : un endpoint qui répond 401 ou 403 existe, se résout et
// parle ; un endpoint déplacé répondrait 404, et c'est précisément la régression
// qu'un descripteur ne peut pas voir venir. Aucun identifiant n'entre, donc aucun
// relevé ne peut en porter — c'est une propriété de la chaîne, pas une consigne.
//
// # Ce qu'il n'établit pas, et le dire est la moitié du travail
//
// Un refus de SIGNATURE n'est pas un refus de DROIT. Le canari n'établit ni les
// noms et types des champs du contrat natif, ni ce qu'un tenant contient, ni
// qu'un droit SUFFISANT rende 200. Il ne vaut donc PAS validation live d'un
// contrôle, et la carte de qualité de détection compte zéro de ce côté-là plutôt
// que d'emprunter ce chiffre-ci — un relevé d'accessibilité d'endpoint n'est pas
// une preuve de verdict.
//
// # Pourquoi la fraîcheur se juge au préflight, et pas en CI
//
// Une porte de test qui rougirait avec le temps est une porte qui rougira un
// matin sans que rien n'ait changé, et qu'on désarmera dans la semaine. La
// COMPLÉTUDE (chaque fournisseur cloud a son relevé, lisible et substantiel) se
// vérifie donc en CI, où elle ne dépend pas de la date ; la FRAÎCHEUR se vérifie
// à la qualification de release, dans tools/release/preflight.sh, qui est le seul
// moment où la question « ce relevé est-il encore d'actualité ? » se pose.
package canary

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	yaml "go.yaml.in/yaml/v3"
)

// Dir est le dossier des relevés, relatif à la racine du dépôt.
const Dir = "references/canary"

// MaxAge est la fenêtre de fraîcheur qu'exige la qualification de release.
//
// Quatre-vingt-dix jours : assez long pour qu'un mainteneur ne relance pas le
// canari à chaque correctif, assez court pour qu'un endpoint déplacé ne traverse
// pas deux releases sans être vu. La valeur est ici, et non dans le script, pour
// que le préflight et la documentation citent la MÊME constante.
const MaxAge = 90 * 24 * time.Hour

// Les trois verdicts d'un endpoint, et c'est tout le vocabulaire.
const (
	// Answered : l'endpoint a répondu un statut HTTP. Il existe, il se résout, il parle.
	Answered = "answered"
	// Moved : 404 — le chemin déclaré n'est plus là.
	Moved = "moved"
	// Unreachable : aucune réponse HTTP (DNS, TCP, TLS, proxy). Non concluant.
	Unreachable = "unreachable"
)

// Unit est ce qu'un endpoint déclaré a répondu. Rien d'autre n'y entre : ni corps
// de réponse, ni query string, ni nom de ressource.
type Unit struct {
	Unit       string `yaml:"unit"`
	Verdict    string `yaml:"verdict"`
	Status     int    `yaml:"status"`
	Method     string `yaml:"method"`
	Host       string `yaml:"host"`
	Path       string `yaml:"path"`
	Classified string `yaml:"classified"`
}

// Summary est le décompte par verdict, tel que le relevé le porte.
type Summary struct {
	Answered    int `yaml:"answered"`
	Moved       int `yaml:"moved"`
	Unreachable int `yaml:"unreachable"`
}

// Record est le relevé d'un fournisseur : ce qui a été mesuré, et quand.
type Record struct {
	Path          string  `yaml:"-"` // chemin du fichier, pour les messages d'échec
	Provider      string  `yaml:"provider"`
	Recorded      string  `yaml:"recorded"` // date ISO du jour de la mesure
	PepinVersion  string  `yaml:"pepin_version"`
	Authenticated bool    `yaml:"authenticated"`
	Summary       Summary `yaml:"summary"`
	Units         []Unit  `yaml:"units"`
}

// RecordedOn rend le jour de la mesure.
func (r Record) RecordedOn() (time.Time, error) {
	d, err := time.Parse("2006-01-02", strings.TrimSpace(r.Recorded))
	if err != nil {
		return time.Time{}, fmt.Errorf("date de relevé %q illisible dans %s : %w", r.Recorded, r.Path, err)
	}
	return d, nil
}

// Stale dit si le relevé est plus vieux que la fenêtre de fraîcheur. Une date
// illisible est traitée comme périmée : un relevé qu'on ne sait pas dater ne peut
// pas attester qu'il est récent.
func (r Record) Stale(now time.Time) bool {
	d, err := r.RecordedOn()
	if err != nil {
		return true
	}
	return now.Sub(d) > MaxAge
}

// Substantive dit si le relevé mesure quelque chose. Un relevé sans unité, ou dont
// TOUTES les unités sont injoignables, parle du réseau d'où il a été lancé et non
// du fournisseur : le consigner ferait lire une régression du plan de contrôle là
// où il n'y a qu'un proxy.
func (r Record) Substantive() bool {
	if len(r.Units) == 0 {
		return false
	}
	for _, u := range r.Units {
		if u.Verdict != Unreachable {
			return true
		}
	}
	return false
}

// Load lit tous les relevés sous <root>/references/canary, triés par fournisseur.
func Load(root string) ([]Record, error) {
	base := filepath.Join(root, Dir)
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil, fmt.Errorf("lecture de %s : %w", base, err)
	}
	var out []Record
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		path := filepath.Join(base, e.Name())
		raw, rerr := os.ReadFile(path) // #nosec G304 -- artefact du dépôt, chemin construit depuis un dossier constant.
		if rerr != nil {
			return nil, fmt.Errorf("lecture de %s : %w", path, rerr)
		}
		var r Record
		if uerr := yaml.Unmarshal(raw, &r); uerr != nil {
			return nil, fmt.Errorf("relevé %s invalide : %w", path, uerr)
		}
		r.Path = path
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Provider < out[j].Provider })
	return out, nil
}

// ByProvider indexe les relevés par fournisseur.
func ByProvider(records []Record) map[string]Record {
	out := make(map[string]Record, len(records))
	for _, r := range records {
		out[r.Provider] = r
	}
	return out
}

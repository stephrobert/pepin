package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/stephrobert/pepin/internal/assess"
	"github.com/stephrobert/pepin/internal/exempt"
	"github.com/stephrobert/pepin/internal/policy"
	"github.com/stephrobert/pepin/referentiel"
)

// La POLITIQUE d'un scan : un seul fichier, deux sections.
//
// `--policy` lit `controls:` (les réglages) et `exceptions:` (les dérogations).
// `--exceptions` est le nom historique du MÊME fichier et lit le MÊME schéma :
// une ligne de commande existante ne bouge pas, et un fichier de dérogations
// peut gagner une section `controls:` sans changer d'invocation. Les deux
// drapeaux sont mutuellement exclusifs, parce que deux fichiers de politique
// sont deux fichiers qui divergeront.

// scanConfig porte l'état de la politique pour la durée d'un scan. Un struct
// plutôt que cinq variables de paquet : les trois choses (configuration résolue,
// assouplissements, empreinte) sont dérivées les unes des autres et se lisent
// ensemble.
type scanConfig struct {
	// File : le fichier chargé, s'il y en avait un.
	File *policy.File
	// Resolved : la configuration EFFECTIVE, défauts compris. Toujours renseignée.
	Resolved policy.Resolved
	// Relaxations : par contrôle, les assouplissements qui ont fait tomber ses
	// correspondances normatives. Vide au profil par défaut, par construction.
	Relaxations map[string][]policy.Relaxation
}

// loadPolicy résout la politique du scan depuis les deux drapeaux. Un fichier
// absent n'est pas une erreur : c'est le profil par défaut, et c'est le cas
// courant.
func loadPolicy(policyPath, exceptionsPath string) (scanConfig, exempt.Policy, error) {
	path := policyPath
	if path == "" {
		path = exceptionsPath
	}
	cfg := scanConfig{Resolved: policy.Defaults()}
	if path == "" {
		return cfg, exempt.Policy{}, nil
	}
	f, err := policy.Load(path)
	if err != nil {
		return scanConfig{}, exempt.Policy{}, err
	}
	cfg.File = &f
	cfg.Resolved = policy.Resolve(f.Controls)
	cfg.Relaxations = policy.Relaxations(cfg.Resolved, referentiel.ConfigConstraintsByControl(), controlReferences())
	return cfg, f.Exemptions(), nil
}

// controlReferences rend, par contrôle, ses correspondances normatives sous la
// forme `framework:id`. C'est ce qu'un assouplissement fait PERDRE, et le rapport
// doit pouvoir le nommer plutôt que d'annoncer une perte abstraite.
func controlReferences() map[string][]string {
	out := map[string][]string{}
	for code, c := range referentiel.All() {
		out[code] = assess.ReferencesOf(assess.References(c))
	}
	return out
}

// relaxedControls rend les contrôles assouplis, triés.
func (c scanConfig) relaxedControls() []string {
	out := make([]string, 0, len(c.Relaxations))
	for code := range c.Relaxations {
		out = append(out, code)
	}
	sort.Strings(out)
	return out
}

// droppedReferences rend l'ensemble des correspondances normatives abandonnées,
// triées et dédoublonnées.
func (c scanConfig) droppedReferences() []string {
	seen := map[string]bool{}
	var out []string
	for _, rs := range c.Relaxations {
		for _, r := range rs {
			for _, ref := range r.DroppedReferences {
				if !seen[ref] {
					seen[ref] = true
					out = append(out, ref)
				}
			}
		}
	}
	sort.Strings(out)
	return out
}

// configDocument est le document config.json du bundle : la configuration
// effective, son empreinte, et les assouplissements qu'elle produit. C'est la
// pièce qui permet à un tiers de dire, sans lire le code, sous quelle exigence le
// dossier a été rendu.
type configDocument struct {
	PolicyDigest string              `json:"policy_digest"`
	Effective    policy.Resolved     `json:"effective"`
	Relaxations  []policy.Relaxation `json:"relaxations,omitempty"`
	Notice       map[string]string   `json:"notice,omitempty"`
}

// bundleConfig sérialise la configuration appliquée pour la sceller au bundle.
// Rendue seulement quand un fichier de politique a été fourni : au profil par
// défaut, il n'y a rien à déclarer que l'input.json ne porte déjà.
func bundleConfig(cfg scanConfig) (assess.BundleExtras, error) {
	if cfg.File == nil {
		return assess.BundleExtras{}, nil
	}
	doc := configDocument{PolicyDigest: cfg.Resolved.Digest(), Effective: cfg.Resolved}
	for _, code := range cfg.relaxedControls() {
		doc.Relaxations = append(doc.Relaxations, cfg.Relaxations[code]...)
	}
	if len(doc.Relaxations) > 0 {
		// L'avertissement est écrit DANS le document scellé, et dans les deux
		// langues : un auditeur qui ouvre config.json ne lit pas forcément le
		// terminal de celui qui a lancé le scan.
		doc.Notice = map[string]string{
			"fr": "Configuration assouplie : les correspondances normatives listées ne sont plus tenues par ce scan.",
			"en": "Relaxed configuration: the normative mappings listed here are no longer held by this scan.",
		}
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return assess.BundleExtras{}, fmt.Errorf(tr("sérialisation de la configuration : %w",
			"serializing the configuration: %w"), err)
	}
	return assess.BundleExtras{
		Config: raw,
		ConfigSummary: &assess.ConfigSummary{
			PolicyDigest:      doc.PolicyDigest,
			RelaxedControls:   cfg.relaxedControls(),
			DroppedReferences: cfg.droppedReferences(),
		},
	}, nil
}

// renderRelaxations affiche les assouplissements, en clair et en évidence, juste
// comme les dérogations. Un réglage discret est un réglage que personne ne revoit :
// il porte donc ce qu'il change ET ce qu'il fait perdre.
func renderRelaxations(w io.Writer, cfg scanConfig) {
	codes := cfg.relaxedControls()
	if len(codes) == 0 {
		return
	}
	_, _ = fmt.Fprintf(w, "\n%s\n", exemptStyle.Render(tr(
		"CONFIGURATION ASSOUPLIE : correspondances normatives NON TENUES",
		"RELAXED CONFIGURATION — normative mappings NOT held")))
	for _, code := range codes {
		_, _ = fmt.Fprintf(w, "  · %s\n", code)
		for _, r := range cfg.Relaxations[code] {
			_, _ = fmt.Fprintf(w, "    %s\n", r.Sentence())
			if len(r.DroppedReferences) > 0 {
				_, _ = fmt.Fprintf(w, "    %s %s\n", tr("correspondances abandonnées :",
					"mappings dropped:"), strings.Join(r.DroppedReferences, ", "))
			}
		}
	}
}

// replayedConfig relit la configuration DÉJÀ PRÉSENTE dans un input (rejeu de
// l'input.json d'un bundle scellé). Le dossier a été rendu sous cette
// configuration-là, pas sous celle du jour : la rejouer est ce qui rend
// `verify --re-derive` fidèle. Défensif — l'input peut venir d'un tiers : une
// configuration illisible retombe sur celle du scan courant plutôt que de faire
// paniquer un vérificateur.
func replayedConfig(raw any, fallback policy.Resolved) policy.Resolved {
	b, err := json.Marshal(raw)
	if err != nil {
		return fallback
	}
	var out policy.Resolved
	if err := json.Unmarshal(b, &out); err != nil {
		return fallback
	}
	return out
}

// jsonConfig rend la configuration pour `--format json` : l'empreinte de la
// politique effective et les assouplissements qu'elle produit. Toujours publiée,
// y compris au profil par défaut — un pipeline doit pouvoir vérifier qu'un scan
// a bien tourné sous la configuration attendue, pas seulement constater qu'il
// n'a rien dit.
func jsonConfig(cfg scanConfig) map[string]any {
	out := map[string]any{
		"policy_digest": cfg.Resolved.Digest(),
		"effective":     cfg.Resolved,
	}
	if len(cfg.Relaxations) == 0 {
		return out
	}
	var rs []policy.Relaxation
	for _, code := range cfg.relaxedControls() {
		rs = append(rs, cfg.Relaxations[code]...)
	}
	out["relaxations"] = rs
	out["dropped_references"] = cfg.droppedReferences()
	return out
}

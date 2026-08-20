package docgen

// Les régions générées des pages de référence : la CLI (verbes, drapeaux, aides) et les
// formats de sortie.
//
// Rien n'y est recopié à la main. Les verbes et les drapeaux viennent de la surface GELÉE
// (cmd/testdata/frozen/cli.json), c'est-à-dire de la MÊME source que le test de gel de la
// CLI : une page de référence qui aurait sa propre liste de drapeaux divergerait au premier
// ajout, et c'est exactement ce qu'une « interface publique stable » ne peut pas se permettre.
// Les aides, elles, sont capturées en exécutant `pepin <verbe> --help`, donc dans la langue
// de la page.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// frozenDir est le dossier des surfaces gelées, relatif à la racine du dépôt.
const frozenDir = "cmd/testdata/frozen"

// documentedVerbs : les verbes dont la référence CLI montre l'aide, dans l'ordre où la page
// les présente. La liste est CONTRÔLÉE contre la surface gelée par frozenVerbsAreDocumented :
// un verbe ajouté à la CLI et oublié ici fait échouer la génération, plutôt que de manquer
// silencieusement à la page.
var documentedVerbs = []string{
	"scan", "verify", "provider", "provider list", "provider validate", "provider new",
	"scsl", "version",
}

// blockID rend l'identifiant de région d'une aide de verbe (« provider list » → « cli-help-provider-list »).
func helpBlockID(verb string) string {
	if verb == "" {
		return "cli-help-root"
	}
	return "cli-help-" + strings.ReplaceAll(verb, " ", "-")
}

// cliFrozen est la forme gelée de la CLI : verbes, drapeaux, codes de sortie.
type cliFrozen struct {
	ExitCodes map[string]int `json:"exit_codes"`
	Verbs     map[string]struct {
		Flags map[string]string `json:"flags"` // nom -> raccourci
	} `json:"verbs"`
}

// frozenHistory est l'historique en ajout seul d'une surface gelée.
type frozenHistory struct {
	History []struct {
		SchemaVersion int             `json:"schema_version"`
		Content       json.RawMessage `json:"content"`
	} `json:"history"`
}

// loadFrozen lit la DERNIÈRE entrée d'une surface gelée : sa version et son contenu. C'est
// la promesse courante du dépôt, celle que cmd/frozen_test.go compare au code vivant.
func loadFrozen(root, name string) (int, json.RawMessage, error) {
	path := filepath.Join(root, frozenDir, name+".json")
	raw, err := os.ReadFile(path) // #nosec G304 -- chemin construit depuis une liste constante du paquet.
	if err != nil {
		return 0, nil, fmt.Errorf("lecture de la surface gelée %s : %w", name, err)
	}
	var h frozenHistory
	if err := json.Unmarshal(raw, &h); err != nil {
		return 0, nil, fmt.Errorf("surface gelée %s illisible : %w", name, err)
	}
	if len(h.History) == 0 {
		return 0, nil, fmt.Errorf("surface gelée %s sans historique : rien à documenter", name)
	}
	last := h.History[len(h.History)-1]
	return last.SchemaVersion, last.Content, nil
}

// loadCLISurface rend la surface CLI gelée courante.
func loadCLISurface(root string) (cliFrozen, error) {
	_, content, err := loadFrozen(root, "cli")
	if err != nil {
		return cliFrozen{}, err
	}
	var s cliFrozen
	if err := json.Unmarshal(content, &s); err != nil {
		return cliFrozen{}, fmt.Errorf("surface CLI gelée illisible : %w", err)
	}
	if len(s.Verbs) == 0 || len(s.ExitCodes) == 0 {
		return cliFrozen{}, fmt.Errorf("surface CLI gelée vide : la référence ne mesurerait plus rien")
	}
	return s, nil
}

// frozenVerbsAreDocumented vérifie que la page de référence montre l'aide de CHAQUE verbe de
// la surface gelée. Sans ce contrôle, ajouter un verbe laisserait la référence incomplète
// sans qu'aucun test ne bronche — la documentation « stable » qui ne l'est plus.
func frozenVerbsAreDocumented(s cliFrozen) error {
	documented := map[string]bool{}
	for _, v := range documentedVerbs {
		documented[v] = true
	}
	var missing []string
	for verb := range s.Verbs {
		if !documented[verb] {
			missing = append(missing, verb)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		return fmt.Errorf("verbes présents dans la surface gelée et absents de la référence CLI : %s "+
			"(ajouter à documentedVerbs dans internal/docgen/reference.go et à docs/reference/cli.md)",
			strings.Join(missing, ", "))
	}
	return nil
}

// cliVerbsTable rend les verbes et leurs drapeaux, tels que la surface gelée les promet.
func cliVerbsTable(t refStrings, s cliFrozen) string {
	verbs := make([]string, 0, len(s.Verbs))
	for v := range s.Verbs {
		verbs = append(verbs, v)
	}
	sort.Strings(verbs)
	var b strings.Builder
	b.WriteString("| " + t.colCommand + " | " + t.colFlags + " |\n|---|---|\n")
	for _, v := range verbs {
		flags := s.Verbs[v].Flags
		names := make([]string, 0, len(flags))
		for n := range flags {
			names = append(names, n)
		}
		sort.Strings(names)
		rendered := make([]string, 0, len(names))
		for _, n := range names {
			if short := flags[n]; short != "" {
				rendered = append(rendered, "`--"+n+"` / `-"+short+"`")
				continue
			}
			rendered = append(rendered, "`--"+n+"`")
		}
		cell := t.noFlag
		if len(rendered) > 0 {
			cell = strings.Join(rendered, ", ")
		}
		_, _ = fmt.Fprintf(&b, "| `pepin %s` | %s |\n", v, cell)
	}
	return b.String()
}

// cliExitCodesTable rend les codes de sortie avec le nom de la constante qui les porte dans
// cmd/surface.go : ce que la CLI promet, sous le nom sous lequel elle le promet.
func cliExitCodesTable(t refStrings, s cliFrozen) string {
	type row struct {
		name string
		code int
	}
	rows := make([]row, 0, len(s.ExitCodes))
	for name, code := range s.ExitCodes {
		rows = append(rows, row{name: name, code: code})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].code < rows[j].code })
	var b strings.Builder
	b.WriteString("| " + t.colExit + " | " + t.colConstant + " | " + t.colMeaning + " |\n|:-:|---|---|\n")
	for _, r := range rows {
		_, _ = fmt.Fprintf(&b, "| **%d** | `%s` | %s |\n", r.code, r.name, t.exitMeaning[r.name])
	}
	return b.String()
}

// surfaceVersionsTable rend la version de FORME de chaque surface gelée. C'est le numéro
// qu'un intégrateur épingle : il monte quand les verbes, les drapeaux, les codes de sortie
// ou la forme d'un document parsable bougent, ajout compris.
func surfaceVersionsTable(t refStrings, versions map[string]int) string {
	surfaces := []struct{ name, what string }{
		{"cli", t.surfCLI},
		{"findings", t.surfFindings},
		{"assessment", t.surfAssessment},
		{"bundle", t.surfBundle},
		{"inventory", t.surfInventory},
	}
	var b strings.Builder
	b.WriteString("| " + t.colSurface + " | " + t.colWhat + " | " + t.colVersion + " |\n|---|---|:-:|\n")
	for _, s := range surfaces {
		_, _ = fmt.Fprintf(&b, "| `%s` | %s | **v%d** |\n", s.name, s.what, versions[s.name])
	}
	return b.String()
}

// jsonField extrait un champ d'un document JSON capturé et le rend indenté. Un champ absent
// rend une note explicite : une page doit dire qu'elle n'a pas l'exemple, jamais l'inventer.
func jsonField(c Capture, path ...string) string {
	var doc any
	if err := json.Unmarshal([]byte(c.Stdout), &doc); err != nil {
		return Fence("text", "(sortie JSON illisible : "+err.Error()+")")
	}
	cur := doc
	for _, key := range path {
		switch v := cur.(type) {
		case map[string]any:
			next, ok := v[key]
			if !ok {
				return Fence("text", "(champ « "+strings.Join(path, ".")+" » absent de cette sortie)")
			}
			cur = next
		case []any:
			if key != "0" || len(v) == 0 {
				return Fence("text", "(champ « "+strings.Join(path, ".")+" » absent de cette sortie)")
			}
			cur = v[0]
		default:
			return Fence("text", "(champ « "+strings.Join(path, ".")+" » absent de cette sortie)")
		}
	}
	return Fence("json", mustIndent(cur))
}

// refStrings porte les libellés des régions de référence dans une langue.
type refStrings struct {
	colCommand, colFlags, colExit, colConstant, colMeaning string
	colSurface, colWhat, colVersion                        string
	noFlag                                                 string
	surfCLI, surfFindings, surfAssessment, surfBundle      string
	surfInventory                                          string
	exitMeaning                                            map[string]string
}

func refText(lang string) refStrings {
	if lang == "fr" {
		return refStrings{
			colCommand: "Commande", colFlags: "Drapeaux", colExit: "Code",
			colConstant: "Constante (`cmd/surface.go`)", colMeaning: "Signification",
			colSurface: "Surface", colWhat: "Ce qui est gelé", colVersion: "Version",
			noFlag:         "_(aucun drapeau propre)_",
			surfCLI:        "verbes, drapeaux et codes de sortie",
			surfFindings:   "forme de `--format json` (`findings` + `summary`)",
			surfAssessment: "forme du document `--format assessment`",
			surfBundle:     "forme du bundle de preuve (fichiers, rôles, manifest)",
			surfInventory:  "forme de l'inventaire normalisé (enveloppe, ressource, types et attributs)",
			exitMeaning: map[string]string{
				"conforme":       "aucun écart critical/high, et au moins un contrôle réellement mesuré",
				"non_conformite": "au moins un écart critical ou high",
				"erreur":         "erreur technique : le scan n'a pas pu conclure",
				"strict":         "le scan n'établit pas la conformité : rien n'a été mesuré, ou la collecte n'a pas pu lire tout le périmètre (les deux sans `--strict`), ou écarts medium/low restants avec `--strict`",
				"derogation":     "tout écart critical/high restant est couvert par une dérogation datée et attribuée (`--exceptions`)",
			},
		}
	}
	return refStrings{
		colCommand: "Command", colFlags: "Flags", colExit: "Code",
		colConstant: "Constant (`cmd/surface.go`)", colMeaning: "Meaning",
		colSurface: "Surface", colWhat: "What is frozen", colVersion: "Version",
		noFlag:         "_(no flag of its own)_",
		surfCLI:        "verbs, flags and exit codes",
		surfFindings:   "shape of `--format json` (`findings` + `summary`)",
		surfAssessment: "shape of the `--format assessment` document",
		surfBundle:     "shape of the evidence bundle (files, roles, manifest)",
		surfInventory:  "shape of the normalized inventory (envelope, resource, types and attributes)",
		exitMeaning: map[string]string{
			"conforme":       "no critical/high deviation, and at least one control actually measured",
			"non_conformite": "at least one critical or high deviation",
			"erreur":         "technical error: the scan could not conclude",
			"strict":         "the scan does not establish compliance: nothing was measured, or the collection could not read the whole scope (both without `--strict`), or medium/low deviations remain with `--strict`",
			"derogation":     "every remaining critical/high deviation is covered by a dated, attributed exemption (`--exceptions`)",
		},
	}
}

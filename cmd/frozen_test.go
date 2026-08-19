package cmd

// Ce que les consommateurs de Pépin ont le droit d'intégrer est gelé par un
// test, pas par une phrase de README.
//
// Les surfaces gelées ici sont celles qu'une chaîne de conformité branche :
//
//   - cli         : les verbes, leurs flags, les codes de sortie ;
//   - findings    : la forme de `--format json` ({"findings": ..., "summary": ...}) ;
//   - assessment  : la forme du document assessment (statuts typés, références
//     normatives, provenance), servie par `--format assessment` et scellée
//     dans les bundles ;
//   - bundle      : la forme du bundle de preuve (fichiers, rôles, manifest,
//     chaîne de format versionnée).
//
// Chaque surface a une fixture committée sous testdata/frozen/ : un historique
// d'entrées, chaque entrée une version de schéma et la forme canonique observée
// sous cette version. Ce qui est gelé est la FORME (chemins et types JSON, noms
// de verbes et de flags), jamais une valeur : une fixture qui porterait un
// horodatage ou un compte de findings passerait au rouge à chaque évolution de
// règle et serait désarmée dans la semaine.
//
// Les formes qui viennent de structs Go (assessment, findings, manifest) sont
// dérivées par RÉFLEXION sur le type, pas d'une exécution : une exécution
// pauvre (aucun contrôle not-applicable ce jour-là) ferait disparaître des
// champs omitempty et le gel dépendrait de la richesse des données. Le type,
// lui, est le contrat, et c'est précisément lui qu'une montée de version de
// scankit peut déplacer sans qu'une ligne de Pépin ne change.
//
// Le verrou a deux dents, et les deux sont nécessaires :
//
//   - TestTheFrozenSurfacesStillMatchTheirFixture échoue quand la forme bouge
//     et que la fixture n'a pas bougé : le changement subi ;
//   - TestASurfaceChangeDemandsItsVersionBump échoue quand la fixture a bougé
//     et que la version déclarée par le code n'a pas suivi : le « régénéré
//     sans décider ». La régénération (mise run frozen-update) AJOUTE une
//     entrée à la version suivante et ne réécrit jamais une entrée passée : la
//     seule façon de déplacer une forme sous une version inchangée est d'éditer
//     en place une entrée committée, un diff qu'aucune relecture ne prend pour
//     de la routine.
//
// Un changement délibéré suit donc quatre pas, dans un seul commit : changer le
// code, `mise run frozen-update`, incrémenter la constante correspondante
// (cliSurfaceVersion, findingsSurfaceVersion, assess.AssessmentSurfaceVersion,
// ou le suffixe /vN de assess.BundleFormat), écrire la ligne de CHANGELOG dont
// l'incrément est le signal.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/stephrobert/pepin/internal/assess"
	"github.com/stephrobert/pepin/internal/scoring"
	"github.com/stephrobert/scankit/assessment"
	"github.com/stephrobert/scankit/finding"
)

const frozenDir = "testdata/frozen"

// frozenEntry est une version d'une surface : la version déclarée et la forme
// canonique observée sous cette version.
type frozenEntry struct {
	SchemaVersion int             `json:"schema_version"`
	Content       json.RawMessage `json:"content"`
}

// frozenFixture est l'historique en ajout seul d'une surface. La dernière
// entrée est la promesse courante ; les précédentes sont ce qui rend une
// édition en place visible dans un diff.
type frozenFixture struct {
	History []frozenEntry `json:"history"`
}

// frozenSurface lie un nom à la forme que le code rend aujourd'hui et à la
// version que le code déclare pour elle.
type frozenSurface struct {
	name    string
	version int
	content json.RawMessage
}

// TestTheFrozenSurfacesStillMatchTheirFixture est le gel : la forme vivante de
// chaque surface égale la dernière entrée committée de sa fixture. Rouge sur
// tout changement de forme (clé ajoutée, retirée ou retypée, verbe ou flag
// apparu ou disparu), vert à travers les valeurs.
func TestTheFrozenSurfacesStillMatchTheirFixture(t *testing.T) {
	for _, s := range observedSurfaces(t) {
		t.Run(s.name, func(t *testing.T) {
			fixture := loadFrozen(t, s.name)
			last := fixture.History[len(fixture.History)-1]
			if !sameJSON(t, last.Content, s.content) {
				t.Errorf("la surface gelée %q a bougé :\n  committée : %s\n  observée  : %s\n"+
					"Si le changement est délibéré : `mise run frozen-update`, incrémenter la constante de "+
					"version de la surface (cmd/surface.go, internal/assess), et écrire la ligne de CHANGELOG.",
					s.name, compactJSON(t, last.Content), compactJSON(t, s.content))
			}
		})
	}
}

// TestASurfaceChangeDemandsItsVersionBump est l'autre dent : la version que le
// code déclare égale la version de la dernière entrée de la fixture. Après
// `mise run frozen-update`, ce test reste rouge tant que la constante n'a pas
// bougé : c'est le moment où un humain décide le changement et doit sa ligne de
// CHANGELOG.
func TestASurfaceChangeDemandsItsVersionBump(t *testing.T) {
	for _, s := range observedSurfaces(t) {
		t.Run(s.name, func(t *testing.T) {
			fixture := loadFrozen(t, s.name)
			last := fixture.History[len(fixture.History)-1]
			if s.version != last.SchemaVersion {
				t.Errorf("surface %q : le code déclare la version %d, la dernière entrée de la fixture est %d.\n"+
					"Une surface gelée ne bouge qu'avec sa version : incrémenter la constante et dire "+
					"ce qui change dans le CHANGELOG.",
					s.name, s.version, last.SchemaVersion)
			}
		})
	}
}

// TestFrozenHistoryOnlyGrows refuse une fixture dont les entrées ont été
// réordonnées ou dont les versions se répètent : l'historique est la piste
// d'audit qui rend une édition en place visible, il doit rester strictement
// croissant.
func TestFrozenHistoryOnlyGrows(t *testing.T) {
	for _, s := range observedSurfaces(t) {
		fixture := loadFrozen(t, s.name)
		prev := 0
		for i, e := range fixture.History {
			if e.SchemaVersion <= prev {
				t.Errorf("surface %q : l'entrée %d porte la version %d après %d ; les versions croissent strictement",
					s.name, i, e.SchemaVersion, prev)
			}
			prev = e.SchemaVersion
		}
	}
}

// observedSurfaces rend chaque surface gelée une fois. Quand PEPIN_UPDATE_FROZEN
// est posé, la forme changée est AJOUTÉE à l'historique de la fixture, à la
// version suivante, jamais en réécrivant une entrée, ce qui est ce qui permet
// au test de version d'exiger l'incrément.
func observedSurfaces(t *testing.T) []frozenSurface {
	t.Helper()
	surfaces := []frozenSurface{
		{name: "cli", version: cliSurfaceVersion, content: mustJSON(t, cliSurface())},
		{name: "findings", version: findingsSurfaceVersion, content: mustJSON(t, map[string]any{
			"findings": []any{typeTree(reflect.TypeOf(finding.Finding{}), nil)},
			"summary":  typeTree(reflect.TypeOf(scoring.Result{}), nil),
		})},
		{name: "assessment", version: assess.AssessmentSurfaceVersion,
			content: mustJSON(t, typeTree(reflect.TypeOf(assessment.Assessment{}), nil))},
		{name: "bundle", version: bundleFormatVersion(t), content: mustJSON(t, bundleSurface(t))},
	}
	if os.Getenv("PEPIN_UPDATE_FROZEN") != "" {
		for _, s := range surfaces {
			appendFrozen(t, s)
		}
	}
	return surfaces
}

// cliSurface énumère les verbes, leurs flags (nom -> raccourci) et les codes de
// sortie. Les valeurs derrière les flags et la prose d'aide ne sont pas gelées.
func cliSurface() map[string]any {
	verbs := map[string]any{}
	var walk func(prefix string, c *cobra.Command)
	walk = func(prefix string, c *cobra.Command) {
		for _, sub := range c.Commands() {
			name := sub.Name()
			if name == "help" || name == "completion" {
				continue // verbes implicites de cobra, hors promesse
			}
			path := strings.TrimSpace(prefix + " " + name)
			flags := map[string]string{}
			sub.Flags().VisitAll(func(f *pflag.Flag) {
				if f.Hidden {
					return
				}
				flags[f.Name] = f.Shorthand
			})
			verbs[path] = map[string]any{"flags": flags}
			walk(path, sub)
		}
	}
	walk("", rootCmd)
	return map[string]any{
		"verbs": verbs,
		"exit_codes": map[string]int{
			"conforme":       exitConforme,
			"non_conformite": exitNonConformite,
			"erreur":         exitErreur,
			"strict":         exitStrict,
		},
	}
}

// bundleSurface écrit un vrai bundle dans un dossier jetable et gèle ce qu'un
// vérificateur tiers en lit : la chaîne de format versionnée, la liste des
// fichiers et leurs rôles, et l'arbre de champs du manifest.
func bundleSurface(t *testing.T) map[string]any {
	t.Helper()
	dir := t.TempDir()
	a := assessment.Assessment{Run: assessment.Run{Timestamp: "1970-01-01T00:00:00Z"}}
	if _, err := assess.WriteBundle(dir, a, []byte("{}\n")); err != nil {
		t.Fatalf("écriture du bundle témoin : %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "manifest.json")) // #nosec G304 -- dossier de test.
	if err != nil {
		t.Fatalf("lecture du manifest témoin : %v", err)
	}
	var m assess.Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("manifest témoin invalide : %v", err)
	}
	files := map[string]string{"manifest.json": "manifest", "checksums.txt": "checksums"}
	for _, art := range m.Artifacts {
		files[art.File] = art.Role
	}
	return map[string]any{
		"format":   assess.BundleFormat,
		"files":    files,
		"manifest": typeTree(reflect.TypeOf(assess.Manifest{}), nil),
	}
}

// bundleFormatVersion extrait la version de surface du suffixe /vN de
// assess.BundleFormat : pour le bundle, la constante et le signal sur le fil
// sont la même chose, ce qui est le cas idéal.
func bundleFormatVersion(t *testing.T) int {
	t.Helper()
	i := strings.LastIndex(assess.BundleFormat, "/v")
	if i < 0 {
		t.Fatalf("assess.BundleFormat %q ne porte pas de suffixe /vN", assess.BundleFormat)
	}
	n, err := strconv.Atoi(assess.BundleFormat[i+2:])
	if err != nil {
		t.Fatalf("assess.BundleFormat %q : suffixe de version illisible : %v", assess.BundleFormat, err)
	}
	return n
}

// typeTree dérive l'arbre de champs JSON d'un type Go : les chemins et les
// types, jamais les valeurs. omitempty est ignoré à dessein : le contrat est le
// champ, pas sa présence sur une exécution donnée.
func typeTree(t reflect.Type, seen map[reflect.Type]bool) any {
	if seen == nil {
		seen = map[reflect.Type]bool{}
	}
	switch t.Kind() {
	case reflect.Pointer:
		return typeTree(t.Elem(), seen)
	case reflect.Interface:
		return "any"
	case reflect.String:
		return "string"
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Slice, reflect.Array:
		if t.Elem().Kind() == reflect.Uint8 {
			return "string" // []byte se sérialise en base64
		}
		return []any{typeTree(t.Elem(), seen)}
	case reflect.Map:
		return map[string]any{"*": typeTree(t.Elem(), seen)}
	case reflect.Struct:
		if t == reflect.TypeOf(struct{}{}) {
			return map[string]any{}
		}
		if seen[t] {
			return "cycle"
		}
		seen[t] = true
		defer delete(seen, t)
		out := map[string]any{}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			tag := f.Tag.Get("json")
			name, _, _ := strings.Cut(tag, ",")
			if name == "-" {
				continue
			}
			if f.Anonymous && name == "" {
				// champ embarqué sans tag : ses champs remontent au niveau parent
				if sub, ok := typeTree(f.Type, seen).(map[string]any); ok {
					for k, v := range sub {
						out[k] = v
					}
					continue
				}
			}
			if name == "" {
				name = f.Name
			}
			out[name] = typeTree(f.Type, seen)
		}
		return out
	default:
		return fmt.Sprintf("unhandled:%s", t.Kind())
	}
}

// loadFrozen lit la fixture committée d'une surface, et échoue si elle manque :
// une surface sans fixture n'est pas gelée, elle est promise en l'air.
func loadFrozen(t *testing.T, name string) frozenFixture {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(frozenDir, name+".json")) // #nosec G304 -- fixture du dépôt.
	if err != nil {
		t.Fatalf("fixture gelée %q absente (%v) ; la générer : mise run frozen-update", name, err)
	}
	var f frozenFixture
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("fixture gelée %q invalide : %v", name, err)
	}
	if len(f.History) == 0 {
		t.Fatalf("fixture gelée %q sans historique", name)
	}
	return f
}

// appendFrozen ajoute la forme observée à l'historique de la fixture, à la
// version suivante, si et seulement si elle diffère de la dernière entrée.
// Idempotent : relancer la mise à jour n'ajoute rien de plus.
func appendFrozen(t *testing.T, s frozenSurface) {
	t.Helper()
	path := filepath.Join(frozenDir, s.name+".json")
	var f frozenFixture
	if raw, err := os.ReadFile(path); err == nil { // #nosec G304 -- fixture du dépôt.
		if err := json.Unmarshal(raw, &f); err != nil {
			t.Fatalf("fixture gelée %q invalide : %v", s.name, err)
		}
	}
	next := 1
	if n := len(f.History); n > 0 {
		if sameJSON(t, f.History[n-1].Content, s.content) {
			return
		}
		next = f.History[n-1].SchemaVersion + 1
	}
	f.History = append(f.History, frozenEntry{SchemaVersion: next, Content: s.content})
	if err := os.MkdirAll(frozenDir, 0o750); err != nil {
		t.Fatalf("création de %s : %v", frozenDir, err)
	}
	out, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		t.Fatalf("sérialisation de la fixture %q : %v", s.name, err)
	}
	if err := os.WriteFile(path, append(out, '\n'), 0o600); err != nil {
		t.Fatalf("écriture de la fixture %q : %v", s.name, err)
	}
	t.Logf("fixture %q : entrée ajoutée à la version %d ; incrémenter la constante correspondante", s.name, next)
}

// mustJSON sérialise en JSON canonique (clés triées par encoding/json).
func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("sérialisation de la surface : %v", err)
	}
	return b
}

// sameJSON compare deux documents JSON structurellement, indépendamment de
// l'ordre des clés et de l'indentation.
func sameJSON(t *testing.T, a, b json.RawMessage) bool {
	t.Helper()
	var va, vb any
	if err := json.Unmarshal(a, &va); err != nil {
		t.Fatalf("JSON invalide : %v", err)
	}
	if err := json.Unmarshal(b, &vb); err != nil {
		t.Fatalf("JSON invalide : %v", err)
	}
	return reflect.DeepEqual(va, vb)
}

// compactJSON rend un document sur une ligne pour les messages d'échec.
func compactJSON(t *testing.T, raw json.RawMessage) string {
	t.Helper()
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	b, err := json.Marshal(v)
	if err != nil {
		return string(raw)
	}
	return string(b)
}

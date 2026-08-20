package cmd

import (
	"encoding/json"
	"io"
	"strconv"

	"github.com/stephrobert/pepin/internal/model"
	"github.com/stephrobert/scankit/finding"
)

// L'ORIGINE TERRAFORM D'UN FINDING : fichier, ligne, module.
//
// Un finding issu d'un plan désigne la ressource fautive, jamais l'endroit du
// code qui l'a produite. Sur un dépôt d'infrastructure de taille réelle, avec des
// modules imbriqués, ce trajet se refait à la main pour chaque finding. Porter
// l'origine transforme le parcours en « trouver, comprendre, corriger ».
//
// # Par quel canal elle voyage
//
// Par les LABELS du finding. `finding.Finding` vient du module partagé scankit et
// n'a pas de champ de localisation ; sa carte `labels` est en revanche extensible,
// et c'est déjà par elle que voyagent le fournisseur, la catégorie et le check
// agnostique. Aucune ligne de scankit ne bouge, et la forme gelée de
// `--format json` non plus, puisque c'est le TYPE qui y est figé.
//
// Contrairement aux labels de traduction, ceux-ci ne sont PAS consommés puis
// retirés : ils ne sont pas un transport, ils sont une donnée du rapport. Un
// consommateur de `--format json` doit pouvoir les lire.
//
// # Ce qui n'est jamais fabriqué
//
// Sur une collecte live, l'origine n'existe pas : aucun label n'est posé, et rien
// n'échoue. Sur un plan dont les sources HCL ne sont pas lisibles, seul le module
// est posé. Une origine partielle est rendue partielle.

// Les trois labels par lesquels l'origine voyage. Préfixés `tf_` parce qu'ils
// n'ont de sens que pour la source Terraform : un consommateur qui les cherche
// sur un scan live doit constater leur absence, pas les voir vides.
const (
	labelTFFile   = "tf_file"
	labelTFLine   = "tf_line"
	labelTFModule = "tf_module"
)

// originIndex associe un SUJET de finding à l'origine de la ressource qui le
// porte. Les règles choisissent elles-mêmes leur sujet — l'identifiant ou le nom
// de la ressource selon ce qui parle le plus à un lecteur —, donc les deux formes
// sont indexées, exactement comme le fait la résolution des dérogations.
type originIndex map[string]model.SourceRef

// originsOf construit l'index depuis l'inventaire évalué. Il accepte les deux
// formes que le scan manipule : les ressources typées (plan Terraform) et la
// forme générique d'un export JSON relu — dont l'input.json d'un bundle scellé,
// pour qu'une re-dérivation retrouve les mêmes annotations.
func originsOf(input any) originIndex {
	idx := originIndex{}
	m, ok := input.(map[string]any)
	if !ok {
		return idx
	}
	add := func(id, name string, src *model.SourceRef) {
		if src == nil {
			return
		}
		for _, key := range []string{id, name} {
			if key == "" {
				continue
			}
			// Deux ressources différentes peuvent partager un nom : dans ce cas, la
			// première gagne, et c'est le seul point où l'index peut se tromper. Il ne
			// peut pas désigner un fichier qui n'existe pas, seulement l'un des deux
			// blocs qui portent ce nom — dégradation acceptable, et l'identifiant, lui,
			// reste exact.
			if _, seen := idx[key]; !seen {
				idx[key] = *src
			}
		}
	}
	switch rs := m["resources"].(type) {
	case []model.Resource:
		for _, r := range rs {
			add(r.ID, r.Name, r.Source)
		}
	case []any:
		for _, it := range rs {
			rm, ok := it.(map[string]any)
			if !ok {
				continue
			}
			id, _ := rm["id"].(string)
			name, _ := rm["name"].(string)
			raw, ok := rm["source"].(map[string]any)
			if !ok {
				continue
			}
			file, _ := raw["file"].(string)
			mod, _ := raw["module"].(string)
			line := 0
			if f, ok := raw["line"].(float64); ok {
				line = int(f)
			}
			add(id, name, &model.SourceRef{File: file, Line: line, Module: mod})
		}
	}
	return idx
}

// withTerraformOrigin annote chaque finding de l'origine de son sujet. Passe
// POSTÉRIEURE, qui n'écrit que des labels et ne lit aucun verdict : elle ne peut
// pas en déplacer un.
func withTerraformOrigin(findings []finding.Finding, idx originIndex) {
	if len(idx) == 0 {
		return
	}
	for i := range findings {
		src, ok := idx[findings[i].Subject]
		if !ok {
			continue
		}
		if findings[i].Labels == nil {
			findings[i].Labels = map[string]string{}
		}
		if src.File != "" {
			findings[i].Labels[labelTFFile] = src.File
		}
		if src.Line > 0 {
			findings[i].Labels[labelTFLine] = strconv.Itoa(src.Line)
		}
		if src.Module != "" {
			findings[i].Labels[labelTFModule] = src.Module
		}
	}
}

// writeSARIF rend le SARIF de scankit, puis y INJECTE la localisation de chaque
// résultat quand le finding la porte.
//
// Pourquoi une passe postérieure sur le document plutôt qu'un rendu local. Le
// rendu SARIF appartient à scankit (le moteur, le modèle et le rendu y vivent,
// pour que Pépin et pitstop soient identiques), et son modèle de résultat n'a
// aujourd'hui qu'une URI d'artefact, la même pour tous : pas de `region`, donc
// pas de ligne. Réécrire un rendu SARIF dans Pépin donnerait deux rendus à tenir,
// qui divergeraient. Compléter le document produit ajoute l'information sans
// dupliquer le rendu — et le jour où scankit sait l'exprimer, cette passe
// disparaît sans que le format vu par un consommateur ne bouge.
//
// L'annotation de code d'un GitHub ou d'un GitLab tient à cette `region` : sans
// elle, un finding s'affiche sur le fichier de plan, pas sur la ligne fautive.
func writeSARIF(w io.Writer, render func(io.Writer) error, findings []finding.Finding) error {
	var buf jsonBuffer
	if err := render(&buf); err != nil {
		return err
	}
	doc, ok := buf.decode()
	if !ok {
		// Document illisible : on rend ce que scankit a produit, tel quel. Une
		// annotation manquante vaut mieux qu'un SARIF cassé.
		_, err := w.Write(buf.Bytes())
		return err
	}
	locate(doc, findings)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(doc)
}

// locate pose la localisation de chaque résultat SARIF depuis le finding de même
// rang. Le rendu de scankit émet un résultat par finding, dans l'ordre : c'est ce
// que `locate` suppose, et le seul point où les deux sont couplés. Si le nombre
// de résultats ne correspond pas, rien n'est annoté — mieux vaut aucune ligne que
// la ligne d'un autre finding.
func locate(doc map[string]any, findings []finding.Finding) {
	runs, _ := doc["runs"].([]any)
	if len(runs) != 1 {
		return
	}
	run, _ := runs[0].(map[string]any)
	results, _ := run["results"].([]any)
	if len(results) != len(findings) {
		return
	}
	for i, r := range results {
		res, ok := r.(map[string]any)
		if !ok {
			continue
		}
		file := findings[i].Label(labelTFFile)
		if file == "" {
			continue
		}
		phys := map[string]any{"artifactLocation": map[string]any{"uri": file}}
		if n, err := strconv.Atoi(findings[i].Label(labelTFLine)); err == nil && n > 0 {
			phys["region"] = map[string]any{"startLine": n}
		}
		res["locations"] = []any{map[string]any{"physicalLocation": phys}}
	}
}

// jsonBuffer capte la sortie d'un rendu pour la relire. Un `bytes.Buffer` nommé,
// pour que la relecture reste explicite au point d'appel.
type jsonBuffer struct{ b []byte }

func (j *jsonBuffer) Write(p []byte) (int, error) { j.b = append(j.b, p...); return len(p), nil }

// Bytes rend le document capté tel quel.
func (j *jsonBuffer) Bytes() []byte { return j.b }

func (j *jsonBuffer) decode() (map[string]any, bool) {
	var doc map[string]any
	if err := json.Unmarshal(j.b, &doc); err != nil {
		return nil, false
	}
	return doc, true
}

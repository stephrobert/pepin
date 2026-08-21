// Package tenants porte les TENANTS DE RÉFÉRENCE : des configurations réelles,
// écrites par des tiers, contre lesquelles Pépin est rejoué à chaque build.
//
// # Pourquoi une fixture ne suffit pas
//
// Une fixture est écrite par l'auteur de la règle. Elle vérifie donc que la règle
// se déclenche — jamais qu'elle a RAISON sur une configuration que personne n'a
// conçue pour elle. C'est un test auto-confirmant : il mesure l'intention de son
// auteur, pas la réalité du parc.
//
// Le précédent est net. Le rejeu de stacks Terraform de tiers contre le binaire a
// trouvé, en une séance, un faux positif CRITICAL sur la configuration Scaleway la
// plus courante. Aucune fixture maison ne l'aurait révélé, parce qu'aucune fixture
// maison ne décrit une configuration CORRECTE que son auteur n'a pas dessinée.
//
// # Ce qu'un tenant de référence est, et ce qu'il n'est pas
//
// Un tenant est un plan Terraform d'une configuration tierce publiée, avec sa
// provenance (dépôt, commit, chemin, licence) et les verdicts que Pépin y rend,
// consignés. Rien n'est provisionné : un plan ne crée aucune ressource cloud, ce
// que CONTRIBUTING.md exige et que la facture confirme.
//
// Il n'est PAS un tenant réel : un plan porte l'état PLANIFIÉ, et ce qu'un
// fournisseur RÉPOND reste dû à une collecte live. La carte de qualité de
// détection le dit à sa place plutôt que de le laisser croire.
//
// # Le plan committé est RÉDUIT, et c'est une garde
//
// Pépin ne lit d'un plan que deux choses (internal/tfparse.ParsePlan) :
// `planned_values` (ou `values`) et les `source` des appels de modules sous
// `configuration.root_module`. Tout le reste — `variables`, `provider_config`,
// `prior_state`, `resource_changes` — est ignoré par le produit, et c'est
// précisément là qu'un tenant réel cacherait ses identifiants, ses UUID et ses
// adresses. Un tenant de référence ne porte donc QUE ce que Pépin lit, et toute
// valeur que Terraform lui-même marque `sensitive` y est mise à null.
//
// La réduction n'est pas une commodité de taille (elle divise pourtant le corpus
// par sept) : c'est la même discipline que celle des enregistrements de trace —
// rien n'entre au dépôt sans avoir été relu valeur par valeur, et une règle
// mécanique se relit mieux qu'une bonne intention.
package tenants

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	yaml "go.yaml.in/yaml/v3"

	"github.com/stephrobert/pepin/internal/genprovider"
	"github.com/stephrobert/pepin/internal/veracity"
)

// Dir est le dossier des tenants, relatif à la racine du dépôt.
const Dir = "references/tenants"

// Les deux postures qu'un tenant peut porter. Elles ne sont pas décrétées : elles
// sont CONSTATÉES par le scan, et une porte le vérifie — un tenant annoncé durci
// qui rendrait un écart critical/high mentirait sur ce qu'il prouve.
const (
	// PostureExposed : le tenant porte au moins un écart, quelle qu'en soit la sévérité.
	PostureExposed = "exposed"
	// PostureHardened : le tenant ne porte aucun écart critical ou high.
	PostureHardened = "hardened"
)

// Upstream est la provenance d'une configuration tierce. Sans elle, un plan
// committé est un plan que le dépôt a fini par s'écrire à lui-même.
type Upstream struct {
	Repo      string `yaml:"repo"`
	Commit    string `yaml:"commit"`
	Path      string `yaml:"path"`
	Licence   string `yaml:"licence"`
	Retrieved string `yaml:"retrieved"`
}

// Tenant est un tenant de référence : d'où vient la configuration, ce qu'elle
// met en scène, et pourquoi elle n'est pas auto-confirmante.
type Tenant struct {
	Dir      string   `yaml:"-"` // chemin du dossier, pour les messages d'échec
	Name     string   `yaml:"-"` // <fournisseur>/<nom>
	Provider string   `yaml:"provider"`
	Source   string   `yaml:"source"` // live | terraform
	Posture  string   `yaml:"posture"`
	Title    string   `yaml:"title"`
	TitleEn  string   `yaml:"title_en"`
	Upstream Upstream `yaml:"upstream"`
	Why      string   `yaml:"why"`
	WhyEn    string   `yaml:"why_en"`
}

// PlanPath rend le chemin du plan réduit.
func (t Tenant) PlanPath() string { return filepath.Join(t.Dir, "plan.json") }

// ExpectedPath rend le chemin du relevé de verdicts attendus.
func (t Tenant) ExpectedPath() string { return filepath.Join(t.Dir, "expected.txt") }

// PathOf rend le chemin de véracité qu'un contrôle emprunte sur ce tenant.
func (t Tenant) PathOf(control string) veracity.Path {
	return veracity.Path{Control: control, Provider: t.Provider, Source: t.Source}
}

// Load lit tous les tenants de référence sous <root>/references/tenants.
func Load(root string) ([]Tenant, error) {
	base := filepath.Join(root, Dir)
	providers, err := os.ReadDir(base)
	if err != nil {
		return nil, fmt.Errorf("lecture de %s : %w", base, err)
	}
	var out []Tenant
	for _, p := range providers {
		if !p.IsDir() {
			continue
		}
		names, rerr := os.ReadDir(filepath.Join(base, p.Name()))
		if rerr != nil {
			return nil, fmt.Errorf("lecture de %s : %w", filepath.Join(base, p.Name()), rerr)
		}
		for _, n := range names {
			if !n.IsDir() {
				continue
			}
			dir := filepath.Join(base, p.Name(), n.Name())
			t, lerr := loadOne(dir)
			if lerr != nil {
				return nil, lerr
			}
			t.Name = p.Name() + "/" + n.Name()
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func loadOne(dir string) (Tenant, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "tenant.yaml")) // #nosec G304 -- chemin construit depuis le dossier des tenants du dépôt.
	if err != nil {
		return Tenant{}, fmt.Errorf("lecture du manifeste de %s : %w", dir, err)
	}
	var t Tenant
	if uerr := yaml.Unmarshal(raw, &t); uerr != nil {
		return Tenant{}, fmt.Errorf("manifeste %s invalide : %w", dir, uerr)
	}
	t.Dir = dir
	return t, nil
}

// LoadExpected lit un relevé de verdicts : une ligne `<contrôle> <statut>`,
// commentaires `#` et lignes vides ignorés.
func LoadExpected(path string) (map[string]string, error) {
	raw, err := os.ReadFile(path) // #nosec G304 -- fixture du dépôt.
	if err != nil {
		return nil, fmt.Errorf("lecture du relevé %s : %w", path, err)
	}
	out := map[string]string{}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return nil, fmt.Errorf("%s : ligne « %s » : deux champs attendus (contrôle, statut)", path, line)
		}
		out[fields[0]] = fields[1]
	}
	return out, nil
}

// RenderExpected rend le contenu d'un relevé, en-tête comprise. La forme est
// pauvre à dessein : une ligne par contrôle, triée, pour qu'un diff se lise — un
// verdict qui bascule est alors une ligne, et une seule.
func RenderExpected(t Tenant, got map[string]string) string {
	codes := make([]string, 0, len(got))
	for c := range got {
		codes = append(codes, c)
	}
	sort.Strings(codes)
	var b strings.Builder
	b.WriteString("# Verdicts rendus par Pépin sur ce tenant de référence — RÉGÉNÉRÉ.\n")
	b.WriteString("#   mise run tenants-update\n#\n")
	b.WriteString("# Amont : " + t.Upstream.Repo + "\n")
	b.WriteString("#   commit " + t.Upstream.Commit + " · " + t.Upstream.Path + " · " + t.Upstream.Licence + "\n#\n")
	b.WriteString("# Une ligne qui bascule est une DÉCISION : soit le produit s'est amélioré,\n")
	b.WriteString("# soit il a régressé sur une configuration que personne n'a écrite pour lui.\n")
	b.WriteString("# Aucune régénération sans avoir dit laquelle des deux.\n\n")
	for _, c := range codes {
		b.WriteString(c + " " + got[c] + "\n")
	}
	return b.String()
}

// WriteExpected écrit le relevé sur disque.
func WriteExpected(t Tenant, got map[string]string) error {
	body := RenderExpected(t, got)
	if err := os.WriteFile(t.ExpectedPath(), []byte(body), 0o600); err != nil {
		return fmt.Errorf("écriture de %s : %w", t.ExpectedPath(), err)
	}
	return nil
}

// planKeys : les clés de premier niveau qu'un plan de tenant a le droit de porter.
// Ce sont EXACTEMENT celles que internal/tfparse.ParsePlan lit, plus les deux
// champs de version qui identifient le format.
//
// La liste est courte parce que la surface de lecture de Pépin l'est. Si un jour
// ParsePlan lit un bloc de plus, cette liste doit bouger avec lui — le commentaire
// réciproque est dans internal/tfparse/tfparse.go.
var planKeys = map[string]bool{
	"format_version":    true,
	"terraform_version": true,
	"planned_values":    true,
	"values":            true,
	"configuration":     true,
}

// CheckPlanShape refuse un plan qui porte autre chose que ce que Pépin lit.
//
// La garde est une garde de SÉCURITÉ avant d'être une garde de taille : `variables`,
// `provider_config`, `prior_state` et `resource_changes` sont les blocs où un plan
// pris sur un tenant réel porte ses identifiants et ses UUID. Committer un plan brut
// est l'erreur que l'audit de livraison a déjà vue passer une fois.
func CheckPlanShape(path string) ([]string, error) {
	raw, err := os.ReadFile(path) // #nosec G304 -- fixture du dépôt.
	if err != nil {
		return nil, fmt.Errorf("lecture de %s : %w", path, err)
	}
	var doc map[string]json.RawMessage
	if uerr := json.Unmarshal(raw, &doc); uerr != nil {
		return nil, fmt.Errorf("plan %s illisible : %w", path, uerr)
	}
	var extra []string
	for k := range doc {
		if !planKeys[k] {
			extra = append(extra, k)
		}
	}
	sort.Strings(extra)
	return extra, nil
}

// rootField rend le premier segment d'un chemin de projection : `audit.0.endpoint`
// donne `audit`, `_parent.id` donne `id`. C'est la racine qu'un plan doit porter
// pour que internal/tfmap.Apply puisse descendre dedans.
func rootField(path string) string {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "_parent.")
	if i := strings.IndexAny(path, ".["); i >= 0 {
		path = path[:i]
	}
	return path
}

// MappedFields rend, par tf_type, les champs Terraform que les descripteurs lisent
// RÉELLEMENT — c'est-à-dire exactement ce que internal/tfmap.Apply consulte : les
// racines des chemins de `map` (`||` est un repli, `_parent.` désigne la ressource
// porteuse), le champ `region`, le bloc répété de `items`, et le `name` qu'Apply lit
// sur chaque ressource pour nommer la ressource normalisée.
//
// La même liste est calculée côté écriture par scripts/tenant-plan.py. Les deux
// existent à dessein : le script RÉDUIT, ce test REFUSE. Si elles divergent, c'est
// la porte qui rougit, et c'est le bon sens de la panne — un plan trop bavard se
// voit, un plan trop pauvre ferait bouger un verdict et casserait `expected.txt`.
func MappedFields(root string) (map[string]map[string]bool, error) {
	entries, err := os.ReadDir(filepath.Join(root, "providers"))
	if err != nil {
		return nil, fmt.Errorf("lecture des descripteurs : %w", err)
	}
	out := map[string]map[string]bool{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		desc, lerr := genprovider.Load(os.DirFS(root), "providers/"+e.Name())
		if lerr != nil {
			return nil, fmt.Errorf("descripteur %s : %w", e.Name(), lerr)
		}
		for _, r := range desc.MappingTerraform.Resources {
			if r.TFType == "" {
				continue
			}
			if out[r.TFType] == nil {
				// Apply lit `name` sur chaque ressource pour nommer la ressource normalisée.
				out[r.TFType] = map[string]bool{"name": true}
			}
			keep := out[r.TFType]
			if r.Region != "" {
				keep[r.Region] = true
			}
			if r.Items != "" {
				keep[rootField(r.Items)] = true
			}
			for _, path := range r.Map {
				for _, alt := range strings.Split(path, "||") {
					if f := rootField(alt); f != "" {
						keep[f] = true
					}
				}
			}
		}
	}
	return out, nil
}

// CheckPlanAttributes refuse un plan qui porte un ATTRIBUT qu'aucun mapping ne lit.
//
// CheckPlanShape s'arrête aux sections ; cette garde-ci descend jusqu'au champ, et
// c'est là que se jouait le défaut : un `helm_release` embarquait tout son blob de
// valeurs Helm, un `kubectl_manifest` son `yaml_body`, un `kubernetes_secret` son
// contenu — aucun n'est lu par une règle commune, et tous étaient republiés depuis
// la configuration applicative d'un tiers.
//
// Rend les entrées `tf_type.attribut` en trop, triées.
func CheckPlanAttributes(root, path string) ([]string, error) {
	allow, err := MappedFields(root)
	if err != nil {
		return nil, err
	}
	if len(allow) == 0 {
		return nil, fmt.Errorf("aucun mapping_terraform lu depuis %s : la garde ne mesure rien", root)
	}
	raw, err := os.ReadFile(path) // #nosec G304 -- fixture du dépôt.
	if err != nil {
		return nil, fmt.Errorf("lecture de %s : %w", path, err)
	}
	// `planned_values` ou `values` : les deux formes qu'un plan porte selon la
	// commande qui l'a produit, exactement comme les lit internal/tfparse.
	var doc struct {
		PlannedValues *planModuleWrap `json:"planned_values"`
		Values        *planModuleWrap `json:"values"`
	}
	if uerr := json.Unmarshal(raw, &doc); uerr != nil {
		return nil, fmt.Errorf("plan %s illisible : %w", path, uerr)
	}
	wrap := doc.PlannedValues
	if wrap == nil {
		wrap = doc.Values
	}
	if wrap == nil {
		return nil, nil
	}
	seen := map[string]bool{}
	walkAttributes(wrap.RootModule, allow, seen)
	var extra []string
	for k := range seen {
		extra = append(extra, k)
	}
	sort.Strings(extra)
	return extra, nil
}

// walkAttributes collecte les `tf_type.attribut` qu'aucun mapping ne lit.
func walkAttributes(m planModule, allow map[string]map[string]bool, out map[string]bool) {
	for _, r := range m.Resources {
		keep := allow[r.Type]
		for k := range r.Values {
			if !keep[k] {
				out[r.Type+"."+k] = true
			}
		}
	}
	for _, c := range m.ChildModules {
		walkAttributes(c, allow, out)
	}
}

// PresentTypes rend les types NORMALISÉS que le plan d'un tenant produit, en
// empruntant le mapping Terraform du descripteur du fournisseur.
//
// Sert au filtre « substantiel » : un `not-evaluated` sur un contrôle dont le
// tenant ne porte AUCUNE ressource du type visé ne prouve pas la garde de
// capacité, il constate une absence.
func PresentTypes(root string, t Tenant) (map[string]bool, error) {
	desc, err := genprovider.Load(os.DirFS(root), "providers/"+t.Provider+".yaml")
	if err != nil {
		return nil, fmt.Errorf("descripteur de %s : %w", t.Provider, err)
	}
	byTF := map[string]string{}
	for _, r := range desc.MappingTerraform.Resources {
		byTF[r.TFType] = r.Type
	}
	raw, err := os.ReadFile(t.PlanPath()) // #nosec G304 -- fixture du dépôt.
	if err != nil {
		return nil, fmt.Errorf("lecture de %s : %w", t.PlanPath(), err)
	}
	var doc struct {
		PlannedValues *planModuleWrap `json:"planned_values"`
		Values        *planModuleWrap `json:"values"`
	}
	if uerr := json.Unmarshal(raw, &doc); uerr != nil {
		return nil, fmt.Errorf("plan %s illisible : %w", t.PlanPath(), uerr)
	}
	root2 := doc.PlannedValues
	if root2 == nil {
		root2 = doc.Values
	}
	out := map[string]bool{}
	if root2 != nil {
		walkModule(root2.RootModule, byTF, out)
	}
	// Les contrôles de gouvernance ne visent aucun type ; la ressource de
	// souveraineté est posée par le scan sur TOUT inventaire.
	out[""] = true
	return out, nil
}

type planModuleWrap struct {
	RootModule planModule `json:"root_module"`
}

type planModule struct {
	Resources []struct {
		Type string `json:"type"`
		// Values sert à CheckPlanAttributes : la garde descend jusqu'au champ, là où
		// walkModule ne regarde que le type. Brut, parce qu'on ne juge ici que la
		// PRÉSENCE d'un attribut, jamais sa valeur.
		Values map[string]json.RawMessage `json:"values"`
	} `json:"resources"`
	ChildModules []planModule `json:"child_modules"`
}

func walkModule(m planModule, byTF map[string]string, out map[string]bool) {
	for _, r := range m.Resources {
		if typ, ok := byTF[r.Type]; ok {
			out[typ] = true
		}
	}
	for _, c := range m.ChildModules {
		walkModule(c, byTF, out)
	}
}

// Substantive dit si un verdict observé sur un tenant PROUVE quelque chose.
//
// La règle, et l'arbitrage qu'elle porte :
//
//   - `fail`, `pass`, `not-applicable` : toujours. La chaîne a conclu sur une
//     donnée réelle, et c'est exactement ce que le contrat de véracité demande.
//   - `not-evaluated` : seulement si le tenant porte au moins une ressource du
//     type visé. Sinon le verdict dit « ce tenant n'a rien de cette sorte », ce
//     qui est vrai, utile, et n'éprouve PAS la garde de capacité.
//
// Sans ce filtre, six tenants paieraient quatre-vingt-dix-sept obligations au lieu
// de cinquante — et la moitié le seraient par des absences. Un compteur qu'on fait
// baisser avec des cases vides est pire qu'un compteur qui ne baisse pas : il
// déplace le faux vert dans le tableau de bord.
func Substantive(code, status string, present map[string]bool) bool {
	if status != string(veracity.NotEvaluated) {
		return true
	}
	return present[genprovider.ControlType(code)]
}

// Covered rend, par chemin de véracité, les verdicts que les tenants de référence
// prouvent — filtre « substantiel » appliqué.
//
// C'est ce qui relie les deux artefacts : le relevé d'un tenant est à la fois la
// non-régression de l'issue #58 et une preuve de véracité au sens de l'issue #43.
// Les compter DEUX fois, ou les recalculer autrement, ferait diverger le registre
// de la carte — et celui qui diverge est toujours celui qu'on lit.
func Covered(root string) (map[veracity.Path][]veracity.Verdict, error) {
	list, err := Load(root)
	if err != nil {
		return nil, err
	}
	out := map[veracity.Path][]veracity.Verdict{}
	for _, t := range list {
		expected, lerr := LoadExpected(t.ExpectedPath())
		if lerr != nil {
			return nil, lerr
		}
		present, perr := PresentTypes(root, t)
		if perr != nil {
			return nil, perr
		}
		for code, status := range expected {
			if !Substantive(code, status, present) {
				continue
			}
			p := t.PathOf(code)
			out[p] = append(out[p], veracity.Verdict(status))
		}
	}
	return out, nil
}

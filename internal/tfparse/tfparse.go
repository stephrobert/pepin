// Package tfparse lit la sortie `terraform show -json` (un plan Terraform) et en
// extrait les ressources sous une forme générique, indépendante du provider.
//
// On consomme le plan plutôt que le HCL brut pour la FIDÉLITÉ : `planned_values`
// porte des valeurs résolues (variables, locals, modules, fonctions comme
// jsonencode), qu'un audit du HCL brut manquerait silencieusement.
//
// Avec une RÉSERVE, mesurée et longtemps ignorée : ce que le plan ne peut pas encore
// connaître — l'identifiant d'une ressource que le même plan va créer — n'est pas
// résolu, il est simplement ABSENT. Sur un plan Outscale réaliste, cela emporte
// `vm_id`, `public_ip`, `security_group_ids` de chaque VM et `security_group_id` de
// chaque règle : tout ce sur quoi les règles de corrélation se joignent. Le verdict
// restait honnête (`not-evaluated`, jamais un `pass`), mais la corrélation ne
// fonctionnait pas — contrairement à ce que ce commentaire affirmait.
//
// Ce que le plan porte quand même, ailleurs, c'est la RELATION : `configuration`
// garde les références déclarées par l'exploitant (`vm_id` → `outscale_vm.web`).
// C'est une observation du plan, pas une estimation, et `resolveReferences` la lit.
//
// La projection des types Terraform (`scaleway_*`, `outscale_*`) vers le modèle
// normalisé agnostique de Pépin est faite par chaque provider (mapper dédié).
package tfparse

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/stephrobert/pepin/internal/i18n"
)

// Resource est une ressource Terraform extraite du plan : son type HCL (ex.
// "outscale_security_group_rule"), son nom local, son adresse complète, et ses
// valeurs résolues (`values` du plan).
type Resource struct {
	Type    string
	Name    string
	Address string
	Values  map[string]any
	// Origin situe la ressource dans le CODE qui la déclare (module toujours ;
	// fichier et ligne quand les sources HCL ont pu être lues à côté du plan).
	// Vide sur toute source qui n'est pas un plan : rien n'y est inventé.
	Origin Origin
	// References : par argument, les ADRESSES de ressources que l'exploitant y a
	// référencées (`vm_id` → ["outscale_vm.web"]). Lu dans `configuration`, qui
	// garde la relation là où `planned_values` n'a qu'un trou.
	//
	// Ne contient QUE des adresses de ressources présentes dans ce même plan : une
	// référence à une variable, un local ou une source de données ne résout rien.
	// C'est ce qui rend la résolution exacte plutôt qu'approchée — on ne projette
	// jamais une adresse qu'on n'a pas vue.
	References map[string][]string
}

// plan reflète la structure minimale de `terraform show -json`. La sortie d'un
// fichier de plan porte `planned_values` (valeurs futures, certains champs
// calculés encore inconnus) ; celle d'un état appliqué porte `values` (tout est
// résolu). On accepte les deux.
type plan struct {
	PlannedValues *valuesBlock `json:"planned_values"`
	Values        *valuesBlock `json:"values"`
	// Configuration porte les appels de modules, dont la SOURCE permet de situer
	// les sources HCL d'un module local. C'est le seul indice de localisation de
	// document que le format JSON de Terraform expose (vérifié dans
	// internal/command/jsonconfig/config.go) : il n'y a ni fichier ni ligne.
	Configuration *configBlock `json:"configuration"`
}

// configBlock est la représentation `configuration` du plan : ce qui situe un
// module local, et les références déclarées par chaque ressource.
type configBlock struct {
	RootModule configModule `json:"root_module"`
}

type configModule struct {
	ModuleCalls map[string]moduleCall `json:"module_calls"`
	Resources   []configResource      `json:"resources"`
}

// configResource est une ressource telle que la CONFIGURATION la déclare. Seules
// les références nous intéressent : les valeurs constantes sont déjà dans
// `planned_values`, mieux résolues.
type configResource struct {
	Address string `json:"address"`
	// Expressions est délibérément laissé BRUT. Terraform met sous cette clé, selon
	// l'argument, un objet (`{"references": [...]}`) ou un TABLEAU d'objets (un bloc
	// répété, comme `block_device_mappings`). Une structure stricte échoue sur le
	// second, et cet échec ne serait pas local : `json.Unmarshal` rend une erreur pour
	// tout le plan, donc un plan parfaitement valide deviendrait illisible à cause
	// d'un bloc qui ne nous intéresse pas. Mesuré sur la fixture du dépôt, qui porte
	// un `block_device_mappings`.
	Expressions map[string]json.RawMessage `json:"expressions"`
}

// expression est un argument de la configuration. Terraform y met soit une valeur
// constante, soit les traversées référencées. On ne lit que les secondes.
type expression struct {
	References []string `json:"references"`
}

// referencesOf décode un argument et rend ses traversées. Ce qui ne se décode pas en
// objet — un bloc répété, une forme qu'une version future introduirait — rend une
// liste vide plutôt qu'une erreur : c'est une entrée de tiers, et n'en pas
// comprendre une partie ne doit pas empêcher d'en lire le reste.
func referencesOf(raw json.RawMessage) []string {
	var e expression
	if err := json.Unmarshal(raw, &e); err != nil {
		return nil
	}
	return e.References
}

type moduleCall struct {
	Source string       `json:"source"`
	Module configModule `json:"module"`
	// Expressions : les ARGUMENTS passés au module, dans le scope de l'appelant.
	// C'est le seul endroit où se trouve ce qu'un `var.x` vaut à l'intérieur du
	// module — sans quoi une référence franchissant une frontière de module ne
	// résout rien, et la corrélation ne fonctionne que sur des plans à plat.
	Expressions map[string]json.RawMessage `json:"expressions"`
}

type valuesBlock struct {
	RootModule module `json:"root_module"`
}

type module struct {
	Resources    []planResource `json:"resources"`
	ChildModules []module       `json:"child_modules"`
}

type planResource struct {
	Address string         `json:"address"`
	Type    string         `json:"type"`
	Name    string         `json:"name"`
	Values  map[string]any `json:"values"`
}

// ParsePlan lit un fichier `terraform show -json` et retourne les ressources de
// tous les modules (racine + enfants), triées de façon déterministe.
func ParsePlan(path string) ([]Resource, error) {
	raw, err := os.ReadFile(path) // #nosec G304 -- chemin de plan Terraform fourni par l'utilisateur en argument CLI, lu en seule lecture.
	if err != nil {
		return nil, fmt.Errorf(i18n.T("lecture du plan %s : %w", "reading the plan %s: %w"), path, err)
	}
	var p plan
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf(i18n.T("plan terraform JSON invalide : %w", "invalid Terraform JSON plan: %w"), err)
	}
	root := p.PlannedValues
	if root == nil {
		root = p.Values
	}
	if root == nil {
		return nil, errors.New(i18n.T("plan terraform sans bloc planned_values ni values (sortie de `terraform show -json` attendue)", "Terraform plan with neither a planned_values nor a values block (`terraform show -json` output expected)"))
	}
	var out []Resource
	collect(&out, root.RootModule)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Address < out[j].Address })
	// L'origine est renseignée APRÈS le tri, sur les ressources déjà extraites : elle
	// n'entre dans aucune décision de parsing et son absence ne fait rien échouer.
	// Sur un plan dont les sources HCL ne sont pas à côté, seul le module est rendu.
	resolveOrigins(out, path, moduleDirsOf(p.Configuration))
	// Les références sont renseignées APRÈS le tri, comme l'origine, et pour la même
	// raison : elles n'entrent dans aucune décision de parsing, et leur absence ne
	// fait rien échouer. Un plan sans bloc `configuration` reste lisible.
	resolveReferences(out, p.Configuration)
	return out, nil
}

// collect parcourt récursivement un module et ses enfants (équivalent du walk
// d'osc-policy sur planned_values).
func collect(out *[]Resource, m module) {
	for _, r := range m.Resources {
		// Une ressource sans type n'est évaluable par AUCUNE règle : toutes
		// sélectionnent par `resources_of_type(...)`. La laisser entrer ajoute une
		// entrée que rien ne contrôle et pollue le relevé des types collectés, sur
		// lequel repose le verrou de capacité. Trouvé par FuzzParsePlan, sur un plan
		// où `resources: [{}]` — encoding/json apparie les clés sans tenir compte de
		// la casse, donc un plan forgé atteint ce chemin plus facilement qu'il n'y
		// paraît.
		if r.Type == "" {
			continue
		}
		*out = append(*out, Resource{Type: r.Type, Name: r.Name, Address: r.Address, Values: r.Values})
	}
	for _, child := range m.ChildModules {
		collect(out, child)
	}
}

// resolveReferences renseigne `Resource.References` depuis le bloc `configuration`.
//
// Terraform émet, pour un argument référençant une autre ressource, la traversée
// complète ET l'adresse de la ressource :
//
//	"vm_id": {"references": ["outscale_vm.web.vm_id", "outscale_vm.web"]}
//
// C'est l'ADRESSE qui nous intéresse, parce que c'est elle qui identifie la ressource
// dans l'inventaire quand le plan ne connaît pas encore son identifiant natif. On la
// reconnaît sans l'analyser : c'est celle des entrées qui correspond à une ressource
// RÉELLEMENT présente dans le plan. Une référence à `var.x`, `local.y` ou
// `data.z.w` n'en désigne aucune et ne résout donc rien — la résolution est exacte,
// pas heuristique, et c'est ce qui la sépare d'une supposition.
func resolveReferences(out []Resource, cfg *configBlock) {
	if cfg == nil {
		return
	}
	// Une adresse de CONFIGURATION peut correspondre à plusieurs INSTANCES (`count`,
	// `for_each`). On garde la liste, parce que c'est le nombre qui décide : une
	// instance unique se désigne sans ambiguïté, plusieurs ne se départagent pas.
	instances := map[string][]string{}
	for _, r := range out {
		c := configAddress(r.Address)
		instances[c] = append(instances[c], r.Address)
	}
	refs := map[string]map[string][]string{}
	args := map[string]map[string][]string{}
	collectConfig(refs, args, cfg.RootModule, "")
	r := resolveur{instances: instances, args: args}
	for i := range out {
		adresse := configAddress(out[i].Address)
		champs := refs[adresse]
		if champs == nil {
			continue
		}
		scope := modulePrefix(adresse)
		resolu := map[string][]string{}
		for champ, cibles := range champs {
			if adresses := r.resoudre(cibles, scope, 0); len(adresses) > 0 {
				resolu[champ] = adresses
			}
		}
		if len(resolu) > 0 {
			out[i].References = resolu
		}
	}
}

// resolveur porte ce qu'il faut pour transformer une liste de traversées en
// adresses de ressources réellement présentes dans le plan.
type resolveur struct {
	instances map[string][]string
	args      map[string]map[string][]string
}

// profondeurMax borne la traversée des frontières de module. Une configuration
// Terraform ne peut pas être cyclique, mais ce parseur lit une entrée de TIERS :
// une borne coûte une ligne et retire une classe entière de panne.
const profondeurMax = 8

// resoudre rend les adresses de ressources désignées par des traversées, dans le
// scope donné.
//
// Deux cas, et un seul refus :
//
//   - la traversée désigne une ressource du plan → son adresse d'INSTANCE ;
//   - la traversée désigne un `var.x` d'un module → l'argument que l'appelant lui a
//     passé, résolu à son tour dans le scope de l'appelant ;
//   - tout le reste (`local.`, `data.`, une variable de racine, une ressource
//     absente) → rien. On ne projette jamais une adresse qu'on n'a pas vue.
func (r resolveur) resoudre(cibles []string, scope string, profondeur int) []string {
	if profondeur > profondeurMax {
		return nil
	}
	var adresses []string
	vues := map[string]bool{}
	ajoute := func(a string) {
		if !vues[a] {
			vues[a] = true
			adresses = append(adresses, a)
		}
	}
	for _, c := range cibles {
		// AMBIGUÏTÉ = SILENCE. Une déclaration démultipliée par `count` produit N
		// instances que la configuration ne distingue pas : joindre la VM n° 0 au
		// groupe n° 1 fabriquerait un écart sur une ressource qui ne le porte pas.
		// C'est la classe de faux positif la plus coûteuse du dépôt, et elle vaut
		// mieux d'être muette.
		if inst := r.instances[scope+c]; len(inst) == 1 {
			ajoute(inst[0])
			continue
		}
		nom, ok := strings.CutPrefix(c, "var.")
		if !ok || scope == "" {
			continue
		}
		passe := r.args[scope][nom]
		if len(passe) == 0 {
			continue
		}
		for _, a := range r.resoudre(passe, parentPrefix(scope), profondeur+1) {
			ajoute(a)
		}
	}
	sort.Strings(adresses)
	return adresses
}

// collectConfig parcourt la configuration comme `collect` parcourt les valeurs.
// Le préfixe reconstruit l'adresse d'un module enfant (`module.reseau.` +
// `outscale_net.a`), sans quoi une ressource de module ne se rattacherait jamais à
// sa contrepartie de `planned_values`.
//
// `args` recueille au passage les ARGUMENTS de chaque appel de module, indexés par
// le préfixe du module appelé : c'est ce qui permet de traverser `var.x`.
func collectConfig(out map[string]map[string][]string, args map[string]map[string][]string, m configModule, prefixe string) {
	for _, r := range m.Resources {
		if r.Address == "" || len(r.Expressions) == 0 {
			continue
		}
		champs := map[string][]string{}
		for nom, brut := range r.Expressions {
			if refs := referencesOf(brut); len(refs) > 0 {
				champs[nom] = refs
			}
		}
		if len(champs) > 0 {
			out[prefixe+r.Address] = champs
		}
	}
	for nom, appel := range m.ModuleCalls {
		sous := prefixe + "module." + nom + "."
		if len(appel.Expressions) > 0 {
			passes := map[string][]string{}
			for arg, brut := range appel.Expressions {
				if refs := referencesOf(brut); len(refs) > 0 {
					passes[arg] = refs
				}
			}
			if len(passes) > 0 {
				args[sous] = passes
			}
		}
		collectConfig(out, args, appel.Module, sous)
	}
}

// modulePrefix rend le préfixe de module d'une adresse de configuration :
// `module.a.module.b.outscale_vm.x` donne `module.a.module.b.`, une adresse de
// racine donne la chaîne vide. C'est le scope dans lequel un `var.` se résout.
func modulePrefix(addr string) string {
	fin := 0
	reste := addr
	for strings.HasPrefix(reste, "module.") {
		suite := reste[len("module."):]
		i := strings.Index(suite, ".")
		if i < 0 {
			break
		}
		avance := len("module.") + i + 1
		fin += avance
		reste = reste[avance:]
	}
	return addr[:fin]
}

// parentPrefix rend le scope APPELANT d'un module : `module.a.module.b.` donne
// `module.a.`. La chaîne vide n'a pas de parent.
func parentPrefix(prefixe string) string {
	if prefixe == "" {
		return ""
	}
	coupe := strings.TrimSuffix(prefixe, ".")
	if i := strings.LastIndex(coupe, "module."); i > 0 {
		return coupe[:i]
	}
	return ""
}

// configAddress rend l'adresse de CONFIGURATION correspondant à une adresse de
// `planned_values`.
//
// Les deux ne coïncident pas dès qu'il y a un `count` ou un `for_each` : la
// configuration décrit la DÉCLARATION (`module.vm_tier.outscale_vm.this`), les
// valeurs décrivent chaque INSTANCE (`module.vm_tier[0].outscale_vm.this["a"]`).
// Retirer les index rattache l'instance à sa déclaration — et c'est fidèle, puisque
// toutes les instances d'une même déclaration référencent bien les mêmes choses.
//
// Sans cela, un plan modulaire — c'est-à-dire tout plan sérieux — ne résoudrait
// aucune référence, et la fonctionnalité n'existerait que sur les plans à plat.
func configAddress(addr string) string {
	var b strings.Builder
	profondeur := 0
	for _, r := range addr {
		switch r {
		case '[':
			profondeur++
		case ']':
			if profondeur > 0 {
				profondeur--
			}
		default:
			if profondeur == 0 {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

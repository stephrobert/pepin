package tfparse

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// L'ORIGINE d'une ressource d'un plan : module, fichier, ligne.
//
// # Ce que le plan porte, et ce qu'il ne porte pas
//
// Un finding issu d'un plan Terraform désigne la ressource, jamais l'endroit du
// code qui l'a produite. Sur un dépôt d'infrastructure réel, avec des modules
// imbriqués, retrouver ce point à la main coûte cher et se refait à chaque
// finding.
//
// La sortie de `terraform show -json` ne suffit PAS à le donner. Vérifié dans la
// source de Terraform (`internal/command/jsonconfig/config.go`) : la
// représentation `configuration` d'une ressource porte `address`, `mode`, `type`,
// `name`, `provider_config_key`, `expressions`, `schema_version`,
// `count_expression`, `for_each_expression` et `depends_on` — et RIEN d'autre.
// Aucun nom de fichier, aucune ligne. Le seul indice de localisation du document
// est `module_calls[<nom>].source`, l'adresse SOURCE d'un appel de module, qui
// désigne un répertoire (ou une adresse de registre), pas un fichier.
//
// Donc :
//
//   - Le MODULE se déduit de l'adresse de la ressource. C'est une lecture, pas une
//     supposition, et elle est toujours possible.
//   - Le FICHIER et la LIGNE se MESURENT dans les sources HCL, quand elles sont là.
//     On y cherche l'en-tête de bloc `resource "<type>" "<nom>"`, ce qui est une
//     observation d'un fichier réel — jamais une reconstruction.
//   - Quand les sources ne sont pas là, ou quand l'en-tête se trouve zéro ou
//     plusieurs fois, l'origine est ABSENTE. Une ligne fausse envoie corriger le
//     mauvais endroit : elle est pire qu'une ligne manquante, parce qu'on la croit.
//
// # Où les sources sont cherchées
//
// Dans le répertoire du PLAN, et dans les répertoires de modules locaux que la
// configuration du plan déclare (`source: ./modules/x`). C'est le répertoire d'où
// `terraform show -json` est lancé dans tous les enchaînements documentés. Aucun
// drapeau ne permet d'en désigner un autre, et c'est délibéré : résoudre
// `resource "x" "y"` contre un arbre qui n'est pas celui d'où vient le plan
// produirait une ligne PLAUSIBLE ET FAUSSE, exactement ce qu'on cherche à éviter.
//
// # Sur une collecte live
//
// L'information n'existe pas, par construction. Son absence est propre : rien
// n'échoue, rien n'est inventé, et les formats analysables ne portent simplement
// pas les champs.

// Origin situe une ressource dans le code qui l'a déclarée. Les trois champs sont
// indépendants : le module est presque toujours connu, le fichier et la ligne ne
// le sont que si les sources ont pu être lues.
type Origin struct {
	// File : chemin du fichier `.tf`, relatif au répertoire du plan.
	File string `json:"file,omitempty"`
	// Line : ligne de l'en-tête de bloc `resource "<type>" "<nom>"` (1-basée).
	Line int `json:"line,omitempty"`
	// Module : adresse du module qui déclare la ressource ("" = module racine).
	Module string `json:"module,omitempty"`
}

// Empty indique qu'aucune origine n'a pu être établie.
func (o Origin) Empty() bool { return o.File == "" && o.Line == 0 && o.Module == "" }

// moduleOf extrait l'adresse du module d'une adresse de ressource. Une adresse
// est `module.a.module.b.type.nom[clef]` ; le module en est le préfixe fait des
// couples `module.<nom>`. Rendu "" pour le module racine.
func moduleOf(address string) string {
	parts := strings.Split(address, ".")
	var mod []string
	for i := 0; i+1 < len(parts); i += 2 {
		if parts[i] != "module" {
			break
		}
		mod = append(mod, parts[i], parts[i+1])
	}
	return strings.Join(mod, ".")
}

// blockHeader capture l'en-tête d'un bloc `resource "<type>" "<nom>"`. Un
// balayage de lignes plutôt qu'un analyseur HCL complet : on ne cherche qu'une
// position, la grammaire d'un en-tête de bloc est stable depuis HCL2, et
// introduire un analyseur entier pour cela ferait entrer une dépendance majeure
// dans un outil dont la chaîne d'approvisionnement est elle-même un argument.
//
// Ce que ce choix coûte est explicite : un en-tête écrit sur plusieurs lignes ou
// une ressource déclarée en `.tf.json` ne sera pas trouvée. L'origine est alors
// absente, ce qui est le mode de défaillance voulu.
var blockHeader = regexp.MustCompile(`^\s*resource\s+"([A-Za-z0-9_-]+)"\s+"([A-Za-z0-9_-]+)"`)

// index associe (type, nom) à sa position, pour un répertoire de sources.
type index map[string]Origin

// ambiguous marque une clé vue plusieurs fois. Deux déclarations du même couple
// dans le périmètre cherché ne se départagent pas sans analyser la configuration
// entière : on rend alors AUCUNE origine plutôt qu'une des deux au hasard.
var ambiguous = Origin{Line: -1}

// indexDir balaie les fichiers `.tf` d'un répertoire (sans récursion : un module
// Terraform est un répertoire, ses sous-répertoires sont d'autres modules) et
// indexe les en-têtes de blocs `resource`.
func indexDir(dir, rel string) index {
	idx := index{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return idx
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".tf") {
			continue
		}
		// #nosec G304 -- fichier `.tf` du répertoire du plan que l'utilisateur a désigné
		// en argument, lu en seule lecture pour y chercher une position.
		raw, rerr := os.ReadFile(filepath.Join(dir, e.Name()))
		if rerr != nil {
			continue
		}
		for i, line := range strings.Split(string(raw), "\n") {
			m := blockHeader.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			key := m[1] + "." + m[2]
			if _, seen := idx[key]; seen {
				idx[key] = ambiguous
				continue
			}
			idx[key] = Origin{File: filepath.Join(rel, e.Name()), Line: i + 1}
		}
	}
	return idx
}

// resolveOrigins renseigne l'origine de chaque ressource du plan. `planPath` est
// le fichier de plan : son répertoire est celui du module racine.
//
// `moduleDirs` associe une adresse de module à son répertoire local, dérivée de
// `configuration.root_module.module_calls`. Un module dont la source n'est pas un
// chemin local (registre, dépôt git) n'y figure pas : ses ressources gardent leur
// module et n'ont ni fichier ni ligne.
func resolveOrigins(resources []Resource, planPath string, moduleDirs map[string]string) {
	root := filepath.Dir(planPath)
	indexes := map[string]index{"": indexDir(root, "")}
	for addr, rel := range moduleDirs {
		indexes[addr] = indexDir(filepath.Join(root, rel), rel)
	}
	for i := range resources {
		mod := moduleOf(resources[i].Address)
		resources[i].Origin.Module = mod
		idx, ok := indexes[mod]
		if !ok {
			continue
		}
		o, found := idx[resources[i].Type+"."+resources[i].Name]
		if !found || o == ambiguous {
			continue
		}
		resources[i].Origin.File = o.File
		resources[i].Origin.Line = o.Line
	}
}

// moduleDirsOf lit, dans la configuration du plan, les répertoires LOCAUX des
// modules appelés, indexés par adresse de module (`module.a`, `module.a.module.b`).
//
// Seules les sources locales (`./`, `../`) sont retenues : une source de registre
// ou de dépôt git ne désigne aucun fichier de l'arbre de travail, et prétendre le
// contraire ferait pointer une origine vers un chemin qui n'existe pas.
func moduleDirsOf(cfg *configBlock) map[string]string {
	out := map[string]string{}
	if cfg == nil {
		return out
	}
	var walk func(prefix, base string, m map[string]moduleCall)
	walk = func(prefix, base string, calls map[string]moduleCall) {
		for name, mc := range calls {
			addr := "module." + name
			if prefix != "" {
				addr = prefix + ".module." + name
			}
			if !isLocalSource(mc.Source) {
				continue
			}
			dir := filepath.Join(base, filepath.Clean(mc.Source))
			out[addr] = dir
			walk(addr, dir, mc.Module.ModuleCalls)
		}
	}
	walk("", ".", cfg.RootModule.ModuleCalls)
	return out
}

// isLocalSource indique qu'une adresse de source de module désigne un répertoire
// de l'arbre de travail.
func isLocalSource(src string) bool {
	return strings.HasPrefix(src, "./") || strings.HasPrefix(src, "../")
}

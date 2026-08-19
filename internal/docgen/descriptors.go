package docgen

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/stephrobert/pepin/internal/genprovider"
)

// providersDir est le dossier des descripteurs, relatif à la racine du dépôt.
const providersDir = "providers"

// registered mémorise les racines déjà enregistrées : provider.Register PANIQUE sur un
// doublon, et un même processus (le générateur, ou plusieurs tests) construit la matrice
// plusieurs fois.
var (
	registerMu sync.Mutex
	registered = map[string]error{}
)

// registerOnce enregistre les descripteurs d'une racine, une seule fois par processus.
func registerOnce(root string, fsys fs.FS) error {
	registerMu.Lock()
	defer registerMu.Unlock()
	if err, done := registered[root]; done {
		return err
	}
	err := genprovider.RegisterAll(fsys, providersDir)
	registered[root] = err
	return err
}

// loadDescriptors charge tous les providers/*.yaml ET les enregistre dans genprovider, pour
// que le calcul de couverture emprunte les MÊMES fonctions que le scan (TypeEtat,
// NonApplicableReason) au lieu de réinterpréter le YAML pour son compte.
func loadDescriptors(root string) (map[string]genprovider.Descriptor, error) {
	fsys := os.DirFS(root)
	if err := registerOnce(root, fsys); err != nil {
		return nil, fmt.Errorf("enregistrement des descripteurs de %s : %w", root, err)
	}
	entries, err := os.ReadDir(filepath.Join(root, providersDir))
	if err != nil {
		return nil, fmt.Errorf("lecture de %s : %w", providersDir, err)
	}
	out := map[string]genprovider.Descriptor{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		desc, lerr := genprovider.Load(fsys, providersDir+"/"+e.Name())
		if lerr != nil {
			return nil, lerr
		}
		out[desc.Name] = desc
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("aucun descripteur trouvé dans %s", filepath.Join(root, providersDir))
	}
	return out, nil
}

// goCollectorSources : les deux collecteurs écrits en Go (hors spec YAML `collecte`) et le
// type normalisé qu'ils alimentent. Ils échappent au descripteur : leur projection ne peut
// donc pas se lire dans providers/*.yaml.
var goCollectorSources = map[string]string{
	"object_storage_bucket": "internal/objectstorage/s3.go",
	"kubernetes_cluster":    "internal/oks/oks.go",
}

// attrAssign capture les attributs qu'un collecteur Go pose sur la ressource normalisée
// (`attrs["nom"] = …`).
var attrAssign = regexp.MustCompile(`attrs\["([a-z0-9_]+)"\]`)

// goCollectorAttributes DÉRIVE, depuis le source des collecteurs Go, l'ensemble des attributs
// qu'ils savent poser. Rien n'est recopié : ajouter un attribut au collecteur change la page
// de couverture, et le test de régénération le signale.
//
// La lecture échoue si un fichier ne rend AUCUN attribut : un motif devenu muet produirait
// une couverture faussement vide sans rien casser, c'est-à-dire la panne qui se mesure
// elle-même plutôt que son sujet.
func goCollectorAttributes(root string) (map[string]map[string]bool, error) {
	out := map[string]map[string]bool{}
	for typ, rel := range goCollectorSources {
		path := filepath.Join(root, rel)
		raw, err := os.ReadFile(path) // #nosec G304 -- chemin construit depuis une table constante du paquet.
		if err != nil {
			return nil, fmt.Errorf("lecture du collecteur %s : %w", rel, err)
		}
		attrs := map[string]bool{}
		for _, m := range attrAssign.FindAllStringSubmatch(string(raw), -1) {
			attrs[m[1]] = true
		}
		if len(attrs) == 0 {
			return nil, fmt.Errorf("aucun attribut extrait de %s : le motif de lecture ne mesure plus rien", rel)
		}
		out[typ] = attrs
	}
	return out, nil
}

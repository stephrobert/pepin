// Package provider définit le contrat d'un plugin cloud et le registre où
// chaque provider s'enregistre. Ajouter un cloud = créer un package sous
// providers/<nom>/ qui implémente Provider et appelle Register depuis son
// init() ; aucune autre modification du cœur n'est nécessaire.
package provider

import (
	"context"
	"fmt"
	"sort"

	"github.com/stephrobert/pepin/internal/model"
	"github.com/stephrobert/pepin/internal/tfparse"
)

// Config porte les paramètres d'une collecte live (authentification, région).
type Config struct {
	Profile string
	Region  string
	Options map[string]string
}

// Provider est un plugin cloud : un COLLECTEUR qui projette l'inventaire d'un
// compte (API live ou plan Terraform) dans le modèle normalisé commun. Il ne
// porte AUCUNE règle : toutes les règles de posture sont communes à tous les
// providers (internal/commonrules) et s'appliquent au modèle normalisé — seule
// la source (le collecteur) change d'un provider à l'autre.
type Provider interface {
	// Name est l'identifiant court et stable du provider (ex. "scaleway").
	Name() string
	// Description est un libellé court pour l'aide en ligne.
	Description() string
	// Collect interroge l'API du fournisseur et projette ses ressources dans le
	// modèle commun. Non appelé en mode hors-ligne (scan d'un export).
	//
	// Il rend un model.Inventory, donc les ressources ET l'état de collecte : ce
	// qui a été lu et ce qui ne l'a pas été. Rendre la seule liste de ressources
	// rendait indiscernables « ce service n'a aucune ressource » et « ce service
	// n'a pas répondu » — deux situations dont l'une autorise un « conforme » et
	// l'autre l'interdit. L'erreur, elle, reste réservée à ce qui empêche le scan
	// de commencer (identifiants inutilisables) ; un endpoint refusé n'est pas une
	// erreur de scan, c'est un périmètre non lu, et il s'enregistre.
	Collect(ctx context.Context, cfg Config) (model.Inventory, error)
}

// TerraformMapper projette des ressources d'un plan Terraform (types `<provider>_*`)
// vers le modèle normalisé agnostique de Pépin. Optionnel : un provider qui ne
// supporte pas l'audit Terraform ne l'implémente pas. Les attributs produits
// restent ancrés sur le contrat natif du SDK (cf. règles correspondantes).
type TerraformMapper interface {
	// MapTerraform rend, comme Collect, l'inventaire ET son état : un plan porte
	// des types de ressources qu'aucune spec ne projette, et les taire laisserait
	// croire que le plan a été audité en entier.
	MapTerraform(resources []tfparse.Resource) (model.Inventory, error)
}

// GovernanceProvider expose une ressource synthétique `governance_provider`
// portant les métadonnées de souveraineté du fournisseur (siège, contrôle
// capitalistique, qualification…), indépendantes de la collecte et évaluées par
// CLD-GVN-4. Optionnel : ok=false si la souveraineté n'est pas renseignée.
type GovernanceProvider interface {
	GovernanceResource() (model.Resource, bool)
}

var registry = map[string]Provider{}

// Register enregistre un provider. Appelé depuis l'init() de chaque plugin ;
// panique en cas de doublon (erreur de programmation détectée au démarrage).
func Register(p Provider) {
	name := p.Name()
	if name == "" {
		panic("provider sans nom")
	}
	if _, dup := registry[name]; dup {
		panic(fmt.Sprintf("provider déjà enregistré : %q", name))
	}
	registry[name] = p
}

// Get retourne le provider enregistré sous ce nom.
func Get(name string) (Provider, bool) {
	p, ok := registry[name]
	return p, ok
}

// All retourne tous les providers enregistrés, triés par nom.
func All() []Provider {
	out := make([]Provider, 0, len(registry))
	for _, p := range registry {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out
}

// Package tfmap transpose les ressources d'un plan Terraform vers le modèle
// normalisé commun de Pépin, PILOTÉ PAR UNE SPEC YAML (déclarative) — la même
// grammaire de projection (map + transforms) que la collecte live
// (internal/collect). Objectif : aucun code de mapping Go par provider ; chaque
// provider fournit un mapping-terraform.yaml. Anti-invention (§2) : une garde
// valide les attributs référencés contre le schéma réel du provider Terraform.
package tfmap

import (
	"fmt"

	yaml "go.yaml.in/yaml/v3"

	"github.com/stephrobert/pepin/internal/collect"
	"github.com/stephrobert/pepin/internal/i18n"
	"github.com/stephrobert/pepin/internal/model"
	"github.com/stephrobert/pepin/internal/tfparse"
)

// ResourceSpec : transposition d'un type de ressource Terraform (tf_type) vers un
// type normalisé commun. `map`/`transforms` suivent la grammaire de collect.Project ;
// `const` pose des attributs littéraux ; `region` nomme le champ de localisation.
type ResourceSpec struct {
	TFType     string            `yaml:"tf_type"`
	Type       string            `yaml:"type"`
	ID         string            `yaml:"id"`
	Region     string            `yaml:"region"`
	Items      string            `yaml:"items"` // bloc répété à éclater (ex. "inbound_rule[*]") : 1 ressource TF -> N
	Map        map[string]string `yaml:"map"`
	Transforms map[string]any    `yaml:"transforms"`
	Const      map[string]any    `yaml:"const"`
}

// Spec est la configuration de mapping Terraform d'un provider.
type Spec struct {
	Provider  string         `yaml:"provider"`
	Resources []ResourceSpec `yaml:"resources"`
}

// Parse lit une spec de mapping Terraform (YAML).
func Parse(raw []byte) (Spec, error) {
	var s Spec
	if err := yaml.Unmarshal(raw, &s); err != nil {
		return Spec{}, fmt.Errorf(i18n.T("spec de mapping Terraform invalide : %w", "invalid Terraform mapping spec: %w"), err)
	}
	return s, nil
}

// Apply transpose les ressources du plan vers le modèle commun selon la spec.
// Plusieurs ResourceSpec peuvent viser le même tf_type (ex. SG : inbound_rule +
// outbound_rule). Une spec avec `items` éclate un bloc répété en N ressources.
// Les types non déclarés sont ignorés.
func Apply(spec Spec, resources []tfparse.Resource) []model.Resource {
	var out []model.Resource
	for _, res := range resources {
		for _, rs := range spec.Resources {
			if rs.TFType != res.Type {
				continue
			}
			items := []any{any(res.Values)}
			if rs.Items != "" {
				items = collect.ExtractItems(res.Values, rs.Items)
			}
			region := ""
			if rs.Region != "" {
				region, _ = res.Values[rs.Region].(string)
			}
			// La source atteste le TYPE de ressource du plan, pas l'adresse : l'adresse
			// est déjà l'identifiant de la ressource, le type est ce qui dit d'où la
			// valeur a été lue. Une valeur issue d'un plan n'est PAS une observation de
			// la configuration effective, et l'origine `terraform-plan` le porte.
			src := collect.Source{Origin: model.OriginTerraform, Ref: res.Type}
			for _, it := range items {
				attrs, prov := collect.ProjectAttested(it, rs.Map, rs.Transforms, src)
				for k, v := range rs.Const {
					attrs[k] = v
				}
				collect.AttestConst(&prov, rs.Const, "descriptor:const")
				id, _ := attrs[rs.ID].(string)
				if id == "" {
					id = res.Address
				}
				name := id
				if n, ok := res.Values["name"].(string); ok && n != "" {
					name = n
				}
				out = append(out, model.Resource{Provider: spec.Provider, Type: rs.Type, ID: id, Name: name, Region: region, Attributes: attrs, Provenance: prov})
			}
		}
	}
	return out
}

package genprovider

import (
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/stephrobert/pepin/internal/collect"
)

// authTypes et credFormats définissent le CONTRAT d'un provider : seuls ces
// schémas d'authentification et formats de fichier d'identifiants sont supportés.
var (
	authTypes   = map[string]bool{"header": true, "sigv4": true, "exoscale-hmac": true}
	credFormats = map[string]bool{"": true, "scw": true, "osc": true, "exoscale": true}
)

// Validate vérifie qu'un descripteur respecte le contrat : identité complète,
// authentification et identifiants cohérents, sources collecte/mapping non vides.
// Retourne la liste des écarts (vide = conforme). Aucun appel réseau.
func Validate(name string, desc Descriptor) []string {
	var errs []string
	add := func(f string, a ...any) { errs = append(errs, fmt.Sprintf(f, a...)) }

	if desc.Name == "" {
		add("champ `name` manquant")
	} else if name != "" && desc.Name != name {
		add("name %q ≠ nom du fichier %q", desc.Name, name)
	}
	if desc.Description == "" {
		add("champ `description` manquant")
	}
	if !authTypes[desc.Auth.Type] {
		add("auth.type %q inconnu (header|sigv4|exoscale-hmac)", desc.Auth.Type)
	}
	switch desc.Auth.Type {
	case "header":
		if desc.Auth.Header == "" || desc.Auth.Value == "" {
			add("auth header requiert `header` et `value`")
		}
	case "sigv4":
		if desc.Auth.Service == "" || desc.Auth.Region == "" {
			add("auth sigv4 requiert `service` et `region`")
		}
	}
	if !credFormats[desc.Credentials.File.Format] {
		add("credentials.file.format %q inconnu (scw|osc|exoscale)", desc.Credentials.File.Format)
	}
	if desc.Credentials.Env["access_key"] == "" && desc.Credentials.File.Format == "" {
		add("aucune source d'identifiants (env.access_key ou file)")
	}

	// Source : collecte live.
	if desc.Collecte.BaseURL == "" {
		add("collecte.base_url manquant")
	}
	if len(desc.Collecte.Resources) == 0 {
		add("collecte : aucune ressource déclarée")
	}
	for i, r := range desc.Collecte.Resources {
		if r.Type == "" {
			add("collecte : ressource #%d sans `type`", i)
		}
		for attr, tr := range r.Transforms {
			for _, bad := range collect.ValidateTransform(tr) {
				add("collecte : ressource #%d (%s) transform inconnu %q sur `%s`", i, r.Type, bad, attr)
			}
		}
	}

	// Source : mapping Terraform (optionnel, mais cohérent si présent).
	for i, r := range desc.MappingTerraform.Resources {
		if r.TFType == "" || r.Type == "" {
			add("mapping_terraform : ressource #%d : `tf_type` et `type` requis", i)
		}
		for attr, tr := range r.Transforms {
			for _, bad := range collect.ValidateTransform(tr) {
				add("mapping_terraform : ressource #%d (%s) transform inconnu %q sur `%s`", i, r.TFType, bad, attr)
			}
		}
	}
	return errs
}

// ValidateAll valide tous les providers <nom>.yaml d'un dossier. Retourne, par
// provider, la liste des écarts.
func ValidateAll(fsys fs.FS, root string) (map[string][]string, error) {
	entries, err := fs.ReadDir(fsys, root)
	if err != nil {
		return nil, fmt.Errorf("lecture de %s : %w", root, err)
	}
	out := map[string][]string{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".yaml")
		desc, err := Load(fsys, path.Join(root, e.Name()))
		if err != nil {
			out[name] = []string{err.Error()}
			continue
		}
		out[name] = Validate(name, desc)
	}
	return out, nil
}

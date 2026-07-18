package genprovider

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
	yaml "go.yaml.in/yaml/v3"
)

// pathEnv : variable d'environnement surchargeant le chemin du fichier de config,
// par format (comme les CLI natives).
var pathEnv = map[string]string{
	"scw":      "SCW_CONFIG_PATH",
	"osc":      "OSC_CONFIG_FILE",
	"exoscale": "EXOSCALE_CONFIG",
}

// loadCredFile lit le fichier d'identifiants natif d'un provider et en extrait
// les clés logiques communes (access_key, secret_key, region, zone, org, host).
// Fichier absent ⇒ (nil, nil) : on retombe sur l'environnement et les défauts.
func loadCredFile(f credFile, profile string) (map[string]string, error) {
	if f.Format == "" {
		return nil, nil
	}
	path := os.Getenv(pathEnv[f.Format])
	if path == "" {
		path = expandHome(f.Path)
	}
	if path == "" {
		return nil, nil
	}
	raw, err := os.ReadFile(path) // #nosec G304 G703 -- chemin de config standard de la CLI du provider (descripteur ou variable d'env), lecture seule.
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("lecture de %s : %w", path, err)
	}
	switch f.Format {
	case "scw":
		return parseSCW(raw, profile, path)
	case "osc":
		return parseOSC(raw, profile, path)
	case "exoscale":
		return parseExoscale(raw, profile)
	}
	return nil, fmt.Errorf("format de fichier d'identifiants inconnu : %q", f.Format)
}

// parseSCW : ~/.config/scw/config.yaml (profil par défaut à la racine, profils
// nommés sous `profiles:`).
func parseSCW(raw []byte, profile, path string) (map[string]string, error) {
	type scw struct {
		AccessKey string         `yaml:"access_key"`
		SecretKey string         `yaml:"secret_key"`
		Region    string         `yaml:"default_region"`
		Zone      string         `yaml:"default_zone"`
		Org       string         `yaml:"default_organization_id"`
		Profiles  map[string]scw `yaml:"profiles"`
	}
	var c scw
	if err := yaml.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("config Scaleway %s invalide : %w", path, err)
	}
	if profile != "" {
		p, ok := c.Profiles[profile]
		if !ok {
			return nil, fmt.Errorf("profil Scaleway %q introuvable dans %s", profile, path)
		}
		c = p
	}
	return map[string]string{
		"access_key": c.AccessKey, "secret_key": c.SecretKey,
		"region": c.Region, "zone": c.Zone, "org": c.Org,
	}, nil
}

// parseOSC : ~/.osc/config.json (map profil -> identifiants, format osc-cli).
func parseOSC(raw []byte, profile, path string) (map[string]string, error) {
	type osc struct {
		AccessKey  string `json:"access_key"`
		SecretKey  string `json:"secret_key"`
		Region     string `json:"region"`
		RegionName string `json:"region_name"`
		Host       string `json:"host"`
	}
	var all map[string]osc
	if err := json.Unmarshal(raw, &all); err != nil {
		return nil, fmt.Errorf("config Outscale %s invalide : %w", path, err)
	}
	if profile == "" {
		profile = "default"
	}
	p, ok := all[profile]
	if !ok {
		return nil, fmt.Errorf("profil Outscale %q introuvable dans %s", profile, path)
	}
	region := p.Region
	if region == "" {
		region = p.RegionName
	}
	return map[string]string{
		"access_key": p.AccessKey, "secret_key": p.SecretKey,
		"region": region, "host": p.Host,
	}, nil
}

// parseExoscale : ~/.config/exoscale/exoscale.toml (compte par défaut + liste).
func parseExoscale(raw []byte, profile string) (map[string]string, error) {
	var conf struct {
		DefaultAccount string `toml:"defaultaccount"`
		Accounts       []struct {
			Name        string `toml:"name"`
			Key         string `toml:"key"`
			Secret      string `toml:"secret"`
			DefaultZone string `toml:"defaultZone"`
		} `toml:"accounts"`
	}
	if err := toml.Unmarshal(raw, &conf); err != nil {
		return nil, fmt.Errorf("config Exoscale invalide : %w", err)
	}
	want := profile
	if want == "" {
		want = conf.DefaultAccount
	}
	for _, a := range conf.Accounts {
		if want == "" || a.Name == want {
			return map[string]string{
				"access_key": a.Key, "secret_key": a.Secret, "zone": a.DefaultZone,
			}, nil
		}
	}
	if profile != "" {
		return nil, fmt.Errorf("profil Exoscale %q introuvable", profile)
	}
	return nil, nil
}

// expandHome remplace un préfixe ~/ par le répertoire personnel.
func expandHome(p string) string {
	if !strings.HasPrefix(p, "~/") {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, p[2:])
}

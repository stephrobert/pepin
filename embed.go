package main

import (
	"embed"
	"log"

	"github.com/stephrobert/pepin/internal/genprovider"
)

// providersFS embarque les descripteurs déclaratifs des providers (provider.yaml
// + collecte.yaml + mapping-terraform.yaml). Le dossier providers/ ne contient
// AUCUN code Go : un provider = trois fichiers YAML, chargés par genprovider.
//
//go:embed all:providers
var providersFS embed.FS

func init() {
	if err := genprovider.RegisterAll(providersFS, "providers"); err != nil {
		log.Fatalf("chargement des providers : %v", err)
	}
}

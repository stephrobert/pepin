package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/stephrobert/pepin/internal/genprovider"
	"github.com/stephrobert/pepin/internal/provider"
)

// providerCmd regroupe tout ce qui touche aux providers déclaratifs
// (providers/<nom>.yaml) : lister, valider, créer. Sans sous-commande, il liste.
var providerCmd = &cobra.Command{
	Use:     "provider",
	Aliases: []string{"providers"},
	Short:   "Gérer les providers déclaratifs (lister, valider, créer)",
	Run:     func(_ *cobra.Command, _ []string) { listProviders() },
}

// providerListCmd liste les providers enregistrés (descripteurs chargés).
var providerListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "Lister les providers cloud disponibles",
	Run:     func(_ *cobra.Command, _ []string) { listProviders() },
}

// listProviders affiche les providers enregistrés (nom + description).
func listProviders() {
	fmt.Println()
	fmt.Println(eyebrow.Render("// pépin") + muted.Render("  providers enregistrés"))
	for _, p := range provider.All() {
		fmt.Printf("  %s  %s\n", titre.Render(p.Name()), muted.Render(p.Description()))
	}
	fmt.Println()
}

// providerValidateCmd valide les descripteurs contre le contrat genprovider.
var providerValidateCmd = &cobra.Command{
	Use:   "validate [dossier]",
	Short: "Valider les providers d'un dossier (défaut : providers/) contre le contrat",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		dir := "providers"
		if len(args) > 0 {
			dir = args[0]
		}
		res, err := genprovider.ValidateAll(os.DirFS(dir), ".")
		if err != nil {
			return err
		}
		names := make([]string, 0, len(res))
		for n := range res {
			names = append(names, n)
		}
		sort.Strings(names)

		fmt.Println()
		fmt.Println(eyebrow.Render("// pépin") + muted.Render("  validation des providers — "+dir))
		conforme := true
		for _, n := range names {
			errs := res[n]
			if len(errs) == 0 {
				fmt.Printf("  %s  %s\n", titre.Render("✓"), n)
				continue
			}
			conforme = false
			fmt.Printf("  %s  %s\n", errStyle.Render("✗"), titre.Render(n))
			for _, e := range errs {
				fmt.Printf("      %s\n", errStyle.Render("- "+e))
			}
		}
		fmt.Println()
		if !conforme {
			os.Exit(exitNonConformite)
		}
		return nil
	},
}

// providerNewCmd scaffolde un providers/<nom>.yaml commenté à compléter.
var providerNewCmd = &cobra.Command{
	Use:   "new <nom>",
	Short: "Créer le squelette d'un provider (providers/<nom>.yaml)",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		name := args[0]
		path := filepath.Join("providers", name+".yaml")
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s existe déjà", path)
		}
		content := strings.ReplaceAll(providerSkeleton, "{{name}}", name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil { // #nosec G306 -- descripteur de provider versionné (config partageable), pas un secret.
			return fmt.Errorf("écriture de %s : %w", path, err)
		}
		fmt.Printf("Créé : %s\n", path)
		fmt.Println("Complétez auth/credentials/collecte/mapping_terraform, puis : pepin provider validate")
		return nil
	},
}

func init() {
	providerCmd.AddCommand(providerListCmd, providerValidateCmd, providerNewCmd)
}

// providerSkeleton est le gabarit d'un descripteur de provider.
const providerSkeleton = `# Provider {{name}} — descripteur déclaratif (chargé par internal/genprovider).
# Aucun code Go : ce fichier décrit l'identité, l'authentification, la résolution
# des identifiants, et les deux sources (collecte API live + mapping Terraform).
name: {{name}}
description: "{{name}} — à décrire"
# region_key: zone        # si --region alimente une autre clé que "region"

auth:
  type: header            # header | sigv4 | exoscale-hmac
  header: X-Auth-Token    # (header) nom de l'en-tête
  value: "{secret_key}"   # (header) valeur ; ou (sigv4) service/region :
  # service: oapi
  # region: eu-west-2

credentials:
  env:
    access_key: {{name}}_ACCESS_KEY
    secret_key: {{name}}_SECRET_KEY
    region: {{name}}_REGION
  file: { path: "~/.config/{{name}}/config.yaml", format: scw }   # scw | osc | exoscale
  defaults: { region: "" }

s3:
  endpoint: "https://s3.{region}.example.com"   # vide si pas de stockage objet
  # region: "{region}"

collecte:                 # source : API live (moteur internal/collect)
  base_url: "https://api.{region}.example.com"
  resources:
    - type: security_group_rule
      path: /...
      items: ...
      id: security_group_id
      map: {}

mapping_terraform:        # source : plan Terraform (moteur internal/tfmap)
  resources:
    - tf_type: {{name}}_security_group_rule
      type: security_group_rule
      id: security_group_id
      map: {}
`

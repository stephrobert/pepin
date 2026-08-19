package cmd

// Localisation de la CLI.
//
// Cobra FIGE les chaînes d'aide à l'initialisation des variables de commande :
// un `Short:` écrit dans un littéral de struct est évalué avant que la moindre
// option n'ait été lue, donc avant qu'on sache dans quelle langue parler. Les
// littéraux restent donc en français (langue de référence, et ce qu'on lit dans
// le code), et `localize` les REMPLACE une fois la langue résolue, juste avant
// `Execute` et juste après la lecture de --lang.
//
// Les chaînes produites PENDANT l'exécution (messages d'erreur, verdict,
// bandeau) n'ont pas ce problème : `tr` y est appelée au moment du rendu.

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/stephrobert/pepin/internal/i18n"
)

// langFlag porte --lang. Déclaré en persistant sur la racine : la langue vaut
// pour toutes les sous-commandes, jamais pour une seule.
var langFlag string

// tr choisit entre la chaîne française (référence) et sa contrepartie anglaise.
func tr(fr, en string) string { return i18n.T(fr, en) }

// brandEyebrow rend le sur-titre des sorties propres à la CLI de Pépin. L'accent
// tombe en anglais : la marque reste « Pépin », mais une sortie anglaise n'a aucune
// raison de porter un caractère que son lecteur ne tape pas.
func brandEyebrow() string { return tr("// pépin", "// pepin") }

// langFromArgs extrait la valeur de --lang des arguments bruts, AVANT que cobra
// ne parse quoi que ce soit. Sans cette lecture anticipée, `pepin --lang=en
// scan --help` afficherait une aide française : au moment où cobra connaît la
// valeur du drapeau, les chaînes d'aide sont déjà posées.
//
// La lecture s'arrête au séparateur `--` : ce qui le suit est un argument
// positionnel, pas une option de Pépin.
func langFromArgs(args []string) string {
	for i, a := range args {
		if a == "--" {
			return ""
		}
		if v, ok := strings.CutPrefix(a, "--lang="); ok {
			return v
		}
		if a == "--lang" && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

// setUsage remplace le texte d'aide d'un drapeau déjà enregistré. Silencieux si
// le drapeau n'existe pas : `localize` ne doit jamais être la raison d'une panique
// au démarrage.
func setUsage(c *cobra.Command, name, usage string) {
	if f := c.Flags().Lookup(name); f != nil {
		f.Usage = usage
	}
}

// localize réécrit toutes les chaînes d'aide de la CLI dans la langue résolue.
// Appelée une fois, depuis Execute, avant que cobra ne construise la moindre aide.
func localize() {
	rootCmd.Short = tr(
		"Pépin — trouve les pépins de votre cloud souverain",
		"Pepin — finds the flaws in your sovereign cloud")
	rootCmd.Long = tr(
		"Pépin — CSPM multi-cloud souverain.\n\n"+
			"Évalue la posture d'un cloud (OVH, Scaleway, Exoscale, Outscale…) contre un\n"+
			"référentiel commun ancré sur SCSL, SecNumCloud, CIS et ISO.",
		"Pepin — sovereign multi-cloud CSPM.\n\n"+
			"Assesses the posture of a cloud (OVH, Scaleway, Exoscale, Outscale…) against a\n"+
			"common reference anchored on SCSL, SecNumCloud, CIS and ISO.")
	if f := rootCmd.PersistentFlags().Lookup("lang"); f != nil {
		f.Usage = tr(
			"langue de l'interface : fr | en (défaut : PEPIN_LANG, puis LC_ALL/LANG, sinon en)",
			"interface language: fr | en (default: PEPIN_LANG, then LC_ALL/LANG, otherwise en)")
	}

	scanCmd.Short = tr(
		"Évaluer la posture d'un cloud contre les politiques",
		"Assess the posture of a cloud against the policies")
	scanCmd.Long = tr(
		"Évalue un inventaire contre les règles embarquées du provider (+ règles\n"+
			"externes via --policy-dir). Trois sources : un export JSON normalisé, un plan\n"+
			"Terraform (--terraform), ou une collecte live de l'API (--live).",
		"Assesses an inventory against the embedded common rules (plus external rules\n"+
			"through --policy-dir). Three sources: a normalized JSON export, a Terraform\n"+
			"plan (--terraform), or a live collection from the provider API (--live).")
	setUsage(scanCmd, "format", tr(
		"format de sortie : table | json | assessment | oscal | sarif",
		"output format: table | json | assessment | oscal | sarif"))
	setUsage(scanCmd, "policy-dir", tr(
		"répertoire de règles externes (.rego), répétable — chargé sans recompilation",
		"directory of external rules (.rego), repeatable — loaded without recompiling"))
	setUsage(scanCmd, "terraform", tr(
		"auditer un plan Terraform (`terraform show -json`) au lieu d'un export d'inventaire",
		"audit a Terraform plan (`terraform show -json`) instead of an inventory export"))
	setUsage(scanCmd, "live", tr(
		"collecter l'inventaire en direct via l'API du provider (identifiants requis)",
		"collect the inventory live through the provider API (credentials required)"))
	setUsage(scanCmd, "region", tr(
		"région cible pour la collecte live",
		"target region for the live collection"))
	setUsage(scanCmd, "kubeconfig", tr(
		"chemin d'un kubeconfig pour auditer l'état DANS un cluster Kubernetes (utiliser un accès en LECTURE SEULE, TTL court — jamais cluster-admin)",
		"path to a kubeconfig to audit the state INSIDE a Kubernetes cluster (use READ-ONLY, short-lived access — never cluster-admin)"))
	setUsage(scanCmd, "profile", tr(
		"profil d'identifiants pour la collecte live (ex. ~/.osc/config.json)",
		"credentials profile for the live collection (e.g. ~/.osc/config.json)"))
	setUsage(scanCmd, "s3-endpoint", tr(
		"endpoint S3 custom pour le stockage objet (collecte live ; ex. MinIO http://localhost:9000)",
		"custom S3 endpoint for object storage (live collection; e.g. MinIO http://localhost:9000)"))
	setUsage(scanCmd, "seal", tr(
		"écrire un bundle de preuve opposable (assessment + OSCAL + manifest + checksums) dans ce dossier",
		"write a defensible evidence bundle (assessment + OSCAL + manifest + checksums) into this directory"))
	setUsage(scanCmd, "exceptions", tr(
		"`fichier` YAML de dérogations (control, justification, expires_at, owner, approved_by) — un écart couvert passe au statut exempted, jamais conforme",
		"exemptions YAML `file` (control, justification, expires_at, owner, approved_by) — a covered deviation becomes exempted, never compliant"))
	setUsage(scanCmd, "strict", tr(
		"porte CI stricte : code de sortie ≠ 0 si aucun contrôle n'est mesuré (hors gouvernance) ou s'il subsiste un écart medium/low",
		"strict CI gate: non-zero exit code if no control is measured (governance aside) or if medium/low deviations remain"))
	setUsage(scanCmd, "redact", tr(
		"caviarder les valeurs sensibles (user-data, policies) de l'input.json du bundle — pour partage à un tiers ; INCOMPATIBLE avec verify --re-derive",
		"redact the sensitive values (user-data, policies) from the bundle's input.json — for sharing with a third party; INCOMPATIBLE with verify --re-derive"))

	providerCmd.Short = tr(
		"Gérer les providers déclaratifs (lister, valider, créer)",
		"Manage the declarative providers (list, validate, create)")
	providerListCmd.Short = tr(
		"Lister les providers cloud disponibles",
		"List the available cloud providers")
	providerValidateCmd.Short = tr(
		"Valider les providers d'un dossier (défaut : providers/) contre le contrat",
		"Validate the providers of a directory (default: providers/) against the contract")
	providerNewCmd.Short = tr(
		"Créer le squelette d'un provider (providers/<nom>.yaml)",
		"Create the skeleton of a provider (providers/<name>.yaml)")

	versionCmd.Short = tr("Afficher la version", "Print the version")

	scslCmd.Short = tr(
		"Vérifier la cohérence avec l'index SCSL et piloter la roadmap",
		"Check consistency with the SCSL index and drive the roadmap")
	setUsage(scslCmd, "index", tr(
		"chemin de l'API SCSL (api/v1/exigences.json du framework)",
		"path to the SCSL API (the framework's api/v1/exigences.json)"))

	verifyCmd.Short = tr(
		"Vérifier l'intégrité (et la signature) d'un bundle de preuve",
		"Verify the integrity (and the signature) of an evidence bundle")
	verifyCmd.Long = tr(
		"Recalcule l'empreinte SHA-256 de chaque fichier listé dans checksums.txt et signale\n"+
			"toute altération. C'est la vérification tierce d'un bundle produit par `scan --seal`.\n\n"+
			"Sans --pubkey, seule l'intégrité (altération accidentelle) est vérifiée : un attaquant\n"+
			"peut régénérer fichiers + checksums. Avec --pubkey, la SIGNATURE cosign de checksums.txt\n"+
			"est vérifiée (non-répudiation) — l'opérateur ayant scellé le bundle avec (cosign 3.x) :\n"+
			"  cosign sign-blob --key cosign.key --bundle checksums.txt.bundle checksums.txt",
		"Recomputes the SHA-256 digest of every file listed in checksums.txt and reports any\n"+
			"tampering. This is the third-party verification of a bundle produced by `scan --seal`.\n\n"+
			"Without --pubkey, only integrity (accidental alteration) is verified: an attacker can\n"+
			"regenerate both files and checksums. With --pubkey, the cosign SIGNATURE of checksums.txt\n"+
			"is verified (non-repudiation) — the operator having sealed the bundle with (cosign 3.x):\n"+
			"  cosign sign-blob --key cosign.key --bundle checksums.txt.bundle checksums.txt")
	setUsage(verifyCmd, "pubkey", tr(
		"clé publique cosign pour vérifier la signature de checksums.txt",
		"cosign public key used to verify the signature of checksums.txt"))
	setUsage(verifyCmd, "bundle", tr(
		"bundle de signature cosign (défaut : <dossier>/checksums.txt.bundle)",
		"cosign signature bundle (default: <directory>/checksums.txt.bundle)"))
	setUsage(verifyCmd, "re-derive", tr(
		"rejouer les règles sur input.json et vérifier que l'assessment scellé en découle (opposabilité forte)",
		"replay the rules on input.json and check that the sealed assessment follows from it (strong defensibility)"))
}

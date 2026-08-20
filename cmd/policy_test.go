package cmd

// Les deux propriétés directrices de la configuration des contrôles, mesurées sur
// le PRODUIT : on compile le binaire, on lui donne un fichier de politique, et on
// lit ce qu'il imprime, ce qu'il scelle et ce qu'il rend comme code de sortie.
//
//  1. À configuration par DÉFAUT, aucun verdict ne bouge. Un réglage qui change le
//     comportement par défaut n'est pas un réglage, c'est un changement de contrôle
//     déguisé.
//  2. Aucune configuration ne peut produire un `pass` que la configuration par
//     défaut ne produirait pas SANS que le rapport le dise — dans le rendu
//     terminal, dans l'assessment, dans les formats analysables ET dans le bundle
//     scellé. Une configuration qui n'apparaît pas dans la preuve est une porte
//     dérobée vers le vert.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// policyFixture : le plan Terraform non conforme du dépôt, le même que celui des
// dérogations. Il porte l'écart d'étiquetage sur lequel la politique se règle.
const policyFixture = "examples/scaleway/terraform/plan.json"

// writePolicy écrit un fichier de politique jetable et rend son chemin ABSOLU :
// le binaire tourne depuis la racine du dépôt.
func writePolicy(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "pepin-policy.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("écriture du fichier de politique : %v", err)
	}
	return path
}

// defaultPolicy : le profil par défaut, écrit EXPLICITEMENT. Le donner à un scan
// doit produire, octet pour octet, la sortie d'un scan sans fichier du tout.
const defaultPolicy = `controls:
  tagging:
    required_tags: [CostCenter, Project, Env, Owner]
    network_required_tags: [Owner, Project, Env]
    aliases:
      CostCenter: [CostCenter, cost-center, cc, billing-code, billing]
      Project: [Project, app, application, service]
      Env: [Env, environment, stage]
      Owner: [Owner, team, responsible, contact]
    resource_types:
      - compute_instance
      - blockstorage_volume
      - blockstorage_snapshot
      - compute_image
      - load_balancer
      - object_storage_bucket
      - managed_database
      - kubernetes_cluster
  snapshots:
    max_age_days: 7
    accepted_states: [completed, created]
  secrets:
    min_confidence: low
`

// TestAnExplicitDefaultPolicyMovesNoVerdict est la PROPRIÉTÉ 1, mesurée de bout en
// bout : le profil par défaut écrit à la main et l'absence de fichier rendent la
// même sortie et le même code. Sans elle, « configurable » voudrait dire « le
// comportement dépend de si vous avez écrit un fichier », ce qui n'est pas un
// réglage mais un piège.
func TestAnExplicitDefaultPolicyMovesNoVerdict(t *testing.T) {
	bin := buildPepin(t)
	bare, _, bareCode := runScan(t, bin, "scan", "scaleway", "-t", policyFixture)
	withFile, _, fileCode := runScan(t, bin, "scan", "scaleway", "-t", policyFixture,
		"--policy", writePolicy(t, defaultPolicy))

	if bareCode != fileCode {
		t.Errorf("code de sortie %d sans fichier, %d avec le profil par défaut écrit — un réglage par défaut ne doit rien déplacer", bareCode, fileCode)
	}
	if bare != withFile {
		t.Errorf("la sortie a bougé sous le profil par défaut écrit explicitement :\n%s", firstDiff(bare, withFile))
	}
}

// TestADefaultScanDeclaresItsConfiguration : même sans fichier, le rapport DIT
// sous quelle configuration il a été rendu. Une preuve muette sur sa politique
// laisserait un lecteur supposer le défaut sans pouvoir le vérifier.
func TestADefaultScanDeclaresItsConfiguration(t *testing.T) {
	bin := buildPepin(t)
	stdout, _, _ := runScan(t, bin, "scan", "scaleway", "-t", policyFixture, "--format", "json")
	var doc struct {
		Config struct {
			PolicyDigest string `json:"policy_digest"`
			Relaxations  []any  `json:"relaxations"`
			Effective    struct {
				Snapshots struct {
					MaxAgeDays int `json:"max_age_days"`
				} `json:"snapshots"`
				Secrets struct {
					MinConfidence string `json:"min_confidence"`
				} `json:"secrets"`
			} `json:"effective"`
		} `json:"config"`
	}
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("sortie JSON illisible : %v", err)
	}
	if !strings.HasPrefix(doc.Config.PolicyDigest, "sha256:") {
		t.Errorf("la configuration doit porter son empreinte, obtenu %q", doc.Config.PolicyDigest)
	}
	if len(doc.Config.Relaxations) != 0 {
		t.Errorf("un scan par défaut ne doit annoncer AUCUN assouplissement, obtenu %d", len(doc.Config.Relaxations))
	}
	if doc.Config.Effective.Snapshots.MaxAgeDays != 7 || doc.Config.Effective.Secrets.MinConfidence != "low" {
		t.Errorf("la configuration effective publiée ne correspond pas au profil par défaut : %+v", doc.Config.Effective)
	}
}

// relaxedPolicy : une politique qui desserre les trois contrôles réglables, dans le
// sens que les contraintes normatives refusent.
const relaxedPolicy = `controls:
  tagging:
    required_tags: [Owner]
  snapshots:
    max_age_days: 90
  secrets:
    min_confidence: high
`

// TestARelaxedConfigurationIsVisibleEverywhere est la PROPRIÉTÉ 2, mesurée aux
// QUATRE endroits où un lecteur peut arriver : le terminal, l'assessment, les
// formats analysables et le bundle scellé. Un seul de ces quatre qui se tait
// suffirait à faire de la configuration une porte dérobée vers le vert.
func TestARelaxedConfigurationIsVisibleEverywhere(t *testing.T) {
	bin := buildPepin(t)
	pol := writePolicy(t, relaxedPolicy)

	t.Run("terminal", func(t *testing.T) {
		stdout, _, _ := runScan(t, bin, "scan", "scaleway", "-t", policyFixture, "--policy", pol)
		if !strings.Contains(stdout, "RELAXED CONFIGURATION") {
			t.Errorf("le rendu terminal ne nomme pas l'assouplissement :\n%s", stdout)
		}
		if !strings.Contains(stdout, "governance_resource_required_tags") {
			t.Errorf("le rendu terminal ne nomme pas le contrôle assoupli :\n%s", stdout)
		}
		if !strings.Contains(stdout, "scsl:CLD-GVN-1") {
			t.Errorf("le rendu terminal ne nomme pas la correspondance abandonnée :\n%s", stdout)
		}
	})

	t.Run("assessment", func(t *testing.T) {
		stdout, _, _ := runScan(t, bin, "scan", "scaleway", "-t", policyFixture,
			"--policy", pol, "--format", "assessment")
		var doc struct {
			Results []struct {
				Control    string            `json:"control"`
				References []any             `json:"references"`
				Labels     map[string]string `json:"labels"`
				Evidence   struct {
					Observed string `json:"observed"`
				} `json:"evidence"`
			} `json:"results"`
		}
		if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
			t.Fatalf("assessment illisible : %v", err)
		}
		seen := false
		for _, r := range doc.Results {
			if r.Control != "governance_resource_required_tags" {
				continue
			}
			seen = true
			if len(r.References) != 0 {
				t.Errorf("un contrôle assoupli garde ses correspondances normatives : %+v", r.References)
			}
			if r.Labels["config_relaxed"] != "true" {
				t.Errorf("le résultat ne porte pas `config_relaxed` : %+v", r.Labels)
			}
			if !strings.Contains(r.Labels["references_dropped"], "CLD-GVN-1") {
				t.Errorf("le résultat ne nomme pas ce qui est abandonné : %q", r.Labels["references_dropped"])
			}
			if !strings.Contains(r.Evidence.Observed, "RELAXED CONFIGURATION") {
				t.Errorf("la preuve ne porte pas l'assouplissement : %q", r.Evidence.Observed)
			}
		}
		if !seen {
			t.Fatal("le contrôle assoupli est absent de l'assessment")
		}
	})

	t.Run("format analysable", func(t *testing.T) {
		stdout, _, _ := runScan(t, bin, "scan", "scaleway", "-t", policyFixture,
			"--policy", pol, "--format", "json")
		var doc struct {
			Config struct {
				Relaxations []struct {
					Control           string   `json:"control"`
					Parameter         string   `json:"parameter"`
					Default           string   `json:"default"`
					Effective         string   `json:"effective"`
					DroppedReferences []string `json:"dropped_references"`
				} `json:"relaxations"`
				DroppedReferences []string `json:"dropped_references"`
			} `json:"config"`
		}
		if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
			t.Fatalf("sortie JSON illisible : %v", err)
		}
		if len(doc.Config.Relaxations) == 0 {
			t.Fatal("`--format json` ne publie aucun assouplissement")
		}
		if len(doc.Config.DroppedReferences) == 0 {
			t.Error("`--format json` ne publie pas les correspondances abandonnées")
		}
		for _, r := range doc.Config.Relaxations {
			if r.Default == "" || r.Effective == "" || r.Default == r.Effective {
				t.Errorf("un assouplissement doit publier deux valeurs distinctes : %+v", r)
			}
		}
	})

	t.Run("bundle scellé", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "bundle")
		runScan(t, bin, "scan", "scaleway", "-t", policyFixture, "--policy", pol, "--seal", dir)

		manifest := readJSON(t, filepath.Join(dir, "manifest.json"))
		cfg, ok := manifest["config"].(map[string]any)
		if !ok {
			t.Fatalf("le manifeste ne porte pas la configuration appliquée : %v", manifest["config"])
		}
		if relaxed, _ := cfg["relaxed_controls"].([]any); len(relaxed) == 0 {
			t.Error("le manifeste ne nomme aucun contrôle assoupli")
		}
		if dropped, _ := cfg["dropped_references"].([]any); len(dropped) == 0 {
			t.Error("le manifeste ne nomme aucune correspondance abandonnée")
		}
		// L'artefact est SCELLÉ : il figure au manifeste et à checksums.txt, donc le
		// digest du dossier dépend des réglages. Un dossier ne peut pas taire
		// l'assouplissement sous lequel il a été rendu sans se trahir.
		sums, err := os.ReadFile(filepath.Join(dir, "checksums.txt")) // #nosec G304 -- dossier de test.
		if err != nil {
			t.Fatalf("checksums.txt illisible : %v", err)
		}
		if !strings.Contains(string(sums), "config.json") {
			t.Errorf("config.json n'est pas scellé :\n%s", sums)
		}
		doc := readJSON(t, filepath.Join(dir, "config.json"))
		if doc["notice"] == nil {
			t.Error("config.json ne porte pas l'avertissement bilingue d'assouplissement")
		}
		// La configuration EFFECTIVE voyage aussi dans l'inventaire scellé : c'est
		// elle qui rend `verify --re-derive` fidèle.
		input := readJSON(t, filepath.Join(dir, "input.json"))
		if input["config"] == nil {
			t.Error("input.json ne porte pas la configuration effective : un rejeu appliquerait la politique du jour")
		}
	})
}

// readJSON lit un document JSON d'un bundle.
func readJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path) // #nosec G304 -- dossier de test.
	if err != nil {
		t.Fatalf("lecture de %s : %v", path, err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("%s illisible : %v", path, err)
	}
	return out
}

// TestARelaxedConfigurationFailsTheStrictGate : `--strict` est la porte de CI plus
// exigeante. Elle refusait déjà une couverture nulle, un écart medium/low et un
// fichier de dérogations périmé ; elle refuse maintenant aussi une correspondance
// normative tombée. Un pipeline qui vend de la conformité ne doit pas rendre 0 sur
// un contrôle qu'il a lui-même abaissé.
func TestARelaxedConfigurationFailsTheStrictGate(t *testing.T) {
	bin := buildPepin(t)
	// Une fixture CONFORME : sans elle, le code 1 de la non-conformité masquerait
	// ce que la porte stricte ajoute, et le test ne mesurerait rien.
	_, _, code := runScan(t, bin, "scan", "scaleway", "-t", "examples/scaleway/terraform-fixed/plan.json",
		"--strict", "--policy", writePolicy(t, relaxedPolicy))
	if code == exitConforme {
		t.Error("une correspondance normative tombée a rendu 0 sous --strict — un assouplissement ne doit pas passer la porte stricte en silence")
	}
	if code != exitStrict {
		t.Errorf("code de sortie %d, attendu %d (porte stricte)", code, exitStrict)
	}
}

// TestATightenedConfigurationDropsNoMapping : durcir n'est pas assouplir. Sans
// cette propriété, la seule configuration confortable serait le défaut, et la
// configurabilité n'aurait servi à personne.
func TestATightenedConfigurationDropsNoMapping(t *testing.T) {
	bin := buildPepin(t)
	tightened := `controls:
  tagging:
    required_tags: [CostCenter, Project, Env, Owner, DataClassification]
  snapshots:
    max_age_days: 1
`
	stdout, _, _ := runScan(t, bin, "scan", "scaleway", "-t", policyFixture,
		"--policy", writePolicy(t, tightened), "--format", "json")
	var doc struct {
		Config struct {
			Relaxations []any `json:"relaxations"`
		} `json:"config"`
	}
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("sortie JSON illisible : %v", err)
	}
	if len(doc.Config.Relaxations) != 0 {
		t.Errorf("un durcissement a été signalé comme un assouplissement : %+v", doc.Config.Relaxations)
	}
}

// TestTwoPolicyFilesAreRefused : `--policy` et `--exceptions` désignent le MÊME
// fichier, sous deux noms. En accepter deux différents, c'est garantir que l'un des
// deux dérivera — et une politique qui diverge d'elle-même n'engage plus personne.
func TestTwoPolicyFilesAreRefused(t *testing.T) {
	bin := buildPepin(t)
	_, stderr, code := runScan(t, bin, "scan", "scaleway", "-t", policyFixture,
		"--policy", writePolicy(t, defaultPolicy),
		"--exceptions", writePolicy(t, defaultPolicy))
	if code == exitConforme {
		t.Fatal("deux fichiers de politique ont été acceptés")
	}
	if !strings.Contains(stderr, "policy") || !strings.Contains(stderr, "exceptions") {
		t.Errorf("le refus doit nommer les deux drapeaux :\n%s", stderr)
	}
}

// TestAnExceptionsFileStillWorksUnderItsHistoricName : le fichier de dérogations
// d'hier reste un fichier de politique valide. Sans cette propriété, l'unification
// aurait cassé toutes les invocations en place — et un opérateur aurait vu ses
// dérogations disparaître d'un rapport, sans rien avoir changé.
func TestAnExceptionsFileStillWorksUnderItsHistoricName(t *testing.T) {
	bin := buildPepin(t)
	_, _, code := runScan(t, bin, "scan", "scaleway", "-t", exemptFixture,
		"--exceptions", writeExceptions(t, allDeviations))
	if code != exitDerogation {
		t.Errorf("code de sortie %d, attendu %d — un fichier `--exceptions` doit continuer de fonctionner", code, exitDerogation)
	}
}

// TestASecretConfidenceTravelsInTheParsableFormats : le niveau de confiance de
// chaque détection est publié (issue #48), et la VALEUR du secret ne l'est jamais.
func TestASecretConfidenceTravelsInTheParsableFormats(t *testing.T) {
	bin := buildPepin(t)
	stdout, _, _ := runScan(t, bin, "scan", "scaleway", "-t", policyFixture, "--format", "json")
	var doc struct {
		Findings []struct {
			Labels map[string]string `json:"labels"`
		} `json:"findings"`
	}
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("sortie JSON illisible : %v", err)
	}
	levels := map[string]bool{"low": true, "medium": true, "high": true}
	found := 0
	for _, f := range doc.Findings {
		if f.Labels["check"] != "compute_instance_no_secrets_in_user_data" {
			continue
		}
		found++
		if !levels[f.Labels["confidence"]] {
			t.Errorf("détection sans niveau de confiance lisible : %q", f.Labels["confidence"])
		}
	}
	if found == 0 {
		t.Fatal("aucune détection de secret dans la fixture : le test ne mesure rien")
	}
}

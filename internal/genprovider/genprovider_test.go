package genprovider

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stephrobert/pepin/internal/tfmap"
)

const providersDir = "../../providers"

// TestProvidersValid : chaque providers/<nom>.yaml respecte le CONTRAT
// (genprovider.Validate) — identité, auth, identifiants, sources non vides.
func TestProvidersValid(t *testing.T) {
	res, err := ValidateAll(os.DirFS(providersDir), ".")
	if err != nil {
		t.Fatalf("validation des providers : %v", err)
	}
	if len(res) == 0 {
		t.Fatal("aucun provider trouvé")
	}
	for name, errs := range res {
		for _, e := range errs {
			t.Errorf("%s : %s", name, e)
		}
	}
}

// TestProviderMappingsMatchSchema ANCRE le mapping Terraform de chaque provider
// sur le schéma réel du provider Terraform (anti-invention §2). Ignoré par
// provider si terraform/le schéma ne sont pas disponibles.
func TestProviderMappingsMatchSchema(t *testing.T) {
	var verifies, sautes []string
	entries, err := os.ReadDir(providersDir)
	if err != nil {
		t.Skipf("dossier providers indisponible : %v", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".yaml")
		desc, err := Load(os.DirFS(providersDir), e.Name())
		if err != nil {
			t.Errorf("%s : %v", name, err)
			continue
		}
		missing, ok := tfmap.CheckSchema(filepath.Join("../../examples", name, "terraform"), desc.MappingTerraform)
		if !ok {
			// UN SAUT QUI SE VOIT. `CheckSchema` rend ok=false quand terraform manque
			// ou quand le dossier d'exemple n'a jamais été initialisé — et il rendait
			// alors la porte muette pour ce fournisseur, sans le dire. Mesuré :
			// `examples/outscale/terraform` n'a pas de `.terraform/`, si bien que le
			// mapping Outscale n'était ancré sur AUCUN schéma, depuis toujours.
			// Une porte qui ne tourne pas est une porte qui ment sur sa couverture.
			sautes = append(sautes, name)
			continue
		}
		verifies = append(verifies, name)
		for _, m := range missing {
			t.Errorf("%s : %s", name, m)
		}
	}
	if len(sautes) > 0 {
		t.Logf("schéma non lu, donc mapping NON ancré, pour : %v\n"+
			"  (dossier d'exemple non initialisé — `terraform init` l'y ramènerait)", sautes)
	}
	if len(verifies) == 0 {
		// UNE PORTE QUI NE TOURNE PAS LE DIT, plutôt que de passer au vert.
		//
		// `CheckSchema` a besoin d'un `terraform providers schema -json`, donc d'un
		// dossier d'exemple RÉELLEMENT initialisé — ce qui suppose d'avoir téléchargé
		// les providers. La CI ne le fait pas : mesuré, elle saute les quatre
		// fournisseurs. Cette porte n'y a donc jamais tourné, et elle y comptait
		// pourtant comme verte.
		//
		// Échouer serait faux : l'environnement n'est pas fautif, il est simplement
		// dépourvu. Passer en silence serait pire : la matrice de couverture
		// afficherait un ancrage que personne n'a fait. Un SKIP nommé est la seule
		// réponse honnête — il se voit dans la sortie, et il ne prétend rien.
		t.Skipf("aucun fournisseur ancré sur un schéma : cette porte n'a pas tourné.\n" +
			"  `terraform` dans le PATH, et `terraform init` dans examples/<fournisseur>/terraform,\n" +
			"  la rendraient effective. Voir #260.")
	}
}

// TestContractVerifiedTypesAreCollected — invariant de cohérence (audit A.3) : un type de
// ressource déclaré `verifie` au contrat DOIT être réellement collecté (live) ou mappé
// (Terraform), sinon les règles qui le consomment sont MORTES (chargées mais sans donnée).
func TestContractVerifiedTypesAreCollected(t *testing.T) {
	entries, err := os.ReadDir(providersDir)
	if err != nil {
		t.Skipf("dossier providers indisponible : %v", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".yaml")
		desc, err := Load(os.DirFS(providersDir), e.Name())
		if err != nil {
			t.Errorf("%s : %v", name, err)
			continue
		}
		collected := map[string]bool{}
		for _, r := range desc.Collecte.Resources {
			collected[r.Type] = true
		}
		for _, r := range desc.MappingTerraform.Resources {
			collected[r.Type] = true
		}
		// Le stockage objet est collecté par le collecteur S3 partagé (desc.S3), hors spec `collecte`.
		if desc.S3.Endpoint != "" {
			collected["object_storage_bucket"] = true
		}
		// Les clusters Kubernetes managés sont collectés par le collecteur Go OKS (desc.OKS).
		if desc.OKS.Endpoint != "" {
			collected["kubernetes_cluster"] = true
		}
		for typ, tc := range desc.Contrat.Types {
			if tc.Etat == "verifie" && !collected[typ] {
				t.Errorf("%s : type %q déclaré `verifie` au contrat mais NI collecté NI mappé (règle orpheline) — le collecter, ou passer à `a_verifier`/`absent`", name, typ)
			}
		}
	}
}

// TestEveryContractJustificationIsBilingual : toute justification de
// non-applicabilité porte sa traduction anglaise.
//
// Ces justifications ne sont pas de la décoration : ce sont elles qu'un auditeur
// lit dans `--format assessment`, dans l'OSCAL et dans le bundle scellé, en face
// d'un `not-applicable`. Un N/A sans motif n'est pas opposable ; un N/A dont le
// motif bascule au français dans un rapport anglais ne l'est pas davantage pour
// l'auditeur qui ne lit pas le français.
//
// Le contrôle porte aussi sur l'ABSENCE d'accent dans la version anglaise :
// c'est le même critère que pour le référentiel et les règles, appliqué à la
// troisième source de prose du produit.
func TestEveryContractJustificationIsBilingual(t *testing.T) {
	entries, err := os.ReadDir(providersDir)
	if err != nil {
		t.Fatalf("dossier providers indisponible : %v", err)
	}
	check := func(t *testing.T, where, fr, en string) {
		t.Helper()
		if strings.TrimSpace(fr) == "" {
			return // pas de justification française : rien à traduire (repli générique, lui traduit dans le code)
		}
		if strings.TrimSpace(en) == "" {
			t.Errorf("%s : `reason` sans `reason_en` — un rapport anglais afficherait cette justification en français", where)
			return
		}
		if en == fr {
			t.Errorf("%s : `reason_en` est identique au français — la traduction n'a pas été faite", where)
		}
		for _, r := range en {
			if r > 127 && !strings.ContainsRune("—…«»·≥≠→⚠✓", r) {
				t.Errorf("%s : `reason_en` porte le caractère non ASCII %q", where, r)
				break
			}
		}
	}
	seen := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".yaml")
		d, lerr := Load(os.DirFS(providersDir), e.Name())
		if lerr != nil {
			t.Errorf("%s : %v", name, lerr)
			continue
		}
		for _, na := range d.Contrat.NonApplicable {
			check(t, name+"/non_applicable/"+na.Control, na.Reason, na.ReasonEn)
			seen++
		}
		for typ, tc := range d.Contrat.Types {
			if tc.Etat != "absent" {
				continue
			}
			check(t, name+"/types/"+typ, tc.Reason, tc.ReasonEn)
		}
	}
	if seen == 0 {
		t.Fatal("aucune justification de non-applicabilité lue : le test ne mesurerait rien")
	}
}

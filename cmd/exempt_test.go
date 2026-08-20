package cmd

// Les critères d'acceptation des dérogations, mesurés sur le PRODUIT : on compile
// le binaire, on lui donne un fichier de dérogations, et on lit ce qu'il imprime et
// ce qu'il rend comme code de sortie.
//
// L'invariant que ces tests existent pour tenir : **une dérogation écarte, elle ne
// déclare jamais conforme**. Il est vérifié aux trois endroits où un faux vert
// pourrait naître — le code de sortie, le décompte, et les formats analysables.

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// exemptFixture : le plan Terraform non conforme du dépôt. Ses écarts critical et
// high sont exactement ce qu'une dérogation doit pouvoir couvrir.
const exemptFixture = "examples/scaleway/terraform/plan.json"

// Les deux seuls écarts critical/high du plan qu'il faut couvrir pour que la porte
// bascule. Nommer les autres serait inutile : le contrôle bascule quand il ne reste
// AUCUN écart critical/high ouvert.
const allDeviations = `exceptions:
  - control: objectstorage_bucket_public_access
    justification: "Bucket de diffusion publique assumé, contenu non sensible"
    expires_at: 2099-12-31
    owner: platform-security
    approved_by: security@example.org
  - control: compute_instance_no_secrets_in_user_data
    justification: "Jeton de bootstrap à rotation horaire, hors périmètre"
    expires_at: 2099-12-31
    owner: platform-security
    approved_by: security@example.org
  - control: database_backup_enabled
    justification: "Base de recette, restaurée depuis la production"
    expires_at: 2099-12-31
    owner: data-platform
    approved_by: security@example.org
  - control: database_encryption_at_rest_enabled
    justification: "Base de recette, aucune donnée réelle"
    expires_at: 2099-12-31
    owner: data-platform
    approved_by: security@example.org
  - control: database_service_not_open_to_internet
    justification: "Accès depuis le runner de CI, ACL en cours de pose"
    expires_at: 2099-12-31
    owner: data-platform
    approved_by: security@example.org
  - control: iam_policy_no_privilege_escalation
    justification: "Rôle de déploiement, revue trimestrielle"
    expires_at: 2099-12-31
    owner: platform-security
    approved_by: security@example.org
  - control: network_securitygroup_allow_ingress_from_internet_to_tcp_port_22
    justification: "Bastion administré, accès restreint par IP source"
    expires_at: 2099-12-31
    owner: platform-security
    approved_by: security@example.org
  - control: network_securitygroup_default_deny
    justification: "Groupe par défaut en cours de durcissement"
    expires_at: 2099-12-31
    owner: platform-security
    approved_by: security@example.org
`

// writeExceptions écrit un fichier de dérogations jetable et rend son chemin
// ABSOLU : le binaire tourne depuis la racine du dépôt.
func writeExceptions(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "exceptions.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("écriture du fichier de dérogations : %v", err)
	}
	return path
}

// runScan exécute un scan et rend ses flux et son code de sortie.
func runScan(t *testing.T, bin string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	cmd := exec.Command(bin, args...) // #nosec G204 -- binaire compilé par le test, arguments constants.
	cmd.Dir = repoRoot
	cmd.Env = []string{"NO_COLOR=1", "TERM=dumb", "PEPIN_LANG=en", "HOME=" + t.TempDir()}
	var o, e strings.Builder
	cmd.Stdout = &o
	cmd.Stderr = &e
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if !asExitError(err, &ee) {
			t.Fatalf("exécution de pepin %s : %v", strings.Join(args, " "), err)
		}
		code = ee.ExitCode()
	}
	return o.String(), e.String(), code
}

// TestAnExemptionNeverProducesAZeroExitCode : le point qui décide de tout. Une
// dérogation ne rend pas la porte verte en silence ; elle la rend d'une couleur que
// le pipeline doit avoir explicitement acceptée.
func TestAnExemptionNeverProducesAZeroExitCode(t *testing.T) {
	bin := buildPepin(t)
	_, _, code := runScan(t, bin, "scan", "scaleway", "-t", exemptFixture,
		"--exceptions", writeExceptions(t, allDeviations))

	if code == 0 {
		t.Fatal("un scan dont tous les écarts sont exemptés a rendu 0 — une exemption ne doit JAMAIS rendre vert en silence")
	}
	if code != exitDerogation {
		t.Errorf("code de sortie %d, attendu %d (dérogation appliquée)", code, exitDerogation)
	}
}

// TestAnUncoveredDeviationStillFailsTheGate : la dérogation ne couvre qu'un écart
// sur plusieurs, la porte reste rouge au code de la non-conformité. Sinon une seule
// exemption suffirait à masquer tout le reste.
func TestAnUncoveredDeviationStillFailsTheGate(t *testing.T) {
	bin := buildPepin(t)
	partial := `exceptions:
  - control: objectstorage_bucket_public_access
    justification: "Bucket de diffusion publique assumé"
    expires_at: 2099-12-31
    owner: platform-security
    approved_by: security@example.org
`
	_, _, code := runScan(t, bin, "scan", "scaleway", "-t", exemptFixture,
		"--exceptions", writeExceptions(t, partial))
	if code != exitNonConformite {
		t.Errorf("code de sortie %d, attendu %d — des écarts non couverts subsistent", code, exitNonConformite)
	}
}

// TestAnExpiredExemptionDoesNotOpenTheGate : la même dérogation, une date passée.
// Elle ne s'applique plus, la porte reste celle de la non-conformité, et
// l'expiration est dite à l'opérateur.
func TestAnExpiredExemptionDoesNotOpenTheGate(t *testing.T) {
	bin := buildPepin(t)
	expired := strings.ReplaceAll(allDeviations, "2099-12-31", "2020-01-01")
	_, stderr, code := runScan(t, bin, "scan", "scaleway", "-t", exemptFixture,
		"--exceptions", writeExceptions(t, expired))
	if code != exitNonConformite {
		t.Errorf("code de sortie %d, attendu %d — une dérogation expirée n'écarte rien", code, exitNonConformite)
	}
	if !strings.Contains(stderr, "EXPIRED") {
		t.Errorf("l'expiration n'est pas signalée sur stderr :\n%s", stderr)
	}
}

// TestAnOrphanExemptionIsReportedByTheBinary : une exception oubliée après le
// retrait d'un contrôle doit se voir, pas se taire.
func TestAnOrphanExemptionIsReportedByTheBinary(t *testing.T) {
	bin := buildPepin(t)
	orphan := `exceptions:
  - control: controle_supprime_il_y_a_deux_ans
    justification: "Exception héritée d'une ancienne campagne"
    expires_at: 2099-12-31
    owner: platform-security
    approved_by: security@example.org
`
	_, stderr, _ := runScan(t, bin, "scan", "scaleway", "-t", exemptFixture,
		"--exceptions", writeExceptions(t, orphan))
	if !strings.Contains(stderr, "ORPHAN") {
		t.Errorf("la dérogation orpheline n'est pas signalée :\n%s", stderr)
	}
}

// TestExemptedIsAFirstClassStatusInTheParsableFormats : `exempted` existe dans
// l'assessment et dans `--format json`, et n'y est jamais compté conforme.
func TestExemptedIsAFirstClassStatusInTheParsableFormats(t *testing.T) {
	bin := buildPepin(t)
	file := writeExceptions(t, allDeviations)

	// --- assessment -------------------------------------------------------
	stdout, _, _ := runScan(t, bin, "scan", "scaleway", "-t", exemptFixture, "-f", "assessment", "--exceptions", file)
	var asmt struct {
		Results []struct {
			Control string `json:"control"`
			Status  string `json:"status"`
			Waiver  *struct {
				Justification string `json:"justification"`
				Until         string `json:"until"`
			} `json:"waiver"`
			Labels map[string]string `json:"labels"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(stdout), &asmt); err != nil {
		t.Fatalf("assessment illisible : %v", err)
	}
	exempted := 0
	for _, r := range asmt.Results {
		if r.Status != "exempted" {
			continue
		}
		exempted++
		if r.Waiver == nil || r.Waiver.Justification == "" || r.Waiver.Until == "" {
			t.Errorf("%s : exempté sans justification ni échéance scellées", r.Control)
		}
		if r.Labels["exemption_owner"] == "" || r.Labels["exemption_approved_by"] == "" {
			t.Errorf("%s : exempté sans responsable ni approbateur", r.Control)
		}
	}
	if exempted == 0 {
		t.Fatal("aucun résultat `exempted` dans l'assessment")
	}

	// --- json -------------------------------------------------------------
	stdout, _, _ = runScan(t, bin, "scan", "scaleway", "-t", exemptFixture, "-f", "json", "--exceptions", file)
	var report struct {
		Findings []struct {
			Code string `json:"code"`
		} `json:"findings"`
		Summary struct {
			Conforme bool `json:"conforme"`
		} `json:"summary"`
		Exemptions *struct {
			Records []struct {
				Effect string `json:"effect"`
			} `json:"records"`
		} `json:"exemptions"`
	}
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("rapport json illisible : %v", err)
	}
	if report.Exemptions == nil || len(report.Exemptions.Records) == 0 {
		t.Fatal("`--format json` ne publie pas les dérogations : un pipeline qui accepte le code 4 ne peut pas savoir ce qu'il accepte")
	}
	if report.Summary.Conforme {
		t.Error("le résumé se déclare conforme alors que des écarts sont exemptés — un exempté n'est PAS un conforme")
	}
	if len(report.Findings) == 0 {
		t.Error("les écarts exemptés ont disparu de `findings` : une dérogation écarte de la PORTE, jamais du rapport")
	}
}

// TestTheTerminalShowsTheExemptionsInPlainSight : une exemption discrète est une
// exemption qu'on oublie de revoir.
func TestTheTerminalShowsTheExemptionsInPlainSight(t *testing.T) {
	bin := buildPepin(t)
	stdout, _, _ := runScan(t, bin, "scan", "scaleway", "-t", exemptFixture,
		"--exceptions", writeExceptions(t, allDeviations))

	for _, want := range []string{
		"EXEMPTIONS APPLIED", "NOT compliant",
		"platform-security", "security@example.org", "2099-12-31",
		"Bastion administré, accès restreint par IP source",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("le rendu terminal ne montre pas %q :\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "Verdict: compliant") {
		t.Error("le verdict se dit conforme alors que des écarts sont exemptés")
	}
	if !strings.Contains(stdout, "NON-COMPLIANT under waiver") {
		t.Errorf("le verdict ne qualifie pas la dérogation :\n%s", stdout)
	}
}

// TestAnIncompleteExemptionFileStopsTheScan : les champs obligatoires sont validés
// au chargement. Le scan s'arrête sur une erreur technique plutôt que de produire
// un rapport qui tairait la moitié de ses exceptions.
func TestAnIncompleteExemptionFileStopsTheScan(t *testing.T) {
	bin := buildPepin(t)
	incomplete := `exceptions:
  - control: objectstorage_bucket_public_access
    justification: "sans responsable ni date"
`
	_, stderr, code := runScan(t, bin, "scan", "scaleway", "-t", exemptFixture,
		"--exceptions", writeExceptions(t, incomplete))
	if code != exitErreur {
		t.Errorf("code de sortie %d, attendu %d (erreur technique)", code, exitErreur)
	}
	for _, want := range []string{"expires_at", "owner", "approved_by"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("l'erreur ne nomme pas le champ manquant %q :\n%s", want, stderr)
		}
	}
}

// TestASealedBundleCarriesItsExemptions : un dossier qui tait ses exemptions n'est
// pas opposable. Elles entrent au bundle, donc au checksums.txt que l'opérateur
// signe — et la re-dérivation les rejoue, sinon un bundle fidèle serait déclaré
// falsifié.
func TestASealedBundleCarriesItsExemptions(t *testing.T) {
	bin := buildPepin(t)
	dir := t.TempDir()
	runScan(t, bin, "scan", "scaleway", "-t", exemptFixture,
		"--exceptions", writeExceptions(t, allDeviations), "--seal", dir)

	raw, err := os.ReadFile(filepath.Join(dir, "exemptions.json")) // #nosec G304 -- dossier de test.
	if err != nil {
		t.Fatalf("exemptions.json absent du bundle : %v", err)
	}
	if !strings.Contains(string(raw), "security@example.org") {
		t.Error("exemptions.json ne porte pas l'approbateur")
	}
	checksums, err := os.ReadFile(filepath.Join(dir, "checksums.txt")) // #nosec G304 -- dossier de test.
	if err != nil {
		t.Fatalf("checksums.txt : %v", err)
	}
	if !strings.Contains(string(checksums), "exemptions.json") {
		t.Error("exemptions.json n'est pas dans checksums.txt : la dérogation ne serait pas scellée")
	}
	manifest, err := os.ReadFile(filepath.Join(dir, "manifest.json")) // #nosec G304 -- dossier de test.
	if err != nil {
		t.Fatalf("manifest.json : %v", err)
	}
	for _, want := range []string{`"exemptions"`, `"policy_digest"`, `"inventory_schema"`} {
		if !strings.Contains(string(manifest), want) {
			t.Errorf("le manifeste ne porte pas %s", want)
		}
	}
	// La re-dérivation rejoue les dérogations scellées : un bundle fidèle ne doit
	// pas être déclaré falsifié parce que le vérificateur n'a pas le fichier.
	stdout, stderr, code := runScan(t, bin, "verify", dir, "--re-derive")
	if code != 0 {
		t.Errorf("re-dérivation d'un bundle avec dérogations : code %d\n%s\n%s", code, stdout, stderr)
	}
}

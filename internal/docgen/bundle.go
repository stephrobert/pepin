package docgen

// Le cycle de vie complet d'un bundle de preuve, OBSERVÉ : sceller, lister, vérifier,
// re-dériver, altérer, caviarder.
//
// Aucune de ces sorties n'est écrite à la main. Le bundle altéré l'est réellement (un `fail`
// passé en `pass` dans l'assessment scellé), et le refus qu'on lit est celui que `pepin
// verify` a rendu. Une page qui affirmerait « le bundle serait refusé » sans le montrer
// documenterait une intention, pas un produit.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// bundleCaptures rassemble les exécutions du cycle de preuve et ce qu'elles ont écrit sur
// disque.
type bundleCaptures struct {
	seal        Capture // scan --seal
	manifest    string  // manifest.json, champs volatils neutralisés
	checksums   string  // checksums.txt, empreintes neutralisées
	files       []bundleFile
	verify      Capture // verify (intégrité seule)
	reDerive    Capture // verify --re-derive
	tampered    Capture // verify sur un bundle altéré
	redact      Capture // scan --seal --redact
	redacted    string  // extrait de l'input.json caviardé
	redactRD    Capture // verify --re-derive sur le bundle caviardé
	crossVerify Capture // verify --re-derive d'un bundle scellé dans l'AUTRE langue
	crossLang   []crossLangFile
}

// bundleFile est un fichier du bundle et le rôle que le manifest lui donne.
type bundleFile struct {
	Name string
	Role string
}

// crossLangFile dit si un fichier du bundle est identique quand le MÊME scan est scellé dans
// les deux langues. C'est un fait mesuré, pas une affirmation : le lecteur doit savoir que
// l'empreinte d'un bundle dépend de la langue.
type crossLangFile struct {
	Name      string
	Identical bool
}

// sha64 repère une empreinte SHA-256 complète. Elle change à chaque évolution des règles ET
// à chaque build (la version est injectée dans l'assessment) : la figer dans une page ferait
// diverger la documentation sans qu'aucun comportement n'ait bougé.
var sha64 = regexp.MustCompile(`[0-9a-f]{64}`)

// vcsDigest repère l'empreinte de provenance du binaire (`vcs:<commit>[+modified]`).
var vcsDigest = regexp.MustCompile(`vcs:[0-9a-f]+(\+modified)?`)

// rfc3339 repère un horodatage UTC.
var rfc3339 = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z`)

// byteCount repère la taille d'un artefact déclarée au manifest.
var byteCount = regexp.MustCompile(`"bytes": \d+`)

// uuidRe repère un UUID. L'OSCAL en porte un par document et par observation, dérivé de
// l'instant du scan : le figer dans une page ferait diverger la documentation à chaque
// exécution, sans qu'aucun comportement n'ait bougé.
var uuidRe = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)

// normalizeVolatile marque ce qui change à chaque exécution d'un document normatif :
// horodatage, identifiants dérivés de cet horodatage, empreinte de provenance du binaire.
// Ce qui décrit la POSTURE, lui, reste intact — c'est ce que la page doit montrer.
func normalizeVolatile(s string) string {
	s = vcsDigest.ReplaceAllString(s, "<provenance>")
	s = rfc3339.ReplaceAllString(s, "<timestamp>")
	return uuidRe.ReplaceAllString(s, "<uuid>")
}

// captureBundle joue le cycle complet dans un dossier jetable et rend les captures, avec les
// chemins absolus ramenés au nom que le lecteur aura tapé (`bundle`, `bundle-tampered`…).
func captureBundle(r Runner, tmp, version string) (bundleCaptures, error) {
	var bc bundleCaptures
	dir := filepath.Join(tmp, "bundle")
	tampered := filepath.Join(tmp, "bundle-tampered")
	redacted := filepath.Join(tmp, "bundle-redacted")
	// Le bundle de l'autre langue porte cette langue dans son nom : la page anglaise montre
	// donc une vérification de `bundle-fr`, et la française de `bundle-en`.
	otherLang := "en"
	if r.Lang != "fr" {
		otherLang = "fr"
	}
	other := filepath.Join(tmp, "bundle-"+otherLang)

	steps := []struct {
		into    *Capture
		args    []string
		display []string
	}{
		{&bc.seal,
			[]string{"scan", "scaleway", "--terraform", planVulnerable, "--seal", dir},
			[]string{"scan", "scaleway", "--terraform", planVulnerable, "--seal", "bundle"}},
		{&bc.verify, []string{"verify", dir}, []string{"verify", "bundle"}},
		{&bc.reDerive, []string{"verify", dir, "--re-derive"}, []string{"verify", "bundle", "--re-derive"}},
		{&bc.redact,
			[]string{"scan", "scaleway", "--terraform", planVulnerable, "--seal", redacted, "--redact"},
			[]string{"scan", "scaleway", "--terraform", planVulnerable, "--seal", "bundle-redacted", "--redact"}},
		{&bc.redactRD,
			[]string{"verify", redacted, "--re-derive"},
			[]string{"verify", "bundle-redacted", "--re-derive"}},
	}
	for _, s := range steps {
		got, err := r.Run(s.args...)
		if err != nil {
			return bundleCaptures{}, err
		}
		got.Args = s.display
		*s.into = got
	}

	raw, err := os.ReadFile(filepath.Join(dir, "manifest.json")) // #nosec G304 -- dossier jetable du générateur.
	if err != nil {
		return bundleCaptures{}, fmt.Errorf("lecture du manifest du bundle : %w", err)
	}
	bc.files, err = bundleFiles(raw)
	if err != nil {
		return bundleCaptures{}, err
	}
	bc.manifest = normalizeBundleText(string(raw), tmp, version)
	sums, err := os.ReadFile(filepath.Join(dir, "checksums.txt")) // #nosec G304 -- dossier jetable du générateur.
	if err != nil {
		return bundleCaptures{}, fmt.Errorf("lecture de checksums.txt : %w", err)
	}
	bc.checksums = normalizeBundleText(string(sums), tmp, version)

	// Altération : un `fail` scellé passé en `pass`, la falsification la plus tentante.
	if err := copyDir(dir, tampered); err != nil {
		return bundleCaptures{}, err
	}
	flipped, err := flipOneFailToPass(filepath.Join(tampered, "assessment.json"))
	if err != nil {
		return bundleCaptures{}, err
	}
	if !flipped {
		return bundleCaptures{}, fmt.Errorf("aucun résultat « fail » à altérer dans l'assessment scellé : la démonstration ne mesurerait rien")
	}
	bc.tampered, err = r.Run("verify", tampered)
	if err != nil {
		return bundleCaptures{}, err
	}
	bc.tampered.Args = []string{"verify", "bundle-tampered"}

	// Caviardage : ce que devient un attribut sensible dans l'inventaire embarqué.
	bc.redacted, err = redactedExcerpt(filepath.Join(redacted, "input.json"))
	if err != nil {
		return bundleCaptures{}, err
	}

	// Le MÊME scan, scellé dans l'AUTRE langue : quels fichiers changent, et surtout, est-ce
	// que ce bundle se vérifie depuis un shell qui ne parle pas la langue de son auteur. Un
	// faux positif de falsification serait ici le pire verdict possible, donc il se MONTRE.
	ro := r
	ro.Lang = otherLang
	if _, err := ro.Run("scan", "scaleway", "--terraform", planVulnerable, "--seal", other); err != nil {
		return bundleCaptures{}, err
	}
	bc.crossVerify, err = r.Run("verify", other, "--re-derive")
	if err != nil {
		return bundleCaptures{}, err
	}
	bc.crossVerify.Args = []string{"verify", "bundle-" + otherLang, "--re-derive"}
	bc.crossLang, err = compareBundles(dir, other)
	if err != nil {
		return bundleCaptures{}, err
	}

	for _, c := range []*Capture{&bc.seal, &bc.verify, &bc.reDerive, &bc.tampered, &bc.redact,
		&bc.redactRD, &bc.crossVerify} {
		c.Stdout = normalizeBundleText(c.Stdout, tmp, version)
		c.Stderr = normalizeBundleText(c.Stderr, tmp, version)
	}
	return bc, nil
}

// normalizeBundleText ramène un texte de bundle à ce qui est REPRODUCTIBLE : le dossier
// jetable devient le nom relatif que le lecteur aura tapé, et les empreintes, horodatages et
// versions de build sont marqués comme tels.
func normalizeBundleText(s, tmp, version string) string {
	s = strings.ReplaceAll(s, tmp+string(os.PathSeparator), "")
	s = strings.ReplaceAll(s, tmp, "")
	if version != "" {
		s = strings.ReplaceAll(s, `"`+version+`"`, `"<version>"`)
	}
	s = vcsDigest.ReplaceAllString(s, "<provenance>")
	s = sha64.ReplaceAllString(s, "<sha256>")
	s = rfc3339.ReplaceAllString(s, "<timestamp>")
	// La TAILLE d'un artefact dépend de la longueur de la version injectée au build
	// (l'assessment et l'OSCAL la portent) : la figer ferait diverger la page d'un build à
	// l'autre sans qu'aucun comportement n'ait bougé.
	s = byteCount.ReplaceAllString(s, `"bytes": "<bytes>"`)
	return s
}

// bundleFiles rend les fichiers du bundle et leur rôle, tels que le manifest les déclare.
// Le manifest et la liste de sommes ne s'y déclarent pas eux-mêmes : ils sont ajoutés avec
// le rôle que cmd/frozen_test.go leur donne dans la surface gelée du bundle.
func bundleFiles(manifest []byte) ([]bundleFile, error) {
	var m struct {
		Artifacts []struct {
			File string `json:"file"`
			Role string `json:"role"`
		} `json:"artifacts"`
	}
	if err := json.Unmarshal(manifest, &m); err != nil {
		return nil, fmt.Errorf("manifest de bundle illisible : %w", err)
	}
	if len(m.Artifacts) == 0 {
		return nil, fmt.Errorf("manifest de bundle sans artefact : le tableau des fichiers serait vide")
	}
	out := make([]bundleFile, 0, len(m.Artifacts)+2)
	for _, a := range m.Artifacts {
		out = append(out, bundleFile{Name: a.File, Role: a.Role})
	}
	out = append(out, bundleFile{Name: "manifest.json", Role: "manifest"},
		bundleFile{Name: "checksums.txt", Role: "checksums"})
	return out, nil
}

// flipOneFailToPass altère l'assessment scellé comme le ferait quelqu'un qui veut faire dire
// à un bundle l'inverse de ce qu'il atteste. Rend faux si aucun `fail` n'était présent.
func flipOneFailToPass(path string) (bool, error) {
	raw, err := os.ReadFile(path) // #nosec G304 -- dossier jetable du générateur.
	if err != nil {
		return false, fmt.Errorf("lecture de l'assessment à altérer : %w", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return false, fmt.Errorf("assessment à altérer illisible : %w", err)
	}
	results, _ := doc["results"].([]any)
	flipped := false
	for _, it := range results {
		r, ok := it.(map[string]any)
		if !ok {
			continue
		}
		if s, _ := r["status"].(string); s == "fail" && !flipped {
			r["status"] = "pass"
			flipped = true
		}
	}
	if !flipped {
		return false, nil
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return false, err
	}
	return true, os.WriteFile(path, append(out, '\n'), 0o600)
}

// redactedExcerpt rend l'attribut caviardé tel qu'il apparaît dans l'inventaire embarqué.
func redactedExcerpt(path string) (string, error) {
	raw, err := os.ReadFile(path) // #nosec G304 -- dossier jetable du générateur.
	if err != nil {
		return "", fmt.Errorf("lecture de l'input caviardé : %w", err)
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.Contains(line, "[REDACTED") {
			return strings.TrimSpace(line), nil
		}
	}
	return "", fmt.Errorf("aucune valeur caviardée dans %s : la démonstration ne mesurerait rien", path)
}

// compareBundles dit, fichier par fichier, si les deux scellés (une langue chacun) portent
// les mêmes octets.
func compareBundles(a, b string) ([]crossLangFile, error) {
	entries, err := os.ReadDir(a)
	if err != nil {
		return nil, fmt.Errorf("lecture du bundle %s : %w", a, err)
	}
	var out []crossLangFile
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		da, err := digestOf(filepath.Join(a, e.Name()))
		if err != nil {
			return nil, err
		}
		db, err := digestOf(filepath.Join(b, e.Name()))
		if err != nil {
			return nil, err
		}
		out = append(out, crossLangFile{Name: e.Name(), Identical: da == db})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	if len(out) == 0 {
		return nil, fmt.Errorf("bundle %s vide : la comparaison ne mesurerait rien", a)
	}
	return out, nil
}

// digestOf rend l'empreinte d'un fichier de bundle, HORODATAGE NEUTRALISÉ. Deux scans
// successifs ne partagent jamais leur instant d'évaluation : comparer les octets bruts
// mesurerait l'horloge, pas la langue, et le tableau conclurait n'importe quoi.
func digestOf(path string) (string, error) {
	raw, err := os.ReadFile(path) // #nosec G304 -- dossier jetable du générateur.
	if err != nil {
		return "", fmt.Errorf("lecture de %s : %w", path, err)
	}
	// Seul l'horodatage est neutralisé. Les empreintes, elles, sont GARDÉES : checksums.txt
	// n'est fait que de cela, et les effacer ferait conclure « identique » à un fichier dont
	// tout le contenu a changé.
	sum := sha256.Sum256(rfc3339.ReplaceAll(raw, []byte("<timestamp>")))
	return hex.EncodeToString(sum[:]), nil
}

// copyDir recopie un bundle (fichiers plats uniquement) vers un dossier jetable.
func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0o750); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("lecture de %s : %w", src, err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(src, e.Name())) // #nosec G304 -- dossier jetable du générateur.
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dst, e.Name()), raw, 0o600); err != nil {
			return err
		}
	}
	return nil
}

// bundleFilesTable rend les fichiers du bundle et leur rôle.
func bundleFilesTable(t bundleStrings, files []bundleFile) string {
	var b strings.Builder
	b.WriteString("| " + t.colFile + " | " + t.colRole + " |\n|---|---|\n")
	for _, f := range files {
		_, _ = fmt.Fprintf(&b, "| `%s` | `%s` |\n", f.Name, f.Role)
	}
	return b.String()
}

// crossLangTable rend, fichier par fichier, ce que la langue du scan change dans le bundle.
func crossLangTable(t bundleStrings, files []crossLangFile) string {
	var b strings.Builder
	b.WriteString("| " + t.colFile + " | " + t.colSameBytes + " |\n|---|---|\n")
	for _, f := range files {
		mark := t.differs
		if f.Identical {
			mark = t.identical
		}
		_, _ = fmt.Fprintf(&b, "| `%s` | %s |\n", f.Name, mark)
	}
	return b.String()
}

// bundleStrings porte les libellés des régions du guide de preuve.
type bundleStrings struct {
	colFile, colRole, colSameBytes string
	identical, differs             string
}

func bundleText(lang string) bundleStrings {
	if lang == "fr" {
		return bundleStrings{
			colFile: "Fichier", colRole: "Rôle déclaré au manifest",
			colSameBytes: "Mêmes octets dans les deux langues ?",
			identical:    "✅ identique", differs: "❌ diffère (l'empreinte change)",
		}
	}
	return bundleStrings{
		colFile: "File", colRole: "Role declared in the manifest",
		colSameBytes: "Same bytes in both languages?",
		identical:    "✅ identical", differs: "❌ differs (the digest changes)",
	}
}

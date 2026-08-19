// Package genprovider fournit un provider GÉNÉRIQUE entièrement piloté par un
// unique fichier déclaratif providers/<nom>.yaml (identité + auth + identifiants
// + collecte API + mapping Terraform). Il implémente provider.Provider sans code
// spécifique : ajouter un cloud = poser UN fichier YAML (aucun Go). Le seul code
// irréductible — résolution des identifiants et signature d'auth — vit ici.
package genprovider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	yaml "go.yaml.in/yaml/v3"

	"github.com/stephrobert/pepin/internal/collect"
	"github.com/stephrobert/pepin/internal/collectkit"
	"github.com/stephrobert/pepin/internal/i18n"
	"github.com/stephrobert/pepin/internal/kubeauth"
	"github.com/stephrobert/pepin/internal/model"
	"github.com/stephrobert/pepin/internal/provider"
	"github.com/stephrobert/pepin/internal/tfmap"
	"github.com/stephrobert/pepin/internal/tfparse"
)

// Descriptor est le contenu de providers/<nom>.yaml : identité, authentification,
// résolution des identifiants, endpoint S3, et les DEUX sources (collecte live +
// mapping Terraform).
type Descriptor struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Scope       string `yaml:"scope"`      // "cloud" (défaut) | "in-cluster" : une portée différente ne se compare pas en parité
	RegionKey   string `yaml:"region_key"` // clé logique alimentée par --region (défaut "region" ; Exoscale: "zone")
	Auth        struct {
		Type    string `yaml:"type"`    // header | sigv4 | exoscale-hmac
		Header  string `yaml:"header"`  // header : nom de l'en-tête
		Value   string `yaml:"value"`   // header : valeur (ex. "{secret_key}")
		Service string `yaml:"service"` // sigv4 : service signé (ex. oapi)
		Region  string `yaml:"region"`  // sigv4 : région de signature
	} `yaml:"auth"`
	Credentials struct {
		Env      map[string]string `yaml:"env"`      // clé logique -> variable d'environnement
		File     credFile          `yaml:"file"`     // fichier de config natif
		Defaults map[string]string `yaml:"defaults"` // clé logique -> valeur par défaut ({vars} substituées)
	} `yaml:"credentials"`
	S3 struct {
		Endpoint string `yaml:"endpoint"` // {region}/{zone} substitués ; vide = pas de buckets
		Region   string `yaml:"region"`   // {region} ou {zone} (défaut {region})
		SSEKMS   bool   `yaml:"sse_kms"`  // expose une clé client au niveau bucket (SSE-KMS, CLD-CHF-4)
	} `yaml:"s3"`
	EIM struct {
		InlinePolicies bool `yaml:"inline_policies"` // collecter les politiques EIM inline (chaîne à 3 niveaux)
	} `yaml:"eim"`
	OKS struct {
		Endpoint string `yaml:"endpoint"` // API Kubernetes managé (host distinct, {region} substitué) ; vide = pas de collecte OKS
	} `yaml:"oks"`
	Collecte         collect.Spec `yaml:"collecte"`          // source : API live
	MappingTerraform tfmap.Spec   `yaml:"mapping_terraform"` // source : plan Terraform
	Contrat          Contrat      `yaml:"contrat"`           // ancrage API + applicabilité SCSL
	Souverainete     Souverainete `yaml:"souverainete"`      // métadonnées de souveraineté du fournisseur (CLD-GVN-4)
}

// Souverainete porte les FAITS de souveraineté du fournisseur (siège, contrôle
// capitalistique, qualification, exposition extraterritoriale), ancrés sur des
// sources officielles (cf. `sources`). Indépendants de la collecte : projetés en
// ressource synthétique `governance_provider` évaluée par CLD-GVN-4.
type Souverainete struct {
	EUEtabli                    *bool  `yaml:"eu_etabli"`                    // siège du fournisseur établi dans l'UE
	Juridiction                 string `yaml:"juridiction"`                  // pays du siège (ex. FR, CH)
	ControleCapitalistique      string `yaml:"controle_capitalistique"`      // juridiction du contrôle ultime (FR, UE, extra_ue, a_verifier)
	SecNumCloud                 string `yaml:"secnumcloud"`                  // qualifie | en_cours | non
	ExpositionExtraterritoriale *bool  `yaml:"exposition_extraterritoriale"` // soumis à une loi extraterritoriale (ex. Cloud Act)
	Sources                     string `yaml:"sources"`                      // URLs/justification de l'ancrage
}

// Contrat : ancrage sur l'API native (état de chaque type) et applicabilité des
// contrôles (ce que l'API n'expose pas n'est pas testable). Lu par `pepin scsl`
// et par l'assessment opposable (statut not-applicable + justification).
type Contrat struct {
	Types         map[string]TypeContrat `yaml:"types"`
	NonApplicable []NAEntry              `yaml:"non_applicable"` // contrôles non testables (mécanisme inexistant), avec justification
}

// TypeContrat est l'ancrage d'un type de ressource sur l'API native.
type TypeContrat struct {
	Etat     string `yaml:"etat"`      // verifie | a_verifier | absent
	Reason   string `yaml:"reason"`    // justification quand etat=absent (mécanisme inexistant côté API, source doc)
	ReasonEn string `yaml:"reason_en"` // la même justification en anglais (Pépin est bilingue, cf. internal/i18n)
}

// NAEntry déclare un contrôle non applicable à un provider AVEC sa justification —
// indispensable pour l'opposabilité : un auditeur rejette un N/A non justifié. Accepte
// deux formes YAML : un simple code (`- objectstorage_bucket_x`, sans justification, repli
// générique) ou un mapping justifié (`- {control: ..., reason: ...}`).
type NAEntry struct {
	Control  string `yaml:"control"`   // code de contrôle agnostique
	Reason   string `yaml:"reason"`    // pourquoi le contrôle est non testable/non applicable (ancré doc)
	ReasonEn string `yaml:"reason_en"` // la même justification en anglais (Pépin est bilingue, cf. internal/i18n)
}

// UnmarshalYAML accepte la forme courte (scalaire = code) et la forme justifiée (mapping).
func (e *NAEntry) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode {
		return node.Decode(&e.Control)
	}
	type raw NAEntry
	var r raw
	if err := node.Decode(&r); err != nil {
		return err
	}
	*e = NAEntry(r)
	return nil
}

type credFile struct {
	Path   string `yaml:"path"`   // chemin (~ étendu) ; surchargé par l'env de path du format
	Format string `yaml:"format"` // scw | osc | exoscale
}

// GenericProvider implémente provider.Provider + TerraformMapper à partir du descripteur.
type GenericProvider struct{ desc Descriptor }

// Name retourne l'identifiant du provider.
func (g GenericProvider) Name() string { return g.desc.Name }

// Description retourne le libellé d'aide.
func (g GenericProvider) Description() string { return g.desc.Description }

// Collect résout les identifiants, construit l'auth et exécute la collecte (spec
// `collecte` + stockage objet) via le kit commun.
func (g GenericProvider) Collect(ctx context.Context, cfg provider.Config) ([]model.Resource, error) {
	creds, err := g.resolveCreds(cfg)
	if err != nil {
		return nil, err
	}
	auth, err := g.buildAuth(creds)
	if err != nil {
		return nil, err
	}
	vars := map[string]string{}
	for _, k := range []string{"region", "zone", "org", "host"} {
		if creds[k] != "" {
			vars[k] = creds[k]
		}
	}
	s3 := collectkit.S3{}
	if g.desc.S3.Endpoint != "" {
		regionKey := g.desc.S3.Region
		if regionKey == "" {
			regionKey = "{region}"
		}
		s3 = collectkit.S3{
			Endpoint: collectkit.FirstNonEmpty(cfg.Options["s3_endpoint"], subst(g.desc.S3.Endpoint, vars)),
			Region:   subst(regionKey, vars),
			Key:      creds["access_key"],
			Secret:   creds["secret_key"],
			SSEKMS:   g.desc.S3.SSEKMS,
		}
	}
	// Auth par kubeconfig : le serveur d'API et le client mTLS viennent du fichier, pas
	// du descripteur. Le kubeconfig porte les DROITS — utiliser un kubeconfig en lecture
	// seule (groupe readonly, TTL court), jamais cluster-admin.
	var hc *http.Client
	if g.desc.Auth.Type == "kubeconfig" {
		path := collectkit.FirstNonEmpty(cfg.Options["kubeconfig"], creds["kubeconfig"])
		if path == "" {
			return nil, fmt.Errorf(i18n.T("provider %s : chemin du kubeconfig requis (--kubeconfig ou KUBECONFIG)", "provider %s: a kubeconfig path is required (--kubeconfig or KUBECONFIG)"), g.desc.Name)
		}
		kc, err := kubeauth.Load(path, 30*time.Second)
		if err != nil {
			return nil, err
		}
		vars["server"] = kc.Server
		hc = kc.Client
	}

	oksCfg := collectkit.OKS{}
	if g.desc.OKS.Endpoint != "" {
		oksCfg = collectkit.OKS{
			Endpoint: subst(g.desc.OKS.Endpoint, vars),
			Region:   subst("{region}", vars),
			Key:      creds["access_key"],
			Secret:   creds["secret_key"],
		}
	}
	spec := g.desc.Collecte
	spec.Provider = g.desc.Name
	eimCfg := collectkit.EIM{}
	if g.desc.EIM.InlinePolicies {
		eimCfg.BaseURL = subst(g.desc.Collecte.BaseURL, vars)
	}
	return collectkit.Run(ctx, g.desc.Name, spec, auth, vars, s3, oksCfg, eimCfg, hc)
}

// MapTerraform projette un plan Terraform via la spec de mapping déclarative.
func (g GenericProvider) MapTerraform(resources []tfparse.Resource) ([]model.Resource, error) {
	spec := g.desc.MappingTerraform
	spec.Provider = g.desc.Name
	return tfmap.Apply(spec, resources), nil
}

// GovernanceResource projette les métadonnées de souveraineté du descripteur en
// une ressource synthétique `governance_provider` (indépendante de la collecte),
// évaluée par la règle CLD-GVN-4. Le second retour est faux si la souveraineté
// n'est pas renseignée pour ce provider.
func (g GenericProvider) GovernanceResource() (model.Resource, bool) {
	s := g.desc.Souverainete
	if s.EUEtabli == nil && s.Juridiction == "" && s.ControleCapitalistique == "" {
		return model.Resource{}, false
	}
	attrs := map[string]any{}
	if s.EUEtabli != nil {
		attrs["eu_established"] = *s.EUEtabli
	}
	if s.Juridiction != "" {
		attrs["jurisdiction"] = s.Juridiction
	}
	if s.ControleCapitalistique != "" {
		attrs["capital_control"] = s.ControleCapitalistique
	}
	if s.SecNumCloud != "" {
		attrs["secnumcloud"] = s.SecNumCloud
	}
	if s.ExpositionExtraterritoriale != nil {
		attrs["extraterritorial_exposure"] = *s.ExpositionExtraterritoriale
	}
	// Ces attributs sont des FAITS DÉCLARÉS au descripteur, jamais mesurés sur le
	// tenant : leur attestation le dit explicitement. C'est la même honnêteté que la
	// preuve d'un `pass` de gouvernance, portée cette fois par la donnée elle-même.
	var prov model.Provenance
	for attr := range attrs {
		prov.Attest(attr, model.Attestation{
			Origin: model.OriginDerived, Source: "descriptor:souverainete", Derived: true,
		})
	}
	return model.Resource{
		Provider:   g.desc.Name,
		Type:       "governance_provider",
		ID:         g.desc.Name,
		Name:       g.desc.Name,
		Attributes: attrs,
		Provenance: prov,
	}, true
}

// buildAuth construit le signer selon le schéma déclaré.
func (g GenericProvider) buildAuth(creds map[string]string) (collect.Auth, error) {
	a := g.desc.Auth
	switch a.Type {
	case "header":
		return collect.HeaderAuth{Header: a.Header, Value: subst(a.Value, creds)}, nil
	case "sigv4":
		return collect.SigV4Auth{Key: creds["access_key"], Secret: creds["secret_key"], Service: a.Service, Region: a.Region}, nil
	case "kubeconfig":
		// L'authentification est portée par le transport (mTLS/jeton du kubeconfig),
		// pas par une signature de requête : aucun signer à construire ici.
		return nil, nil
	case "exoscale-hmac":
		return collect.ExoscaleAuth{Key: creds["access_key"], Secret: creds["secret_key"]}, nil
	}
	return nil, fmt.Errorf(i18n.T("type d'authentification inconnu : %q (provider %s)", "unknown authentication type: %q (provider %s)"), a.Type, g.desc.Name)
}

// resolveCreds résout les identifiants : variables d'environnement, puis fichier
// de config natif (si clés manquantes), puis valeurs par défaut.
func (g GenericProvider) resolveCreds(cfg provider.Config) (map[string]string, error) {
	cr := map[string]string{}
	for logical, env := range g.desc.Credentials.Env {
		if v := os.Getenv(env); v != "" {
			cr[logical] = v
		}
	}
	regionKey := g.desc.RegionKey
	if regionKey == "" {
		regionKey = "region"
	}
	if cfg.Region != "" {
		cr[regionKey] = cfg.Region
	}
	if cr["access_key"] == "" || cr["secret_key"] == "" {
		fileCreds, err := loadCredFile(g.desc.Credentials.File, cfg.Profile)
		if err != nil {
			return nil, err
		}
		for k, v := range fileCreds {
			if cr[k] == "" {
				cr[k] = v
			}
		}
	}
	for logical, def := range g.desc.Credentials.Defaults {
		if cr[logical] == "" {
			cr[logical] = def
		}
	}
	for k, v := range cr { // résout les références {region} (ex. zone = "{region}-1")
		cr[k] = subst(v, cr)
	}
	// L'auth par kubeconfig ne repose pas sur une paire de clés : les identifiants (et les
	// droits) sont portés par le fichier lui-même, validé plus tard par kubeauth.Load.
	if g.desc.Auth.Type != "kubeconfig" && (cr["access_key"] == "" || cr["secret_key"] == "") {
		return nil, fmt.Errorf(i18n.T("identifiants %s absents : variables d'environnement ou fichier de configuration", "%s credentials missing: environment variables or configuration file"), g.desc.Name)
	}
	return cr, nil
}

// Load lit et désérialise un descripteur de provider depuis fsys.
func Load(fsys fs.FS, file string) (Descriptor, error) {
	raw, err := fs.ReadFile(fsys, file)
	if err != nil {
		return Descriptor{}, err
	}
	var desc Descriptor
	if err := yaml.Unmarshal(raw, &desc); err != nil {
		return Descriptor{}, fmt.Errorf(i18n.T("%s invalide : %w", "invalid %s: %w"), file, err)
	}
	return desc, nil
}

// RegisterAll enregistre un GenericProvider pour chaque providers/<nom>.yaml de fsys.
func RegisterAll(fsys fs.FS, root string) error {
	entries, err := fs.ReadDir(fsys, root)
	if err != nil {
		return fmt.Errorf(i18n.T("lecture du dossier des providers %s : %w", "reading the providers directory %s: %w"), root, err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names) // digest déterministe
	h := sha256.New()
	for _, name := range names {
		raw, rerr := fs.ReadFile(fsys, path.Join(root, name))
		if rerr != nil {
			return rerr
		}
		h.Write([]byte(name))
		h.Write(raw)
		desc, derr := Load(fsys, path.Join(root, name))
		if derr != nil {
			return derr
		}
		registry[desc.Name] = desc
		provider.Register(GenericProvider{desc: desc})
	}
	descriptorsDigest = "sha256:" + hex.EncodeToString(h.Sum(nil))
	return nil
}

// descriptorsDigest : empreinte de contenu des descripteurs providers chargés (déterminent
// `verified`, les N/A, la projection des attributs, la souveraineté) — pour la provenance.
var descriptorsDigest string

// DescriptorsDigest retourne l'empreinte des descripteurs providers chargés par RegisterAll.
func DescriptorsDigest() string { return descriptorsDigest }

// registry conserve les descripteurs chargés, pour exposer l'applicabilité des
// contrôles (contrats) à la roadmap `pepin scsl`.
var registry = map[string]Descriptor{}

// subst remplace les variables {clef} par leur valeur.
func subst(s string, vars map[string]string) string {
	for k, v := range vars {
		s = strings.ReplaceAll(s, "{"+k+"}", v)
	}
	return s
}

// Package docgen DÉRIVE la documentation de couverture et les sorties de commande
// montrées dans docs/ depuis les artefacts qui font foi : le référentiel commun
// (referentiel/controles.yaml), les descripteurs de fournisseurs (providers/*.yaml),
// le verrou de « pass » de l'assessment (internal/assess) et le binaire lui-même.
//
// Raison d'être : une page de couverture recopiée à la main diverge au premier contrôle
// ajouté, et une documentation de CSPM qui ment sur ce qu'il mesure est pire que pas de
// documentation. Ici, la page N'EST PAS une source : elle est le rendu d'un calcul, et
// TestGeneratedDocsAreUpToDate échoue dès que le rendu ne correspond plus au dépôt.
package docgen

import (
	"sort"
	"strings"

	"github.com/stephrobert/pepin/internal/assess"
	"github.com/stephrobert/pepin/internal/genprovider"
	"github.com/stephrobert/pepin/internal/i18n"
	"github.com/stephrobert/pepin/referentiel"
)

// Source est l'origine des données évaluées par un contrôle : un plan Terraform projeté par
// le mapping du descripteur, ou une collecte live de l'API du fournisseur. La distinction
// n'est pas cosmétique : un plan porte l'état PLANIFIÉ et ignore ce qui est « known after
// apply », une collecte live porte la configuration EFFECTIVE mais dépend des droits.
type Source string

// Les deux sources d'un scan de plan de contrôle cloud.
const (
	SourceTerraform Source = "terraform"
	SourceLive      Source = "live"
)

// Status est le verdict de couverture d'un couple (contrôle, fournisseur, source). Il décrit
// ce que Pépin PEUT conclure, pas ce qu'il a conclu sur un inventaire donné.
type Status string

// Les quatre verdicts de couverture. Leur définition exacte est rendue dans la légende de
// docs/coverage.md, et calculée par cellStatus.
const (
	// Supported : la source produit le type visé, le contrat le déclare `verifie`, et
	// l'attribut décisif est projeté → Pépin peut rendre `pass` ou `fail`.
	Supported Status = "supported"
	// Partial : la source produit le type, mais le verrou du « pass » ne peut pas être levé
	// (contrat non vérifié, ou attribut décisif non projeté) → `not-evaluated`.
	Partial Status = "partial"
	// NotApplicable : le contrat du fournisseur déclare le contrôle non testable, avec sa
	// justification (mécanisme inexistant côté API, ou type de ressource absent).
	NotApplicable Status = "not-applicable"
	// Unsupported : le contrôle n'est pas déclaré pour ce fournisseur, ou cette source ne
	// produit aucune ressource du type qu'il lit.
	Unsupported Status = "unsupported"
)

// Cell est le verdict d'une case de la matrice, avec la raison qui le rend opposable : un
// statut sans motif n'apprend rien à qui doit décider d'adopter l'outil.
type Cell struct {
	Status Status
	Reason string // actionnable, dans la langue de la matrice ; vide pour Supported
	// Undeclared distingue le seul cas d'Unsupported qui n'apprend rien : le contrôle
	// n'est pas déclaré pour ce fournisseur au référentiel. Un drapeau plutôt qu'un
	// test sur le texte du motif — un motif est traduit, un drapeau ne l'est pas.
	Undeclared bool
}

// Row est la ligne d'un contrôle dans la matrice de couverture.
type Row struct {
	Code     string
	Title    string
	Severity string
	Family   string
	SCSL     []string
	// Cells est indexé par fournisseur puis par source.
	Cells map[string]map[Source]Cell
	// RequiredAttrs liste les attributs dont la présence conditionne un « pass » (table
	// assess.RequiredAttrs). Vide = le contrôle se juge à la présence d'un finding.
	RequiredAttrs []string
	// Type est le type de ressource normalisé que le contrôle lit ("" pour la gouvernance).
	Type string
}

// Matrix est la couverture complète : les fournisseurs de plan de contrôle cloud d'un côté,
// les fournisseurs d'une autre portée (ex. `kubernetes`, in-cluster) de l'autre — les
// comparer en parité n'aurait aucun sens, aucun ne pouvant couvrir la portée de l'autre.
type Matrix struct {
	CloudProviders []string
	OtherProviders []string
	Rows           []Row
	// Descriptors garde les descripteurs chargés (souveraineté, contrat) pour le rendu.
	Descriptors map[string]genprovider.Descriptor
}

// projection est ce qu'une source d'un fournisseur sait produire : l'ensemble des types de
// ressources normalisés, et par type l'ensemble des attributs communs réellement projetés.
type projection struct {
	types map[string]bool
	attrs map[string]map[string]bool
}

func newProjection() projection {
	return projection{types: map[string]bool{}, attrs: map[string]map[string]bool{}}
}

func (p projection) add(typ string, keys ...map[string]bool) {
	if typ != "" {
		p.types[typ] = true
	}
	if p.attrs[typ] == nil {
		p.attrs[typ] = map[string]bool{}
	}
	for _, set := range keys {
		for k := range set {
			p.attrs[typ][k] = true
		}
	}
}

// BuildMatrix calcule la couverture depuis le dépôt monté à `root`. Les descripteurs sont
// enregistrés dans genprovider pour que le calcul emprunte EXACTEMENT les fonctions du scan
// (assess.Verified, genprovider.NonApplicableReason, genprovider.ControlType).
// La matrice porte de la PROSE (titres de contrôles, motifs) : elle est donc calculée
// pour une langue donnée, et la documentation bilingue en construit une par page.
func BuildMatrix(root, lang string) (Matrix, error) {
	l := i18n.Lang(lang)
	descs, err := loadDescriptors(root)
	if err != nil {
		return Matrix{}, err
	}
	goAttrs, err := goCollectorAttributes(root)
	if err != nil {
		return Matrix{}, err
	}

	nonCloud := genprovider.NonCloudProviders()
	var cloud, other []string
	for name := range descs {
		if nonCloud[name] {
			other = append(other, name)
			continue
		}
		cloud = append(cloud, name)
	}
	sort.Strings(cloud)
	sort.Strings(other)

	projections := map[string]map[Source]projection{}
	for name, desc := range descs {
		projections[name] = map[Source]projection{
			SourceLive:      liveProjection(desc, goAttrs),
			SourceTerraform: terraformProjection(desc),
		}
	}

	required := assess.RequiredAttrs()
	all := referentiel.All()
	codes := make([]string, 0, len(all))
	for code := range all {
		codes = append(codes, code)
	}
	sort.Strings(codes)

	rows := make([]Row, 0, len(codes))
	for _, code := range codes {
		ctl := all[code]
		row := Row{
			Code:          code,
			Title:         ctl.TitreIn(l),
			Severity:      ctl.Severite,
			Family:        ctl.Famille,
			SCSL:          ctl.Scsl,
			Type:          genprovider.ControlType(code),
			RequiredAttrs: required[code],
			Cells:         map[string]map[Source]Cell{},
		}
		sort.Strings(row.RequiredAttrs)
		for name := range descs {
			row.Cells[name] = map[Source]Cell{}
			for _, src := range []Source{SourceTerraform, SourceLive} {
				row.Cells[name][src] = cellStatus(l, name, ctl, src, projections[name][src], required[code])
			}
		}
		rows = append(rows, row)
	}
	return Matrix{CloudProviders: cloud, OtherProviders: other, Rows: rows, Descriptors: descs}, nil
}

// cellStatus applique, dans l'ordre, les mêmes portes que le scan : non-applicabilité
// justifiée par le contrat, périmètre déclaré au référentiel, présence du type dans la
// source, verrou `verifie` du contrat, puis projection de l'attribut décisif. Toute
// divergence avec cmd/scan.go se verrait comme une case verte que le scan rend
// « non évaluée » : c'est pourquoi le verrou lui-même (assess.Verified) est partagé.
func cellStatus(l i18n.Lang, provider string, ctl referentiel.Control, src Source, proj projection, required []string) Cell {
	code := ctl.Code
	if reason := genprovider.NonApplicableReasonIn(l, provider, code); reason != "" {
		return Cell{Status: NotApplicable, Reason: reason}
	}
	if !contains(ctl.Fournisseurs, provider) {
		return Cell{Status: Unsupported, Undeclared: true, Reason: i18n.TIn(l,
			"contrôle non déclaré pour ce fournisseur dans referentiel/controles.yaml",
			"control not declared for this provider in referentiel/controles.yaml")}
	}
	typ := genprovider.ControlType(code)
	if typ != "" && !proj.types[typ] {
		return Cell{Status: Unsupported, Reason: i18n.TIn(l,
			"cette source ne produit aucune ressource de type « "+typ+" »",
			"this source produces no resource of type \""+typ+"\"")}
	}
	if !assess.Verified(provider, code) {
		if typ == "" {
			return Cell{Status: Partial, Reason: i18n.TIn(l,
				"aucun type de ressource visé et le contrôle ne lit pas le descripteur du fournisseur : le verrou du « pass » ne peut pas être levé, le scan rend « not-evaluated » tant qu'aucun écart n'est détecté",
				"no targeted resource type, and the control does not read the provider descriptor: the \"pass\" lock cannot be lifted, so the scan returns \"not-evaluated\" as long as no deviation is detected")}
		}
		return Cell{Status: Partial, Reason: i18n.TIn(l,
			"contrat du fournisseur : le type « "+typ+" » n'est pas déclaré `verifie` ("+etatLabel(l, provider, typ)+")",
			"provider contract: type \""+typ+"\" is not declared `verifie` ("+etatLabel(l, provider, typ)+")")}
	}
	if len(required) > 0 && !anyProjected(proj.attrs[typ], required) {
		return Cell{Status: Partial, Reason: i18n.TIn(l,
			"attribut décisif « "+strings.Join(required, " / ")+" » non projeté par cette source : garde de capacité, le scan rend « not-evaluated »",
			"deciding attribute \""+strings.Join(required, " / ")+"\" not projected by this source: a capability guard, so the scan returns \"not-evaluated\"")}
	}
	return Cell{Status: Supported}
}

// etatLabel rend l'état contractuel d'un type, pour que la raison d'un `partial` cite la
// donnée du descripteur plutôt qu'une paraphrase.
func etatLabel(l i18n.Lang, provider, typ string) string {
	etat := genprovider.TypeEtat(provider, typ)
	if etat == "" {
		return i18n.TIn(l, "type absent du contrat", "type absent from the contract")
	}
	return i18n.TIn(l, "état : ", "state: ") + etat
}

func anyProjected(have map[string]bool, want []string) bool {
	for _, a := range want {
		if have[a] {
			return true
		}
	}
	return false
}

// liveProjection décrit ce que la collecte live d'un fournisseur sait produire : la spec
// `collecte` du descripteur, plus les deux collecteurs Go partagés (stockage objet S3,
// Kubernetes managé) qui ne passent pas par la spec. Le moteur de collecte pose la RÉGION
// sur chaque ressource (internal/collect/engine.go), ce qui rend la localisation observable
// en live pour tout fournisseur qui collecte quoi que ce soit.
func liveProjection(desc genprovider.Descriptor, goAttrs map[string]map[string]bool) projection {
	p := newProjection()
	for _, r := range desc.Collecte.Resources {
		p.add(r.Type, keysOfString(r.Map), keysOfAny(r.Const))
	}
	if desc.S3.Endpoint != "" {
		attrs := copySet(goAttrs["object_storage_bucket"])
		if !desc.S3.SSEKMS {
			// Le collecteur ne pose sse_kms_enabled/kms_key_id que si le descripteur
			// déclare la capacité (BYOK au niveau bucket).
			delete(attrs, "sse_kms_enabled")
			delete(attrs, "kms_key_id")
		}
		p.add("object_storage_bucket", attrs)
	}
	if desc.OKS.Endpoint != "" {
		p.add("kubernetes_cluster", copySet(goAttrs["kubernetes_cluster"]))
	}
	if len(p.types) > 0 {
		p.add("", map[string]bool{"region": true})
	}
	return p
}

// terraformProjection décrit ce que le mapping Terraform du descripteur sait produire. La
// région n'y est observable que si une spec nomme son champ (`region:`) — sur un plan, rien
// ne la pose implicitement.
func terraformProjection(desc genprovider.Descriptor) projection {
	p := newProjection()
	for _, r := range desc.MappingTerraform.Resources {
		p.add(r.Type, keysOfString(r.Map), keysOfAny(r.Const))
		if r.Region != "" {
			p.add("", map[string]bool{"region": true})
		}
	}
	return p
}

func keysOfString(m map[string]string) map[string]bool {
	out := make(map[string]bool, len(m))
	for k := range m {
		out[k] = true
	}
	return out
}

func keysOfAny(m map[string]any) map[string]bool {
	out := make(map[string]bool, len(m))
	for k := range m {
		out[k] = true
	}
	return out
}

func copySet(in map[string]bool) map[string]bool {
	out := make(map[string]bool, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func contains(vals []string, v string) bool {
	for _, x := range vals {
		if x == v {
			return true
		}
	}
	return false
}

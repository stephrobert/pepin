package docgen

// Les régions générées des pages « un fournisseur, une page ».
//
// Tout ce qu'une page de fournisseur affirme sur ce qu'il collecte est DÉRIVÉ de son
// descripteur (`providers/<nom>.yaml`) : variables d'identifiants, appels d'API live,
// ressources Terraform reconnues, non-applicabilités justifiées. Recopier ces listes à la
// main, c'est promettre une couverture que le prochain commit dément — et une page de CSPM
// qui ment sur ce qu'il regarde est pire que pas de page du tout.

import (
	"fmt"
	"sort"
	"strings"

	"github.com/stephrobert/pepin/internal/genprovider"
)

// documentedProviders : les fournisseurs de plan de contrôle cloud qui ont leur page.
// `kubernetes` en est absent à dessein : sa portée (l'intérieur d'un cluster) n'est pas
// celle d'un cloud, et docs/coverage.md le traite déjà à part.
var documentedProviders = []string{"exoscale", "outscale", "scaleway"}

// providerBlocks rend toutes les régions des trois pages de fournisseurs.
func providerBlocks(lang string, m Matrix, scans map[string]*Capture) map[string]string {
	t := providerText(lang)
	out := map[string]string{}
	for _, name := range documentedProviders {
		desc, ok := m.Descriptors[name]
		if !ok {
			// Un fournisseur documenté qui aurait perdu son descripteur : la page doit le
			// dire, pas rendre un tableau vide qui se lirait comme « rien à collecter ».
			out["provider-"+name+"-identity"] = t.noDescriptor
			continue
		}
		out["provider-"+name+"-identity"] = providerIdentityTable(t, desc)
		out["provider-"+name+"-credentials"] = providerCredentialsTable(t, desc)
		out["provider-"+name+"-live"] = providerLiveTable(t, desc)
		out["provider-"+name+"-terraform"] = providerTerraformTable(t, desc)
		out["provider-"+name+"-coverage"] = providerCoverageTable(t, m, name)
		out["provider-"+name+"-na"] = providerNATable(t, m, name)
		out["provider-"+name+"-onesource"] = providerOneSourceTable(t, m, name)
		if c, ok := scans[name]; ok {
			out["provider-"+name+"-scan"] = Fence("text", Tail(c.Stdout, 16))
		}
	}
	return out
}

// providerIdentityTable rend l'identité et la souveraineté déclarées, avec leurs sources.
// Ces champs alimentent le contrôle de gouvernance CLD-GVN-4 : ils sont opposables, donc ils
// se citent depuis le descripteur, jamais de mémoire.
func providerIdentityTable(t providerStrings, d genprovider.Descriptor) string {
	s := d.Souverainete
	var b strings.Builder
	b.WriteString("| " + t.colField + " | " + t.colValue + " |\n|---|---|\n")
	row := func(label, value string) {
		if value == "" {
			value = t.unset
		}
		_, _ = fmt.Fprintf(&b, "| %s | %s |\n", label, value)
	}
	row(t.fieldDescription, d.Description)
	row(t.fieldScope, orDefault(d.Scope, "cloud"))
	row(t.fieldRegionKey, "`"+orDefault(d.RegionKey, "region")+"`")
	row(t.fieldAuth, "`"+d.Auth.Type+"`")
	row(t.fieldJurisdiction, s.Juridiction)
	row(t.fieldEUEstablished, boolLabel(t, s.EUEtabli))
	row(t.fieldCapital, s.ControleCapitalistique)
	row(t.fieldSecNumCloud, "`"+s.SecNumCloud+"`")
	row(t.fieldExtraterritorial, boolLabel(t, s.ExpositionExtraterritoriale))
	row(t.fieldSources, oneLine(s.Sources))
	return b.String()
}

// providerCredentialsTable rend ce que la collecte live lit dans l'environnement, plus le
// fichier de configuration natif et les valeurs par défaut.
func providerCredentialsTable(t providerStrings, d genprovider.Descriptor) string {
	keys := make([]string, 0, len(d.Credentials.Env))
	for k := range d.Credentials.Env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString("| " + t.colLogicalKey + " | " + t.colEnvVar + " | " + t.colDefault + " |\n|---|---|---|\n")
	for _, k := range keys {
		def := d.Credentials.Defaults[k]
		if def == "" {
			def = t.unset
		} else {
			def = "`" + def + "`"
		}
		_, _ = fmt.Fprintf(&b, "| `%s` | `%s` | %s |\n", k, d.Credentials.Env[k], def)
	}
	if d.Credentials.File.Path != "" {
		_, _ = fmt.Fprintf(&b, "| %s | `%s` | `%s` |\n",
			t.rowCredFile, d.Credentials.File.Path, d.Credentials.File.Format)
	}
	return b.String()
}

// providerLiveTable rend TOUS les appels d'API que la collecte live effectue, y compris les
// listes parentes d'une jointure : c'est la liste exacte des droits qu'une clé de lecture
// doit couvrir, et elle ne se devine pas depuis la page de couverture.
func providerLiveTable(t providerStrings, d genprovider.Descriptor) string {
	type call struct{ typ, method, path, note string }
	var calls []call
	base := d.Collecte.BaseURL
	for _, r := range d.Collecte.Resources {
		if r.ForEach != nil {
			calls = append(calls, call{
				typ: r.Type, method: methodOf(r.ForEach.Method), path: r.ForEach.Path, note: t.noteParent,
			})
		}
		note := ""
		if r.BaseURL != "" && r.BaseURL != base {
			note = "`" + r.BaseURL + "`"
		}
		calls = append(calls, call{typ: r.Type, method: methodOf(r.Method), path: r.Path, note: note})
	}
	sort.SliceStable(calls, func(i, j int) bool {
		if calls[i].typ != calls[j].typ {
			return calls[i].typ < calls[j].typ
		}
		return calls[i].path < calls[j].path
	})

	var b strings.Builder
	b.WriteString("| " + t.colNormalizedType + " | " + t.colCall + " | " + t.colNote + " |\n|---|---|---|\n")
	seen := map[string]bool{}
	for _, c := range calls {
		key := c.typ + "\x00" + c.method + "\x00" + c.path + "\x00" + c.note
		if seen[key] {
			continue // deux entrées ne diffèrent parfois que par le sens des règles collectées
		}
		seen[key] = true
		note := c.note
		if note == "" {
			note = t.unset
		}
		_, _ = fmt.Fprintf(&b, "| `%s` | `%s %s` | %s |\n", c.typ, c.method, c.path, note)
	}
	if d.S3.Endpoint != "" {
		_, _ = fmt.Fprintf(&b, "| `object_storage_bucket` | `%s` | %s |\n", d.S3.Endpoint, t.noteS3)
	}
	if d.OKS.Endpoint != "" {
		_, _ = fmt.Fprintf(&b, "| `kubernetes_cluster` | `%s` | %s |\n", d.OKS.Endpoint, t.noteManagedK8s)
	}
	if base != "" {
		b.WriteString("\n" + t.baseURL + " `" + base + "`\n")
	}
	return b.String()
}

// providerTerraformTable rend les ressources Terraform reconnues et le type normalisé
// qu'elles alimentent.
func providerTerraformTable(t providerStrings, d genprovider.Descriptor) string {
	type row struct{ tf, typ, items string }
	rows := make([]row, 0, len(d.MappingTerraform.Resources))
	for _, r := range d.MappingTerraform.Resources {
		rows = append(rows, row{tf: r.TFType, typ: r.Type, items: r.Items})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].tf != rows[j].tf {
			return rows[i].tf < rows[j].tf
		}
		return rows[i].items < rows[j].items
	})
	var b strings.Builder
	b.WriteString("| " + t.colTFResource + " | " + t.colNormalizedType + " | " + t.colExploded + " |\n|---|---|---|\n")
	for _, r := range rows {
		items := t.unset
		if r.items != "" {
			items = "`" + r.items + "`"
		}
		_, _ = fmt.Fprintf(&b, "| `%s` | `%s` | %s |\n", r.tf, r.typ, items)
	}
	return b.String()
}

// providerCoverageTable rend les totaux de couverture du fournisseur, par source. Le détail
// contrôle par contrôle reste dans docs/coverage.md : le dupliquer ici garantirait la dérive.
func providerCoverageTable(t providerStrings, m Matrix, provider string) string {
	var b strings.Builder
	b.WriteString("| " + t.colSource + " | " + mark(Supported) + " `supported` | " + mark(Partial) +
		" `partial` | " + mark(NotApplicable) + " `not-applicable` | " + mark(Unsupported) +
		" `unsupported` |\n|---|---:|---:|---:|---:|\n")
	for _, src := range []Source{SourceTerraform, SourceLive} {
		c := map[Status]int{}
		for _, r := range m.Rows {
			c[r.Cells[provider][src].Status]++
		}
		_, _ = fmt.Fprintf(&b, "| %s | %d | %d | %d | %d |\n",
			src, c[Supported], c[Partial], c[NotApplicable], c[Unsupported])
	}
	return b.String()
}

// providerNATable rend les contrôles que le contrat du fournisseur déclare non testables,
// avec leur justification. Un N/A sans motif n'est pas opposable.
func providerNATable(t providerStrings, m Matrix, provider string) string {
	var b strings.Builder
	b.WriteString("| " + t.colControl + " | " + t.colJustification + " |\n|---|---|\n")
	rows := 0
	for _, r := range m.Rows {
		c := r.Cells[provider][SourceLive]
		if c.Status != NotApplicable {
			continue
		}
		_, _ = fmt.Fprintf(&b, "| `%s` | %s |\n", r.Code, oneLine(c.Reason))
		rows++
	}
	if rows == 0 {
		return t.none + "\n"
	}
	return b.String()
}

// providerOneSourceTable rend les contrôles que ce fournisseur n'observe que par UNE source,
// avec le motif du côté aveugle. C'est ce qui rend comparable — ou non — un scan de plan et
// un scan live du même tenant.
func providerOneSourceTable(t providerStrings, m Matrix, provider string) string {
	var b strings.Builder
	b.WriteString("| " + t.colControl + " | " + t.colOnlyVia + " | " + t.colReason + " |\n|---|---|---|\n")
	rows := 0
	for _, r := range m.Rows {
		tf := r.Cells[provider][SourceTerraform]
		live := r.Cells[provider][SourceLive]
		switch {
		case tf.Status == Supported && live.Status != Supported:
			_, _ = fmt.Fprintf(&b, "| `%s` | terraform | %s |\n", r.Code, oneLine(live.Reason))
			rows++
		case live.Status == Supported && tf.Status != Supported:
			_, _ = fmt.Fprintf(&b, "| `%s` | live | %s |\n", r.Code, oneLine(tf.Reason))
			rows++
		}
	}
	if rows == 0 {
		return t.none + "\n"
	}
	return b.String()
}

func methodOf(m string) string {
	if m == "" {
		return "GET"
	}
	return m
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func boolLabel(t providerStrings, v *bool) string {
	if v == nil {
		return t.unset
	}
	if *v {
		return t.yes
	}
	return t.no
}

// providerStrings porte les libellés des régions des pages de fournisseurs.
type providerStrings struct {
	colField, colValue, colLogicalKey, colEnvVar, colDefault string
	colNormalizedType, colCall, colNote, colTFResource       string
	colExploded, colSource, colControl, colJustification     string
	colOnlyVia, colReason                                    string
	fieldDescription, fieldScope, fieldRegionKey, fieldAuth  string
	fieldJurisdiction, fieldEUEstablished, fieldCapital      string
	fieldSecNumCloud, fieldExtraterritorial, fieldSources    string
	rowCredFile, noteParent, noteS3, noteManagedK8s, baseURL string
	unset, yes, no, none, noDescriptor                       string
}

func providerText(lang string) providerStrings {
	if lang == "fr" {
		return providerStrings{
			colField: "Champ du descripteur", colValue: "Valeur",
			colLogicalKey: "Clé logique", colEnvVar: "Variable d'environnement", colDefault: "Défaut",
			colNormalizedType: "Type normalisé", colCall: "Appel", colNote: "Note",
			colTFResource: "Ressource Terraform", colExploded: "Bloc éclaté",
			colSource: "Source", colControl: "Contrôle", colJustification: "Justification consignée au contrat",
			colOnlyVia: "Observable uniquement via", colReason: "Motif du côté aveugle",
			fieldDescription: "Description", fieldScope: "Portée", fieldRegionKey: "Clé de région (`--region`)",
			fieldAuth: "Authentification de l'API", fieldJurisdiction: "Juridiction du siège",
			fieldEUEstablished: "Établi dans l'UE", fieldCapital: "Contrôle capitalistique",
			fieldSecNumCloud: "SecNumCloud", fieldExtraterritorial: "Exposition extraterritoriale",
			fieldSources:   "Sources de l'ancrage",
			rowCredFile:    "fichier de configuration natif",
			noteParent:     "liste parente d'une jointure (appelée en premier)",
			noteS3:         "API S3 du stockage objet (collecteur Go)",
			noteManagedK8s: "API du Kubernetes managé (collecteur Go)",
			baseURL:        "URL de base :",
			unset:          "—", yes: "oui", no: "non", none: "_Aucun._",
			noDescriptor: "_Descripteur absent du dépôt : cette page ne peut rien affirmer sur ce fournisseur._",
		}
	}
	return providerStrings{
		colField: "Descriptor field", colValue: "Value",
		colLogicalKey: "Logical key", colEnvVar: "Environment variable", colDefault: "Default",
		colNormalizedType: "Normalized type", colCall: "Call", colNote: "Note",
		colTFResource: "Terraform resource", colExploded: "Exploded block",
		colSource: "Source", colControl: "Control", colJustification: "Justification recorded in the contract",
		colOnlyVia: "Observable only through", colReason: "Reason on the blind side",
		fieldDescription: "Description", fieldScope: "Scope", fieldRegionKey: "Region key (`--region`)",
		fieldAuth: "API authentication", fieldJurisdiction: "Jurisdiction of the head office",
		fieldEUEstablished: "Established in the EU", fieldCapital: "Capital control",
		fieldSecNumCloud: "SecNumCloud", fieldExtraterritorial: "Extraterritorial exposure",
		fieldSources:   "Anchoring sources",
		rowCredFile:    "native configuration file",
		noteParent:     "parent listing of a join (called first)",
		noteS3:         "object storage S3 API (Go collector)",
		noteManagedK8s: "managed Kubernetes API (Go collector)",
		baseURL:        "Base URL:",
		unset:          "—", yes: "yes", no: "no", none: "_None._",
		noDescriptor: "_Descriptor missing from the repository: this page can assert nothing about this provider._",
	}
}

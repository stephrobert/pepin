// Package quality porte la CARTE DE QUALITÉ DE DÉTECTION : ce que Pépin peut
// prouver de ses propres verdicts, et ce qu'il ne peut pas.
//
// # La règle qui gouverne ce paquet
//
// Aucun chiffre publié ici ne peut être meilleur que ce qui est mesuré. La carte
// est DÉRIVÉE des artefacts qui font foi — les obligations calculées depuis la
// matrice de couverture, les scénarios de véracité, les tenants de référence, les
// relevés de canari — et jamais d'une saisie. Un pourcentage sans mesure derrière
// est le faux vert que ce projet combat, simplement déplacé dans un tableau de
// bord, et il y serait pire : personne ne relit un tableau de bord.
//
// Les chiffres réels sont laids. C'est le point. « 57 contrôles » ne dit rien de
// la qualité d'une détection ; « 63 verdicts prouvés sur 458 » dit exactement où
// en est le produit, et il rétrécit dans le bon sens à chaque scénario écrit.
//
// # Pourquoi la carte et `control explain` lisent la MÊME source
//
// Deux calculs de couverture divergent toujours, et celui qui diverge est celui
// qu'on lit. La carte publiée et la commande d'explication consomment donc un seul
// instantané (Snapshot), calculé une fois par le générateur de documentation et
// committé. Le binaire l'embarque, parce qu'un utilisateur qui lance `pepin
// control explain` n'a pas le dépôt sous la main : les scénarios, les tenants et
// le registre de dette vivent dans le dépôt, pas dans l'exécutable.
//
// # Ce que « validé en live » compte, et pourquoi c'est zéro
//
// Un scan canari interroge le vrai plan de contrôle d'un fournisseur SANS
// identifiant : il prouve qu'un endpoint existe et refuse, jamais qu'un droit
// suffisant rende 200 sur un tenant réel. Il ne vaut donc pas validation live d'un
// contrôle. Le compteur ne s'incrémente que sur un relevé AUTHENTIFIÉ, et il n'en
// existe aucun — d'où zéro, dérivé et non écrit. Le jour où un mainteneur consigne
// un relevé authentifié, le chiffre montera tout seul.
package quality

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/stephrobert/pepin/internal/canary"
	"github.com/stephrobert/pepin/internal/veracity"
)

// SnapshotFile est le chemin de l'instantané committé, relatif à la racine du
// dépôt. Il est GÉNÉRÉ (internal/docgen) et embarqué par le binaire.
const SnapshotFile = "internal/quality/snapshot.json"

// VerdictCount est, pour un verdict, ce qu'il faut prouver et ce qui l'est.
type VerdictCount struct {
	Required int `json:"required"`
	Proven   int `json:"proven"`
}

// PathProof est ce qu'un chemin contrôle × fournisseur × source doit prouver, et
// ce qu'il prouve aujourd'hui.
type PathProof struct {
	Provider string   `json:"provider"`
	Source   string   `json:"source"`
	Status   string   `json:"status"`
	Required []string `json:"required,omitempty"`
	Proven   []string `json:"proven,omitempty"`
}

// ControlProof rassemble, pour un contrôle, ce qui l'éprouve.
type ControlProof struct {
	Paths []PathProof `json:"paths,omitempty"`
	// Scenarios, Tenants, RegoTests : les artefacts qui portent la preuve, nommés
	// pour qu'un lecteur puisse aller les ouvrir. Un chiffre de couverture qu'on ne
	// peut pas remonter jusqu'à un fichier est un chiffre qu'on doit croire.
	Scenarios []string `json:"scenarios,omitempty"`
	Tenants   []string `json:"tenants,omitempty"`
	RegoTests []string `json:"rego_tests,omitempty"`
}

// CanaryProvider résume un relevé de canari, sans ses unités : la carte publie une
// date et un décompte, le détail se lit dans references/canary/.
type CanaryProvider struct {
	Provider      string `json:"provider"`
	Recorded      string `json:"recorded"`
	Authenticated bool   `json:"authenticated"`
	Answered      int    `json:"answered"`
	Moved         int    `json:"moved"`
	Unreachable   int    `json:"unreachable"`
}

// Snapshot est la carte entière, telle qu'elle est committée et embarquée.
type Snapshot struct {
	// Controls : contrôles au référentiel. Le chiffre que tout le monde publie, et
	// celui qui apprend le moins.
	Controls int `json:"controls"`
	// Paths : chemins contrôle × fournisseur × source sur lesquels Pépin conclut,
	// donc porteurs d'au moins une obligation.
	Paths       int `json:"paths"`
	PathsProven int `json:"paths_proven"`
	Obligations int `json:"obligations"`
	Proven      int `json:"proven"`
	// Verdicts : le détail par verdict, indexé par `fail`/`pass`/`not-evaluated`/
	// `not-applicable`. C'est la ventilation que demande l'issue #62 : une fixture
	// vulnérable, une fixture conforme, une donnée absente, un contrat de fournisseur.
	Verdicts map[string]VerdictCount `json:"verdicts"`
	// LivePaths : chemins dont la source est une collecte live.
	// LiveValidated : ceux qu'un relevé AUTHENTIFIÉ atteste. Voir l'en-tête du paquet.
	LivePaths     int `json:"live_paths"`
	LiveValidated int `json:"live_validated"`
	// Counterwitnesses : tenants tiers déclarés durcis sur lesquels Pépin ne relève
	// aucun écart critical/high. C'est le seul endroit du dépôt où un faux positif
	// se voit, et le seul chiffre de faux positifs qui soit MESURÉ.
	Counterwitnesses int                     `json:"counterwitnesses"`
	Tenants          int                     `json:"tenants"`
	Canary           []CanaryProvider        `json:"canary"`
	ByControl        map[string]ControlProof `json:"by_control"`
}

// Percent rend un pourcentage entier, et 0 quand rien n'est dû — jamais 100 sur un
// dénominateur nul : « tout est prouvé » et « il n'y avait rien à prouver » ne se
// lisent pas de la même façon.
func Percent(proven, required int) int {
	if required <= 0 {
		return 0
	}
	return proven * 100 / required
}

// Inputs rassemble ce dont le calcul a besoin. Un struct plutôt qu'une longue
// liste d'arguments : chaque champ vient d'un artefact différent, et les nommer
// est ce qui rend le calcul relisible.
type Inputs struct {
	// Cells : la matrice de couverture, réduite à ce dont les obligations ont besoin.
	Cells []veracity.Cell
	// Covered : les verdicts réellement prouvés, scénarios ET tenants fusionnés.
	Covered map[veracity.Path][]veracity.Verdict
	// Controls : le nombre de contrôles au référentiel.
	Controls int
	// Records : les relevés de canari.
	Records []canary.Record
	// ScenariosByPath : les fichiers de scénarios qui couvrent un chemin.
	ScenariosByPath map[veracity.Path][]string
	// TenantsByPath : les tenants de référence qui prouvent un verdict sur un chemin.
	TenantsByPath map[veracity.Path][]string
	// RegoTestsByControl : les fichiers *_test.rego qui citent le code du contrôle.
	RegoTestsByControl map[string][]string
	// Counterwitnesses, Tenants : tenants durcis sans écart critical/high, et total.
	Counterwitnesses int
	TenantsTotal     int
}

// Compute dérive la carte. Rien n'y est saisi : chaque champ vient d'un artefact.
func Compute(in Inputs) Snapshot {
	obligations := veracity.Obligations(in.Cells)
	counts := veracity.Count(obligations, in.Covered)

	s := Snapshot{
		Controls:         in.Controls,
		Paths:            counts.Paths,
		PathsProven:      counts.PathsProven,
		Obligations:      counts.Obligations,
		Proven:           counts.Obligations - counts.Remaining,
		Verdicts:         map[string]VerdictCount{},
		Counterwitnesses: in.Counterwitnesses,
		Tenants:          in.TenantsTotal,
		ByControl:        map[string]ControlProof{},
	}

	// Un relevé AUTHENTIFIÉ est le seul qui atteste qu'un droit suffisant a rendu
	// 200 sur un vrai tenant. Aucun n'existe : le compteur est dérivé, pas écrit.
	authenticated := map[string]bool{}
	for _, r := range in.Records {
		s.Canary = append(s.Canary, CanaryProvider{
			Provider: r.Provider, Recorded: r.Recorded, Authenticated: r.Authenticated,
			Answered: r.Summary.Answered, Moved: r.Summary.Moved, Unreachable: r.Summary.Unreachable,
		})
		if r.Authenticated {
			authenticated[r.Provider] = true
		}
	}
	sort.Slice(s.Canary, func(i, j int) bool { return s.Canary[i].Provider < s.Canary[j].Provider })

	// Le statut de couverture de chaque chemin, pour que `control explain` puisse
	// dire POURQUOI un chemin ne doit prouver qu'un `not-evaluated`.
	status := make(map[veracity.Path]string, len(in.Cells))
	for _, c := range in.Cells {
		status[veracity.Path{Control: c.Control, Provider: c.Provider, Source: c.Source}] = c.Status
	}

	byControl := map[string][]PathProof{}
	for path, want := range obligations {
		have := map[veracity.Verdict]bool{}
		for _, v := range in.Covered[path] {
			have[v] = true
		}
		p := PathProof{Provider: path.Provider, Source: path.Source, Status: status[path]}
		for _, v := range want {
			vc := s.Verdicts[string(v)]
			vc.Required++
			p.Required = append(p.Required, string(v))
			if have[v] {
				vc.Proven++
				p.Proven = append(p.Proven, string(v))
			}
			s.Verdicts[string(v)] = vc
		}
		if path.Source == "live" {
			s.LivePaths++
			if authenticated[path.Provider] {
				s.LiveValidated++
			}
		}
		byControl[path.Control] = append(byControl[path.Control], p)
	}

	for code, paths := range byControl {
		sort.Slice(paths, func(i, j int) bool {
			if paths[i].Provider != paths[j].Provider {
				return paths[i].Provider < paths[j].Provider
			}
			return paths[i].Source < paths[j].Source
		})
		proof := ControlProof{Paths: paths, RegoTests: in.RegoTestsByControl[code]}
		seenScenario, seenTenant := map[string]bool{}, map[string]bool{}
		for _, p := range paths {
			key := veracity.Path{Control: code, Provider: p.Provider, Source: p.Source}
			for _, f := range in.ScenariosByPath[key] {
				if !seenScenario[f] {
					seenScenario[f] = true
					proof.Scenarios = append(proof.Scenarios, f)
				}
			}
			for _, tn := range in.TenantsByPath[key] {
				if !seenTenant[tn] {
					seenTenant[tn] = true
					proof.Tenants = append(proof.Tenants, tn)
				}
			}
		}
		sort.Strings(proof.Scenarios)
		sort.Strings(proof.Tenants)
		s.ByControl[code] = proof
	}
	return s
}

// Encode rend l'instantané sous sa forme committée : JSON indenté et trié, pour
// qu'un diff se lise. Un instantané qu'on ne peut pas relire en revue est un
// instantané que personne ne conteste.
func Encode(s Snapshot) ([]byte, error) {
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("sérialisation de la carte de qualité : %w", err)
	}
	return append(raw, '\n'), nil
}

// Decode lit un instantané.
func Decode(raw []byte) (Snapshot, error) {
	var s Snapshot
	if err := json.Unmarshal(raw, &s); err != nil {
		return Snapshot{}, fmt.Errorf("carte de qualité illisible : %w", err)
	}
	return s, nil
}

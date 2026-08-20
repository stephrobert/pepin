package exempt

import (
	"fmt"
	"sort"
	"time"

	"github.com/stephrobert/scankit/assessment"

	"github.com/stephrobert/pepin/internal/i18n"
)

// StatusExempted est le CINQUIÈME statut d'un résultat, à côté de pass, fail,
// not-applicable et not-evaluated.
//
// Il est de premier rang, et distinct de `pass` partout : dans le rendu terminal,
// dans les formats analysables, dans le décompte. Un contrôle exempté n'est PAS
// conforme — il est écarté, sciemment et de façon traçable. S'il pouvait compter
// comme conforme quelque part, on aurait donné un nom respectable au faux vert.
//
// `assessment.Status` est un type chaîne : le statut s'ajoute sans modifier scankit,
// et l'OSCAL le rend « not-satisfied » comme tout ce qui n'est pas `pass`, ce qui
// est exactement la lecture voulue.
const StatusExempted assessment.Status = "exempted"

// Effect nomme ce qu'une dérogation a réellement produit sur un scan.
type Effect string

const (
	// EffectApplied : la dérogation a écarté au moins un écart.
	EffectApplied Effect = "applied"
	// EffectExpired : sa date est passée ; elle ne s'applique plus, et le dit.
	EffectExpired Effect = "expired"
	// EffectOrphan : elle vise un contrôle ou une ressource qui n'existe pas.
	// C'est le symptôme d'une exception oubliée — on la signale, jamais on
	// l'ignore.
	EffectOrphan Effect = "orphan"
)

// Record est ce qu'une dérogation a produit, scellé dans le bundle. Un dossier qui
// tait ses exemptions n'est pas opposable.
type Record struct {
	Exemption
	// Effect : applied | expired | orphan.
	Effect Effect `json:"effect"`
	// Subjects : les sujets réellement écartés (vide hors `applied`).
	Subjects []string `json:"subjects,omitempty"`
	// Reason : pourquoi elle est expirée ou orpheline, en clair.
	Reason string `json:"reason,omitempty"`
}

// Report est l'effet complet d'une politique de dérogations sur un scan.
type Report struct {
	// PolicyDigest : empreinte de la politique appliquée.
	PolicyDigest string `json:"policy_digest,omitempty"`
	// Exemptions : la politique telle qu'elle a été chargée (rejouable).
	Exemptions []Exemption `json:"exceptions,omitempty"`
	// Records : ce que chacune a produit. Les dérogations DORMANTES (valides mais
	// dont le contrôle ne défaille pas ce jour-là) n'y figurent pas : ce n'est pas
	// une anomalie, c'est le cas normal d'un écart corrigé.
	Records []Record `json:"records,omitempty"`
}

// Count compte les enregistrements d'un effet donné.
func (r Report) Count(e Effect) int {
	n := 0
	for _, rec := range r.Records {
		if rec.Effect == e {
			n++
		}
	}
	return n
}

// Applied indique qu'au moins une dérogation a réellement écarté un écart. C'est
// le fait qui déplace le code de sortie — et rien d'autre ne le déplace.
func (r Report) Applied() bool { return r.Count(EffectApplied) > 0 }

// Stale indique qu'au moins une dérogation est périmée ou orpheline : le fichier
// demande une revue. Sous `--strict`, ce fait suffit à refuser la porte.
func (r Report) Stale() bool { return r.Count(EffectExpired)+r.Count(EffectOrphan) > 0 }

// Notices rend les avertissements destinés à l'opérateur, dans la langue courante.
// Une dérogation expirée ou orpheline se DIT ; elle ne disparaît pas en silence.
func (r Report) Notices() []string {
	var out []string
	for _, rec := range r.Records {
		switch rec.Effect {
		case EffectExpired:
			out = append(out, fmt.Sprintf(i18n.T(
				"dérogation EXPIRÉE le %s sur %s%s : elle ne s'applique plus, l'écart redevient un écart (%s)",
				"exemption EXPIRED on %s for %s%s: it no longer applies, the deviation is a deviation again (%s)"),
				rec.ExpiresAt, rec.Control, subjectSuffix(rec.Resource), rec.Owner))
		case EffectOrphan:
			out = append(out, fmt.Sprintf(i18n.T(
				"dérogation ORPHELINE sur %s%s : %s. Exception probablement oubliée, à retirer du fichier",
				"ORPHAN exemption for %s%s: %s — likely a forgotten exception, remove it from the file"),
				rec.Control, subjectSuffix(rec.Resource), rec.Reason))
		case EffectApplied:
		}
	}
	return out
}

func subjectSuffix(resource string) string {
	if resource == "" {
		return ""
	}
	return " / " + resource
}

// Apply écarte les écarts couverts par une dérogation valide et rend l'effet complet.
//
// Trois invariants, et ils tiennent par construction :
//
//  1. SEUL un résultat `fail` peut devenir `exempted`. Un `not-evaluated` ne peut
//     pas être exempté : on ne déroge pas à un contrôle qu'on n'a pas su mesurer,
//     ce serait taire une absence de mesure derrière une exception.
//  2. Aucun résultat ne devient `pass`. La fonction n'écrit jamais cette valeur.
//  3. Une dérogation expirée ou orpheline ne change AUCUN statut.
//
// `now` est l'instant d'évaluation du scan (pas l'horloge), pour qu'un bundle
// rejoué rende exactement le même verdict. `subjects` porte les sujets nommables de
// l'inventaire ; les sujets des résultats s'y ajoutent (voir plus bas).
func Apply(a assessment.Assessment, pol Policy, now time.Time, subjects, controls map[string]bool) (assessment.Assessment, Report) {
	rep := Report{PolicyDigest: pol.Digest(), Exemptions: pol.Exemptions}
	if len(pol.Exemptions) == 0 {
		return a, Report{}
	}

	// Les sujets connus : ceux de l'inventaire ET ceux que les résultats portent
	// réellement. La distinction compte. Une règle choisit son sujet, et ce n'est pas
	// toujours l'identifiant de la ressource : la règle SSH ouvert désigne le GROUPE de
	// sécurité, pas la règle qui le compose. Or `resource:` est écrit par un humain qui
	// recopie ce que le rapport lui montre — c'est donc le sujet du rapport qui fait foi,
	// sans quoi une dérogation parfaitement légitime serait déclarée orpheline.
	known := make(map[string]bool, len(subjects)+len(a.Results))
	for s := range subjects {
		known[s] = true
	}
	for _, r := range a.Results {
		if r.Subject != "" {
			known[r.Subject] = true
		}
	}

	// Tri des dérogations en trois tas, AVANT de toucher au moindre résultat.
	var candidates []*candidate
	for _, e := range pol.Exemptions {
		switch {
		case !controls[e.Control]:
			rep.Records = append(rep.Records, Record{Exemption: e, Effect: EffectOrphan,
				Reason: fmt.Sprintf(i18n.T(
					"aucun contrôle « %s » au référentiel", "no control %q in the reference"), e.Control)})
		case e.Resource != "" && !known[e.Resource]:
			rep.Records = append(rep.Records, Record{Exemption: e, Effect: EffectOrphan,
				Reason: fmt.Sprintf(i18n.T(
					"aucune ressource « %s » dans l'inventaire évalué", "no resource %q in the assessed inventory"), e.Resource)})
		case expired(e, now):
			rep.Records = append(rep.Records, Record{Exemption: e, Effect: EffectExpired,
				Reason: fmt.Sprintf(i18n.T("échue depuis le %s", "lapsed since %s"), e.ExpiresAt)})
		default:
			candidates = append(candidates, &candidate{ex: e, subjects: map[string]bool{}})
		}
	}

	res := append([]assessment.Result(nil), a.Results...)
	for i := range res {
		if res[i].Status != assessment.Fail {
			continue // invariant 1 : rien d'autre qu'un `fail` ne s'exempte
		}
		c := match(candidates, res[i].Control, res[i].Subject)
		if c == nil {
			continue
		}
		c.subjects[res[i].Subject] = true
		res[i].Status = StatusExempted // invariant 2 : jamais `pass`
		res[i].Waiver = &assessment.Waiver{Justification: c.ex.Justification, Until: string(c.ex.ExpiresAt)}
		res[i].Labels = withExemptionLabels(res[i].Labels, c.ex)
	}
	a.Results = res

	for _, c := range candidates {
		if len(c.subjects) == 0 {
			continue // dormante : le contrôle ne défaille pas, rien à signaler
		}
		subs := make([]string, 0, len(c.subjects))
		for s := range c.subjects {
			subs = append(subs, s)
		}
		sort.Strings(subs)
		rep.Records = append(rep.Records, Record{Exemption: c.ex, Effect: EffectApplied, Subjects: subs})
	}
	sort.SliceStable(rep.Records, func(i, j int) bool {
		if rep.Records[i].Effect != rep.Records[j].Effect {
			return rep.Records[i].Effect < rep.Records[j].Effect
		}
		if rep.Records[i].Control != rep.Records[j].Control {
			return rep.Records[i].Control < rep.Records[j].Control
		}
		return rep.Records[i].Resource < rep.Records[j].Resource
	})
	return a, rep
}

// candidate est une dérogation VALIDE (contrôle et ressource connus, non échue),
// et les sujets qu'elle a effectivement écartés.
type candidate struct {
	ex       Exemption
	subjects map[string]bool
}

// match trouve la dérogation qui couvre un (contrôle, sujet). Une entrée sans
// `resource` couvre tous les sujets du contrôle ; une entrée nommée ne couvre que
// le sien. La PREMIÈRE qui couvre l'emporte : l'ordre du fichier décide, ce qui est
// la seule règle qu'un relecteur peut vérifier de tête.
func match(candidates []*candidate, control, subject string) *candidate {
	for _, c := range candidates {
		if c.ex.Control != control {
			continue
		}
		if c.ex.Resource == "" || c.ex.Resource == subject {
			return c
		}
	}
	return nil
}

// expired dit si la dérogation a dépassé sa date à l'instant d'évaluation. Une date
// illisible ne peut pas arriver ici (Load la refuse) ; par prudence, elle expire.
func expired(e Exemption, now time.Time) bool {
	t, err := e.ExpiresAt.Time()
	if err != nil {
		return true
	}
	return now.After(t)
}

// withExemptionLabels ajoute les labels de traçabilité de la dérogation, sans
// muter la carte du résultat d'origine.
func withExemptionLabels(labels map[string]string, e Exemption) map[string]string {
	out := make(map[string]string, len(labels)+3)
	for k, v := range labels {
		out[k] = v
	}
	out["exemption_owner"] = e.Owner
	out["exemption_approved_by"] = e.ApprovedBy
	out["exemption_expires_at"] = string(e.ExpiresAt)
	return out
}

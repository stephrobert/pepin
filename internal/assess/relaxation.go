package assess

import (
	"fmt"
	"sort"
	"strings"

	"github.com/stephrobert/scankit/assessment"

	"github.com/stephrobert/pepin/internal/i18n"
	"github.com/stephrobert/pepin/internal/policy"
)

// Labels par lesquels un résultat porte l'assouplissement sous lequel il a été
// rendu. `Result.Labels` est la carte « spécificités produit » du modèle amont :
// la mention voyage donc jusqu'à `--format assessment`, jusqu'à l'OSCAL et
// jusqu'au bundle scellé, sans qu'une ligne de scankit ne bouge.
const (
	// LabelConfigRelaxed vaut "true" sur tout résultat d'un contrôle assoupli.
	LabelConfigRelaxed = "config_relaxed"
	// LabelConfigRelaxedDetail porte les réglages en cause, en clair.
	LabelConfigRelaxedDetail = "config_relaxed_detail"
	// LabelReferencesDropped porte les correspondances normatives abandonnées.
	LabelReferencesDropped = "references_dropped"
)

// ReferencesOf rend, par contrôle, ses correspondances normatives sous leur forme
// `framework:id` — la forme qu'un rapport affiche quand il annonce ce qu'un
// assouplissement fait perdre.
func ReferencesOf(refs []assessment.Reference) []string {
	out := make([]string, 0, len(refs))
	for _, r := range refs {
		out = append(out, r.Framework+":"+r.ID)
	}
	return out
}

// WithRelaxations retire à chaque contrôle ASSOUPLI ses correspondances
// normatives, et dit pourquoi.
//
// C'est la clé de voûte du modèle de configuration. Un réglage desserré qui
// continuerait d'afficher « CIS 11.2 » transformerait le rapport en affirmation
// fausse : le contrôle a bien été évalué, mais contre une barre plus basse que
// celle de l'exigence citée. On ne touche donc PAS au statut — la mesure a eu
// lieu, elle reste publiée telle quelle — et on retire la seule chose qui n'est
// plus vraie : la correspondance.
//
// Passe POSTÉRIEURE, comme la provenance et les dérogations : elle n'invente
// aucun résultat et n'en fait disparaître aucun.
func WithRelaxations(a assessment.Assessment, relax map[string][]policy.Relaxation) assessment.Assessment {
	if len(relax) == 0 {
		return a
	}
	res := append([]assessment.Result(nil), a.Results...)
	for i := range res {
		rs := relax[res[i].Control]
		if len(rs) == 0 {
			continue
		}
		dropped := ReferencesOf(res[i].References)
		res[i].References = nil
		details := make([]string, 0, len(rs))
		for _, r := range rs {
			details = append(details, r.Sentence())
		}
		labels := map[string]string{}
		for k, v := range res[i].Labels {
			labels[k] = v
		}
		labels[LabelConfigRelaxed] = "true"
		labels[LabelConfigRelaxedDetail] = strings.Join(details, " · ")
		if len(dropped) > 0 {
			sort.Strings(dropped)
			labels[LabelReferencesDropped] = strings.Join(dropped, ", ")
		}
		res[i].Labels = labels
		// La preuve dit ce qu'elle vaut. Un auditeur qui ne lit que l'OSCAL doit
		// trouver l'assouplissement DANS l'observation, pas seulement dans un label.
		res[i].Evidence.Observed = strings.TrimSpace(res[i].Evidence.Observed + " " + fmt.Sprintf(i18n.T(
			"— CONFIGURATION ASSOUPLIE : %s. La correspondance normative de ce contrôle n'est plus tenue.",
			"— RELAXED CONFIGURATION: %s. This control's normative mapping no longer holds."),
			strings.Join(details, " · ")))
	}
	a.Results = res
	return a
}

package veracity

import "sort"

// Le CONTRE-EXEMPLE légitime (issue #115), et ce qu'il permet de publier (#118).
//
// Le contrat de véracité prouve qu'un chemin sait produire le verdict attendu. Il ne
// prouve pas qu'il RETIENT ce verdict sur une configuration voisine et légitime, et
// c'est de cette moitié-là qu'est faite la précision : une règle qui se déclenche sur
// tout est parfaitement sensible.
//
// Ce calcul vit ici, et non dans un fichier de test, pour une raison qui a déjà servi
// dans ce paquet : la carte de qualité et la porte qui refuse un contrôle sans
// contre-exemple doivent lire la MÊME source. Deux calculs de couverture divergent
// toujours, et celui qui diverge est celui qu'on publie.

// CounterexamplePairs rend les contrôles pour lesquels un MÊME chemin contrôle ×
// fournisseur × source porte à la fois un cas `fail` et un cas `pass`.
//
// L'unicité du chemin n'est pas un détail : un `fail` sur un plan Terraform et un
// `pass` sur une collecte live n'éprouvent pas la même chaîne, et la règle pourrait
// se taire dans la seconde pour une raison entièrement étrangère à la configuration
// légitime qu'on croit avoir prouvée.
//
// Ce que ce calcul NE mesure PAS, et qui ne se mesure pas : la PROXIMITÉ du cas
// légitime au cas fautif. Un `pass` sur un inventaire vide satisferait cette fonction
// sans rien prouver. C'est la revue qui juge cette proximité, et les scénarios la
// portent dans leur champ `why`.
func CounterexamplePairs(files []File) map[string]bool {
	surLeChemin := map[Path]map[Verdict]bool{}
	for _, f := range files {
		p := f.PathOf()
		if surLeChemin[p] == nil {
			surLeChemin[p] = map[Verdict]bool{}
		}
		for _, s := range f.Scenarios {
			surLeChemin[p][s.Expect] = true
		}
	}
	out := map[string]bool{}
	for p, v := range surLeChemin {
		if v[Fail] && v[Pass] {
			out[p.Control] = true
		}
	}
	return out
}

// DetectionProven rend les contrôles dont AU MOINS un chemin prouve le verdict
// `fail` de bout en bout — c'est-à-dire que la chaîne complète, de la source au
// verdict, a été observée sur une configuration réellement fautive.
//
// C'est la SENSIBILITÉ mesurée, à distinguer de la précision : les deux se publient
// côte à côte, parce qu'un chiffre seul se lit toujours dans le sens le plus
// flatteur.
func DetectionProven(covered map[Path][]Verdict) map[string]bool {
	out := map[string]bool{}
	for p, verdicts := range covered {
		for _, v := range verdicts {
			if v == Fail {
				out[p.Control] = true
				break
			}
		}
	}
	return out
}

// WithoutCounterexample rend, triés, les contrôles de `codes` qui n'ont pas de
// couple. C'est la dette que le registre consigne, et elle se lit dans les deux
// sens : une ligne de trop est une dette inventée, une ligne manquante un contrôle
// ajouté sans sa preuve de précision.
func WithoutCounterexample(codes []string, pairs map[string]bool) []string {
	var out []string
	for _, c := range codes {
		if !pairs[c] {
			out = append(out, c)
		}
	}
	sort.Strings(out)
	return out
}

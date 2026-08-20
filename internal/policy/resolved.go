package policy

import "strings"

// Resolved est la configuration EFFECTIVE d'un scan : le profil par défaut, sur
// lequel le fichier de politique a été appliqué. C'est elle que le scan injecte
// dans l'input (clé `config`), donc elle qui est scellée dans l'input.json du
// bundle et rejouée à l'identique par `verify --re-derive`.
//
// Sa forme est le CONTRAT que lisent les règles Rego : noms d'étiquettes déjà
// normalisés et alias déjà résolus, pour que la règle n'ait qu'un test
// d'appartenance à faire. La normalisation vit ici, en Go, en un seul endroit ;
// la dupliquer dans chaque règle serait la faire diverger.
type Resolved struct {
	Tagging   ResolvedTagging   `json:"tagging"`
	Snapshots ResolvedSnapshots `json:"snapshots"`
	Secrets   ResolvedSecrets   `json:"secrets"`
}

// ResolvedTagging est la politique d'étiquetage effective.
type ResolvedTagging struct {
	// ResourceTypes : les types normalisés sur lesquels l'étiquetage est exigé.
	ResourceTypes []string `json:"resource_types"`
	// Required : les étiquettes exigées sur ces ressources.
	Required []RequiredTag `json:"required"`
	// NetworkRequired : les étiquettes exigées sur les réseaux (cartographie).
	NetworkRequired []RequiredTag `json:"network_required"`
}

// RequiredTag est une étiquette exigée : son nom LOGIQUE (celui que lira un
// humain dans le message du finding) et les écritures acceptées, normalisées.
type RequiredTag struct {
	Name string   `json:"name"`
	Keys []string `json:"keys"`
}

// ResolvedSnapshots est la politique de fraîcheur effective.
type ResolvedSnapshots struct {
	MaxAgeDays     int      `json:"max_age_days"`
	AcceptedStates []string `json:"accepted_states"`
}

// ResolvedSecrets est la politique de détection effective.
type ResolvedSecrets struct {
	MinConfidence string `json:"min_confidence"`
}

// ConfidenceLevels énumère les niveaux de confiance d'une détection de secret,
// du plus faible au plus fort. L'ORDRE est signifiant : c'est lui qui décide
// qu'un seuil plus haut est un assouplissement.
var ConfidenceLevels = []string{"low", "medium", "high"}

// rankOfConfidence rend le rang d'un niveau de confiance, -1 s'il est inconnu.
func rankOfConfidence(level string) int {
	for i, l := range ConfidenceLevels {
		if strings.EqualFold(strings.TrimSpace(level), l) {
			return i
		}
	}
	return -1
}

// Le PROFIL PAR DÉFAUT. C'est une RECOMMANDATION, pas une norme : aucune
// convention d'étiquetage ne fait autorité, et Pépin ne prétend pas le contraire.
// Il est cependant ce à quoi les correspondances normatives du référentiel sont
// adossées (`config_requise`) : s'en écarter dans le sens du relâchement fait
// perdre la correspondance, et le rapport le dit.
var (
	// defaultBillableTags : les quatre étiquettes de gouvernance exigées sur une
	// ressource facturable. Elles répondent à quatre questions qu'un inventaire
	// doit savoir trancher : qui paye, pour quoi, à quel stade, et qui répond.
	//
	// Ce sont des noms d'AFFICHAGE, pas des littéraux à matcher : ce sont eux
	// qu'un message de finding nomme, et ils gardent l'écriture historique pour
	// qu'un rapport ne change pas de vocabulaire sous les yeux de ses lecteurs.
	// La COMPARAISON, elle, est insensible à la casse et aux séparateurs et passe
	// par les alias ci-dessous : `cost-center`, `costcenter` et `CostCenter` sont
	// la même exigence. L'ORDRE est celui du message, donc il est signifiant et
	// se conserve (voir resolveTags, qui ne trie pas).
	defaultBillableTags = []string{"CostCenter", "Project", "Env", "Owner"}

	// defaultNetworkTags : la cartographie réseau (CLD-NET-5) demande de savoir à
	// QUI et à QUOI sert un réseau, et dans quel environnement. Le centre de coût
	// n'en fait pas partie : un réseau n'est pas une ligne de facture.
	defaultNetworkTags = []string{"Owner", "Project", "Env"}

	// defaultTagAliases : les écritures acceptées pour chaque nom logique. La
	// comparaison étant déjà insensible à la casse et aux séparateurs, ces alias
	// ne couvrent que les SYNONYMES — `team` pour `Owner`, `environment` pour
	// `Env` —, pas les variantes typographiques.
	//
	// C'est ici que se joue le faux positif de l'issue #61 : une organisation qui
	// écrit `cost-center, application, environment, team` est gouvernée, et ne
	// doit pas récolter un FAIL sur sa convention d'écriture.
	defaultTagAliases = map[string][]string{
		"CostCenter": {"CostCenter", "cost-center", "cc", "billing-code", "billing"},
		"Project":    {"Project", "app", "application", "service"},
		"Env":        {"Env", "environment", "stage"},
		"Owner":      {"Owner", "team", "responsible", "contact"},
	}

	// defaultTaggedTypes : les types de ressources sur lesquels l'étiquetage est
	// exigé, et POURQUOI chacun — le critère est « facturable et étiquetable » :
	// une ressource qui coûte sans propriétaire connu est un coût orphelin autant
	// qu'un risque orphelin.
	//
	//   compute_instance, blockstorage_volume, blockstorage_snapshot,
	//   compute_image, load_balancer, object_storage_bucket, managed_database,
	//   kubernetes_cluster
	//
	// Sont EXCLUS, et pour des raisons écrites : `network`, `subnet`,
	// `network_peering`, `security_group*` (non facturés en propre ; le réseau a
	// sa propre exigence de cartographie, CLD-NET-5), `iam_*`, `access_key`,
	// `api_access_*` (ni facturés ni porteurs d'étiquettes), `governance_provider`
	// (ressource synthétique, pas une ressource du tenant) et les `k8s_*`
	// (portée INTRA-cluster, hors périmètre de la posture cloud).
	defaultTaggedTypes = []string{
		"compute_instance",
		"blockstorage_volume",
		"blockstorage_snapshot",
		"compute_image",
		"load_balancer",
		"object_storage_bucket",
		"managed_database",
		"kubernetes_cluster",
	}

	// defaultMaxAgeDays : sept jours. Une semaine est la période de revue la plus
	// courante, et c'est la valeur que le contrôle appliquait en dur avant d'être
	// réglable — la rendre configurable ne devait déplacer aucun verdict.
	defaultMaxAgeDays = 7

	// defaultSnapshotStates : les états NATIFS d'une snapshot réellement
	// exploitable, ANCRÉS sur le contrat de chaque API (jamais devinés) :
	//   - Outscale, Snapshot.State ∈ in-queue | pending | completed | error |
	//     deleting (osc-api/outscale.yaml, schéma Snapshot) → seul `completed`
	//     désigne une snapshot terminée ;
	//   - Exoscale, block-storage-snapshot.state ∈ partially-destroyed |
	//     destroying | creating | created | promoting | error | destroyed |
	//     allocated (openapi-v2.exoscale.com, schéma block-storage-snapshot) →
	//     `created` désigne la snapshot terminée.
	// Les autres états sont en cours, en erreur ou en suppression : une snapshot
	// qui les porte ne restaure rien.
	defaultSnapshotStates = []string{"completed", "created"}

	// defaultMinConfidence : `low`, c'est-à-dire TOUT signaler. C'est le
	// comportement d'avant ce lot, et c'est le seul défaut défendable pour un
	// détecteur de secrets : taire par défaut ce qu'on ne sait pas confirmer,
	// c'est choisir le faux négatif contre le faux positif, sur le seul sujet où
	// le faux négatif se paye en fuite.
	defaultMinConfidence = ConfidenceLevels[0]
)

// Defaults rend le profil par défaut résolu.
func Defaults() Resolved {
	return Resolved{
		Tagging: ResolvedTagging{
			ResourceTypes:   sortedUnique(defaultTaggedTypes),
			Required:        resolveTags(defaultBillableTags, defaultTagAliases),
			NetworkRequired: resolveTags(defaultNetworkTags, defaultTagAliases),
		},
		Snapshots: ResolvedSnapshots{
			MaxAgeDays:     defaultMaxAgeDays,
			AcceptedStates: sortedUnique(defaultSnapshotStates),
		},
		Secrets: ResolvedSecrets{MinConfidence: defaultMinConfidence},
	}
}

// Resolve applique la section `controls:` au profil par défaut. Une section
// absente laisse le défaut intact — c'est ce qui garantit qu'un scan sans
// fichier de politique et un scan dont le fichier ne parle pas d'un contrôle
// rendent le même verdict.
func Resolve(c *Controls) Resolved {
	out := Defaults()
	if c == nil {
		return out
	}
	if t := c.Tagging; t != nil {
		aliases := defaultTagAliases
		if t.Aliases != nil {
			aliases = t.Aliases
		}
		names, netNames := defaultBillableTags, defaultNetworkTags
		if t.RequiredTags != nil {
			names = t.RequiredTags
		}
		if t.NetworkRequiredTags != nil {
			netNames = t.NetworkRequiredTags
		}
		out.Tagging.Required = resolveTags(names, aliases)
		out.Tagging.NetworkRequired = resolveTags(netNames, aliases)
		if t.ResourceTypes != nil {
			out.Tagging.ResourceTypes = sortedUnique(t.ResourceTypes)
		}
	}
	if s := c.Snapshots; s != nil {
		if s.MaxAgeDays != nil {
			out.Snapshots.MaxAgeDays = *s.MaxAgeDays
		}
		if s.AcceptedStates != nil {
			out.Snapshots.AcceptedStates = sortedUnique(s.AcceptedStates)
		}
	}
	if s := c.Secrets; s != nil && s.MinConfidence != "" {
		out.Secrets.MinConfidence = strings.ToLower(strings.TrimSpace(s.MinConfidence))
	}
	return out
}

// resolveTags projette des noms logiques et leurs alias en étiquettes exigées :
// le nom tel qu'écrit (pour le message lu par un humain) et les écritures
// acceptées, NORMALISÉES et triées. Le nom logique fait toujours partie de ses
// propres écritures acceptées : déclarer `owner` sans alias exige `owner`.
func resolveTags(names []string, aliases map[string][]string) []RequiredTag {
	// Les alias sont indexés par nom NORMALISÉ : une politique qui exige `owner`
	// doit hériter des alias déclarés sous `Owner`, et réciproquement. Sans cette
	// indexation, la casse du nom logique déciderait silencieusement de la
	// tolérance appliquée.
	byName := make(map[string][]string, len(aliases))
	for name, vals := range aliases {
		byName[normalizeTagKey(name)] = vals
	}
	out := make([]RequiredTag, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		keys := []string{normalizeTagKey(name)}
		for _, a := range byName[normalizeTagKey(name)] {
			if k := normalizeTagKey(a); k != "" {
				keys = append(keys, k)
			}
		}
		out = append(out, RequiredTag{Name: name, Keys: sortedUnique(keys)})
	}
	// PAS de tri : l'ordre est celui que l'auteur de la politique a écrit, et
	// c'est l'ordre dans lequel le message du finding énumère ce qui manque.
	return out
}

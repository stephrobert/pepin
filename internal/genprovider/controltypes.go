package genprovider

// extraControlTypes — les types SECONDAIRES qu'un contrôle lit, en plus de celui
// que `ControlType` déduit de son code.
//
// Pourquoi cette table existe. `ControlType` rend UN type, déduit du préfixe de
// service : c'est ce qui décide de l'applicabilité, et c'est suffisant pour la
// majorité des contrôles. Mais six règles CORRÈLENT plusieurs types, et une
// collecte incomplète sur le second passait alors inaperçue :
//
//	Collecte des VMs             -> OK
//	Collecte des règles SG       -> 403
//
// Le contrôle rattaché à `compute_instance` concluait sans que la donnée qui
// établit l'exposition soit jamais arrivée. C'est la dette que l'ADR-0006 nommait.
//
// Pourquoi elle est DÉCLARÉE et pas dérivée à l'exécution. Lire le Rego au moment
// du scan ferait dépendre un verdict d'une analyse textuelle, ce qui est fragile.
// La table est donc écrite, et `TestControlTypesMatchTheRules` la confronte à ce
// que les règles lisent RÉELLEMENT : une divergence casse la CI, dans les deux
// sens. Deux tables qu'on maintient à la main divergent ; une table gardée par sa
// dérivation, non.
var extraControlTypes = map[string][]string{
	"blockstorage_volume_snapshots_exist":                {"blockstorage_snapshot"},
	"compute_instance_public_ip_with_open_securitygroup": {"network_interface", "security_group_rule"},
	"iam_apiaccesspolicy_max_key_expiration":             {"api_access_rule", "api_access_summary"},
	"iam_apiaccessrule_defined":                          {"api_access_policy", "api_access_rule", "api_access_summary"},
	"iam_apiaccessrule_no_public_cidr":                   {"api_access_policy", "api_access_summary"},
	"iam_policy_no_privilege_escalation":                 {"iam_role"},

	// Contrôle transverse : `ControlType` rend "" pour la gouvernance, mais celui-ci
	// lit bien un type — la ressource synthétique que le descripteur publie. Sans
	// cette entrée, une collecte incomplète de cette ressource ne le dégradait pas.
	"governance_provider_sovereignty": {"governance_provider"},

	// Les types LOCALISÉS, ceux sur lesquels la souveraineté se mesure. La règle les
	// énumère dans son ensemble `_located_types` et n'en regarde aucun autre : une
	// identité ou une règle de filtrage n'héberge rien.
	//
	// Sans cette déclaration, le contrôle n'avait AUCUN type — `ControlType` rend ""
	// pour la gouvernance — et la région d'une ressource quelconque suffisait à le
	// rendre « conforme ». Mesuré avant correction : un `iam_user` en fr-par certifiait
	// une VM dont la région n'avait jamais été collectée.
	"governance_resource_region_in_eu": {
		"compute_instance",
		"object_storage_bucket",
		"blockstorage_volume",
		"blockstorage_snapshot",
		"kubernetes_cluster",
		"load_balancer",
		"managed_database",
	},
}

// ControlTypes rend TOUS les types normalisés qu'un contrôle lit : celui qui décide
// de son applicabilité, puis ceux qu'il corrèle.
//
// L'incomplétude de N'IMPORTE LEQUEL doit dégrader le contrôle : conclure sur une
// corrélation dont un côté n'a pas été lu, c'est conclure sur ce qu'on n'a pas vu.
func ControlTypes(code string) []string {
	primary := ControlType(code)
	extra := extraControlTypes[code]
	if primary == "" {
		return append([]string(nil), extra...)
	}
	out := make([]string, 0, 1+len(extra))
	out = append(out, primary)
	for _, t := range extra {
		if t == primary {
			continue
		}
		out = append(out, t)
	}
	return out
}

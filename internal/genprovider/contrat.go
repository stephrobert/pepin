package genprovider

import (
	"strings"

	"github.com/stephrobert/pepin/internal/i18n"
)

// TypeEtat retourne l'état du contrat d'un type pour un provider (verifie |
// a_verifier | absent ; vide si non catalogué). Lu depuis le descripteur chargé.
func TypeEtat(providerName, typ string) string {
	return registry[providerName].Contrat.Types[typ].Etat
}

// ControlNonApplicable indique qu'un contrôle (code agnostique) n'est pas testable
// pour un provider : soit le check est listé dans `contrat.non_applicable`, soit
// le type qu'il vise est `absent` (mécanisme inexistant côté API, confirmé doc).
func ControlNonApplicable(providerName, code string) bool {
	return NonApplicableReason(providerName, code) != ""
}

// NonApplicableReason retourne la justification opposable du caractère non applicable
// d'un contrôle pour un provider, ou "" si le contrôle EST applicable. Deux sources : une
// entrée explicite `contrat.non_applicable` (avec sa raison), ou un type visé marqué
// `absent` (mécanisme inexistant). Sans justification, un N/A n'est pas opposable.
//
// La justification est BILINGUE : elle est rendue à l'auditeur dans `--format
// assessment`, dans l'OSCAL et dans le bundle scellé. i18n.Pick dégrade vers le
// français quand `reason_en` manque : l'absence est refusée en CI
// (TestEveryContractJustificationIsBilingual), jamais découverte par le lecteur.
func NonApplicableReason(providerName, code string) string {
	return NonApplicableReasonIn(i18n.Current(), providerName, code)
}

// NonApplicableReasonIn est NonApplicableReason pour une langue explicite : la
// documentation bilingue rend les deux versions dans une seule exécution.
func NonApplicableReasonIn(l i18n.Lang, providerName, code string) string {
	d := registry[providerName]
	for _, e := range d.Contrat.NonApplicable {
		if e.Control == code {
			if e.Reason != "" {
				return i18n.PickIn(l, e.Reason, e.ReasonEn)
			}
			return i18n.TIn(l,
				"déclaré non applicable pour "+providerName+" (mécanisme inexistant)",
				"declared not applicable for "+providerName+" (no such mechanism)")
		}
	}
	if t := ControlType(code); t != "" {
		if tc, ok := d.Contrat.Types[t]; ok && tc.Etat == "absent" {
			if tc.Reason != "" {
				return i18n.PickIn(l, tc.Reason, tc.ReasonEn)
			}
			return i18n.TIn(l,
				"type de ressource « "+t+" » absent de l'API "+providerName,
				"resource type \""+t+"\" absent from the "+providerName+" API")
		}
	}
	return ""
}

// ControlType retourne le type de ressource normalisé visé par un contrôle,
// déduit de son code agnostique (préfixe de service). Les contrôles de
// gouvernance sont multi-types et renvoient "".
func ControlType(code string) string {
	switch {
	// Cas spécifique AVANT le préfixe network_securitygroup : cette règle lit l'OBJET
	// security_group (politique par défaut), pas une security_group_rule.
	case code == "network_securitygroup_default_deny":
		return "security_group"
	case strings.HasPrefix(code, "network_securitygroup"):
		return "security_group_rule"
	case code == "network_flow_matrix_documented":
		return "security_group_rule"
	case code == "network_documented":
		return "network"
	case code == "network_peering_cross_organization":
		return "network_peering"
	case code == "network_subnet_no_public_ip_by_default":
		return "subnet"
	case strings.HasPrefix(code, "database_"):
		return "managed_database"
	case strings.HasPrefix(code, "compute_image"):
		return "compute_image"
	case strings.HasPrefix(code, "compute_instance"):
		return "compute_instance"
	case strings.HasPrefix(code, "kubernetes_cluster"):
		return "kubernetes_cluster"
	// État DANS le cluster (mode --kubeconfig), distinct du plan de contrôle managé.
	case strings.HasPrefix(code, "k8s_rbac"):
		return "k8s_cluster_role_binding"
	case strings.HasPrefix(code, "k8s_namespace"):
		return "k8s_namespace"
	case strings.HasPrefix(code, "k8s_secrets"):
		return "k8s_crd"
	case strings.HasPrefix(code, "objectstorage_bucket"):
		return "object_storage_bucket"
	case strings.HasPrefix(code, "blockstorage_snapshot"):
		return "blockstorage_snapshot"
	case strings.HasPrefix(code, "blockstorage_volume"):
		return "blockstorage_volume"
	case strings.HasPrefix(code, "loadbalancer"):
		return "load_balancer"
	case code == "iam_user_mfa_enabled":
		return "iam_user"
	case code == "iam_account_mfa_enforced":
		return "api_access_policy"
	case code == "iam_no_root_access_key", code == "iam_accesskey_expiration_set":
		return "access_key"
	case strings.HasPrefix(code, "iam_role"):
		return "iam_role"
	case strings.HasPrefix(code, "iam_policy"):
		return "iam_policy"
	case code == "iam_apiaccesspolicy_max_key_expiration":
		return "api_access_policy"
	case code == "iam_apiaccessrule_no_public_cidr":
		return "api_access_rule"
	case code == "iam_apiaccessrule_defined":
		return "api_access_summary"
	}
	return ""
}

// NonCloudProviders retourne les providers dont la PORTÉE n'est pas le plan de contrôle
// cloud (ex. `kubernetes`, qui audite l'intérieur d'un cluster). Les comparer en parité
// avec des clouds n'a pas de sens : ni les uns ni les autres ne peuvent couvrir les
// exigences de l'autre portée.
func NonCloudProviders() map[string]bool {
	out := map[string]bool{}
	for name, d := range registry {
		if d.Scope != "" && d.Scope != "cloud" {
			out[name] = true
		}
	}
	return out
}

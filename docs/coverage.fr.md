> [🇬🇧 English](coverage.md) · 🇫🇷 Français

<!-- PAGE GÉNÉRÉE : ne pas éditer à la main. Régénérer avec `mise run gen-docs`. -->

# Matrice de couverture

Ce que Pépin sait réellement mesurer, par contrôle, par fournisseur et par source.
La page est **calculée** depuis `referentiel/controles.yaml` (codes, sévérités, `fournisseurs`),
`providers/*.yaml` (spec de collecte live, mapping Terraform, contrat d'API) et le verrou du
« pass » de `internal/assess`. Elle décrit ce que Pépin **peut conclure**, pas le résultat d'un
scan donné : un contrôle marqué ✅ peut parfaitement rendre `not-evaluated` sur un inventaire
qui ne contient aucune ressource du type visé.

Les titres de contrôles et les justifications proviennent du référentiel et des contrats de
fournisseurs, qui sont bilingues : cette page cite la version française, la page anglaise
cite la version anglaise. Le français reste la langue de référence du contenu normatif.

## Légende

| Symbole | Statut | Ce que Pépin peut conclure |
|---|---|---|
| ✅ | `supported` | la source produit le type de ressource visé, le contrat le déclare `verifie`, et l'attribut décisif est projeté. Pépin peut rendre `pass` ou `fail`. |
| ◐ | `partial` | la source produit le type, mais le verrou du « pass » ne peut pas être levé (contrat non `verifie`, ou attribut décisif non projeté par cette source). Le scan rend `not-evaluated`, jamais un vert silencieux. |
| ∅ | `not-applicable` | le contrat du fournisseur déclare le contrôle non testable, avec sa justification (mécanisme inexistant côté API, ou type de ressource absent). Le scan rend `not-applicable` accompagné de ce motif. |
| ✗ | `unsupported` | le contrôle n'est pas déclaré pour ce fournisseur dans le référentiel, ou cette source ne produit aucune ressource du type qu'il lit. Rien ne sera conclu depuis cette source. |

**◐ n'est pas « à moitié conforme ».** C'est « Pépin ne peut pas décider depuis cette source »,
et le rapport le dit à chaque scan, contrôle par contrôle, avec le motif.

## Synthèse par famille du référentiel

| Famille | Contrôles | exoscale | outscale | scaleway |
|---|---:|---|---|---|
| `chiffrement` | 7 | ✅ 1 · ◐ 0 · ∅ 3 · ✗ 3 | ✅ 2 · ◐ 0 · ∅ 3 · ✗ 2 | ✅ 2 · ◐ 0 · ∅ 1 · ✗ 4 |
| `compute` | 9 | ✅ 4 · ◐ 0 · ∅ 0 · ✗ 5 | ✅ 7 · ◐ 0 · ∅ 0 · ✗ 2 | ✅ 2 · ◐ 0 · ∅ 0 · ✗ 7 |
| `gouvernance` | 3 | ✅ 2 · ◐ 1 · ∅ 0 · ✗ 0 | ✅ 2 · ◐ 1 · ∅ 0 · ✗ 0 | ✅ 2 · ◐ 1 · ∅ 0 · ✗ 0 |
| `iam` | 14 | ✅ 4 · ◐ 0 · ∅ 0 · ✗ 10 | ✅ 10 · ◐ 0 · ∅ 1 · ✗ 3 | ✅ 3 · ◐ 1 · ∅ 0 · ✗ 10 |
| `journalisation` | 2 | ✅ 1 · ◐ 0 · ∅ 1 · ✗ 0 | ✅ 1 · ◐ 0 · ∅ 0 · ✗ 1 | ✅ 0 · ◐ 0 · ∅ 0 · ✗ 2 |
| `reseau` | 15 | ✅ 9 · ◐ 0 · ∅ 0 · ✗ 6 | ✅ 11 · ◐ 0 · ∅ 0 · ✗ 4 | ✅ 9 · ◐ 1 · ∅ 0 · ✗ 5 |
| `stockage` | 7 | ✅ 4 · ◐ 0 · ∅ 1 · ✗ 2 | ✅ 6 · ◐ 0 · ∅ 0 · ✗ 1 | ✅ 4 · ◐ 0 · ∅ 1 · ✗ 2 |

## Matrice complète (contrôle × fournisseur × source)

| Contrôle | Sévérité | SCSL | exoscale TF | exoscale live | outscale TF | outscale live | scaleway TF | scaleway live |
|---|---|---|:-:|:-:|:-:|:-:|:-:|:-:|
| `blockstorage_snapshot_not_public` | high | CLD-STO-2 | ∅ | ∅ | ✗ | ✅ | ∅ | ∅ |
| `blockstorage_volume_encryption` | high | CLD-CHF-2 | ✅ | ✅ | ∅ | ∅ | ∅ | ∅ |
| `blockstorage_volume_snapshots_exist` | high | CLD-STO-3 | ✅ | ✅ | ✗ | ✅ | ✗ | ✗ |
| `compute_image_not_public` | high | CLD-STO-2 | ✗ | ✗ | ✅ | ✅ | ✗ | ✗ |
| `compute_instance_deletion_protection` | medium | CLD-CMP-10 | ✗ | ✗ | ◐ | ✅ | ✗ | ✗ |
| `compute_instance_has_security_group` | critical | CLD-CMP-1 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `compute_instance_no_secrets_in_user_data` | high | CLD-CMP-9 | ✅ | ✅ | ✅ | ✅ | ✅ | ◐ |
| `compute_instance_public_ip_with_open_securitygroup` | critical | CLD-NET-3 | ✅ | ✅ | ✅ | ✅ | ◐ | ✅ |
| `database_backup_enabled` | high | CLD-STO-3 | ✗ | ✗ | ✗ | ✗ | ✅ | ✗ |
| `database_encryption_at_rest_enabled` | high | CLD-CHF-2 | ✗ | ✗ | ✗ | ✗ | ✅ | ✗ |
| `database_service_not_open_to_internet` | high | CLD-NET-1 | ✗ | ✗ | ✗ | ✗ | ✅ | ✗ |
| `governance_provider_sovereignty` | high | CLD-GVN-4 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `governance_resource_region_in_eu` | high | CLD-GVN-3 | ✅ | ✅ | ◐ | ✅ | ✅ | ✅ |
| `governance_resource_required_tags` | medium | CLD-GVN-1 | ◐ | ◐ | ◐ | ◐ | ◐ | ◐ |
| `iam_accesskey_expiration_set` | critical | CLD-IAM-2 | ✗ | ✗ | ✗ | ✅ | ✅ | ✅ |
| `iam_account_mfa_enforced` | high | CLD-IAM-3 | ✗ | ✗ | ✗ | ✅ | ✗ | ✗ |
| `iam_apiaccesspolicy_max_key_expiration` | medium | CLD-IAM-2 | ✗ | ✗ | ✗ | ✅ | ✗ | ✗ |
| `iam_apiaccessrule_defined` | high | CLD-IAM-4 | ✗ | ✗ | ✗ | ✅ | ✗ | ✗ |
| `iam_apiaccessrule_no_public_cidr` | high | CLD-IAM-4 | ✗ | ✗ | ✗ | ✅ | ✗ | ✗ |
| `iam_no_root_access_key` | high | CLD-IAM-1 | ✗ | ✗ | ✗ | ✅ | ◐ | ◐ |
| `iam_policy_no_administrative_privileges` | critical | CLD-IAM-1 | ✗ | ✗ | ✅ | ✅ | ✗ | ✗ |
| `iam_policy_no_notaction_notresource` | critical | CLD-IAM-1 | ✗ | ✗ | ✅ | ✅ | ✗ | ✗ |
| `iam_policy_no_privilege_escalation` | high | CLD-IAM-12 | ✗ | ✗ | ✅ | ✅ | ✅ | ✗ |
| `iam_policy_no_wildcard_resource` | high | CLD-IAM-1 | ✗ | ✗ | ✅ | ✅ | ✗ | ✗ |
| `iam_role_key_lifetime_bounded` | critical | CLD-IAM-2 | ✅ | ✅ | ✗ | ✗ | ✗ | ✗ |
| `iam_role_no_admin_privileges` | high | CLD-IAM-1 | ✅ | ✅ | ✗ | ✗ | ✗ | ✗ |
| `iam_role_source_ip_restricted` | high | CLD-IAM-4 | ✅ | ✅ | ✗ | ✗ | ✗ | ✗ |
| `iam_user_mfa_enabled` | high | CLD-IAM-3 | ✗ | ✅ | ∅ | ∅ | ✗ | ✅ |
| `k8s_namespace_network_policy_defined` | high | CLD-K8S-6 | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| `k8s_namespace_pod_security_enforced` | high | CLD-K8S-5 | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| `k8s_rbac_no_cluster_admin_binding` | critical | CLD-K8S-4 | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| `k8s_secrets_external_manager` | high | CLD-K8S-10 | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |
| `kubernetes_cluster_audit_logging_enabled` | medium | CLD-LOG-1 | ✅ | ✅ | ✗ | ✗ | ✗ | ✗ |
| `kubernetes_cluster_auto_upgrade_enabled` | medium | CLD-K8S-3 | ✅ | ✅ | ✗ | ✅ | ✗ | ✗ |
| `kubernetes_cluster_control_plane_highly_available` | high | CLD-K8S-2 | ✅ | ✅ | ✗ | ✅ | ✗ | ✗ |
| `kubernetes_cluster_deletion_protection` | medium | CLD-K8S-3 | ✗ | ✗ | ✗ | ✅ | ✗ | ✗ |
| `kubernetes_cluster_not_publicly_accessible` | critical | CLD-K8S-1 | ✗ | ✗ | ✗ | ✅ | ✗ | ✗ |
| `loadbalancer_http_redirect_to_https` | medium | CLD-CHF-1 | ∅ | ∅ | ∅ | ∅ | ✗ | ✗ |
| `loadbalancer_logging_enabled` | medium | CLD-LOG-1 | ∅ | ∅ | ✗ | ✅ | ✗ | ✗ |
| `loadbalancer_ssl_listeners` | high | CLD-CHF-1 | ∅ | ∅ | ✗ | ✅ | ✗ | ✗ |
| `network_documented` | low | CLD-NET-5 | ✅ | ✅ | ✅ | ✅ | ◐ | ✗ |
| `network_flow_matrix_documented` | medium | CLD-NET-5 | ✅ | ✅ | ✗ | ✗ | ✗ | ✗ |
| `network_peering_cross_organization` | high | CLD-NET-7 | ✗ | ✗ | ✗ | ✅ | ✗ | ✗ |
| `network_securitygroup_allow_ingress_from_internet_to_all_ports` | critical | CLD-NET-2 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `network_securitygroup_allow_ingress_from_internet_to_high_risk_tcp_ports` | high | CLD-NET-1 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `network_securitygroup_allow_ingress_from_internet_to_high_risk_udp_ports` | high | CLD-NET-1 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `network_securitygroup_allow_ingress_from_internet_to_tcp_port_22` | high | CLD-NET-1, CLD-IAM-6, CLD-NET-6 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `network_securitygroup_allow_ingress_from_internet_to_tcp_port_3389` | high | CLD-NET-1, CLD-IAM-6, CLD-NET-6 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `network_securitygroup_default_deny` | high | CLD-NET-2 | ✗ | ✗ | ✗ | ✗ | ✅ | ✗ |
| `network_securitygroup_default_restrict_traffic` | high | CLD-NET-4 | ✗ | ✗ | ◐ | ✅ | ✗ | ✗ |
| `network_securitygroup_unrestricted_egress` | medium | CLD-NET-4 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `network_subnet_no_public_ip_by_default` | medium | CLD-NET-3 | ✗ | ✗ | ✅ | ✅ | ✗ | ✗ |
| `objectstorage_bucket_default_encryption` | high | CLD-CHF-2 | ✗ | ✗ | ✗ | ✅ | ✗ | ✗ |
| `objectstorage_bucket_kms_encryption` | medium | CLD-CHF-4 | ∅ | ∅ | ∅ | ∅ | ◐ | ✅ |
| `objectstorage_bucket_object_lock_enabled` | low | CLD-STO-8 | ✗ | ✅ | ✗ | ✅ | ✅ | ✅ |
| `objectstorage_bucket_public_access` | critical | CLD-STO-1 | ✗ | ✅ | ✗ | ✅ | ✅ | ✅ |
| `objectstorage_bucket_versioning_enabled` | medium | CLD-STO-4 | ✗ | ✅ | ✗ | ✅ | ◐ | ✅ |

## Pourquoi une case n'est pas ✅

Une case qui n'est pas ✅ **alors que le contrôle est déclaré pour ce fournisseur** porte
toujours son motif. Les cases « contrôle non déclaré pour ce fournisseur » ne sont pas reprises
ici : la matrice les montre déjà, et elles n'apprennent rien de plus.

| Contrôle | Fournisseur | Source | Statut | Motif |
|---|---|---|---|---|
| `blockstorage_snapshot_not_public` | exoscale | terraform | ∅ `not-applicable` | Snapshots block-storage Exoscale non exportables/partageables (doc officielle) : le risque d'exposition publique est structurellement absent, conforme par construction (STO-2). |
| `blockstorage_snapshot_not_public` | exoscale | live | ∅ `not-applicable` | Snapshots block-storage Exoscale non exportables/partageables (doc officielle) : le risque d'exposition publique est structurellement absent, conforme par construction (STO-2). |
| `blockstorage_snapshot_not_public` | outscale | terraform | ✗ `unsupported` | cette source ne produit aucune ressource de type « blockstorage_snapshot » |
| `blockstorage_snapshot_not_public` | scaleway | terraform | ∅ `not-applicable` | Les snapshots block Scaleway (api/block/v1) n'exposent aucun mécanisme de partage ou d'export public : le risque d'exposition publique est structurellement absent, conforme par construction (STO-2). |
| `blockstorage_snapshot_not_public` | scaleway | live | ∅ `not-applicable` | Les snapshots block Scaleway (api/block/v1) n'exposent aucun mécanisme de partage ou d'export public : le risque d'exposition publique est structurellement absent, conforme par construction (STO-2). |
| `blockstorage_volume_encryption` | outscale | terraform | ∅ `not-applicable` | osc-sdk-go/v2 Volume n'expose aucun champ de chiffrement ; le chiffrement au repos est côté invité (EncFS/LUKS), responsabilité du client → non observable côté plateforme (CHF-2). |
| `blockstorage_volume_encryption` | outscale | live | ∅ `not-applicable` | osc-sdk-go/v2 Volume n'expose aucun champ de chiffrement ; le chiffrement au repos est côté invité (EncFS/LUKS), responsabilité du client → non observable côté plateforme (CHF-2). |
| `blockstorage_volume_encryption` | scaleway | terraform | ∅ `not-applicable` | Chiffrement au repos des volumes block côté invité (LUKS/Cryptsetup), responsabilité du client (responsabilité partagée) ; l'API block n'expose aucun champ de chiffrement → non observable côté plateforme (CHF-2). |
| `blockstorage_volume_encryption` | scaleway | live | ∅ `not-applicable` | Chiffrement au repos des volumes block côté invité (LUKS/Cryptsetup), responsabilité du client (responsabilité partagée) ; l'API block n'expose aucun champ de chiffrement → non observable côté plateforme (CHF-2). |
| `blockstorage_volume_snapshots_exist` | outscale | terraform | ✗ `unsupported` | cette source ne produit aucune ressource de type « blockstorage_volume » |
| `compute_instance_deletion_protection` | outscale | terraform | ◐ `partial` | attribut décisif « deletion_protection » non projeté par cette source : garde de capacité, le scan rend « not-evaluated » |
| `compute_instance_no_secrets_in_user_data` | scaleway | live | ◐ `partial` | attribut décisif « user_data » non projeté par cette source : garde de capacité, le scan rend « not-evaluated » |
| `compute_instance_public_ip_with_open_securitygroup` | scaleway | terraform | ◐ `partial` | attribut décisif « public_ip » non projeté par cette source : garde de capacité, le scan rend « not-evaluated » |
| `database_backup_enabled` | scaleway | live | ✗ `unsupported` | cette source ne produit aucune ressource de type « managed_database » |
| `database_encryption_at_rest_enabled` | scaleway | live | ✗ `unsupported` | cette source ne produit aucune ressource de type « managed_database » |
| `database_service_not_open_to_internet` | scaleway | live | ✗ `unsupported` | cette source ne produit aucune ressource de type « managed_database » |
| `governance_resource_region_in_eu` | outscale | terraform | ◐ `partial` | attribut décisif « region » non projeté par cette source : garde de capacité, le scan rend « not-evaluated » |
| `governance_resource_required_tags` | exoscale | terraform | ◐ `partial` | aucun type de ressource visé et le contrôle ne lit pas le descripteur du fournisseur : le verrou du « pass » ne peut pas être levé, le scan rend « not-evaluated » tant qu'aucun écart n'est détecté |
| `governance_resource_required_tags` | exoscale | live | ◐ `partial` | aucun type de ressource visé et le contrôle ne lit pas le descripteur du fournisseur : le verrou du « pass » ne peut pas être levé, le scan rend « not-evaluated » tant qu'aucun écart n'est détecté |
| `governance_resource_required_tags` | outscale | terraform | ◐ `partial` | aucun type de ressource visé et le contrôle ne lit pas le descripteur du fournisseur : le verrou du « pass » ne peut pas être levé, le scan rend « not-evaluated » tant qu'aucun écart n'est détecté |
| `governance_resource_required_tags` | outscale | live | ◐ `partial` | aucun type de ressource visé et le contrôle ne lit pas le descripteur du fournisseur : le verrou du « pass » ne peut pas être levé, le scan rend « not-evaluated » tant qu'aucun écart n'est détecté |
| `governance_resource_required_tags` | scaleway | terraform | ◐ `partial` | aucun type de ressource visé et le contrôle ne lit pas le descripteur du fournisseur : le verrou du « pass » ne peut pas être levé, le scan rend « not-evaluated » tant qu'aucun écart n'est détecté |
| `governance_resource_required_tags` | scaleway | live | ◐ `partial` | aucun type de ressource visé et le contrôle ne lit pas le descripteur du fournisseur : le verrou du « pass » ne peut pas être levé, le scan rend « not-evaluated » tant qu'aucun écart n'est détecté |
| `iam_accesskey_expiration_set` | outscale | terraform | ✗ `unsupported` | cette source ne produit aucune ressource de type « access_key » |
| `iam_account_mfa_enforced` | outscale | terraform | ✗ `unsupported` | cette source ne produit aucune ressource de type « api_access_policy » |
| `iam_apiaccesspolicy_max_key_expiration` | outscale | terraform | ✗ `unsupported` | cette source ne produit aucune ressource de type « api_access_policy » |
| `iam_apiaccessrule_defined` | outscale | terraform | ✗ `unsupported` | cette source ne produit aucune ressource de type « api_access_summary » |
| `iam_apiaccessrule_no_public_cidr` | outscale | terraform | ✗ `unsupported` | cette source ne produit aucune ressource de type « api_access_rule » |
| `iam_no_root_access_key` | outscale | terraform | ✗ `unsupported` | cette source ne produit aucune ressource de type « access_key » |
| `iam_no_root_access_key` | scaleway | terraform | ◐ `partial` | attribut décisif « root_owned / scope » non projeté par cette source : garde de capacité, le scan rend « not-evaluated » |
| `iam_no_root_access_key` | scaleway | live | ◐ `partial` | attribut décisif « root_owned / scope » non projeté par cette source : garde de capacité, le scan rend « not-evaluated » |
| `iam_policy_no_privilege_escalation` | scaleway | live | ✗ `unsupported` | cette source ne produit aucune ressource de type « iam_policy » |
| `iam_user_mfa_enabled` | exoscale | terraform | ✗ `unsupported` | cette source ne produit aucune ressource de type « iam_user » |
| `iam_user_mfa_enabled` | outscale | terraform | ∅ `not-applicable` | type de ressource « iam_user » absent de l'API outscale |
| `iam_user_mfa_enabled` | outscale | live | ∅ `not-applicable` | type de ressource « iam_user » absent de l'API outscale |
| `iam_user_mfa_enabled` | scaleway | terraform | ✗ `unsupported` | cette source ne produit aucune ressource de type « iam_user » |
| `kubernetes_cluster_auto_upgrade_enabled` | outscale | terraform | ✗ `unsupported` | cette source ne produit aucune ressource de type « kubernetes_cluster » |
| `kubernetes_cluster_control_plane_highly_available` | outscale | terraform | ✗ `unsupported` | cette source ne produit aucune ressource de type « kubernetes_cluster » |
| `kubernetes_cluster_deletion_protection` | outscale | terraform | ✗ `unsupported` | cette source ne produit aucune ressource de type « kubernetes_cluster » |
| `kubernetes_cluster_not_publicly_accessible` | outscale | terraform | ✗ `unsupported` | cette source ne produit aucune ressource de type « kubernetes_cluster » |
| `loadbalancer_http_redirect_to_https` | exoscale | terraform | ∅ `not-applicable` | type de ressource « load_balancer » absent de l'API exoscale |
| `loadbalancer_http_redirect_to_https` | exoscale | live | ∅ `not-applicable` | type de ressource « load_balancer » absent de l'API exoscale |
| `loadbalancer_http_redirect_to_https` | outscale | terraform | ∅ `not-applicable` | Le LBU Outscale ne peut pas rediriger : `ListenerRule.Action` est documenté « always forward » au contrat OAPI (aucune action de redirection), et aucun attribut de redirection n'existe sur `Listener`. Le mécanisme est inexistant → contrôle non applicable (CHF-1). |
| `loadbalancer_http_redirect_to_https` | outscale | live | ∅ `not-applicable` | Le LBU Outscale ne peut pas rediriger : `ListenerRule.Action` est documenté « always forward » au contrat OAPI (aucune action de redirection), et aucun attribut de redirection n'existe sur `Listener`. Le mécanisme est inexistant → contrôle non applicable (CHF-1). |
| `loadbalancer_logging_enabled` | exoscale | terraform | ∅ `not-applicable` | type de ressource « load_balancer » absent de l'API exoscale |
| `loadbalancer_logging_enabled` | exoscale | live | ∅ `not-applicable` | type de ressource « load_balancer » absent de l'API exoscale |
| `loadbalancer_logging_enabled` | outscale | terraform | ✗ `unsupported` | cette source ne produit aucune ressource de type « load_balancer » |
| `loadbalancer_ssl_listeners` | exoscale | terraform | ∅ `not-applicable` | type de ressource « load_balancer » absent de l'API exoscale |
| `loadbalancer_ssl_listeners` | exoscale | live | ∅ `not-applicable` | type de ressource « load_balancer » absent de l'API exoscale |
| `loadbalancer_ssl_listeners` | outscale | terraform | ✗ `unsupported` | cette source ne produit aucune ressource de type « load_balancer » |
| `network_documented` | scaleway | terraform | ◐ `partial` | contrat du fournisseur : le type « network » n'est pas déclaré `verifie` (état : a_verifier) |
| `network_documented` | scaleway | live | ✗ `unsupported` | cette source ne produit aucune ressource de type « network » |
| `network_peering_cross_organization` | outscale | terraform | ✗ `unsupported` | cette source ne produit aucune ressource de type « network_peering » |
| `network_securitygroup_default_deny` | scaleway | live | ✗ `unsupported` | cette source ne produit aucune ressource de type « security_group » |
| `network_securitygroup_default_restrict_traffic` | outscale | terraform | ◐ `partial` | attribut décisif « security_group_name » non projeté par cette source : garde de capacité, le scan rend « not-evaluated » |
| `objectstorage_bucket_default_encryption` | outscale | terraform | ✗ `unsupported` | cette source ne produit aucune ressource de type « object_storage_bucket » |
| `objectstorage_bucket_kms_encryption` | exoscale | terraform | ∅ `not-applicable` | SOS chiffre au repos par défaut (SSE-SOS, clés gérées par Exoscale, type SSE-S3) mais n'expose pas de BYOK/KMS géré par le client au niveau bucket (SSE-C reste par-objet, non observable) → le contrôle BYOK-au-bucket est sans objet (CHF-4). |
| `objectstorage_bucket_kms_encryption` | exoscale | live | ∅ `not-applicable` | SOS chiffre au repos par défaut (SSE-SOS, clés gérées par Exoscale, type SSE-S3) mais n'expose pas de BYOK/KMS géré par le client au niveau bucket (SSE-C reste par-objet, non observable) → le contrôle BYOK-au-bucket est sans objet (CHF-4). |
| `objectstorage_bucket_kms_encryption` | outscale | terraform | ∅ `not-applicable` | OOS chiffre côté serveur en AES256 avec une clé FOURNISSEUR ; il n'existe ni service KMS ni clé maître gérée par le client, donc pas de BYOK à auditer au niveau bucket (CHF-4). NB : l'activation du SSE elle-même est opt-in et observable — elle relève d'un contrôle distinct, pas de ce N/A. |
| `objectstorage_bucket_kms_encryption` | outscale | live | ∅ `not-applicable` | OOS chiffre côté serveur en AES256 avec une clé FOURNISSEUR ; il n'existe ni service KMS ni clé maître gérée par le client, donc pas de BYOK à auditer au niveau bucket (CHF-4). NB : l'activation du SSE elle-même est opt-in et observable — elle relève d'un contrôle distinct, pas de ce N/A. |
| `objectstorage_bucket_kms_encryption` | scaleway | terraform | ◐ `partial` | attribut décisif « sse_kms_enabled » non projeté par cette source : garde de capacité, le scan rend « not-evaluated » |
| `objectstorage_bucket_object_lock_enabled` | exoscale | terraform | ✗ `unsupported` | cette source ne produit aucune ressource de type « object_storage_bucket » |
| `objectstorage_bucket_object_lock_enabled` | outscale | terraform | ✗ `unsupported` | cette source ne produit aucune ressource de type « object_storage_bucket » |
| `objectstorage_bucket_public_access` | exoscale | terraform | ✗ `unsupported` | cette source ne produit aucune ressource de type « object_storage_bucket » |
| `objectstorage_bucket_public_access` | outscale | terraform | ✗ `unsupported` | cette source ne produit aucune ressource de type « object_storage_bucket » |
| `objectstorage_bucket_versioning_enabled` | exoscale | terraform | ✗ `unsupported` | cette source ne produit aucune ressource de type « object_storage_bucket » |
| `objectstorage_bucket_versioning_enabled` | outscale | terraform | ✗ `unsupported` | cette source ne produit aucune ressource de type « object_storage_bucket » |
| `objectstorage_bucket_versioning_enabled` | scaleway | terraform | ◐ `partial` | attribut décisif « versioning » non projeté par cette source : garde de capacité, le scan rend « not-evaluated » |

## Fournisseur d'une autre portée : `kubernetes` (in-cluster)

Ce fournisseur audite l'état **dans** un cluster (RBAC, Pod Security, NetworkPolicy), pas un
plan de contrôle cloud. Le comparer en parité avec un cloud n'aurait pas de sens : aucun des
deux ne peut couvrir la portée de l'autre. Une seule source : la collecte live via kubeconfig.

| Contrôle | Sévérité | kubernetes |
|---|---|:-:|
| `k8s_namespace_network_policy_defined` | high | ✅ |
| `k8s_namespace_pod_security_enforced` | high | ✅ |
| `k8s_rbac_no_cluster_admin_binding` | critical | ✅ |
| `k8s_secrets_external_manager` | high | ✅ |

## Totaux

| Fournisseur | Source | ✅ `supported` | ◐ `partial` | ∅ `not-applicable` | ✗ `unsupported` |
|---|---|---:|---:|---:|---:|
| exoscale | terraform | 21 | 1 | 5 | 30 |
| exoscale | live | 25 | 1 | 5 | 26 |
| outscale | terraform | 17 | 4 | 4 | 32 |
| outscale | live | 39 | 1 | 4 | 13 |
| scaleway | terraform | 18 | 6 | 2 | 31 |
| scaleway | live | 16 | 3 | 2 | 36 |
| kubernetes | live | 4 | 0 | 0 | 53 |

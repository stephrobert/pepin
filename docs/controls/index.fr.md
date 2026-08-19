> [🇬🇧 English](index.md) · 🇫🇷 Français

<!-- PAGE GÉNÉRÉE : ne pas éditer à la main. Régénérer avec `mise run gen-docs`. -->

# Catalogue des contrôles

Un contrôle par page, calculé depuis `referentiel/controles.yaml` (la source de vérité),
les descripteurs `providers/*.yaml` et le verrou du « pass » de `internal/assess`.
Aucune métadonnée n'est recopiée ici : ajouter un contrôle, changer une sévérité ou
retirer un fournisseur réécrit ces pages, et la CI refuse une documentation en retard.

Ce catalogue dit ce que Pépin **peut conclure**, pas le résultat d'un scan. Pour la
vue d'ensemble par fournisseur et par source, voir la [matrice de couverture](../coverage.md).

## Chiffres

| Chiffre | Nombre |
|---|---:|
| Contrôles au référentiel | 57 |
| Contrôles actifs | 56 |
| Contrôles dormants | 1 |
| `critical` | 10 |
| `high` | 32 |
| `medium` | 13 |
| `low` | 2 |
| Preuves de remédiation déployables | 4 / 95 |

## Comment lire ce catalogue

- **Contrôle actif** : déclaré pour au moins un fournisseur (`fournisseurs:` non vide).
  Un scan de ce fournisseur peut lui donner un statut.
- **Contrôle dormant** : écrit et testé, déclaré pour aucun fournisseur. Il n'est jamais
  évalué, et il ne compte dans aucune couverture. La liste complète est en bas de page.
- **Preuves de remédiation** : les montages déployables présents sous
  [`references/remediation/`](../../references/remediation/README.md). Le compte est
  celui des couples (contrôle, fournisseur) déclarés, et il est aujourd'hui partiel :
  la remédiation *textuelle*, elle, est portée par chaque écart émis.
- Les contrôles encore au triage, non retenus dans le référentiel actif, vivent au
  [catalogue de triage](../../referentiel/catalogue.yaml).

## `iam`

| Contrôle | Sévérité | SCSL | Actif pour | Preuves |
|---|---|---|---|---|
| [`iam_accesskey_expiration_set`](iam_accesskey_expiration_set.fr.md) Clé d'accès sans expiration ni rotation | critical | `CLD-IAM-2` | `outscale`, `scaleway` | 0 / 2 |
| [`iam_account_mfa_enforced`](iam_account_mfa_enforced.fr.md) MFA non imposée au niveau du compte | high | `CLD-IAM-3` | `outscale` | 0 / 1 |
| [`iam_apiaccesspolicy_max_key_expiration`](iam_apiaccesspolicy_max_key_expiration.fr.md) Politique d'accès API sans expiration maximale des clés | medium | `CLD-IAM-2` | `outscale` | 0 / 1 |
| [`iam_apiaccessrule_defined`](iam_apiaccessrule_defined.fr.md) Aucune règle d'accès API définie | high | `CLD-IAM-4` | `outscale` | 0 / 1 |
| [`iam_apiaccessrule_no_public_cidr`](iam_apiaccessrule_no_public_cidr.fr.md) Règle d'accès API ouverte à un CIDR public | high | `CLD-IAM-4` | `outscale` | 0 / 1 |
| [`iam_no_root_access_key`](iam_no_root_access_key.fr.md) Clé d'accès rattachée au compte root | high | `CLD-IAM-1` | `outscale`, `scaleway` | 0 / 2 |
| [`iam_policy_no_administrative_privileges`](iam_policy_no_administrative_privileges.fr.md) Politique IAM à privilèges administratifs (action joker) | critical | `CLD-IAM-1` | `outscale` | 0 / 1 |
| [`iam_policy_no_notaction_notresource`](iam_policy_no_notaction_notresource.fr.md) Politique IAM avec NotAction/NotResource (anti-pattern) | critical | `CLD-IAM-1` | `outscale` | 0 / 1 |
| [`iam_policy_no_privilege_escalation`](iam_policy_no_privilege_escalation.fr.md) Politique IAM permettant une élévation de privilèges | high | `CLD-IAM-12` | `outscale`, `scaleway` | 0 / 2 |
| [`iam_policy_no_wildcard_resource`](iam_policy_no_wildcard_resource.fr.md) Politique IAM avec ressource joker | high | `CLD-IAM-1` | `outscale` | 0 / 1 |
| [`iam_role_key_lifetime_bounded`](iam_role_key_lifetime_bounded.fr.md) Rôle IAM sans borne de durée de vie des accès | critical | `CLD-IAM-2` | `exoscale` | 1 / 1 |
| [`iam_role_no_admin_privileges`](iam_role_no_admin_privileges.fr.md) Rôle IAM aux privilèges excessifs | high | `CLD-IAM-1` | `exoscale` | 1 / 1 |
| [`iam_role_source_ip_restricted`](iam_role_source_ip_restricted.fr.md) Rôle IAM sans restriction d'IP source | high | `CLD-IAM-4` | `exoscale` | 1 / 1 |
| [`iam_user_mfa_enabled`](iam_user_mfa_enabled.fr.md) MFA non activée sur un compte | high | `CLD-IAM-3` | `exoscale`, `scaleway` | 0 / 2 |

## `reseau`

| Contrôle | Sévérité | SCSL | Actif pour | Preuves |
|---|---|---|---|---|
| [`compute_instance_public_ip_with_open_securitygroup`](compute_instance_public_ip_with_open_securitygroup.fr.md) Machine exposée publiquement sans filtrage restrictif | critical | `CLD-NET-3` | `exoscale`, `outscale`, `scaleway` | 0 / 3 |
| [`database_service_not_open_to_internet`](database_service_not_open_to_internet.fr.md) Base de données managée joignable depuis Internet | high | `CLD-NET-1` | `scaleway` | 0 / 1 |
| [`k8s_namespace_network_policy_defined`](k8s_namespace_network_policy_defined.fr.md) Namespace sans NetworkPolicy (réseau plat) | high | `CLD-K8S-6` | `kubernetes` | 0 / 1 |
| [`network_documented`](network_documented.fr.md) Réseau non documenté (cartographie non tenue) | low | `CLD-NET-5` | `exoscale`, `outscale`, `scaleway` | 0 / 3 |
| [`network_flow_matrix_documented`](network_flow_matrix_documented.fr.md) Flux entrant sans justification dans la matrice des flux | medium | `CLD-NET-5` | `exoscale` | 0 / 1 |
| [`network_peering_cross_organization`](network_peering_cross_organization.fr.md) Appairage réseau vers un autre système d'information | high | `CLD-NET-7` | `outscale` | 0 / 1 |
| [`network_securitygroup_allow_ingress_from_internet_to_all_ports`](network_securitygroup_allow_ingress_from_internet_to_all_ports.fr.md) Tout le trafic entrant autorisé depuis Internet (any/any) | critical | `CLD-NET-2` | `exoscale`, `outscale`, `scaleway` | 0 / 3 |
| [`network_securitygroup_allow_ingress_from_internet_to_high_risk_tcp_ports`](network_securitygroup_allow_ingress_from_internet_to_high_risk_tcp_ports.fr.md) Port sensible (base de données, annuaire…) ouvert à Internet | high | `CLD-NET-1` | `exoscale`, `outscale`, `scaleway` | 0 / 3 |
| [`network_securitygroup_allow_ingress_from_internet_to_high_risk_udp_ports`](network_securitygroup_allow_ingress_from_internet_to_high_risk_udp_ports.fr.md) Service UDP sensible (amplification, non authentifié) ouvert à Internet | high | `CLD-NET-1` | `exoscale`, `outscale`, `scaleway` | 0 / 3 |
| [`network_securitygroup_allow_ingress_from_internet_to_tcp_port_22`](network_securitygroup_allow_ingress_from_internet_to_tcp_port_22.fr.md) SSH (port 22) ouvert à Internet | high | `CLD-NET-1`, `CLD-IAM-6`, `CLD-NET-6` | `exoscale`, `outscale`, `scaleway` | 0 / 3 |
| [`network_securitygroup_allow_ingress_from_internet_to_tcp_port_3389`](network_securitygroup_allow_ingress_from_internet_to_tcp_port_3389.fr.md) RDP (port 3389) ouvert à Internet | high | `CLD-NET-1`, `CLD-IAM-6`, `CLD-NET-6` | `exoscale`, `outscale`, `scaleway` | 0 / 3 |
| [`network_securitygroup_default_deny`](network_securitygroup_default_deny.fr.md) Politique entrante par défaut d'un groupe de sécurité en « accept » | high | `CLD-NET-2` | `scaleway` | 0 / 1 |
| [`network_securitygroup_default_restrict_traffic`](network_securitygroup_default_restrict_traffic.fr.md) Security group « default » non restrictif | high | `CLD-NET-4` | `outscale` | 0 / 1 |
| [`network_securitygroup_unrestricted_egress`](network_securitygroup_unrestricted_egress.fr.md) Filtrage sortant non restreint | medium | `CLD-NET-4` | `exoscale`, `outscale`, `scaleway` | 0 / 3 |
| [`network_subnet_no_public_ip_by_default`](network_subnet_no_public_ip_by_default.fr.md) Sous-réseau attribuant une IP publique par défaut | medium | `CLD-NET-3` | `outscale` | 0 / 1 |

## `compute`

| Contrôle | Sévérité | SCSL | Actif pour | Preuves |
|---|---|---|---|---|
| [`compute_instance_deletion_protection`](compute_instance_deletion_protection.fr.md) Instance sans protection contre la suppression | medium | `CLD-CMP-10` | `outscale` | 0 / 1 |
| [`compute_instance_has_security_group`](compute_instance_has_security_group.fr.md) Machine sans filtrage réseau | critical | `CLD-CMP-1` | `exoscale`, `outscale`, `scaleway` | 0 / 3 |
| [`compute_instance_no_secrets_in_user_data`](compute_instance_no_secrets_in_user_data.fr.md) Secret en clair dans les données utilisateur (user-data) | high | `CLD-CMP-9` | `exoscale`, `outscale`, `scaleway` | 0 / 3 |
| [`k8s_namespace_pod_security_enforced`](k8s_namespace_pod_security_enforced.fr.md) Pod Security Standards non imposés sur un namespace | high | `CLD-K8S-5` | `kubernetes` | 0 / 1 |
| [`k8s_rbac_no_cluster_admin_binding`](k8s_rbac_no_cluster_admin_binding.fr.md) cluster-admin accordé au-delà du binding natif | critical | `CLD-K8S-4` | `kubernetes` | 0 / 1 |
| [`kubernetes_cluster_auto_upgrade_enabled`](kubernetes_cluster_auto_upgrade_enabled.fr.md) Mises à jour automatiques du cluster Kubernetes désactivées | medium | `CLD-K8S-3` | `exoscale`, `outscale` | 0 / 2 |
| [`kubernetes_cluster_control_plane_highly_available`](kubernetes_cluster_control_plane_highly_available.fr.md) Plan de contrôle Kubernetes non hautement disponible | high | `CLD-K8S-2` | `exoscale`, `outscale` | 0 / 2 |
| [`kubernetes_cluster_deletion_protection`](kubernetes_cluster_deletion_protection.fr.md) Cluster Kubernetes sans protection contre la suppression | medium | `CLD-K8S-3` | `outscale` | 0 / 1 |
| [`kubernetes_cluster_not_publicly_accessible`](kubernetes_cluster_not_publicly_accessible.fr.md) API server Kubernetes exposé à Internet | critical | `CLD-K8S-1` | `outscale` | 0 / 1 |

## `stockage`

| Contrôle | Sévérité | SCSL | Actif pour | Preuves |
|---|---|---|---|---|
| [`blockstorage_snapshot_not_public`](blockstorage_snapshot_not_public.fr.md) Instantané ou image partagé publiquement | high | `CLD-STO-2` | `outscale` | 0 / 1 |
| [`blockstorage_volume_snapshots_exist`](blockstorage_volume_snapshots_exist.fr.md) Absence de sauvegarde récente | high | `CLD-STO-3` | `exoscale`, `outscale` | 0 / 2 |
| [`compute_image_not_public`](compute_image_not_public.fr.md) Image machine partagée publiquement | high | `CLD-STO-2` | `outscale` | 0 / 1 |
| [`database_backup_enabled`](database_backup_enabled.fr.md) Sauvegardes automatiques d'une base managée désactivées | high | `CLD-STO-3` | `scaleway` | 0 / 1 |
| [`objectstorage_bucket_object_lock_enabled`](objectstorage_bucket_object_lock_enabled.fr.md) Object Lock (immutabilité) désactivé sur le stockage objet | low | `CLD-STO-8` | `exoscale`, `outscale`, `scaleway` | 0 / 3 |
| [`objectstorage_bucket_public_access`](objectstorage_bucket_public_access.fr.md) Stockage objet exposé publiquement | critical | `CLD-STO-1` | `exoscale`, `outscale`, `scaleway` | 0 / 3 |
| [`objectstorage_bucket_versioning_enabled`](objectstorage_bucket_versioning_enabled.fr.md) Versioning du stockage objet désactivé | medium | `CLD-STO-4` | `exoscale`, `outscale`, `scaleway` | 0 / 3 |

## `chiffrement`

| Contrôle | Sévérité | SCSL | Actif pour | Preuves |
|---|---|---|---|---|
| [`blockstorage_volume_encryption`](blockstorage_volume_encryption.fr.md) Chiffrement au repos désactivé | high | `CLD-CHF-2` | `exoscale` | 0 / 1 |
| [`database_encryption_at_rest_enabled`](database_encryption_at_rest_enabled.fr.md) Base de données managée sans chiffrement au repos | high | `CLD-CHF-2` | `scaleway` | 0 / 1 |
| [`k8s_secrets_external_manager`](k8s_secrets_external_manager.fr.md) Secrets Kubernetes sans coffre externe | high | `CLD-K8S-10` | `kubernetes` | 0 / 1 |
| [`loadbalancer_http_redirect_to_https`](loadbalancer_http_redirect_to_https.fr.md) Listener HTTP sans redirection HTTPS | medium | `CLD-CHF-1` | _dormant_ | aucun |
| [`loadbalancer_ssl_listeners`](loadbalancer_ssl_listeners.fr.md) Chiffrement en transit absent | high | `CLD-CHF-1` | `outscale` | 0 / 1 |
| [`objectstorage_bucket_default_encryption`](objectstorage_bucket_default_encryption.fr.md) Bucket sans chiffrement par défaut au repos | high | `CLD-CHF-2` | `outscale` | 0 / 1 |
| [`objectstorage_bucket_kms_encryption`](objectstorage_bucket_kms_encryption.fr.md) Clé de chiffrement gérée par le client absente (BYOK) sur un bucket sensible | medium | `CLD-CHF-4` | `scaleway` | 0 / 1 |

## `journalisation`

| Contrôle | Sévérité | SCSL | Actif pour | Preuves |
|---|---|---|---|---|
| [`kubernetes_cluster_audit_logging_enabled`](kubernetes_cluster_audit_logging_enabled.fr.md) Journalisation d'audit du cluster Kubernetes désactivée | medium | `CLD-LOG-1` | `exoscale` | 1 / 1 |
| [`loadbalancer_logging_enabled`](loadbalancer_logging_enabled.fr.md) Journalisation des accès désactivée | medium | `CLD-LOG-1` | `outscale` | 0 / 1 |

## `gouvernance`

| Contrôle | Sévérité | SCSL | Actif pour | Preuves |
|---|---|---|---|---|
| [`governance_provider_sovereignty`](governance_provider_sovereignty.fr.md) Souveraineté du fournisseur non établie | high | `CLD-GVN-4` | `exoscale`, `outscale`, `scaleway` | 0 / 3 |
| [`governance_resource_region_in_eu`](governance_resource_region_in_eu.fr.md) Ressource hébergée hors Union européenne | high | `CLD-GVN-3` | `exoscale`, `outscale`, `scaleway` | 0 / 3 |
| [`governance_resource_required_tags`](governance_resource_required_tags.fr.md) Inventaire et étiquetage incomplets | medium | `CLD-GVN-1` | `exoscale`, `outscale`, `scaleway` | 0 / 3 |

## Contrôles dormants

Ces contrôles existent au référentiel mais ne sont déclarés pour aucun fournisseur :
aucun scan ne les évalue aujourd'hui. Ils apparaissent ici pour que le catalogue ne se
lise pas comme une couverture.

| Contrôle | Sévérité | Famille |
|---|---|---|
| [`loadbalancer_http_redirect_to_https`](loadbalancer_http_redirect_to_https.fr.md) Listener HTTP sans redirection HTTPS | medium | `chiffrement` |

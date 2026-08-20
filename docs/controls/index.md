> 🇬🇧 English · [🇫🇷 Français](index.fr.md)

<!-- GENERATED PAGE — do not edit by hand. Regenerate with `mise run gen-docs`. -->

# Control catalogue

One page per control, computed from `referentiel/controles.yaml` (the source of truth),
the `providers/*.yaml` descriptors and the `pass` lock in `internal/assess`.
No metadata is copied here: adding a control, changing a severity or removing a provider
rewrites these pages, and CI rejects documentation that lags behind.

This catalogue states what Pépin **can conclude**, not the result of any scan. For the
overview per provider and per source, see the [coverage matrix](../coverage.md).

## Figures

| Figure | Count |
|---|---:|
| Controls in the reference | 57 |
| Active controls | 56 |
| Dormant controls | 1 |
| `critical` | 10 |
| `high` | 32 |
| `medium` | 13 |
| `low` | 2 |
| Deployable remediation proofs | 26 / 95 |

## How to read this catalogue

- **Active control**: declared for at least one provider (non-empty `fournisseurs:`).
  A scan of that provider can give it a status.
- **Dormant control**: written and tested, declared for no provider. It is never evaluated,
  and it counts towards no coverage. The full list is at the bottom of this page.
- **Remediation proofs**: the deployable setups present under
  [`references/remediation/`](../../references/remediation/README.md). The count is over
  declared (control, provider) pairs, and it is partial today: the *textual* remediation,
  on the other hand, is carried by every deviation reported.
- Controls still being triaged, not retained in the active reference, live in the
  [triage catalogue](../../referentiel/catalogue.yaml).

## `iam`

| Control | Severity | SCSL | Active for | Proofs |
|---|---|---|---|---|
| [`iam_accesskey_expiration_set`](iam_accesskey_expiration_set.md) Access key without expiry or rotation | critical | `CLD-IAM-2` | `outscale`, `scaleway` | 0 / 2 |
| [`iam_account_mfa_enforced`](iam_account_mfa_enforced.md) MFA not enforced at the account level | high | `CLD-IAM-3` | `outscale` | 0 / 1 |
| [`iam_apiaccesspolicy_max_key_expiration`](iam_apiaccesspolicy_max_key_expiration.md) API access policy without a maximum key expiry | medium | `CLD-IAM-2` | `outscale` | 0 / 1 |
| [`iam_apiaccessrule_defined`](iam_apiaccessrule_defined.md) No API access rule defined | high | `CLD-IAM-4` | `outscale` | 0 / 1 |
| [`iam_apiaccessrule_no_public_cidr`](iam_apiaccessrule_no_public_cidr.md) API access rule open to a public CIDR | high | `CLD-IAM-4` | `outscale` | 0 / 1 |
| [`iam_no_root_access_key`](iam_no_root_access_key.md) Access key attached to the root account | high | `CLD-IAM-1` | `outscale`, `scaleway` | 0 / 2 |
| [`iam_policy_no_administrative_privileges`](iam_policy_no_administrative_privileges.md) IAM policy with administrative privileges (wildcard action) | critical | `CLD-IAM-1` | `outscale` | 0 / 1 |
| [`iam_policy_no_notaction_notresource`](iam_policy_no_notaction_notresource.md) IAM policy using NotAction/NotResource (anti-pattern) | critical | `CLD-IAM-1` | `outscale` | 0 / 1 |
| [`iam_policy_no_privilege_escalation`](iam_policy_no_privilege_escalation.md) IAM policy allowing privilege escalation | high | `CLD-IAM-12` | `outscale`, `scaleway` | 0 / 2 |
| [`iam_policy_no_wildcard_resource`](iam_policy_no_wildcard_resource.md) IAM policy with a wildcard resource | high | `CLD-IAM-1` | `outscale` | 0 / 1 |
| [`iam_role_key_lifetime_bounded`](iam_role_key_lifetime_bounded.md) IAM role without a bounded credential lifetime | critical | `CLD-IAM-2` | `exoscale` | 1 / 1 |
| [`iam_role_no_admin_privileges`](iam_role_no_admin_privileges.md) IAM role with excessive privileges | high | `CLD-IAM-1` | `exoscale` | 1 / 1 |
| [`iam_role_source_ip_restricted`](iam_role_source_ip_restricted.md) IAM role without a source IP restriction | high | `CLD-IAM-4` | `exoscale` | 1 / 1 |
| [`iam_user_mfa_enabled`](iam_user_mfa_enabled.md) MFA not enabled on an account | high | `CLD-IAM-3` | `exoscale`, `scaleway` | 1 / 2 |

## `reseau`

| Control | Severity | SCSL | Active for | Proofs |
|---|---|---|---|---|
| [`compute_instance_public_ip_with_open_securitygroup`](compute_instance_public_ip_with_open_securitygroup.md) Instance publicly exposed without restrictive filtering | critical | `CLD-NET-3` | `exoscale`, `outscale`, `scaleway` | 1 / 3 |
| [`database_service_not_open_to_internet`](database_service_not_open_to_internet.md) Managed database reachable from the internet | high | `CLD-NET-1` | `scaleway` | 0 / 1 |
| [`k8s_namespace_network_policy_defined`](k8s_namespace_network_policy_defined.md) Namespace without a NetworkPolicy (flat network) | high | `CLD-K8S-6` | `kubernetes` | 0 / 1 |
| [`network_documented`](network_documented.md) Network without mapping tags | low | `CLD-NET-5` | `exoscale`, `outscale`, `scaleway` | 1 / 3 |
| [`network_flow_matrix_documented`](network_flow_matrix_documented.md) Inbound flow without a justification in the flow matrix | medium | `CLD-NET-5` | `exoscale` | 1 / 1 |
| [`network_peering_cross_organization`](network_peering_cross_organization.md) Network peering towards another information system | high | `CLD-NET-7` | `outscale` | 0 / 1 |
| [`network_securitygroup_allow_ingress_from_internet_to_all_ports`](network_securitygroup_allow_ingress_from_internet_to_all_ports.md) All inbound traffic allowed from the internet (any/any) | critical | `CLD-NET-2` | `exoscale`, `outscale`, `scaleway` | 1 / 3 |
| [`network_securitygroup_allow_ingress_from_internet_to_high_risk_tcp_ports`](network_securitygroup_allow_ingress_from_internet_to_high_risk_tcp_ports.md) Sensitive port (database, directory…) open to the internet | high | `CLD-NET-1` | `exoscale`, `outscale`, `scaleway` | 1 / 3 |
| [`network_securitygroup_allow_ingress_from_internet_to_high_risk_udp_ports`](network_securitygroup_allow_ingress_from_internet_to_high_risk_udp_ports.md) Sensitive UDP service (amplification, unauthenticated) open to the internet | high | `CLD-NET-1` | `exoscale`, `outscale`, `scaleway` | 1 / 3 |
| [`network_securitygroup_allow_ingress_from_internet_to_tcp_port_22`](network_securitygroup_allow_ingress_from_internet_to_tcp_port_22.md) SSH (port 22) open to the internet | high | `CLD-NET-1`, `CLD-IAM-6`, `CLD-NET-6` | `exoscale`, `outscale`, `scaleway` | 1 / 3 |
| [`network_securitygroup_allow_ingress_from_internet_to_tcp_port_3389`](network_securitygroup_allow_ingress_from_internet_to_tcp_port_3389.md) RDP (port 3389) open to the internet | high | `CLD-NET-1`, `CLD-IAM-6`, `CLD-NET-6` | `exoscale`, `outscale`, `scaleway` | 1 / 3 |
| [`network_securitygroup_default_deny`](network_securitygroup_default_deny.md) Security group inbound default policy set to "accept" | high | `CLD-NET-2` | `scaleway` | 0 / 1 |
| [`network_securitygroup_default_restrict_traffic`](network_securitygroup_default_restrict_traffic.md) The "default" security group is not restrictive | high | `CLD-NET-4` | `outscale` | 0 / 1 |
| [`network_securitygroup_unrestricted_egress`](network_securitygroup_unrestricted_egress.md) Unrestricted outbound filtering | medium | `CLD-NET-4` | `exoscale`, `outscale`, `scaleway` | 1 / 3 |
| [`network_subnet_no_public_ip_by_default`](network_subnet_no_public_ip_by_default.md) Subnet assigning a public IP by default | medium | `CLD-NET-3` | `outscale` | 0 / 1 |

## `compute`

| Control | Severity | SCSL | Active for | Proofs |
|---|---|---|---|---|
| [`compute_instance_deletion_protection`](compute_instance_deletion_protection.md) Instance without deletion protection | medium | `CLD-CMP-10` | `outscale` | 0 / 1 |
| [`compute_instance_has_security_group`](compute_instance_has_security_group.md) Instance without network filtering | critical | `CLD-CMP-1` | `exoscale`, `outscale`, `scaleway` | 1 / 3 |
| [`compute_instance_no_secrets_in_user_data`](compute_instance_no_secrets_in_user_data.md) Cleartext secret in the user data (user-data) | high | `CLD-CMP-9` | `exoscale`, `outscale`, `scaleway` | 1 / 3 |
| [`k8s_namespace_pod_security_enforced`](k8s_namespace_pod_security_enforced.md) Pod Security Standards not enforced on a namespace | high | `CLD-K8S-5` | `kubernetes` | 0 / 1 |
| [`k8s_rbac_no_cluster_admin_binding`](k8s_rbac_no_cluster_admin_binding.md) cluster-admin granted beyond the native binding | critical | `CLD-K8S-4` | `kubernetes` | 0 / 1 |
| [`kubernetes_cluster_auto_upgrade_enabled`](kubernetes_cluster_auto_upgrade_enabled.md) Automatic Kubernetes cluster upgrades disabled | medium | `CLD-K8S-3` | `exoscale`, `outscale` | 1 / 2 |
| [`kubernetes_cluster_control_plane_highly_available`](kubernetes_cluster_control_plane_highly_available.md) Kubernetes control plane not highly available | high | `CLD-K8S-2` | `exoscale`, `outscale` | 1 / 2 |
| [`kubernetes_cluster_deletion_protection`](kubernetes_cluster_deletion_protection.md) Kubernetes cluster without deletion protection | medium | `CLD-K8S-3` | `outscale` | 0 / 1 |
| [`kubernetes_cluster_not_publicly_accessible`](kubernetes_cluster_not_publicly_accessible.md) Kubernetes API server exposed to the internet | critical | `CLD-K8S-1` | `outscale` | 0 / 1 |

## `stockage`

| Control | Severity | SCSL | Active for | Proofs |
|---|---|---|---|---|
| [`blockstorage_snapshot_not_public`](blockstorage_snapshot_not_public.md) Snapshot or image shared publicly | high | `CLD-STO-2` | `outscale` | 0 / 1 |
| [`blockstorage_volume_snapshots_exist`](blockstorage_volume_snapshots_exist.md) No recent, completed snapshot | high | `CLD-STO-3` | `exoscale`, `outscale` | 1 / 2 |
| [`compute_image_not_public`](compute_image_not_public.md) Machine image shared publicly | high | `CLD-STO-2` | `outscale` | 0 / 1 |
| [`database_backup_enabled`](database_backup_enabled.md) Automatic backups disabled on a managed database | high | `CLD-STO-3` | `scaleway` | 0 / 1 |
| [`objectstorage_bucket_object_lock_enabled`](objectstorage_bucket_object_lock_enabled.md) Object Lock (immutability) disabled on object storage | low | `CLD-STO-8` | `exoscale`, `outscale`, `scaleway` | 1 / 3 |
| [`objectstorage_bucket_public_access`](objectstorage_bucket_public_access.md) Object storage publicly exposed | critical | `CLD-STO-1` | `exoscale`, `outscale`, `scaleway` | 1 / 3 |
| [`objectstorage_bucket_versioning_enabled`](objectstorage_bucket_versioning_enabled.md) Object storage versioning disabled | medium | `CLD-STO-4` | `exoscale`, `outscale`, `scaleway` | 1 / 3 |

## `chiffrement`

| Control | Severity | SCSL | Active for | Proofs |
|---|---|---|---|---|
| [`blockstorage_volume_encryption`](blockstorage_volume_encryption.md) Encryption at rest disabled | high | `CLD-CHF-2` | `exoscale` | 1 / 1 |
| [`database_encryption_at_rest_enabled`](database_encryption_at_rest_enabled.md) Managed database without encryption at rest | high | `CLD-CHF-2` | `scaleway` | 0 / 1 |
| [`k8s_secrets_external_manager`](k8s_secrets_external_manager.md) Kubernetes secrets without an external vault | high | `CLD-K8S-10` | `kubernetes` | 0 / 1 |
| [`loadbalancer_http_redirect_to_https`](loadbalancer_http_redirect_to_https.md) HTTP listener without an HTTPS redirect | medium | `CLD-CHF-1` | _dormant_ | — |
| [`loadbalancer_ssl_listeners`](loadbalancer_ssl_listeners.md) No encryption in transit | high | `CLD-CHF-1` | `outscale` | 0 / 1 |
| [`objectstorage_bucket_default_encryption`](objectstorage_bucket_default_encryption.md) Bucket without default encryption at rest | high | `CLD-CHF-2` | `outscale` | 0 / 1 |
| [`objectstorage_bucket_kms_encryption`](objectstorage_bucket_kms_encryption.md) No customer-managed encryption key (BYOK) on a sensitive bucket | medium | `CLD-CHF-4` | `scaleway` | 0 / 1 |

## `journalisation`

| Control | Severity | SCSL | Active for | Proofs |
|---|---|---|---|---|
| [`kubernetes_cluster_audit_logging_enabled`](kubernetes_cluster_audit_logging_enabled.md) Kubernetes cluster audit logging disabled | medium | `CLD-LOG-1` | `exoscale` | 1 / 1 |
| [`loadbalancer_logging_enabled`](loadbalancer_logging_enabled.md) Access logging disabled | medium | `CLD-LOG-1` | `outscale` | 0 / 1 |

## `gouvernance`

| Control | Severity | SCSL | Active for | Proofs |
|---|---|---|---|---|
| [`governance_provider_sovereignty`](governance_provider_sovereignty.md) Provider sovereignty not established | high | `CLD-GVN-4` | `exoscale`, `outscale`, `scaleway` | 1 / 3 |
| [`governance_resource_region_in_eu`](governance_resource_region_in_eu.md) Resource hosted outside the European Union | high | `CLD-GVN-3` | `exoscale`, `outscale`, `scaleway` | 1 / 3 |
| [`governance_resource_required_tags`](governance_resource_required_tags.md) Incomplete inventory and tagging | medium | `CLD-GVN-1` | `exoscale`, `outscale`, `scaleway` | 1 / 3 |

## Dormant controls

These controls exist in the reference but are declared for no provider: no scan evaluates
them today. They appear here so that the catalogue is not read as coverage.

| Control | Severity | Family |
|---|---|---|
| [`loadbalancer_http_redirect_to_https`](loadbalancer_http_redirect_to_https.md) HTTP listener without an HTTPS redirect | medium | `chiffrement` |

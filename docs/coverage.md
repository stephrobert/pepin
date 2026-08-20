> 🇬🇧 English · [🇫🇷 Français](coverage.fr.md)

<!-- GENERATED PAGE — do not edit by hand. Regenerate with `mise run gen-docs`. -->

# Coverage matrix

What Pépin actually measures, per control, per provider and per source.
This page is **computed** from `referentiel/controles.yaml` (codes, severities, `fournisseurs`),
`providers/*.yaml` (live collection spec, Terraform mapping, API contract) and the `pass` lock in
`internal/assess`. It describes what Pépin **can conclude**, not the result of any given scan: a
control marked ✅ may well return `not-evaluated` on an inventory that contains no resource of the
type it reads.

**What the "live" column is, and what it is not.** It is DERIVED from the descriptors: it
says which resource type the `collecte` spec declares it produces, not what a provider
actually answered. Two things are now MEASURED rather than merely declared: that the announced
endpoints really go out on the wire, and that parent/child joins fire. That comes from a
recording of real calls, replayed on every build (`internal/genprovider/testdata/transcripts/`,
see [Tracing real API calls](guides/tracing-api-calls.md)). That recording was taken against a
LOCAL EMULATOR: it proves what Pépin does, never what the provider answers. The field names and
types of the native contract, the real pagination bounds and the behaviour on a permission
refusal remain UNOBSERVED; they are owed to a real scan, which this repository does not run
because it holds no cloud credentials.

Control titles and justifications come from the reference and from the provider contracts, which
are bilingual: this page quotes their English version, the French page quotes the French one.
French remains the reference language of the normative content.

## Legend

| Mark | Status | What Pépin can conclude |
|---|---|---|
| ✅ | `supported` | the source produces the resource type the control reads, the contract marks it `verifie`, and the deciding attribute is projected. Pépin can return `pass` or `fail`. |
| ◐ | `partial` | the source produces the type, but the `pass` lock cannot be lifted (contract not `verifie`, or the deciding attribute is not projected by this source). The scan returns `not-evaluated` — never a silent green. |
| ∅ | `not-applicable` | the provider contract declares the control untestable, with its justification (no such mechanism in the API, or the resource type does not exist). The scan returns `not-applicable` together with that reason. |
| ✗ | `unsupported` | the control is not declared for this provider in the reference, or this source produces no resource of the type it reads. Nothing will be concluded from this source. |

**◐ does not mean "half compliant".** It means "Pépin cannot decide from this source",
and the report says so on every scan, control by control, with the reason.

## Summary by reference family

| Family | Controls | exoscale | outscale | scaleway |
|---|---:|---|---|---|
| `chiffrement` | 7 | ✅ 1 · ◐ 0 · ∅ 3 · ✗ 3 | ✅ 2 · ◐ 0 · ∅ 3 · ✗ 2 | ✅ 2 · ◐ 0 · ∅ 1 · ✗ 4 |
| `compute` | 9 | ✅ 4 · ◐ 0 · ∅ 0 · ✗ 5 | ✅ 7 · ◐ 0 · ∅ 0 · ✗ 2 | ✅ 2 · ◐ 0 · ∅ 0 · ✗ 7 |
| `gouvernance` | 3 | ✅ 2 · ◐ 1 · ∅ 0 · ✗ 0 | ✅ 2 · ◐ 1 · ∅ 0 · ✗ 0 | ✅ 2 · ◐ 1 · ∅ 0 · ✗ 0 |
| `iam` | 14 | ✅ 4 · ◐ 0 · ∅ 0 · ✗ 10 | ✅ 10 · ◐ 0 · ∅ 1 · ✗ 3 | ✅ 3 · ◐ 1 · ∅ 0 · ✗ 10 |
| `journalisation` | 2 | ✅ 1 · ◐ 0 · ∅ 1 · ✗ 0 | ✅ 1 · ◐ 0 · ∅ 0 · ✗ 1 | ✅ 0 · ◐ 0 · ∅ 0 · ✗ 2 |
| `reseau` | 15 | ✅ 9 · ◐ 0 · ∅ 0 · ✗ 6 | ✅ 11 · ◐ 0 · ∅ 0 · ✗ 4 | ✅ 9 · ◐ 1 · ∅ 0 · ✗ 5 |
| `stockage` | 7 | ✅ 4 · ◐ 0 · ∅ 1 · ✗ 2 | ✅ 6 · ◐ 0 · ∅ 0 · ✗ 1 | ✅ 4 · ◐ 0 · ∅ 1 · ✗ 2 |

## Full matrix (control × provider × source)

| Control | Severity | SCSL | exoscale TF | exoscale live | outscale TF | outscale live | scaleway TF | scaleway live |
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

## Why a cell is not ✅

Every cell that is not ✅ **while the control is declared for that provider** carries its
reason. Cells that merely say "control not declared for this provider" are left out: the matrix
already shows them, and they add nothing.

| Control | Provider | Source | Status | Reason |
|---|---|---|---|---|
| `blockstorage_snapshot_not_public` | exoscale | terraform | ∅ `not-applicable` | Exoscale block-storage snapshots cannot be exported or shared (official documentation): the risk of public exposure is structurally absent, compliant by construction (STO-2). |
| `blockstorage_snapshot_not_public` | exoscale | live | ∅ `not-applicable` | Exoscale block-storage snapshots cannot be exported or shared (official documentation): the risk of public exposure is structurally absent, compliant by construction (STO-2). |
| `blockstorage_snapshot_not_public` | outscale | terraform | ✗ `unsupported` | this source produces no resource of type "blockstorage_snapshot" |
| `blockstorage_snapshot_not_public` | scaleway | terraform | ∅ `not-applicable` | Scaleway block snapshots (api/block/v1) expose no sharing or public export mechanism: the risk of public exposure is structurally absent, compliant by construction (STO-2). |
| `blockstorage_snapshot_not_public` | scaleway | live | ∅ `not-applicable` | Scaleway block snapshots (api/block/v1) expose no sharing or public export mechanism: the risk of public exposure is structurally absent, compliant by construction (STO-2). |
| `blockstorage_volume_encryption` | outscale | terraform | ∅ `not-applicable` | osc-sdk-go/v2 Volume exposes no encryption field; encryption at rest is guest-side (EncFS/LUKS), a customer responsibility, hence unobservable on the platform side (CHF-2). |
| `blockstorage_volume_encryption` | outscale | live | ∅ `not-applicable` | osc-sdk-go/v2 Volume exposes no encryption field; encryption at rest is guest-side (EncFS/LUKS), a customer responsibility, hence unobservable on the platform side (CHF-2). |
| `blockstorage_volume_encryption` | scaleway | terraform | ∅ `not-applicable` | Encryption at rest of block volumes is guest-side (LUKS/Cryptsetup), a customer responsibility (shared responsibility model); the block API exposes no encryption field, hence unobservable on the platform side (CHF-2). |
| `blockstorage_volume_encryption` | scaleway | live | ∅ `not-applicable` | Encryption at rest of block volumes is guest-side (LUKS/Cryptsetup), a customer responsibility (shared responsibility model); the block API exposes no encryption field, hence unobservable on the platform side (CHF-2). |
| `blockstorage_volume_snapshots_exist` | outscale | terraform | ✗ `unsupported` | this source produces no resource of type "blockstorage_volume" |
| `compute_instance_deletion_protection` | outscale | terraform | ◐ `partial` | deciding attribute "deletion_protection" not projected by this source: a capability guard, so the scan returns "not-evaluated" |
| `compute_instance_no_secrets_in_user_data` | scaleway | live | ◐ `partial` | deciding attribute "user_data" not projected by this source: a capability guard, so the scan returns "not-evaluated" |
| `compute_instance_public_ip_with_open_securitygroup` | scaleway | terraform | ◐ `partial` | deciding attribute "public_ip" not projected by this source: a capability guard, so the scan returns "not-evaluated" |
| `database_backup_enabled` | scaleway | live | ✗ `unsupported` | this source produces no resource of type "managed_database" |
| `database_encryption_at_rest_enabled` | scaleway | live | ✗ `unsupported` | this source produces no resource of type "managed_database" |
| `database_service_not_open_to_internet` | scaleway | live | ✗ `unsupported` | this source produces no resource of type "managed_database" |
| `governance_resource_region_in_eu` | outscale | terraform | ◐ `partial` | deciding attribute "region" not projected by this source: a capability guard, so the scan returns "not-evaluated" |
| `governance_resource_required_tags` | exoscale | terraform | ◐ `partial` | no targeted resource type, and the control does not read the provider descriptor: the "pass" lock cannot be lifted, so the scan returns "not-evaluated" as long as no deviation is detected |
| `governance_resource_required_tags` | exoscale | live | ◐ `partial` | no targeted resource type, and the control does not read the provider descriptor: the "pass" lock cannot be lifted, so the scan returns "not-evaluated" as long as no deviation is detected |
| `governance_resource_required_tags` | outscale | terraform | ◐ `partial` | no targeted resource type, and the control does not read the provider descriptor: the "pass" lock cannot be lifted, so the scan returns "not-evaluated" as long as no deviation is detected |
| `governance_resource_required_tags` | outscale | live | ◐ `partial` | no targeted resource type, and the control does not read the provider descriptor: the "pass" lock cannot be lifted, so the scan returns "not-evaluated" as long as no deviation is detected |
| `governance_resource_required_tags` | scaleway | terraform | ◐ `partial` | no targeted resource type, and the control does not read the provider descriptor: the "pass" lock cannot be lifted, so the scan returns "not-evaluated" as long as no deviation is detected |
| `governance_resource_required_tags` | scaleway | live | ◐ `partial` | no targeted resource type, and the control does not read the provider descriptor: the "pass" lock cannot be lifted, so the scan returns "not-evaluated" as long as no deviation is detected |
| `iam_accesskey_expiration_set` | outscale | terraform | ✗ `unsupported` | this source produces no resource of type "access_key" |
| `iam_account_mfa_enforced` | outscale | terraform | ✗ `unsupported` | this source produces no resource of type "api_access_policy" |
| `iam_apiaccesspolicy_max_key_expiration` | outscale | terraform | ✗ `unsupported` | this source produces no resource of type "api_access_policy" |
| `iam_apiaccessrule_defined` | outscale | terraform | ✗ `unsupported` | this source produces no resource of type "api_access_summary" |
| `iam_apiaccessrule_no_public_cidr` | outscale | terraform | ✗ `unsupported` | this source produces no resource of type "api_access_rule" |
| `iam_no_root_access_key` | outscale | terraform | ✗ `unsupported` | this source produces no resource of type "access_key" |
| `iam_no_root_access_key` | scaleway | terraform | ◐ `partial` | deciding attribute "root_owned / scope" not projected by this source: a capability guard, so the scan returns "not-evaluated" |
| `iam_no_root_access_key` | scaleway | live | ◐ `partial` | deciding attribute "root_owned / scope" not projected by this source: a capability guard, so the scan returns "not-evaluated" |
| `iam_policy_no_privilege_escalation` | scaleway | live | ✗ `unsupported` | this source produces no resource of type "iam_policy" |
| `iam_user_mfa_enabled` | exoscale | terraform | ✗ `unsupported` | this source produces no resource of type "iam_user" |
| `iam_user_mfa_enabled` | outscale | terraform | ∅ `not-applicable` | resource type "iam_user" absent from the outscale API |
| `iam_user_mfa_enabled` | outscale | live | ∅ `not-applicable` | resource type "iam_user" absent from the outscale API |
| `iam_user_mfa_enabled` | scaleway | terraform | ✗ `unsupported` | this source produces no resource of type "iam_user" |
| `kubernetes_cluster_auto_upgrade_enabled` | outscale | terraform | ✗ `unsupported` | this source produces no resource of type "kubernetes_cluster" |
| `kubernetes_cluster_control_plane_highly_available` | outscale | terraform | ✗ `unsupported` | this source produces no resource of type "kubernetes_cluster" |
| `kubernetes_cluster_deletion_protection` | outscale | terraform | ✗ `unsupported` | this source produces no resource of type "kubernetes_cluster" |
| `kubernetes_cluster_not_publicly_accessible` | outscale | terraform | ✗ `unsupported` | this source produces no resource of type "kubernetes_cluster" |
| `loadbalancer_http_redirect_to_https` | exoscale | terraform | ∅ `not-applicable` | resource type "load_balancer" absent from the exoscale API |
| `loadbalancer_http_redirect_to_https` | exoscale | live | ∅ `not-applicable` | resource type "load_balancer" absent from the exoscale API |
| `loadbalancer_http_redirect_to_https` | outscale | terraform | ∅ `not-applicable` | The Outscale LBU cannot redirect: `ListenerRule.Action` is documented as "always forward" in the OAPI contract (no redirect action), and no redirect attribute exists on `Listener`. The mechanism does not exist, so the control is not applicable (CHF-1). |
| `loadbalancer_http_redirect_to_https` | outscale | live | ∅ `not-applicable` | The Outscale LBU cannot redirect: `ListenerRule.Action` is documented as "always forward" in the OAPI contract (no redirect action), and no redirect attribute exists on `Listener`. The mechanism does not exist, so the control is not applicable (CHF-1). |
| `loadbalancer_logging_enabled` | exoscale | terraform | ∅ `not-applicable` | resource type "load_balancer" absent from the exoscale API |
| `loadbalancer_logging_enabled` | exoscale | live | ∅ `not-applicable` | resource type "load_balancer" absent from the exoscale API |
| `loadbalancer_logging_enabled` | outscale | terraform | ✗ `unsupported` | this source produces no resource of type "load_balancer" |
| `loadbalancer_ssl_listeners` | exoscale | terraform | ∅ `not-applicable` | resource type "load_balancer" absent from the exoscale API |
| `loadbalancer_ssl_listeners` | exoscale | live | ∅ `not-applicable` | resource type "load_balancer" absent from the exoscale API |
| `loadbalancer_ssl_listeners` | outscale | terraform | ✗ `unsupported` | this source produces no resource of type "load_balancer" |
| `network_documented` | scaleway | terraform | ◐ `partial` | provider contract: type "network" is not declared `verifie` (state: a_verifier) |
| `network_documented` | scaleway | live | ✗ `unsupported` | this source produces no resource of type "network" |
| `network_peering_cross_organization` | outscale | terraform | ✗ `unsupported` | this source produces no resource of type "network_peering" |
| `network_securitygroup_default_deny` | scaleway | live | ✗ `unsupported` | this source produces no resource of type "security_group" |
| `network_securitygroup_default_restrict_traffic` | outscale | terraform | ◐ `partial` | deciding attribute "security_group_name" not projected by this source: a capability guard, so the scan returns "not-evaluated" |
| `objectstorage_bucket_default_encryption` | outscale | terraform | ✗ `unsupported` | this source produces no resource of type "object_storage_bucket" |
| `objectstorage_bucket_kms_encryption` | exoscale | terraform | ∅ `not-applicable` | SOS encrypts at rest by default (SSE-SOS, keys managed by Exoscale, SSE-S3 style) but exposes no customer-managed BYOK/KMS key at the bucket level (SSE-C stays per-object and unobservable), so the BYOK-at-bucket control is moot (CHF-4). |
| `objectstorage_bucket_kms_encryption` | exoscale | live | ∅ `not-applicable` | SOS encrypts at rest by default (SSE-SOS, keys managed by Exoscale, SSE-S3 style) but exposes no customer-managed BYOK/KMS key at the bucket level (SSE-C stays per-object and unobservable), so the BYOK-at-bucket control is moot (CHF-4). |
| `objectstorage_bucket_kms_encryption` | outscale | terraform | ∅ `not-applicable` | OOS encrypts server-side in AES256 with a PROVIDER key; there is neither a KMS service nor a customer-managed master key, so there is no BYOK to audit at the bucket level (CHF-4). Note: enabling SSE itself is opt-in and observable, which is a separate control, not this N/A. |
| `objectstorage_bucket_kms_encryption` | outscale | live | ∅ `not-applicable` | OOS encrypts server-side in AES256 with a PROVIDER key; there is neither a KMS service nor a customer-managed master key, so there is no BYOK to audit at the bucket level (CHF-4). Note: enabling SSE itself is opt-in and observable, which is a separate control, not this N/A. |
| `objectstorage_bucket_kms_encryption` | scaleway | terraform | ◐ `partial` | deciding attribute "sse_kms_enabled" not projected by this source: a capability guard, so the scan returns "not-evaluated" |
| `objectstorage_bucket_object_lock_enabled` | exoscale | terraform | ✗ `unsupported` | this source produces no resource of type "object_storage_bucket" |
| `objectstorage_bucket_object_lock_enabled` | outscale | terraform | ✗ `unsupported` | this source produces no resource of type "object_storage_bucket" |
| `objectstorage_bucket_public_access` | exoscale | terraform | ✗ `unsupported` | this source produces no resource of type "object_storage_bucket" |
| `objectstorage_bucket_public_access` | outscale | terraform | ✗ `unsupported` | this source produces no resource of type "object_storage_bucket" |
| `objectstorage_bucket_versioning_enabled` | exoscale | terraform | ✗ `unsupported` | this source produces no resource of type "object_storage_bucket" |
| `objectstorage_bucket_versioning_enabled` | outscale | terraform | ✗ `unsupported` | this source produces no resource of type "object_storage_bucket" |
| `objectstorage_bucket_versioning_enabled` | scaleway | terraform | ◐ `partial` | deciding attribute "versioning" not projected by this source: a capability guard, so the scan returns "not-evaluated" |

## Different-scope provider: `kubernetes` (in-cluster)

This provider audits the state **inside** a cluster (RBAC, Pod Security, NetworkPolicy), not a
cloud control plane. Comparing it with a cloud for parity would be meaningless: neither can cover
the other's scope. One source only: live collection through a kubeconfig.

| Control | Severity | kubernetes |
|---|---|:-:|
| `k8s_namespace_network_policy_defined` | high | ✅ |
| `k8s_namespace_pod_security_enforced` | high | ✅ |
| `k8s_rbac_no_cluster_admin_binding` | critical | ✅ |
| `k8s_secrets_external_manager` | high | ✅ |

## Totals

| Provider | Source | ✅ `supported` | ◐ `partial` | ∅ `not-applicable` | ✗ `unsupported` |
|---|---|---:|---:|---:|---:|
| exoscale | terraform | 21 | 1 | 5 | 30 |
| exoscale | live | 25 | 1 | 5 | 26 |
| outscale | terraform | 17 | 4 | 4 | 32 |
| outscale | live | 39 | 1 | 4 | 13 |
| scaleway | terraform | 18 | 6 | 2 | 31 |
| scaleway | live | 16 | 3 | 2 | 36 |
| kubernetes | live | 4 | 0 | 0 | 53 |

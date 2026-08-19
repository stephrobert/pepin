> 🇬🇧 English · [🇫🇷 Français](outscale.fr.md)

# Outscale

Everything on this page that describes what Pépin collects is derived from
`providers/outscale.yaml`, the descriptor the scanner itself reads. You do not need to open that
file to understand the coverage.

<!-- pepin:gen provider-outscale-identity -->
| Descriptor field | Value |
|---|---|
| Description | Outscale (3DS) — VM, BSU, OOS, EIM, security groups, OKS, LBU |
| Scope | cloud |
| Region key (`--region`) | `region` |
| API authentication | `sigv4` |
| Jurisdiction of the head office | FR |
| Established in the EU | yes |
| Capital control | FR |
| SecNumCloud | `qualifie` |
| Extraterritorial exposure | no |
| Anchoring sources | 3ds.com/newsroom (Outscale 1er cloud SecNumCloud 3.2) ; en.outscale.com/our-certifications |
<!-- /pepin:gen provider-outscale-identity -->

Outscale is the only one of the three whose descriptor records `secnumcloud: qualifie`. That
qualification applies to the **provider**, never to your tenant: a Pépin report says nothing
about it either way ([Scope and non-goals](../concepts/scope.md)).

## Authentication

<!-- pepin:gen provider-outscale-credentials -->
| Logical key | Environment variable | Default |
|---|---|---|
| `access_key` | `OSC_ACCESS_KEY` | — |
| `region` | `OSC_REGION` | `eu-west-2` |
| `secret_key` | `OSC_SECRET_KEY` | — |
| native configuration file | `~/.osc/config.json` | `osc` |
<!-- /pepin:gen provider-outscale-credentials -->

- The OAPI is called with **AWS Signature V4** (service `oapi`), which is a protocol, not a
  kinship: everything else here is Outscale's own vocabulary.
- The same access key and secret key are used for OOS (object storage) and for OKS.
- The native configuration file is read when the variables are absent; `OSC_CONFIG_FILE`
  overrides its location, and `--profile` selects a profile inside it.

```bash
export OSC_ACCESS_KEY=... OSC_SECRET_KEY=...
pepin scan outscale --live --region eu-west-2
```

## What a live scan calls

Every endpoint, including the parent listing of a join. All OAPI calls are `POST`.

<!-- pepin:gen provider-outscale-live -->
| Normalized type | Call | Note |
|---|---|---|
| `access_key` | `POST /ReadAccessKeys` | — |
| `access_key` | `POST /ReadUsers` | parent listing of a join (called first) |
| `api_access_policy` | `POST /ReadApiAccessPolicy` | — |
| `api_access_rule` | `POST /ReadApiAccessRules` | — |
| `api_access_summary` | `POST /ReadApiAccessRules` | — |
| `blockstorage_snapshot` | `POST /ReadAccounts` | parent listing of a join (called first) |
| `blockstorage_snapshot` | `POST /ReadSnapshots` | — |
| `blockstorage_volume` | `POST /ReadVolumes` | — |
| `compute_image` | `POST /ReadAccounts` | parent listing of a join (called first) |
| `compute_image` | `POST /ReadImages` | — |
| `compute_instance` | `POST /ReadVms` | — |
| `iam_policy` | `POST /ReadPolicies` | parent listing of a join (called first) |
| `iam_policy` | `POST /ReadPolicyVersion` | — |
| `load_balancer` | `POST /ReadLoadBalancers` | — |
| `network` | `POST /ReadNets` | — |
| `network_peering` | `POST /ReadNetPeerings` | — |
| `security_group_rule` | `POST /ReadSecurityGroups` | — |
| `subnet` | `POST /ReadSubnets` | — |
| `object_storage_bucket` | `https://oos.{region}.outscale.com` | object storage S3 API (Go collector) |
| `kubernetes_cluster` | `https://api.{region}.oks.outscale.com` | managed Kubernetes API (Go collector) |

Base URL: `https://api.{region}.{host}/api/v1`
<!-- /pepin:gen provider-outscale-live -->

Three collectors are written in Go and therefore do not appear in the descriptor's collection
spec:

- **Inline EIM policies** (`internal/eimpolicy`): `ReadUsers` → `ReadUserPolicies` →
  `ReadUserPolicy`, and `ReadUserGroups` → `ReadUserGroupPolicies`. An inline
  `Action:* / Resource:*` policy grants everything, and reading only managed policies would miss
  it silently.
- **Object storage** (`internal/objectstorage`): `ListBuckets`, then per bucket
  `GetBucketAcl`, `GetBucketVersioning`, `GetBucketPolicy`, `GetBucketTagging`,
  `GetObjectLockConfiguration`, `GetBucketEncryption`.
- **OKS** (`internal/oks`): `GET /api/v2/clusters/all` on a distinct host, authenticated with
  two plain headers rather than SigV4.

`ReadImages` and `ReadSnapshots` are **filtered on the account** (`ReadAccounts` feeds
`Filters.AccountIds`). Without that filter, `ReadImages` returns the whole public catalogue and
every public image of a third party becomes a finding — a false-positive storm the descriptor
exists to prevent.

## Minimal permissions for a live scan

Outscale's vocabulary is an **EIM policy**: `Effect`, `Action` as `<service>:<APIMethod>`, and
`Resource`. Deny by default, explicit deny wins.

The smallest documented policy that covers every OAPI call above is the read-only policy
Outscale publishes itself, described as "an EIM policy that only allows Read calls in the
OUTSCALE API":

```json
{
  "Statement": [
    {
      "Effect": "Allow",
      "Action": ["api:Read*"],
      "Resource": ["*"]
    }
  ]
}
```

Verified in `docs.outscale.com/en/userguide/Managing-Access-for-Cloud-Automation.html` (the
policy) and `EIM-Policy-Elements.html` (the `service:APIMethod` syntax and the service codes
`api`, `ec2`, `elasticloadbalancing`, `iam`, `directconnect`).

Enumerating the actions instead of using `api:Read*` gives this list, which is **inferred from
the documented syntax rather than copied from a source**:
`ReadSecurityGroups`, `ReadVms`, `ReadImages`, `ReadNets`, `ReadNetPeerings`, `ReadSubnets`,
`ReadVolumes`, `ReadSnapshots`, `ReadAccounts`, `ReadUsers`, `ReadAccessKeys`, `ReadPolicies`,
`ReadPolicyVersion`, `ReadUserPolicies`, `ReadUserPolicy`, `ReadUserGroups`,
`ReadUserGroupPolicies`, `ReadApiAccessRules`, `ReadApiAccessPolicy`, `ReadLoadBalancers`.

**What could not be verified, and matters.**

- **Object storage is not governed by EIM.** No object-storage service code appears in the EIM
  policy elements, and the full EIM API reference never mentions OOS. With the account owner's
  keys, OOS access comes with the account; how to grant it to an EIM user is **not verified**.
  Through a bucket policy, the documented read actions are `s3:GetBucketAcl`,
  `s3:GetBucketPolicy`, `s3:GetBucketTagging`, `s3:GetBucketVersioning`,
  `s3:GetEncryptionConfiguration`, `s3:GetBucketObjectLockConfiguration`, `s3:ListBucket`,
  `s3:HeadBucket` — with no action for *listing* the buckets of an account, which is the first
  call Pépin makes.
- **OKS.** The permission model of `GET /api/v2/clusters/all` is **not verified**; the
  documentation states only that OKS is "partially compatible with EIM users", and the two
  documented OKS roles concern volumes and snapshots, not clusters. A refused call degrades
  cleanly: a warning on standard error, and the Kubernetes controls come back `not-evaluated`.
- **Root account keys.** `ReadAccessKeys` without a user name returns the keys of the calling
  identity, so auditing the root account's keys requires account-owner credentials. This is
  recorded in the descriptor as an API limitation; we did not find it stated in the official
  documentation.
- **API access rules.** If the account restricts API access by source IP, the scanner's address
  must be allowed, otherwise the whole collection fails — with exit code 2, not a false green.

## Terraform resources recognized

<!-- pepin:gen provider-outscale-terraform -->
| Terraform resource | Normalized type | Exploded block |
|---|---|---|
| `outscale_image_launch_permission` | `compute_image` | — |
| `outscale_net` | `network` | — |
| `outscale_policy` | `iam_policy` | — |
| `outscale_security_group_rule` | `security_group_rule` | — |
| `outscale_security_group_rule` | `security_group_rule` | `rules[*]` |
| `outscale_subnet` | `subnet` | — |
| `outscale_vm` | `compute_instance` | — |
<!-- /pepin:gen provider-outscale-terraform -->

```bash
terraform plan -out tfplan && terraform show -json tfplan > plan.json
pepin scan outscale --terraform plan.json
```

Watch out for one real type difference between the two sources: a plan renders
`outscale_image_launch_permission`'s `global_permission` as the **string** `"true"`, while the
API returns a boolean. The rules normalize both
([Terraform plan vs live scan](../concepts/terraform-vs-live.md#divergence-2--a-boolean-the-plan-renders-as-a-string)).

## Coverage

<!-- pepin:gen provider-outscale-coverage -->
| Source | ✅ `supported` | ◐ `partial` | ∅ `not-applicable` | ✗ `unsupported` |
|---|---:|---:|---:|---:|
| terraform | 17 | 4 | 4 | 32 |
| live | 39 | 1 | 4 | 13 |
<!-- /pepin:gen provider-outscale-coverage -->

Control by control, with the reason for every cell that is not fully supported, the source of
truth is the [coverage matrix](../coverage.md).

### Declared not applicable

<!-- pepin:gen provider-outscale-na -->
| Control | Justification recorded in the contract |
|---|---|
| `blockstorage_volume_encryption` | osc-sdk-go/v2 Volume exposes no encryption field; encryption at rest is guest-side (EncFS/LUKS), a customer responsibility, hence unobservable on the platform side (CHF-2). |
| `iam_user_mfa_enabled` | resource type "iam_user" absent from the outscale API |
| `loadbalancer_http_redirect_to_https` | The Outscale LBU cannot redirect: `ListenerRule.Action` is documented as "always forward" in the OAPI contract (no redirect action), and no redirect attribute exists on `Listener`. The mechanism does not exist, so the control is not applicable (CHF-1). |
| `objectstorage_bucket_kms_encryption` | OOS encrypts server-side in AES256 with a PROVIDER key; there is neither a KMS service nor a customer-managed master key, so there is no BYOK to audit at the bucket level (CHF-4). Note: enabling SSE itself is opt-in and observable, which is a separate control, not this N/A. |
<!-- /pepin:gen provider-outscale-na -->

### Observable from only one source

<!-- pepin:gen provider-outscale-onesource -->
| Control | Observable only through | Reason on the blind side |
|---|---|---|
| `blockstorage_snapshot_not_public` | live | this source produces no resource of type "blockstorage_snapshot" |
| `blockstorage_volume_snapshots_exist` | live | this source produces no resource of type "blockstorage_volume" |
| `compute_instance_deletion_protection` | live | deciding attribute "deletion_protection" not projected by this source: a capability guard, so the scan returns "not-evaluated" |
| `governance_resource_region_in_eu` | live | deciding attribute "region" not projected by this source: a capability guard, so the scan returns "not-evaluated" |
| `iam_accesskey_expiration_set` | live | this source produces no resource of type "access_key" |
| `iam_account_mfa_enforced` | live | this source produces no resource of type "api_access_policy" |
| `iam_apiaccesspolicy_max_key_expiration` | live | this source produces no resource of type "api_access_policy" |
| `iam_apiaccessrule_defined` | live | this source produces no resource of type "api_access_summary" |
| `iam_apiaccessrule_no_public_cidr` | live | this source produces no resource of type "api_access_rule" |
| `iam_no_root_access_key` | live | this source produces no resource of type "access_key" |
| `kubernetes_cluster_auto_upgrade_enabled` | live | this source produces no resource of type "kubernetes_cluster" |
| `kubernetes_cluster_control_plane_highly_available` | live | this source produces no resource of type "kubernetes_cluster" |
| `kubernetes_cluster_deletion_protection` | live | this source produces no resource of type "kubernetes_cluster" |
| `kubernetes_cluster_not_publicly_accessible` | live | this source produces no resource of type "kubernetes_cluster" |
| `loadbalancer_logging_enabled` | live | this source produces no resource of type "load_balancer" |
| `loadbalancer_ssl_listeners` | live | this source produces no resource of type "load_balancer" |
| `network_peering_cross_organization` | live | this source produces no resource of type "network_peering" |
| `network_securitygroup_default_restrict_traffic` | live | deciding attribute "security_group_name" not projected by this source: a capability guard, so the scan returns "not-evaluated" |
| `objectstorage_bucket_default_encryption` | live | this source produces no resource of type "object_storage_bucket" |
| `objectstorage_bucket_object_lock_enabled` | live | this source produces no resource of type "object_storage_bucket" |
| `objectstorage_bucket_public_access` | live | this source produces no resource of type "object_storage_bucket" |
| `objectstorage_bucket_versioning_enabled` | live | this source produces no resource of type "object_storage_bucket" |
<!-- /pepin:gen provider-outscale-onesource -->

## A real scan

`examples/outscale/terraform/` holds a deliberately misconfigured module. Its plan is committed,
so this runs with no account:

```bash
pepin scan outscale --terraform examples/outscale/terraform/plan.json
```

<!-- pepin:gen provider-outscale-scan -->
```text
[…]
  ├────────────┼──────────────────────────────────────────────────┼──────────┼──────────┼───┤
  │ CLD-IAM-1  │ IAM policy with administrative privileges (wild… │ CRITICAL │ outscale │ 2 │
  │ CLD-IAM-12 │ IAM policy allowing privilege escalation         │ HIGH     │ outscale │ 1 │
  │ CLD-NET-1  │ SSH (port 22) open to the internet               │ HIGH     │ outscale │ 1 │
  │ CLD-STO-2  │ Machine image shared publicly                    │ HIGH     │ outscale │ 1 │
  │ CLD-GVN-1  │ Incomplete inventory and tagging                 │ MEDIUM   │ outscale │ 1 │
  │ CLD-NET-3  │ Subnet assigning a public IP by default          │ MEDIUM   │ outscale │ 1 │
  │ CLD-NET-5  │ Undocumented network (mapping not maintained)    │ LOW      │ outscale │ 1 │
  ╰────────────┴──────────────────────────────────────────────────┴──────────┴──────────┴───╯
──────────────────────────────────────────────────────────────────────────────
 Summary

 Verdict: NON-COMPLIANT

 🔴 CRITICAL 1   🟠 HIGH 4   🟡 MEDIUM 2   🔵 LOW 1
──────────────────────────────────────────────────────────────────────────────
```
<!-- /pepin:gen provider-outscale-scan -->

## Limitations

- **OOS and OKS permissions** are the two places where a strictly least-privilege key is not
  fully documented (see above). Both degrade to `not-evaluated`, never to a false `pass`.
- **Encryption at rest of BSU volumes** is guest-side and unobservable from the API — a declared
  `not-applicable`.
- **The LBU cannot redirect**: the OAPI contract documents `ListenerRule.Action` as always
  forwarding, so the HTTP-to-HTTPS redirect control is not applicable rather than failing.
- The general limits of the tool are in [Known limitations](../known-limitations.md).

## Where to go next

- [Coverage matrix](../coverage.md) — every control, every source.
- [Terraform plan vs live scan](../concepts/terraform-vs-live.md) — choosing the source.
- [Scaleway](scaleway.md) · [Exoscale](exoscale.md) — the other two sovereign clouds.
- [GitHub Actions](../guides/github-actions.md) · [GitLab CI](../guides/gitlab-ci.md).

## How this page stays true

The identity, credentials, endpoint, Terraform-mapping, coverage and not-applicable tables are
computed from `providers/outscale.yaml` and from the reference by `internal/docgen`; the scan
output is a real run of the committed fixture. The Go collectors' calls and the permission
section are written by hand — the descriptor does not carry them — and each states the source in
which it was verified, or says plainly that it was not.

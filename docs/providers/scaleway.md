> 🇬🇧 English · [🇫🇷 Français](scaleway.fr.md)

# Scaleway

Everything on this page that describes what Pépin collects is derived from
`providers/scaleway.yaml`, the descriptor the scanner itself reads. You do not need to open
that file to understand the coverage.

<!-- pepin:gen provider-scaleway-identity -->
| Descriptor field | Value |
|---|---|
| Description | Scaleway — object storage, instances, IAM, security groups |
| Scope | cloud |
| Region key (`--region`) | `region` |
| API authentication | `header` |
| Jurisdiction of the head office | FR |
| Established in the EU | yes |
| Capital control | FR |
| SecNumCloud | `en_cours` |
| Extraterritorial exposure | no |
| Anchoring sources | iliad.fr (entrée processus SecNumCloud) ; en.wikipedia.org/wiki/Scaleway (capital européen) |
<!-- /pepin:gen provider-scaleway-identity -->

The sovereignty fields are not decoration: they feed the governance control `CLD-GVN-4`, and
their sources are recorded in the descriptor. `secnumcloud: en_cours` means the qualification
process is under way, **not** that it is obtained.

## Authentication

Pépin reads the provider's own environment variables. It never invents its own names, and it
never accepts credentials as command-line arguments.

<!-- pepin:gen provider-scaleway-credentials -->
| Logical key | Environment variable | Default |
|---|---|---|
| `access_key` | `SCW_ACCESS_KEY` | — |
| `org` | `SCW_DEFAULT_ORGANIZATION_ID` | — |
| `region` | `SCW_DEFAULT_REGION` | `fr-par` |
| `secret_key` | `SCW_SECRET_KEY` | — |
| `zone` | `SCW_DEFAULT_ZONE` | `{region}-1` |
| native configuration file | `~/.config/scw/config.yaml` | `scw` |
<!-- /pepin:gen provider-scaleway-credentials -->

- The API authenticates with the **secret key** in the `X-Auth-Token` header.
- The same access key and secret key are used for Object Storage (S3).
- `SCW_DEFAULT_ORGANIZATION_ID` is required: it is substituted into the IAM paths, and without
  it the IAM collection cannot run.
- The native configuration file is read when the variables are absent;
  `SCW_CONFIG_PATH` overrides its location.
- `--region` sets the region (and the zone defaults to `{region}-1`).

```bash
export SCW_ACCESS_KEY=... SCW_SECRET_KEY=... SCW_DEFAULT_ORGANIZATION_ID=...
pepin scan scaleway --live --region fr-par
```

## What a live scan calls

Every endpoint, including the parent listing of a join. This is the exact surface a read-only
key has to cover.

<!-- pepin:gen provider-scaleway-live -->
| Normalized type | Call | Note |
|---|---|---|
| `access_key` | `GET /iam/v1alpha1/api-keys?organization_id={org}` | — |
| `compute_instance` | `GET /instance/v1/zones/{zone}/servers` | — |
| `iam_user` | `GET /iam/v1alpha1/users?organization_id={org}` | — |
| `security_group_rule` | `GET /instance/v1/zones/{zone}/security_groups` | parent listing of a join (called first) |
| `security_group_rule` | `GET /instance/v1/zones/{zone}/security_groups/{sg.id}/rules` | — |
| `object_storage_bucket` | `https://s3.{region}.scw.cloud` | object storage S3 API (Go collector) |

Base URL: `https://api.scaleway.com`
<!-- /pepin:gen provider-scaleway-live -->

The object storage collector (`internal/objectstorage`) issues `ListBuckets`, then per bucket
`GetBucketAcl`, `GetBucketVersioning`, `GetBucketPolicy`, `GetBucketTagging`,
`GetObjectLockConfiguration` and `GetBucketEncryption`. Each of those is best-effort: a call
that is refused leaves the attribute uncollected, and the control returns `not-evaluated` — never
a silent `pass`.

Note what is **not** collected live: `network` and `kubernetes_cluster` exist only in the
Terraform mapping. A live scan of a Scaleway tenant says nothing about VPCs or Kapsule clusters.

## Minimal permissions for a live scan

Derived from the provider descriptor, so this table and what the scan reports cannot diverge:
when a call is refused, the capability report and the `not-evaluated` reason name the very
grant listed here. **Confirmed** means the grant is stated in the official source cited beside
it — not that a scan was run with a deliberately reduced role. This repository holds no cloud
credentials and no automated check reaches a provider API.

<!-- pepin:gen provider-scaleway-permissions -->
| Collection unit | Minimum grant | Confirmed | Source |
|---|---|:-:|---|
| `access_key` | `IAMReadOnly (Organization scope)` | yes | scaleway/docs-content — pages/iam/reference-content/permission-sets.mdx |
| `iam_user` | `IAMReadOnly (Organization scope)` | yes | scaleway/docs-content — pages/iam/reference-content/permission-sets.mdx |
| `security_group_rule` | `InstancesReadOnly (Project scope)` | yes | scaleway/docs-content — pages/iam/reference-content/permission-sets.mdx |
| `compute_instance` | `InstancesReadOnly (Project scope)` | yes | scaleway/docs-content — pages/iam/reference-content/permission-sets.mdx |
| `object_storage_bucket` | `ObjectStorageReadOnly (Project scope)` | no | scaleway/docs-content — pages/object-storage/reference-content/s3-iam-permissions-equivalence.mdx |

**Reservations, stated rather than papered over.**

- **`object_storage_bucket`** — Covers ListBuckets, GetBucketAcl, GetBucketVersioning, GetBucketTagging and GetObjectLockConfiguration. GetBucketPolicy and GetBucketEncryption appear in NO read-only permission set of the equivalence table: the right that grants them could not be established, and it is not guessed. Both calls are best-effort - a 403 costs coverage, never correctness.
<!-- /pepin:gen provider-scaleway-permissions -->

Scaleway's vocabulary is an **IAM policy** made of rules, each carrying **permission sets** and
a scope. Create a dedicated Application, give it an API key, and attach a policy with two
rules:

| Rule scope | Permission set | Covers |
|---|---|---|
| Organization | `IAMReadOnly` | `/iam/v1alpha1/users`, `/iam/v1alpha1/api-keys` |
| Project | `InstancesReadOnly` | security groups, their rules, servers |
| Project | `ObjectStorageReadOnly` | `ListBuckets`, `GetBucketAcl`, `GetBucketVersioning`, `GetBucketTagging`, `GetObjectLockConfiguration` |

The gap the table above records for `object_storage_bucket` has a practical consequence worth
naming: `GetBucketPolicy` and `GetBucketEncryption` are best-effort calls, so a `403` on them
costs coverage — `objectstorage_bucket_public_access` may lose its policy signal and
`objectstorage_bucket_kms_encryption` returns `not-evaluated` — and never correctness.

`ObjectStorageBucketsRead` alone is **not** sufficient: its equivalence table does not include
`GetObjectLockConfiguration`.

A key scoped to one project sees only that project's buckets and instances.

## Terraform resources recognized

<!-- pepin:gen provider-scaleway-terraform -->
| Terraform resource | Normalized type | Exploded block |
|---|---|---|
| `scaleway_iam_api_key` | `access_key` | — |
| `scaleway_iam_policy` | `iam_policy` | — |
| `scaleway_instance_security_group` | `security_group` | — |
| `scaleway_instance_security_group` | `security_group_rule` | `inbound_rule[*]` |
| `scaleway_instance_security_group` | `security_group_rule` | `outbound_rule[*]` |
| `scaleway_instance_security_group_rules` | `security_group_rule` | `inbound_rule[*]` |
| `scaleway_instance_security_group_rules` | `security_group_rule` | `outbound_rule[*]` |
| `scaleway_instance_server` | `compute_instance` | — |
| `scaleway_object_bucket` | `object_storage_bucket` | — |
| `scaleway_object_bucket_acl` | `object_storage_bucket` | — |
| `scaleway_rdb_acl` | `managed_database` | `acl_rules[*]` |
| `scaleway_rdb_instance` | `managed_database` | — |
| `scaleway_vpc_private_network` | `network` | — |
<!-- /pepin:gen provider-scaleway-terraform -->

```bash
terraform plan -out tfplan && terraform show -json tfplan > plan.json
pepin scan scaleway --terraform plan.json
```

## Coverage

<!-- pepin:gen provider-scaleway-coverage -->
| Source | ✅ `supported` | ◐ `partial` | ∅ `not-applicable` | ✗ `unsupported` |
|---|---:|---:|---:|---:|
| terraform | 18 | 6 | 2 | 31 |
| live | 16 | 3 | 2 | 36 |
<!-- /pepin:gen provider-scaleway-coverage -->

Control by control, with the reason for every cell that is not fully supported, the source of
truth is the [coverage matrix](../coverage.md). It is computed from the same descriptor as this
page.

### Declared not applicable

A `not-applicable` is a claim, so it carries its justification, taken from the provider contract:

<!-- pepin:gen provider-scaleway-na -->
| Control | Justification recorded in the contract |
|---|---|
| `blockstorage_snapshot_not_public` | Scaleway block snapshots (api/block/v1) expose no sharing or public export mechanism: the risk of public exposure is structurally absent, compliant by construction (STO-2). |
| `blockstorage_volume_encryption` | Encryption at rest of block volumes is guest-side (LUKS/Cryptsetup), a customer responsibility (shared responsibility model); the block API exposes no encryption field, hence unobservable on the platform side (CHF-2). |
<!-- /pepin:gen provider-scaleway-na -->

### Observable from only one source

<!-- pepin:gen provider-scaleway-onesource -->
| Control | Observable only through | Reason on the blind side |
|---|---|---|
| `compute_instance_no_secrets_in_user_data` | terraform | deciding attribute "user_data" not projected by this source: a capability guard, so the scan returns "not-evaluated" |
| `compute_instance_public_ip_with_open_securitygroup` | live | deciding attribute "public_ip" not projected by this source: a capability guard, so the scan returns "not-evaluated" |
| `database_backup_enabled` | terraform | this source produces no resource of type "managed_database" |
| `database_encryption_at_rest_enabled` | terraform | this source produces no resource of type "managed_database" |
| `database_service_not_open_to_internet` | terraform | this source produces no resource of type "managed_database" |
| `iam_policy_no_privilege_escalation` | terraform | this source produces no resource of type "iam_policy" |
| `iam_user_mfa_enabled` | live | this source produces no resource of type "iam_user" |
| `network_securitygroup_default_deny` | terraform | this source produces no resource of type "security_group" |
| `objectstorage_bucket_kms_encryption` | live | deciding attribute "sse_kms_enabled" not projected by this source: a capability guard, so the scan returns "not-evaluated" |
| `objectstorage_bucket_versioning_enabled` | live | deciding attribute "versioning" not projected by this source: a capability guard, so the scan returns "not-evaluated" |
<!-- /pepin:gen provider-scaleway-onesource -->

This is the table to read before comparing a plan scan with a live scan of the same tenant:
some of those differences are coverage, not drift
([Terraform plan vs live scan](../concepts/terraform-vs-live.md)).

## A real scan

`examples/scaleway/terraform/` holds a deliberately misconfigured module. Its plan is committed,
so this runs with no account:

```bash
pepin scan scaleway --terraform examples/scaleway/terraform/plan.json
```

<!-- pepin:gen provider-scaleway-scan -->
```text
[…]
  │ CLD-CHF-2  │ Managed database without encryption at rest      │ HIGH     │ scaleway │ 1 │
  │ CLD-CMP-9  │ Cleartext secret in the user data (user-data)    │ HIGH     │ scaleway │ 1 │
  │ CLD-IAM-12 │ IAM policy allowing privilege escalation         │ HIGH     │ scaleway │ 1 │
  │ CLD-NET-1  │ Managed database reachable from the internet     │ HIGH     │ scaleway │ 2 │
  │ CLD-NET-2  │ Security group inbound default policy set to "a… │ HIGH     │ scaleway │ 1 │
  │ CLD-STO-3  │ Automatic backups disabled on a managed database │ HIGH     │ scaleway │ 1 │
  │ CLD-GVN-1  │ Incomplete inventory and tagging                 │ MEDIUM   │ scaleway │ 1 │
  │ CLD-STO-8  │ Object Lock (immutability) disabled on object s… │ LOW      │ scaleway │ 1 │
  ╰────────────┴──────────────────────────────────────────────────┴──────────┴──────────┴───╯
──────────────────────────────────────────────────────────────────────────────
 Summary

 Verdict: NON-COMPLIANT

 🔴 CRITICAL 1   🟠 HIGH 7   🟡 MEDIUM 1   🔵 LOW 1
──────────────────────────────────────────────────────────────────────────────
```
<!-- /pepin:gen provider-scaleway-scan -->

The same module, fixed, is in `examples/scaleway/terraform-fixed/`, and scanning it returns exit
code 0. The walkthrough is the [quickstart](../getting-started/quickstart.md).

## Limitations

- **No live VPC or Kubernetes collection.** Both types are mapped from Terraform only.
- **Bucket policy and bucket encryption** may be refused by a strictly read-only key (see
  above); the affected controls then return `not-evaluated`.
- **Encryption at rest of block volumes** is guest-side and unobservable from the API — this is
  a declared `not-applicable`, not an oversight.
- The general limits of the tool, all providers taken together, are in
  [Known limitations](../known-limitations.md).

## Where to go next

- [Coverage matrix](../coverage.md) — every control, every source.
- [Terraform plan vs live scan](../concepts/terraform-vs-live.md) — choosing the source.
- [Outscale](outscale.md) · [Exoscale](exoscale.md) — the other two sovereign clouds.
- [GitHub Actions](../guides/github-actions.md) · [GitLab CI](../guides/gitlab-ci.md).

## How this page stays true

The identity, credentials, endpoint, Terraform-mapping, coverage and not-applicable tables are
computed from `providers/scaleway.yaml` and from the reference by `internal/docgen`; the scan
output is a real run of the committed fixture. Adding a collected resource type changes this
page, and `TestGeneratedDocsAreUpToDate` fails until it is regenerated. The permission section
is written by hand, because the descriptor does not carry Scaleway's permission-set names: it
cites the source in which each name was verified, and says plainly what could not be verified.

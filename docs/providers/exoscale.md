> 🇬🇧 English · [🇫🇷 Français](exoscale.fr.md)

# Exoscale

Everything on this page that describes what Pépin collects is derived from
`providers/exoscale.yaml`, the descriptor the scanner itself reads. You do not need to open that
file to understand the coverage.

<!-- pepin:gen provider-exoscale-identity -->
| Descriptor field | Value |
|---|---|
| Description | Exoscale (CH) — instances, security groups, block storage, SKS, SOS |
| Scope | cloud |
| Region key (`--region`) | `zone` |
| API authentication | `exoscale-hmac` |
| Jurisdiction of the head office | CH |
| Established in the EU | no |
| Capital control | extra_ue |
| SecNumCloud | `non` |
| Extraterritorial exposure | no |
| Anchoring sources | exoscale.com/about-us ; newsroom.a1.com (A1 Digital acquiert Exoscale) ; América Móvil 60,8 % / ÖBAG 28,4 % du groupe A1 Telekom Austria (09/2025) |
<!-- /pepin:gen provider-exoscale-identity -->

Read those fields carefully: Exoscale is **Swiss, not EU-established**, and its capital control
is outside the European Union. That is not a disqualification — Switzerland has an adequacy
decision, and Exoscale operates European zones — but it is a fact the governance control
`CLD-GVN-4` reports, and a fact anyone building a sovereignty argument has to handle
explicitly. The descriptor records its sources.

## Authentication

<!-- pepin:gen provider-exoscale-credentials -->
| Logical key | Environment variable | Default |
|---|---|---|
| `access_key` | `EXOSCALE_API_KEY` | — |
| `secret_key` | `EXOSCALE_API_SECRET` | — |
| `zone` | `EXOSCALE_ZONE` | `ch-gva-2` |
| native configuration file | `~/.config/exoscale/exoscale.toml` | `exoscale` |
<!-- /pepin:gen provider-exoscale-credentials -->

- The API authenticates with Exoscale's own **HMAC** scheme (`exoscale-hmac`).
- The same API key and secret are used for SOS (object storage).
- The native configuration file is read when the variables are absent; `EXOSCALE_CONFIG`
  overrides its location.
- **The region key is `zone`.** `--region` therefore sets the zone, and the API host itself
  carries it (`api-{zone}.exoscale.com`): **one scan covers one zone**. Scanning several zones
  means several runs.

```bash
export EXOSCALE_API_KEY=... EXOSCALE_API_SECRET=...
pepin scan exoscale --live --region ch-gva-2
```

## What a live scan calls

Every endpoint, including the parent listing of a join.

<!-- pepin:gen provider-exoscale-live -->
| Normalized type | Call | Note |
|---|---|---|
| `blockstorage_snapshot` | `GET /block-storage-snapshot` | — |
| `blockstorage_volume` | `GET /block-storage` | — |
| `compute_instance` | `GET /instance` | parent listing of a join (called first) |
| `compute_instance` | `GET /instance/{vm.id}` | — |
| `iam_role` | `GET /iam-role` | — |
| `iam_user` | `GET /user` | — |
| `kubernetes_cluster` | `GET /sks-cluster` | — |
| `network` | `GET /private-network` | — |
| `security_group_rule` | `GET /security-group` | — |
| `object_storage_bucket` | `https://sos-{zone}.exo.io` | object storage S3 API (Go collector) |

Base URL: `https://api-{zone}.exoscale.com/v2`
<!-- /pepin:gen provider-exoscale-live -->

The object storage collector (`internal/objectstorage`) issues `ListBuckets`, then per bucket
`GetBucketAcl`, `GetBucketVersioning`, `GetBucketPolicy`, `GetBucketTagging`,
`GetObjectLockConfiguration` and `GetBucketEncryption`.

Exoscale is the only one of the three where a live scan reaches managed Kubernetes (SKS) through
the descriptor's own collection spec.

## Minimal permissions for a live scan

Exoscale's vocabulary is an **IAM Role** carrying a policy, and an **API Key bound to that
role** (`exo iam role create`, then `exo iam api-key create <key> <role>`; a key cannot be
reassigned to another role afterwards).

The policy denies by default and allows exactly the operations Pépin calls:

```json
{
  "default-service-strategy": "deny",
  "services": {
    "compute": {
      "type": "rules",
      "rules": [{
        "action": "allow",
        "expression": "operation in ['list-security-groups','list-private-networks','list-instances','get-instance','list-sks-clusters','list-block-storage-volumes','list-block-storage-snapshots']"
      }]
    },
    "iam": {
      "type": "rules",
      "rules": [{
        "action": "allow",
        "expression": "operation in ['list-users','list-iam-roles']"
      }]
    },
    "sos": {
      "type": "rules",
      "rules": [{
        "action": "allow",
        "expression": "operation in ['list-buckets','get-bucket-acl','get-bucket-versioning','get-bucket-policy','get-bucket-tagging','get-object-lock-configuration','get-bucket-encryption']"
      }]
    }
  }
}
```

Every operation name above was verified in Exoscale's official IAM references (the `compute`,
`iam` and `sos` operation catalogues), and the CEL policy grammar in the official policy guide.
Exoscale is the only one of the three providers that documents an IAM operation for reading a
bucket policy and a bucket's encryption configuration — the two calls whose read-only
permission we could not establish on Scaleway.

**One trap, documented by Exoscale itself:** if **no rule matches**, the request is refused
*regardless of the default service strategy*. Each service block must therefore contain a rule
that actually matches, which the `operation in [...]` form above does. A looser but equally
valid form is `operation.startsWith('get-') || operation.startsWith('list-')`.

Two optional hardening levers, both read from the same policy grammar: `source_ip.inIpRange(…)`
to bind the key to the scanner's address, and a short `max-session-ttl` on the role. Pépin reads
both notions as attributes of `iam_role`, which is how it can report on your *other* roles.

## Terraform resources recognized

<!-- pepin:gen provider-exoscale-terraform -->
| Terraform resource | Normalized type | Exploded block |
|---|---|---|
| `exoscale_block_storage_volume` | `blockstorage_volume` | — |
| `exoscale_compute_instance` | `compute_instance` | — |
| `exoscale_iam_role` | `iam_role` | — |
| `exoscale_private_network` | `network` | — |
| `exoscale_security_group_rule` | `security_group_rule` | — |
| `exoscale_sks_cluster` | `kubernetes_cluster` | — |
<!-- /pepin:gen provider-exoscale-terraform -->

```bash
terraform plan -out tfplan && terraform show -json tfplan > plan.json
pepin scan exoscale --terraform plan.json
```

## Coverage

<!-- pepin:gen provider-exoscale-coverage -->
| Source | ✅ `supported` | ◐ `partial` | ∅ `not-applicable` | ✗ `unsupported` |
|---|---:|---:|---:|---:|
| terraform | 21 | 1 | 5 | 30 |
| live | 25 | 1 | 5 | 26 |
<!-- /pepin:gen provider-exoscale-coverage -->

Control by control, with the reason for every cell that is not fully supported, the source of
truth is the [coverage matrix](../coverage.md).

### Declared not applicable

<!-- pepin:gen provider-exoscale-na -->
| Control | Justification recorded in the contract |
|---|---|
| `blockstorage_snapshot_not_public` | Exoscale block-storage snapshots cannot be exported or shared (official documentation): the risk of public exposure is structurally absent, compliant by construction (STO-2). |
| `loadbalancer_http_redirect_to_https` | resource type "load_balancer" absent from the exoscale API |
| `loadbalancer_logging_enabled` | resource type "load_balancer" absent from the exoscale API |
| `loadbalancer_ssl_listeners` | resource type "load_balancer" absent from the exoscale API |
| `objectstorage_bucket_kms_encryption` | SOS encrypts at rest by default (SSE-SOS, keys managed by Exoscale, SSE-S3 style) but exposes no customer-managed BYOK/KMS key at the bucket level (SSE-C stays per-object and unobservable), so the BYOK-at-bucket control is moot (CHF-4). |
<!-- /pepin:gen provider-exoscale-na -->

### Observable from only one source

<!-- pepin:gen provider-exoscale-onesource -->
| Control | Observable only through | Reason on the blind side |
|---|---|---|
| `iam_user_mfa_enabled` | live | this source produces no resource of type "iam_user" |
| `objectstorage_bucket_object_lock_enabled` | live | this source produces no resource of type "object_storage_bucket" |
| `objectstorage_bucket_public_access` | live | this source produces no resource of type "object_storage_bucket" |
| `objectstorage_bucket_versioning_enabled` | live | this source produces no resource of type "object_storage_bucket" |
<!-- /pepin:gen provider-exoscale-onesource -->

## A real scan

`examples/exoscale/terraform/` holds a deliberately misconfigured module. Its plan is committed,
so this runs with no account:

```bash
pepin scan exoscale --terraform examples/exoscale/terraform/plan.json
```

<!-- pepin:gen provider-exoscale-scan -->
```text
[…]
  │ CLD-IAM-1  │ IAM role with excessive privileges             │ HIGH     │ exoscale │ 1 │
  │ CLD-IAM-12 │ IAM policy allowing privilege escalation       │ HIGH     │ exoscale │ 1 │
  │ CLD-IAM-4  │ IAM role without a source IP restriction       │ HIGH     │ exoscale │ 2 │
  │ CLD-K8S-2  │ Kubernetes control plane not highly available  │ HIGH     │ exoscale │ 1 │
  │ CLD-NET-1  │ SSH (port 22) open to the internet             │ HIGH     │ exoscale │ 2 │
  │ CLD-GVN-1  │ Incomplete inventory and tagging               │ MEDIUM   │ exoscale │ 1 │
  │ CLD-K8S-3  │ Automatic Kubernetes cluster upgrades disabled │ MEDIUM   │ exoscale │ 1 │
  │ CLD-GVN-3  │ Resource hosted outside the European Union     │ LOW      │ exoscale │ 3 │
  ╰────────────┴────────────────────────────────────────────────┴──────────┴──────────┴───╯
──────────────────────────────────────────────────────────────────────────────
 Summary

 Verdict: NON-COMPLIANT

 🔴 CRITICAL 2   🟠 HIGH 11   🟡 MEDIUM 2   🔵 LOW 3
──────────────────────────────────────────────────────────────────────────────
```
<!-- /pepin:gen provider-exoscale-scan -->

## Limitations

- **One zone per scan.** The API host carries the zone; a multi-zone tenant needs one run per
  zone, and each run only knows about its own.
- **No load balancer type.** The controls that read `load_balancer` are declared not applicable,
  with that justification in the contract.
- **SOS encrypts at rest by default** with provider-managed keys, and exposes no
  customer-managed key at bucket level — so the BYOK control is not applicable rather than
  failing.
- The general limits of the tool are in [Known limitations](../known-limitations.md).

## Where to go next

- [Coverage matrix](../coverage.md) — every control, every source.
- [Terraform plan vs live scan](../concepts/terraform-vs-live.md) — choosing the source.
- [Scaleway](scaleway.md) · [Outscale](outscale.md) — the other two sovereign clouds.
- [GitHub Actions](../guides/github-actions.md) · [GitLab CI](../guides/gitlab-ci.md).

## How this page stays true

The identity, credentials, endpoint, Terraform-mapping, coverage and not-applicable tables are
computed from `providers/exoscale.yaml` and from the reference by `internal/docgen`; the scan
output is a real run of the committed fixture. The permission section is written by hand,
because the descriptor does not carry Exoscale's IAM vocabulary: every operation name in it was
read in an official reference, and anything that could not be verified would be marked as such.

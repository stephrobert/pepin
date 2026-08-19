> 🇬🇧 English · [🇫🇷 Français](known-limitations.fr.md)

# Known limitations and blind spots

For a posture scanner, a known limitation is part of the trust contract. An unstated one gets
discovered at the worst possible moment: during an audit.

This page states what Pépin **cannot** measure, and why. It applies to **v0.2.0** and is
regenerated with the code: the tables below are computed from the reference, the provider
descriptors and the `pass` lock — see [How this page stays true](#how-this-page-stays-true).

The justifications and reasons in those tables are quoted **verbatim** from the reference and
the provider contracts, which the repository keeps in French by convention. Translating them
here would put a second, unverified wording next to the one the scan actually prints.

## The five categories

| # | Category | Who can lift it |
|---|---|---|
| 1 | **Provider API** — the API exposes no such field | the provider |
| 2 | **Terraform provider** — the schema has no such attribute, or it is `known after apply` | HashiCorp / the provider's Terraform maintainers |
| 3 | **Pépin limitation** — the data exists and is reachable, but is not collected or not wired | this project |
| 4 | **Not technically observable** — the property does not live where an API can see it | nobody, by construction |
| 5 | **Organisational or contractual requirement** — the requirement is not about configuration | you, with documents, not with a scanner |

Categories 1, 2 and 4 surface as `not-applicable` (justified) or `not-evaluated` (with a
reason). Category 3 surfaces as `not-evaluated` and is the one that shrinks over time.
Category 5 has no status at all, because such requirements are not in the reference.

## 1 & 4 — declared not applicable, with justification

Every entry below comes from a provider contract (`providers/<name>.yaml`, `contrat`). The
scan returns `not-applicable` and carries the justification in `waiver.justification`, quoted
verbatim. An unjustified N/A is never produced.

<!-- pepin:gen not-applicable-list -->
| Control | Provider | Justification recorded in the contract |
|---|---|---|
| `blockstorage_snapshot_not_public` | exoscale | Exoscale block-storage snapshots cannot be exported or shared (official documentation): the risk of public exposure is structurally absent, compliant by construction (STO-2). |
| `blockstorage_snapshot_not_public` | scaleway | Scaleway block snapshots (api/block/v1) expose no sharing or public export mechanism: the risk of public exposure is structurally absent, compliant by construction (STO-2). |
| `blockstorage_volume_encryption` | outscale | osc-sdk-go/v2 Volume exposes no encryption field; encryption at rest is guest-side (EncFS/LUKS), a customer responsibility, hence unobservable on the platform side (CHF-2). |
| `blockstorage_volume_encryption` | scaleway | Encryption at rest of block volumes is guest-side (LUKS/Cryptsetup), a customer responsibility (shared responsibility model); the block API exposes no encryption field, hence unobservable on the platform side (CHF-2). |
| `iam_user_mfa_enabled` | outscale | resource type "iam_user" absent from the outscale API |
| `loadbalancer_http_redirect_to_https` | exoscale | resource type "load_balancer" absent from the exoscale API |
| `loadbalancer_http_redirect_to_https` | outscale | The Outscale LBU cannot redirect: `ListenerRule.Action` is documented as "always forward" in the OAPI contract (no redirect action), and no redirect attribute exists on `Listener`. The mechanism does not exist, so the control is not applicable (CHF-1). |
| `loadbalancer_logging_enabled` | exoscale | resource type "load_balancer" absent from the exoscale API |
| `loadbalancer_ssl_listeners` | exoscale | resource type "load_balancer" absent from the exoscale API |
| `objectstorage_bucket_kms_encryption` | exoscale | SOS encrypts at rest by default (SSE-SOS, keys managed by Exoscale, SSE-S3 style) but exposes no customer-managed BYOK/KMS key at the bucket level (SSE-C stays per-object and unobservable), so the BYOK-at-bucket control is moot (CHF-4). |
| `objectstorage_bucket_kms_encryption` | outscale | OOS encrypts server-side in AES256 with a PROVIDER key; there is neither a KMS service nor a customer-managed master key, so there is no BYOK to audit at the bucket level (CHF-4). Note: enabling SSE itself is opt-in and observable, which is a separate control, not this N/A. |
<!-- /pepin:gen not-applicable-list -->

The recurring one deserves its own paragraph. **Encryption at rest of block volumes is not
observable on Outscale or Scaleway**: it is performed inside the guest (LUKS/cryptsetup), under
the customer's responsibility in the shared-responsibility split, and the block API exposes no
encryption field at all. On Exoscale it is transparent and compliant by construction. There is
no workaround at the API level; if you need evidence, produce it from inside the instance.

## 3 — controls that can never reach `pass` today

These controls are declared for at least one provider, yet **no provider and no source** can
currently satisfy the four gates of a `pass`. They can still emit a `fail` when a rule fires,
but they will never confirm compliance.

<!-- pepin:gen never-pass -->
| Control | Severity | Reason |
|---|---|---|
| `governance_resource_required_tags` | medium | no targeted resource type, and the control does not read the provider descriptor: the "pass" lock cannot be lifted, so the scan returns "not-evaluated" as long as no deviation is detected |
<!-- /pepin:gen never-pass -->

**Consequence.** A tenant that genuinely satisfies one of these controls still gets
`not-evaluated`, not `pass`. That is the honest answer, and it is also a gap to close: the fix
is in `internal/assess` and in the provider collectors, not in the documentation.

**Workaround.** None inside Pépin. Do not build a CI gate that expects these controls to turn
green.

## 2 & 3 — observable through one source only

The reference declares these controls for the provider, but only one of the two sources can
actually decide them. The reason given is the one that applies to the source that cannot.

<!-- pepin:gen single-source -->
| Control | Provider | Observable only through | Reason |
|---|---|---|---|
| `blockstorage_snapshot_not_public` | outscale | live | this source produces no resource of type "blockstorage_snapshot" |
| `blockstorage_volume_snapshots_exist` | outscale | live | this source produces no resource of type "blockstorage_volume" |
| `compute_instance_deletion_protection` | outscale | live | deciding attribute "deletion_protection" not projected by this source: a capability guard, so the scan returns "not-evaluated" |
| `compute_instance_no_secrets_in_user_data` | scaleway | terraform | deciding attribute "user_data" not projected by this source: a capability guard, so the scan returns "not-evaluated" |
| `compute_instance_public_ip_with_open_securitygroup` | scaleway | live | deciding attribute "public_ip" not projected by this source: a capability guard, so the scan returns "not-evaluated" |
| `database_backup_enabled` | scaleway | terraform | this source produces no resource of type "managed_database" |
| `database_encryption_at_rest_enabled` | scaleway | terraform | this source produces no resource of type "managed_database" |
| `database_service_not_open_to_internet` | scaleway | terraform | this source produces no resource of type "managed_database" |
| `governance_resource_region_in_eu` | outscale | live | deciding attribute "region" not projected by this source: a capability guard, so the scan returns "not-evaluated" |
| `iam_accesskey_expiration_set` | outscale | live | this source produces no resource of type "access_key" |
| `iam_account_mfa_enforced` | outscale | live | this source produces no resource of type "api_access_policy" |
| `iam_apiaccesspolicy_max_key_expiration` | outscale | live | this source produces no resource of type "api_access_policy" |
| `iam_apiaccessrule_defined` | outscale | live | this source produces no resource of type "api_access_summary" |
| `iam_apiaccessrule_no_public_cidr` | outscale | live | this source produces no resource of type "api_access_rule" |
| `iam_no_root_access_key` | outscale | live | this source produces no resource of type "access_key" |
| `iam_policy_no_privilege_escalation` | scaleway | terraform | this source produces no resource of type "iam_policy" |
| `iam_user_mfa_enabled` | exoscale | live | this source produces no resource of type "iam_user" |
| `iam_user_mfa_enabled` | scaleway | live | this source produces no resource of type "iam_user" |
| `kubernetes_cluster_auto_upgrade_enabled` | outscale | live | this source produces no resource of type "kubernetes_cluster" |
| `kubernetes_cluster_control_plane_highly_available` | outscale | live | this source produces no resource of type "kubernetes_cluster" |
| `kubernetes_cluster_deletion_protection` | outscale | live | this source produces no resource of type "kubernetes_cluster" |
| `kubernetes_cluster_not_publicly_accessible` | outscale | live | this source produces no resource of type "kubernetes_cluster" |
| `loadbalancer_logging_enabled` | outscale | live | this source produces no resource of type "load_balancer" |
| `loadbalancer_ssl_listeners` | outscale | live | this source produces no resource of type "load_balancer" |
| `network_peering_cross_organization` | outscale | live | this source produces no resource of type "network_peering" |
| `network_securitygroup_default_deny` | scaleway | terraform | this source produces no resource of type "security_group" |
| `network_securitygroup_default_restrict_traffic` | outscale | live | deciding attribute "security_group_name" not projected by this source: a capability guard, so the scan returns "not-evaluated" |
| `objectstorage_bucket_default_encryption` | outscale | live | this source produces no resource of type "object_storage_bucket" |
| `objectstorage_bucket_kms_encryption` | scaleway | live | deciding attribute "sse_kms_enabled" not projected by this source: a capability guard, so the scan returns "not-evaluated" |
| `objectstorage_bucket_object_lock_enabled` | exoscale | live | this source produces no resource of type "object_storage_bucket" |
| `objectstorage_bucket_object_lock_enabled` | outscale | live | this source produces no resource of type "object_storage_bucket" |
| `objectstorage_bucket_public_access` | exoscale | live | this source produces no resource of type "object_storage_bucket" |
| `objectstorage_bucket_public_access` | outscale | live | this source produces no resource of type "object_storage_bucket" |
| `objectstorage_bucket_versioning_enabled` | exoscale | live | this source produces no resource of type "object_storage_bucket" |
| `objectstorage_bucket_versioning_enabled` | outscale | live | this source produces no resource of type "object_storage_bucket" |
| `objectstorage_bucket_versioning_enabled` | scaleway | live | deciding attribute "versioning" not projected by this source: a capability guard, so the scan returns "not-evaluated" |
<!-- /pepin:gen single-source -->

Two structural asymmetries explain most of these rows.

**Live collection always carries the region**, which a Terraform plan does not, unless a
mapping names the field. So `governance_resource_region_in_eu` is decidable live and
`not-evaluated` on most plans.

**A Terraform plan omits everything `known after apply`.** This is a limit of the source, not a
defect of Pépin, and it has a consequence worth naming: on a plan, an instance genuinely
without a security group is **indistinguishable** from an instance whose security group is
created by the same plan, since `security_group_id` is then absent from `planned_values`.
Pépin treats the attribute as *not collected*: the control returns `not-evaluated` with its
reason, rather than a guessed `fail` on a machine that is in fact attached. That is exactly
what the lock described in [the assessment model](concepts/assessment-model.md) is for — better
to withhold a conclusion than to reach a wrong one, in either direction.

**Workaround:** audit the effective configuration with `--live`, or reference a literal value
known at plan time when that matches your use case.

## The full picture

Per provider and per source, over all controls in the reference:

<!-- pepin:gen coverage-totals -->
| Provider | Source | ✅ `supported` | ◐ `partial` | ∅ `not-applicable` | ✗ `unsupported` |
|---|---|---:|---:|---:|---:|
| exoscale | terraform | 21 | 1 | 5 | 30 |
| exoscale | live | 25 | 1 | 5 | 26 |
| outscale | terraform | 17 | 4 | 4 | 32 |
| outscale | live | 39 | 1 | 4 | 13 |
| scaleway | terraform | 18 | 6 | 2 | 31 |
| scaleway | live | 16 | 3 | 2 | 36 |
| kubernetes | live | 4 | 0 | 0 | 53 |
<!-- /pepin:gen coverage-totals -->

The per-control detail, with the reason for every cell that is not fully supported, is the
[coverage matrix](coverage.md).

## Remediation proofs

Beyond the remediation **text** carried by every finding (which is guaranteed by
`TestEveryFindingCarriesRemediation`), the repository aims to ship a remediation **proof** per
(control, provider) under `references/remediation/` — a self-contained Terraform module, or a
documented note. Today:

<!-- pepin:gen remediation-coverage -->
| Provider | Remediation proofs |
|---|---:|
| exoscale | 4 / 26 |
| kubernetes | 0 / 4 |
| outscale | 0 / 40 |
| scaleway | 0 / 25 |
| **Total** | **4 / 95** |
<!-- /pepin:gen remediation-coverage -->

This is deliberately **not** wired into `mise run validate`. A gate that is permanently red is
a gate people learn to ignore; it will be reconnected one provider at a time, starting with the
first to reach 100 %. Run `mise run check-remediation` for the per-control list.

**Consequence for you:** every finding tells you what to do, in prose. Not every finding comes
with a tested Terraform module proving it.

## Limitations of the tool itself

### An unredacted evidence bundle is a sensitive artifact

`--seal` embeds the **raw** evaluated inventory in `input.json`: user-data (which is exactly
where Pépin finds hardcoded secrets), IAM policy documents, bucket policies. Pépin warns about
it on stderr at seal time.

- **Workaround:** `--seal --redact` replaces sensitive attribute values with digests. The
  finding stays; the value leaves.
- **Trade-off:** a redacted bundle is **incompatible with `verify --re-derive`**, since
  detection cannot replay against redacted data. A shared bundle relies on the cosign
  signature instead.

### A Terraform plan is not a state

`--terraform` audits the **planned** state. It says nothing about drift, nothing about
resources created outside the code, and nothing about attributes still unknown at plan time.
The verdict says *"declared scope (Terraform plan, planned state)"* rather than "compliant" for
that reason. Use `--live` for the effective configuration.

### `--live` sees exactly what your credentials see

A permission missing from the read-only role turns into `not-evaluated`, not into an error.
That is the safe failure mode — but it means an under-privileged scan produces a *quieter*
report, not a louder one. Check the `not-evaluated` count, not just the deviation count.

### Sovereignty facts are declared, not verified

`governance_provider_sovereignty` evaluates the `souverainete` block of the provider
descriptor: legal seat, ownership, SecNumCloud status, extraterritorial exposure, with their
sources. Those are **attestations transcribed from public sources**, not measurements. The
evidence string says so, and the control is excluded from the "something was measured" count
that drives exit code `3`.

### `evidence.proves` is always empty

The assessment's `evidence` object carries a `proves` field, a three-slot array inherited from
the shared `scankit` module (`[running, persistent, reboot-survivable]`). That notion belongs
to host hardening, not to cloud posture: Pépin never fills it, and it serialises as
`["", "", ""]`. Consumers should ignore it; it is not a Pépin signal.

### The `live` column of the coverage matrix is derived, not observed

The coverage matrix is **computed from the descriptors**: it states what a provider's live
collection spec and Terraform mapping are declared to project, and what the API contract marks
as verified. It is not the record of an observed run: no live scan produces that column, and
nothing in this repository's automated checks calls a provider API.

That distinction matters when reading a green cell in the `live` column. It means "this
descriptor projects the deciding attribute, and the contract is verified", not "an API returned
this field during a measured run". Should the two diverge on your tenant, the scan says so with
a `not-evaluated` and its reason, never with a silent green — but the matrix itself is a
statement of intent by the descriptor, not evidence.

### Nothing is measured between two runs

A result describes an instant. Pépin has no agent, no watch mode and no history. Continuous
posture is a scheduling problem you solve in CI, and the sealed bundles are what you keep.

## 5 — requirements a scanner cannot answer

Some SCSL, SecNumCloud and ISO requirements are about organisation, contracts, procedures or
personnel: reversibility clauses, staff vetting, incident procedures, subcontractor management.
They have **no configuration surface**, so they are not in the active reference at all — they
are not silently reported as passing. `referentiel/gaps.md` tracks what is triaged and why, and
`pepin scsl` reports coherence with the frozen SCSL index.

## Resolved limitations

None yet: this page is published with v0.2.0. When a limitation is lifted, it is removed from
the sections above and recorded here with the version that lifted it, so that a reader of an
older report can tell what was true at the time.

| Limitation | Lifted in |
|---|---|
| _(none)_ | |

## How this page stays true

The tables above are computed by `internal/docgen` from `referentiel/controles.yaml`,
`providers/*.yaml` and the `pass` lock in `internal/assess` — the same functions the scan
itself calls, not a second implementation. `mise run gen-docs` regenerates them;
`TestGeneratedDocsAreUpToDate` fails the build if what is committed no longer matches what the
code computes. A limitation cannot quietly disappear from this page, and a new one cannot
quietly stay off it.

## See also

- [Coverage matrix](coverage.md) — the per-control detail.
- [The assessment model](concepts/assessment-model.md) — why `not-evaluated` exists.
- [Scope and non-goals](concepts/scope.md) — what Pépin never claimed to do.

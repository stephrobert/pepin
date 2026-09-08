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
| exoscale | 26 / 26 |
| kubernetes | 0 / 4 |
| outscale | 0 / 40 |
| scaleway | 0 / 25 |
| **Total** | **26 / 95** |
<!-- /pepin:gen remediation-coverage -->

This is deliberately **not** wired into `mise run validate`: over all providers the count is
still partial, and a gate that is permanently red is a gate people learn to ignore. Exoscale is
the first provider at 100 %, and a test holds that ground —
`TestExoscaleRemediationCoverageStaysComplete` fails when an exoscale control lands without its
proof. The other providers join that guard as they reach 100 %. Run
`mise run check-remediation` for the per-control list.

**Consequence for you:** every finding tells you what to do, in prose. Not every finding comes
with a tested Terraform module proving it.

## The veracity contract, and what it still owes

For a posture scanner the unit that matters is not `fixture → Rego → expected FAIL`. It is the
whole chain: **API response or plan → collector → normalization → capability guard → Rego →
assessment → verdict**. The inline EIM policy incident proved why: a policy granting `Action:
"*"` escaped *every* `iam_policy_*` control, the Rego rule was not wrong, and the data simply
never reached it. A perfect Rego test would have stayed green while the scanner produced a
false green.

A veracity scenario therefore runs against the **binary**, on the whole chain, and reads the
status the assessment publishes. Each path — control × provider × source — must prove the
verdicts it can actually reach: three for a path that can conclude (`fail`, `pass`,
`not-evaluated`), one for a path that cannot lift the `pass` lock, one for a path the provider
contract declares not applicable.

What is not yet proven is **counted**, not hidden:

<!-- pepin:gen veracity-debt -->
| Figure | Count |
|---|---:|
| Control x provider x source paths on which Pépin concludes | 178 |
| Paths whose every reachable verdict is proven end to end | 23 |
| Verdicts to prove in total | 458 |
| Verdicts left to prove | 395 |
<!-- /pepin:gen veracity-debt -->

The remainder is listed path by path in `internal/veracity/testdata/debt.txt`. That ledger is a
gate in both directions: an unproven obligation that is *not* written there fails the build — so
a control added without its scenarios cannot land — and a line that is no longer owed fails it
too.

**Why a counted debt rather than a green matrix.** Four scenarios for every path would be some
seven hundred cases. Nobody can *test* seven hundred cases — that is, break each one to check it
turns red — and a matrix generated from a template would be exactly the false green this
scanner exists to fight: a test that passes because its cases are hollow measures only itself.

**What the reference tenants add to it, and what they do not.** Part of that debt is now paid
by real third-party configurations, pinned to a commit and replayed on every build
([Reference tenants](guides/reference-tenants.md)). They close a gap a fixture cannot: a
fixture is written by the rule's author, so it proves the rule fires, never that it is right
about a configuration nobody designed for it. They do **not** close the live one — a Terraform
plan carries the *planned* state, and what a provider *answers* stays owed to a real
collection. And a verdict only counts when it is substantive: a `not-evaluated` on a tenant
that carries no resource of the targeted type states an absence, not a capability guard, so it
is not counted.

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
That is the safe failure mode, and since v0.3.0 it is also a **loud** one: the scan records
every collection unit it could not read, prints a capability report before any verdict, names
the missing unit as the reason of each affected control, and does not return `0`. An
under-privileged scan therefore no longer produces a quieter report than a privileged one — it
produces a report that says what it could not see. See
[exit code 3](reference/exit-codes.md#3--the-scan-does-not-establish-compliance).

What remains true: Pépin cannot tell you what a *broader* credential would have found. It
reports the surface it was given, and the absence of that surface, never what lies beyond it.

### The minimum permissions are documented, not measured

Each provider page lists the API calls a scan makes and the read permissions they need. Those
lists are **derived from the collection descriptors** — the endpoints the spec declares — and
checked against the provider's public IAM documentation. Each page marks which lines are
confirmed by documentation and which remain unverified.

Two halves, and they are not in the same state. The **endpoints** half is now measured: a
recorded session proves the collector really emits what the descriptor declares, so the list is
no longer a claim about code nobody watched run
([Tracing real API calls](guides/tracing-api-calls.md)). The **grants** half is not, and cannot
be from here: confirming that `InstancesReadOnly` and nothing more suffices for
`ListSecurityGroups` requires running a scan with a deliberately reduced role against a real
tenant. This repository holds no cloud credentials, by design, and no automated check reaches a
provider API. Most of Outscale's entries are `a_verifier` for a sharper reason still: the action
names are **inferred** from the documented EIM syntax rather than read from a published
permission set, and an inferred grant is a guess, which is precisely what a CSPM must not ask
you to grant.

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

### The `live` column of the coverage matrix is derived; only its plumbing is observed

The coverage matrix is **computed from the descriptors**: it states what a provider's live
collection spec and Terraform mapping are declared to project, and what the API contract marks
as verified. Nothing in this repository's automated checks calls a provider API.

Since v0.4.0, one half of that statement is no longer merely declared. A recorded session of
real HTTP calls — taken against a **local emulator**, `feint serve`, with no cloud credential
whatsoever — is replayed on every build
(`internal/genprovider/testdata/transcripts/`, [Tracing real API calls](guides/tracing-api-calls.md)).
It measures that the endpoints a descriptor announces really go out on the wire, that
parent/child joins fire on an id read from the parent's response, and that the pagination
parameters are the declared ones. An endpoint declared and never emitted now fails the build.

What that recording does **not** establish, and what a green `live` cell therefore still does
not mean:

- that the provider returns the field under that name and that type — the native contract is
  read in the SDK, never measured here;
- that the provider's real pagination bounds match the declared ones;
- how it behaves under rate limiting;
- that it refuses with `403` rather than `200` and an error body.

So a green cell means "this descriptor projects the deciding attribute, the contract is marked
verified, and the call that would fetch it is provably emitted". It does not mean "an API
returned this field during a measured run". Should the two diverge on your tenant, the scan
says so with a `not-evaluated` and its reason, never with a silent green.

### An emulator is not the provider

The recordings above come from `feint`, a local emulator of Scaleway, Outscale and Exoscale.
The distinction it forces is worth stating on its own, because collapsing it would manufacture
exactly the false confidence this page exists to prevent: **an emulator proves what Pépin does,
not what the cloud answers.**

One consequence is structural rather than incidental. The emulator **accepts every credential**
— measured: no auth header returns `200`, a junk token returns `200`, and it exposes no fault
injection. It can therefore never produce a `403`, and no recording against it can measure how
a real permission refusal is classified. That classification is exercised by
`internal/collect/status_test.go` against a socket that really refuses; what stays unknown is
whether a given provider refuses with that status at all.

The same holds for every value in a recording: they are minted by the emulator at startup, not
returned by a cloud. They are safe to commit for exactly that reason, and useless as evidence
about a provider's contract for exactly that reason too.

### A finding's file and line need the `.tf` sources beside the plan

`terraform show -json` carries no file and no line — the configuration representation of a
resource holds `address`, `type`, `name` and `expressions`, and nothing about the document it
came from. Pépin therefore *measures* the origin in the `.tf` files sitting next to the plan.
Three consequences, and none of them is worked around:

- A plan produced elsewhere and handed over alone gets a **module** and no file or line.
- A module pulled from a registry or a git repository has no directory in the working tree: its
  resources keep their module, and nothing more.
- A `resource "type" "name"` header written across several lines, or declared in a `.tf.json`
  file, is not found; the origin is absent rather than approximate.

There is no flag to point at a different source tree, and that is deliberate: resolving a block
header against a tree the plan did not come from would produce a plausible and wrong line, which
is worse than no line at all.

### Nothing is measured between two runs

A result describes an instant. Pépin has no agent, no watch mode and no history. Continuous
posture is a scheduling problem you solve in CI, and the sealed bundles are what you keep.

### A recent snapshot is not a backup policy

`blockstorage_volume_snapshots_exist` measures one thing: does a volume in use have, within
the configured window (7 days by default), at least one snapshot whose native state says it is
completed? The state is checked, not just the date.

**Which state means "completed" is read, not observed.** `completed` for Outscale and `created`
for Exoscale come from those providers' published OpenAPI schemas, cited in the contracts. No
snapshot in a terminal state has ever been seen by this repository, and the emulator recordings
do not close that gap either: it returns an empty list for both types, so no state value passed
through them. If a provider ever returns a value outside its own documented enum, the control
does not conclude — the guard is on the value, not on its absence — but nothing here would
notice the drift before a real scan did.

It does not prove that the snapshot is **restorable** — no restore is attempted —, that it is
**complete** in the application sense, that a **retention** is honoured (one snapshot
satisfies the control), or that a **backup policy** exists at all. Symmetrically, a volume
backed up by other means — application backup, replication, a provider backup service, an
external tool — is reported here: an accepted false positive, to be handled with a dated
exemption rather than by disabling the control.

This control therefore takes no part in any "backup compliant" claim. Its wording says so, in
both languages, and the reference records it in the control description.

### Secret detection has confidence levels, not certainty

Generic heuristics (`password=…`, `api_key=…`) cannot tell `password=changeme123456` from a
real secret. Each finding carries `labels.confidence` (`high` | `medium` | `low`) so a
pipeline can triage, and the reporting threshold is configurable. Raising it is a relaxation:
"no cleartext secret" cannot be proven once you have chosen not to look at part of what was
found, so the CLD-CMP-9 mapping is dropped and the report says so. The detected value never
appears, at any level.

### A relaxed setting removes the mapping, not the measurement

A control whose configuration falls outside its normative constraint keeps its status: it was
genuinely evaluated, against a lower bar. What it loses are its `references` — it no longer
claims to cover CIS, ISO or SecNumCloud. Pépin cannot tell you whether your own bar is
appropriate for your context; it can only refuse to let a lowered bar keep a badge it no
longer earns. See [Configuring controls](guides/control-configuration.md).

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

> 🇬🇧 English · [🇫🇷 Français](README.fr.md)

# Exoscale qualification tenant

A deliberately faulty Exoscale organisation: **one fault per resource, one
counterexample per control**, applied on a real account, scanned `--live` in all five
formats, sealed, verified, rescanned `--terraform` on the same plan, destroyed, proven
destroyed, and compared with [`expected.yaml`](expected.yaml). Any difference is NO-GO.
The doctrine and the runner are described in [`../README.md`](../README.md); what
follows is what is specific to Exoscale.

```bash
PROVIDER=exoscale mise run qualify:plan                       # plan + scan --terraform, nothing is created
PEPIN_GATE_LIVE=1 PROVIDER=exoscale mise run qualify           # apply, scan, seal, destroy, prove, compare
```

The account is confirmed before any apply: the API binds the key to an organisation
(`GET /api-key/{key}` → `org-id` — `GET /organization` is refused to a key whose role
does not cover the organisation, and a qualification key has no reason to), and that
identifier must equal **`PEPIN_QUAL_EXO_ORG`**. A missing variable is a refusal,
nothing is applied ([ADR-0012](../../docs/adr/0012-aucun-identifiant-en-ci.md): no
account identifier is committed). The native credentials are those of
`~/.config/exoscale/exoscale.toml` or `EXOSCALE_API_KEY`/`EXOSCALE_API_SECRET`; since
the Terraform provider does not read the file, the `variables` hook hands them to it
through the run environment, without writing them.

## What the stack creates

| Family | Count | Faulty resource → control | Counterexample (must stay silent) |
|---|---:|---|---|
| Security groups | 9 (+ 15 rules) | `ssh_open` → `…tcp_port_22` · `rdp_open` → `…tcp_port_3389` · `db_open` (5432) → `…high_risk_tcp_ports` · `snmp_open` (UDP 161) → `…high_risk_udp_ports` · `any_open` (all TCP and all UDP) → the four above · `quartet` (four `/2`) → `…tcp_port_22` by evasion · `egress_any` (egress TCP 1-65535 to the Internet) → `network_securitygroup_unrestricted_egress` · `undocumented_flow` (a rule without description, port 443) → `network_flow_matrix_documented` | `hardened` (SSH from `10.0.0.0/8`, HTTPS from anywhere, egress TCP 443, every flow described) |
| Private networks | 2 | `undocumented` → `network_documented` | `documented` (Owner, Project, Env, description) |
| Instances | 4 applied + 1 on the plan | `exposed` (public IP, open SSH) → `compute_instance_public_ip_with_open_securitygroup` · `untagged` → `governance_resource_required_tags` · `secrets` (cloud-init with a password) → `compute_instance_no_secrets_in_user_data` · `swiss` (zone `ch-gva-2`, **plan only**) → `governance_resource_region_in_eu` (low: outside the EU, inside the European trusted area) | `hardened` (public IP behind a closed group, labelled, `de-fra-1`, plain cloud-init); `untagged` is also the NET-3 counterexample: the same open SSH group as `exposed`, but `private = true`, hence no public IP |
| Block storage volumes | 2 (+ 1 snapshot) | `unsnapshotted` (attached to `secrets`, never backed up) → `blockstorage_volume_snapshots_exist` | `snapshotted` (attached to `hardened`, one completed snapshot of the day); both encrypted by construction (`blockstorage_volume_encryption` passes) |
| SKS clusters | 2, no nodepool | `weak` (`starter`, no auto-upgrade, audit disabled) → `kubernetes_cluster_control_plane_highly_available`, `…auto_upgrade_enabled`, `…audit_logging_enabled` | `strong` (`pro`, auto-upgrade, audit endpoint) |
| IAM roles | 4 | `admin` (`allow` by default) → `iam_role_no_admin_privileges` · `no_source_ip` → `iam_role_source_ip_restricted` · `unbounded` (no `duration(`) → `iam_role_key_lifetime_bounded` | `restricted` (`deny` by default, `source_ip`, `duration(`) |
| SOS buckets (`extra` hook, outside Terraform) | 3 | `public` (ACL `public-read`) → `objectstorage_bucket_public_access` · `unversioned` → `objectstorage_bucket_versioning_enabled` · `public` and `unversioned` (no lock) → `objectstorage_bucket_object_lock_enabled` | `hardened` (private, created with Object Lock — through the S3 API, `exo storage` cannot —, hence versioned) |
| Provider | — | `governance_provider_sovereignty` fails on `exoscale` itself: head office in Switzerland, decisive non-EU capital control — facts of the descriptor, not of the tenant | — |

What is specific to Exoscale, and shaped this stack:

- **Four instances.** That is the organisation's quota (`GET /quota`: `instance` 4),
  discovered at the first apply (`Usage of resource 'instance' has been exceeded`, two
  instances refused, everything else destroyed and proven destroyed). Hence a
  counterexample carried by a resource already faulty elsewhere (`untagged`, private),
  a faulty volume carried by `secrets`, and `swiss` on the plan only
  (`terraform_only_resources`, which the runner sets to `false` to apply).
- **One zone scanned.** Resources are zonal and `scan --live --region de-fra-1` reads
  that zone only: an instance in `ch-gva-2` would be invisible to it, and
  `governance_resource_region_in_eu` is proven to fail on the plan only. The inventory
  of the destruction proof, on the other hand, walks both zones (`zones:` in
  `tenant.yaml`).
- **No "all protocols".** A group rule accepts only `tcp`, `udp`, `icmp`, `icmpv6`,
  `ah`, `esp`, `gre`, `ipip` (provider 0.71.0 and API v2).
  `network_securitygroup_allow_ingress_from_internet_to_all_ports` is therefore **not
  applicable** on Exoscale (#202, #206) — `any_open` opens all TCP and all UDP, and the
  four port-family controls fire. "All egress" is written `tcp 1-65535 → 0.0.0.0/0`,
  and `network_securitygroup_unrestricted_egress` recognises that form (#206):
  `egress_any` is a deviation.
- **SOS buckets cannot be tagged.** SOS accepts `PutBucketTagging` (200, through SigV4
  as through the `aws` CLI) and persists nothing: `GetBucketTagging` returns
  `NoSuchTagSet` right after. An SOS bucket therefore never has a tag, whatever one
  does, and `governance_resource_required_tags` fails on the three buckets **by
  construction** — pinned as measured in `expected.yaml`, an unfixable false positive,
  issue #208.
- **An SKS cluster creates an IAM role `sks-ccm-<cluster>`** (editable, `deny` by
  default, no source-IP nor lifetime bound) and deletes it with the cluster: every
  cluster brings two `iam_role_*` deviations by construction, outside the tenant here
  (unprefixed subjects), reported and not gated — issue #213. The role of the
  qualifying key (`deny` by default, no CEL rule) carries the same two deviations.
- **A private instance has no security group.** `private = true`
  (`public-ip-assignment: none`): the API ignores the declared groups and returns
  `security-groups: []`. `compute_instance_has_security_group` turns that into a
  deviation with an impossible remediation — pinned as measured on `untagged`, issue
  #212.
- **Live false green on SKS audit.** The API returns `audit: {}` for a cluster whose
  audit is disabled; the collector derives `audit_enabled` from `audit.endpoint`,
  absent, projects nothing on `weak`, and `kubernetes_cluster_audit_logging_enabled`
  concludes `pass` on the only cluster observed (`observed=1/2`) — the plan fails on
  `weak`. Pinned as a **known defect** (#209); the fix will turn this gate
  NO-GO on purpose.
- **Live false green on instance tagging.** The live collector does not project
  `labels` on instances, volumes and clusters (the contract says `a_verifier`); only
  buckets carry `tags`, the rule silently skips the rest, and `untagged` passes — issue
  #211. The plan catches it.
- **`region` is projected on nothing live**: `governance_resource_region_in_eu`
  renders `not-evaluated` on every Exoscale organisation (a named collection debt,
  ADR-0010, issue #210).

## What the upstream provider is known to leave behind on `destroy`

Read before choosing the resource types, in `exoscale/terraform-provider-exoscale`
(0.71.0 pinned, lockfile committed). What each one changed in this stack:

| Upstream | State | What it does | What this tenant does about it |
|---|---|---|---|
| [#453](https://github.com/exoscale/terraform-provider-exoscale/issues/453) the provider does not detach before deleting (elastic IPs, security groups); [#537](https://github.com/exoscale/terraform-provider-exoscale/issues/537) `Cannot delete group when it's in use by virtual machines` | #453 closed 2025-08 (IP fixed in 0.69.2, [#460](https://github.com/exoscale/terraform-provider-exoscale/pull/460)); #537 open | A group referenced by an instance neither destroys nor renames. | No elastic IP; each group is attached to Terraform instances only, destroyed first by dependency. The rescue cleanup deletes the instances, waits, then the groups. |
| [#375](https://github.com/exoscale/terraform-provider-exoscale/issues/375) an instance too small for a block storage volume disappears from the state (`Instance size must be at least small`) | open | A `micro` instance with `block_storage_volume_ids` is created, then lost by Terraform: a leftover. | The two instances carrying a volume are `standard.small` (`instance_type_with_volume`). |
| [#207](https://github.com/exoscale/terraform-provider-exoscale/issues/207) destroy does not wait for dependent resources; [#210](https://github.com/exoscale/terraform-provider-exoscale/issues/210) an NLB created by an SKS cluster's CCM survives the destroy; [#240](https://github.com/exoscale/terraform-provider-exoscale/issues/240) cluster destroy in `context deadline exceeded` | closed (2022-2023) | What a cluster creates outside Terraform outlives it. | Clusters **without nodepool** and `create_default_security_group = false`: nothing is created outside Terraform, and the inventory lists NLBs, pools and groups to prove it. |
| [#438](https://github.com/exoscale/terraform-provider-exoscale/issues/438) `aws_s3_bucket` never finishes on SOS | closed 2025-06 | A bucket through the `aws` provider is neither reliable nor cleanly destroyed. | Buckets go through `exo storage` (the reference CLI) and the S3 API, outside Terraform, in the `extra` hook; deleted recursively and re-listed. |
| [#76](https://github.com/exoscale/terraform-provider-exoscale/issues/76) destroy fails after a private network was removed by hand | closed 2020 | — | Nothing is removed by hand during a run; the rescue cleanup only runs after the destroy. |
| Measured here: **`GET /private-network` on `api-de-fra-1` sometimes never answers** (one call in three or four), then answers in 0.2 s | to report upstream | A proof listing that would return "unknown" on a glitch. | The hook retries three times (60 s timeout) before concluding. |
| Measured here: **SOS enables versioning on a bucket created with Object Lock** | — | The buckets' counterexample is versioned by construction. | `hardened` is the counterexample of all three bucket controls. |
| Measured here: **the `instance` quota still counts, minutes after a successful destroy, instances no listing shows** (`GET /quota`: usage 2, 0 instances listed, 5 min later) | to report to Exoscale | The next apply is refused (`Usage of resource 'instance' has been exceeded`) on an empty account: run 4 hit it. | The `variables` hook waits (15 min at most) for the counter to fall back to zero before the runner applies. |
| Measured here (once, on a cluster outside the tenant): **the `sks-ccm-<cluster>` API key SKS creates for the CCM outlived its cluster by some twenty minutes**; its role was deleted, and `DELETE /api-key` answered 403 `API Key not in organization`, through the API as through `exo iam api-key delete`; it eventually vanished on its own | — (asynchronous deletion, platform side) | Meanwhile an inert key (its role no longer exists) is listed, and the organisation cannot remove it. The tenant's own clusters left none within the proof's window (empty `api_keys` delta on every run). | The inventory lists API keys: a survivor shows up in the delta, and the rescue cleanup attempts the deletion and reports the 403 without blocking. |

The destruction proof lists 15 families through API v2 (instances, private networks,
volumes, snapshots, SKS clusters, elastic IPs, instance pools, NLBs, private templates,
SSH keys, anti-affinity groups, security groups, IAM roles, API keys) and the buckets
through `exo storage list`, in both zones, filtered on the tenant's label and prefix,
and compares the whole listing with the one taken before apply. `exo`, the provider's
reference CLI, is what a maintainer replays by hand:

```bash
exo compute instance list -O table
exo compute security-group list -O table
exo compute private-network list -O table
exo compute block-storage list -O table
exo compute block-storage snapshot list -O table
exo compute sks list -O table
exo compute elastic-ip list -O table
exo iam role list -O table
exo iam api-key list -O table
exo storage list -O table
```

Each command lists every zone by default (`-z` for one).

## What this tenant cannot exercise, and why

`tenant.yaml`, `not_exercised:` — `…_all_ports` (not applicable: no "all protocols"),
`governance_resource_region_in_eu` live (`region` not collected; and one zone read,
four instances in the quota), users' MFA (real people), the deviations of the
qualifying key's role and of the `sks-ccm-*` roles, SOS bucket tagging (impossible by
construction), the provider's sovereignty (a fact of the descriptor).

## Cost and budget

Hourly prices in [`tenant.yaml`](tenant.yaml), read from the portal's pricing API
(EUR): 2 × `standard.micro` at 0.0073 EUR/h, 2 × `standard.small` at 0.0233 EUR/h,
40 GB of local disk and 30 GB of block storage at 0.00014 EUR/GB/h, one SKS Pro control
plane at 0.055 EUR/h (Starter is free) — about **0.13 EUR per hour** of tenant life.
Budget: 20 minutes wall clock, otherwise NO-GO.

## Measured

<!-- qualification:measured -->
Run of 2026-09-09 (`GO`), on the organisation the environment names:

| | |
|---|---|
| Resources | 40 in the full plan, 39 applied (`swiss` is plan-only), plus 3 ressource(s) hors Terraform : pepin-qual-<org>-public, pepin-qual-<org>-unversioned, pepin-qual-<org>-hardened |
| `apply` | 40.6 s |
| `scan --live` × 5 formats + seal | 11.3 s, exit code 1 in every format |
| bundle | `verify --re-derive` passes; the bundle altered by one byte is refused |
| `scan --terraform` | exit code 1 |
| `destroy` | 50.7 s — terraform destroy rc=0 ; état vide ; 2 fichier(s) d'état purgé(s) |
| proof of destruction | 15 familles listées, aucune ressource du tenant, aucun delta avant/après |
| comparison | live: 46 results, 0 difference; terraform: 37 results, 0 difference |
| falsification | 3 expectations broken in memory on each source, every one NO-GO |
| wall clock | 183.8 s (budget 20 min) |
| cost estimate | 0.0037 EUR (0.029 h × 0.126 EUR/h) |

Deviations outside the tenant reported by the live scan and not gated: 7
(the organisation's own user without MFA, the role of the qualifying key and the two
`sks-ccm-*` roles of the clusters, with no source-IP nor lifetime bound — #213).
<!-- /qualification:measured -->

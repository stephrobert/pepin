> 🇬🇧 English · [🇫🇷 Français](README.fr.md)

# Scaleway qualification tenant

40 resources in the plan, **29 of them applied**, in `fr-par` / `fr-par-1`, on the
project pinned in [`expected.yaml`](expected.yaml). One fault per resource, one
counterexample per control. Applied, scanned, sealed, destroyed and compared by
`PEPIN_GATE_LIVE=1 mise run qualify` (see [the generic README](../README.md)).

**Two passes that do not measure the same thing.** A Scaleway live scan collects
five types (`access_key`, `iam_user`, `compute_instance`, `security_group_rule`,
`object_storage_bucket`, contract in `providers/scaleway.yaml`); a plan carries
eight (plus `managed_database`, `security_group`, `iam_policy`, `network`).
Provisioning a managed database on the real account would cost money no live scan
measures, so the resources only a plan reads are behind `terraform_only_resources`
(default `true`): the runner applies with `false` and scans the full plan with
`--terraform`. `expected.yaml` pins the two sources separately.

## Prerequisite: a dedicated project, never the default one

This stack applies **deliberately exposed** resources — SSH open to the whole internet,
public buckets, permissive policies — and its proof of destruction relies on a
before/after delta for what carries no tag. A **third-party** resource moving during the
run falsifies that delta, in either direction.

At Scaleway, resources live in a **Project** and never leave it, while IAM and quotas
stay at the **Organization** level. The project is therefore the only boundary this
tenant can use — and the **default project is not one**: it carries the Organization's
ID, cannot be deleted or transferred, and is where everything lands when nobody chose.

Measured on 2026-09-10: the default project held a `pavois-repro-…` VM started two hours
earlier, from other work. The run could only proceed after emptying the whole account,
which is not a procedure. Hence issue #240, and the refusal the `identity` hook now
raises.

```bash
scw account project create name=pepin-qualification     description="Pepin qualification tenant — emptied between runs"
scw config set default-project-id=<PROJECT_ID>
export PEPIN_QUAL_SCW_PROJECT=<PROJECT_ID>
```

The project is **not recreated** on each run — it is **emptied**. A reused project keeps
its consumption history, consumes the 25-project-per-organization quota only once, and
needs its VPC created only once (a project created since 13 May 2025 no longer receives
a default one).

## What the stack creates

| Family | Count | Faulty resource → control | Counterexample (must stay silent) |
|---|---:|---|---|
| IAM applications | 1 (+1 plan-only) | — (`keys` carries the keys and no policy; `admin`, plan-only, carries the policies and no key: the escalation path is never a live credential) | — |
| IAM API keys | 2 (+1 plan-only) | `expired` (expiry a few minutes after apply; the runner waits for it before scanning, and an expired key **stays listed** by the API) → `iam_accesskey_expiration_set` (high) · `no_expiry`, **plan-only**: the organisation refuses to create it — `organization security settings require an expiration date for API keys`, measured 2026-09-09 — → the same control (critical) | `expiring` (expiry at D+2, set by the runner) |
| IAM policies | 2, plan-only | `iam_manager` (PermissionSet `IAMManager`, organisation scope) → `iam_policy_no_privilege_escalation` | `read_only` (`InstancesReadOnly`, project scope) |
| Security groups | 9 | `ssh_open` → `…tcp_port_22` · `rdp_open` → `…tcp_port_3389` · `db_open` (5432) → `…high_risk_tcp_ports` · `snmp_open` (UDP 161) → `…high_risk_udp_ports` · `any_open` → `…all_ports` **and the four above** · `quartet` (four `/2`) → `…tcp_port_22` by evasion · `egress_any` → `network_securitygroup_unrestricted_egress` · `default_accept` → `network_securitygroup_default_deny` | `hardened`: drop by default, SSH from `10.0.0.0/8`, HTTPS from anywhere, egress TCP 443 only |
| Flexible IPs | 2 | — | — |
| Servers (DEV1-S) | 5 | `exposed` (public IP + `ssh_open`) → `compute_instance_public_ip_with_open_securitygroup` · `untagged` → `governance_resource_required_tags` · `secrets` (cloud-init with a password) → `compute_instance_no_secrets_in_user_data` | `private` (same open group, no public IP) · `hardened` (public IP, restrictive group) |
| VPC | 1, plan-only | — | — |
| Private networks | 2, plan-only | `undocumented` → `network_documented` | `documented` (Owner, Project, Env) |
| Buckets (empty) | 7 | `public` (ACL `public-read`) and `policy` (bucket policy `Principal: *`) → `objectstorage_bucket_public_access` · `unversioned` → `…versioning_enabled` **and** `…object_lock_enabled` (no lock without versioning) · `unlocked` → `…object_lock_enabled` · `sensitive` (`classification=confidential`, no SSE-KMS) → `…kms_encryption` · every bucket without SSE → `objectstorage_bucket_default_encryption`, a control the referentiel declares for Outscale only (**known defect**, pinned as such, #192) | `hardened` (private, versioned, locked, classified public) · `encrypted` (SSE-ONE configured: silent on default encryption) |
| Managed databases (db-dev-s) | 2, plan-only | `exposed` (ACL `0.0.0.0/0`, no encryption at rest, backups disabled) → `database_service_not_open_to_internet`, `database_encryption_at_rest_enabled`, `database_backup_enabled` | `hardened` (encrypted on a Block volume, daily backups, ACL `10.0.0.0/8`) |

Every taggable resource carries the tag `pepin-qual`, every name starts with
`pepin-qual-`, and the bucket names embed the first eight characters of the
project id: that is what the proof of destruction filters on.

## What the upstream provider is known to leave behind on `destroy`

Read before choosing the resource types, in `scaleway/terraform-provider-scaleway`
and in the maintainer's own field notes
([Terraform sur Scaleway : provisionner, et surtout détruire](https://blog.stephane-robert.info/docs/cloud/scaleway/iac/terraform-provider-scaleway/),
2026-09-09). What each one changed in this stack:

| Upstream | State | What it does | What this tenant does about it |
|---|---|---|---|
| [#4338](https://github.com/scaleway/terraform-provider-scaleway/issues/4338) `scaleway_instance_private_nic`: destroy fails with `412 Can't delete a private network interface attached to a server` since v2.81.0 (API v2 migration) | **open**, confirmed on 2.82.0 by Pépin's maintainer on 2026-09-09 (2.80.0: exit 0, 7 s, 0 left; 2.81.0 and 2.82.0: exit 1, 4/4 left). Stopping the server or targeting it does not help; Terraform cannot invert the order. | Deadlocks every destroy of a server with an attached private NIC. | **No private NIC at all.** The private networks exist alone, nothing is attached to them. Provider pinned exactly at `2.82.0` and the lockfile committed; the rescue path is `scw instance private-nic delete` (v1 route) should one ever be added. Consequence: the two-NIC machine of the external audit (#E) cannot be exercised on Scaleway today. |
| [#2853](https://github.com/scaleway/terraform-provider-scaleway/issues/2853) instance: SBS volumes not deleted on destroy | closed 2024-12-19 by `fix(instance_server): delete_after_termination for sbs volumes` | An orphan Block volume, billed, that nobody finds again. | Root volumes on the range's **local** storage, `delete_on_termination = true` written rather than assumed, no additional volume; Instance **and** Block volumes are in the leftovers listing. |
| [#2869](https://github.com/scaleway/terraform-provider-scaleway/issues/2869) a bucket with objects cannot be deleted | closed 2025-01-10 (`force_destroy` empties the bucket since 2.49.0) | A non-empty, versioned or locked bucket blocks its own deletion. | Buckets stay **empty**, `force_destroy = true` everywhere. |
| [#2821](https://github.com/scaleway/terraform-provider-scaleway/issues/2821) the security group created by the product blocks the destruction | closed 2025-02-14: "explicitly create a security group and use it with the instance" | A server that landed in the default group holds it. | Nine **explicit** groups, every server attached to one explicitly. |
| [#4262](https://github.com/scaleway/terraform-provider-scaleway/issues/4262) orphaned IPv4 when `ip_ids` is set on a load balancer | open | An untracked flexible IP, billed. | No load balancer; the two flexible IPs are Terraform resources, and the rescue cleanup deletes servers with `with-ip=true` (the option the field notes single out). |
| [#2125](https://github.com/scaleway/terraform-provider-scaleway/issues/2125) cannot destroy `scaleway_vpc_private_network` · [#3243](https://github.com/scaleway/terraform-provider-scaleway/issues/3243) `precondition failed: resource is still in use` | closed (provider upgrade; no reproduction) | A private network that still has something attached. | Explicit VPC, nothing ever attached to the private networks. |
| [#4320](https://github.com/scaleway/terraform-provider-scaleway/issues/4320) ephemeral `iam_api_key` piles up unmanaged keys | open | Keys recreated at each apply, unknown to the state. | Plain managed `scaleway_iam_api_key`; the IAM listing filters on the tenant's applications and on the description prefix. |
| [#2426](https://github.com/scaleway/terraform-provider-scaleway/issues/2426) `account_project`: 412 on delete | closed | A project that refuses to go. | No project is created: the tenant lives in the pinned project. |
| b_ssd volumes are no longer supported (field notes) | — | `scaleway_instance_volume` is refused by the API. | No Instance volume; if a data volume were ever needed it would be `scaleway_block_volume`. |

The proof of destruction is the field notes' eleven listings, plus the IAM and
Object Storage families they do not need: [`hooks.py inventory`](hooks.py) lists
servers, security groups, flexible IPs, Instance volumes, Instance snapshots,
Instance images, Block volumes, Block snapshots, managed databases, their backups
and snapshots, private networks, VPCs, load balancers, public gateways, IAM
applications, policies and API keys, and buckets, through the same API routes the
`scw` CLI calls. The runner requires the tenant subset of every family to be empty
**and** the whole listing to equal the one taken before apply.

## What this tenant cannot exercise, and why

- `governance_resource_region_in_eu`: every Scaleway region is in the EU, no
  deviation is constructible; only the `pass` is proven.
- `iam_user_mfa_enabled`: an IAM user is a real person invited into the
  organisation, out of a disposable tenant's scope. The control is observed on the
  organisation's existing users, and `expected.yaml` only pins that it **concludes**.
- `compute_instance_has_security_group`: a Scaleway server always has a group.
- `iam_no_root_access_key`: `root_owned` is not derived yet by the collector; the
  control stays `not-evaluated` on both sources, and that is pinned.
- Nine controls are `not-evaluated` on the **live** source (#195) because the collector does
  not read the data yet (security group default policy, private networks, managed
  databases, server user-data, IAM policies): each one is pinned as such in
  `expected.yaml`, so a collector that starts reading them moves a verdict on
  purpose, with a CHANGELOG line.
- The two-NIC machine (external audit #E): see #4338 above.

## Cost and budget

Hourly rates in [`tenant.yaml`](tenant.yaml) for what is applied: 5 × DEV1-S at
0.008976 EUR/h and 2 flexible IPv4 at 0.004 EUR/h — about **0.053 EUR per hour** of
tenant life; everything else applied (IAM, security groups, empty buckets) is free,
and the managed databases (0.0347 EUR/h each) are never provisioned. The runner
multiplies the rates by the apply→destroy duration and writes the estimate to the
report. Budget: 25 minutes wall clock, or NO-GO.

## Measured

<!-- qualification:measured -->
Run of 2026-09-09 (`GO`), on the project pinned in `expected.yaml`:

| | |
|---|---|
| Resources | 40 in the plan, 29 applied |
| `apply` | 11.2 s |
| wait for the expired key's deadline | 184.9 s |
| `scan --live` × 5 formats + seal | 37.3 s, exit code 1 in every format |
| bundle | `verify --re-derive` passes; the bundle altered by one byte is refused |
| `scan --terraform` | exit code 1 |
| `destroy` | 18.3 s, `terraform destroy` exit 0, state empty, state files purged |
| proof of destruction | 19 familles listées, aucune ressource du tenant, aucun delta avant/après |
| comparison | live: 40 results, 0 difference; terraform: 34 results, 0 difference |
| falsification | 3 expectations broken in memory on each source, every one NO-GO |
| wall clock | 259.9 s (budget 25 min) |
| cost estimate | 0.0037 EUR (0.07 h × 0.0529 EUR/h) |

Informational lines of the run (not gated): `iam_user_mfa_enabled` fails on an
organisation user (out of the tenant's perimeter), and
`objectstorage_bucket_default_encryption` is pinned as a known defect (#192, see
`expected.yaml`).
<!-- /qualification:measured -->

> 🇬🇧 English · [🇫🇷 Français](README.fr.md)

# Outscale qualification tenant

79 Terraform resources plus 6 OOS buckets and one inline EIM policy created by the
`extra` hook, in `eu-west-2` / `eu-west-2a`, on the account the environment names
(`PEPIN_QUAL_OSC_ACCOUNT`, confirmed by `ReadAccounts` before any apply). One fault
per resource, one counterexample per control. Applied, scanned, sealed, destroyed
and compared by `PEPIN_GATE_LIVE=1 PROVIDER=outscale mise run qualify` (see
[the generic README](../README.md)).

Unlike Scaleway, an Outscale live scan reads **everything this stack creates** (16
types in the `providers/outscale.yaml` contract): the two passes see the same tenant,
and this is where the cases Scaleway cannot exercise live — the two-NIC machine, the
root access key, the `iam_policy_*` family through EIM documents, the snapshots, the
public OMI, the load balancers.

## What the stack creates

| Family | Count | Faulty resource → control | Counterexample (must stay silent) |
|---|---:|---|---|
| Security groups (all in the Net) | 10 (+ 14 rules) | `ssh_open` → `…tcp_port_22` · `rdp_open` → `…tcp_port_3389` · `db_open` (5432) → `…high_risk_tcp_ports` · `snmp_open` (UDP 161) → `…high_risk_udp_ports` · `any_open` → `…all_ports` **and the four above** · `quartet` (four `/2`) → `…tcp_port_22` by evasion · `egress_any` (explicit `-1 → 0.0.0.0/0` outbound) → `network_securitygroup_unrestricted_egress` · `net_ssh_open` → `…tcp_port_22` (the secondary card's group) · the Net's **`default` group**, created by the product and read through a data source → `network_securitygroup_default_restrict_traffic` (factory rule) and `unrestricted_egress` (its default outbound rule) | `hardened` (SSH from `10.0.0.0/8`, HTTPS from anywhere, egress TCP 443) · `net_closed` · every group has `remove_default_outbound_rule = true` |
| VMs (`tinav6.c2r4p2`) | 8 | `exposed` (public IP + `ssh_open`) → `compute_instance_public_ip_with_open_securitygroup` · `two_nics` (primary card private and closed, **secondary** card public in `net_ssh_open`) → the same control, through the NIC pairing of #187 · `untagged` → `governance_resource_required_tags` · `secrets` (cloud-init with a password, base64 `UserData`) → `compute_instance_no_secrets_in_user_data` · `unprotected` (`Env=prod`, no deletion protection) → `compute_instance_deletion_protection` | `private` (same open group, no IP) · `hardened` (public IP, restrictive group, `Env=prod`, **protected**) · `two_nics_inverse` (primary public and closed, secondary private and open: flattened it would look exposed, paired it is not) · `untagged` also gets an *inconclusive* on deletion protection (no readable environment, ADR-0015) |
| Secondary NICs, public IPs | 2 + 4 | — | `outscale_nic` + `outscale_nic_link`, never the `nics` block (see below) |
| Volumes | 2 (+ 8 root) | `unsnapshotted` and the eight root volumes → `blockstorage_volume_snapshots_exist` | `snapshotted` (a completed private snapshot, also the source of both OMIs) · root volumes tagged through `outscale_tag` |
| Snapshots | 2 | `public` (`GlobalPermission`) → `blockstorage_snapshot_not_public` | `private` |
| Images (OMI) | 2 | `public` (global launch permission) → `compute_image_not_public` | `private` |
| Nets, subnets | 2 + 2 | `undocumented` → `network_documented` · `autoip` (`MapPublicIpOnLaunch`) → `network_subnet_no_public_ip_by_default` | `documented` · `main` · a **peering between the two Nets** (same account) is the counterexample of `network_peering_cross_organization` |
| Load balancers (LBU) | 2 | `http` (HTTP listener only, no access log) → `loadbalancer_ssl_listeners`, `loadbalancer_logging_enabled` · `tcp443` (TCP 443 listener) → *inconclusive* on `loadbalancer_ssl_listeners` (TLS passthrough candidate), fails `loadbalancer_logging_enabled` | — (an enabled access log needs an OOS target bucket; not built) |
| EIM | 1 user, 3 keys, 5 policies | `no_expiry` → `iam_accesskey_expiration_set` · `root` (a key of the calling account) → `iam_no_root_access_key` · `admin` (`Action: *`) → `iam_policy_no_administrative_privileges` · `wildcard`, `notaction`, `escalation` (`Resource: *`) → `iam_policy_no_wildcard_resource` · `notaction` → `iam_policy_no_notaction_notresource` · `escalation` (`CreateAccessKey`, `LinkPolicy`) → `iam_policy_no_privilege_escalation` | `expiring` (expiry at D+2) · `scoped` (a named OOS resource) · every key is created **INACTIVE**: the rules do not read the state, and no usable secret circulates |
| API access rules | 2 | `public` (`0.0.0.0/0`) → `iam_apiaccessrule_no_public_cidr` | `private` (`10.0.0.0/8`); rules are OR-ed, none locks anyone out |
| OOS buckets (`extra` hook, `aws s3api` on `oos.eu-west-2.outscale.com`) | 6 | `public` (ACL) and `policy` (`Principal: *`) → `objectstorage_bucket_public_access` · `unversioned` → `…versioning_enabled` and `…object_lock_enabled` · `unlocked` → `…object_lock_enabled` · `unencrypted` (no SSE) → `objectstorage_bucket_default_encryption` | `hardened` (private, locked, SSE, tagged) |
| Inline EIM policy (`extra` hook, `PutUserPolicy`) | 1 | `pepin-qual-inline-admin` (`Action: *` on the tenant's user, the founding incident of ADR-0006) → `iam_policy_no_administrative_privileges` | — |

Every taggable resource carries the tag `pepin-qual`, every name starts with
`pepin-qual-`, and the bucket names embed the first eight digits of the account id.

## What the upstream provider is known to leave behind on `destroy`

Read before choosing the resource types, in `outscale/terraform-provider-outscale`
(1.8.0 pinned, lockfile committed). What each one changed in this stack:

| Upstream | State | What it does | What this tenant does about it |
|---|---|---|---|
| [#88](https://github.com/outscale/terraform-provider-outscale/issues/88) deletion_protection: a protected VM makes destroy fail (`Error deleting the VM`) | open, confirmed by the maintainers 2024-08 ("you must apply the change first") | The protected counterexample would block the whole destroy. | `deletion_protection = var.deletion_protection`, and `tenant.yaml` declares `pre_destroy_vars: {deletion_protection: false}`: the runner **applies** the unprotection before destroying. The rescue cleanup does `UpdateVm DeletionProtection=false` first. |
| [#28](https://github.com/outscale/terraform-provider-outscale/issues/28) / [#137](https://github.com/outscale/terraform-provider-outscale/issues/137) `outscale_nic_private_ip`: 400/409 on destroy, NIC and primary IP not deleted | open | Secondary private IPs deadlock the NIC's destroy. | No `outscale_nic_private_ip`: one primary private IP per NIC. |
| [#778](https://github.com/outscale/terraform-provider-outscale/issues/778), [#424](https://github.com/outscale/terraform-provider-outscale/issues/424), [#50](https://github.com/outscale/terraform-provider-outscale/issues/50), [#448](https://github.com/outscale/terraform-provider-outscale/issues/448) `nics` / `primary_nic` blocks in `outscale_vm`: VM replaced on every plan, NIC "forgotten" | #778 closed 2026-07 (won't fix: "use `primary_nic` with a separate `nic_link`") | A second card declared inline is not destroyable cleanly. | Secondary cards are `outscale_nic` + `outscale_nic_link` (+ `outscale_public_ip_link` on the NIC), the workaround the maintainers recommend. |
| [#622](https://github.com/outscale/terraform-provider-outscale/issues/622) a security group still attached cannot be destroyed | closed 2026-02 | A group referenced by a VM or LBU holds. | Every group is attached only to Terraform VMs, destroyed first by dependency. |
| [#663](https://github.com/outscale/terraform-provider-outscale/issues/663) LBU stickiness policy destroy fails | closed | — | No stickiness policy. |
| [#6](https://github.com/outscale/terraform-provider-outscale/issues/6) public IP released too early | closed 2022 | — | `outscale_public_ip` + `outscale_public_ip_link` resources, ordered by dependency. |
| Measured here: **`terraform destroy` returns 0 while a net peering survives** | to report upstream | An accepted `outscale_net_peering` (+ `_acceptation`) was still `active` after a successful destroy. | The rescue cleanup deletes it (`DeleteNetPeering`) and the proof of destruction catches it. |
| Measured here: **deletions are asynchronous** | — | 26 s after a successful destroy, both LBUs, ten volumes and two snapshots were still listed; two minutes later they "did not exist". | The proof of destruction **re-lists until empty**, bounded by `settle_seconds` (240 s by default): what survives the delay is a leftover. |
| Measured here: a public-cloud security group cannot drop its default outbound rule (`remove_default_outbound_rule` needs `net_id`) | — | Every public-cloud group carries `unrestricted_egress` by construction. | Every group lives in the Net. |
| Measured here: OOS sets versioning on an Object-Lock bucket and **freezes it** (`InvalidBucketState` on `PutBucketVersioning`) | — | — | The hook enables versioning explicitly only on `unlocked`. |
| Measured here (three runs): **`LinkPublicIp` by `VmId` is refused (400 `InvalidResource`) on a VM that already has two cards** | — | The `two_nics_inverse` counterexample could not be built once the second card was attached first. | The public IP is linked while the VM has its primary card only; the second card's `outscale_nic_link` depends on that link. |
| Measured here: **`DeleteImage` does not delete the snapshot an OMI created from a VM implies** | — | Two orphan snapshots after a successful destroy, caught by the before/after diff. | Both OMIs are built from the tenant's own managed snapshot (`block_device_mappings`), so nothing implicit is created. |

The proof of destruction lists 20 families through the OAPI (VMs, NICs, public IPs,
security groups, volumes, snapshots, images, Nets, subnets, internet services, NAT
services, route tables, peerings, LBUs, keypairs, API access rules, users, policies,
access keys, buckets), filtered on the tenant's tag and prefix, and diffs the whole
listing with the one taken before apply. `octl`, the provider's reference CLI, is
what a maintainer replays by hand:

```bash
octl iaas vm list --filter 'Tags[].Key:pepin-qual' -o table
octl iaas securitygroup list -o table --filter 'SecurityGroupName:pepin-qual-'
octl iaas publicip list -o table
octl iaas volume list -o table
octl iaas snapshot list -o table
octl iaas image list -o table --filter 'ImageName:pepin-qual-'
octl iaas net list -o table
octl iaas netpeering list -o table
octl iaas loadbalancer list -o table
octl iaas user list -o table
octl iaas policy list -o table
octl storage bucket list -o table
```

Never `octl profile current`: it prints the secret key in clear. And read the states:
for a while after a destroy, `octl iaas vm list` still shows the tenant's VMs as
`terminated` and `octl iaas netpeering list` its peering as `deleted` (32 and 4 after
four runs). They are terminal entries, not resources; the API inventory of the proof
excludes those states.

## What this tenant cannot exercise, and why

`tenant.yaml`, `not_exercised:` — a resource outside the EU (an Outscale account
lives in one region), a key older than 90 days, a cross-account peering, the account's
API access policy (MFA enforcement, key expiration ceiling: forcing them would lock
the qualifying key out), a tenant without any API access rule, an enabled LBU access
log, a `default` group emptied of its factory rule, OKS clusters.

## Cost and budget

Hourly rates in [`tenant.yaml`](tenant.yaml): 8 × `tinav6.c2r4p2` at 0.09 EUR/h (the type
the maintainer gave for this account), 2 LBUs at 0.03 EUR/h, 4 public IPs, ~100 GB of
volumes — about **0.82 EUR per hour** of tenant life. Budget: 40 minutes wall clock,
or NO-GO.

## Measured

<!-- qualification:measured -->
Run of 2026-09-09 (`GO`), on the account the environment names:

| | |
|---|---|
| Resources | 79 in the plan, all applied, plus 7 ressource(s) hors Terraform : pepin-qual-30044641-public, pepin-qual-30044641-policy, pepin-qual-30044641-unversioned, pepin-qual-30044641-unlocked, pepin-qual-30044641-unencrypted, pepin-qual-30044641-hardened, pepin-qual-user/pepin-qual-inline-admin |
| `apply` | 210.3 s |
| `scan --live` × 5 formats + seal | 50.8 s, exit code 1 in every format |
| bundle | `verify --re-derive` passes; the bundle altered by one byte is refused |
| `scan --terraform` | exit code 1 |
| `destroy` | 155.3 s — pré-destroy (deletion_protection=False) rc=0 ; terraform destroy rc=0 ; état vide ; 2 fichier(s) d'état purgé(s) |
| proof of destruction | 20 familles listées, aucune ressource du tenant, aucun delta avant/après (après 40s de suppressions asynchrones) |
| comparison | live: 145 results, 0 difference; terraform: 54 results, 0 difference |
| falsification | 3 expectations broken in memory on each source, every one NO-GO |
| wall clock | 511.5 s (budget 40 min) |
| cost estimate | 0.0989 EUR (0.121 h × 0.815 EUR/h) |

Out-of-tenant deviations reported by the live scan and not gated: 74 (the
account's own keys, policies, API access rules, buckets and default security groups).
<!-- /qualification:measured -->

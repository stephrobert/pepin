> 🇬🇧 English · [🇫🇷 Français](README.fr.md)

# Exoscale qualification tenant (Terraform source only)

**No Exoscale account is available.** This tenant is therefore qualified in
`--plan-only` mode: 37 resources are planned, nothing is created, and
[`expected.yaml`](expected.yaml) pins what `pepin scan exoscale --terraform` returns
on that plan — the path a user follows without an account
([ADR-0012](../../docs/adr/0012-aucun-identifiant-en-ci.md) prefers it). That already
has value: it fixes the Terraform verdicts of 26 controls, with counterexamples.

What it does **not** do, and says so (`tenant.yaml`: `live: unavailable`): apply,
`scan --live`, seal a bundle, destroy, prove the destruction. `mise run qualify` on this
tenant is refused; `mise run qualify:plan` (`PROVIDER=exoscale`) is the only mode.
[`hooks.py`](hooks.py) provides the placeholder credentials a plan needs and refuses
everything else. Reading the `exoscale/terraform-provider-exoscale` issues on `destroy`
is due before the first apply, not before the first plan.

```bash
PROVIDER=exoscale mise run qualify:plan
```

## What the plan carries

| Family | Count | Faulty resource → control | Counterexample (must stay silent) |
|---|---:|---|---|
| Security groups | 9 (+ 15 rules) | `ssh_open` → `…tcp_port_22` · `rdp_open` → `…tcp_port_3389` · `db_open` (5432) → `…high_risk_tcp_ports` · `snmp_open` (UDP 161) → `…high_risk_udp_ports` · `any_open` (all TCP and all UDP ports) → the four above · `quartet` (four `/2`) → `…tcp_port_22` by evasion · `undocumented_flow` (a rule without description, on port 443) → `network_flow_matrix_documented` | `hardened` (SSH from `10.0.0.0/8`, HTTPS from anywhere, egress TCP 443, every flow described) |
| Private networks | 2 | `undocumented` → `network_documented` | `documented` (Owner, Project, Env, description) |
| Instances | 4 | `untagged` → `governance_resource_required_tags` · `secrets` (cloud-init with a password) → `compute_instance_no_secrets_in_user_data` · `swiss` (zone `ch-gva-2`) → `governance_resource_region_in_eu` (low: outside the EU, inside the European trusted area) | `hardened` (labelled, `de-fra-1`, plain cloud-init) |
| Block storage volume | 1 | — (encrypted by construction: `blockstorage_volume_encryption` passes) | — |
| SKS clusters | 2 | `weak` (`starter`, no auto-upgrade, audit disabled) → `kubernetes_cluster_control_plane_highly_available`, `…auto_upgrade_enabled`, `…audit_logging_enabled` | `strong` (`pro`, auto-upgrade, audit endpoint) |
| IAM roles | 4 | `admin` (`allow` by default) → `iam_role_no_admin_privileges` · `no_source_ip` → `iam_role_source_ip_restricted` · `unbounded` (no `duration(`) → `iam_role_key_lifetime_bounded` | `restricted` (`deny` by default, `source_ip`, `duration(`) |
| Provider | — | `governance_provider_sovereignty` fails on `exoscale` itself: head office in Switzerland, decisive non-EU capital control — facts of the descriptor, not of the tenant | — |

## What this tenant cannot exercise, and why

- **`network_securitygroup_allow_ingress_from_internet_to_all_ports` and
  `network_securitygroup_unrestricted_egress` cannot fire on Exoscale**, on either
  source: a security group rule has no "all protocols" value (`tcp`, `udp`, `icmp`,
  `icmpv6`, `ah`, `esp`, `gre`, `ipip` only, provider 0.71.0 and the API), and both
  common rules require `protocol == all`. `any_open` opens every TCP and UDP port and
  the four other ingress controls fire; these two are pinned `pass` by absence, which
  the referentiel and the coverage matrix should stop presenting as coverage (#202).
- **`network_flow_matrix_documented` returns `pass` on a plan whose `undocumented_flow`
  rule has no description**: a description that is not written is *computed* on a
  plan, so the attribute is not projected, the capability guard stays silent, and the
  assessment concludes "compliant" — the false green [ADR-0006](../../docs/adr/0006-jamais-un-pass-non-prouve.md)'s
  intersection exists to prevent. Pinned as a **known defect** (#201); the fix will
  turn this gate NO-GO on purpose.
- The public IP of an instance and the attachment state of a volume are computed at
  apply: `compute_instance_public_ip_with_open_securitygroup` and
  `blockstorage_volume_snapshots_exist` wait for the live half.
- SOS buckets: the provider has no bucket resource (`exoscale_sos_bucket_policy`
  only); they will be created by an `extra` hook, as on Outscale, when an account
  exists.
- IAM users are real people, not Terraform resources.

## Measured

Plan-only run (`PROVIDER=exoscale mise run qualify:plan`): 37 resources planned,
`scan --terraform` exit code 1, 26 controls pinned on the Terraform source, GO, with
one expectation broken in memory turning NO-GO. Nothing created, nothing to destroy,
no cost.

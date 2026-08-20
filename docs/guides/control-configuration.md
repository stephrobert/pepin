> 🇬🇧 English · [🇫🇷 Français](control-configuration.fr.md)

# Configuring controls — and what a relaxation costs you

Some controls have to be adjustable. A tagging convention is not a standard; a snapshot
window is an operational decision; a generic secret heuristic is noisier than a PEM block.
A control that cannot be adjusted gets disabled, and a disabled control measures nothing.

But every setting is a handle that can manufacture green. Loosen a threshold, drop a required
tag, stretch a freshness window — and keep displaying the same CIS or SecNumCloud mapping. A
`pass` obtained by lowering the bar is exactly the unproven `PASS` this project refuses to
produce.

So Pépin ties configuration to normative mapping. **You can lower the bar. You cannot lower
it and keep the badge.**

## One file, two sections

There is a single policy file. It carries the control settings and the exemptions, because
both answer the same question — *what this organisation knowingly accepts* — and both are
reviewed at the same moment, by the same people. Two files would have two review cycles, and
anyone made to maintain three policy files will let two of them drift.

```yaml
# pepin-policy.yaml
controls:
  tagging:
    required_tags: [CostCenter, Project, Env, Owner]
    network_required_tags: [Owner, Project, Env]
    aliases:
      Owner: [Owner, team, responsible, contact]
    resource_types: [compute_instance, blockstorage_volume, object_storage_bucket]
  snapshots:
    max_age_days: 7
    accepted_states: [completed, created]
  secrets:
    min_confidence: low

exceptions:
  - control: objectstorage_bucket_public_access
    resource: public-assets
    justification: "Public distribution bucket, non-sensitive content"
    expires_at: 2026-12-31
    owner: platform-security
    approved_by: security@example.org
```

```console
$ pepin scan scaleway --terraform plan.json --policy pepin-policy.yaml
```

`--exceptions` is the historic name of the same file and reads the same schema: an existing
invocation keeps working, and an existing exemptions file can grow a `controls:` section
without changing a command line. **The two flags are mutually exclusive** — accepting two
different policy files is a guarantee that one of them will drift.

Every section is optional. An absent section leaves the default profile untouched: a partial
policy file relaxes only what it names.

## The default profile is a recommendation, not a standard

No tagging convention is authoritative, and Pépin does not pretend otherwise. What the
default profile encodes are the **questions an inventory must be able to answer** — who pays,
for what, at which stage, who is accountable — never the exact words.

| Setting | Default | What it means |
|---|---|---|
| `tagging.required_tags` | `CostCenter, Project, Env, Owner` | tags required on a billable resource |
| `tagging.network_required_tags` | `Owner, Project, Env` | tags required on a network (mapping) |
| `tagging.aliases` | see below | other spellings a logical name accepts |
| `tagging.resource_types` | 8 types, listed below | where tagging is required |
| `snapshots.max_age_days` | `7` | freshness window of a snapshot |
| `snapshots.accepted_states` | `completed, created` | native states of a usable snapshot |
| `secrets.min_confidence` | `low` | report everything, including generic heuristics |

**The comparison ignores case and separators.** `cost-center`, `cost_center`, `Cost Center`
and `CostCenter` are the same requirement — a tool that cries wolf over typography gets
switched off. On top of that, aliases widen each logical name:

| Logical name | Accepted spellings |
|---|---|
| `CostCenter` | `CostCenter`, `cost-center`, `cc`, `billing-code`, `billing` |
| `Project` | `Project`, `app`, `application`, `service` |
| `Env` | `Env`, `environment`, `stage` |
| `Owner` | `Owner`, `team`, `responsible`, `contact` |

The names in the table are **display names**: they are what a finding message lists as
missing. Matching goes through the normalized aliases.

**Which resource types are in scope, and why.** The criterion is *billable and taggable*: a
resource that costs money without a known owner is an orphan cost as much as an orphan risk.

- In scope: `compute_instance`, `blockstorage_volume`, `blockstorage_snapshot`,
  `compute_image`, `load_balancer`, `object_storage_bucket`, `managed_database`,
  `kubernetes_cluster`.
- Out of scope, with reasons: `network`, `subnet`, `network_peering`, `security_group*` (not
  billed in their own right; a network has its own mapping requirement, CLD-NET-5);
  `iam_*`, `access_key`, `api_access_*` (neither billed nor tag-bearing);
  `governance_provider` (a synthetic resource, not a tenant resource); `k8s_*` (in-cluster
  scope, outside cloud posture).

## What a relaxation costs

Each mapping in the reference carries the configuration constraints under which it holds:

```yaml
- code: blockstorage_volume_snapshots_exist
  scsl: [CLD-STO-3]
  frameworks:
    secnumcloud_3_2: ["12.5"]
    cis_controls_v8: ["11.2"]
    iso_27001_2022: ["A.8.13"]
  config_requise:
    - parametre: snapshots.max_age_days
      contrainte: au_plus_le_defaut
    - parametre: snapshots.accepted_states
      contrainte: sous_ensemble_du_defaut
```

Four constraint kinds, each naming the side of the default where the promise survives:

| Constraint | Holds while | Relaxing it means |
|---|---|---|
| `au_plus_le_defaut` | the value stays at or below the default | a value beyond the default silences what the requirement asked to see |
| `superset_du_defaut` | the value contains at least the default | dropping a member means no longer checking what the requirement asks |
| `sous_ensemble_du_defaut` | the value stays within the default | widening means accepting what the requirement rejects |
| `au_moins_aussi_strict_que_le_defaut` | every requirement of the default profile still has a counterpart at least as strict | a requirement was dropped, or what it accepts was widened |

The last one is what the tagging profile uses, and it is not the same as comparing names.
The default requires *"the environment question is answered by one of `env`, `environment`,
`stage`"*. An organisation that writes `environment` accepts **fewer** spellings, so it
requires more: that is a tightening. Comparing names would have flagged it as relaxed and
taken away the normative mapping of someone who narrowed their own convention — the most
expensive false positive there is, because it punishes the right behaviour.

**Tightening is not relaxing.** Requiring one more tag, shortening the window, narrowing the
accepted states or the accepted spellings: the mapping holds and nothing is reported. The
constraint does not say *do not touch anything*, it says which side of the default the
promise survives on.

When the effective configuration falls outside a constraint, the control is **relaxed**, and
five things happen at once:

1. the result **loses its normative references** in the assessment — it no longer claims to
   cover CLD-STO-3, CIS 11.2 or ISO A.8.13;
2. the result carries `labels.config_relaxed`, `labels.config_relaxed_detail` and
   `labels.references_dropped`;
3. the evidence (`evidence.observed`, hence the OSCAL) states the relaxation in words;
4. the terminal prints a `RELAXED CONFIGURATION` block naming the setting, the default, the
   effective value and the mappings dropped;
5. `--format json` publishes `config.relaxations` and `config.dropped_references`, and a
   sealed bundle gains `config.json` plus a `config` entry in its manifest — both covered by
   `checksums.txt`, so the bundle digest depends on the settings.

The verdict banner changes too, and `--strict` returns exit code `3`. A pipeline that sells
compliance must not return `0` on a control it lowered itself.

The status does **not** change. The control was genuinely evaluated — against a lower bar. We
remove the one thing that stopped being true, the mapping, and keep the measurement.

## Setting the blocking threshold for secret detection in CI

Secret detection carries a confidence level on every finding, in `labels.confidence`:

| Level | Basis | Example |
|---|---|---|
| `high` | confirmed by its shape | `-----BEGIN … PRIVATE KEY-----` |
| `medium` | recognized prefix and expected format | `ghp_…`, `AKIA…`, `SCW…`, `glpat-…`, JWT |
| `low` | generic heuristic | `password=…`, `api_key=…` |

The default is `low`: everything is reported. That is the only defensible default for a
secret detector — silencing by default what you cannot confirm trades a false positive for a
false negative, on the one subject where a false negative is paid in a leak.

Triage in CI without changing the scan:

```console
$ pepin scan scaleway --terraform plan.json --format json \
  | jq '[.findings[] | select(.labels.check == "compute_instance_no_secrets_in_user_data")
        | {subject, confidence: .labels.confidence}]'
```

Make the scan itself quieter — and accept the cost:

```yaml
controls:
  secrets:
    min_confidence: medium   # generic heuristics are no longer reported
```

This drops the CLD-CMP-9 / SecNumCloud 10.5 mapping, and the report says so. "No cleartext
secret" cannot be proven once you have chosen not to look at part of what was found.

**The detected value never appears**, at any level. The message interpolates the pattern's
label only, never what matched — a report travels through SARIF, CI artifacts and sealed
bundles, and a secret detector that copies the secret into its report turns the report into
the leak. This is enforced by tests, not by intent.

## What the snapshot control does not prove

`blockstorage_volume_snapshots_exist` measures one thing: does a volume in use have, within
the configured window, at least one snapshot whose native state says it is completed? The
state is checked, not just the date — a failed or in-progress snapshot restores nothing.

It does **not** prove:

- that the snapshot is **restorable** — no restore is attempted;
- that it is **complete** in the application sense (hot databases, multi-volume layouts);
- that a **retention** is honoured — a single snapshot satisfies the control;
- that a **backup policy** exists.

A volume backed up by other means — application backup, replication, a provider backup
service, an external tool — will be reported here. That is an accepted false positive: handle
it with a dated, justified exemption rather than by disabling the control. This control takes
no part in any "backup compliant" claim.

It is also **not evaluable on a Terraform plan**: `state` arrives as `after_unknown` and no
`blockstorage_snapshot` exists in a plan. The scan returns `not-evaluated`, with its reason.

## The configuration travels with the proof

The effective configuration is injected into the evaluated inventory under `config`, exactly
like `evaluated_at`. It therefore lands in a sealed bundle's `input.json`, and
`verify --re-derive` replays the same verdict under the same policy without being handed the
policy file again. A replayed `input.json` keeps its own `config`: replaying never applies
today's policy to yesterday's dossier.

Which also means a default scan is never silent about its policy: `--format json` always
publishes `config.policy_digest` and `config.effective`. Two scans separated only by a
setting cannot carry the same digest.

## See also

- [Exit codes](../reference/exit-codes.md) — including `--strict` and exemptions.
- [Evidence bundles](evidence-bundles.md) — what is sealed, and how to verify it.
- [Known limitations](../known-limitations.md) — the blind spots, named.

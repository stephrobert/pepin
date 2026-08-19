> 🇬🇧 English · [🇫🇷 Français](understanding-a-scan.fr.md)

# Understanding a scan

This page reads one real Pépin run end to end. The command:

```bash
./pepin scan scaleway --terraform examples/scaleway/terraform/plan.json
```

Everything reproduced below was captured from that exact command on this repository — the
terminal report, the assessment document, the exit code. Nothing is illustrative. The output
itself is in French: Pépin's CLI, reference and rules are French by repository convention, and
quoting them verbatim is what lets this page be checked against the tool.

## The two streams

Pépin separates them on purpose, so that `pepin scan … > report.txt` gives you the report and
nothing else.

**`stderr`** carries the banner and the scope disclaimer:

<!-- pepin:gen scan-vulnerable-banner -->
```text

 ██████╗ ███████╗██████╗ ██╗ ███╗   ██╗
 ██╔══██╗██╔════╝██╔══██╗██║ ████╗  ██║
 ██████╔╝█████╗  ██████╔╝██║ ██╔██╗ ██║
 ██╔═══╝ ██╔══╝  ██╔═══╝ ██║ ██║╚██╗██║
 ██║     ███████╗██║     ██║ ██║ ╚████║
 ╚═╝     ╚══════╝╚═╝     ╚═╝ ╚═╝  ╚═══╝

 v<version>  · cloud posture scanner (security · compliance)


ⓘ This report assesses the configuration of a tenant (customer-side scope). The normative mappings (SecNumCloud, ISO, CIS) are indicative: they are not a proof of qualification or certification, which applies to the cloud service provider.
```
<!-- /pepin:gen scan-vulnerable-banner -->

The version is shown as `<version>`: it is injected at build time from `git describe`, so it
differs between machines and commits, and freezing it here would make this page diverge on
every build without any behaviour changing. Everything else is the captured output.

The banner is printed *before* collection starts, not at the end: on a live scan against a
large tenant, you want to know the tool started. The closing line is
`assess.ScopeDisclaimer`, emitted on every single scan — see [Scope](../concepts/scope.md).

**`stdout`** carries the report:

<!-- pepin:gen scan-vulnerable-full -->
```text
──────────────────────────────────────────────────────────────────────────────
 Mode      scan scaleway (terraform)
 Source    examples/scaleway/terraform/plan.json
──────────────────────────────────────────────────────────────────────────────

──────────────────────────────────────────────────────────────────────────────
 ⚡ Immediate action — top 3 most severe deviations
──────────────────────────────────────────────────────────────────────────────

  1. 🔴 CRIT  CLD-STO-1 — Bucket "scaleway_object_bucket_acl.backups" is publicly accessi…
     subject: scaleway_object_bucket_acl.backups
  2. 🟠 HIGH  CLD-CMP-9 — VM "scaleway_instance_server.web": cleartext secret in user-dat…
     subject: scaleway_instance_server.web
  3. 🟠 HIGH  CLD-STO-3 — Managed database "pepin-test-rdb": automatic backups are disabl…
     subject: pepin-test-rdb


──────────────────────────────────────────────────────────────────────────────
 CRITICAL  ·  CLD-STO-1  ·  scaleway
 Object storage publicly exposed
──────────────────────────────────────────────────────────────────────────────
  Total deviations: 1

  Details:
      CRIT  scaleway_object_bucket_acl.backups — Bucket "scaleway_object_bucket_acl.backups" is publicly accessible (public ACL).

  Remediation
    Make the bucket private (private ACL, remove the AllUsers grant, delete the public policy); serve through pre-signed URLs if needed.

  ↳ docs: https://stephane-robert.info/scsl/CLD-STO-1

──────────────────────────────────────────────────────────────────────────────
 HIGH  ·  CLD-CHF-2  ·  scaleway
 Managed database without encryption at rest
──────────────────────────────────────────────────────────────────────────────
  Total deviations: 1

  Details:
      HIGH  pepin-test-rdb — Managed database "pepin-test-rdb" has no encryption at rest.

  Remediation
    Enable encryption at rest on the instance (at creation time, or through an upgrade).

  ↳ docs: https://stephane-robert.info/scsl/CLD-CHF-2

──────────────────────────────────────────────────────────────────────────────
 HIGH  ·  CLD-CMP-9  ·  scaleway
 Cleartext secret in the user data (user-data)
──────────────────────────────────────────────────────────────────────────────
  Total deviations: 1

  Details:
      HIGH  scaleway_instance_server.web — VM "scaleway_instance_server.web": cleartext secret in user-data (cleartext password).

  Remediation
    Ban secrets from user data; use a secrets vault and inject them at boot. Revoke the exposed secret.

  ↳ docs: https://stephane-robert.info/scsl/CLD-CMP-9

──────────────────────────────────────────────────────────────────────────────
 HIGH  ·  CLD-IAM-12  ·  scaleway
 IAM policy allowing privilege escalation
──────────────────────────────────────────────────────────────────────────────
  Total deviations: 1

  Details:
      HIGH  ci-deployer — Policy "ci-deployer": grants IAM management (PermissionSet) — a privilege escalation path.

  Remediation
    Reserve IAM management for a dedicated administration policy; remove the management PermissionSet from everyday policies.

  ↳ docs: https://stephane-robert.info/scsl/CLD-IAM-12

──────────────────────────────────────────────────────────────────────────────
 HIGH  ·  CLD-NET-1  ·  scaleway
 Managed database reachable from the internet
──────────────────────────────────────────────────────────────────────────────
  Total deviations: 2

  Details:
      HIGH  fr-par/11111111-1111-1111-1111-111111111111 — Managed database "fr-par/11111111-1111-1111-1111-111111111111": ACL allowing a public CIDR (0.0.0.0/0) — the service is exposed to the internet.
      HIGH  scaleway_instance_security_group.web — Security group "scaleway_instance_security_group.web": SSH (port 22) accepted from/to the internet.

  Remediation
    Restrict the database ACL to the application CIDRs only (a private network where one is available); remove 0.0.0.0/0.

  ↳ docs: https://stephane-robert.info/scsl/CLD-NET-1

──────────────────────────────────────────────────────────────────────────────
 HIGH  ·  CLD-NET-2  ·  scaleway
 Security group inbound default policy set to "accept"
──────────────────────────────────────────────────────────────────────────────
  Total deviations: 1

  Details:
      HIGH  sg-open-default — Security group "sg-open-default": default inbound policy set to "accept" — any unfiltered traffic is admitted.

  Remediation
    Switch the default inbound policy to "drop" and open only the legitimate flows through explicit rules.

  ↳ docs: https://stephane-robert.info/scsl/CLD-NET-2

──────────────────────────────────────────────────────────────────────────────
 HIGH  ·  CLD-STO-3  ·  scaleway
 Automatic backups disabled on a managed database
──────────────────────────────────────────────────────────────────────────────
  Total deviations: 1

  Details:
      HIGH  pepin-test-rdb — Managed database "pepin-test-rdb": automatic backups are disabled.

  Remediation
    Re-enable automatic backups and set a retention that matches the RPO.

  ↳ docs: https://stephane-robert.info/scsl/CLD-STO-3

──────────────────────────────────────────────────────────────────────────────
 MEDIUM  ·  CLD-GVN-1  ·  scaleway
 Incomplete inventory and tagging
──────────────────────────────────────────────────────────────────────────────
  Total deviations: 1

  Details:
      MED   scaleway_instance_server.web — Resource "scaleway_instance_server.web": governance tags missing (CostCenter, Project, Env, Owner).

  Remediation
    Add the mandatory tags (CostCenter, Project, Env, Owner) to the resource.

  ↳ docs: https://stephane-robert.info/scsl/CLD-GVN-1

──────────────────────────────────────────────────────────────────────────────
 LOW  ·  CLD-STO-8  ·  scaleway
 Object Lock (immutability) disabled on object storage
──────────────────────────────────────────────────────────────────────────────
  Total deviations: 1

  Details:
      LOW   backups-prod — Bucket "backups-prod" has no Object Lock: objects are mutable (no WORM protection against deletion or overwrite).

  Remediation
    Enable Object Lock (compliance or governance mode) on backup buckets and critical objects.

  ↳ docs: https://stephane-robert.info/scsl/CLD-STO-8

  Controls
  ╭────────────┬──────────────────────────────────────────────────┬──────────┬──────────┬───╮
  │ Code       │ Control                                          │ Sev      │ Tier     │ # │
  ├────────────┼──────────────────────────────────────────────────┼──────────┼──────────┼───┤
  │ CLD-STO-1  │ Object storage publicly exposed                  │ CRITICAL │ scaleway │ 1 │
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
<!-- /pepin:gen scan-vulnerable-full -->

## Reading it section by section

### Header

<!-- pepin:gen scan-header -->
```text
──────────────────────────────────────────────────────────────────────────────
 Mode      scan scaleway (terraform)
 Source    examples/scaleway/terraform/plan.json
──────────────────────────────────────────────────────────────────────────────
[…]
```
<!-- /pepin:gen scan-header -->

`Mode` names the **provider** and the **source**. `(terraform)` means a plan was projected
through the provider's Terraform mapping; `(live)` would mean the API was queried, and `Source`
would then read `collecte live · profil … · région …` instead of a file path. Plain
`scan scaleway` with a file means a pre-normalized inventory export.

The distinction is not cosmetic. A plan carries the **planned** state and omits anything
`known after apply`; a live collection carries the **effective** configuration but depends on
the rights of the credentials you used.

### Immediate action

The three most severe deviations, ranked, with their subject. It exists so that the first
screen of a long report is already actionable.

### One block per control

<!-- pepin:gen scan-control-objectstore -->
```text
──────────────────────────────────────────────────────────────────────────────
 CRITICAL  ·  CLD-STO-1  ·  scaleway
 Object storage publicly exposed
──────────────────────────────────────────────────────────────────────────────
  Total deviations: 1

  Details:
      CRIT  scaleway_object_bucket_acl.backups — Bucket "scaleway_object_bucket_acl.backups" is publicly accessible (public ACL).

  Remediation
    Make the bucket private (private ACL, remove the AllUsers grant, delete the public policy); serve through pre-signed URLs if needed.

  ↳ docs: https://stephane-robert.info/scsl/CLD-STO-1
```
<!-- /pepin:gen scan-control-objectstore -->

Read the header line first:

- **severity** — `critical`, `high`, `medium`, `low`, from the reference.
- **`CLD-STO-1`** — the **control** identifier, that is the frozen SCSL requirement. The rules
  themselves emit an agnostic check name (`objectstorage_bucket_public_access`); the scan
  resolves it against `referentiel/controles.yaml` and keeps the check name in `labels.check`.
- **`scaleway`** — the tier, taken from the resource itself (`labels.provider`), never
  hardcoded in a rule.
- the title, then `Total deviations`, then one `Details` line per offending subject.

`Details` is the **evidence**: the resource that failed, and the observed fact about it. Then
the **remediation**, and a `docs:` link to the SCSL page of the requirement.

### Controls table and summary

The table recaps code, title, severity, tier and the number of deviations. The summary carries
the **verdict** and the counts per severity.

The verdict phrasing is load-bearing. On a Terraform plan it reads *"périmètre déclaré (plan
Terraform, état planifié)"*; on a scan where nothing was measured it reads
`Verdict : INDÉTERMINÉ`, and the exit code follows.

### Exit code

```bash
echo $?   # 1
```

<!-- pepin:gen exit-codes -->
| Situation | Command | Exit code |
|---|---|:-:|
| No deviation in the evaluated scope | `./pepin scan scaleway --terraform examples/scaleway/terraform-fixed/plan.json` | **0** |
| At least one critical or high deviation | `./pepin scan scaleway --terraform examples/scaleway/terraform/plan.json` | **1** |
| Technical error (unreadable file, unknown provider, unreachable API) | `./pepin scan scaleway examples/scaleway/plan-absent.json` | **2** |
| Nothing was measured (empty inventory): **without having to ask for `--strict`** | `./pepin scan scaleway empty-inventory.json` | **3** |
| Medium/low deviations only, without `--strict` | `./pepin scan scaleway tagless-inventory.json` | **0** |
| Medium/low deviations only, with `--strict` | `./pepin scan scaleway tagless-inventory.json --strict` | **3** |
<!-- /pepin:gen exit-codes -->

## The same run, as an assessment

The terminal report shows deviations. The **assessment** shows every control, including the
ones that did *not* deviate — which is what an auditor asks for.

```bash
./pepin scan scaleway --terraform examples/scaleway/terraform/plan.json --format assessment
```

### Provenance

<!-- pepin:gen assessment-run -->
```json
{
  "run": {
    "ruleset": {
      "digest": "\u003cempreinte règles + descripteurs + référentiel\u003e",
      "name": "pepin-config"
    },
    "scope": {
      "included": [
        "compute_instance",
        "governance_provider",
        "iam_policy",
        "managed_database",
        "object_storage_bucket",
        "security_group",
        "security_group_rule"
      ]
    },
    "source": "terraform-plan",
    "target": {
      "id": "scaleway",
      "platform": "scaleway",
      "provider": "scaleway"
    },
    "timestamp": "\u003chorodatage RFC3339 du scan\u003e",
    "tool": {
      "digest": "\u003cempreinte du binaire\u003e",
      "name": "pepin",
      "version": "\u003cversion\u003e"
    }
  }
}
```
<!-- /pepin:gen assessment-run -->

*(timestamp, version and digests are volatile by nature and are shown as placeholders here;
everything else is the captured document.)*

- `tool.digest` pins the **binary** that produced the verdict (VCS revision, `+modified` when
  the tree was dirty).
- `ruleset.digest` pins **everything that determines the result**: the embedded Rego rules, the
  provider descriptors, the reference, and the content of any `--policy-dir`.
- `scope.included` lists the resource types actually evaluated. A control about a type absent
  from that list cannot have been measured, and the assessment says so.

### The four statuses, on this one run

<!-- pepin:gen assessment-counts -->
| Status | Count |
|---|---:|
| `pass` | 6 |
| `fail` | 10 |
| `not-applicable` | 2 |
| `not-evaluated` | 9 |
<!-- /pepin:gen assessment-counts -->

A single account-free scan produces all four. Each is documented in
[the assessment model](../concepts/assessment-model.md); here is one of each, verbatim.

**`fail`** — a deviation, with its subject and evidence:

<!-- pepin:gen assessment-fail -->
```json
{
  "control": "objectstorage_bucket_public_access",
  "evidence": {
    "attribute": "acl",
    "observed": "Bucket \"scaleway_object_bucket_acl.backups\" is publicly accessible (public ACL).",
    "proves": [
      "",
      "",
      ""
    ],
    "source": "acl=terraform-plan:scaleway_object_bucket + terraform-plan:scaleway_object_bucket_acl observed=2/2"
  },
  "labels": {
    "category": "security",
    "provider": "scaleway"
  },
  "references": [
    {
      "framework": "scsl",
      "id": "CLD-STO-1"
    },
    {
      "framework": "cis-v8",
      "id": "3.3"
    },
    {
      "framework": "iso-27001",
      "id": "A.5.15"
    },
    {
      "framework": "iso-27001",
      "id": "A.8.3"
    },
    {
      "framework": "iso-27017",
      "id": "CLD.9.5.1"
    },
    {
      "framework": "secnumcloud-3.2",
      "id": "9.7"
    },
    {
      "framework": "secnumcloud-3.2",
      "id": "13.2"
    }
  ],
  "remediation": "Make the bucket private (private ACL, remove the AllUsers grant, delete the public policy); serve through pre-signed URLs if needed.",
  "severity": "critical",
  "status": "fail",
  "subject": "scaleway_object_bucket_acl.backups",
  "title": "Object storage publicly exposed"
}
```
<!-- /pepin:gen assessment-fail -->

**`pass`** — note that `evidence.observed` names *what* was checked:

<!-- pepin:gen assessment-pass -->
```json
{
  "control": "network_securitygroup_allow_ingress_from_internet_to_all_ports",
  "evidence": {
    "observed": "no deviation detected on the collected resources of type \"security_group_rule\" (contract verified)",
    "proves": [
      "",
      "",
      ""
    ],
    "source": "terraform-plan"
  },
  "references": [
    {
      "framework": "scsl",
      "id": "CLD-NET-2"
    },
    {
      "framework": "cis-v8",
      "id": "4.4"
    },
    {
      "framework": "cis-v8",
      "id": "12.2"
    },
    {
      "framework": "iso-27001",
      "id": "A.8.20"
    },
    {
      "framework": "iso-27001",
      "id": "A.8.22"
    },
    {
      "framework": "secnumcloud-3.2",
      "id": "13.2"
    }
  ],
  "severity": "critical",
  "status": "pass",
  "subject": "scaleway",
  "title": "All inbound traffic allowed from the internet (any/any)"
}
```
<!-- /pepin:gen assessment-pass -->

**`not-applicable`** — with the justification recorded in the provider contract:

<!-- pepin:gen assessment-na -->
```json
{
  "control": "blockstorage_volume_encryption",
  "references": [
    {
      "framework": "scsl",
      "id": "CLD-CHF-2"
    },
    {
      "framework": "iso-27001",
      "id": "A.8.24"
    },
    {
      "framework": "secnumcloud-3.2",
      "id": "10.1"
    }
  ],
  "severity": "high",
  "status": "not-applicable",
  "subject": "scaleway",
  "title": "Encryption at rest disabled",
  "waiver": {
    "justification": "Encryption at rest of block volumes is guest-side (LUKS/Cryptsetup), a customer responsibility (shared responsibility model); the block API exposes no encryption field, hence unobservable on the platform side (CHF-2)."
  }
}
```
<!-- /pepin:gen assessment-na -->

**`not-evaluated`** — with the exact reason Pépin could not decide:

<!-- pepin:gen assessment-ne -->
```json
{
  "control": "compute_instance_public_ip_with_open_securitygroup",
  "evidence": {
    "observed": "attribute \"public_ip\" not collected on the resources of type \"compute_instance\" (capability guard)",
    "proves": [
      "",
      "",
      ""
    ],
    "source": "terraform-plan"
  },
  "references": [
    {
      "framework": "scsl",
      "id": "CLD-NET-3"
    },
    {
      "framework": "cis-v8",
      "id": "4.4"
    },
    {
      "framework": "cis-v8",
      "id": "12.2"
    },
    {
      "framework": "iso-27001",
      "id": "A.8.20"
    },
    {
      "framework": "iso-27001",
      "id": "A.8.22"
    },
    {
      "framework": "iso-27017",
      "id": "CLD.9.5.2"
    },
    {
      "framework": "secnumcloud-3.2",
      "id": "13.2"
    }
  ],
  "severity": "critical",
  "status": "not-evaluated",
  "subject": "scaleway",
  "title": "Instance publicly exposed without restrictive filtering"
}
```
<!-- /pepin:gen assessment-ne -->

## The chain, end to end

Take the object storage failure and follow it backwards through the repository:

| Link | Artifact | Content |
|---|---|---|
| **finding** | terminal report / assessment `fail` | `scaleway_object_bucket_acl.backups`, publicly readable |
| **evidence** | `examples/scaleway/terraform/plan.json` | `scaleway_object_bucket_acl.backups` declares `acl = "public-read"` |
| **projection** | `providers/scaleway.yaml`, `mapping_terraform` | `scaleway_object_bucket_acl` → type `object_storage_bucket`, attribute `acl` |
| **rule** | `internal/commonrules/rules/objectstorage_bucket_public_access.rego` | one common rule, `labels.provider` read from the resource |
| **control** | `referentiel/controles.yaml` | code `objectstorage_bucket_public_access`, severity `critical` |
| **normative references** | same entry | SCSL `CLD-STO-1`, plus SecNumCloud / CIS / ISO mappings |
| **remediation** | the rule's `remediation` field | make the bucket private, drop the `AllUsers` grant, use pre-signed URLs |

Each link is a file in this repository, and each is checked: `mise run validate` refuses a
control without a rule, a code emitted but not catalogued, or an SCSL reference absent from
the frozen index.

## Sealing it

```bash
./pepin scan scaleway --terraform examples/scaleway/terraform/plan.json --seal ./bundle
./pepin verify ./bundle
```

The bundle holds the evaluated inventory, the assessment, its OSCAL 1.1.2 rendering, a digested
manifest and a checksums file, so a third party can re-check integrity — and, with
`--re-derive`, re-run the evaluation on the sealed inventory.

**Without `--redact`, `input.json` embeds the raw inventory**: user-data, IAM policy documents,
policies. Pépin warns about it on stderr. Treat an unredacted bundle as sensitive, or produce a
shareable one with `--redact` (which replaces sensitive values by digests, and is therefore
incompatible with `verify --re-derive`).

## Other formats

| Flag | For |
|---|---|
| `--format table` | humans (the default, shown above) |
| `--format json` | `{"findings": …, "summary": …}` — scripts |
| `--format assessment` | the typed, referenced, provenanced document |
| `--format oscal` | OSCAL 1.1.2 assessment results, schema-validated in CI |
| `--format sarif` | code scanning (GitHub, GitLab) |

## See also

- [Quickstart](quickstart.md) — the same example, with a fix and a second scan.
- [The assessment model](../concepts/assessment-model.md) — what each status asserts.
- [Coverage matrix](../coverage.md) — what is measurable, per provider and per source.
- [Scope and non-goals](../concepts/scope.md) — what this report does not prove.

> 🇬🇧 English · [🇫🇷 Français](output-formats.fr.md)

# Output formats

`pepin scan` renders the same evaluation in five shapes. They are not five renderings of the
same information: two of them can say "nothing was measured", the other three cannot. Choosing
the wrong one is how a compliance chain ends up unable to tell a clean tenant from an
uncollected one.

Every example below is the same scan of the same Terraform plan, in the format named by the
section.

| Format | Audience | Parse it? | Says `pass` / `not-evaluated`? |
|---|---|---|---|
| `table` (default) | a human, in a terminal | **no** | no, but the verdict line says so in prose |
| `json` | a pipeline, a dashboard | yes, frozen | no: deviations only |
| `assessment` | a compliance chain, an auditor | yes, frozen | **yes** |
| `oscal` | a GRC tool that ingests OSCAL | yes, standard | yes, through observations |
| `sarif` | GitHub Code Scanning, an IDE | yes, standard | no: deviations only |

The exit code is the same in all five: the format changes what is written, never the verdict.
See [Exit codes](exit-codes.md).

## Which shapes are frozen

<!-- pepin:gen surface-versions -->
| Surface | What is frozen | Version |
|---|---|:-:|
| `cli` | verbs, flags and exit codes | **v3** |
| `findings` | shape of `--format json` (`findings` + `summary`) | **v1** |
| `assessment` | shape of the `--format assessment` document | **v1** |
| `bundle` | shape of the evidence bundle (files, roles, manifest) | **v2** |
| `inventory` | shape of the normalized inventory (envelope, resource, types and attributes) | **v1** |
<!-- /pepin:gen surface-versions -->

"Frozen" means a test fails when the shape moves and its version has not: field paths and JSON
types are the promise, values are not. `table` is deliberately absent from that list — its
layout is meant to change as the terminal report improves, which is exactly why nothing should
parse it.

## `table` — the default, for humans

<!-- pepin:gen scan-vulnerable-tail -->
```text
[…]

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
<!-- /pepin:gen scan-vulnerable-tail -->

The banner and the scope disclaimer go to **standard error**, the report to standard output.
Colour is dropped when the output is not a terminal, or when `NO_COLOR` is set. A commented,
line-by-line reading of this output is in
[Understanding a scan](../getting-started/understanding-a-scan.md).

## `json` — deviations and their count

Frozen shape: `{"findings": [...], "summary": {...}}`.

<!-- pepin:gen format-json-summary -->
```json
{
  "conforme": false,
  "critical": 1,
  "high": 7,
  "low": 1,
  "medium": 1,
  "total": 10
}
```
<!-- /pepin:gen format-json-summary -->

<!-- pepin:gen format-json-finding -->
```json
{
  "code": "CLD-STO-1",
  "labels": {
    "category": "security",
    "check": "objectstorage_bucket_public_access",
    "provider": "scaleway"
  },
  "message": "Bucket \"scaleway_object_bucket_acl.backups\" is publicly accessible (public ACL).",
  "remediation": "Make the bucket private (private ACL, remove the AllUsers grant, delete the public policy); serve through pre-signed URLs if needed.",
  "severity": "critical",
  "subject": "scaleway_object_bucket_acl.backups",
  "title": "Object storage publicly exposed"
}
```
<!-- /pepin:gen format-json-finding -->

`code` is the normative identifier (`CLD-*`), and `labels.check` is the agnostic check
identifier, common to every provider. Both are stable across languages; `title`, `message` and
`remediation` are translated.

**What this format cannot tell you**: it lists deviations, so an empty `findings` array means
"no deviation found", which covers both "the tenant is clean" and "nothing was collected". The
`summary.conforme` boolean says nothing about coverage either. If that distinction matters —
and for a compliance gate it does — read the exit code (3 means nothing was measured) or use
the assessment format.

### `exemptions` — present only under `--exceptions`

When a scan is given an exemptions file, the document carries a third top-level key next to
`findings` and `summary`:

```json
{
  "exemptions": {
    "policy_digest": "sha256:…",
    "exceptions": [ { "control": "…", "justification": "…", "expires_at": "…", "owner": "…", "approved_by": "…" } ],
    "records": [ { "control": "…", "effect": "applied", "subjects": ["…"] } ]
  }
}
```

`effect` is `applied`, `expired` or `orphan`. Nothing is removed from `findings` or from
`summary` for being exempted: an exemption moves the **exit code**, never the record. A
pipeline that wants to refuse exemptions altogether checks that `exemptions` is absent.

## `assessment` — the typed, defensible document

This is the format built for a compliance chain: one entry per control, with a typed status, a
piece of evidence, the normative references, and the provenance of the run.

<!-- pepin:gen assessment-counts -->
| Status | Count |
|---|---:|
| `pass` | 6 |
| `fail` | 10 |
| `not-applicable` | 2 |
| `not-evaluated` | 9 |
<!-- /pepin:gen assessment-counts -->

Four measured statuses, and the two that the other formats cannot express are
`not-applicable` (the provider contract declares the control untestable, with its
justification) and `not-evaluated` (Pépin could not decide, and says on what it stumbled). A
fifth, `exempted`, appears when `--exceptions` supplies a waiver; it is never a compliance.
Their exact meaning is in [The assessment model](../concepts/assessment-model.md).

`evidence.attribute` and `evidence.source` carry the **provenance of the deciding attribute**
when Pépin has one: which attribute the control reads, where its value came from (`api:` plus
the request actually served, `terraform-plan:` plus the plan resource type, or `derived:` plus
the descriptor element), and on how many resources the source really carried it
(`observed=n/m`). A `derived:` source is the honest name for a value no API call produced.

One result:

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

And the provenance envelope every document carries:

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

The volatile fields are marked here because they change at every run; in a real document they
carry the scan's timestamp, the tool's build digest and the digest of the rules, descriptors
and reference that produced the verdict. This is the document `scan --seal` writes into an
evidence bundle, and the one `verify --re-derive` replays: see
[Evidence bundles](../guides/evidence-bundles.md).

## `oscal` — assessment results for a GRC tool

The same assessment, rendered as an OSCAL 1.1.2 `assessment-results` document.

<!-- pepin:gen format-oscal-head -->
```json
{
  "assessment-results": {
    "uuid": "<uuid>",
    "metadata": {
      "title": "pepin assessment of scaleway",
      "last-modified": "<timestamp>",
      "version": "<version>",
      "oscal-version": "1.1.2",
      "props": [
        {
          "name": "tool-name",
          "value": "pepin",
          "ns": "https://github.com/stephrobert/scankit/ns/oscal"
        },
        {
          "name": "tool-version",
          "value": "<version>",
          "ns": "https://github.com/stephrobert/scankit/ns/oscal"
        },
        {
          "name": "tool-digest",
          "value": "<provenance>",
          "ns": "https://github.com/stephrobert/scankit/ns/oscal"
        },
[…]
```
<!-- /pepin:gen format-oscal-head -->

Use it when the consumer is a GRC platform that already ingests OSCAL: the format is a NIST
standard, not a Pépin invention, and the mapping is done by `scankit`, shared with the other
tools of the same family. Note that the normative mappings are **indicative**: an OSCAL
document produced by Pépin describes a tenant's configuration, not a qualification of the cloud
provider ([Scope and non-goals](../concepts/scope.md)).

## `sarif` — for GitHub Code Scanning

SARIF 2.1.0, the format GitHub's Code Scanning tab reads. This is the one to upload with
`github/codeql-action/upload-sarif`.

<!-- pepin:gen format-sarif-head -->
```json
{
  "$schema": "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/sarif-2.1/schema/sarif-schema-2.1.0.json",
  "version": "2.1.0",
  "runs": [
    {
      "tool": {
        "driver": {
          "name": "pepin",
          "version": "<version>",
          "rules": [
            {
              "id": "CLD-CHF-2",
              "shortDescription": {
                "text": "CLD-CHF-2 (scaleway)"
              },
              "helpUri": "https://stephane-robert.info/scsl/CLD-CHF-2"
            },
            {
              "id": "CLD-CMP-9",
              "shortDescription": {
                "text": "CLD-CMP-9 (scaleway)"
              },
[…]
```
<!-- /pepin:gen format-sarif-head -->

One result:

<!-- pepin:gen format-sarif-result -->
```json
{
  "level": "error",
  "locations": [
    {
      "physicalLocation": {
        "artifactLocation": {
          "uri": "examples/scaleway/terraform/plan.json"
        }
      }
    }
  ],
  "message": {
    "text": "Bucket \"scaleway_object_bucket_acl.backups\" is publicly accessible (public ACL)."
  },
  "ruleId": "CLD-STO-1"
}
```
<!-- /pepin:gen format-sarif-result -->

Two things to know before wiring it up:

- **Locations point at the scanned file, not at a line.** The `artifactLocation.uri` is the
  input Pépin read (here the Terraform plan); no `region` is emitted, because a normalized
  resource does not carry the line of the `.tf` file it came from. Alerts therefore land on the
  file, not on the guilty attribute.
- **`level`** is derived from the severity: `error` for `critical` and `high`, `warning` for
  `medium`, `note` for `low`.

The upload step and the permissions it needs are in
[GitHub Actions](../guides/github-actions.md#publish-the-findings-to-code-scanning).

## Formats and language

Codes, check identifiers, severities and statuses are identical in French and in English.
Titles, messages, remediations and evidence are translated — in `json`, `assessment`, `oscal`
and `sarif` alike, since the prose travels with the finding.

Two consequences for a pipeline:

1. **Diffing report text between two runs requires pinning `PEPIN_LANG`.** A runner whose
   `LANG` changes will otherwise produce a diff nobody caused.
2. **Keying on `code`, `labels.check`, `status` and `severity` requires pinning nothing.**

## Where to go next

- [The normalized inventory](inventory.md) — the shape every one of these documents derives
  from, and its frozen version.

- [Exit codes](exit-codes.md) — what to gate on.
- [The assessment model](../concepts/assessment-model.md) — what each status asserts.
- [Evidence bundles](../guides/evidence-bundles.md) — sealing the assessment and the OSCAL.
- [CLI reference](cli.md) — `--format` and its neighbours.

## How this page stays true

Every document above was produced by running the binary on the repository's Terraform plan
fixture, captured by `internal/docgen`. Timestamps, UUIDs, digests and the build version are
replaced by explicit markers, because they change at every run and would otherwise make this
page diverge without any behaviour having moved. `mise run gen-docs` rewrites the blocks;
`TestGeneratedDocsAreUpToDate` fails when the committed page no longer matches what the binary
produces.

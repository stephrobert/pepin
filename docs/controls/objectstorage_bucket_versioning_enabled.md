> 🇬🇧 English · [🇫🇷 Français](objectstorage_bucket_versioning_enabled.fr.md)

<!-- GENERATED PAGE — do not edit by hand. Regenerate with `mise run gen-docs`. -->

# `objectstorage_bucket_versioning_enabled`

**Object storage versioning disabled**

[Back to the catalogue](index.md)

| Field | Value |
|---|---|
| Code | `objectstorage_bucket_versioning_enabled` |
| Family | `stockage` |
| Severity | `medium` |
| SCSL requirement (frozen index) | `CLD-STO-4` |
| Resource type read | `object_storage_bucket` |
| Deciding attribute | `versioning` |
| State | active |
| Declared for | `exoscale`, `outscale`, `scaleway` |
| Remediation proofs | 0 / 3 |

## The risk

An object storage bucket does not have versioning enabled: a deletion or an overwrite (accidental or malicious, ransomware for instance) is irreversible.

This description comes from the reference: it is the text the report quotes, in the
reader's language.

## Normative mappings

Taken verbatim from the reference. The SCSL requirement comes from the **frozen** index:
a control maps onto an existing requirement, never onto one created for it. These mappings
are **indicative**: a Pépin report is not proof of qualification.

| Framework | References |
|---|---|
| `scsl` | `CLD-STO-4` |
| `cis_controls_v8` | `11.1` |
| `iso_27001_2022` | `A.8.13` |
| `secnumcloud_3_2` | `12.5` |

## Where Pépin can measure it

A ✅ cell means the source produces the targeted type, the provider contract marks it
`verifie`, and the deciding attribute is projected. ◐ means "Pépin cannot decide from this
source", ∅ "not testable, with justification", ✗ "not declared, or type absent from this
source".

| Provider | Terraform plan | Live collection |
|---|:-:|:-:|
| exoscale | ✗ | ✅ |
| outscale | ✗ | ✅ |
| scaleway | ◐ | ✅ |
| kubernetes | n/a | ✗ |

Every cell that is not ✅ **while the control is declared for that provider** carries its
reason:

| Provider | Source | Status | Reason |
|---|---|---|---|
| exoscale | terraform | ✗ `unsupported` | this source produces no resource of type "object_storage_bucket" |
| outscale | terraform | ✗ `unsupported` | this source produces no resource of type "object_storage_bucket" |
| scaleway | terraform | ◐ `partial` | deciding attribute "versioning" not projected by this source: a capability guard, so the scan returns "not-evaluated" |

## What Pépin can conclude

| Status | What the status asserts | Reachable from |
|---|---|---|
| `fail` | a deviation was detected on a real resource | exoscale / live · outscale / live · scaleway / terraform · scaleway / live |
| `pass` | the deciding data was collected, and it is compliant | exoscale / live · outscale / live · scaleway / live |
| `not-applicable` | the provider contract declares the control untestable, with its justification | — |
| `not-evaluated` | the control is implemented, but the data it depends on was not confirmed | scaleway / terraform |

An observable control still returns `not-evaluated` on an inventory that contains no
resource of the targeted type: "nothing to look at" is not "compliant".

## How to investigate

- Normalized resource type the rule reads: `object_storage_bucket`
- Attribute the decision depends on: `versioning`
- Without that attribute on a resource of the targeted type, the scan returns `not-evaluated` rather than `pass` (`internal/assess`, `requiredAttr` table).
- What each source projects is readable in the descriptor: [`providers/exoscale.yaml`](../../providers/exoscale.yaml) · [`providers/outscale.yaml`](../../providers/outscale.yaml) · [`providers/scaleway.yaml`](../../providers/scaleway.yaml)
- The rule that emits this code lives in [`internal/commonrules/rules/`](../../internal/commonrules/rules): it is **common** to every provider, only the source changes.

## How to remediate

Enable versioning on the buckets holding critical data (PutBucketVersioning).

| Provider | Deployable setup |
|---|---|
| exoscale | _no proof filed yet_ |
| outscale | _no proof filed yet_ |
| scaleway | _no proof filed yet_ |

A remediation proof is a self-contained, **compliant** Terraform module that deploys as
is, or a note anchored on the official documentation. See
[the remediation guide](../guides/remediation.md).

## How to verify the fix

```bash
# from a Terraform plan: nothing is provisioned
./pepin scan scaleway --terraform plan.json --format assessment

# from the provider API: effective configuration
./pepin scan exoscale --live --format assessment
```

In the `assessment` output, look for `"control": "objectstorage_bucket_versioning_enabled"`: its `status` must be `pass`.
If it stays `not-evaluated`, the deciding data was not collected and the fix is **not**
demonstrated: the reasons table above says why.

**One of the two sources cannot lift the `pass` lock** for this control: the provider
quoted does produce the targeted type there, but the scan will return `not-evaluated`.
The reasons table says which one, and why.

## See also

- [The assessment model](../concepts/assessment-model.md): what each status asserts.
- [Coverage matrix](../coverage.md): the same information, across every control.
- [Terraform plan or live scan](../concepts/terraform-vs-live.md): choosing the source.
- [Adding a control](../contributing/adding-a-control.md): the end-to-end procedure.

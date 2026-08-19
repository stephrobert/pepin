> 🇬🇧 English · [🇫🇷 Français](blockstorage_snapshot_not_public.fr.md)

<!-- GENERATED PAGE — do not edit by hand. Regenerate with `mise run gen-docs`. -->

# `blockstorage_snapshot_not_public`

**Snapshot or image shared publicly**

[Back to the catalogue](index.md)

| Field | Value |
|---|---|
| Code | `blockstorage_snapshot_not_public` |
| Family | `stockage` |
| Severity | `high` |
| SCSL requirement (frozen index) | `CLD-STO-2` |
| Resource type read | `blockstorage_snapshot` |
| Deciding attribute | `global_permission` |
| State | active |
| Declared for | `outscale` |
| Remediation proofs | 0 / 1 |

## The risk

A snapshot or a disk image is shared publicly, potentially exposing data or secrets.

This description comes from the reference: it is the text the report quotes, in the
reader's language.

## Normative mappings

Taken verbatim from the reference. The SCSL requirement comes from the **frozen** index:
a control maps onto an existing requirement, never onto one created for it. These mappings
are **indicative**: a Pépin report is not proof of qualification.

| Framework | References |
|---|---|
| `scsl` | `CLD-STO-2` |
| `cis_controls_v8` | `3.3` |
| `iso_27001_2022` | `A.5.15`, `A.8.3` |
| `iso_27017` | `CLD.9.5.1` |
| `secnumcloud_3_2` | `9.7`, `13.2` |

## Where Pépin can measure it

A ✅ cell means the source produces the targeted type, the provider contract marks it
`verifie`, and the deciding attribute is projected. ◐ means "Pépin cannot decide from this
source", ∅ "not testable, with justification", ✗ "not declared, or type absent from this
source".

| Provider | Terraform plan | Live collection |
|---|:-:|:-:|
| exoscale | ∅ | ∅ |
| outscale | ✗ | ✅ |
| scaleway | ∅ | ∅ |
| kubernetes | n/a | ✗ |

Every cell that is not ✅ **while the control is declared for that provider** carries its
reason:

| Provider | Source | Status | Reason |
|---|---|---|---|
| exoscale | terraform | ∅ `not-applicable` | Exoscale block-storage snapshots cannot be exported or shared (official documentation): the risk of public exposure is structurally absent, compliant by construction (STO-2). |
| exoscale | live | ∅ `not-applicable` | Exoscale block-storage snapshots cannot be exported or shared (official documentation): the risk of public exposure is structurally absent, compliant by construction (STO-2). |
| outscale | terraform | ✗ `unsupported` | this source produces no resource of type "blockstorage_snapshot" |
| scaleway | terraform | ∅ `not-applicable` | Scaleway block snapshots (api/block/v1) expose no sharing or public export mechanism: the risk of public exposure is structurally absent, compliant by construction (STO-2). |
| scaleway | live | ∅ `not-applicable` | Scaleway block snapshots (api/block/v1) expose no sharing or public export mechanism: the risk of public exposure is structurally absent, compliant by construction (STO-2). |

## What Pépin can conclude

| Status | What the status asserts | Reachable from |
|---|---|---|
| `fail` | a deviation was detected on a real resource | outscale / live |
| `pass` | the deciding data was collected, and it is compliant | outscale / live |
| `not-applicable` | the provider contract declares the control untestable, with its justification | exoscale / terraform · exoscale / live · scaleway / terraform · scaleway / live |
| `not-evaluated` | the control is implemented, but the data it depends on was not confirmed | — |

An observable control still returns `not-evaluated` on an inventory that contains no
resource of the targeted type: "nothing to look at" is not "compliant".

## How to investigate

- Normalized resource type the rule reads: `blockstorage_snapshot`
- Attribute the decision depends on: `global_permission`
- Without that attribute on a resource of the targeted type, the scan returns `not-evaluated` rather than `pass` (`internal/assess`, `requiredAttr` table).
- What each source projects is readable in the descriptor: [`providers/outscale.yaml`](../../providers/outscale.yaml)
- The rule that emits this code lives in [`internal/commonrules/rules/`](../../internal/commonrules/rules): it is **common** to every provider, only the source changes.

## How to remediate

Remove the public sharing; restrict sharing to legitimate accounts.

| Provider | Deployable setup |
|---|---|
| outscale | _no proof filed yet_ |

A remediation proof is a self-contained, **compliant** Terraform module that deploys as
is, or a note anchored on the official documentation. See
[the remediation guide](../guides/remediation.md).

## How to verify the fix

```bash
# from the provider API: effective configuration
./pepin scan outscale --live --format assessment
```

In the `assessment` output, look for `"control": "blockstorage_snapshot_not_public"`: its `status` must be `pass`.
If it stays `not-evaluated`, the deciding data was not collected and the fix is **not**
demonstrated: the reasons table above says why.

## See also

- [The assessment model](../concepts/assessment-model.md): what each status asserts.
- [Coverage matrix](../coverage.md): the same information, across every control.
- [Terraform plan or live scan](../concepts/terraform-vs-live.md): choosing the source.
- [Adding a control](../contributing/adding-a-control.md): the end-to-end procedure.

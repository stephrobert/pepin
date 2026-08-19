> 🇬🇧 English · [🇫🇷 Français](blockstorage_volume_encryption.fr.md)

<!-- GENERATED PAGE — do not edit by hand. Regenerate with `mise run gen-docs`. -->

# `blockstorage_volume_encryption`

**Encryption at rest disabled**

[Back to the catalogue](index.md)

| Field | Value |
|---|---|
| Code | `blockstorage_volume_encryption` |
| Family | `chiffrement` |
| Severity | `high` |
| SCSL requirement (frozen index) | `CLD-CHF-2` |
| Resource type read | `blockstorage_volume` |
| Deciding attribute | `encrypted` |
| State | active |
| Declared for | `exoscale` |
| Remediation proofs | 0 / 1 |

## The risk

A block storage volume does not have encryption at rest enabled. The model differs from one provider to the next: transparent encryption always on (Exoscale, compliant by construction), or guest-side encryption left to the customer and not exposed by the API (Outscale, Scaleway: not applicable on the platform side).

This description comes from the reference: it is the text the report quotes, in the
reader's language.

## Normative mappings

Taken verbatim from the reference. The SCSL requirement comes from the **frozen** index:
a control maps onto an existing requirement, never onto one created for it. These mappings
are **indicative**: a Pépin report is not proof of qualification.

| Framework | References |
|---|---|
| `scsl` | `CLD-CHF-2` |
| `iso_27001_2022` | `A.8.24` |
| `secnumcloud_3_2` | `10.1` |

## Where Pépin can measure it

A ✅ cell means the source produces the targeted type, the provider contract marks it
`verifie`, and the deciding attribute is projected. ◐ means "Pépin cannot decide from this
source", ∅ "not testable, with justification", ✗ "not declared, or type absent from this
source".

| Provider | Terraform plan | Live collection |
|---|:-:|:-:|
| exoscale | ✅ | ✅ |
| outscale | ∅ | ∅ |
| scaleway | ∅ | ∅ |
| kubernetes | n/a | ✗ |

Every cell that is not ✅ **while the control is declared for that provider** carries its
reason:

| Provider | Source | Status | Reason |
|---|---|---|---|
| outscale | terraform | ∅ `not-applicable` | osc-sdk-go/v2 Volume exposes no encryption field; encryption at rest is guest-side (EncFS/LUKS), a customer responsibility, hence unobservable on the platform side (CHF-2). |
| outscale | live | ∅ `not-applicable` | osc-sdk-go/v2 Volume exposes no encryption field; encryption at rest is guest-side (EncFS/LUKS), a customer responsibility, hence unobservable on the platform side (CHF-2). |
| scaleway | terraform | ∅ `not-applicable` | Encryption at rest of block volumes is guest-side (LUKS/Cryptsetup), a customer responsibility (shared responsibility model); the block API exposes no encryption field, hence unobservable on the platform side (CHF-2). |
| scaleway | live | ∅ `not-applicable` | Encryption at rest of block volumes is guest-side (LUKS/Cryptsetup), a customer responsibility (shared responsibility model); the block API exposes no encryption field, hence unobservable on the platform side (CHF-2). |

## What Pépin can conclude

| Status | What the status asserts | Reachable from |
|---|---|---|
| `fail` | a deviation was detected on a real resource | exoscale / terraform · exoscale / live |
| `pass` | the deciding data was collected, and it is compliant | exoscale / terraform · exoscale / live |
| `not-applicable` | the provider contract declares the control untestable, with its justification | outscale / terraform · outscale / live · scaleway / terraform · scaleway / live |
| `not-evaluated` | the control is implemented, but the data it depends on was not confirmed | — |

An observable control still returns `not-evaluated` on an inventory that contains no
resource of the targeted type: "nothing to look at" is not "compliant".

## How to investigate

- Normalized resource type the rule reads: `blockstorage_volume`
- Attribute the decision depends on: `encrypted`
- Without that attribute on a resource of the targeted type, the scan returns `not-evaluated` rather than `pass` (`internal/assess`, `requiredAttr` table).
- What each source projects is readable in the descriptor: [`providers/exoscale.yaml`](../../providers/exoscale.yaml)
- The rule that emits this code lives in [`internal/commonrules/rules/`](../../internal/commonrules/rules): it is **common** to every provider, only the source changes.

## How to remediate

Enable encryption at rest on the volume (transparent on the provider side, or client-side encryption with EncFS/LUKS, depending on the shared responsibility model).

| Provider | Deployable setup |
|---|---|
| exoscale | _no proof filed yet_ |

A remediation proof is a self-contained, **compliant** Terraform module that deploys as
is, or a note anchored on the official documentation. See
[the remediation guide](../guides/remediation.md).

## How to verify the fix

```bash
# from a Terraform plan: nothing is provisioned
./pepin scan exoscale --terraform plan.json --format assessment

# from the provider API: effective configuration
./pepin scan exoscale --live --format assessment
```

In the `assessment` output, look for `"control": "blockstorage_volume_encryption"`: its `status` must be `pass`.
If it stays `not-evaluated`, the deciding data was not collected and the fix is **not**
demonstrated: the reasons table above says why.

## See also

- [The assessment model](../concepts/assessment-model.md): what each status asserts.
- [Coverage matrix](../coverage.md): the same information, across every control.
- [Terraform plan or live scan](../concepts/terraform-vs-live.md): choosing the source.
- [Adding a control](../contributing/adding-a-control.md): the end-to-end procedure.

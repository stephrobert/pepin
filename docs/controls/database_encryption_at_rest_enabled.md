> 🇬🇧 English · [🇫🇷 Français](database_encryption_at_rest_enabled.fr.md)

<!-- GENERATED PAGE — do not edit by hand. Regenerate with `mise run gen-docs`. -->

# `database_encryption_at_rest_enabled`

**Managed database without encryption at rest**

[Back to the catalogue](index.md)

| Field | Value |
|---|---|
| Code | `database_encryption_at_rest_enabled` |
| Family | `chiffrement` |
| Severity | `high` |
| SCSL requirement (frozen index) | `CLD-CHF-2` |
| Resource type read | `managed_database` |
| Deciding attribute | `encryption_at_rest` |
| State | active |
| Declared for | `scaleway` |
| Remediation proofs | 0 / 1 |

## The risk

A managed database does not have encryption at rest enabled: should the storage media be accessed, the data is readable in cleartext.

This description comes from the reference: it is the text the report quotes, in the
reader's language.

## Normative mappings

Taken verbatim from the reference. The SCSL requirement comes from the **frozen** index:
a control maps onto an existing requirement, never onto one created for it. These mappings
are **indicative**: a Pépin report is not proof of qualification.

| Framework | References |
|---|---|
| `scsl` | `CLD-CHF-2` |
| `cis_controls_v8` | `3.11` |
| `iso_27001_2022` | `A.8.24` |
| `secnumcloud_3_2` | `10.1` |

## Where Pépin can measure it

A ✅ cell means the source produces the targeted type, the provider contract marks it
`verifie`, and the deciding attribute is projected. ◐ means "Pépin cannot decide from this
source", ∅ "not testable, with justification", ✗ "not declared, or type absent from this
source".

| Provider | Terraform plan | Live collection |
|---|:-:|:-:|
| exoscale | ✗ | ✗ |
| outscale | ✗ | ✗ |
| scaleway | ✅ | ✗ |
| kubernetes | n/a | ✗ |

Every cell that is not ✅ **while the control is declared for that provider** carries its
reason:

| Provider | Source | Status | Reason |
|---|---|---|---|
| scaleway | live | ✗ `unsupported` | this source produces no resource of type "managed_database" |

## What Pépin can conclude

| Status | What the status asserts | Reachable from |
|---|---|---|
| `fail` | a deviation was detected on a real resource | scaleway / terraform |
| `pass` | the deciding data was collected, and it is compliant | scaleway / terraform |
| `not-applicable` | the provider contract declares the control untestable, with its justification | — |
| `not-evaluated` | the control is implemented, but the data it depends on was not confirmed | — |

An observable control still returns `not-evaluated` on an inventory that contains no
resource of the targeted type: "nothing to look at" is not "compliant".

## How to investigate

- Normalized resource type the rule reads: `managed_database`
- Attribute the decision depends on: `encryption_at_rest`
- Without that attribute on a resource of the targeted type, the scan returns `not-evaluated` rather than `pass` (`internal/assess`, `requiredAttr` table).
- What each source projects is readable in the descriptor: [`providers/scaleway.yaml`](../../providers/scaleway.yaml)
- The rule that emits this code lives in [`internal/commonrules/rules/`](../../internal/commonrules/rules): it is **common** to every provider, only the source changes.

## How to remediate

Enable encryption at rest on the instance (at creation time, or through an upgrade).

| Provider | Deployable setup |
|---|---|
| scaleway | _no proof filed yet_ |

A remediation proof is a self-contained, **compliant** Terraform module that deploys as
is, or a note anchored on the official documentation. See
[the remediation guide](../guides/remediation.md).

## How to verify the fix

```bash
# from a Terraform plan: nothing is provisioned
./pepin scan scaleway --terraform plan.json --format assessment
```

In the `assessment` output, look for `"control": "database_encryption_at_rest_enabled"`: its `status` must be `pass`.
If it stays `not-evaluated`, the deciding data was not collected and the fix is **not**
demonstrated: the reasons table above says why.

## See also

- [The assessment model](../concepts/assessment-model.md): what each status asserts.
- [Coverage matrix](../coverage.md): the same information, across every control.
- [Terraform plan or live scan](../concepts/terraform-vs-live.md): choosing the source.
- [Adding a control](../contributing/adding-a-control.md): the end-to-end procedure.

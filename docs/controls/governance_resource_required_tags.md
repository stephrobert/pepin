> 🇬🇧 English · [🇫🇷 Français](governance_resource_required_tags.fr.md)

<!-- GENERATED PAGE — do not edit by hand. Regenerate with `mise run gen-docs`. -->

# `governance_resource_required_tags`

**Incomplete inventory and tagging**

[Back to the catalogue](index.md)

| Field | Value |
|---|---|
| Code | `governance_resource_required_tags` |
| Family | `gouvernance` |
| Severity | `medium` |
| SCSL requirement (frozen index) | `CLD-GVN-1` |
| Resource type read | _none: cross-cutting control_ |
| Deciding attribute | _none: judged on the presence of a deviation_ |
| State | active |
| Declared for | `exoscale`, `outscale`, `scaleway` |
| Remediation proofs | 1 / 3 |

## The risk

WHAT IS CHECKED: a billable resource carries the required governance tags - by default cost centre, project, environment and owner. What is required are the QUESTIONS an inventory must answer (who pays, for what, at which stage, who is accountable), never exact words: the comparison ignores case and separators (`cost-center` = `CostCenter`), and the profile, its aliases and the targeted resource types are configurable (`controls.tagging`). The shipped profile is a RECOMMENDATION, not a standard. Without collected tags the control returns "not evaluated".

This description comes from the reference: it is the text the report quotes, in the
reader's language.

## Normative mappings

Taken verbatim from the reference. The SCSL requirement comes from the **frozen** index:
a control maps onto an existing requirement, never onto one created for it. These mappings
are **indicative**: a Pépin report is not proof of qualification.

| Framework | References |
|---|---|
| `scsl` | `CLD-GVN-1` |
| `cis_controls_v8` | `1.1` |
| `iso_27001_2022` | `A.5.9`, `A.8.9` |
| `secnumcloud_3_2` | `8.1` |

## Where Pépin can measure it

A ✅ cell means the source produces the targeted type, the provider contract marks it
`verifie`, and the deciding attribute is projected. ◐ means "Pépin cannot decide from this
source", ∅ "not testable, with justification", ✗ "not declared, or type absent from this
source".

| Provider | Terraform plan | Live collection |
|---|:-:|:-:|
| exoscale | ◐ | ◐ |
| outscale | ◐ | ◐ |
| scaleway | ◐ | ◐ |
| kubernetes | n/a | ✗ |

Every cell that is not ✅ **while the control is declared for that provider** carries its
reason:

| Provider | Source | Status | Reason |
|---|---|---|---|
| exoscale | terraform | ◐ `partial` | no targeted resource type, and the control does not read the provider descriptor: the "pass" lock cannot be lifted, so the scan returns "not-evaluated" as long as no deviation is detected |
| exoscale | live | ◐ `partial` | no targeted resource type, and the control does not read the provider descriptor: the "pass" lock cannot be lifted, so the scan returns "not-evaluated" as long as no deviation is detected |
| outscale | terraform | ◐ `partial` | no targeted resource type, and the control does not read the provider descriptor: the "pass" lock cannot be lifted, so the scan returns "not-evaluated" as long as no deviation is detected |
| outscale | live | ◐ `partial` | no targeted resource type, and the control does not read the provider descriptor: the "pass" lock cannot be lifted, so the scan returns "not-evaluated" as long as no deviation is detected |
| scaleway | terraform | ◐ `partial` | no targeted resource type, and the control does not read the provider descriptor: the "pass" lock cannot be lifted, so the scan returns "not-evaluated" as long as no deviation is detected |
| scaleway | live | ◐ `partial` | no targeted resource type, and the control does not read the provider descriptor: the "pass" lock cannot be lifted, so the scan returns "not-evaluated" as long as no deviation is detected |

## What Pépin can conclude

| Status | What the status asserts | Reachable from |
|---|---|---|
| `fail` | a deviation was detected on a real resource | exoscale / terraform · exoscale / live · outscale / terraform · outscale / live · scaleway / terraform · scaleway / live |
| `pass` | the deciding data was collected, and it is compliant | — |
| `not-applicable` | the provider contract declares the control untestable, with its justification | — |
| `not-evaluated` | the control is implemented, but the data it depends on was not confirmed | exoscale / terraform · exoscale / live · outscale / terraform · outscale / live · scaleway / terraform · scaleway / live |

An observable control still returns `not-evaluated` on an inventory that contains no
resource of the targeted type: "nothing to look at" is not "compliant".

## How to investigate

- Cross-cutting control: it does not read one particular resource type.
- No attribute lock: the control is judged on the presence of a deviation, absence of a bad configuration counting as compliance.
- What each source projects is readable in the descriptor: [`providers/exoscale.yaml`](../../providers/exoscale.yaml) · [`providers/outscale.yaml`](../../providers/outscale.yaml) · [`providers/scaleway.yaml`](../../providers/scaleway.yaml)
- The rule that emits this code lives in [`internal/commonrules/rules/`](../../internal/commonrules/rules): it is **common** to every provider, only the source changes.

## How to remediate

Add the missing governance tags; check their presence at creation time. If your convention differs, declare it in the policy file rather than disabling the control.

| Provider | Deployable setup |
|---|---|
| exoscale | [`references/remediation/exoscale/governance_resource_required_tags`](../../references/remediation/exoscale/governance_resource_required_tags) |
| outscale | _no proof filed yet_ |
| scaleway | _no proof filed yet_ |

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

In the `assessment` output, look for `"control": "governance_resource_required_tags"`: its `status` must be `pass`.
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

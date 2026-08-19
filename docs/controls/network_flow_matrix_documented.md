> 🇬🇧 English · [🇫🇷 Français](network_flow_matrix_documented.fr.md)

<!-- GENERATED PAGE — do not edit by hand. Regenerate with `mise run gen-docs`. -->

# `network_flow_matrix_documented`

**Inbound flow without a justification in the flow matrix**

[Back to the catalogue](index.md)

| Field | Value |
|---|---|
| Code | `network_flow_matrix_documented` |
| Family | `reseau` |
| Severity | `medium` |
| SCSL requirement (frozen index) | `CLD-NET-5` |
| Resource type read | `security_group_rule` |
| Deciding attribute | `description` |
| State | active |
| Declared for | `exoscale` |
| Remediation proofs | 1 / 1 |

## The risk

An inbound flow rule carries no justification (description): the matrix of allowed flows (services, protocols, ports and justification) is not maintained, which prevents any review of, and accountability for, network openings.

This description comes from the reference: it is the text the report quotes, in the
reader's language.

## Normative mappings

Taken verbatim from the reference. The SCSL requirement comes from the **frozen** index:
a control maps onto an existing requirement, never onto one created for it. These mappings
are **indicative**: a Pépin report is not proof of qualification.

| Framework | References |
|---|---|
| `scsl` | `CLD-NET-5` |
| `iso_27001_2022` | `A.8.20`, `A.8.22` |
| `secnumcloud_3_2` | `13.1`, `13.2` |

## Where Pépin can measure it

A ✅ cell means the source produces the targeted type, the provider contract marks it
`verifie`, and the deciding attribute is projected. ◐ means "Pépin cannot decide from this
source", ∅ "not testable, with justification", ✗ "not declared, or type absent from this
source".

| Provider | Terraform plan | Live collection |
|---|:-:|:-:|
| exoscale | ✅ | ✅ |
| outscale | ✗ | ✗ |
| scaleway | ✗ | ✗ |
| kubernetes | n/a | ✗ |

Every cell that is not ✅ **while the control is declared for that provider** carries its
reason:

_None: every declared cell is fully observable._

## What Pépin can conclude

| Status | What the status asserts | Reachable from |
|---|---|---|
| `fail` | a deviation was detected on a real resource | exoscale / terraform · exoscale / live |
| `pass` | the deciding data was collected, and it is compliant | exoscale / terraform · exoscale / live |
| `not-applicable` | the provider contract declares the control untestable, with its justification | — |
| `not-evaluated` | the control is implemented, but the data it depends on was not confirmed | — |

An observable control still returns `not-evaluated` on an inventory that contains no
resource of the targeted type: "nothing to look at" is not "compliant".

## How to investigate

- Normalized resource type the rule reads: `security_group_rule`
- Attribute the decision depends on: `description`
- Without that attribute on a resource of the targeted type, the scan returns `not-evaluated` rather than `pass` (`internal/assess`, `requiredAttr` table).
- What each source projects is readable in the descriptor: [`providers/exoscale.yaml`](../../providers/exoscale.yaml)
- The rule that emits this code lives in [`internal/commonrules/rules/`](../../internal/commonrules/rules): it is **common** to every provider, only the source changes.

## How to remediate

Document every security group rule (service served, reason) in its description; keep the flow matrix up to date and review it periodically.

| Provider | Deployable setup |
|---|---|
| exoscale | [`references/remediation/exoscale/network_flow_matrix_documented`](../../references/remediation/exoscale/network_flow_matrix_documented) |

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

In the `assessment` output, look for `"control": "network_flow_matrix_documented"`: its `status` must be `pass`.
If it stays `not-evaluated`, the deciding data was not collected and the fix is **not**
demonstrated: the reasons table above says why.

## See also

- [The assessment model](../concepts/assessment-model.md): what each status asserts.
- [Coverage matrix](../coverage.md): the same information, across every control.
- [Terraform plan or live scan](../concepts/terraform-vs-live.md): choosing the source.
- [Adding a control](../contributing/adding-a-control.md): the end-to-end procedure.

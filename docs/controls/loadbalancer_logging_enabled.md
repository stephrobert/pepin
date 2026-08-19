> 🇬🇧 English · [🇫🇷 Français](loadbalancer_logging_enabled.fr.md)

<!-- GENERATED PAGE — do not edit by hand. Regenerate with `mise run gen-docs`. -->

# `loadbalancer_logging_enabled`

**Access logging disabled**

[Back to the catalogue](index.md)

| Field | Value |
|---|---|
| Code | `loadbalancer_logging_enabled` |
| Family | `journalisation` |
| Severity | `medium` |
| SCSL requirement (frozen index) | `CLD-LOG-1` |
| Resource type read | `load_balancer` |
| Deciding attribute | _none: judged on the presence of a deviation_ |
| State | active |
| Declared for | `outscale` |
| Remediation proofs | 0 / 1 |

## The risk

The audit trail and the access logs of the exposed services are not enabled: an incident could not be investigated.

This description comes from the reference: it is the text the report quotes, in the
reader's language.

## Normative mappings

Taken verbatim from the reference. The SCSL requirement comes from the **frozen** index:
a control maps onto an existing requirement, never onto one created for it. These mappings
are **indicative**: a Pépin report is not proof of qualification.

| Framework | References |
|---|---|
| `scsl` | `CLD-LOG-1` |
| `iso_27001_2022` | `A.8.15`, `A.8.16` |
| `secnumcloud_3_2` | `12.6` |

## Where Pépin can measure it

A ✅ cell means the source produces the targeted type, the provider contract marks it
`verifie`, and the deciding attribute is projected. ◐ means "Pépin cannot decide from this
source", ∅ "not testable, with justification", ✗ "not declared, or type absent from this
source".

| Provider | Terraform plan | Live collection |
|---|:-:|:-:|
| exoscale | ∅ | ∅ |
| outscale | ✗ | ✅ |
| scaleway | ✗ | ✗ |
| kubernetes | n/a | ✗ |

Every cell that is not ✅ **while the control is declared for that provider** carries its
reason:

| Provider | Source | Status | Reason |
|---|---|---|---|
| exoscale | terraform | ∅ `not-applicable` | resource type "load_balancer" absent from the exoscale API |
| exoscale | live | ∅ `not-applicable` | resource type "load_balancer" absent from the exoscale API |
| outscale | terraform | ✗ `unsupported` | this source produces no resource of type "load_balancer" |

## What Pépin can conclude

| Status | What the status asserts | Reachable from |
|---|---|---|
| `fail` | a deviation was detected on a real resource | outscale / live |
| `pass` | the deciding data was collected, and it is compliant | outscale / live |
| `not-applicable` | the provider contract declares the control untestable, with its justification | exoscale / terraform · exoscale / live |
| `not-evaluated` | the control is implemented, but the data it depends on was not confirmed | — |

An observable control still returns `not-evaluated` on an inventory that contains no
resource of the targeted type: "nothing to look at" is not "compliant".

## How to investigate

- Normalized resource type the rule reads: `load_balancer`
- No attribute lock: the control is judged on the presence of a deviation, absence of a bad configuration counting as compliance.
- What each source projects is readable in the descriptor: [`providers/outscale.yaml`](../../providers/outscale.yaml)
- The rule that emits this code lives in [`internal/commonrules/rules/`](../../internal/commonrules/rules): it is **common** to every provider, only the source changes.

## How to remediate

Enable logging (audit trail, access logs); centralise the logs and keep them according to a retention policy.

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

In the `assessment` output, look for `"control": "loadbalancer_logging_enabled"`: its `status` must be `pass`.
If it stays `not-evaluated`, the deciding data was not collected and the fix is **not**
demonstrated: the reasons table above says why.

## See also

- [The assessment model](../concepts/assessment-model.md): what each status asserts.
- [Coverage matrix](../coverage.md): the same information, across every control.
- [Terraform plan or live scan](../concepts/terraform-vs-live.md): choosing the source.
- [Adding a control](../contributing/adding-a-control.md): the end-to-end procedure.

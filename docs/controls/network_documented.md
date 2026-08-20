> 🇬🇧 English · [🇫🇷 Français](network_documented.fr.md)

<!-- GENERATED PAGE — do not edit by hand. Regenerate with `mise run gen-docs`. -->

# `network_documented`

**Network without mapping tags**

[Back to the catalogue](index.md)

| Field | Value |
|---|---|
| Code | `network_documented` |
| Family | `reseau` |
| Severity | `low` |
| SCSL requirement (frozen index) | `CLD-NET-5` |
| Resource type read | `network` |
| Deciding attribute | `tags` |
| State | active |
| Declared for | `exoscale`, `outscale`, `scaleway` |
| Remediation proofs | 1 / 3 |

## The risk

WHAT IS CHECKED: every network (VPC / Net / private network) carries the governance tags that document it - by default owner, project and environment. An off-topic tag documents nothing: the mere presence of some tag is not enough. The comparison ignores case and separators, and the tagging profile is configurable (`controls.tagging.network_required_tags`); the shipped profile is a recommendation, not a standard. Without collected tags the control returns "not evaluated" rather than concluding.

This description comes from the reference: it is the text the report quotes, in the
reader's language.

## Normative mappings

Taken verbatim from the reference. The SCSL requirement comes from the **frozen** index:
a control maps onto an existing requirement, never onto one created for it. These mappings
are **indicative**: a Pépin report is not proof of qualification.

| Framework | References |
|---|---|
| `scsl` | `CLD-NET-5` |
| `iso_27001_2022` | `A.5.9` |
| `secnumcloud_3_2` | `13.1` |

## Where Pépin can measure it

A ✅ cell means the source produces the targeted type, the provider contract marks it
`verifie`, and the deciding attribute is projected. ◐ means "Pépin cannot decide from this
source", ∅ "not testable, with justification", ✗ "not declared, or type absent from this
source".

| Provider | Terraform plan | Live collection |
|---|:-:|:-:|
| exoscale | ✅ | ✅ |
| outscale | ✅ | ✅ |
| scaleway | ◐ | ✗ |
| kubernetes | n/a | ✗ |

Every cell that is not ✅ **while the control is declared for that provider** carries its
reason:

| Provider | Source | Status | Reason |
|---|---|---|---|
| scaleway | terraform | ◐ `partial` | provider contract: type "network" is not declared `verifie` (state: a_verifier) |
| scaleway | live | ✗ `unsupported` | this source produces no resource of type "network" |

## What Pépin can conclude

| Status | What the status asserts | Reachable from |
|---|---|---|
| `fail` | a deviation was detected on a real resource | exoscale / terraform · exoscale / live · outscale / terraform · outscale / live · scaleway / terraform |
| `pass` | the deciding data was collected, and it is compliant | exoscale / terraform · exoscale / live · outscale / terraform · outscale / live |
| `not-applicable` | the provider contract declares the control untestable, with its justification | — |
| `not-evaluated` | the control is implemented, but the data it depends on was not confirmed | scaleway / terraform |

An observable control still returns `not-evaluated` on an inventory that contains no
resource of the targeted type: "nothing to look at" is not "compliant".

## How to investigate

- Normalized resource type the rule reads: `network`
- Attribute the decision depends on: `tags`
- Without that attribute on a resource of the targeted type, the scan returns `not-evaluated` rather than `pass` (`internal/assess`, `requiredAttr` table).
- What each source projects is readable in the descriptor: [`providers/exoscale.yaml`](../../providers/exoscale.yaml) · [`providers/outscale.yaml`](../../providers/outscale.yaml) · [`providers/scaleway.yaml`](../../providers/scaleway.yaml)
- The rule that emits this code lives in [`internal/commonrules/rules/`](../../internal/commonrules/rules): it is **common** to every provider, only the source changes.

## How to remediate

Tag every network with its owner, project and environment; keep the network mapping up to date and review it periodically.

| Provider | Deployable setup |
|---|---|
| exoscale | [`references/remediation/exoscale/network_documented`](../../references/remediation/exoscale/network_documented) |
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

In the `assessment` output, look for `"control": "network_documented"`: its `status` must be `pass`.
If it stays `not-evaluated`, the deciding data was not collected and the fix is **not**
demonstrated: the reasons table above says why.

## See also

- [The assessment model](../concepts/assessment-model.md): what each status asserts.
- [Coverage matrix](../coverage.md): the same information, across every control.
- [Terraform plan or live scan](../concepts/terraform-vs-live.md): choosing the source.
- [Adding a control](../contributing/adding-a-control.md): the end-to-end procedure.

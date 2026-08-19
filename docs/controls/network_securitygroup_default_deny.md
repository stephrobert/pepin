> 🇬🇧 English · [🇫🇷 Français](network_securitygroup_default_deny.fr.md)

<!-- GENERATED PAGE — do not edit by hand. Regenerate with `mise run gen-docs`. -->

# `network_securitygroup_default_deny`

**Security group inbound default policy set to "accept"**

[Back to the catalogue](index.md)

| Field | Value |
|---|---|
| Code | `network_securitygroup_default_deny` |
| Family | `reseau` |
| Severity | `high` |
| SCSL requirement (frozen index) | `CLD-NET-2` |
| Resource type read | `security_group` |
| Deciding attribute | `inbound_default_policy` |
| State | active |
| Declared for | `scaleway` |
| Remediation proofs | 0 / 1 |

## The risk

A security group's default inbound policy is "accept": any traffic not explicitly filtered is admitted (implicit any/any), whatever the explicit rule set says.

This description comes from the reference: it is the text the report quotes, in the
reader's language.

## Normative mappings

Taken verbatim from the reference. The SCSL requirement comes from the **frozen** index:
a control maps onto an existing requirement, never onto one created for it. These mappings
are **indicative**: a Pépin report is not proof of qualification.

| Framework | References |
|---|---|
| `scsl` | `CLD-NET-2` |
| `cis_controls_v8` | `4.4`, `12.2` |
| `iso_27001_2022` | `A.8.20`, `A.8.22` |
| `secnumcloud_3_2` | `13.2` |

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
| scaleway | live | ✗ `unsupported` | this source produces no resource of type "security_group" |

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

- Normalized resource type the rule reads: `security_group`
- Attribute the decision depends on: `inbound_default_policy`
- Without that attribute on a resource of the targeted type, the scan returns `not-evaluated` rather than `pass` (`internal/assess`, `requiredAttr` table).
- What each source projects is readable in the descriptor: [`providers/scaleway.yaml`](../../providers/scaleway.yaml)
- The rule that emits this code lives in [`internal/commonrules/rules/`](../../internal/commonrules/rules): it is **common** to every provider, only the source changes.

## How to remediate

Switch the default inbound policy to "drop" and open only the legitimate flows through explicit rules.

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

In the `assessment` output, look for `"control": "network_securitygroup_default_deny"`: its `status` must be `pass`.
If it stays `not-evaluated`, the deciding data was not collected and the fix is **not**
demonstrated: the reasons table above says why.

## See also

- [The assessment model](../concepts/assessment-model.md): what each status asserts.
- [Coverage matrix](../coverage.md): the same information, across every control.
- [Terraform plan or live scan](../concepts/terraform-vs-live.md): choosing the source.
- [Adding a control](../contributing/adding-a-control.md): the end-to-end procedure.

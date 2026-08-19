> 🇬🇧 English · [🇫🇷 Français](governance_provider_sovereignty.fr.md)

<!-- GENERATED PAGE — do not edit by hand. Regenerate with `mise run gen-docs`. -->

# `governance_provider_sovereignty`

**Provider sovereignty not established**

[Back to the catalogue](index.md)

| Field | Value |
|---|---|
| Code | `governance_provider_sovereignty` |
| Family | `gouvernance` |
| Severity | `high` |
| SCSL requirement (frozen index) | `CLD-GVN-4` |
| Resource type read | _none: cross-cutting control_ |
| Deciding attribute | _none: judged on the presence of a deviation_ |
| State | active |
| Declared for | `exoscale`, `outscale`, `scaleway` |
| Remediation proofs | 0 / 3 |

## The risk

The cloud provider is not established in the European Union, or its non-EU capital control is decisive, or it is exposed to an extraterritorial law without recognised immunity (SecNumCloud qualification): the data and the control plane may fall under a foreign jurisdiction.

This description comes from the reference: it is the text the report quotes, in the
reader's language.

## Normative mappings

Taken verbatim from the reference. The SCSL requirement comes from the **frozen** index:
a control maps onto an existing requirement, never onto one created for it. These mappings
are **indicative**: a Pépin report is not proof of qualification.

| Framework | References |
|---|---|
| `scsl` | `CLD-GVN-4` |
| `secnumcloud_3_2` | `19.2`, `19.6` |

## Where Pépin can measure it

A ✅ cell means the source produces the targeted type, the provider contract marks it
`verifie`, and the deciding attribute is projected. ◐ means "Pépin cannot decide from this
source", ∅ "not testable, with justification", ✗ "not declared, or type absent from this
source".

| Provider | Terraform plan | Live collection |
|---|:-:|:-:|
| exoscale | ✅ | ✅ |
| outscale | ✅ | ✅ |
| scaleway | ✅ | ✅ |
| kubernetes | n/a | ✗ |

Every cell that is not ✅ **while the control is declared for that provider** carries its
reason:

_None: every declared cell is fully observable._

## What Pépin can conclude

| Status | What the status asserts | Reachable from |
|---|---|---|
| `fail` | a deviation was detected on a real resource | exoscale / terraform · exoscale / live · outscale / terraform · outscale / live · scaleway / terraform · scaleway / live |
| `pass` | the deciding data was collected, and it is compliant | exoscale / terraform · exoscale / live · outscale / terraform · outscale / live · scaleway / terraform · scaleway / live |
| `not-applicable` | the provider contract declares the control untestable, with its justification | — |
| `not-evaluated` | the control is implemented, but the data it depends on was not confirmed | — |

An observable control still returns `not-evaluated` on an inventory that contains no
resource of the targeted type: "nothing to look at" is not "compliant".

## How to investigate

- Cross-cutting control: it does not read one particular resource type.
- No attribute lock: the control is judged on the presence of a deviation, absence of a bad configuration counting as compliance.
- What each source projects is readable in the descriptor: [`providers/exoscale.yaml`](../../providers/exoscale.yaml) · [`providers/outscale.yaml`](../../providers/outscale.yaml) · [`providers/scaleway.yaml`](../../providers/scaleway.yaml)
- The rule that emits this code lives in [`internal/commonrules/rules/`](../../internal/commonrules/rules): it is **common** to every provider, only the source changes.

## How to remediate

For a sovereignty requirement, pick a provider established in the EU, under European capital control, ideally SecNumCloud qualified (immunity recognised by the ANSSI).

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
./pepin scan exoscale --terraform plan.json --format assessment

# from the provider API: effective configuration
./pepin scan exoscale --live --format assessment
```

In the `assessment` output, look for `"control": "governance_provider_sovereignty"`: its `status` must be `pass`.
If it stays `not-evaluated`, the deciding data was not collected and the fix is **not**
demonstrated: the reasons table above says why.

## See also

- [The assessment model](../concepts/assessment-model.md): what each status asserts.
- [Coverage matrix](../coverage.md): the same information, across every control.
- [Terraform plan or live scan](../concepts/terraform-vs-live.md): choosing the source.
- [Adding a control](../contributing/adding-a-control.md): the end-to-end procedure.

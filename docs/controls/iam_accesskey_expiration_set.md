> 🇬🇧 English · [🇫🇷 Français](iam_accesskey_expiration_set.fr.md)

<!-- GENERATED PAGE — do not edit by hand. Regenerate with `mise run gen-docs`. -->

# `iam_accesskey_expiration_set`

**Access key without expiry or rotation**

[Back to the catalogue](index.md)

| Field | Value |
|---|---|
| Code | `iam_accesskey_expiration_set` |
| Family | `iam` |
| Severity | `critical` |
| SCSL requirement (frozen index) | `CLD-IAM-2` |
| Resource type read | `access_key` |
| Deciding attribute | _none: judged on the presence of a deviation_ |
| State | active |
| Declared for | `outscale`, `scaleway` |
| Remediation proofs | 0 / 2 |

## The risk

A long-lived access key has no expiry date: a leak stays exploitable indefinitely.

This description comes from the reference: it is the text the report quotes, in the
reader's language.

## Normative mappings

Taken verbatim from the reference. The SCSL requirement comes from the **frozen** index:
a control maps onto an existing requirement, never onto one created for it. These mappings
are **indicative**: a Pépin report is not proof of qualification.

| Framework | References |
|---|---|
| `scsl` | `CLD-IAM-2` |
| `iso_27001_2022` | `A.5.17`, `A.8.2` |
| `secnumcloud_3_2` | `9.5`, `10.5` |

## Where Pépin can measure it

A ✅ cell means the source produces the targeted type, the provider contract marks it
`verifie`, and the deciding attribute is projected. ◐ means "Pépin cannot decide from this
source", ∅ "not testable, with justification", ✗ "not declared, or type absent from this
source".

| Provider | Terraform plan | Live collection |
|---|:-:|:-:|
| exoscale | ✗ | ✗ |
| outscale | ✗ | ✅ |
| scaleway | ✅ | ✅ |
| kubernetes | n/a | ✗ |

Every cell that is not ✅ **while the control is declared for that provider** carries its
reason:

| Provider | Source | Status | Reason |
|---|---|---|---|
| outscale | terraform | ✗ `unsupported` | this source produces no resource of type "access_key" |

## What Pépin can conclude

| Status | What the status asserts | Reachable from |
|---|---|---|
| `fail` | a deviation was detected on a real resource | outscale / live · scaleway / terraform · scaleway / live |
| `pass` | the deciding data was collected, and it is compliant | outscale / live · scaleway / terraform · scaleway / live |
| `not-applicable` | the provider contract declares the control untestable, with its justification | — |
| `not-evaluated` | the control is implemented, but the data it depends on was not confirmed | — |

An observable control still returns `not-evaluated` on an inventory that contains no
resource of the targeted type: "nothing to look at" is not "compliant".

## How to investigate

- Normalized resource type the rule reads: `access_key`
- No attribute lock: the control is judged on the presence of a deviation, absence of a bad configuration counting as compliance.
- What each source projects is readable in the descriptor: [`providers/outscale.yaml`](../../providers/outscale.yaml) · [`providers/scaleway.yaml`](../../providers/scaleway.yaml)
- The rule that emits this code lives in [`internal/commonrules/rules/`](../../internal/commonrules/rules): it is **common** to every provider, only the source changes.

## How to remediate

Enforce an expiry and a rotation for access keys; prefer a short-lived identity (OIDC) over static secrets.

| Provider | Deployable setup |
|---|---|
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
./pepin scan outscale --live --format assessment
```

In the `assessment` output, look for `"control": "iam_accesskey_expiration_set"`: its `status` must be `pass`.
If it stays `not-evaluated`, the deciding data was not collected and the fix is **not**
demonstrated: the reasons table above says why.

## See also

- [The assessment model](../concepts/assessment-model.md): what each status asserts.
- [Coverage matrix](../coverage.md): the same information, across every control.
- [Terraform plan or live scan](../concepts/terraform-vs-live.md): choosing the source.
- [Adding a control](../contributing/adding-a-control.md): the end-to-end procedure.

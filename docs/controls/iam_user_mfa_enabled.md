> 🇬🇧 English · [🇫🇷 Français](iam_user_mfa_enabled.fr.md)

<!-- GENERATED PAGE — do not edit by hand. Regenerate with `mise run gen-docs`. -->

# `iam_user_mfa_enabled`

**MFA not enabled on an account**

[Back to the catalogue](index.md)

| Field | Value |
|---|---|
| Code | `iam_user_mfa_enabled` |
| Family | `iam` |
| Severity | `high` |
| SCSL requirement (frozen index) | `CLD-IAM-3` |
| Resource type read | `iam_user` |
| Deciding attribute | `mfa_enabled` |
| State | active |
| Declared for | `exoscale`, `scaleway` |
| Remediation proofs | 1 / 2 |

## The risk

A cloud user account does not have multi-factor authentication (MFA) enabled: a compromised password is enough to reach the console or the API.

This description comes from the reference: it is the text the report quotes, in the
reader's language.

## Normative mappings

Taken verbatim from the reference. The SCSL requirement comes from the **frozen** index:
a control maps onto an existing requirement, never onto one created for it. These mappings
are **indicative**: a Pépin report is not proof of qualification.

| Framework | References |
|---|---|
| `scsl` | `CLD-IAM-3` |
| `cis_controls_v8` | `6.5` |
| `iso_27001_2022` | `A.5.17`, `A.8.5` |
| `secnumcloud_3_2` | `9.5`, `9.6` |

## Where Pépin can measure it

A ✅ cell means the source produces the targeted type, the provider contract marks it
`verifie`, and the deciding attribute is projected. ◐ means "Pépin cannot decide from this
source", ∅ "not testable, with justification", ✗ "not declared, or type absent from this
source".

| Provider | Terraform plan | Live collection |
|---|:-:|:-:|
| exoscale | ✗ | ✅ |
| outscale | ∅ | ∅ |
| scaleway | ✗ | ✅ |
| kubernetes | n/a | ✗ |

Every cell that is not ✅ **while the control is declared for that provider** carries its
reason:

| Provider | Source | Status | Reason |
|---|---|---|---|
| exoscale | terraform | ✗ `unsupported` | this source produces no resource of type "iam_user" |
| outscale | terraform | ∅ `not-applicable` | resource type "iam_user" absent from the outscale API |
| outscale | live | ∅ `not-applicable` | resource type "iam_user" absent from the outscale API |
| scaleway | terraform | ✗ `unsupported` | this source produces no resource of type "iam_user" |

## What Pépin can conclude

| Status | What the status asserts | Reachable from |
|---|---|---|
| `fail` | a deviation was detected on a real resource | exoscale / live · scaleway / live |
| `pass` | the deciding data was collected, and it is compliant | exoscale / live · scaleway / live |
| `not-applicable` | the provider contract declares the control untestable, with its justification | outscale / terraform · outscale / live |
| `not-evaluated` | the control is implemented, but the data it depends on was not confirmed | — |

An observable control still returns `not-evaluated` on an inventory that contains no
resource of the targeted type: "nothing to look at" is not "compliant".

## How to investigate

- Normalized resource type the rule reads: `iam_user`
- Attribute the decision depends on: `mfa_enabled`
- Without that attribute on a resource of the targeted type, the scan returns `not-evaluated` rather than `pass` (`internal/assess`, `requiredAttr` table).
- What each source projects is readable in the descriptor: [`providers/exoscale.yaml`](../../providers/exoscale.yaml) · [`providers/scaleway.yaml`](../../providers/scaleway.yaml)
- The rule that emits this code lives in [`internal/commonrules/rules/`](../../internal/commonrules/rules): it is **common** to every provider, only the source changes.

## How to remediate

Enable MFA on every account, starting with administrative and console access; enforce it by policy at the organisation level.

| Provider | Deployable setup |
|---|---|
| exoscale | [`references/remediation/exoscale/iam_user_mfa_enabled.md`](../../references/remediation/exoscale/iam_user_mfa_enabled.md) |
| scaleway | _no proof filed yet_ |

A remediation proof is a self-contained, **compliant** Terraform module that deploys as
is, or a note anchored on the official documentation. See
[the remediation guide](../guides/remediation.md).

## How to verify the fix

```bash
# from the provider API: effective configuration
./pepin scan exoscale --live --format assessment
```

In the `assessment` output, look for `"control": "iam_user_mfa_enabled"`: its `status` must be `pass`.
If it stays `not-evaluated`, the deciding data was not collected and the fix is **not**
demonstrated: the reasons table above says why.

## See also

- [The assessment model](../concepts/assessment-model.md): what each status asserts.
- [Coverage matrix](../coverage.md): the same information, across every control.
- [Terraform plan or live scan](../concepts/terraform-vs-live.md): choosing the source.
- [Adding a control](../contributing/adding-a-control.md): the end-to-end procedure.

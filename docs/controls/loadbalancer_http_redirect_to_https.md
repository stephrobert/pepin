> 🇬🇧 English · [🇫🇷 Français](loadbalancer_http_redirect_to_https.fr.md)

<!-- GENERATED PAGE — do not edit by hand. Regenerate with `mise run gen-docs`. -->

# `loadbalancer_http_redirect_to_https`

**HTTP listener without an HTTPS redirect**

[Back to the catalogue](index.md)

| Field | Value |
|---|---|
| Code | `loadbalancer_http_redirect_to_https` |
| Family | `chiffrement` |
| Severity | `medium` |
| SCSL requirement (frozen index) | `CLD-CHF-1` |
| Resource type read | `load_balancer` |
| Deciding attribute | `redirect_to_https` |
| State | dormant (declared for no provider) |
| Declared for | — |
| Remediation proofs | 0 / 0 |

## The risk

A load balancer exposes an HTTP listener (port 80): without a redirect to HTTPS, traffic can travel in cleartext.

This description comes from the reference: it is the text the report quotes, in the
reader's language.

## Normative mappings

Taken verbatim from the reference. The SCSL requirement comes from the **frozen** index:
a control maps onto an existing requirement, never onto one created for it. These mappings
are **indicative**: a Pépin report is not proof of qualification.

| Framework | References |
|---|---|
| `scsl` | `CLD-CHF-1` |
| `cis_controls_v8` | `3.10` |
| `iso_27001_2022` | `A.8.24` |
| `secnumcloud_3_2` | `10.2` |

## Where Pépin can measure it

A ✅ cell means the source produces the targeted type, the provider contract marks it
`verifie`, and the deciding attribute is projected. ◐ means "Pépin cannot decide from this
source", ∅ "not testable, with justification", ✗ "not declared, or type absent from this
source".

| Provider | Terraform plan | Live collection |
|---|:-:|:-:|
| exoscale | ∅ | ∅ |
| outscale | ∅ | ∅ |
| scaleway | ✗ | ✗ |
| kubernetes | n/a | ✗ |

Every cell that is not ✅ **while the control is declared for that provider** carries its
reason:

| Provider | Source | Status | Reason |
|---|---|---|---|
| exoscale | terraform | ∅ `not-applicable` | resource type "load_balancer" absent from the exoscale API |
| exoscale | live | ∅ `not-applicable` | resource type "load_balancer" absent from the exoscale API |
| outscale | terraform | ∅ `not-applicable` | The Outscale LBU cannot redirect: `ListenerRule.Action` is documented as "always forward" in the OAPI contract (no redirect action), and no redirect attribute exists on `Listener`. The mechanism does not exist, so the control is not applicable (CHF-1). |
| outscale | live | ∅ `not-applicable` | The Outscale LBU cannot redirect: `ListenerRule.Action` is documented as "always forward" in the OAPI contract (no redirect action), and no redirect attribute exists on `Listener`. The mechanism does not exist, so the control is not applicable (CHF-1). |

## What Pépin can conclude

| Status | What the status asserts | Reachable from |
|---|---|---|
| `fail` | a deviation was detected on a real resource | — |
| `pass` | the deciding data was collected, and it is compliant | — |
| `not-applicable` | the provider contract declares the control untestable, with its justification | exoscale / terraform · exoscale / live · outscale / terraform · outscale / live |
| `not-evaluated` | the control is implemented, but the data it depends on was not confirmed | — |

An observable control still returns `not-evaluated` on an inventory that contains no
resource of the targeted type: "nothing to look at" is not "compliant".

## How to investigate

- Normalized resource type the rule reads: `load_balancer`
- Attribute the decision depends on: `redirect_to_https`
- Without that attribute on a resource of the targeted type, the scan returns `not-evaluated` rather than `pass` (`internal/assess`, `requiredAttr` table).
- The rule that emits this code lives in [`internal/commonrules/rules/`](../../internal/commonrules/rules): it is **common** to every provider, only the source changes.

## How to remediate

Set up a 301 redirect from the HTTP:80 listener to HTTPS.

_Dormant control: no provider declared, so no proof expected._

A remediation proof is a self-contained, **compliant** Terraform module that deploys as
is, or a note anchored on the official documentation. See
[the remediation guide](../guides/remediation.md).

## How to verify the fix

No source can conclude `pass` on this control today: a scan can make the deviation
disappear, it cannot **demonstrate** compliance. The reasons table above says what is
missing.
## See also

- [The assessment model](../concepts/assessment-model.md): what each status asserts.
- [Coverage matrix](../coverage.md): the same information, across every control.
- [Terraform plan or live scan](../concepts/terraform-vs-live.md): choosing the source.
- [Adding a control](../contributing/adding-a-control.md): the end-to-end procedure.

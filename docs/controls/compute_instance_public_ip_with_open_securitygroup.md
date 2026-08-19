> 🇬🇧 English · [🇫🇷 Français](compute_instance_public_ip_with_open_securitygroup.fr.md)

<!-- GENERATED PAGE — do not edit by hand. Regenerate with `mise run gen-docs`. -->

# `compute_instance_public_ip_with_open_securitygroup`

**Instance publicly exposed without restrictive filtering**

[Back to the catalogue](index.md)

| Field | Value |
|---|---|
| Code | `compute_instance_public_ip_with_open_securitygroup` |
| Family | `reseau` |
| Severity | `critical` |
| SCSL requirement (frozen index) | `CLD-NET-3` |
| Resource type read | `compute_instance` |
| Deciding attribute | `public_ip` |
| State | active |
| Declared for | `exoscale`, `outscale`, `scaleway` |
| Remediation proofs | 0 / 3 |

## The risk

An instance carries a public IP and a security group open to 0.0.0.0/0: it is directly reachable from the internet.

This description comes from the reference: it is the text the report quotes, in the
reader's language.

## Normative mappings

Taken verbatim from the reference. The SCSL requirement comes from the **frozen** index:
a control maps onto an existing requirement, never onto one created for it. These mappings
are **indicative**: a Pépin report is not proof of qualification.

| Framework | References |
|---|---|
| `scsl` | `CLD-NET-3` |
| `cis_controls_v8` | `4.4`, `12.2` |
| `iso_27001_2022` | `A.8.20`, `A.8.22` |
| `iso_27017` | `CLD.9.5.2` |
| `secnumcloud_3_2` | `13.2` |

## Where Pépin can measure it

A ✅ cell means the source produces the targeted type, the provider contract marks it
`verifie`, and the deciding attribute is projected. ◐ means "Pépin cannot decide from this
source", ∅ "not testable, with justification", ✗ "not declared, or type absent from this
source".

| Provider | Terraform plan | Live collection |
|---|:-:|:-:|
| exoscale | ✅ | ✅ |
| outscale | ✅ | ✅ |
| scaleway | ◐ | ✅ |
| kubernetes | n/a | ✗ |

Every cell that is not ✅ **while the control is declared for that provider** carries its
reason:

| Provider | Source | Status | Reason |
|---|---|---|---|
| scaleway | terraform | ◐ `partial` | deciding attribute "public_ip" not projected by this source: a capability guard, so the scan returns "not-evaluated" |

## What Pépin can conclude

| Status | What the status asserts | Reachable from |
|---|---|---|
| `fail` | a deviation was detected on a real resource | exoscale / terraform · exoscale / live · outscale / terraform · outscale / live · scaleway / terraform · scaleway / live |
| `pass` | the deciding data was collected, and it is compliant | exoscale / terraform · exoscale / live · outscale / terraform · outscale / live · scaleway / live |
| `not-applicable` | the provider contract declares the control untestable, with its justification | — |
| `not-evaluated` | the control is implemented, but the data it depends on was not confirmed | scaleway / terraform |

An observable control still returns `not-evaluated` on an inventory that contains no
resource of the targeted type: "nothing to look at" is not "compliant".

## How to investigate

- Normalized resource type the rule reads: `compute_instance`
- Attribute the decision depends on: `public_ip`
- Without that attribute on a resource of the targeted type, the scan returns `not-evaluated` rather than `pass` (`internal/assess`, `requiredAttr` table).
- What each source projects is readable in the descriptor: [`providers/exoscale.yaml`](../../providers/exoscale.yaml) · [`providers/outscale.yaml`](../../providers/outscale.yaml) · [`providers/scaleway.yaml`](../../providers/scaleway.yaml)
- The rule that emits this code lives in [`internal/commonrules/rules/`](../../internal/commonrules/rules): it is **common** to every provider, only the source changes.

## How to remediate

Remove the public IP if it is not needed; restrict the security group; place the instance behind a load balancer or a VPN.

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

In the `assessment` output, look for `"control": "compute_instance_public_ip_with_open_securitygroup"`: its `status` must be `pass`.
If it stays `not-evaluated`, the deciding data was not collected and the fix is **not**
demonstrated: the reasons table above says why.

## See also

- [The assessment model](../concepts/assessment-model.md): what each status asserts.
- [Coverage matrix](../coverage.md): the same information, across every control.
- [Terraform plan or live scan](../concepts/terraform-vs-live.md): choosing the source.
- [Adding a control](../contributing/adding-a-control.md): the end-to-end procedure.

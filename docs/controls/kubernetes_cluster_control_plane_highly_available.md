> 🇬🇧 English · [🇫🇷 Français](kubernetes_cluster_control_plane_highly_available.fr.md)

<!-- GENERATED PAGE — do not edit by hand. Regenerate with `mise run gen-docs`. -->

# `kubernetes_cluster_control_plane_highly_available`

**Kubernetes control plane not highly available**

[Back to the catalogue](index.md)

| Field | Value |
|---|---|
| Code | `kubernetes_cluster_control_plane_highly_available` |
| Family | `compute` |
| Severity | `high` |
| SCSL requirement (frozen index) | `CLD-K8S-2` |
| Resource type read | `kubernetes_cluster` |
| Deciding attribute | `control_plane_multi_az` |
| State | active |
| Declared for | `exoscale`, `outscale` |
| Remediation proofs | 1 / 2 |

## The risk

The managed cluster's control plane is not spread across several zones (single master / single AZ): losing one zone interrupts the control plane.

This description comes from the reference: it is the text the report quotes, in the
reader's language.

## Normative mappings

Taken verbatim from the reference. The SCSL requirement comes from the **frozen** index:
a control maps onto an existing requirement, never onto one created for it. These mappings
are **indicative**: a Pépin report is not proof of qualification.

| Framework | References |
|---|---|
| `scsl` | `CLD-K8S-2` |
| `iso_27001_2022` | `A.8.14` |
| `secnumcloud_3_2` | `17.4` |

## Where Pépin can measure it

A ✅ cell means the source produces the targeted type, the provider contract marks it
`verifie`, and the deciding attribute is projected. ◐ means "Pépin cannot decide from this
source", ∅ "not testable, with justification", ✗ "not declared, or type absent from this
source".

| Provider | Terraform plan | Live collection |
|---|:-:|:-:|
| exoscale | ✅ | ✅ |
| outscale | ✗ | ✅ |
| scaleway | ✗ | ✗ |
| kubernetes | n/a | ✗ |

Every cell that is not ✅ **while the control is declared for that provider** carries its
reason:

| Provider | Source | Status | Reason |
|---|---|---|---|
| outscale | terraform | ✗ `unsupported` | this source produces no resource of type "kubernetes_cluster" |

## What Pépin can conclude

| Status | What the status asserts | Reachable from |
|---|---|---|
| `fail` | a deviation was detected on a real resource | exoscale / terraform · exoscale / live · outscale / live |
| `pass` | the deciding data was collected, and it is compliant | exoscale / terraform · exoscale / live · outscale / live |
| `not-applicable` | the provider contract declares the control untestable, with its justification | — |
| `not-evaluated` | the control is implemented, but the data it depends on was not confirmed | — |

An observable control still returns `not-evaluated` on an inventory that contains no
resource of the targeted type: "nothing to look at" is not "compliant".

## How to investigate

- Normalized resource type the rule reads: `kubernetes_cluster`
- Attribute the decision depends on: `control_plane_multi_az`
- Without that attribute on a resource of the targeted type, the scan returns `not-evaluated` rather than `pass` (`internal/assess`, `requiredAttr` table).
- What each source projects is readable in the descriptor: [`providers/exoscale.yaml`](../../providers/exoscale.yaml) · [`providers/outscale.yaml`](../../providers/outscale.yaml)
- The rule that emits this code lives in [`internal/commonrules/rules/`](../../internal/commonrules/rules): it is **common** to every provider, only the source changes.

## How to remediate

Enable a multi-AZ / multi-master control plane on the managed cluster.

| Provider | Deployable setup |
|---|---|
| exoscale | [`references/remediation/exoscale/kubernetes_cluster_control_plane_highly_available`](../../references/remediation/exoscale/kubernetes_cluster_control_plane_highly_available) |
| outscale | _no proof filed yet_ |

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

In the `assessment` output, look for `"control": "kubernetes_cluster_control_plane_highly_available"`: its `status` must be `pass`.
If it stays `not-evaluated`, the deciding data was not collected and the fix is **not**
demonstrated: the reasons table above says why.

## See also

- [The assessment model](../concepts/assessment-model.md): what each status asserts.
- [Coverage matrix](../coverage.md): the same information, across every control.
- [Terraform plan or live scan](../concepts/terraform-vs-live.md): choosing the source.
- [Adding a control](../contributing/adding-a-control.md): the end-to-end procedure.

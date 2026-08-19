> 🇬🇧 English · [🇫🇷 Français](k8s_rbac_no_cluster_admin_binding.fr.md)

<!-- GENERATED PAGE — do not edit by hand. Regenerate with `mise run gen-docs`. -->

# `k8s_rbac_no_cluster_admin_binding`

**cluster-admin granted beyond the native binding**

[Back to the catalogue](index.md)

| Field | Value |
|---|---|
| Code | `k8s_rbac_no_cluster_admin_binding` |
| Family | `compute` |
| Severity | `critical` |
| SCSL requirement (frozen index) | `CLD-K8S-4` |
| Resource type read | `k8s_cluster_role_binding` |
| Deciding attribute | _none: judged on the presence of a deviation_ |
| State | active |
| Declared for | `kubernetes` |
| Remediation proofs | 0 / 1 |

## The risk

A ClusterRoleBinding grants the cluster-admin role to a subject other than the native system:masters group: that subject holds full power over the cluster, beyond least privilege.

This description comes from the reference: it is the text the report quotes, in the
reader's language.

## Normative mappings

Taken verbatim from the reference. The SCSL requirement comes from the **frozen** index:
a control maps onto an existing requirement, never onto one created for it. These mappings
are **indicative**: a Pépin report is not proof of qualification.

| Framework | References |
|---|---|
| `scsl` | `CLD-K8S-4` |
| `cis_controls_v8` | `6.8` |
| `iso_27001_2022` | `A.5.15`, `A.8.2` |
| `secnumcloud_3_2` | `9.1`, `9.3` |

## Where Pépin can measure it

A ✅ cell means the source produces the targeted type, the provider contract marks it
`verifie`, and the deciding attribute is projected. ◐ means "Pépin cannot decide from this
source", ∅ "not testable, with justification", ✗ "not declared, or type absent from this
source".

| Provider | Terraform plan | Live collection |
|---|:-:|:-:|
| exoscale | ✗ | ✗ |
| outscale | ✗ | ✗ |
| scaleway | ✗ | ✗ |
| kubernetes | n/a | ✅ |

Every cell that is not ✅ **while the control is declared for that provider** carries its
reason:

| Provider | Source | Status | Reason |
|---|---|---|---|
| kubernetes | terraform | ✗ `unsupported` | this source produces no resource of type "k8s_cluster_role_binding" |

## What Pépin can conclude

| Status | What the status asserts | Reachable from |
|---|---|---|
| `fail` | a deviation was detected on a real resource | kubernetes / live |
| `pass` | the deciding data was collected, and it is compliant | kubernetes / live |
| `not-applicable` | the provider contract declares the control untestable, with its justification | — |
| `not-evaluated` | the control is implemented, but the data it depends on was not confirmed | — |

An observable control still returns `not-evaluated` on an inventory that contains no
resource of the targeted type: "nothing to look at" is not "compliant".

## How to investigate

- Normalized resource type the rule reads: `k8s_cluster_role_binding`
- No attribute lock: the control is judged on the presence of a deviation, absence of a bad configuration counting as compliance.
- What each source projects is readable in the descriptor: [`providers/kubernetes.yaml`](../../providers/kubernetes.yaml)
- The rule that emits this code lives in [`internal/commonrules/rules/`](../../internal/commonrules/rules): it is **common** to every provider, only the source changes.

## How to remediate

Remove cluster-admin from this subject and grant it a Role/ClusterRole restricted to what it strictly needs.

| Provider | Deployable setup |
|---|---|
| kubernetes | _no proof filed yet_ |

A remediation proof is a self-contained, **compliant** Terraform module that deploys as
is, or a note anchored on the official documentation. See
[the remediation guide](../guides/remediation.md).

## How to verify the fix

```bash
# from the provider API: effective configuration
./pepin scan kubernetes --live --format assessment
```

In the `assessment` output, look for `"control": "k8s_rbac_no_cluster_admin_binding"`: its `status` must be `pass`.
If it stays `not-evaluated`, the deciding data was not collected and the fix is **not**
demonstrated: the reasons table above says why.

## See also

- [The assessment model](../concepts/assessment-model.md): what each status asserts.
- [Coverage matrix](../coverage.md): the same information, across every control.
- [Terraform plan or live scan](../concepts/terraform-vs-live.md): choosing the source.
- [Adding a control](../contributing/adding-a-control.md): the end-to-end procedure.

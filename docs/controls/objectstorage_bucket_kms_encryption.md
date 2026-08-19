> 🇬🇧 English · [🇫🇷 Français](objectstorage_bucket_kms_encryption.fr.md)

<!-- GENERATED PAGE — do not edit by hand. Regenerate with `mise run gen-docs`. -->

# `objectstorage_bucket_kms_encryption`

**No customer-managed encryption key (BYOK) on a sensitive bucket**

[Back to the catalogue](index.md)

| Field | Value |
|---|---|
| Code | `objectstorage_bucket_kms_encryption` |
| Family | `chiffrement` |
| Severity | `medium` |
| SCSL requirement (frozen index) | `CLD-CHF-4` |
| Resource type read | `object_storage_bucket` |
| Deciding attribute | `sse_kms_enabled` |
| State | active |
| Declared for | `scaleway` |
| Remediation proofs | 0 / 1 |

## The risk

An object storage bucket holding data classified as sensitive does not use a customer-managed encryption key (SSE-KMS / BYOK): its data stays under the provider's exclusive control. Observable only where the API exposes the bucket's default encryption (Scaleway, through Key Manager); not applicable at providers without a customer key at the object level.

This description comes from the reference: it is the text the report quotes, in the
reader's language.

## Normative mappings

Taken verbatim from the reference. The SCSL requirement comes from the **frozen** index:
a control maps onto an existing requirement, never onto one created for it. These mappings
are **indicative**: a Pépin report is not proof of qualification.

| Framework | References |
|---|---|
| `scsl` | `CLD-CHF-4` |
| `iso_27001_2022` | `A.8.24` |
| `secnumcloud_3_2` | `10.1`, `10.5` |

## Where Pépin can measure it

A ✅ cell means the source produces the targeted type, the provider contract marks it
`verifie`, and the deciding attribute is projected. ◐ means "Pépin cannot decide from this
source", ∅ "not testable, with justification", ✗ "not declared, or type absent from this
source".

| Provider | Terraform plan | Live collection |
|---|:-:|:-:|
| exoscale | ∅ | ∅ |
| outscale | ∅ | ∅ |
| scaleway | ◐ | ✅ |
| kubernetes | n/a | ✗ |

Every cell that is not ✅ **while the control is declared for that provider** carries its
reason:

| Provider | Source | Status | Reason |
|---|---|---|---|
| exoscale | terraform | ∅ `not-applicable` | SOS encrypts at rest by default (SSE-SOS, keys managed by Exoscale, SSE-S3 style) but exposes no customer-managed BYOK/KMS key at the bucket level (SSE-C stays per-object and unobservable), so the BYOK-at-bucket control is moot (CHF-4). |
| exoscale | live | ∅ `not-applicable` | SOS encrypts at rest by default (SSE-SOS, keys managed by Exoscale, SSE-S3 style) but exposes no customer-managed BYOK/KMS key at the bucket level (SSE-C stays per-object and unobservable), so the BYOK-at-bucket control is moot (CHF-4). |
| outscale | terraform | ∅ `not-applicable` | OOS encrypts server-side in AES256 with a PROVIDER key; there is neither a KMS service nor a customer-managed master key, so there is no BYOK to audit at the bucket level (CHF-4). Note: enabling SSE itself is opt-in and observable, which is a separate control, not this N/A. |
| outscale | live | ∅ `not-applicable` | OOS encrypts server-side in AES256 with a PROVIDER key; there is neither a KMS service nor a customer-managed master key, so there is no BYOK to audit at the bucket level (CHF-4). Note: enabling SSE itself is opt-in and observable, which is a separate control, not this N/A. |
| scaleway | terraform | ◐ `partial` | deciding attribute "sse_kms_enabled" not projected by this source: a capability guard, so the scan returns "not-evaluated" |

## What Pépin can conclude

| Status | What the status asserts | Reachable from |
|---|---|---|
| `fail` | a deviation was detected on a real resource | scaleway / terraform · scaleway / live |
| `pass` | the deciding data was collected, and it is compliant | scaleway / live |
| `not-applicable` | the provider contract declares the control untestable, with its justification | exoscale / terraform · exoscale / live · outscale / terraform · outscale / live |
| `not-evaluated` | the control is implemented, but the data it depends on was not confirmed | scaleway / terraform |

An observable control still returns `not-evaluated` on an inventory that contains no
resource of the targeted type: "nothing to look at" is not "compliant".

## How to investigate

- Normalized resource type the rule reads: `object_storage_bucket`
- Attribute the decision depends on: `sse_kms_enabled`
- Without that attribute on a resource of the targeted type, the scan returns `not-evaluated` rather than `pass` (`internal/assess`, `requiredAttr` table).
- What each source projects is readable in the descriptor: [`providers/scaleway.yaml`](../../providers/scaleway.yaml)
- The rule that emits this code lives in [`internal/commonrules/rules/`](../../internal/commonrules/rules): it is **common** to every provider, only the source changes.

## How to remediate

Create a key in the provider's key manager (Key Manager) and attach it to the bucket as its default encryption key (SSE-KMS).

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

# from the provider API: effective configuration
./pepin scan scaleway --live --format assessment
```

In the `assessment` output, look for `"control": "objectstorage_bucket_kms_encryption"`: its `status` must be `pass`.
If it stays `not-evaluated`, the deciding data was not collected and the fix is **not**
demonstrated: the reasons table above says why.

**One of the two sources cannot lift the `pass` lock** for this control: the provider
quoted does produce the targeted type there, but the scan will return `not-evaluated`.
The reasons table says which one, and why.

## See also

- [The assessment model](../concepts/assessment-model.md): what each status asserts.
- [Coverage matrix](../coverage.md): the same information, across every control.
- [Terraform plan or live scan](../concepts/terraform-vs-live.md): choosing the source.
- [Adding a control](../contributing/adding-a-control.md): the end-to-end procedure.

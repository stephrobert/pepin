> 🇬🇧 English · [🇫🇷 Français](remediation.fr.md)

# Remediating what Pépin finds

A scanner that only names problems moves the work, it does not reduce it. This page
describes how Pépin goes from *detecting* to *acting*: what every deviation already
carries, what a remediation must answer, how the fix is **verified** rather than
assumed, and exactly how far the deployable proofs go today.

## Every deviation already carries its remediation

A finding is never emitted without the sentence that says what to do about it. That is
not an editorial habit, it is a gate: `TestEveryFindingCarriesRemediation`
(`referentiel/validate_test.go`, run by `mise run validate`) rejects any rule whose
`remediation` is missing, and it demands the English counterparts
(`labels.message_en`, `labels.remediation_en`) at the same time. An English report must
never fall back to French mid-sentence.

The same applies to the reference: every control carries `remediation` and
`remediation_en` in `referentiel/controles.yaml`. That text is what the report prints,
and what each [control page](../controls/index.md) quotes.

So the **textual** remediation is a solved problem, and it is covered by a test. What
follows is about the layer above: the *deployable proof*.

## The four questions a remediation must answer

| Question | Where the answer lives |
|---|---|
| **Why it matters** | the control's description, from the reference |
| **How to investigate** | the normalized resource type read, and the attribute the decision depends on |
| **How to remediate** | the reference's remediation text, plus a deployable setup when one is filed |
| **How to verify** | the exact `pepin scan` command, and the status that must change |

Every control page answers the four, and none of it is hand-written: the pages are
generated from the reference, the provider descriptors and the `pass` lock. Start from
the [control catalogue](../controls/index.md).

## The loop, measured

A remediation that is not verified is a claim. Pépin closes the loop with the same
command that found the problem, on the same kind of input. The two extracts below are
**captured from real runs** of the repository's own example plans: the deliberately
misconfigured one, then the same module fixed.

Before, on `examples/scaleway/terraform/plan.json`:

<!-- pepin:gen remediation-before -->
```json
{
  "control": "objectstorage_bucket_public_access",
  "evidence": {
    "attribute": "acl",
    "observed": "Bucket \"scaleway_object_bucket_acl.backups\" is publicly accessible (public ACL).",
    "proves": [
      "",
      "",
      ""
    ],
    "source": "acl=terraform-plan:scaleway_object_bucket + terraform-plan:scaleway_object_bucket_acl observed=2/2"
  },
  "labels": {
    "category": "security",
    "provider": "scaleway"
  },
  "references": [
    {
      "framework": "scsl",
      "id": "CLD-STO-1"
    },
    {
      "framework": "cis-v8",
      "id": "3.3"
    },
    {
      "framework": "iso-27001",
      "id": "A.5.15"
    },
    {
      "framework": "iso-27001",
      "id": "A.8.3"
    },
    {
      "framework": "iso-27017",
      "id": "CLD.9.5.1"
    },
    {
      "framework": "secnumcloud-3.2",
      "id": "9.7"
    },
    {
      "framework": "secnumcloud-3.2",
      "id": "13.2"
    }
  ],
  "remediation": "Make the bucket private (private ACL, remove the AllUsers grant, delete the public policy); serve through pre-signed URLs if needed.",
  "severity": "critical",
  "status": "fail",
  "subject": "scaleway_object_bucket_acl.backups",
  "title": "Object storage publicly exposed"
}
```
<!-- /pepin:gen remediation-before -->

After, on `examples/scaleway/terraform-fixed/plan.json`:

<!-- pepin:gen remediation-after -->
```json
{
  "control": "objectstorage_bucket_public_access",
  "evidence": {
    "attribute": "acl",
    "observed": "no deviation detected on the collected resources of type \"object_storage_bucket\" (contract verified)",
    "proves": [
      "",
      "",
      ""
    ],
    "source": "acl=terraform-plan:scaleway_object_bucket + terraform-plan:scaleway_object_bucket_acl observed=2/2"
  },
  "references": [
    {
      "framework": "scsl",
      "id": "CLD-STO-1"
    },
    {
      "framework": "cis-v8",
      "id": "3.3"
    },
    {
      "framework": "iso-27001",
      "id": "A.5.15"
    },
    {
      "framework": "iso-27001",
      "id": "A.8.3"
    },
    {
      "framework": "iso-27017",
      "id": "CLD.9.5.1"
    },
    {
      "framework": "secnumcloud-3.2",
      "id": "9.7"
    },
    {
      "framework": "secnumcloud-3.2",
      "id": "13.2"
    }
  ],
  "severity": "critical",
  "status": "pass",
  "subject": "scaleway",
  "title": "Object storage publicly exposed"
}
```
<!-- /pepin:gen remediation-after -->

Three things changed, and all three matter:

- `status` went from `fail` to `pass`.
- `subject` went from the offending resource to the provider scope: there is no longer
  a resource to name.
- `evidence.observed` states **why** the pass is asserted, and names the lock it went
  through: the contract is verified, so the absence of a deviation is a measured
  compliance rather than an absence of measurement.

That last point is the whole difference between a fix and a blind spot. If the status
had gone from `fail` to `not-evaluated`, nothing would have been demonstrated: the
deviation would merely have stopped being visible. See
[the assessment model](../concepts/assessment-model.md) for what each status asserts,
and [exit codes](../reference/exit-codes.md) for what a pipeline reads from it.

## What a deployable proof is

A proof of remediation is a **self-contained Terraform module that is compliant**: it
deploys as is, in the provider's native vocabulary, and scanning it does not trigger the
rule. It is the mirror image of the non-compliant fixtures in
`examples/<provider>/terraform/`.

```
references/remediation/<provider>/<code>/      # self-contained Terraform module (preferred)
references/remediation/<provider>/<code>.md    # note anchored on the official docs, when Terraform is not relevant
```

Four rules, none of them cosmetic:

1. **One directory per rule.** Terraform merges every `.tf` of a directory, so several
   proofs in one directory could not be deployed or audited separately. A single setup
   satisfying neighbouring rules is duplicated, each copy scoped to its own rule.
2. **Deployable, verified as such.** `terraform init -backend=false` then
   `terraform validate` must pass: complete configuration, declared variables, real
   schema fields. A snippet that cannot be applied is pseudo-code, and pseudo-code
   presented as a configuration is exactly what this repository refuses.
3. **A mandatory header** naming the `code`, the SCSL requirement, the provider, **why
   the setup is compliant**, and the **anchored source** (a page under
   `references/docs/<provider>/`, or the contract in `providers/<provider>.yaml`).
4. **No secret.** Credentials come from the environment or from variables, never from
   the file.

The full convention lives in
[`references/remediation/README.md`](../../references/remediation/README.md).

## Coverage today, and it is partial

The count below is computed from the repository at generation time, over the (control,
provider) pairs the reference actually declares:

<!-- pepin:gen remediation-coverage -->
| Provider | Remediation proofs |
|---|---:|
| exoscale | 26 / 26 |
| kubernetes | 0 / 4 |
| outscale | 0 / 40 |
| scaleway | 0 / 25 |
| **Total** | **26 / 95** |
<!-- /pepin:gen remediation-coverage -->

Read that table for what it is: it counts **deployable proofs**, not remediations. The
textual remediation is at 100 % and gated by a test; the proofs are a work in progress,
and the honest thing is to say so rather than to round it up.

To reproduce the number, and to get the missing list per provider:

```bash
mise run check-remediation                     # every provider
python3 scripts/check-remediation.py exoscale  # one provider
```

## The gate, now that one provider is complete

`mise run check-remediation` stays deliberately **decoupled** from `mise run validate`.
Across all providers it is still red, and a gate that is red permanently is a gate
people learn to ignore: a quality gate that is routinely bypassed is worse than no
gate, because it teaches the habit of overriding one.

What has changed is that the rebranch condition written next to the task in
`mise.toml` — **one provider at 100 %** — is now met. Every control exoscale declares
carries its proof, and that ground is held by a test rather than by good intentions:
`TestExoscaleRemediationCoverageStaysComplete` (`internal/docgen`) fails when an
exoscale control lands without one, naming what is missing. It runs with
`mise run test`, so it sits in the release gate.

The other providers stay outside that gate while their coverage is partial, and each
joins it the day it reaches 100 %. Until then their count is published — above, and on
every control page — rather than enforced.

## Filing a proof

1. Anchor the setup first: the field, its type and its accepted values come from the
   provider's SDK or official documentation, never from memory. Cache the page under
   `references/docs/<provider>/` (`mise run fetch-docs`) and cite it.
2. Write `references/remediation/<provider>/<code>/main.tf` with the mandatory header.
3. Check it deploys: `terraform init -backend=false && terraform validate` in that
   directory.
4. Check it is **compliant**: generate a plan from it and scan it, the rule must not
   fire.

   ```bash
   terraform plan -out tfplan && terraform show -json tfplan > plan.json
   ./pepin scan <provider> --terraform plan.json --format assessment
   ```

5. Regenerate the derived documentation: `mise run gen-docs`. The control page now
   links your proof, and the coverage figures move.
6. `mise run validate` and `mise run test` stay green.

## See also

- [Control catalogue](../controls/index.md) — the four questions, answered per control.
- [The assessment model](../concepts/assessment-model.md) — what `pass` actually asserts.
- [Known limitations](../known-limitations.md) — the blind spots, named.
- [Adding a control](../contributing/adding-a-control.md) — the end-to-end procedure.

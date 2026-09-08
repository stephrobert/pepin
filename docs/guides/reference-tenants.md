> 🇬🇧 English · [🇫🇷 Français](reference-tenants.fr.md)

# Reference tenants

A fixture is written by the author of the rule. It therefore proves that the rule
**fires** — never that the rule is **right** about a configuration nobody designed for
it. That is a self-confirming test: it measures its author's intent, not the estate.

The precedent that settles the argument: replaying third-party Terraform stacks against
the binary found, in one sitting, a `CRITICAL` false positive on the most common
Scaleway configuration. No in-house fixture would have revealed it, because no in-house
fixture describes a **correct** configuration its author did not draw.

A reference tenant is the answer: a real, published, licensed configuration, pinned to a
commit, replayed on every build, and compared to the verdicts recorded beside it.

## What lives in the repository

```
references/tenants/<provider>/<name>/
  tenant.yaml     provenance (repo, commit, path, licence), posture, why it is not self-confirming
  plan.json       the Terraform plan, reduced to what Pépin reads
  expected.txt    the verdict per control, generated, reviewed
```

Nothing is provisioned. `terraform plan` creates no cloud resource, so there is nothing
to destroy — which is what `CONTRIBUTING.md` prefers over a live run.

## What a reference tenant proves, and what it does not

| It establishes | It does **not** establish |
|---|---|
| that a real third-party configuration produces the verdict recorded here | anything about a real tenant's live inventory |
| that a control does not fire on a configuration nobody wrote for it (false positive) | that the provider's API answers what the plan announces |
| that a control still fires on a real vulnerable configuration (false negative) | the rights a scan needs, or how a refusal is classified |
| that a `not-evaluated` is reached with the resource type actually present | the provider's pagination bounds or rate limiting |

A plan carries the **planned** state. What a provider **answers** stays owed to a live
collection, and [Known limitations](../known-limitations.md) says so in its place.

## The plan is reduced, and that is a guard

Pépin reads only two things from a plan (`internal/tfparse.ParsePlan`): `planned_values`
(or `values`), and the `source` of module calls under `configuration.root_module`.
Everything else — `variables`, `provider_config`, `prior_state`, `resource_changes` — is
ignored by the product, and it is exactly where a plan taken on a **real** tenant would
carry its credentials, its UUIDs and its addresses.

A reference tenant therefore carries only what Pépin reads, and every value Terraform
itself marks `sensitive` is nulled. `TestNoReferenceTenantPlanCarriesMoreThanPepinReads`
refuses anything else. The reduction is a security discipline before it is a size one —
though it does divide the corpus by seven.

## The two postures

`tenant.yaml` declares a posture, and a gate checks it against what the scan measures:

- **`exposed`** — the tenant raises at least one deviation. A tenant that stopped raising
  any is a candidate false negative.
- **`hardened`** — the tenant raises no `critical`/`high` deviation. This is the
  counter-witness, and the only place a false positive shows up.

A posture is never decreed. `TestEveryPostureIsTheOneMeasured` fails when the manifest
and the binary disagree.

## Regenerating

```bash
mise run tenants-refresh    # re-derive the plans from the pinned upstreams
mise run tenants-update     # re-derive the expected verdicts
mise run veracity-update    # re-derive the veracity debt ledger
```

`scripts/reference-tenant.sh --all` clones each upstream at its recorded commit, plans it
offline against fake public credentials, and reduces the result. **The six plans come back
byte for byte.** That is what lets a reviewer check that a tenant really comes from the
third-party repository it names, rather than from a file Pépin ended up writing to itself.

> **A verdict that flips on a pinned upstream is a decision, not a refresh.** Either the
> product improved, or it regressed on a configuration nobody wrote for it. Say which
> before regenerating.

## How a tenant pays the veracity contract

The [veracity contract](../../CONTRIBUTING.md) counts, per control × provider × source, the
verdicts proven end to end through the binary. A reference tenant proves some of them, and
they are counted **once**, in the same ledger as the hand-written scenarios — two coverage
figures would diverge, and the one that diverges is always the one people read.

Not every observed verdict counts. The **substantive filter** (`tenants.Substantive`):

- `fail`, `pass`, `not-applicable` always count — the chain concluded on real data;
- `not-evaluated` counts only when the tenant actually carries a resource of the targeted
  type. Otherwise the verdict says "this tenant has nothing of that kind", which is true,
  useful, and does **not** exercise the capability guard.

Without that filter, six tenants would pay ninety-seven obligations instead of fifty, and
half of them with absences. A counter brought down with empty cells is worse than a
counter that does not move: it relocates the false green into the dashboard.

## Related

- [Tracing real API calls](tracing-api-calls.md) — the other half: what the collector emits.
- [Known limitations](../known-limitations.md) — the veracity debt, counted and published.
- [Terraform vs live](../concepts/terraform-vs-live.md) — what each source can conclude.

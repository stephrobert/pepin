> 🇬🇧 English · [🇫🇷 Français](README.fr.md)

# Qualification tenants

A **qualification tenant** is a Terraform stack, written by us, **deliberately
misconfigured**, that is really **applied** on a cloud account, scanned, sealed,
**destroyed**, and compared with a committed expected result. It is stage 3 of the
release gate (issue #178).

It is the mirror image of a [reference tenant](../tenants/): a reference tenant is
a third-party configuration on which Pépin must stay **silent** (that is where a
false positive shows); a qualification tenant carries **known bad practices**, one
resource per control and one counterexample per control, and the scan must find
**all of them, and only them**, through the whole chain — real API → collector →
rules → assessment → bundle → verdict.

```
references/qualification/
  README.md · README.fr.md          this page
  <provider>/
    *.tf                            the stack: one fault per resource, one counterexample per control
    .terraform.lock.hcl             versioned: the provider build that was qualified
    tenant.yaml                     metadata: region, tenant tag, pinned provider version, budget, hourly rates
    expected.yaml                   the contract: control × source × subject → status, and the pinned account
    hooks.py                        provider hooks: identity, inventory by family, rescue cleanup
    README.md · README.fr.md        what the stack creates, what blocks destroy upstream, measured results
tools/qualification/qualify.py      the generic runner (apply, scan, seal, verify, destroy, prove, compare)
```

## What a run does

```bash
PEPIN_GATE_LIVE=1 mise run qualify            # PROVIDER=scaleway by default
mise run qualify:plan                         # same tenant, nothing created: plan + scan --terraform
mise run qualify:compare                      # re-compare the last run with expected.yaml
mise run qualify:selftest                     # the comparison proves itself on synthetic cases
```

1. **Preflight** — tools, the lockfile pins the provider version `tenant.yaml`
   declares, the binary is built from the tree being qualified.
2. **Identity** — the credentials are the provider's native ones (environment or
   its configuration file, never the repository, [ADR-0012](../../docs/adr/0012-aucun-identifiant-en-ci.md)).
   The runner asks the **API** which account they open, and **refuses to start** if
   the project and organisation are not the ones pinned in `expected.yaml`. A tenant
   that opens SSH to the world is never applied on a production account by mistake.
3. **Inventory before** — every family listed; a leftover from a previous run
   refuses the start.
4. **Two plans, one apply** — a live scan does not collect every type a plan
   carries (on Scaleway: five types live, eight on a plan). The **full** plan is
   the one scanned with `--terraform`; what is **applied** is the subset a live scan
   collects, because provisioning the rest would cost money no live scan measures.
   `expected.yaml` pins the two sources separately, and the tenant's variables say
   which resources are plan-only.
5. **`pepin scan --live`** in every format (`table`, `json`, `assessment`, `oscal`,
   `sarif`), sealed with `--seal`; `verify --re-derive` must pass, and a bundle
   altered by one byte must be refused ([ADR-0018](../../docs/adr/0018-ce-quun-bundle-prouve-de-lui-meme.md)).
6. **`pepin scan --terraform`** on the same plan.
7. **Destroy**, in a `finally`: it runs even when a stage before it failed. If
   `terraform destroy` does not finish, the provider hooks delete what carries the
   tenant's tag through the API, and destroy runs again.
8. **Proof of destruction** — `Destroy complete!` is not a proof. The account is
   listed again, family by family, filtered on the tenant's tag and name prefix,
   **and** compared with the inventory taken before apply: anything that appeared
   in between, tagged or not (a renamed root volume, an automatic backup), is a
   leftover and a NO-GO.
9. **Comparison** with `expected.yaml`, then **falsification**: an expectation is
   broken in memory and the comparison must say NO-GO. A gate that was never seen
   red guards nothing.

The output goes to `release-gate/` (ignored by git): `qualification-<provider>.json`
(verdict, evidence, durations, cost estimate) and the run directory with every
artifact. Those artifacts carry the resource identifiers of a real account: nothing
enters the repository from there without reading it.

## What NO-GO means

| Difference | What it is |
|---|---|
| an expected `fail` missing | a **false green**: the fault is there and the scan did not see it |
| a `fail` on a tenant subject nothing expects | a **false positive** |
| the counterexample fails | the rule fires but **does not discriminate** |
| a `not-evaluated` that became `pass` | a **pass nobody proved** ([ADR-0006](../../docs/adr/0006-jamais-un-pass-non-prouve.md)): the collector did not change, so the data is still missing |
| a `pass` or `fail` that became `not-evaluated` | a **coverage regression** |
| a control missing from `expected.yaml` | an unpinned verdict |
| an exit code other than the pinned one | the CI gate itself moved ([ADR-0005](../../docs/adr/0005-codes-de-sortie.md)) |
| a resource surviving destroy | the only unacceptable outcome of the whole exercise |

`expected.yaml` is **not a green matrix** ([ADR-0010](../../docs/adr/0010-dette-de-veracite-comptee.md)):
each `not-evaluated` in it is a **named collection debt**, and it changes when, and
only when, the collector learns to read the data. Such a change moves a verdict on an
unchanged tenant, so it gets its CHANGELOG line.

## Adding a provider

1. Copy the structure of `scaleway/`; keep the same file names, the runner relies on
   them (`tenant.yaml`, `expected.yaml`, `hooks.py`).
2. **Read the provider's Terraform issues about `destroy` first**, and write what you
   found in the provider's README with the links: a resource type known to deadlock a
   destroy is either avoided or its manual cleanup is automated in `hooks.py cleanup`.
3. One resource per control, one counterexample per control. Tag **every** taggable
   resource with the tenant tag, and prefix every name: that is what the proof of
   destruction filters on.
4. Pin the provider version exactly, and commit the lockfile.
5. Write `expected.yaml` from a `--plan-only` run first (Terraform source), then from
   a real run (live source), and **justify each status** from the rule and the
   collector: a status is pinned because it is right, not because it was measured.
6. Write the account pin, then run `PEPIN_GATE_LIVE=1 mise run qualify` and read the
   proof of destruction before reading anything else.

# Architecture decision records

> 🇬🇧 English · [🇫🇷 Français](README.md)

Code describes **the current state**. An ADR explains **why that state exists**,
and which decisions cannot be reopened without an explicit new decision.

The rule that governs this registry:

> **An issue describes what we want to change. An ADR describes decisions already
> taken. An issue cannot silently overturn an ADR.**

## Why this registry exists

This project has replayed the same debates more than once. A rejected solution
comes back under another name, a local fix contradicts a global decision, an aged
comment turns false, and two parts of the code drift into different rules without
anyone deciding so.

The risk is sharper with an agent: it produces, quickly, a solution that is
technically plausible **and already rejected**, because nothing tells it so.

## When to write one

A change to any of these surfaces calls for one:

data model · architecture · network · API · workflow · output format ·
persistence · security · generation · orchestration · **public behaviour**.

A typo, a missing test, a local rename: **no**.

## When not to

- To explain what a piece of code does — that is a comment's job.
- To record a style preference — that is `CONTRIBUTING.md`'s job.
- To state a live count of the repository. An ADR says "every generated module has
  a test", never "the 43 modules have a test". Live figures come from generated
  reports (`docs/detection-quality.md`, `docs/coverage.md`).

## The review contract, before any change

1. Identify the components touched.
2. Find the applicable ADRs — through the scope table, or each ADR's `scope:` field.
3. **Read them in full**, not just their titles.
4. List their invariants.
5. Check whether the change respects, extends, contradicts or reopens a decision.

Do not read the whole registry for every fix. A cross-cutting or architectural
change, on the other hand, warrants reading all of it.

## If the change contradicts an ADR

**Do not modify the code.** Say so first:

```
This change conflicts with ADR-XXXX: <summary of the decision>.

It therefore requires either honouring the ADR, or taking an explicit new
architectural decision.
```

If the decision genuinely must change, **do not rewrite the old ADR**. Write one
that supersedes it (`ADR-0023 supersedes ADR-0007`) and set the old one to
`Status: Superseded by ADR-0023`. The history stays visible: that is what stops
the third round of the same debate.

## If the proposed solution was already rejected

**Alternatives écartées** is not decoration. If an implementation matches an
explicitly rejected alternative, **stop** and say so:

```
ADR-0014 rejected X for reasons A, B and C.
The proposed solution reintroduces X.
Before implementing, establish that A, B and C have ceased to hold.
```

## Scope table

The scope table and the ADR list live in the French [`README.md`](README.md),
which is the maintained index. Each ADR also carries a `scope:` field so the
lookup can be automated.

The ADRs themselves are written in French, like the code comments they replace
(`CLAUDE.md` §1.2: code and comments in French, commits in English). This page and
its French counterpart carry the process, which contributors read first.

## Drift between ADRs and code

`mise run adr-drift` produces a report: ADRs citing a vanished file, `Accepted`
ADRs whose invariant no longer names a test, superseded ADRs left unmarked.

That check is **partial by construction**: it catches stale form, not contradicted
substance. An empty report does not prove an ADR still tells the truth.

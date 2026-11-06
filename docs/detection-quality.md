> 🇬🇧 English · [🇫🇷 Français](detection-quality.fr.md)

<!-- GENERATED page (internal/docgen). Do not edit by hand. -->

# Detection quality map

What Pépin can PROVE about its own verdicts, and what it cannot.
Every figure on this page is derived from the repository's artefacts —
obligations computed from the coverage matrix, veracity scenarios, reference
tenants, canary and qualification records. None is typed in.

## The rule

**No figure published here can be better than what is measured.** A percentage
with no measurement behind it is a false green moved into a dashboard, and it is
worse there than anywhere else: nobody re-reads a dashboard.

The figures are therefore ugly, and that is the point. "58 controls" says nothing
about the quality of a detection; "111 verdicts proven out of 478" says where the
product stands, and shrinks the right way with every scenario written.

## The figures

| Figure | Count |
|---|---:|
| Controls in the reference | 58 |
| Control x provider x source paths on which Pépin concludes | 190 |
| Paths whose EVERY reachable verdict is proven end to end | 38 |
| Verdicts to prove in total | 478 |
| Verdicts proven | 111 |

## Veracity coverage, by verdict

A path must prove the verdicts it can actually REACH, not four everywhere:
demanding a `not-applicable` from a path where the mechanism exists would mean
inventing a non-applicability.

| Verdict | What it stages | To prove | Proven | % |
|---|---|---:|---:|---:|
| `fail` | a vulnerable configuration is detected | 144 | 29 | 20 |
| `pass` | a genuinely correct configuration is confirmed | 144 | 41 | 28 |
| `not-evaluated` | the deciding attribute is missing, and the scan refuses to conclude | 166 | 29 | 17 |
| `not-applicable` | the provider's contract declares the mechanism non-existent | 24 | 12 | 50 |
| **Total** | | **478** | **111** | **23** |

## Validated live

A canary scan queries a provider's REAL control plane, but **with no credential**:
it proves an endpoint exists and refuses, never that a *sufficient* right returns
`200` on a real tenant. It therefore does not count as live validation of a
control.

A path counts when at least one required `pass` or `fail` verdict was explicitly
compared on an authenticated tenant for that control and provider. This does
not prove all verdicts on the path or change the veracity ledger. Silent
counterexamples, outside-tenant expectations and known defects do not prove a
`pass`.

Records are historical facts re-evaluated against the expectations whose hash
is recorded. They identify the binary actually run, not the release binary.
A bundle verifies integrity, not the identity of its author.
A tenant we wrote does not prove a rule universally correct; third-party
tenants complement it.

Historical snapshot as of 2026-09-11T07:34:27Z. Each record expires 90 days after its timestamp;
the binary excludes expired evidence at runtime and the preflight rejects a
provider whose recorded evidence is all stale.

| Figure | Count |
|---|---:|
| Paths whose source is a live collection | 104 |
| Paths with at least one validated live verdict | 40 |
| Distinct verdicts validated live | 40 |
| Validated live | **38 %** |

| Provider | Recorded | Expires at (UTC) | Qualified binary | Record |
|---|---|---|---|---|
| `exoscale` | 2026-09-11T07:34:27Z | 2026-12-10T07:34:27Z | `0.1.0-dev` (`vcs:2cf48ca9cd0967a16aa6b2f5396749503a6a04dd+modified`) | [JSON](../references/qualification/evidence/exoscale-2026-09-11.json) |
| `scaleway` | 2026-09-11T05:13:16Z | 2026-12-10T05:13:16Z | `0.1.0-dev` (`vcs:74ebd40a490ddf0905957982e814949a0eb20584+modified`) | [JSON](../references/qualification/evidence/scaleway-2026-09-11.json) |

## What the real control planes answered

One unauthenticated request per declared endpoint, at release qualification.
An endpoint that answers exists and resolves; a `moved` (404) says it has shifted.

| Provider | Recorded | Answered | Moved | Unreachable |
|---|---|---:|---:|---:|
| `exoscale` | 2026-08-21 | 9 | 0 | 0 |
| `outscale` | 2026-09-12 | 18 | 0 | 0 |
| `scaleway` | 2026-09-21 | 7 | 0 | 0 |

## Precision of the high/critical rules

Catching and STAYING SILENT are two different measurements, and they are published
together: "21 detection paths proven" reads in the most flattering way for as long
as nothing says on how many legitimate configurations those same rules held their
tongue. A rule that fires on everything is perfectly sensitive.

A COUNTEREXAMPLE is a pair on one control × provider × source path: a faulty case,
and a correct one that resembles it. Controls that do not have one yet are counted
in `internal/veracity/testdata/counterexamples-debt.txt`.

| Figure | Count |
|---|---:|
| Active high/critical controls | 43 |
| Of which a detection path is proven end to end | 22 |
| Of which a legitimate counterexample is proven | 20 |
| False positives measured on the counter-witnesses | 0 |

There is no "false negatives" row, and its absence is the most honest figure on
this page. No repository artefact measures them: it would take a corpus of faulty
configurations KNOWN to escape the rules, that is, knowing what one does not know.
Publishing "0" would be the exact false green this page fights — a measured zero is
not an existing zero.

## False positives

The repository keeps no false-positive register, and publishing a count would be
exactly the data entry this page refuses. What is MEASURED is the
counter-witness: a third-party tenant declared hardened on which Pépin raises no
`critical`/`high` deviation. It is the only place a false positive shows up, and
a gate checks it on every build.

| Figure | Count |
|---|---:|
| Hardened third-party tenants with no critical/high deviation (counter-witnesses) | 2 |
| Reference tenants in total | 6 |

## Measurements out of reach

They are documented rather than papered over: see [Known limitations](known-limitations.md)
and the debt ledger `internal/veracity/testdata/debt.txt`, which names every
verdict left to prove, line by line.

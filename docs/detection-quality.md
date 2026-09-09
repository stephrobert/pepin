> 🇬🇧 English · [🇫🇷 Français](detection-quality.fr.md)

<!-- GENERATED page (internal/docgen). Do not edit by hand. -->

# Detection quality map

What Pépin can PROVE about its own verdicts, and what it cannot.
Every figure on this page is derived from the repository's artefacts —
obligations computed from the coverage matrix, veracity scenarios, reference
tenants, canary records. None is typed in.

## The rule

**No figure published here can be better than what is measured.** A percentage
with no measurement behind it is a false green moved into a dashboard, and it is
worse there than anywhere else: nobody re-reads a dashboard.

The figures are therefore ugly, and that is the point. "58 controls" says nothing
about the quality of a detection; "89 verdicts proven out of 461" says where the
product stands, and shrinks the right way with every scenario written.

## The figures

| Figure | Count |
|---|---:|
| Controls in the reference | 58 |
| Control x provider x source paths on which Pépin concludes | 179 |
| Paths whose EVERY reachable verdict is proven end to end | 26 |
| Verdicts to prove in total | 461 |
| Verdicts proven | 89 |

## Veracity coverage, by verdict

A path must prove the verdicts it can actually REACH, not four everywhere:
demanding a `not-applicable` from a path where the mechanism exists would mean
inventing a non-applicability.

| Verdict | What it stages | To prove | Proven | % |
|---|---|---:|---:|---:|
| `fail` | a vulnerable configuration is detected | 141 | 23 | 16 |
| `pass` | a genuinely correct configuration is confirmed | 141 | 35 | 24 |
| `not-evaluated` | the deciding attribute is missing, and the scan refuses to conclude | 157 | 20 | 12 |
| `not-applicable` | the provider's contract declares the mechanism non-existent | 22 | 11 | 50 |
| **Total** | | **461** | **89** | **19** |

## Validated live

A canary scan queries a provider's REAL control plane, but **with no credential**:
it proves an endpoint exists and refuses, never that a *sufficient* right returns
`200` on a real tenant. It therefore does not count as live validation of a
control.

This counter only moves on an **authenticated** record, and none exists. The zero
is derived, not written: the day a maintainer records an authenticated run, it
will rise on its own.

| Figure | Count |
|---|---:|
| Paths whose source is a live collection | 101 |
| Validated live | **0 %** |

## What the real control planes answered

One unauthenticated request per declared endpoint, at release qualification.
An endpoint that answers exists and resolves; a `moved` (404) says it has shifted.

| Provider | Recorded | Answered | Moved | Unreachable |
|---|---|---:|---:|---:|
| `exoscale` | 2026-08-21 | 9 | 0 | 0 |
| `outscale` | 2026-08-21 | 17 | 0 | 0 |
| `scaleway` | 2026-08-21 | 5 | 0 | 0 |

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
| Of which a detection path is proven end to end | 20 |
| Of which a legitimate counterexample is proven | 18 |
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

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

The figures are therefore ugly, and that is the point. "57 controls" says nothing
about the quality of a detection; "63 verdicts proven out of 458" says where the
product stands, and shrinks the right way with every scenario written.

## The figures

| Figure | Count |
|---|---:|
| Controls in the reference | 57 |
| Control x provider x source paths on which Pépin concludes | 178 |
| Paths whose EVERY reachable verdict is proven end to end | 23 |
| Verdicts to prove in total | 458 |
| Verdicts proven | 63 |

## Veracity coverage, by verdict

A path must prove the verdicts it can actually REACH, not four everywhere:
demanding a `not-applicable` from a path where the mechanism exists would mean
inventing a non-applicability.

| Verdict | What it stages | To prove | Proven | % |
|---|---|---:|---:|---:|
| `fail` | a vulnerable configuration is detected | 140 | 10 | 7 |
| `pass` | a genuinely correct configuration is confirmed | 140 | 24 | 17 |
| `not-evaluated` | the deciding attribute is missing, and the scan refuses to conclude | 156 | 18 | 11 |
| `not-applicable` | the provider's contract declares the mechanism non-existent | 22 | 11 | 50 |
| **Total** | | **458** | **63** | **13** |

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
| Paths whose source is a live collection | 100 |
| Validated live | **0 %** |

## What the real control planes answered

One unauthenticated request per declared endpoint, at release qualification.
An endpoint that answers exists and resolves; a `moved` (404) says it has shifted.

| Provider | Recorded | Answered | Moved | Unreachable |
|---|---|---:|---:|---:|
| `exoscale` | 2026-08-21 | 9 | 0 | 0 |
| `outscale` | 2026-08-21 | 17 | 0 | 0 |
| `scaleway` | 2026-08-21 | 5 | 0 | 0 |

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

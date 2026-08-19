> 🇬🇧 English · [🇫🇷 Français](ROADMAP.fr.md)

# Roadmap

What Pépin measures today, where the effort goes next, and what it will not attempt.

This page carries **no dates and no commitments**. It states direction and order, so
that someone deciding whether to build on Pépin can tell what exists from what is
intended. Anything presented as a capability here is either already measurable, or
explicitly named as intent.

Two pages carry the facts, and both are generated from the reference and checked in
CI: [known limitations](docs/known-limitations.md) for the blind spots, and the
[coverage matrix](docs/coverage.md) for what is measurable control by control. This
page does not restate them.

## Where Pépin stands, v0.2.0

Four registered providers: three sovereign clouds, plus an in-cluster Kubernetes
collector.

<!-- pepin:gen provider-list -->
```text

// pepin  registered providers
  exoscale  Exoscale (CH) — instances, security groups, block storage, SKS, SOS
  kubernetes  Kubernetes (in-cluster) — RBAC, Pod Security Standards, NetworkPolicy
  outscale  Outscale (3DS) — VM, BSU, OOS, EIM, security groups, OKS, LBU
  scaleway  Scaleway — object storage, instances, IAM, security groups
```
<!-- /pepin:gen provider-list -->

The reference they are evaluated against:

<!-- pepin:gen control-counts -->
| Figure | Count |
|---|---:|
| Controls in the reference | 57 |
| Controls declared for at least one provider | 56 |
| `critical` | 10 |
| `high` | 32 |
| `medium` | 13 |
| `low` | 2 |
<!-- /pepin:gen control-counts -->

Around that: two analysable sources (a Terraform plan, which creates nothing, and a
read-only live API scan), five output formats, a stable exit-code contract, a sealed
evidence bundle with its OSCAL export, and a bilingual product from the report down to
the reference itself. The [control catalogue](docs/controls/index.md) gives one page
per control, and the [architecture](docs/project/architecture.md) explains how a
resource travels from a cloud API to a finding.

## What comes next, in order

### 1. Close the `◐` cells: collect what the rules already know how to read

The coverage matrix marks `◐` where a source produces the resource type but does not
project the **deciding attribute**. Pépin returns `not-evaluated` there, never a silent
`pass`, so nothing is wrong with the verdict — but a `◐` is a control a reader thought
they had. Closing one adds **no rule**: it adds a field to a provider descriptor. That
is why it comes first, and why it is the cheapest coverage in the project.

### 2. Deployable remediation proofs, one provider at a time

Every finding already carries a remediation **text**, and a reference test refuses a
control that lacks one. What is partial is the **deployable proof**: a self-contained,
compliant Terraform module under `references/remediation/`.

<!-- pepin:gen remediation-coverage -->
| Provider | Remediation proofs |
|---|---:|
| exoscale | 26 / 26 |
| kubernetes | 0 / 4 |
| outscale | 0 / 40 |
| scaleway | 0 / 25 |
| **Total** | **26 / 95** |
<!-- /pepin:gen remediation-coverage -->

`mise run check-remediation` is deliberately not wired into `mise run validate`: a gate
that is permanently red is a gate people learn to ignore. Exoscale is the first provider
at 100 %, and a test now holds that ground; the next providers join it one at a time.
See the [remediation guide](docs/guides/remediation.md).

### 3. The control domains that are thin today

In rough priority order, and each one gated on a **frozen** SCSL requirement, never on
an invented one:

- **Managed databases** beyond the single provider that has them today.
- **Organisation-level logging and audit trail** — the weakest family in the reference,
  and the one an auditor asks about first.
- **Services exposed by default** in the PaaS layers: serverless functions, container
  registries, image namespaces.
- **Cryptographic lifecycle**: key rotation policy, certificate expiry, bounded
  credential lifetimes beyond IAM.

When no frozen SCSL requirement covers a candidate control, it stays in the triage
catalogue (`referentiel/catalogue.yaml`) rather than becoming a control we invented the
justification for. That restraint is the point of the project, not an accident of it.

### 4. Providers

**OVHcloud** is the next sovereign cloud on the list. Adding one is one descriptor and
**zero rules** — the rules are common, only the source changes — which is what
[adding a provider](docs/contributing/adding-a-provider.md) walks through.

Nothing is declared covered before its contract is verified field by field against the
provider's own SDK or API specification. A provider that ships with unverified fields
would put a green cell in the matrix that nobody can defend in an audit.

### 5. Reading the reference outside the repository

The reference, the coverage matrix and the control catalogue are already generated from
the binary. The intent is to publish them as a bilingual site fed by that same
generator, so that the published pages cannot drift from the tool. This is direction,
not a shipped feature.

## What Pépin will not do

The [scope and non-goals](docs/concepts/scope.md) page is the authority. In short:
Pépin reads the **tenant** side of a cloud, at a point in time, in read-only. It is not
a runtime agent, it keeps no history between runs, it changes nothing on your account,
and its normative mappings are indicative correspondences — a Pépin report is not a
proof of qualification, which bears on the cloud provider and not on a tenant scan.

## The limits that shape this order

Three, from [known limitations](docs/known-limitations.md), because they explain why
the list above is ordered the way it is:

- **The API contract is recorded per resource type, not per (type × source).** This is
  what produces `◐` cells, and item 1 above is the answer to it.
- **The `live` column of the coverage matrix is derived from the descriptors, not
  observed.** It states what a descriptor projects, not what an API returned during a
  measured run.
- **Nothing is measured between two runs.** A result describes an instant; continuous
  posture is the job of whatever schedules Pépin.

## How this page stays honest

The figures above are **generated** by `mise run gen-docs` from the reference, the
provider descriptors and the binary itself; `TestGeneratedDocsAreUpToDate` fails the
build when they drift. The prose is written by hand and reviewed at each release.

The detailed investigation log — per-provider audit verdicts, engine bugs, field-level
findings — is a maintainer's working document in French, kept out of the product
documentation at `notes/roadmap-interne.fr.md`. Anything in it that concerns a user
belongs in [known limitations](docs/known-limitations.md) instead, and is moved there
rather than summarised here.

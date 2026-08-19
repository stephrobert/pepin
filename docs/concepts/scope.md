> 🇬🇧 English · [🇫🇷 Français](scope.fr.md)

# Scope and non-goals

Pépin prints this on **every** scan, on `stderr`, before you get a chance to misread the
report:

<!-- pepin:gen scope-disclaimer -->
```text
This report assesses the configuration of a tenant (customer-side scope). The
normative mappings (SecNumCloud, ISO, CIS) are indicative: they are not a proof of
qualification or certification, which applies to the cloud service provider.
```
<!-- /pepin:gen scope-disclaimer -->

This page is that sentence, developed. It is not a softer version of it: where the two could be
read differently, the constant `assess.ScopeDisclaimer` wins, and this page is wrong.

## What Pépin evaluates: a tenant's posture

A cloud has two sides, and they are audited by two different things.

| | Cloud **service provider** | Cloud **customer** (tenant) |
|---|---|---|
| Owns | datacentres, hypervisors, control plane, operating procedures, staff vetting | accounts, IAM, networks, instances, buckets, databases, their configuration |
| Audited by | a qualification or certification body (ANSSI for SecNumCloud, an accredited auditor for ISO 27001) | tools like Pépin, plus your own procedures |
| Produces | a qualification, a certificate, an attestation | evidence about a configuration at a point in time |

**Pépin only ever looks at the right-hand column.** It reads your account through the
provider's API, or your infrastructure code through a Terraform plan. It has no visibility into
how the provider runs its platform, and it makes no claim about it.

## Observable configuration, and nothing else

Pépin measures **configuration facts that an API or a plan exposes**. That boundary has sharp
consequences, and they are documented rather than glossed over:

- A property the API does not expose cannot be measured. Where a provider contract records this
  (`etat: absent`, or an entry in `contrat.non_applicable`), the control returns
  `not-applicable` **with the recorded justification**.
- A property that exists but was not collected on this run returns `not-evaluated`, with the
  reason. It never returns `pass`.
- Encryption performed **inside the guest** (LUKS on a block volume, application-level
  encryption) is invisible to the platform API by construction. Pépin says so; it does not
  guess.

The whole taxonomy is in [the assessment model](assessment-model.md), and the per-provider
consequences are in [known limitations](../known-limitations.md).

## Two sources, two different claims

| Source | Command | Claims |
|---|---|---|
| Terraform plan | `--terraform plan.json` | what your code **declares** it will create. Nothing about drift, nothing about what already runs, nothing about attributes still `known after apply`. The verdict says so: *"declared scope (Terraform plan, planned state)"*. |
| Live API | `--live` | the **effective** configuration of the tenant at the moment of the scan, as far as the credentials could read it. Insufficient rights show up as `not-evaluated`, never as compliance. |

Neither source claims anything about the interval between two scans.

## Mapping is not certification

Every control in `referentiel/controles.yaml` carries mappings: an SCSL requirement (`CLD-*`,
from the frozen index), and correspondences to SecNumCloud 3.2, CIS Controls v8 and
ISO/IEC 27001:2022 / 27017. Those mappings exist so a result can be *filed* against a
framework an auditor already knows.

They do not, and cannot, mean:

- that passing the mapped controls satisfies the framework requirement — a requirement usually
  covers organisation, procedure and evidence, of which configuration is one part;
- that your tenant is SecNumCloud-qualified — **qualification bears on the provider**, is
  granted by ANSSI, and no scanner grants it;
- that your provider is qualified — the sovereignty facts in `providers/<name>.yaml` are
  **declared** from public sources and cited as such, not verified by Pépin.

That last point is visible in the output itself: the `governance_provider_sovereignty` control
carries the evidence *"compliant according to the sovereignty facts declared in the provider
descriptor (attestation, not measured on the tenant)"*. It says "attestation" because that is
what it is.

## What "opposable" means here

"Opposable" in Pépin's vocabulary is a property of the **result document**, not a legal status.
A result is opposable when a third party can contest it on facts rather than on trust. Which
means, concretely:

- **typed statuses** — `pass` / `fail` / `not-applicable` / `not-evaluated`, so "no finding" is
  never presented as "compliant";
- **a justification on every non-measurement** — an unjustified `not-applicable` is not
  produced at all;
- **exact normative references** on every result, not a vague framework name;
- **provenance** — the digest of the binary, and a digest covering the rules, the provider
  descriptors and the reference, so two different configurations cannot produce the same
  fingerprint under the same result;
- **a sealable bundle** — `--seal` writes the evaluated inventory, the assessment, its OSCAL
  1.1.2 rendering and a digested manifest, which `pepin verify` re-checks and can re-derive.

It does **not** mean the result is admissible anywhere, nor that it substitutes for an audit.

## Non-goals

Pépin does not:

- **grant, prove or predict a qualification or certification** of any kind;
- **audit the cloud provider** — its platform, its procedures, its staff, its subcontractors;
- **audit the inside of a Kubernetes cluster as a cloud control plane** — the `kubernetes`
  provider exists for in-cluster state (RBAC, Pod Security, NetworkPolicy) and is deliberately
  kept out of any parity comparison with clouds, since neither scope can cover the other;
- **scan workloads** — no vulnerability scanning of images, no runtime agent, no SAST/DAST;
- **read application data** — it reads configuration metadata, and the evaluated inventory is
  what you can inspect in a sealed bundle;
- **remediate** — it never writes to a cloud API. A live scan needs read-only credentials, and
  the tool cannot use anything more even if you give it more;
- **replace an audit, a risk analysis or a security policy**;
- **measure anything between two runs** — a result describes an instant.

## Terminology, kept consistent

| Term | In Pépin |
|---|---|
| tenant | the customer-side account/organisation being scanned. Never the provider. |
| control | an agnostic entry of `referentiel/controles.yaml`, mapped to a frozen SCSL requirement |
| check | the agnostic code a Rego rule emits, kept in `labels.check` |
| source | `terraform-plan`, `live-api`, or `export` |
| finding | one deviation on one subject |
| assessment | the typed, referenced, provenanced document covering every control |
| evidence bundle | the sealed directory produced by `--seal` |

## The reference, by the numbers

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

The full breakdown, per provider and per source, is in the
[coverage matrix](../coverage.md).

## See also

- [The assessment model](assessment-model.md) — what each status asserts.
- [Known limitations](../known-limitations.md) — the blind spots, named.
- [Coverage matrix](../coverage.md) — what is measurable today.

> 🇬🇧 English · [🇫🇷 Français](README.fr.md)

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/assets/brand/pepin-lockup-dark.svg">
  <img src="docs/assets/brand/pepin-lockup-light.svg" alt="Pépin" width="200">
</picture>

# Pépin

[![CI](https://github.com/stephrobert/pepin/actions/workflows/ci.yml/badge.svg)](https://github.com/stephrobert/pepin/actions/workflows/ci.yml)
[![CodeQL](https://github.com/stephrobert/pepin/actions/workflows/codeql.yml/badge.svg)](https://github.com/stephrobert/pepin/actions/workflows/codeql.yml)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/stephrobert/pepin/badge)](https://securityscorecards.dev/viewer/?uri=github.com/stephrobert/pepin)
[![License: Apache-2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white)](go.mod)
[![OSCAL 1.1.2](https://img.shields.io/badge/OSCAL-1.1.2-1f6feb.svg)](https://pages.nist.gov/OSCAL/)
[![SecNumCloud 3.2](https://img.shields.io/badge/SecNumCloud-3.2-0b3d91.svg)](https://cyber.gouv.fr/secnumcloud-pour-les-fournisseurs-de-services-cloud)

**Pépin finds the pips in your sovereign cloud.**

![Pépin scanning a deliberately misconfigured Terraform plan, then the same module fixed: the verdict and the exit code change](docs/assets/quickstart.gif)

*Every command in that recording really runs. It is regenerated at each release
from `tools/demo/quickstart.en.tape`, and the preflight refuses to tag a version
the GIF does not show. Its French twin is on [README.fr.md](README.fr.md).*

Pépin is a cloud posture scanner (CSPM) for **European sovereign clouds**
(Exoscale, Outscale, Scaleway). It evaluates a tenant's effective configuration
against a common reference anchored on **SCSL**, **SecNumCloud 3.2**, **CIS
Controls v8** and **ISO/IEC 27001:2022 / 27017**, and produces an **opposable**
result: a typed status per control, its exact normative references, and a sealed
evidence bundle.

## What sets Pépin apart

- **Sovereign first**: each provider is described in its native vocabulary
  (Net/Subnet/EIM, BSU/OOS, Kapsule…), never by comparison to a non-sovereign
  hyperscaler.
- **Bilingual, language detected**: reports, help and errors come out in French
  or in English, from the whole tool and not half of it. See
  [Language](#language).
- **An opposable result, not just findings**: every control is
  `pass` / `fail` / `not-applicable` (justified) / `not-evaluated` (with the
  reason). A `pass` is asserted only when the data it needs was actually
  collected — "no finding" is never mistaken for "compliant".
- **Two sources**: **live** collection via the provider API, or auditing a
  **Terraform plan** (`terraform show -json`) — provisioning nothing.
- **Evidence bundle**: `--seal` writes a timestamped bundle (evaluated inventory,
  assessment, **OSCAL 1.1.2**, digested manifest) that `pepin verify` re-checks,
  with optional **cosign signature** verification.

## Getting started

Released binaries (checksummed, signed, with SLSA provenance), a container
image, a GitHub action and a GitLab template are documented in
[docs/install.md](docs/install.md). From source:

```bash
# build (Go 1.26+)
go build -o pepin .

# audit a Terraform plan (no resource provisioned)
terraform plan -out tfplan && terraform show -json tfplan > plan.json
./pepin scan scaleway --terraform plan.json

# live collection (credentials via the environment / native provider config)
./pepin scan outscale --live --region eu-west-2

# sealed evidence bundle + verification
./pepin scan scaleway --terraform plan.json --seal ./bundle
./pepin verify ./bundle --pubkey cosign.pub
```

Output formats: `--format table|json|assessment|oscal|sarif`.
Exit codes: `0` compliant · `1` non-compliance · `2` error · `3` the scan does not
establish compliance — nothing measured, or the collection could not read the whole
scope, or, with `--strict`, remaining medium/low gaps · `4` every critical/high
deviation is covered by a valid exemption (CI-friendly). A scan that collected no
resource never returns `0`, and neither does one that could not read part of its
scope: an empty result is not a compliant one, a partial one is not either, and an
exempted one is not either.

## Language

Pépin speaks French and English, and picks one on its own:

```
--lang=fr|en  →  PEPIN_LANG  →  LC_ALL  →  LANG  →  fallback: en
```

The first non-empty source decides, and any locale Pépin does not speak falls
back to English without an error. The choice applies to everything a human
reads — report, verdict, help, errors — **and** to the parsable formats: `json`,
`sarif`, `oscal` and `assessment` carry their prose in the resolved language.

**What is stable across languages, and what is not.** Codes (`CLD-NET-1`), check
identifiers, severities (`critical`…), statuses (`pass`, `fail`,
`not-applicable`, `not-evaluated`), subjects and exit codes never change with
the language: a pipeline keys on those. Titles, messages, remediations and
evidence are prose and do change. Pin `PEPIN_LANG` in CI if you diff report
text between runs.

**Which one is authoritative.** French is the reference language of the
normative content: the reference (`referentiel/controles.yaml`) and the rules
are written in French first, and the English is their maintained translation.
Where a legal or contractual reading is at stake, the French wording of a
control is the one that governs.

## Documentation

- [Quickstart](docs/getting-started/quickstart.md) — five minutes, no cloud account:
  a real failure, its fix, and a second scan that says something different.
- [Understanding a scan](docs/getting-started/understanding-a-scan.md) — one real run,
  read line by line, down to the exit code.
- [The assessment model](docs/concepts/assessment-model.md) — what `pass`, `fail`,
  `not-applicable`, `not-evaluated` and `exempted` actually assert.
- [Coverage matrix](docs/coverage.md) — what is measurable, per provider and per source.
  **Generated** from the reference and the provider descriptors, and verified in CI.
- [Control catalogue](docs/controls/index.md) — one generated page per control: what it
  concludes, from which source, and how to fix what it finds.
- [Known limitations](docs/known-limitations.md) — the blind spots, named.
- [Scope and non-goals](docs/concepts/scope.md) — what a Pépin report is not.
- [Terraform plan vs live scan](docs/concepts/terraform-vs-live.md) — choosing the
  source, and the real divergences between the two.
- Providers: [Scaleway](docs/providers/scaleway.md) · [Outscale](docs/providers/outscale.md) ·
  [Exoscale](docs/providers/exoscale.md) — authentication, API calls, minimal
  read-only permissions, coverage.
- Reference: [CLI](docs/reference/cli.md) · [Exit codes](docs/reference/exit-codes.md) ·
  [Output formats](docs/reference/output-formats.md) ·
  [Normalized inventory](docs/reference/inventory.md).
- Guides: [Remediation](docs/guides/remediation.md) ·
  [Configuring controls](docs/guides/control-configuration.md) — the policy file, and what a
  relaxation costs ·
  [Evidence bundles](docs/guides/evidence-bundles.md) ·
  [GitHub Actions](docs/guides/github-actions.md) · [GitLab CI](docs/guides/gitlab-ci.md).
- Contributing: [Adding a control](docs/contributing/adding-a-control.md) ·
  [Adding a provider](docs/contributing/adding-a-provider.md) ·
  [Architecture](docs/project/architecture.md).
- [Install](docs/install.md) · [Roadmap](ROADMAP.md) — where the effort goes next.

## Architecture (in brief)

- `internal/collect`: declarative collection engine (YAML specs → normalized model).
- `providers/*.yaml`: a provider descriptor (auth, live collection, Terraform
  mapping, API contract). One provider = three sources in a single YAML file.
- `internal/commonrules/rules/*.rego`: **common** rules (OPA/Rego) that evaluate
  the normalized model, provider-independently.
- `referentiel/`: the control reference (neutral code → severity, SCSL, norm
  mappings, providers). Source of truth, tested against invented references.
- `internal/assess`: builds the opposable assessment (statuses, evidence,
  provenance) and the sealed bundle (OSCAL, digests, cosign).

See [CONTRIBUTING.md](CONTRIBUTING.md) to add a provider or a rule, and
[SECURITY.md](SECURITY.md) for vulnerability disclosure.

## Scope

Pépin evaluates the configuration of a **tenant** (the cloud customer's side). The
normative mappings it reports (SecNumCloud, ISO, CIS) are **indicative
correspondences**: a Pépin report is **not** a proof of qualification or
certification — those bear on the cloud **service provider**, not on a tenant scan.

## License

Apache-2.0: see [LICENSE](LICENSE) and [NOTICE](NOTICE).

The mark and its files are documented in [docs/brand.md](docs/brand.md);
**the name *Pépin* and the logo are not covered by the Apache 2.0
licence**.

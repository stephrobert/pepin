> 🇬🇧 English · [🇫🇷 Français](github-actions.fr.md)

# GitHub Actions integration

Three postures a pipeline can take, and this guide covers all three: **gate** a pull request on
the Terraform plan, **report** the findings to Code Scanning without blocking, and **watch** the
live tenant on a schedule with a sealed evidence bundle.

The complete workflow is committed in `examples/github-actions/pepin.yml` and shown in full at
the end of this page. It is checked by `actionlint`.

## Install: pin the action, and pin at least v0.2.0

```yaml
- uses: stephrobert/pepin/.github/actions/pepin-scan@a02e42d054dc9d8a5a41ece5b46f6c1111659a70 # v0.2.0
  with:
    version: '0.2.0'
```

Two pins, and they are not redundant. The **`@<sha>`** decides which action code runs; the
**`version:` input** decides which released binary it installs. A mutable reference (`@main`,
`@v1`) runs whatever that reference says tomorrow, which is the supply-chain shape this
repository spends a release workflow proving it does not have.

> **v0.2.0 is the minimum.** In v0.1.0 and v0.1.1 the action's installer called
> `gh attestation verify` without a token. The GitHub API refuses that call in a workflow, the
> installer treated it as an unverified provenance, and **every** installation was refused. The
> action now supplies `github.token` itself. Pinning an earlier tag installs an action that
> cannot install anything.

### What the action verifies before it runs a binary

Not just a digest. In order:

1. **Provenance.** `gh attestation verify --repo stephrobert/pepin --signer-workflow
   stephrobert/pepin/.github/workflows/release.yml` — the binary must carry a build attestation
   produced by *that* workflow in *that* repository. Comparing a download against a checksum
   list published beside it proves only that two files from the same origin agree; whoever can
   replace one can replace both. The attestation is what settles authorship.
2. **Integrity.** The SHA-256 of the downloaded asset is then checked against the release's
   `checksums.txt`.
3. Only then is the binary made executable and put on the `PATH`.

If `gh` is absent from the runner (it ships with GitHub-hosted runners), the installer warns
loudly that provenance was not verified and falls back to the digest alone. A CI job in this
repository corrupts one byte of the download and requires the verification to refuse it.

## Gate a pull request on the Terraform plan

No credential, nothing provisioned, nothing billed: `terraform plan` is rendered as JSON and
audited. This is the job to make required in a branch ruleset.

```yaml
permissions: {}

jobs:
  terraform-plan:
    runs-on: ubuntu-24.04
    permissions:
      contents: read
    steps:
      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
        with:
          persist-credentials: false
      - uses: hashicorp/setup-terraform@dfe3c3f87815947d99a8997f908cb6525fc44e9e # v4.0.1
        with:
          terraform_wrapper: false
      - run: |
          terraform init
          terraform plan -out=tfplan
          terraform show -json tfplan > plan.json
      - uses: stephrobert/pepin/.github/actions/pepin-scan@a02e42d054dc9d8a5a41ece5b46f6c1111659a70 # v0.2.0
        with:
          version: '0.2.0'
          provider: scaleway
          terraform-plan: plan.json
```

`permissions: {}` at the workflow level, then the minimum each job needs: this job reads the
repository and nothing else. `persist-credentials: false` keeps the checkout token out of the
git config, where a later step could reuse it.

## The exit codes, treated explicitly

This is where a posture gate is won or lost. The action maps the contract like this:

| Exit | Meaning | `fail-on-nonconformity: 'true'` (default) | `'false'` |
|:-:|---|---|---|
| **0** | compliant | job passes | job passes |
| **1** | critical/high deviation | **job fails**, `::error::` | `::warning::`, job passes |
| **3** | nothing measured, or medium/low under `--strict` | **job fails**, `::error::` | `::warning::`, job passes |
| **2** | technical error | **job fails**, `::error::` | **job fails** — never downgraded |

**2 is never downgraded, by design.** "The tenant is not compliant" is a verdict; "the scan
could not conclude" is a failure of the measurement, and a pipeline that swallows it reports a
posture nobody measured. The same rule applies if you run the binary directly instead of using
the action:

```yaml
- name: Gate on the posture, keeping 2 distinguishable
  run: |
    pepin scan scaleway --terraform plan.json
    code=$?
    case "$code" in
      0) echo "::notice::compliant" ;;
      1|3) echo "::error::non-compliant (exit $code)" ; exit 1 ;;
      2) echo "::error::pepin could not conclude (technical error)" ; exit 2 ;;
      *) echo "::error::unexpected code $code" ; exit 2 ;;
    esac
```

Never `continue-on-error: true` on the scan step, and never `|| true` in the script: both erase
the distinction the table above exists for. To report without gating, use
`fail-on-nonconformity: 'false'`, which keeps 2 fatal.

## Publish the findings to Code Scanning

SARIF is the format the Code Scanning tab reads. The upload needs exactly one extra permission.

```yaml
    permissions:
      contents: read
      security-events: write   # required by upload-sarif, and by nothing else

    steps:
      - uses: stephrobert/pepin/.github/actions/pepin-scan@a02e42d054dc9d8a5a41ece5b46f6c1111659a70 # v0.2.0
        with:
          version: '0.2.0'
          provider: scaleway
          terraform-plan: plan.json
          format: sarif
          output-file: pepin.sarif
          fail-on-nonconformity: 'false'
      - uses: github/codeql-action/upload-sarif@5595ccaf912efad79be6eef63a5619ff05969be3 # v4.37.6
        with:
          sarif_file: pepin.sarif
          category: pepin
```

`category: pepin` keeps these alerts in their own namespace, so another tool uploading SARIF in
the same repository does not close Pépin's alerts and vice versa. Alerts land on the **scanned
file** — the plan — not on a line of a `.tf` file: a normalized resource does not carry the line
it came from ([Output formats](../reference/output-formats.md#sarif--for-github-code-scanning)).

Code Scanning is a *report* surface. Pair it with `fail-on-nonconformity: 'false'` and keep the
blocking decision in the gating job, or you will have the same finding failing the build twice.

## Scan the live tenant

A plan says what the code declares. Only a live scan sees what is actually running, including
what nobody wrote in Terraform ([Terraform plan vs live scan](../concepts/terraform-vs-live.md)).
Run it on a schedule, not on every pull request.

```yaml
    env:
      SCW_ACCESS_KEY: ${{ secrets.SCW_ACCESS_KEY }}
      SCW_SECRET_KEY: ${{ secrets.SCW_SECRET_KEY }}
      SCW_DEFAULT_ORGANIZATION_ID: ${{ secrets.SCW_DEFAULT_ORGANIZATION_ID }}
      SCW_DEFAULT_REGION: fr-par
```

**Credentials are never action inputs.** Pépin reads each provider's native environment
variables, and routing them through inputs would only add a place for them to leak:

| Provider | Variables |
|---|---|
| Scaleway | `SCW_ACCESS_KEY`, `SCW_SECRET_KEY`, `SCW_DEFAULT_ORGANIZATION_ID`, `SCW_DEFAULT_REGION` |
| Outscale | `OSC_ACCESS_KEY`, `OSC_SECRET_KEY`, `OSC_REGION` |
| Exoscale | `EXOSCALE_API_KEY`, `EXOSCALE_API_SECRET`, `EXOSCALE_ZONE` |

Use a **read-only** key, scoped to the smallest permission set that covers what Pépin calls —
each provider page lists those calls and the permissions they need:
[Scaleway](../providers/scaleway.md#minimal-permissions-for-a-live-scan),
[Outscale](../providers/outscale.md#minimal-permissions-for-a-live-scan),
[Exoscale](../providers/exoscale.md#minimal-permissions-for-a-live-scan).

A key with too few rights does not produce a false green: what could not be collected comes back
as `not-evaluated`, and a scan that measured nothing exits **3**
([Exit codes](../reference/exit-codes.md)).

## Archive the evidence bundle

```yaml
      - uses: actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a # v7.0.1
        with:
          name: pepin-evidence
          path: |
            report.json
            bundle/
          retention-days: 7
```

**A bundle names every resource of the tenant.** The action passes `--redact` unless you set
`redact: 'false'`, so user data and policy documents are replaced by their digest — but the
inventory itself, resource by resource, is still in there. Keep the retention short, and
remember that **artifacts of a public repository are downloadable by anyone with read access**.
Drop `--redact` only when the bundle must support `verify --re-derive`, and then treat the
artifact as a secret ([Evidence bundles](evidence-bundles.md)).

## Pin the language

```yaml
env:
  PEPIN_LANG: en
```

Exit codes, control codes, severities and statuses are identical in both languages, so a gate
needs nothing. But a job that diffs report *text* between runs, or that archives bundles and
compares their digests, must pin `PEPIN_LANG`: a runner whose locale changes would otherwise
produce a difference nobody caused.

## The complete workflow

Copy this into `.github/workflows/`. Everything is pinned by commit SHA, and the three jobs are
the three postures this page opened on.

<!-- pepin:gen example-github-workflow -->
```yaml
# Gate a pipeline on the posture of a sovereign cloud — Pépin in GitHub Actions.
#
# Copy this into `.github/workflows/` of a repository holding a Terraform
# configuration for Exoscale, Outscale or Scaleway. The first two jobs need no
# credential at all: they audit the *plan*, so nothing is provisioned and there
# is no secret to create. The third shows the live variant, disabled by
# default, because it reads a real tenant with real credentials.
#
# The exit codes are the contract: 0 compliant, 1 non-compliance, 2 technical
# error, 3 nothing measured (or medium/low under --strict). The action fails
# the job on 1 and 3 unless you set `fail-on-nonconformity: 'false'`, and it
# fails on 2 whatever you set: a swallowed technical error would report a
# posture nobody measured.
#
# Everything is pinned by commit SHA, this repository's own action included.
# A mutable reference (@main, @v1) runs whatever that reference says tomorrow.
#
# Pin at least v0.2.0 of the action: in v0.1.0 and v0.1.1 its installer called
# `gh attestation verify` without a token, which refused EVERY installation.

name: cloud posture

on:
  pull_request:
  schedule:
    - cron: '17 4 * * 1'   # the live job below, once a week
  workflow_dispatch:

permissions: {}

env:
  # Codes, identifiers, severities and exit codes are identical in both
  # languages; titles, messages and remediations are not. A pipeline that
  # compares report TEXT between runs pins the language. One that reads exit
  # codes does not have to, and pinning costs nothing.
  PEPIN_LANG: en

jobs:
  # ---------------------------------------------------------------- blocking
  # No account, no secret, nothing billed: the plan is audited, not the cloud.
  # This job GATES: a critical/high deviation fails the pull request.
  terraform-plan:
    name: audit the Terraform plan (blocking)
    if: github.event_name == 'pull_request'
    runs-on: ubuntu-24.04
    timeout-minutes: 10
    permissions:
      contents: read
    steps:
      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
        with:
          persist-credentials: false

      - uses: hashicorp/setup-terraform@dfe3c3f87815947d99a8997f908cb6525fc44e9e # v4.0.1
        with:
          terraform_wrapper: false

      # `terraform plan` against these providers works offline once `init` has
      # the provider binaries; no credential is needed to *render* a plan for
      # most resources. Adjust to your own layout.
      - name: Render the plan as JSON
        run: |
          terraform init
          terraform plan -out=tfplan
          terraform show -json tfplan > plan.json

      - name: Pépin — the posture gate
        uses: stephrobert/pepin/.github/actions/pepin-scan@a02e42d054dc9d8a5a41ece5b46f6c1111659a70 # v0.2.0
        with:
          version: '0.2.0'
          provider: scaleway
          terraform-plan: plan.json

  # ------------------------------------------------------------- report only
  # The same audit, without gating: the verdict is reported and published to
  # Code Scanning, and only a TECHNICAL error fails the job.
  terraform-plan-report:
    name: audit the Terraform plan (report only)
    if: github.event_name == 'pull_request'
    runs-on: ubuntu-24.04
    timeout-minutes: 10
    permissions:
      contents: read
      security-events: write   # required by upload-sarif, and by nothing else
    steps:
      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
        with:
          persist-credentials: false

      - uses: hashicorp/setup-terraform@dfe3c3f87815947d99a8997f908cb6525fc44e9e # v4.0.1
        with:
          terraform_wrapper: false

      - name: Render the plan as JSON
        run: |
          terraform init
          terraform plan -out=tfplan
          terraform show -json tfplan > plan.json

      - name: Pépin — report the posture as SARIF
        id: scan
        uses: stephrobert/pepin/.github/actions/pepin-scan@a02e42d054dc9d8a5a41ece5b46f6c1111659a70 # v0.2.0
        with:
          version: '0.2.0'
          provider: scaleway
          terraform-plan: plan.json
          format: sarif
          output-file: pepin.sarif
          fail-on-nonconformity: 'false'   # 1 and 3 warn; 2 still fails the job

      # SARIF is the format the Code Scanning tab reads. Alerts land on the
      # scanned file (the plan), because a normalized resource does not carry
      # the line of the .tf file it came from.
      - name: Publish the findings to Code Scanning
        uses: github/codeql-action/upload-sarif@5595ccaf912efad79be6eef63a5619ff05969be3 # v4.37.6
        with:
          sarif_file: pepin.sarif
          category: pepin

      # The three codes, treated explicitly. 2 has already failed the step
      # above, whatever `fail-on-nonconformity` says; this is what a reader of
      # the job summary sees.
      - name: Read the verdict
        env:
          CODE: ${{ steps.scan.outputs.exit-code }}
          VERDICT: ${{ steps.scan.outputs.verdict }}
        run: |
          case "${CODE}" in
            0) echo "::notice::compliant (${VERDICT})" ;;
            1) echo "::warning::non-compliance: at least one critical/high deviation" ;;
            3) echo "::warning::nothing measured, or medium/low deviations under --strict" ;;
            *) echo "::error::pepin could not conclude (exit ${CODE})" ; exit 1 ;;
          esac

  # -------------------------------------------------------------------- live
  # Reads the real tenant. Credentials come from repository secrets, through
  # `env:`, in the provider's own variable names — never as action inputs,
  # never in the repository. Pépin does not print them, but the evidence
  # bundle embeds the tenant's INVENTORY: keep `redact` at its default, keep
  # the artifact retention short, and remember that artifacts of a public
  # repository are downloadable by anyone with read access.
  live-scan:
    name: scan the live tenant
    if: github.event_name == 'schedule' || github.event_name == 'workflow_dispatch'
    runs-on: ubuntu-24.04
    timeout-minutes: 15
    permissions:
      contents: read
    env:
      SCW_ACCESS_KEY: ${{ secrets.SCW_ACCESS_KEY }}
      SCW_SECRET_KEY: ${{ secrets.SCW_SECRET_KEY }}
      SCW_DEFAULT_ORGANIZATION_ID: ${{ secrets.SCW_DEFAULT_ORGANIZATION_ID }}
      SCW_DEFAULT_REGION: fr-par
    steps:
      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
        with:
          persist-credentials: false

      - name: Pépin — live scan, report only, sealed evidence
        id: scan
        uses: stephrobert/pepin/.github/actions/pepin-scan@a02e42d054dc9d8a5a41ece5b46f6c1111659a70 # v0.2.0
        with:
          version: '0.2.0'
          provider: scaleway
          live: 'true'
          format: json
          output-file: report.json
          seal: bundle
          fail-on-nonconformity: 'false'   # report the posture, gate elsewhere

      # The bundle is the auditable proof; it names every resource of the
      # tenant. Short retention, and redacted by default (the action passes
      # --redact unless told otherwise). A bundle meant for a third party
      # verifies with `pepin verify` against its cosign signature.
      - uses: actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a # v7.0.1
        with:
          name: pepin-evidence
          path: |
            report.json
            bundle/
          retention-days: 7

      - name: Read the verdict without failing the job
        env:
          VERDICT: ${{ steps.scan.outputs.verdict }}
          CODE: ${{ steps.scan.outputs.exit-code }}
        run: |
          echo "posture: ${VERDICT} (exit ${CODE})"
```
<!-- /pepin:gen example-github-workflow -->

## Checklist

- [ ] every `uses:` pinned by commit SHA, this action included — no `@main`, no `@v1`
- [ ] `version:` of the action at `0.2.0` or later
- [ ] `permissions: {}` at the workflow level, least privilege per job
- [ ] `security-events: write` only on the job that uploads SARIF
- [ ] no `continue-on-error` and no `|| true` on the scan step
- [ ] `fail-on-nonconformity: 'false'` on report-only jobs, never a blanket ignore
- [ ] credentials as `env:` from secrets, in the provider's own variable names
- [ ] read-only credentials for the live scan
- [ ] evidence bundle artifact: short retention, redacted, not on a public repository
- [ ] `PEPIN_LANG` pinned if anything compares report text or bundle digests

## Where to go next

- [Exit codes](../reference/exit-codes.md) — the contract this guide gates on.
- [Output formats](../reference/output-formats.md) — what to upload, what to archive.
- [Evidence bundles](evidence-bundles.md) — sealing, verifying, sharing.
- [GitLab CI](gitlab-ci.md) — the same three postures, on GitLab.
- [Installation](../install.md) — verifying a release by hand (cosign, SLSA provenance).

## How this page stays true

The complete workflow above is not a transcription: it is the content of
`examples/github-actions/pepin.yml`, injected by `internal/docgen`, so the page cannot pin a
different SHA from the file a reader copies. `TestGeneratedDocsAreUpToDate` fails when the two
diverge. The workflow itself has not been executed by the documentation generator — a workflow
only runs on GitHub — but it is validated by `actionlint`, and the action's own promises
(provenance verification, refusal of a corrupted binary, exit-code mapping) are exercised by the
`entrypoints` workflow of this repository on every pull request.

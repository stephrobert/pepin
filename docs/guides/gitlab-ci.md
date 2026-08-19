> 🇬🇧 English · [🇫🇷 Français](gitlab-ci.fr.md)

# GitLab CI integration

The same three postures as on GitHub — gate a merge request on the Terraform plan, report
without blocking, watch the live tenant — expressed with GitLab's own vocabulary: an includable
template, `allow_failure: exit_codes`, and protected variables.

Two files are committed in `examples/gitlab-ci/` and shown in full at the end of this page: the
template you include, and a minimal pipeline that uses it.

## Install: an included template, pinned to a tag

```yaml
include:
  - remote: 'https://raw.githubusercontent.com/stephrobert/pepin/v0.2.0/examples/gitlab-ci/pepin.gitlab-ci.yml'
```

**Pin the include to a tag.** An unpinned include (`.../main/...`) runs whatever the default
branch says tomorrow — in someone else's repository, at the moment your pipeline starts.

The template downloads the released binary and **verifies its SHA-256 against the release's
checksum list before anything executes it**:

```yaml
  before_script:
    - wget -q "${PEPIN_BASE}/pepin-linux-amd64" "${PEPIN_BASE}/checksums.txt"
    - grep " pepin-linux-amd64$" checksums.txt | sha256sum -c -
    - chmod 0755 pepin-linux-amd64 && mv pepin-linux-amd64 /usr/local/bin/pepin
```

A corrupted download stops the job right there, with the mismatch named in the log. This is
integrity, not authenticity: the binary and the checksum list come from the same origin, so
whoever could replace one could replace both. The stronger half of the chain — the cosign
signature over that list, and the SLSA provenance attestation — needs `cosign` or `gh` in the
image, and the commands are in [Installation](../install.md). If your runner image can carry
`cosign`, verify the signature in `before_script` too.

The version lives in one variable, so a bump is one line:

```yaml
variables:
  PEPIN_VERSION: "0.2.0"
```

## Audit a Terraform plan, with no credential

The `plan` job renders the plan; the Pépin job audits it. Nothing is provisioned, no secret is
needed, and the feedback lands on the merge request.

```yaml
stages: [plan, test]

plan:
  stage: plan
  image:
    name: hashicorp/terraform:1.15
    entrypoint: [""]
  script:
    - terraform init
    - terraform plan -out=tfplan
    - terraform show -json tfplan > plan.json
  artifacts:
    paths: [plan.json]
    expire_in: 1 day

pepin-terraform-plan:
  extends: .pepin
  stage: test
  script:
    - pepin scan scaleway --terraform plan.json --format json > pepin-report.json
  artifacts:
    when: always
    paths: [pepin-report.json]
    expire_in: 1 week
```

As written, this job **gates**: any non-zero exit code fails the pipeline.

## The exit codes: `allow_failure: exit_codes: [1, 3]`, never 2

To report the posture without gating on it, list the **verdict** codes, and only them:

```yaml
pepin-terraform-plan:
  extends: .pepin
  allow_failure:
    exit_codes: [1, 3]
```

| Exit | Meaning | Gating job | `allow_failure: exit_codes: [1, 3]` |
|:-:|---|---|---|
| **0** | compliant | passes | passes |
| **1** | critical/high deviation | **fails** | warning, pipeline continues |
| **3** | nothing measured, or medium/low under `--strict` | **fails** | warning, pipeline continues |
| **2** | technical error | **fails** | **fails** — 2 is not in the list |

**Never write `exit_codes: [1, 2, 3]`, and never `allow_failure: true`.** Both turn "the scan
could not conclude" into a green pipeline: unreachable API, expired credentials, an unreadable
plan — all reported as a posture that was never measured. `allow_failure: true` is the blunt
version of the same mistake, because it covers every code at once.

Note that 3 is a verdict, not an error, and it deserves attention rather than dismissal: it
means the run measured nothing (governance aside), which on a live scan usually points at
credentials or permissions ([Exit codes](../reference/exit-codes.md)).

## Scan the live tenant

```yaml
pepin-live:
  extends: .pepin
  stage: test
  rules:
    - when: manual
  script:
    - pepin scan scaleway --live --format json --seal bundle --redact > pepin-report.json
  allow_failure:
    exit_codes: [1, 3]
```

Credentials come from **masked and protected** CI/CD variables (Settings → CI/CD → Variables),
under the provider's own names — never in the YAML, never echoed in a script line:

| Provider | Variables |
|---|---|
| Scaleway | `SCW_ACCESS_KEY`, `SCW_SECRET_KEY`, `SCW_DEFAULT_ORGANIZATION_ID`, `SCW_DEFAULT_REGION` |
| Outscale | `OSC_ACCESS_KEY`, `OSC_SECRET_KEY`, `OSC_REGION` |
| Exoscale | `EXOSCALE_API_KEY`, `EXOSCALE_API_SECRET`, `EXOSCALE_ZONE` |

*Protected* keeps the variable off unprotected branches, where a merge request from anywhere
could read it. *Masked* keeps it out of job logs. Use a **read-only** key, scoped to what Pépin
actually calls: [Scaleway](../providers/scaleway.md#minimal-permissions-for-a-live-scan),
[Outscale](../providers/outscale.md#minimal-permissions-for-a-live-scan),
[Exoscale](../providers/exoscale.md#minimal-permissions-for-a-live-scan).

`rules: - when: manual` keeps the live scan off every push. A scheduled pipeline
(CI/CD → Schedules) is the other sensible trigger; a live scan on every commit reads a real
tenant far more often than the tenant changes.

## Archive the evidence bundle

```yaml
  artifacts:
    when: always
    paths:
      - pepin-report.json
      - bundle/
    expire_in: 1 week
    public: false
```

Four decisions, all deliberate:

- **`--redact`** in the script: user data and policy documents are replaced by their digest. The
  cost is that a redacted bundle cannot be re-derived
  ([Evidence bundles](evidence-bundles.md#redact-a-bundle-you-hand-to-a-third-party)).
- **`when: always`**: the report is archived even when the job fails, which is precisely the run
  you will want to read.
- **`expire_in: 1 week`**: a bundle names every resource of the tenant. It is not a permanent
  artifact of a build; if you need to keep one, move it to an evidence store with access
  control.
- **not public**: keep the artifact off public pipeline pages.

## Pin the language

```yaml
variables:
  PEPIN_LANG: "en"
```

Exit codes and identifiers are stable across languages, so a gate needs nothing. A job that
diffs report *text* between runs, or compares bundle digests, must pin the language — a runner
image whose locale changes would otherwise produce a difference nobody caused.

## The complete template

`examples/gitlab-ci/pepin.gitlab-ci.yml` — the file the `include` above points at. It ships the
three jobs: gating, report-only, and manual live scan.

<!-- pepin:gen example-gitlab-template -->
```yaml
# Pépin in GitLab CI — an includable template.
#
#   include:
#     - remote: 'https://raw.githubusercontent.com/stephrobert/pepin/v0.2.0/examples/gitlab-ci/pepin.gitlab-ci.yml'
#
# then `extends: .pepin` in your own job (see .gitlab-ci.yml beside this file).
# The remote include is pinned to a tag on purpose: an unpinned include runs
# whatever the default branch says tomorrow.
#
# The released binary is downloaded and its SHA-256 verified against the
# release's checksum list BEFORE anything runs it. The stronger half of the
# chain (the cosign signature over that list, the SLSA provenance) needs
# cosign or gh; docs/install.md carries those commands.
#
# Exit codes are the contract: 0 compliant, 1 non-compliance, 2 technical
# error, 3 nothing measured (or medium/low under --strict). The default below
# fails the job on any non-zero code. To *report* the posture without gating
# on it, allow the verdict codes and only them — never 2, which means the scan
# could not conclude:
#
#   pepin-scan:
#     extends: .pepin
#     allow_failure:
#       exit_codes: [1, 3]

variables:
  PEPIN_VERSION: "0.2.0"
  # Codes, identifiers, severities and exit codes are identical in both
  # languages; titles, messages and remediations are not. A pipeline that
  # compares report TEXT between runs pins the language.
  PEPIN_LANG: "en"

.pepin:
  image: alpine:3.21
  variables:
    PEPIN_BASE: "https://github.com/stephrobert/pepin/releases/download/v${PEPIN_VERSION}"
  before_script:
    - wget -q "${PEPIN_BASE}/pepin-linux-amd64" "${PEPIN_BASE}/checksums.txt"
    # The bytes are checked before anything runs them; a corrupted download
    # stops the job here, with the checksum mismatch named in the log.
    - grep " pepin-linux-amd64$" checksums.txt | sha256sum -c -
    - chmod 0755 pepin-linux-amd64 && mv pepin-linux-amd64 /usr/local/bin/pepin

# Audit a Terraform plan: no credential, nothing provisioned. Expects a
# `plan.json` artifact from an earlier stage (`terraform show -json`).
# Override PEPIN_PROVIDER and the path to match your pipeline.
#
# This job GATES: any non-zero code fails the pipeline, and the three of them
# stay distinguishable in the job log and in its exit code.
pepin-terraform-plan:
  extends: .pepin
  stage: test
  variables:
    PEPIN_PROVIDER: scaleway
  script:
    - pepin scan "${PEPIN_PROVIDER}" --terraform plan.json --format json > pepin-report.json
  artifacts:
    when: always
    paths: [pepin-report.json]
    expire_in: 1 week

# The same audit, reporting instead of gating. `allow_failure.exit_codes`
# lists the VERDICT codes and only them: 2 is absent, so a technical error
# still fails the pipeline. Listing it would turn "the scan could not
# conclude" into a green pipeline.
pepin-terraform-plan-report:
  extends: .pepin
  stage: test
  variables:
    PEPIN_PROVIDER: scaleway
  script:
    - pepin scan "${PEPIN_PROVIDER}" --terraform plan.json --format json > pepin-report.json
  allow_failure:
    exit_codes: [1, 3]
  artifacts:
    when: always
    paths: [pepin-report.json]
    expire_in: 1 week

# Live scan of the real tenant — disabled until triggered by hand, because it
# reads a real account. Credentials come from masked, protected CI/CD
# variables in the provider's OWN names (Settings > CI/CD > Variables):
#
#   Scaleway  SCW_ACCESS_KEY, SCW_SECRET_KEY, SCW_DEFAULT_ORGANIZATION_ID,
#             SCW_DEFAULT_REGION
#   Outscale  OSC_ACCESS_KEY, OSC_SECRET_KEY, OSC_REGION
#   Exoscale  EXOSCALE_API_KEY, EXOSCALE_API_SECRET, EXOSCALE_ZONE
#
# Never write them in this file, and never echo them in a script line. Pépin
# reads them from the environment and does not print them.
#
# The evidence bundle embeds the tenant's inventory. It is sealed with
# --redact here (secrets like user-data and policy documents are replaced by
# their digest), the artifact expires after a week, and it is kept away from
# public pipeline pages. A bundle for a third party verifies with
# `pepin verify` against its cosign signature; only drop --redact when you
# need `verify --re-derive`, and then treat the bundle itself as a secret.
pepin-live:
  extends: .pepin
  stage: test
  rules:
    - when: manual
  variables:
    PEPIN_PROVIDER: scaleway
  script:
    - pepin scan "${PEPIN_PROVIDER}" --live --format json --seal bundle --redact > pepin-report.json
  allow_failure:
    exit_codes: [1, 3]
  artifacts:
    when: always
    paths:
      - pepin-report.json
      - bundle/
    expire_in: 1 week
    public: false
```
<!-- /pepin:gen example-gitlab-template -->

## The pipeline that uses it

`examples/gitlab-ci/.gitlab-ci.yml` — copy this into your Terraform repository.

<!-- pepin:gen example-gitlab-pipeline -->
```yaml
# Gate a GitLab pipeline on cloud posture with Pépin — minimal usage.
#
# Copy into a repository holding a Terraform configuration for Exoscale,
# Outscale or Scaleway. The `plan` job renders the plan; the included template
# audits it with no credential and nothing provisioned. The pipeline fails on
# a non-compliant posture (exit 1), on a scan that measured nothing (exit 3)
# and on a technical error (exit 2), and the three are distinguishable in the
# job log and in its exit code.

include:
  # Pinned to a tag: an unpinned include runs whatever the default branch
  # says tomorrow.
  - remote: 'https://raw.githubusercontent.com/stephrobert/pepin/v0.2.0/examples/gitlab-ci/pepin.gitlab-ci.yml'

stages: [plan, test]

plan:
  stage: plan
  image:
    name: hashicorp/terraform:1.15
    entrypoint: [""]
  script:
    - terraform init
    - terraform plan -out=tfplan
    - terraform show -json tfplan > plan.json
  artifacts:
    paths: [plan.json]
    expire_in: 1 day

# The template ships three jobs: `pepin-terraform-plan` (gating),
# `pepin-terraform-plan-report` (report only, allow_failure on the verdict
# codes 1 and 3 but never on 2) and `pepin-live` (manual). Keep the one you
# want and disable the others, for instance:
#
#   pepin-terraform-plan-report:
#     rules:
#       - when: never
```
<!-- /pepin:gen example-gitlab-pipeline -->

## Checklist

- [ ] `include: remote:` pinned to a tag, never to a branch
- [ ] `PEPIN_VERSION` at `0.2.0` or later
- [ ] the binary's checksum verified before it runs (the template does it)
- [ ] `allow_failure: exit_codes: [1, 3]` on report-only jobs — **never 2**, never
      `allow_failure: true`
- [ ] credentials in masked and protected CI/CD variables, under the provider's own names
- [ ] read-only credentials for the live scan
- [ ] the live job behind `when: manual` or a schedule
- [ ] bundle artifact: `--redact`, short `expire_in`, not public
- [ ] `PEPIN_LANG` pinned if anything compares report text or bundle digests

## Where to go next

- [Exit codes](../reference/exit-codes.md) — the contract `allow_failure` gates on.
- [Output formats](../reference/output-formats.md) — what to archive, and what is safe to parse.
- [Evidence bundles](evidence-bundles.md) — what is inside a bundle, and how to verify it.
- [GitHub Actions](github-actions.md) — the same three postures, on GitHub.

## How this page stays true

The two complete files above are injected from `examples/gitlab-ci/` by `internal/docgen`, so
the page cannot show a version pin that differs from the file a reader copies.
`TestGeneratedDocsAreUpToDate` fails when they diverge. The pipelines themselves have not been
executed by the documentation generator — they only run on GitLab — but every `pepin` command
they contain is documented, with its real output and its real exit code, in
[Exit codes](../reference/exit-codes.md) and [Output formats](../reference/output-formats.md).

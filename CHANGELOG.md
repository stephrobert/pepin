> 🇬🇧 English · [🇫🇷 Français](CHANGELOG.fr.md)

# Changelog

Notable changes, in the format of [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
versioned according to [Semantic Versioning](https://semver.org/).

This file is read by the release workflow: the section matching a tag becomes
the body of its GitHub Release. An entry that is not here is an entry nobody
downloading a binary will ever see.

Two kinds of change deserve their own line whatever their size, because they
are what a compliance chain built on Pépin is judged on: **a surface a
consumer's pipeline parses** (the assessment, findings, bundle or OSCAL shape,
an exit code, a CLI verb or flag), and **a verdict that can change on an
unchanged tenant** (a rule tightened or loosened, a control activated or
retired, a normative mapping retriaged). The first breaks their parsing; the
second makes their user explain to an auditor a change they did not make, and
this file is where that explanation starts. A refactor that changes neither
belongs in `git log`.

## [Unreleased]

### Added

- **Pépin is bilingual, and detects the language.** Reports, verdict, help,
  errors and the parsable formats (`json`, `sarif`, `oscal`, `assessment`) come
  out in French or in English. Resolution order:
  `--lang=fr|en` → `PEPIN_LANG` → `LC_ALL` → `LANG` → fallback `en`; the first
  non-empty source decides, and an unknown locale falls back to English without
  an error. Until now the skeleton was English and the content French, so a
  reader got a report in two languages within one sentence.
  French remains the reference language of the normative content: the reference
  and the rules are written in French first, and where a legal reading is at
  stake it is the French wording of a control that governs.

### Changed

- **A finding's prose changes with the language, its keys do not.** Codes
  (`CLD-*`), check identifiers, severities, statuses, subjects and exit codes are
  identical in both languages; titles, messages, remediations and evidence are
  translated. A pipeline that diffs report *text* between runs should pin
  `PEPIN_LANG`. A pipeline keyed on codes and statuses is unaffected.
- **A sealed bundle carries the language of the scan that produced it.**
  `verify --re-derive` replays the rules in both languages and accepts either
  match, so verifying a French bundle from an English shell is not reported as a
  falsification. Note that the bundle digest does depend on the language, since
  the assessment's prose is part of what is sealed.
- **CLI surface v1 → v2**: the persistent `--lang` flag is added. Pure addition —
  no verb, no other flag and no exit code moved.

## [0.1.1] - 2026-08-19

### Fixed

- **A Scaleway instance whose security group is created by the same plan is no
  longer reported `CRITICAL` "VM without a security group".** At plan time
  `security_group_id` is *unknown after apply*, so it is absent from
  `planned_values`; the `list` transform then fabricated an empty collection,
  which satisfied the rule's capability guard — the guard that exists precisely
  to prevent this. A collection transform now only runs when the source key
  actually exists. Absent means the source does not expose the information;
  present-and-empty is information.

  This changes a verdict on an unchanged tenant: the control moves from `fail`
  to `not-evaluated` on a Terraform plan. On a plan, an instance genuinely
  without a security group is indistinguishable from one whose group is not yet
  known, so Pépin now says so instead of guessing. The live path benefits too,
  where an API omitting a key produced the same fabricated `[]`.

  Found by replaying fifteen third-party Terraform stacks against the binary.

- **The documentation drift gate now compiles what it measures.** It reused a
  `./pepin` already present at the repository root, so a stale binary could
  validate stale pages — it did, reporting "up to date" while the docs still
  advertised the finding above.

### Added

- **Product documentation, generated rather than transcribed.** Six pages in
  English with synchronised French counterparts: a five-minute quickstart that
  needs no cloud account, the assessment model (`pass` / `fail` /
  `not-applicable` / `not-evaluated`), the provider × control coverage matrix,
  known limitations, a commented walkthrough of a real scan, and the exact scope
  and non-goals. Every command output is captured from a real run of the binary,
  the coverage matrix is computed from the reference and the provider
  descriptors, and a CI gate fails when either drifts.

- **Fuzzing over untrusted inputs** — `FuzzParsePlan` and `FuzzInventoryWalk`,
  covering the Terraform plan and inventory export paths. It immediately found a
  resource with an empty type entering the model, now rejected and kept as a
  regression.

### Security

- `SECURITY.md` now links its private reporting channel instead of only
  describing it.

## [0.1.0] - 2026-08-19

### Security

- **Policies loaded at runtime no longer get network access.** `--policy-dir`
  compiled third-party Rego with OPA's default capabilities, `http.send`
  included, so an eight-line rule could POST the evaluated inventory — instance
  user-data, IAM policy documents, bucket policies — to an arbitrary host, or
  reach the runner's internal network from inside the scanner. Fixed upstream in
  `scankit v0.2.2`; a policy calling one of those builtins now fails to compile.
  Policy evaluation also gets a five-minute deadline.
- **Provider credentials no longer survive an HTTP redirect.** Go strips only
  `Authorization`, `Cookie` and `WWW-Authenticate` across domains — not
  `X-Auth-Token` (the Scaleway secret key) nor `AccessKey`/`SecretKey`
  (Outscale). One 302 toward a controlled host handed them over. The collection
  client no longer follows redirects.
- **`pepin verify` no longer reads outside its bundle.** Artifact names came
  from the manifest, supplied by the audited third party, so `../secret` turned
  verification into an existence-and-content oracle. Names must be plain
  basenames.
- **`--seal --redact` no longer ships the tenant's keys.** Redaction covered
  free-form documents only, while `access_key` is a first-class attribute of the
  normalized model and `password`/`certificate` come from managed databases.
- **Toolchain moved to Go 1.26.6**, which resolves five standard-library
  advisories reachable from this code (`net/url`, `crypto/tls`, `encoding/xml`,
  `encoding/asn1`, `net/http`).
- **The published action verifies authenticity, not just integrity.** The binary
  and `checksums.txt` come from the same origin, so whoever can replace release
  assets replaces both. `install.sh` now verifies build provenance via
  `gh attestation verify`.

### Fixed

Every item below can change a verdict on an unchanged tenant.

- **A scan that measured nothing no longer exits `0`.** Expired credentials,
  insufficient rights or a truncated inventory produced the same empty result as
  a clean tenant, and the CI gate went green on a scope never looked at. Exit
  code `3` now says so without requiring `--strict`.
- **Fourteen controls no longer report `pass` without the deciding data.** The
  capability gate gained thirteen entries, and an empty collection no longer
  counts as collected — the IAM collector always sets `statements`, at `[]` when
  a document fails to parse, so four critical/high controls concluded
  "compliant" over zero information.
- **`authenticated-read` and `AuthenticatedUsers` are detected** as public
  exposure: both grant read access to every authenticated user of the platform,
  which is cross-tenant.
- **A bucket made public by an inline `acl`** on `scaleway_object_bucket` is
  collected at last; it previously produced zero findings and a "compliant"
  verdict.
- **Booleans arriving as strings are honoured.** A Terraform plan renders some
  schema attributes as `"true"`/`"false"`, and `== false` is simply false for
  `"false"`; 25 comparisons across 16 rules now go through `truthy()`.
- **An uncatalogued region is reported** instead of silently passing: the
  classification tables are allow-lists, so their silence read as "in the EU".
- **Network normalization**: `-1`, `any` and an empty protocol all mean "every
  protocol", and a scalar where the model expects a list no longer makes the
  rule undefined — an export carrying `"cidrs": "0.0.0.0/0"` went unreported.
- **`CLD-CHF-2` severities aligned** on `high` across its three controls;
  severity drives the CI gate, and the split was unjustified.

### Added

- **The public surface is frozen by tests, not by prose.** The CLI's verbs,
  flags and exit codes, the `--format json` findings document, the assessment
  document and the evidence-bundle layout each have a committed fixture under
  `cmd/testdata/frozen/` — the field tree, never a value. A shape that moves
  without its fixture fails CI; a fixture regenerated without its declared
  version moving fails CI too. The bundle's version travels on the wire as the
  `/vN` suffix of `format` in `manifest.json`; a verifier that meets a version
  it does not know should stop rather than guess.
- **The SCSL index is watched for drift.** `mise run scsl-drift` compares the
  live `framework-scsl` index against a baseline committed in
  `referentiel/scsl-baseline.json` and exits 2 when a CLD requirement was
  added, removed or rewritten upstream without a human retriaging the
  mappings. Note the tooling exit convention (0 ok, 1 error, **2 drift**)
  is deliberately distinct from `pepin scan`'s (where 2 is a technical error).
- **A release is refused before the tag, not regretted after it.**
  `mise run release-check -- vX.Y.Z` replays offline everything that must
  hold: clean tree on `main`, a free tag, tests and referential coherence,
  zero SCSL drift, the exit codes answered by the built binary rather than
  read from a constant, a sealed bundle that verifies, re-derives **and
  refuses itself once tampered with**, the version the Conventional Commits
  imply (`.cz.toml`), and both CHANGELOGs carrying the section the release
  body is read from.
- **A tag builds, attests and signs the release.**
  `.github/workflows/release.yml` builds `linux`/`darwin` × `amd64`/`arm64`
  binaries with the tag stamped in, generates SHA-256 checksums and a
  CycloneDX SBOM, records SLSA build provenance, signs the checksums with
  keyless Cosign, and publishes the GitHub Release with this file's matching
  section as its body.
- **A container image** (`ghcr.io/stephrobert/pepin`, one tag per release, no
  `latest`): the released linux binaries on a distroless base pinned by
  digest — CA roots for `--live`'s TLS, user 65532, no shell. Nothing is
  compiled in the Dockerfile, so the release's checksums, SBOM and provenance
  describe the image's content too; the image carries its own SLSA
  provenance, SBOM attestation and keyless signature, and the release refuses
  an image whose `pepin version` is not the tag or whose exit codes moved
  through `docker run`.
- **A composite GitHub action** (`.github/actions/pepin-scan`) that verifies
  the downloaded binary's SHA-256 against the release's checksum list before
  running it, scans a Terraform plan, an inventory or the live API, and turns
  the exit codes into a gate: `fail-on-nonconformity: 'false'` downgrades a
  non-compliant verdict (1, or 3 under strict) to a warning, and never
  downgrades a technical error (2). Credentials are never action inputs; the
  provider's native variables come from `env:`. CI corrupts one byte of the
  download and requires the refusal (`entrypoints.yml`), and every release
  replays the action against its own published artefacts.
- **A GitLab CI template and CI examples**
  (`examples/gitlab-ci/`, `examples/github-actions/`): same verified
  download, same exit-code contract, report-only via
  `allow_failure: exit_codes: [1, 3]` — never 2. Installation and
  verification for all four entry points are documented in
  `docs/install.md` / `docs/install.fr.md`.

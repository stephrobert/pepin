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

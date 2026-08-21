> 🇬🇧 English · [🇫🇷 Français](RELEASING.fr.md)

# Releasing

A release is a tag. There is no version to bump in any file: `release.yml`
stamps the tag into the binary with `-ldflags`
(`github.com/stephrobert/pepin/cmd.version`), and a binary built any other way
answers `pépin 0.1.0-dev`. Nothing can drift from the tag, because nothing else
holds the number.

## Which number

The commits decide, not taste. Subjects follow
[Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/), and
commitizen derives the increment from them (`.cz.toml`,
`version_provider = "scm"` so the tags are the only source):

```bash
uvx --from 'commitizen==4.16.5' cz bump --dry-run   # e.g. "tag to create: v0.2.0"
```

`fix` moves the patch, `feat` the minor, a `!` marks a break, and
`major_version_zero` keeps a break inside `0.x`. The preflight below checks
that the tag being asked for is the one the commits imply, and reports rather
than refuses when commitizen cannot be reached at all — a maintainer without
the Python tooling must still be able to cut a release.

## Cutting one

1. **Move the `Unreleased` entries** of [CHANGELOG.md](./CHANGELOG.md) **and**
   [CHANGELOG.fr.md](./CHANGELOG.fr.md) under a new `## [X.Y.Z]` heading, in
   both files. The release workflow reads the English section for the body of
   the GitHub Release; the preflight refuses a version present in one language
   and absent from the other, because the repository promises the two stay
   synchronized.

2. **Merge to `main` through a pull request, and wait for CI to be green.**
   The tag builds from that commit, and a published release cannot be
   replayed.

3. **Re-record the demo GIFs** at the version being released, so neither
   README's front page keeps showing an older product:

   ```bash
   PEPIN_DEMO_VERSION=v0.1.0 mise run demo
   ```

   One recording per language — `quickstart.gif` for `README.md`,
   `quickstart.fr.gif` for `README.fr.md` — from two tapes that run exactly the
   same commands and differ only by the language they impose. The preflight
   requires both.

   Both a GIF and an MP4 come out of the same run: the GIF is what the READMEs
   show (it animates without a player), the MP4 is for places that refuse a GIF
   or want a fraction of the weight.

   The version is injected into the build, never typed into the tape, and
   `docs/assets/quickstart.version` records it. The preflight compares that file
   with the tag and refuses to proceed if they differ.

4. **Run the preflight**, which replays offline everything that must hold:

   ```bash
   mise run release-check -- v0.1.0
   ```

   It checks: a clean tree on `main`; the tag free locally *and* on origin;
   `mise run test` (Go with `-race`, the Rego suites, and the frozen-surface
   gates), `mise run validate` and `mise run vet`; **zero SCSL drift** (see
   below); the exit codes answered by the **built binary** on the example
   inventories rather than read back from a constant; a bundle sealed by that
   binary that verifies, re-derives, **and is refused once a single byte is
   altered** — a `verify` that accepts everything proves nothing; the version
   the commits imply; and both CHANGELOGs carrying the section. It reports
   every verdict rather than stopping at the first, and prints the tag
   commands only when everything holds.

   It deliberately does **not** rerun `mise run audit` or the NIST OSCAL
   schema check: those need the network and tools this machine may not have.
   CI runs them on every push; the preflight asks CI whether it did, on this
   exact commit.

5. **Tag and push**:

   ```bash
   git tag -a v0.1.0 -m "v0.1.0"
   git push origin v0.1.0
   ```

Pushing the tag is what publishes. It cannot be undone quietly: a tag has to
be deleted on both sides, and a release that reached the world has been
downloaded.

### What stays manual

- `framework-scsl` must be cloned next to the repository: the SCSL drift check
  reads the live index, and a release cut without it is refused rather than
  cut blind.
- `gh` must be authenticated for the preflight to ask CI about the commit.
- Moving the CHANGELOG entries is writing, not generation, in both languages.
- **The canary scan** (below): a maintainer's gesture, run locally, whose
  result is recorded and dated.

## The canary: what the real control planes answered

```bash
mise run canary                  # the three cloud providers
tools/release/canary.sh scaleway # one of them
```

The `live` column of the coverage matrix is **derived from the descriptors**: it
says what Pépin believes it can collect, never what it has observed. A release
that promotes a live capability validated only on fixtures and an emulator
promotes a belief.

The canary is the one measurement in this repository that queries a sovereign
provider's **real** control plane — and it holds **no credential**, which is the
point. It sends **synthetic** values that the provider refuses, and what it
measures is the refusal: an endpoint answering 401 or 403 exists, resolves and
speaks; a moved endpoint would answer 404, which is exactly the regression a
descriptor cannot see coming. The script clears every credential variable and
sets a disposable `HOME` before it runs anything, so a maintainer cannot send
their keys by accident — a property of the script, not a usage instruction.

| It establishes | It does **not** establish |
|---|---|
| that the host compiled into the descriptor resolves and answers | the names and types of the native contract's fields |
| that the declared path still exists (a 404 would say otherwise) | what a tenant contains |
| the class a refusal is given | that a **sufficient** right returns `200` |

A **signature** refusal is not a **right** refusal. The canary therefore does
**not** count as live validation of a control, and the detection-quality map
counts zero on that side rather than borrowing this figure.

Records live in `references/canary/<provider>.yaml`, are committed, and carry
only endpoint facts: unit, method, host, path, HTTP status, and the class Pépin
gave the refusal. No query string is kept — that is the one place a tenant value
could slip into a URL. **Read a record before committing it**; the canary
produces nothing else, and checking is what lets us say so.

The two halves are checked in two different places, deliberately:

- **completeness** — every cloud provider has a readable, substantive record —
  is checked by `internal/canary`, so by `mise run test` and CI, because it does
  not depend on the date;
- **freshness** — no record older than `canary.MaxAge` (90 days) — is checked by
  the preflight, the only moment the question arises. A test that reddened with
  the passage of time would redden one morning with nothing changed, and be
  disarmed within the week.

A record whose units are **all** unreachable is refused rather than written: it
would describe the maintainer's network (proxy, sandbox, captive DNS), not the
provider, and a later reader would take those `unreachable` entries for a
control-plane regression that never happened.

> **The preflight is a local check, not a gate.** Nothing ties the tag to having
> run it: `git tag && git push` publishes a full signed, attested release on its
> own. `release.yml` therefore replays the offline-verifiable gates in a `gate`
> job that every publishing job depends on. The other half — who may create a
> tag — is repository configuration and cannot live in the repository, so it is
> applied once, from here:
>
> ```bash
> mise run tag-ruleset            # or: tools/release/apply-tag-ruleset.sh owner/repo
> ```
>
> It restricts creating, deleting and moving `v*` tags to administrators, so a
> tag pushed by anyone else is refused *before* release.yml starts. Run it once
> the remote exists; it is idempotent.

## What the tag triggers

`.github/workflows/release.yml`, on `v*`:

- builds `linux` and `darwin` binaries for `amd64` and `arm64`, with the tag
  stamped in, and **refuses to publish** a binary that does not answer the tag
  to `version` or that has lost the documented exit codes,
- generates SHA-256 checksums and a CycloneDX SBOM (Syft),
- records **SLSA build provenance** and attests the SBOM,
- signs the checksums with **keyless Cosign**,
- builds and pushes the **container image** (`ghcr.io/stephrobert/pepin`, one
  tag per release, no `latest`) from the very linux binaries above — nothing
  is compiled in the Dockerfile — after refusing an image whose
  `pepin version` is not the tag or whose exit codes moved through
  `docker run`; the image gets its own SLSA provenance, its own SBOM
  attestation, and a keyless signature over its digest,
- creates the GitHub Release with every artefact attached, including
  `provenance.intoto.jsonl` — the file OpenSSF Scorecard's *Signed-Releases*
  control looks for, distinct from the attestation recorded through GitHub's
  API,
- then exercises the composite action (`.github/actions/pepin-scan`) against
  the release it just published — the only moment that test can exist — and
  requires the documented exit-code contract: 0 passes, 1 fails the job, 1
  only warns under `fail-on-nonconformity: 'false'`, and 2 fails whatever
  that input says.

Anyone can then check a binary really came from this repository's workflow:

```bash
gh release download v0.1.0 --repo stephrobert/pepin --pattern 'pepin-linux-amd64' --pattern 'checksums.txt*'
gh attestation verify pepin-linux-amd64 --repo stephrobert/pepin

cosign verify-blob --bundle checksums.txt.cosign.bundle \
  --certificate-identity-regexp 'https://github.com/stephrobert/pepin/\.github/workflows/release\.yml@refs/tags/v' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  checksums.txt
sha256sum -c checksums.txt --ignore-missing
```

## What you may depend on, and for how long

The surfaces below are the whole of it. **Anything not on this list carries no
promise at all** — in particular, the *set* of controls grows with every
release, and a verdict may legitimately change when a rule stops being wrong;
both get their CHANGELOG line, neither is a break.

| surface | version signal | frozen as |
|---|---|---|
| the CLI's verbs, flags and exit codes | `cliSurfaceVersion` (`cmd/surface.go`) | `cmd/testdata/frozen/cli.json` |
| `--format json` (`findings` + `summary`) | `findingsSurfaceVersion` (`cmd/surface.go`) | `cmd/testdata/frozen/findings.json` |
| the assessment document (`--format assessment`, `assessment.json` in a bundle) | `assess.AssessmentSurfaceVersion` | `cmd/testdata/frozen/assessment.json` |
| the evidence bundle (files, roles, manifest) | the `/vN` suffix of `format` in `manifest.json` (`assess.BundleFormat`) | `cmd/testdata/frozen/bundle.json` |
| `--format oscal` | OSCAL 1.1.2 itself | the NIST schema check in `ci.yml` |
| the control `code` identifiers | — | `referentiel/controles.yaml`, gated by `mise run validate` |

What is frozen is the **form** — field paths and JSON types, verb and flag
names — never a value. Two tests gate it on every run of `mise run test`:
`TestTheFrozenSurfacesStillMatchTheirFixture` fails when a shape moves and the
fixture did not, and `TestASurfaceChangeDemandsItsVersionBump` fails when the
fixture moved and the declared version did not.

Changing one of these surfaces on purpose is four steps, in one commit: change
the code; `mise run frozen-update` (it appends to the fixture's history at the
next version, never rewriting an entry); bump the matching constant — the
tests stay red until this step, which is the point; write the CHANGELOG line
the bump is the signal for.

### The signal a consumer reads

Only the bundle carries its version on the wire: the `format` field of
`manifest.json`, `pepin-assessment-bundle/vN`. A verifier that meets a version
it does not know should **stop rather than guess**. The other surfaces have no
wire field to carry one — their signal is this table, the constants, and the
CHANGELOG line every bump owes.

### The notice

**One minor release.** A field, verb or exit code that is going away is
deprecated in one release and removed no earlier than the next, and both
events are lines in the CHANGELOGs. One release rather than a generous number,
deliberately: this is a 0.x project with one maintainer, and a promise of one
minor version, kept, is worth more than a longer one that is broken.

## Upstream drift: the SCSL index

Pépin's controls map onto the frozen CLD requirements of the SCSL framework,
and that index lives outside this repository and moves under it. A control
whose normative reference was rewritten upstream and that nobody retriaged is
the same defect as an untriaged operation in any scanner: the report would
cite a reference that no longer says what the mapping assumed.

```bash
mise run scsl-drift          # exit 2 if the live index moved since the baseline
mise run scsl-drift-update   # rewrite the baseline, after human triage
```

The baseline (`referentiel/scsl-baseline.json`) is committed, so the knowledge
of what was triaged travels with the repository. The tooling exit convention —
0 nothing to report, 1 error, **2 untriaged drift** — is deliberately distinct
from `pepin scan`'s (`0` compliant, `1` non-compliance, `2` technical error):
one speaks to a release gate, the other to a compliance gate, and the
preflight tells them apart.

## From the second release onward

One check cannot exist before a first release does, and saying so plainly
beats a green gate: replaying the previous release against this one. The risk
it closes is the auditor's — a bundle sealed by release N−1 must still verify
and re-derive under release N, or the change that broke it must be a
CHANGELOG line and a bundle-format bump. Once `v0.1.0` exists, before each
tag:

```bash
git worktree add /tmp/pepin-prev v0.1.0 && (cd /tmp/pepin-prev && mise run build)
/tmp/pepin-prev/pepin scan scaleway examples/scaleway/inventory.json --seal /tmp/bundle-prev || true
./pepin verify /tmp/bundle-prev --re-derive   # the NEW binary reads the OLD bundle
git worktree remove /tmp/pepin-prev
```

When that command earns automation, it belongs in the preflight next to the
fresh-bundle round trip.

## Versioning

[Semantic Versioning](https://semver.org/). Before 1.0, the minor moves on
anything a consumer can observe on the surfaces above.

What counts as breaking is narrower than it looks. **Covering more** — new
controls, new providers, new collected attributes — is never breaking: a
tenant that gains findings gained coverage, and the strict gate exists
precisely for pipelines that want that surfaced. **A verdict corrected to
match the provider's real contract is a fix, not a break**, even when a
pipeline depended on the wrong verdict: a test that relied on a false pass was
measuring Pépin rather than the cloud. What breaks is on this project's own
side: the shapes, codes, verbs and formats in the table above.

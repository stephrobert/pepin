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

- **A recorded collection session, and the gate that replays it.** A provider
  descriptor declares endpoints; nothing proved the collector *emits* them, which is
  the exact shape of the inline-EIM incident (right rule, data that never arrived, no
  Rego test able to see it). `mise run trace` now records a real `--live` collection
  through an intercepting proxy, against a **local emulator** and with **no cloud
  credential**, and the recording is committed to
  `internal/genprovider/testdata/transcripts/`. Two gates replay it on every build:
  `TestTheRecordedCollectionStillHappens` (fewer calls than the recording saw means a
  datum stopped arriving; more means an endpoint declared but never measured) and
  `TestEveryDeclaredEndpointIsObservedOrDeclaredUnobserved` (the `non_observes` ledger
  is exact in both directions). The replay serves the **recorded** responses, never
  responses derived from the spec under test: a harness answering "what the spec
  expects" would measure its own copy of the spec and stay green on a wrong `items:`.
  Measured on the first run: no declared endpoint stays silent, except three Outscale
  child joins and one Exoscale child join whose parent list is not served by the
  emulator or came back empty — each now written down with its reason. New guide:
  [Tracing real API calls](docs/guides/tracing-api-calls.md). No line of Pépin
  changed, so no collection endpoint became overridable and no exfiltration surface
  was created.

- **A veracity contract, and a debt counter rather than a green matrix.** For each
  control × provider × source path, `internal/veracity` derives the verdicts that
  path can actually reach — three where it can conclude, one where it cannot lift
  the `pass` lock, one where the provider contract declares it not applicable — and
  compares them to committed scenarios that run **against the binary**, over the
  whole chain: canned API responses served to the descriptor's real collection spec,
  or a minimal Terraform plan passed to its real mapper. What is not proven is
  recorded in `internal/veracity/testdata/debt.txt`, a gate in both directions: an
  unproven obligation missing from the ledger fails the build, so **a control added
  without its scenarios cannot land**, and a line no longer owed fails it too. The
  counts are published in `docs/known-limitations.md`. Today: 178 paths, 5 fully
  proven, 458 obligations, 445 outstanding. A matrix of seven hundred
  template-generated cases would be green and would prove nothing.

- **A degradation suite with one guarantee: never a `pass`.** A refused endpoint, a
  refused child join, a partial response, an unavailable service, an unreadable
  response, a Terraform attribute still unknown at plan time — each produced for
  real against a live server or a real plan, and checked on **every** control that
  reads the affected type rather than on a chosen witness.

- **A Terraform finding carries its origin: file, line, module.** `--format json`
  gains `labels.tf_file`, `labels.tf_line` and `labels.tf_module`; the SARIF result
  gains a `physicalLocation` with a `region`, which is what makes a forge annotate the
  guilty `resource` block instead of the plan file. The module is read off the resource
  address; the file and the line are **measured** in the `.tf` sources beside the plan,
  because `terraform show -json` carries neither — verified in Terraform's own source,
  where a resource's configuration representation holds `address`, `type`, `name` and
  `expressions` and nothing about the document. When the sources are absent, the module
  is remote, or the same block header appears twice, the origin is simply absent: a
  wrong line sends someone to fix the wrong place, and it is believed. On a live
  collection the notion does not exist and no label is set.

- **Minimum permissions are declared in the descriptor, not only in prose.** Each
  provider descriptor now carries a `permissions:` block, one entry per collection
  unit: the grant in the provider's native vocabulary, the official source that
  states it, and whether it is **confirmed** or still **to verify**. The provider
  pages render that block, so the table a reader follows and the grant the scan names
  in a `not-evaluated` reason cannot diverge. Four gates refuse a silent omission: a
  collection unit with no declared grant, an orphan entry, a missing state or source,
  and an unverified grant with no written reservation. Nothing here is confirmed by a
  scan run with a deliberately reduced role — this repository holds no cloud
  credentials — and every page says so.

- **Collection completeness is recorded, and it moves the verdict.** Every collector
  — the declarative engine, the object-storage collector, the inline EIM policy chain
  and the managed-Kubernetes one — now records, unit by unit, whether it read
  everything the API had to return. A refused endpoint no longer stops the scan and no
  longer disappears into a warning: the inventory carries a `collection` block
  (`attempted`, `complete`, a stable `error` class, the provider's own `detail`), it is
  sealed in the evidence bundle, and it is published in `--format json`. **Every
  control that reads a resource type fed by an incomplete unit becomes
  `not-evaluated`** with that unit named as its reason — the assessment decides, never
  a rule — and the scan returns **`3`, never `0`**. The transition is strictly
  directional: a `pass` is withdrawn, a `fail` is kept (an observed deviation stays
  observed), a `not-applicable` is kept (it comes from the contract, not from the
  collection). *A verdict can now move on an unchanged tenant: a scan whose credentials
  cannot read part of the scope used to return `0`, and returns `3` from now on.*

- **A capability report, printed before any verdict.** A live scan announces what it
  could and could not observe, unit by unit, with the class of each failure and the
  number of controls it costs — the count coming from the same function that degrades
  the assessment, so the report cannot promise what the report does not hold. Outside
  live scans it appears only when there is something to say. Resource types a Terraform
  plan carries and no spec projects are listed too: they are not incompleteness — no
  control reads them, so no verdict depends on them, and they do not gate — but they
  are no longer silent.

- **`exempted`: a fifth, first-class assessment status, and dated exemptions.**
  `scan --exceptions <file.yaml>` reads a versioned exemption policy — `control`,
  `justification`, `expires_at`, `owner`, `approved_by`, all five mandatory and all
  validated at load time. A `fail` covered by a valid entry becomes `exempted`, never
  `pass`: the finding stays in `--format json`, in the SARIF and in the severity
  counts, `summary.conforme` stays false, and the verdict reads `NON-COMPLIANT under
  waiver`. **A new exit code, `4`**, means "every remaining critical/high deviation is
  covered by a dated, attributed exemption" — non-zero, so nothing passes in silence,
  and distinct, so a pipeline that accepts it has to write the number down. An expired
  exemption stops applying and says so; one naming a control or subject that does not
  exist is reported as an orphan; both fail a `--strict` gate. The bundle seals
  `exemptions.json`, so the dossier digest depends on what it set aside, and
  `verify --re-derive` replays the sealed policy at the sealed instant.
  *A status and an exit code are both surfaces a pipeline must know how to read.*

- **Every attribute of the normalized inventory carries its provenance.** Beside
  `attributes`, a parallel `provenance` index says, for each attribute, where the
  value came from — `api` with the request **actually served**, `terraform-plan` with
  the plan resource type, or `derived` for a descriptor literal or a locally computed
  value — and whether the source really carried the field. It is a parallel index and
  not a wrapper around each value: the 59 Rego rules read `attributes.<name>`
  unchanged, so no verdict can move (measured on nine fixtures × two formats × two
  languages: identical findings, statuses and exit codes). `--format assessment` now
  exposes, for every control with a deciding attribute, that attribute and its
  attestation in `evidence.attribute` / `evidence.source`. This makes visible, without
  changing it, that two controls cross their attribute gate thanks to a descriptor
  constant rather than a measurement.

- **The normalized inventory is a versioned internal contract.**
  `pepin-inventory/v1`, frozen in `cmd/testdata/frozen/inventory.json` with its
  envelope, its resource shape and the full vocabulary of resource types and common
  attributes derived from the descriptors and collectors. The version travels with
  every evidence bundle (`manifest.inventory_schema`), and a new reference page states
  what is guaranteed and what is not. **Bundle format `/v2`** (manifest carries the
  inventory schema and the exemption summary), **CLI surface v3** (`--exceptions`,
  exit code 4).


- **Wave 3 of the documentation: the control catalogue is generated, and the
  project explains itself.** One generated page per control under `docs/controls/`
  (what it concludes, from which source, with which reason when it cannot), a
  remediation guide that shows the same control moving from `fail` to `pass` on the
  repository's own example plans, an architecture page that argues the central choice
  — one common rule set, the source is what changes per cloud — and two contribution
  guides, adding a control and adding a provider, each ending in a checklist usable as
  is. A public `ROADMAP.md` replaces the internal working document that used to sit in
  the product documentation.

- **Exoscale is the first provider with a complete set of deployable remediation
  proofs**: 26 of 26, bringing the repository from 4 to 26 out of 95. Twenty
  self-contained Terraform modules, checked with `terraform init -backend=false` and
  `terraform validate` against the real provider schema, plus two documented notes
  where Terraform cannot express the fix (provider sovereignty, account MFA).
  `TestExoscaleRemediationCoverageStaysComplete` now fails the build when an exoscale
  control lands without its proof; the other providers stay outside that guard until
  they reach 100 %.

- **Wave 2 of the product documentation: ten pages, generated where they can be.**
  A CLI reference built from the frozen surface and from real `--help` runs, the
  exit-code contract shown as six executions with the code each one returned, the
  five output formats with a real document each, plan-versus-live with two
  reproducible divergences, the evidence-bundle lifecycle (seal, verify,
  re-derive, tamper, redact) captured end to end, GitHub Actions and GitLab CI
  integrations whose complete pipelines are injected from `examples/`, and one
  page per sovereign cloud with its API calls and its minimal read-only
  permissions. `TestEveryPublicCLIFlagIsDocumented` now fails when a public flag
  is missing from the CLI reference, in either language.

- The published CI examples pin **v0.2.0** and every action by commit SHA. The
  action of v0.1.0 and v0.1.1 could not install anything (`gh attestation verify`
  without a token), so those tags should not be pinned by anyone.

- **Controls become configurable, and a relaxed setting cannot keep its badge.** Four
  controls now read a policy file — the mandatory tagging profile, the snapshot
  freshness window and accepted states, the secret-detection threshold. Every setting
  is a handle that can manufacture green, so each normative mapping in the reference
  carries the **constraints under which it holds** (`config_requise`, with four
  interpretable senses: `au_plus_le_defaut`, `superset_du_defaut`,
  `sous_ensemble_du_defaut`, `au_moins_aussi_strict_que_le_defaut`). A configuration that falls outside a constraint makes the
  control **lose its `references`** in the assessment — it stops claiming CIS,
  ISO or SecNumCloud — and the relaxation appears in five places at once: the terminal
  (`RELAXED CONFIGURATION`), the assessment labels and evidence, `--format json`
  (`config.relaxations`), the verdict banner, and the sealed bundle (`config.json`
  plus a `config` entry in the manifest, both covered by `checksums.txt`). Tightening a
  setting is not a relaxation and is reported nowhere. `mise run validate` refuses a
  constraint naming a setting the policy engine cannot evaluate. See
  `docs/guides/control-configuration.md`.

- **One policy file: `scan --policy`.** It carries `controls:` (the settings) and
  `exceptions:` (the exemptions, unchanged format). `--exceptions` remains as the
  historic name of the same file and reads the same schema, so an existing invocation
  and an existing file keep working; the two flags are **mutually exclusive**, because
  two policy files are two files that will drift. CLI surface v4.

- **Secret detection carries a confidence level.** Every finding of
  `compute_instance_no_secrets_in_user_data` publishes `labels.confidence`: `high` for a
  PEM private-key block, `medium` for a recognized prefix in the expected format
  (`ghp_`, `AKIA`, `SCW`, `EXO`, `glpat-`, JWT), `low` for a generic heuristic
  (`password=…`, `api_key=…`). The default reporting threshold is `low` — everything is
  reported, exactly as before. The detected value still never appears, at any level, and
  that property is now tested at all three levels, on the message and the remediation,
  in both languages.

- **The evaluated inventory carries its configuration.** The envelope gains `config`,
  the effective control configuration, next to `evaluated_at` — so a sealed bundle's
  `input.json` replays under the settings of its own day, and `verify --re-derive` stays
  faithful without being handed the policy file. `--format json` publishes
  `config.policy_digest` and `config.effective` on **every** scan, default included: a
  reader must be able to check that a scan ran under the expected settings, not merely
  observe that it said nothing. Bundle format v3.

### Changed

- **`network_documented` now checks what it announces.** The rule promised owner, project
  and environment, and evaluated `count(tags) > 0`: a single `foo=bar` was enough to
  declare a network documented — a compliance asserted without being measured. It now
  requires the tags that actually document (default `Owner, Project, Env`, configurable),
  and it stays silent when the `tags` attribute was not collected, where it used to report
  a deviation. The **code is unchanged**: it travels in SARIF `ruleId`s, archived
  assessments and exemption files, where a rename would turn a valid exemption into an
  orphan overnight and bring back the deviation it covered. Title and description are
  rewritten in both languages.

- **The mandatory tagging policy is configurable, and the comparison is
  convention-agnostic.** `governance_resource_required_tags` no longer demands four frozen
  literals. The comparison ignores case and separators (`cost-center` = `CostCenter`), and
  aliases widen each logical name (`team` for `Owner`, `environment` for `Env`), so an
  organisation writing `cost-center, application, environment, team` is no longer reported
  as ungoverned. The targeted resource types are explicit and justified, and four billable
  types join the scope — `blockstorage_snapshot`, `compute_image`, `managed_database`,
  `kubernetes_cluster` — which closes a false-negative on paid services that were outside
  it. The shipped profile is documented as a **recommendation, not a standard**.

- **The snapshot freshness control says what it measures, and what it does not prove.**
  `blockstorage_volume_snapshots_exist` now checks the snapshot's **native state** as well
  as its date: a snapshot in `error`, `pending` or `creating` no longer counts as a backup.
  The window is configurable (7 days by default). The title becomes "No recent, completed
  snapshot", and the description states plainly what the control does not prove —
  restorability, application completeness, retention, the existence of a backup policy.
  The code is unchanged, for the same reason as above. Anchored on Outscale
  `Snapshot.State` and Exoscale `block-storage-snapshot.state`, both projected by the
  collectors from this version on. The normalized inventory therefore gains an attribute,
  which is a contract change: inventory schema v4, whose note also records the `config`
  envelope key added above.

- **`--strict` also refuses a dropped normative mapping.** It already refused zero
  coverage, remaining medium/low deviations and a stale exemptions file; it now returns
  `3` when a relaxed setting cost a control its mapping. No new exit code: incompleteness
  and relaxation say the same thing — do not read this scan as a green light — and both
  already sit where `3` sits.

## [0.2.0] - 2026-08-19

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

- **The project has a mark.** `docs/assets/brand/` holds the icon and the
  lockups in SVG and PNG, light, dark and monochrome, with the generators that
  produce them (`scripts/generer-marque.py`, `scripts/generer-png-marque.py`)
  and the usage rules in `docs/brand.md`. Both READMEs open on it.

### Changed

- **Inventory schema `pepin-inventory/v3`.** A resource gains `source` (`file`, `line`,
  `module`), present only where it could be measured. Pure addition.

- **Exit code `3` widens from "nothing measured" to "the scan does not establish
  compliance".** It now also fires when the collection could not read part of the
  intended scope. Deliberately no fifth code: an incompleteness code could never take
  precedence over `1` — hiding a real critical deviation because the rest was missing
  would be the false green this wave exists to prevent — so it would only ever fire
  where `3` already fires. What distinguishes the situations stays readable in the
  capability report, in each control's reason and in the `collection` key.
  *An exit code is a surface every pipeline parses.*

- **Inventory schema `pepin-inventory/v2`.** The envelope gains `collection`. Pure
  addition — no existing field moves — but a consumer that replays an inventory without
  reading `collection` would conclude more firmly than Pépin did, which is exactly what
  the field exists to prevent. The version travels in `manifest.inventory_schema`.

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


- **`docs/doc-cache-brief.md` moves out of the product documentation.** It was a
  maintainer's memo addressing a machine ("already downloaded on this machine"),
  describing a cache a clone cannot have, linked from nowhere, and carrying six
  absolute paths into a home directory. What it held that was worth keeping
  moves to `references/docs/README.md`, beside the `sources.yaml` it describes --
  including the trap that matters most: the documentation is not the contract.

### Fixed

- **The published action installs again.** The provenance check added in 0.1.0
  called `gh attestation verify` without a token; `gh` refuses to run in a
  workflow without `GH_TOKEN`, so the installer treated every binary as
  unverifiable and refused it -- for every consumer, on both 0.1.0 and 0.1.1.
  The action now supplies `github.token` itself: nobody should have to wire up a
  token to install a binary.

  The gap that let it ship is worth naming. The pull-request job served
  `install.sh` over a loopback with the attestation check skipped, so the public
  path was never exercised until the post-publication job ran -- after the tag
  existed. A job now calls the action against an already-published version on
  every pull request.

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

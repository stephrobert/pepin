> 🇬🇧 English · [🇫🇷 Français](cli.fr.md)

# CLI reference

Pépin's command line is a **public interface**. Its verbs, its flags and its exit codes are a
promise that pipelines are built on, and they are frozen by a test rather than by a sentence:
`cmd/testdata/frozen/cli.json` holds the current shape, `cmd/frozen_test.go` fails when the
code moves away from it, and a second test fails when the fixture moves without its version
number being raised.

This page is generated from that same fixture and from the binary itself. The flag tables come
from the frozen surface; every help text below is the output of `pepin <verb> --help`, captured
by running the binary. A public flag that is missing here fails
`TestEveryPublicCLIFlagIsDocumented` in `cmd/`.

## The frozen surface

<!-- pepin:gen cli-verbs -->
| Command | Flags |
|---|---|
| `pepin provider` | _(no flag of its own)_ |
| `pepin provider list` | _(no flag of its own)_ |
| `pepin provider new` | _(no flag of its own)_ |
| `pepin provider validate` | _(no flag of its own)_ |
| `pepin scan` | `--exceptions`, `--format` / `-f`, `--kubeconfig`, `--lang`, `--live`, `--policy-dir` / `-p`, `--profile`, `--redact`, `--region`, `--s3-endpoint`, `--seal`, `--strict`, `--terraform` / `-t` |
| `pepin scsl` | `--index` |
| `pepin verify` | `--bundle`, `--pubkey`, `--re-derive` |
| `pepin version` | _(no flag of its own)_ |
<!-- /pepin:gen cli-verbs -->

`pepin provider` also answers to `pepin providers`, and `pepin provider list` to
`pepin provider ls`. Aliases are a convenience; the names above are the promise.

Four shapes are versioned independently, because a consumer's pipeline parses them
independently:

<!-- pepin:gen surface-versions -->
| Surface | What is frozen | Version |
|---|---|:-:|
| `cli` | verbs, flags and exit codes | **v3** |
| `findings` | shape of `--format json` (`findings` + `summary`) | **v1** |
| `assessment` | shape of the `--format assessment` document | **v1** |
| `bundle` | shape of the evidence bundle (files, roles, manifest) | **v2** |
| `inventory` | shape of the normalized inventory (envelope, resource, types and attributes) | **v1** |
<!-- /pepin:gen surface-versions -->

A version rises on **any** shape change, additions included: the number means "the surface has
moved", not "the surface has broken". The procedure for a deliberate change (regenerate the
fixture, raise the constant, write the CHANGELOG line) is in
[RELEASING.md](../../RELEASING.md).

## `--lang`, the persistent flag

Pépin is bilingual. The language is resolved once, before any help text is built, in this
order — the first non-empty source wins:

`--lang=fr|en` → `PEPIN_LANG` → `LC_ALL` → `LANG` → fallback `en`

An unknown locale falls back to English without an error. The flag is **persistent**: it
applies to the root command and to every subcommand.

What the language changes, and what it does not:

| Stable across languages | Translated |
|---|---|
| control codes (`CLD-*`), check identifiers, severities, statuses, subjects, exit codes | titles, messages, remediations, evidence, help texts, verdict wording |

**A pipeline that compares report *text* between two runs must pin `PEPIN_LANG`.** Otherwise a
runner whose `LANG` changes will produce a diff that no configuration change explains. A
pipeline keyed on codes and statuses is unaffected. The same caveat applies to a sealed
bundle's digest: see [Evidence bundles](../guides/evidence-bundles.md#the-digest-depends-on-the-language).

<!-- pepin:gen cli-help-root -->
```text
Pepin — sovereign multi-cloud CSPM.

Assesses the posture of a cloud (OVH, Scaleway, Exoscale, Outscale…) against a
common reference anchored on SCSL, SecNumCloud, CIS and ISO.

Usage:
  pepin [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  provider    Manage the declarative providers (list, validate, create)
  scan        Assess the posture of a cloud against the policies
  scsl        Check consistency with the SCSL index and drive the roadmap
  verify      Verify the integrity (and the signature) of an evidence bundle
  version     Print the version

Flags:
  -h, --help          help for pepin
      --lang string   interface language: fr | en (default: PEPIN_LANG, then LC_ALL/LANG, otherwise en)

Use "pepin [command] --help" for more information about a command.
```
<!-- /pepin:gen cli-help-root -->

## `pepin scan`

The verb everything else revolves around: it reads one source, evaluates the common rules
against the normalized model, and renders a report.

<!-- pepin:gen cli-help-scan -->
```text
Assesses an inventory against the embedded common rules (plus external rules
through --policy-dir). Three sources: a normalized JSON export, a Terraform
plan (--terraform), or a live collection from the provider API (--live).

Usage:
  pepin scan <provider> [export.json] [flags]

Flags:
      --exceptions file                  exemptions YAML file (control, justification, expires_at, owner, approved_by) — a covered deviation becomes exempted, never compliant
  -f, --format string                    output format: table | json | assessment | oscal | sarif (default "table")
  -h, --help                             help for scan
      --kubeconfig string                path to a kubeconfig to audit the state INSIDE a Kubernetes cluster (use READ-ONLY, short-lived access — never cluster-admin)
      --live                             collect the inventory live through the provider API (credentials required)
  -p, --policy-dir stringArray           directory of external rules (.rego), repeatable — loaded without recompiling
      --profile string                   credentials profile for the live collection (e.g. ~/.osc/config.json)
      --redact                           redact the sensitive values (user-data, policies) from the bundle's input.json — for sharing with a third party; INCOMPATIBLE with verify --re-derive
      --region string                    target region for the live collection
      --s3-endpoint string               custom S3 endpoint for object storage (live collection; e.g. MinIO http://localhost:9000)
      --seal string                      write a defensible evidence bundle (assessment + OSCAL + manifest + checksums) into this directory
      --strict                           strict CI gate: non-zero exit code if no control is measured (governance aside) or if medium/low deviations remain
  -t, --terraform terraform show -json   audit a Terraform plan (terraform show -json) instead of an inventory export

Global Flags:
      --lang string   interface language: fr | en (default: PEPIN_LANG, then LC_ALL/LANG, otherwise en)
```
<!-- /pepin:gen cli-help-scan -->

### One source, and exactly one

| Source | How | Credentials |
|---|---|---|
| Normalized inventory (JSON export) | `pepin scan <provider> inventory.json` | none |
| Terraform plan | `pepin scan <provider> --terraform plan.json` | none |
| Live provider API | `pepin scan <provider> --live` | yes, in the provider's own environment variables |

Two traps worth naming, both of which the CLI refuses rather than guesses:

- **`--terraform` is a boolean switch, not a path.** The plan file is the positional argument.
  `--terraform=plan.json` fails with a parsing error (`strconv.ParseBool`), it does not read
  the file.
- **`--live` and `--terraform` are mutually exclusive.** Setting both is refused by the flag
  group, not silently resolved in favour of one of them.

Choosing between a plan and a live scan is a real decision, and both are legitimate:
[Terraform plan vs live scan](../concepts/terraform-vs-live.md).

### Flag by flag

| Flag | Default | What it does |
|---|---|---|
| `--format`, `-f` | `table` | output format: `table`, `json`, `assessment`, `oscal`, `sarif` — see [Output formats](output-formats.md) |
| `--terraform`, `-t` | `false` | read the positional file as a Terraform plan (`terraform show -json`) |
| `--live` | `false` | collect the inventory from the provider API |
| `--region` | provider default | target region (or zone) for the live collection |
| `--profile` | — | credentials profile for the live collection (e.g. `~/.osc/config.json`) |
| `--s3-endpoint` | provider default | custom S3 endpoint for object storage (live collection) |
| `--kubeconfig` | — | audit the state **inside** a Kubernetes cluster; use read-only, short-lived access, never `cluster-admin` |
| `--policy-dir`, `-p` | — | directory of extra `.rego` rules, repeatable, loaded without recompiling |
| `--seal` | — | write an evidence bundle into this directory — see [Evidence bundles](../guides/evidence-bundles.md) |
| `--redact` | `false` | redact sensitive values from the bundle's `input.json`; **incompatible with `verify --re-derive`** |
| `--strict` | `false` | stricter CI gate: non-zero when medium/low deviations remain |
| `--lang` | detected | interface language, `fr` or `en` |

`--strict` does **not** control the "nothing was measured" gate: a scan that concluded nothing
already exits 3 without it. See [Exit codes](exit-codes.md).

## `pepin verify`

Third-party verification of a bundle produced by `scan --seal`.

<!-- pepin:gen cli-help-verify -->
```text
Recomputes the SHA-256 digest of every file listed in checksums.txt and reports any
tampering. This is the third-party verification of a bundle produced by `scan --seal`.

Without --pubkey, only integrity (accidental alteration) is verified: an attacker can
regenerate both files and checksums. With --pubkey, the cosign SIGNATURE of checksums.txt
is verified (non-repudiation) — the operator having sealed the bundle with (cosign 3.x):
  cosign sign-blob --key cosign.key --bundle checksums.txt.bundle checksums.txt

Usage:
  pepin verify <dossier-bundle> [flags]

Flags:
      --bundle string   cosign signature bundle (default: <directory>/checksums.txt.bundle)
  -h, --help            help for verify
      --pubkey string   cosign public key used to verify the signature of checksums.txt
      --re-derive       replay the rules on input.json and check that the sealed assessment follows from it (strong defensibility)

Global Flags:
      --lang string   interface language: fr | en (default: PEPIN_LANG, then LC_ALL/LANG, otherwise en)
```
<!-- /pepin:gen cli-help-verify -->

Three levels of assurance, and they are not interchangeable:

| Command | What it establishes |
|---|---|
| `pepin verify bundle` | internal consistency — accidental alteration only. Whoever can rewrite a file can rewrite `checksums.txt` too. |
| `pepin verify bundle --pubkey cosign.pub` | non-repudiation: the cosign signature over `checksums.txt` is valid. Requires the `cosign` binary in `PATH`. |
| `pepin verify bundle --re-derive` | defensibility: the rules are replayed on `input.json` and the sealed assessment must follow from it. |

## `pepin provider`

<!-- pepin:gen cli-help-provider -->
```text
Manage the declarative providers (list, validate, create)

Usage:
  pepin provider [flags]
  pepin provider [command]

Aliases:
  provider, providers

Available Commands:
  list        List the available cloud providers
  new         Create the skeleton of a provider (providers/<name>.yaml)
  validate    Validate the providers of a directory (default: providers/) against the contract

Flags:
  -h, --help   help for provider

Global Flags:
      --lang string   interface language: fr | en (default: PEPIN_LANG, then LC_ALL/LANG, otherwise en)

Use "pepin provider [command] --help" for more information about a command.
```
<!-- /pepin:gen cli-help-provider -->

### `pepin provider list`

<!-- pepin:gen cli-help-provider-list -->
```text
List the available cloud providers

Usage:
  pepin provider list [flags]

Aliases:
  list, ls

Flags:
  -h, --help   help for list

Global Flags:
      --lang string   interface language: fr | en (default: PEPIN_LANG, then LC_ALL/LANG, otherwise en)
```
<!-- /pepin:gen cli-help-provider-list -->

<!-- pepin:gen provider-list -->
```text

// pepin  registered providers
  exoscale  Exoscale (CH) — instances, security groups, block storage, SKS, SOS
  kubernetes  Kubernetes (in-cluster) — RBAC, Pod Security Standards, NetworkPolicy
  outscale  Outscale (3DS) — VM, BSU, OOS, EIM, security groups, OKS, LBU
  scaleway  Scaleway — object storage, instances, IAM, security groups
```
<!-- /pepin:gen provider-list -->

### `pepin provider validate`

Checks the descriptors of a directory (default `providers/`) against the contract they must
honour. Exits 1 when a descriptor is invalid, which makes it usable as a CI gate on a
contribution that adds a provider.

<!-- pepin:gen cli-help-provider-validate -->
```text
Validate the providers of a directory (default: providers/) against the contract

Usage:
  pepin provider validate [dossier] [flags]

Flags:
  -h, --help   help for validate

Global Flags:
      --lang string   interface language: fr | en (default: PEPIN_LANG, then LC_ALL/LANG, otherwise en)
```
<!-- /pepin:gen cli-help-provider-validate -->

### `pepin provider new`

<!-- pepin:gen cli-help-provider-new -->
```text
Create the skeleton of a provider (providers/<name>.yaml)

Usage:
  pepin provider new <nom> [flags]

Flags:
  -h, --help   help for new

Global Flags:
      --lang string   interface language: fr | en (default: PEPIN_LANG, then LC_ALL/LANG, otherwise en)
```
<!-- /pepin:gen cli-help-provider-new -->

## `pepin scsl`

Consistency report against the SCSL index and the roadmap it drives. `--index` points at the
framework's static API (`api/v1/exigences.json`); the index is frozen, so this verb reports,
it never creates a requirement.

<!-- pepin:gen cli-help-scsl -->
```text
Check consistency with the SCSL index and drive the roadmap

Usage:
  pepin scsl [flags]

Flags:
  -h, --help           help for scsl
      --index string   path to the SCSL API (the framework's api/v1/exigences.json) (default "../framework-scsl/api/v1/exigences.json")

Global Flags:
      --lang string   interface language: fr | en (default: PEPIN_LANG, then LC_ALL/LANG, otherwise en)
```
<!-- /pepin:gen cli-help-scsl -->

## `pepin version`

<!-- pepin:gen cli-help-version -->
```text
Print the version

Usage:
  pepin version [flags]

Flags:
  -h, --help   help for version

Global Flags:
      --lang string   interface language: fr | en (default: PEPIN_LANG, then LC_ALL/LANG, otherwise en)
```
<!-- /pepin:gen cli-help-version -->

The tool name is written `pepin` in English and `pépin` in French: a script that parses this
output should pin `PEPIN_LANG`, or split on the space.

## Exit codes

<!-- pepin:gen cli-exit-codes -->
| Code | Constant (`cmd/surface.go`) | Meaning |
|:-:|---|---|
| **0** | `conforme` | no critical/high deviation, and at least one control actually measured |
| **1** | `non_conformite` | at least one critical or high deviation |
| **2** | `erreur` | technical error: the scan could not conclude |
| **3** | `strict` | nothing was measured (without `--strict`), or medium/low deviations remain with `--strict` |
| **4** | `derogation` | every remaining critical/high deviation is covered by a dated, attributed exemption (`--exceptions`) |
<!-- /pepin:gen cli-exit-codes -->

The full semantics, one scenario at a time, with the commands that produce each code:
[Exit codes and CI gating](exit-codes.md).

## Where to go next

- [Exit codes](exit-codes.md) — the integration contract of a CI gate.
- [Output formats](output-formats.md) — which format to parse, and what is guaranteed.
- [Evidence bundles](../guides/evidence-bundles.md) — `--seal`, `--redact`, `verify`.
- [Coverage matrix](../coverage.md) — what is measurable, per provider and per source.

## How this page stays true

The flag tables are read from the frozen CLI surface, and the help blocks are the standard
output of `pepin <verb> --help`, captured by `internal/docgen` running the binary.
`mise run gen-docs` rewrites them; `TestGeneratedDocsAreUpToDate` fails when what is committed
differs from what the binary prints today, and `TestEveryPublicCLIFlagIsDocumented` fails when
a public flag never appears on this page.

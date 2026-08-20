> 🇬🇧 English · [🇫🇷 Français](evidence-bundles.fr.md)

# Evidence bundles

`pepin scan --seal <dir>` writes an **evidence bundle**: the evaluated inventory, the typed
assessment, its OSCAL rendering, a manifest and a checksum list. It is what you hand to an
auditor, attach to a change ticket, or keep to explain, six months later, what the tenant
looked like on the day of the scan.

## Read this before you share one

**Without `--redact`, a bundle embeds the RAW inventory.** `input.json` is the exact object the
rules were evaluated against: cloud-init user data, IAM policy documents, bucket policies,
access key identifiers, connection strings. A tool that finds a cleartext password in user data
necessarily *has* that password, and sealing it without redaction puts it in the bundle.

Pépin says so on every seal, on standard error:

<!-- pepin:gen bundle-seal -->
```console
$ ./pepin scan scaleway --terraform examples/scaleway/terraform/plan.json --seal bundle
pepin: ⚠ input.json embeds the RAW inventory (it may contain secrets: user-data, policies). Treat the bundle as SENSITIVE, or use --redact to share it.
pepin: evidence bundle written to bundle — seal it with: cosign sign-blob bundle/checksums.txt
$ echo $?
1
```
<!-- /pepin:gen bundle-seal -->

(The exit code is `1` because the scanned plan is non-compliant, not because sealing failed:
`--seal` never changes the verdict. See [Exit codes](../reference/exit-codes.md).)

Treat an unredacted bundle as **the same secret as the tenant it describes**: private artifact
storage, short retention, no public CI page, no attachment to a ticket a supplier can read.
[`--redact`](#--redact-a-bundle-you-hand-to-a-third-party) exists for the bundles that leave the
perimeter, and it has a cost this page states plainly.

## What a bundle contains

<!-- pepin:gen bundle-files -->
| File | Role declared in the manifest |
|---|---|
| `input.json` | `evaluated-input` |
| `assessment.json` | `assessment` |
| `assessment-oscal.json` | `oscal-assessment-results` |
| `manifest.json` | `manifest` |
| `checksums.txt` | `checksums` |
<!-- /pepin:gen bundle-files -->

The shape of this set (which files, which roles, which manifest fields) is a **frozen surface**:
`cmd/testdata/frozen/bundle.json`. A third-party verifier can rely on it, and a change to it
raises a version number and gets a CHANGELOG line.

### `manifest.json`

<!-- pepin:gen bundle-manifest -->
```json
{
  "format": "pepin-assessment-bundle/v2",
  "inventory_schema": "pepin-inventory/v3",
  "disclaimer": "This report assesses the configuration of a tenant (customer-side scope). The normative mappings (SecNumCloud, ISO, CIS) are indicative: they are not a proof of qualification or certification, which applies to the cloud service provider.",
  "generated": "<timestamp>",
  "tool": {
    "name": "pepin",
    "version": "<version>",
    "digest": "<provenance>"
  },
  "ruleset": {
    "name": "pepin-config",
    "digest": "sha256:<sha256>"
  },
  "target": {
    "id": "scaleway",
    "provider": "scaleway",
    "platform": "scaleway"
  },
  "source": "terraform-plan",
  "summary": {
    "fail": 10,
    "not-applicable": 2,
    "not-evaluated": 9,
    "pass": 6
  },
  "artifacts": [
    {
      "file": "input.json",
      "role": "evaluated-input",
      "sha256": "<sha256>",
      "bytes": "<bytes>"
    },
    {
      "file": "assessment.json",
      "role": "assessment",
      "sha256": "<sha256>",
      "bytes": "<bytes>"
    },
    {
      "file": "assessment-oscal.json",
      "role": "oscal-assessment-results",
      "sha256": "<sha256>",
      "bytes": "<bytes>"
    }
  ]
}
```
<!-- /pepin:gen bundle-manifest -->

The markers `<timestamp>`, `<sha256>`, `<version>`, `<provenance>` and `<bytes>` stand for
values that change at every run; a real manifest carries them in full. Three fields deserve
attention:

- **`tool.digest`** identifies the binary that produced the bundle (`vcs:<commit>`, with a
  `+modified` suffix when the working tree was dirty).
- **`ruleset.digest`** covers the rules, the provider descriptors and the reference together. If
  it differs from yours when you re-derive, you are replaying with a different rule set, and
  `pepin verify` says so rather than reporting a divergence.
- **`source`** records where the data came from — `terraform-plan`, `live` or `export`. Two
  bundles of the same tenant from two different sources are not comparable line by line:
  [Terraform plan vs live scan](../concepts/terraform-vs-live.md).

### `checksums.txt`

<!-- pepin:gen bundle-checksums -->
```text
<sha256>  input.json
<sha256>  assessment.json
<sha256>  assessment-oscal.json
<sha256>  manifest.json
```
<!-- /pepin:gen bundle-checksums -->

Standard `sha256sum` format, so it can be checked without Pépin at all
(`sha256sum -c checksums.txt`). This is the file a cosign signature covers: signing one file
that names the digests of the others is what makes a single signature cover the whole bundle.

### `exemptions.json` — only when a waiver was applied

A dossier that does not say what it set aside is not defensible. When a scan is given
`--exceptions`, the bundle carries a sixth artifact recording the policy as loaded and what
each entry actually produced (`applied`, `expired`, `orphan`).

It is an artifact like any other: its digest is in `checksums.txt`, so it is covered by the
signature, and the bundle digest therefore **depends on the exemptions**. A dossier cannot
drop them without failing its own verification. The manifest carries the summary — how many
applied, expired, orphan, and the digest of the policy itself — so a verifier sees it before
opening anything else.

`verify --re-derive` replays the sealed policy at the sealed instant of evaluation. Without
that, a perfectly faithful bundle would be declared falsified for the sole reason that the
verifier does not hold the operator's exemptions file, and a valid waiver would appear to
"expire" between the scan and the verification.

## Verifying a bundle

Three levels, and they establish different things. Do not confuse them.

### Level 1 — integrity, accidental only

<!-- pepin:gen bundle-verify -->
```console
$ ./pepin verify bundle
⚠ bundle internally consistent: bundle
  (ACCIDENTAL integrity only — NOT defensible without --pubkey for the cosign signature)
$ echo $?
0
```
<!-- /pepin:gen bundle-verify -->

Note the `⚠` and the absence of a reassuring green check. Recomputing digests proves only that
the files agree with `checksums.txt`, and whoever can rewrite a file can rewrite that list too.
This level catches a truncated download, not an adversary.

Here is a bundle whose sealed assessment was edited — one `fail` flipped to `pass`, the single
most tempting falsification:

<!-- pepin:gen bundle-tampered -->
```console
$ ./pepin verify bundle-tampered
error: invalid digest for assessment.json (tampered file)
$ echo $?
2
```
<!-- /pepin:gen bundle-tampered -->

Exit code 2, and the file is named. A verification that could not do this would be decoration.

### Level 2 — signature, for non-repudiation

`--pubkey` verifies the cosign signature over `checksums.txt`, which transitively covers every
file it lists. It shells out to the `cosign` binary, which must be in `PATH`.

```bash
# Once, on the sealing side (cosign 3.x):
cosign sign-blob --key cosign.key --bundle bundle/checksums.txt.bundle bundle/checksums.txt

# On the verifying side:
pepin verify bundle --pubkey cosign.pub
```

Without `--bundle`, Pépin looks for `<dir>/checksums.txt.bundle`. When that file is missing, it
refuses with exit code 2 and prints the exact `sign-blob` command to run — it does not fall
back to unsigned verification.

> **This page shows no captured output for a signed verification.** Signing requires a private
> key and a Sigstore configuration that the documentation generator does not own, and every
> other block on this page is a real execution. The commands above are the ones
> `pepin verify --help` documents ([CLI reference](../reference/cli.md#pepin-verify)).

### Level 3 — re-derivation, the defensible one

A signature attests bytes. It does not attest that the verdict follows from the input: a
perfectly signed bundle can carry an assessment that its own `input.json` does not support.
`--re-derive` replays the rules on `input.json` and compares the result with the sealed
assessment.

<!-- pepin:gen bundle-rederive -->
```console
$ ./pepin verify bundle --re-derive
⚠ bundle internally consistent: bundle
  (ACCIDENTAL integrity only — NOT defensible without --pubkey for the cosign signature)
✓ FAITHFUL re-derivation: the sealed assessment does follow from input.json
$ echo $?
0
```
<!-- /pepin:gen bundle-rederive -->

It also re-renders the OSCAL from the re-derived assessment and compares it, so that
`assessment-oscal.json` — the artifact a GRC tool ingests — cannot be falsified on its own. And
it checks provenance consistency: the scan carves a single instant into both
`input.evaluated_at` and `run.timestamp`, and a gap between them betrays backdating.

Because the rules are replayed with the **current** binary, re-deriving with a different rule
set prints a note naming both digests instead of pretending the bundle is faithful.

## The digest depends on the language

The assessment and the OSCAL carry prose: titles, messages, remediations, evidence. Prose is
translated, so the same scan sealed in French and in English produces different bytes — and
therefore different digests — for every file that carries text:

<!-- pepin:gen bundle-cross-lang -->
| File | Same bytes in both languages? |
|---|---|
| `assessment-oscal.json` | ❌ differs (the digest changes) |
| `assessment.json` | ❌ differs (the digest changes) |
| `checksums.txt` | ❌ differs (the digest changes) |
| `input.json` | ✅ identical |
| `manifest.json` | ❌ differs (the digest changes) |
<!-- /pepin:gen bundle-cross-lang -->

`input.json` is identical, because an inventory carries no prose. Everything else differs.
(The comparison neutralises the run timestamp, which differs between any two runs whatever the
language.)

Two practical consequences.

**Pin `PEPIN_LANG` when you seal**, if you archive bundles and compare their digests over time.
A runner whose locale changes will otherwise produce a "different" bundle from an unchanged
tenant.

**Verification does not care.** `verify --re-derive` replays the rules in both languages and
accepts either match — what it compares (statuses, subjects, references, provenance) is
identical, only the wording changes. Here is this page's own English `pepin` verifying the
bundle sealed in French, re-derivation included:

<!-- pepin:gen bundle-cross-verify -->
```console
$ ./pepin verify bundle-fr --re-derive
⚠ bundle internally consistent: bundle-fr
  (ACCIDENTAL integrity only — NOT defensible without --pubkey for the cosign signature)
✓ FAITHFUL re-derivation: the sealed assessment does follow from input.json
$ echo $?
0
```
<!-- /pepin:gen bundle-cross-verify -->

A false accusation of falsification is the worst verdict a verifier can return, so this case is
shown rather than asserted.

## `--redact`: a bundle you hand to a third party

`--redact` replaces the value of every sensitive attribute in the embedded inventory with its
digest. The finding stays, the secret leaves:

<!-- pepin:gen bundle-redact -->
```json
"user_data": "[REDACTED sha256:2bb6abea90eaa2eb]"
```
<!-- /pepin:gen bundle-redact -->

The redacted attributes are the ones whose value can carry a secret: `user_data`, `document`,
`statements`, `policy`, `access_key`, `secret_key`, `password`, `token`, `ssh_key`,
`public_key`, `private_key`, `certificate`, `connection_string`.

**The cost is explicit: a redacted bundle cannot be re-derived.** The rules would replay on
redacted values and reach a different verdict, which the verifier reports as a divergence:

<!-- pepin:gen bundle-redact-rd -->
```console
$ ./pepin verify bundle-redacted --re-derive
⚠ bundle internally consistent: bundle-redacted
  (ACCIDENTAL integrity only — NOT defensible without --pubkey for the cosign signature)
error: re-derivation DIVERGES from the sealed assessment: the bundle does NOT faithfully attest input.json (fabricated result, or a different configuration)
$ echo $?
2
```
<!-- /pepin:gen bundle-redact-rd -->

That message is severe on purpose — it is the same one a fabricated bundle earns — so redaction
is a deliberate choice, not a default that quietly disables the strongest guarantee.

| | Keep it internal | Hand it to a third party |
|---|---|---|
| Flag | `--seal bundle` | `--seal bundle --redact` |
| `input.json` | raw inventory, **sensitive** | sensitive values replaced by their digest |
| `verify` | yes | yes |
| `verify --pubkey` | yes | yes |
| `verify --re-derive` | yes | **no** |
| What it rests on | re-derivation | the cosign signature |

## A complete run

```bash
# 1. Seal (live scan of the real tenant, in a pinned language)
PEPIN_LANG=fr pepin scan scaleway --live --region fr-par --seal bundle

# 2. Sign the checksum list, which covers every file it names
cosign sign-blob --key cosign.key --bundle bundle/checksums.txt.bundle bundle/checksums.txt

# 3. On the receiving side: integrity, signature, and that the verdict follows from the input
pepin verify bundle --pubkey cosign.pub --re-derive
```

In CI, keep the bundle private and short-lived — the artifact steps for both platforms are in
[GitHub Actions](github-actions.md#archive-the-evidence-bundle) and
[GitLab CI](gitlab-ci.md#archive-the-evidence-bundle).

## What a bundle does not prove

A bundle establishes what Pépin observed, when, with which rules, and that the verdict follows
from what it observed. It does not establish that the cloud provider is qualified, nor that the
tenant is compliant with a framework: normative mappings are indicative, and the scope is the
customer side. See [Scope and non-goals](../concepts/scope.md), and the disclaimer that the
manifest itself carries.

## Where to go next

- [Output formats](../reference/output-formats.md) — the assessment and OSCAL documents in detail.
- [The assessment model](../concepts/assessment-model.md) — what each status asserts.
- [CLI reference](../reference/cli.md#pepin-verify) — every `verify` flag.
- [Exit codes](../reference/exit-codes.md) — `verify` returns 2 when it refuses a bundle.

## How this page stays true

Every console block above is a real execution captured by `internal/docgen`: the bundle is
sealed into a throwaway directory, verified, re-derived, tampered with and re-verified, sealed
again with `--redact`, and sealed once more in the other language for the comparison table. The
paths shown are the relative ones a reader would type. Timestamps, digests, sizes and the build
version are marked as such, because they change at every run. `TestGeneratedDocsAreUpToDate`
fails when the committed page no longer matches what the binary does.

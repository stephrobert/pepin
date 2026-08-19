> 🇬🇧 English · [🇫🇷 Français](exit-codes.fr.md)

# Exit codes and CI gating

A CI gate reads one number. Pépin's exit codes are therefore an **integration contract**:
they are frozen in `cmd/testdata/frozen/cli.json`, tested one by one, and changing their
meaning is a breaking change with its own CHANGELOG line.

<!-- pepin:gen cli-exit-codes -->
| Code | Constant (`cmd/surface.go`) | Meaning |
|:-:|---|---|
| **0** | `conforme` | no critical/high deviation, and at least one control actually measured |
| **1** | `non_conformite` | at least one critical or high deviation |
| **2** | `erreur` | technical error: the scan could not conclude |
| **3** | `strict` | nothing was measured (without `--strict`), or medium/low deviations remain with `--strict` |
<!-- /pepin:gen cli-exit-codes -->

The distinction the whole page turns on: **1 and 3 are verdicts about a tenant, 2 is a failure
of the measurement itself.** A pipeline may legitimately decide to report a verdict without
blocking on it. It may never do that with 2 — a swallowed technical error reports a posture
nobody measured.

## One table, six situations, six real runs

Every command in this table was executed by the documentation generator, and the code in the
last column is the one the process returned.

<!-- pepin:gen exit-codes -->
| Situation | Command | Exit code |
|---|---|:-:|
| No deviation in the evaluated scope | `./pepin scan scaleway --terraform examples/scaleway/terraform-fixed/plan.json` | **0** |
| At least one critical or high deviation | `./pepin scan scaleway --terraform examples/scaleway/terraform/plan.json` | **1** |
| Technical error (unreadable file, unknown provider, unreachable API) | `./pepin scan scaleway examples/scaleway/plan-absent.json` | **2** |
| Nothing was measured (empty inventory): **without having to ask for `--strict`** | `./pepin scan scaleway empty-inventory.json` | **3** |
| Medium/low deviations only, without `--strict` | `./pepin scan scaleway tagless-inventory.json` | **0** |
| Medium/low deviations only, with `--strict` | `./pepin scan scaleway tagless-inventory.json --strict` | **3** |
<!-- /pepin:gen exit-codes -->

The last two rows are the same inventory, with and without `--strict`. The two before them are
the distinction that matters most, and the next section is about it.

## `0` — compliant

No critical or high deviation, **and** at least one control was actually measured.

<!-- pepin:gen exit-run-clean -->
```console
$ ./pepin scan scaleway --terraform examples/scaleway/terraform-fixed/plan.json
[…]
 Summary

 Verdict: compliant on the declared scope (Terraform plan, planned state) (no deviation detected, 16 compliant controls)

 🔴 CRITICAL 0   🟠 HIGH 0   🟡 MEDIUM 0   🔵 LOW 0
──────────────────────────────────────────────────────────────────────────────
$ echo $?
0
```
<!-- /pepin:gen exit-run-clean -->

The verdict line names the scope it covers. "Compliant" here means "no deviation detected on
the declared scope", never "the tenant is certified": see
[Scope and non-goals](../concepts/scope.md).

## `1` — non-compliance

At least one `critical` or `high` deviation.

<!-- pepin:gen exit-run-nonconformity -->
```console
$ ./pepin scan scaleway --terraform examples/scaleway/terraform/plan.json
[…]
 Summary

 Verdict: NON-COMPLIANT

 🔴 CRITICAL 1   🟠 HIGH 7   🟡 MEDIUM 1   🔵 LOW 1
──────────────────────────────────────────────────────────────────────────────
$ echo $?
1
```
<!-- /pepin:gen exit-run-nonconformity -->

Medium and low deviations alone do **not** produce 1. That is deliberate: a gate that blocks on
every low finding is a gate that gets disabled within a week. To include them, ask for
`--strict`, which returns 3.

## `2` — technical error

The scan could not conclude: unreadable file, unknown provider, unreachable API, invalid
credentials, a rule directory that does not parse.

<!-- pepin:gen exit-run-error -->
```console
$ ./pepin scan scaleway examples/scaleway/plan-absent.json

 ██████╗ ███████╗██████╗ ██╗ ███╗   ██╗
 ██╔══██╗██╔════╝██╔══██╗██║ ████╗  ██║
 ██████╔╝█████╗  ██████╔╝██║ ██╔██╗ ██║
 ██╔═══╝ ██╔══╝  ██╔═══╝ ██║ ██║╚██╗██║
 ██║     ███████╗██║     ██║ ██║ ╚████║
 ╚═╝     ╚══════╝╚═╝     ╚═╝ ╚═╝  ╚═══╝

 v<version>  · cloud posture scanner (security · compliance)

error: open examples/scaleway/plan-absent.json: no such file or directory
$ echo $?
2
```
<!-- /pepin:gen exit-run-error -->

The banner goes to standard error, and so does the error message. **2 is never a posture
verdict.** No `allow_failure`, no `continue-on-error`, no `|| true` should ever cover it.

## `3` — nothing measured, or the strict gate

Two situations share this code, and both mean "do not read this run as a green light".

### Nothing was measured — and `--strict` is not required

Since v0.1.0, a scan that measured no control (governance aside) exits **3**, without asking
for `--strict`.

<!-- pepin:gen exit-run-nothing -->
```console
$ ./pepin scan scaleway empty-inventory.json
[…]
 Summary

 Verdict: UNDETERMINED — no control measured on any resource (the assessed scope is empty or was not collected)

 🔴 CRITICAL 0   🟠 HIGH 0   🟡 MEDIUM 0   🔵 LOW 0
──────────────────────────────────────────────────────────────────────────────
$ echo $?
3
```
<!-- /pepin:gen exit-run-nothing -->

The inventory that produces it is a valid, empty one:

<!-- pepin:gen fixture-empty-inventory -->
```json
{
  "provider": "scaleway",
  "resources": []
}
```
<!-- /pepin:gen fixture-empty-inventory -->

"No deviation" and "nothing seen" produce the same empty result set, yet only the first says
something about the posture. Expired credentials, insufficient permissions, an empty region or
a truncated inventory would otherwise turn a CI gate green on a scope nobody ever looked at.
The verdict line says `UNDETERMINED`, and the exit code follows it.

### Medium/low deviations remain, under `--strict`

The same inventory, twice. Without `--strict`, medium and low deviations do not gate:

<!-- pepin:gen exit-run-medium-plain -->
```console
$ ./pepin scan scaleway tagless-inventory.json
[…]
 Summary

 Verdict: no critical/high deviation, but 1 medium/low deviation(s) on the assessed scope (3 compliant)

 🔴 CRITICAL 0   🟠 HIGH 0   🟡 MEDIUM 1   🔵 LOW 0
──────────────────────────────────────────────────────────────────────────────
$ echo $?
0
```
<!-- /pepin:gen exit-run-medium-plain -->

With `--strict`, they do:

<!-- pepin:gen exit-run-strict -->
```console
$ ./pepin scan scaleway tagless-inventory.json --strict
[…]
 Summary

 Verdict: no critical/high deviation, but 1 medium/low deviation(s) on the assessed scope (3 compliant)

 🔴 CRITICAL 0   🟠 HIGH 0   🟡 MEDIUM 1   🔵 LOW 0
──────────────────────────────────────────────────────────────────────────────
$ echo $?
3
```
<!-- /pepin:gen exit-run-strict -->

`--strict` therefore adds one behaviour only: medium/low deviations become blocking. It does
not create the "nothing measured" gate, which exists without it.

## What does not change the exit code

- **The output format.** `--format json`, `sarif`, `oscal` or `assessment` change what is
  written to standard output, never the code returned. A pipeline can gate on the code and
  archive the document.
- **The language.** `--lang fr` and `--lang en` return the same codes on the same input. Codes,
  identifiers, severities and statuses are stable across languages; only prose is translated.
  A pipeline that compares report **text** between runs must pin `PEPIN_LANG`, but a pipeline
  that gates on `$?` has nothing to pin.
- **`--seal`.** Writing an evidence bundle does not alter the verdict.

## Shell

```bash
pepin scan scaleway --terraform plan.json
code=$?
case "$code" in
  0) echo "compliant" ;;
  1) echo "non-compliance: at least one critical/high deviation" ; exit 1 ;;
  3) echo "nothing measured, or medium/low deviations under --strict" ; exit 1 ;;
  2) echo "technical error: the scan could not conclude" ; exit 2 ;;
  *) echo "unexpected code $code" ; exit 2 ;;
esac
```

Never write `pepin scan … || true`: it erases the three codes that carry information, 2
included.

## GitHub Actions

The published action already implements this contract: it fails the job on 1 and 3 (unless
`fail-on-nonconformity: 'false'`), and always fails on 2.

```yaml
- name: Pépin — posture gate
  id: scan
  uses: stephrobert/pepin/.github/actions/pepin-scan@<commit-sha>   # pin by SHA
  with:
    version: '0.2.0'
    provider: scaleway
    terraform-plan: plan.json
    # fail-on-nonconformity: 'false'   # report the verdict without gating; 2 still fails
```

Running the binary by hand in a step needs the case above, because a non-zero exit already
fails the step: the point is to keep 2 distinguishable from 1 and 3 in the job summary. The
full pipeline is in [GitHub Actions](../guides/github-actions.md).

## GitLab CI

```yaml
pepin-terraform-plan:
  script:
    - pepin scan scaleway --terraform plan.json --format json > pepin-report.json
  # Report the posture without gating on it: allow the VERDICT codes, and only them.
  allow_failure:
    exit_codes: [1, 3]
```

`allow_failure: exit_codes: [1, 3]` — **never 2.** Listing 2 turns "the scan could not
conclude" into a green pipeline. The full pipeline is in [GitLab CI](../guides/gitlab-ci.md).

## Where to go next

- [CLI reference](cli.md) — the verbs and flags that produce these codes.
- [The assessment model](../concepts/assessment-model.md) — why `not-evaluated` is not a
  failure, and why it does not gate.
- [GitHub Actions](../guides/github-actions.md) · [GitLab CI](../guides/gitlab-ci.md).

## How this page stays true

The exit codes in the tables are read from the frozen CLI surface; every console block is a
real execution captured by `internal/docgen`, with the code the process actually returned.
`TestTheGeneratorActuallyRunsTheBinary` asserts each of those codes independently, and
`TestEveryExitCodeIsDocumented` fails if a code of the frozen surface never appears here.

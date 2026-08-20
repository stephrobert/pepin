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
| **3** | `strict` | the scan does not establish compliance: nothing was measured, or the collection could not read the whole scope (both without `--strict`), or medium/low deviations remain with `--strict` |
| **4** | `derogation` | every remaining critical/high deviation is covered by a dated, attributed exemption (`--exceptions`) |
<!-- /pepin:gen cli-exit-codes -->

The distinction the whole page turns on: **1 and 3 are verdicts about a tenant, 2 is a failure
of the measurement itself.** A pipeline may legitimately decide to report a verdict without
blocking on it. It may never do that with 2 — a swallowed technical error reports a posture
nobody measured.

## One table, eight situations, eight real runs

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
| No deviation, but one collection unit could not be read | `./pepin scan scaleway partial-inventory.json` | **3** |
| Every critical/high deviation is covered by a valid exemption | `./pepin scan scaleway bastion-inventory.json --exceptions exceptions.yaml` | **4** |
| The same exemption, lapsed: it no longer applies | `./pepin scan scaleway bastion-inventory.json --exceptions exceptions-expired.yaml` | **1** |
<!-- /pepin:gen exit-codes -->

Rows five and six are the same inventory, with and without `--strict`; the last two are the
same inventory and the same exemption, valid then lapsed. The two rows that matter most are
the third and the fourth, and the next section is about them.

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

## `3` — the scan does not establish compliance

Three situations share this code, and all three mean "do not read this run as a green
light": nothing was measured, the collection could not read the whole scope, or the strict
gate caught remaining medium/low deviations.

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

### The collection could not read the whole scope

A collection unit that answered `403`, timed out, or was cut short mid-pagination leaves part
of the scope unread. The inventory below carries that state — this is the shape a live scan
produces, and the shape a sealed bundle's `input.json` replays:

<!-- pepin:gen fixture-partial-inventory -->
```json
{
  "provider": "scaleway",
  "resources": [
    {
      "provider": "scaleway",
      "type": "compute_instance",
      "id": "srv-demo",
      "name": "srv-demo",
      "region": "fr-par",
      "attributes": {
        "vm_id": "srv-demo",
        "security_group_ids": ["sg-front"],
        "tags": [
          {"key": "CostCenter", "value": "R-42"},
          {"key": "Project", "value": "pepin"},
          {"key": "Env", "value": "prod"},
          {"key": "Owner", "value": "platform"}
        ]
      }
    }
  ],
  "collection": {
    "units": [
      {
        "unit": "compute_instance",
        "types": ["compute_instance"],
        "attempted": true,
        "complete": true
      },
      {
        "unit": "security_group_rule",
        "types": ["security_group_rule"],
        "attempted": true,
        "complete": false,
        "error": "permission_denied",
        "detail": "HTTP 403 - GET https://api.scaleway.com/instance/v1/zones/fr-par-1/security_groups - insufficient permissions"
      }
    ]
  }
}
```
<!-- /pepin:gen fixture-partial-inventory -->

The scan announces what it could and could not observe **before** any verdict, on standard
error:

<!-- pepin:gen capability-report -->
```text
Collector capability report
  ✓ compute_instance
  ✗ security_group_rule — insufficient privilege on the scanning account
    HTTP 403 - GET https://api.scaleway.com/instance/v1/zones/fr-par-1/security_groups - insufficient permissions
Result: 6 control(s) cannot be evaluated on this scope.
  · network_securitygroup_allow_ingress_from_internet_to_all_ports
  · network_securitygroup_allow_ingress_from_internet_to_high_risk_tcp_ports
  · network_securitygroup_allow_ingress_from_internet_to_high_risk_udp_ports
  · network_securitygroup_allow_ingress_from_internet_to_tcp_port_22
  · network_securitygroup_allow_ingress_from_internet_to_tcp_port_3389
  · network_securitygroup_unrestricted_egress
```
<!-- /pepin:gen capability-report -->

Every control that reads a resource type fed by the failed unit becomes `not-evaluated`, with
the missing unit named, and the run does not return `0`:

<!-- pepin:gen exit-run-partial -->
```console
$ ./pepin scan scaleway partial-inventory.json
[…]
 Summary

 Verdict: INCOMPLETE — 6 control(s) are not evaluable because the collection is incomplete, 0 medium/low deviation(s) on what could be read

 🔴 CRITICAL 0   🟠 HIGH 0   🟡 MEDIUM 0   🔵 LOW 0
──────────────────────────────────────────────────────────────────────────────
$ echo $?
3
```
<!-- /pepin:gen exit-run-partial -->

The reasoning is the one that gives an empty inventory its `3`: a control that could not be
evaluated says nothing about the posture, and a gate that turns green on a scope nobody read
is the false green this tool exists to prevent. A `fail` observed on the part that *was* read
still returns **1** — an observed deviation stays observed, and incompleteness never erases it.

**Why not a fifth code.** A code for incompleteness could never take precedence over `1`:
hiding a real critical deviation because the rest was missing would be exactly the false green
we are fighting. It would therefore only ever fire where `3` already fires, and two codes for
one position in the order of precedence is a duplicate, not a distinction — it would cost every
consumer a re-read of its `case $?` for no new decision. What separates the situations stays
readable where it is useful: the capability report names the unit and the class of failure,
every affected control carries its reason, and `--format json` publishes a `collection` key.

## `4` — every remaining deviation is under a waiver

A CSPM used in production must allow exceptions. Without them a team disables the control, or
stops reading the tool — two outcomes worse than the deviation itself. But an exemption must
never turn a gate green in silence, which is why it has a code of its own.

Give the scan a versioned exemptions file with `--exceptions`:

<!-- pepin:gen fixture-exceptions -->
```yaml
exceptions:
  - control: network_securitygroup_allow_ingress_from_internet_to_tcp_port_22
    resource: sg-bastion
    justification: "Bastion administre, acces restreint par IP source en amont"
    expires_at: 2099-12-31
    owner: platform-security
    approved_by: security@example.org
```
<!-- /pepin:gen fixture-exceptions -->

Applied to an inventory whose only high deviation is the one it covers:

<!-- pepin:gen fixture-bastion-inventory -->
```json
{
  "provider": "scaleway",
  "resources": [
    {
      "provider": "scaleway",
      "type": "security_group_rule",
      "id": "vm-bastion",
      "name": "vm-bastion",
      "region": "fr-par",
      "attributes": {
        "security_group_id": "sg-bastion",
        "direction": "inbound",
        "action": "accept",
        "protocol": "tcp",
        "port_from": 22,
        "port_to": 22,
        "cidrs": ["0.0.0.0/0"],
        "description": "Acces d administration du bastion"
      }
    }
  ]
}
```
<!-- /pepin:gen fixture-bastion-inventory -->

<!-- pepin:gen exit-run-exempted -->
```console
$ ./pepin scan scaleway bastion-inventory.json --exceptions exceptions.yaml
[…]
  │ CLD-NET-1 │ SSH (port 22) open to the internet │ HIGH │ scaleway │ 1 │
  ╰───────────┴────────────────────────────────────┴──────┴──────────┴───╯
──────────────────────────────────────────────────────────────────────────────
 Summary

 Verdict: NON-COMPLIANT under waiver — 1 critical/high deviation(s), all covered by a dated, attributed exemption

 🔴 CRITICAL 0   🟠 HIGH 1   🟡 MEDIUM 0   🔵 LOW 0
──────────────────────────────────────────────────────────────────────────────

EXEMPTIONS APPLIED — accepted deviations, NOT compliant
  · network_securitygroup_allow_ingress_from_internet_to_tcp_port_22 (sg-bastion)
    Bastion administre, acces restreint par IP source en amont
    until 2099-12-31 · owner platform-security · approved by security@example.org
$ echo $?
4
```
<!-- /pepin:gen exit-run-exempted -->

Read the verdict line: **NON-COMPLIANT under waiver**. The deviation did not disappear, it was
set aside. The control's status in `--format assessment` is `exempted`, never `pass`; the
finding stays in `--format json`, in the SARIF and in the severity counts; only the gate moves.

Why 4 rather than 0 or 1. Returning **0** would make an exemption a silent false green, which
is precisely what the `exempted` status exists to prevent. Returning **1** would make the
exemption useless, and a team that cannot waive a control deletes it instead. **4** is non-zero
— nothing passes in silence — and distinct, so a pipeline that chooses to accept it has to
write the number down, therefore to know it exists.

### An expired exemption does not open the gate

The same file with a date in the past. The exemption stops applying, the deviation is a
deviation again, and the expiry is reported on standard error:

<!-- pepin:gen exit-run-expired -->
```console
$ ./pepin scan scaleway bastion-inventory.json --exceptions exceptions-expired.yaml

 ██████╗ ███████╗██████╗ ██╗ ███╗   ██╗
 ██╔══██╗██╔════╝██╔══██╗██║ ████╗  ██║
 ██████╔╝█████╗  ██████╔╝██║ ██╔██╗ ██║
 ██╔═══╝ ██╔══╝  ██╔═══╝ ██║ ██║╚██╗██║
 ██║     ███████╗██║     ██║ ██║ ╚████║
 ╚═╝     ╚══════╝╚═╝     ╚═╝ ╚═╝  ╚═══╝

 v<version>  · cloud posture scanner (security · compliance)

pepin: ⚠ exemption EXPIRED on 2020-01-01 for network_securitygroup_allow_ingress_from_internet_to_tcp_port_22 / sg-bastion: it no longer applies, the deviation is a deviation again (platform-security)

ⓘ This report assesses the configuration of a tenant (customer-side scope). The normative mappings (SecNumCloud, ISO, CIS) are indicative: they are not a proof of qualification or certification, which applies to the cloud service provider.
$ echo $?
1
```
<!-- /pepin:gen exit-run-expired -->

An exemption that names a control or a resource which does not exist is reported the same way,
as an `ORPHAN` — it is the symptom of an exception forgotten after a rename or a removal.
Under `--strict`, an expired or orphan exemption is enough to fail the gate (code 3): a
pipeline that asks for strictness is asking for its exemption file to be reviewed.

### Order of precedence

When several situations apply at once, the codes are decided in this order:

1. **2** — a technical error: nothing else is a verdict.
2. **1** — at least one critical/high deviation **not** covered by a valid exemption.
3. **3** — nothing was measured (governance aside).
4. **3** — the collection was incomplete: at least one control lost its `pass` because the
   data it needed could not be read.
5. **4** — at least one exemption was applied.
6. **3** — `--strict` and medium/low deviations remain, or the exemption file is stale.
7. **0** — none of the above.

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
  3) echo "the scan does not establish compliance: nothing measured, incomplete collection, or medium/low under --strict" ; exit 1 ;;
  4) echo "deviations remain, all under a dated exemption" ; exit 1 ;;
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

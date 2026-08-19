> 🇬🇧 English · [🇫🇷 Français](adding-a-provider.fr.md)

# Adding a provider, end to end

A provider in Pépin is **a source, not a set of rules**. Every posture rule is common to
every cloud and evaluates the normalized model; what a provider brings is the projection
of its own API and of its Terraform resources onto that model. Adding a cloud therefore
means writing **one descriptor**, `providers/<name>.yaml`, and **zero rules**.

That is not a slogan, it is the acceptance criterion: if your change adds a `.rego` file,
something went wrong in the normalization. See
[the architecture](../project/architecture.md) for why.

Here is what the binary knows about the providers registered today:

<!-- pepin:gen provider-list -->
```text

// pepin  registered providers
  exoscale  Exoscale (CH) — instances, security groups, block storage, SKS, SOS
  kubernetes  Kubernetes (in-cluster) — RBAC, Pod Security Standards, NetworkPolicy
  outscale  Outscale (3DS) — VM, BSU, OOS, EIM, security groups, OKS, LBU
  scaleway  Scaleway — object storage, instances, IAM, security groups
```
<!-- /pepin:gen provider-list -->

## The golden rule: never invent the resource model

The model Pépin evaluates **must** reflect the provider's native contract: real fields,
real types, real JSON tags, read in the SDK or in the official API documentation. A
field you have not verified is not used. A field you **derive** (computed by the
collector or the mapping) is marked as derived, with its formula.

The reason is not purism. A CSPM that reads an invented field does not fail loudly: it
silently returns "compliant", because the attribute is simply never there. That is a
false green, and a false green is invisible by construction.

Two habits make this practical:

- Cache the pages you rely on: `mise run fetch-docs` stores the official documentation
  as Markdown under `references/docs/<provider>/`, and a contract entry then cites a
  file rather than a memory.
- Record the open questions in `references/questions-providers/<provider>.yaml` instead
  of guessing. An unanswered question is a `a_verifier` state, not a `verifie` one.

## Prefer not provisioning anything

Pépin can audit a **Terraform plan** (`terraform show -json`), which is enough to
validate a mapping and a rule **without creating a single resource**:

```bash
terraform plan -out tfplan && terraform show -json tfplan > plan.json
./pepin scan <provider> --terraform plan.json
```

A live scan is only needed to confirm the **API contract** (the fields the API really
returns) when a plan cannot show it. And then the rule is absolute: **every resource
provisioned for a test must be destroyed** (`terraform destroy`, or deletion through the
API or the console). Keep the list of what you create, and confirm the teardown before
you conclude. Cost, exposure surface and drift all argue the same way.

---

## 1. Identity and scope

```yaml
name: acme                       # the identifier the CLI takes: pepin scan acme
description: "ACME (FR) — instances, security groups, object storage"
scope: cloud                     # "cloud" (default) | "in-cluster" for a different scope
region_key: region               # the logical key --region feeds (Exoscale uses "zone")
```

`scope` is not cosmetic: a provider whose scope is not a cloud control plane (the
`kubernetes` provider audits the inside of a cluster) is kept out of the cloud parity
matrix, because neither scope can cover the other.

## 2. Sovereignty facts

These fields feed the `governance_provider` synthetic resource, which control
`CLD-GVN-4` evaluates. They are **facts with sources**, not impressions:

```yaml
souverainete:
  eu_etabli: true                 # is the provider's registered office in the EU
  juridiction: FR                 # country of the registered office
  controle_capitalistique: FR     # jurisdiction of ultimate control: FR | UE | extra_ue | a_verifier
  secnumcloud: non                # qualifie | en_cours | non
  exposition_extraterritoriale: false
  sources: "the URLs that establish the ownership chain"
```

Trace the ownership chain to the end. Exoscale's descriptor is the worked example: a
Swiss company, ultimately controlled from outside the EU, with the chain written out and
sourced.

## 3. Authentication and credential resolution

The descriptor declares how a request is signed and where credentials come from. Three
sources, in order: the environment, the provider's native configuration file, then
defaults.

```yaml
auth:
  type: header                   # header | sigv4 | exoscale-hmac
  header: X-Auth-Token
  value: "{secret_key}"
credentials:
  env:
    access_key: ACME_ACCESS_KEY
    secret_key: ACME_SECRET_KEY
    region: ACME_REGION
  file: { path: "~/.config/acme/config.yaml", format: scw }   # scw | osc | exoscale
  defaults: { region: fr-par }
```

Reading the native configuration file matters more than it looks: a user who already has
the provider's own CLI configured expects Pépin to work without re-exporting anything.

**No secret ever enters the repository.** Credentials come from the environment or from
that file; `mise run secrets` (gitleaks) scans the history for sovereign key patterns.

## 4. Map onto the normalized model

The rules read normalized types (`compute_instance`, `security_group`,
`security_group_rule`, `object_storage_bucket`, `managed_database`, `iam_policy`,
`network`, `kubernetes_cluster`…) and normalized attributes. Your job is to project the
provider's native vocabulary onto them.

Start from the existing types. Read
[`internal/commonrules/rules/`](../../internal/commonrules/rules) to see which
attributes the rules actually read, and
[the control catalogue](../controls/index.md), where each page names the type it reads
and the attribute its decision depends on.

If your provider exposes a mechanism no normalized type covers, that is a normalization
change (a new type, a new attribute) followed by **one common rule** — never a
provider-specific rule.

## 5. The Terraform mapping, anchored on the real schema

```yaml
mapping_terraform:
  resources:
    - tf_type: acme_security_group_rule
      type: security_group_rule
      id: security_group_id
      map:
        security_group_id: security_group_id
        direction: direction
        protocol: protocol
        port_from: from_port
        port_to: to_port
        cidrs: cidr_blocks
      transforms:
        protocol: lower
        cidrs: list
```

This is the part where invention gets caught automatically.
`TestProviderMappingsMatchSchema` runs `terraform providers schema -json` in
`examples/<name>/terraform/` and checks that **every source attribute the spec names
exists in the real provider schema**. A `tf_type` that does not exist, or an attribute
that does not, fails the test. When Terraform or the schema is unavailable the check
skips rather than passing silently — so run it at least once on a machine that has both.

Nested blocks are supported through `items` (the repeated block is exploded, and
`_parent.*` refers to the container), and `for_each` expresses a parent list feeding a
per-item call.

## 6. The live collection spec

The same descriptor describes the live API, driven by the shared collection engine
(`internal/collect`): no Go code for a REST/JSON API.

```yaml
collecte:
  base_url: https://api.{region}.acme.example/v1
  resources:
    - type: compute_instance
      path: /instances
      items: instances
      id: vm_id
      map:
        vm_id: id
        state: state
        public_ip: public_ip
        security_group_ids: security_groups
```

Points that decide whether the result is trustworthy:

- **Pagination.** A list that silently truncates produces a report that misses
  resources. The engine implements four styles (`page`, `token`, `offset-body`,
  `token-body`) and refuses to truncate quietly: it errors out when it hits its page
  bound. Declare the style your API uses rather than accepting the first page.
- **Joins.** `for_each` covers "list the parents, then call the detail endpoint per
  item" — which is how user-data, per-user keys or per-instance details are reached.
- **The region is posted on every resource** by the collection engine, which is what
  makes localisation observable in live for any provider that collects anything.

Two collectors are **shared Go code**, not spec: object storage (S3-compatible) and
managed Kubernetes. You enable them by declaring their endpoint:

```yaml
s3:
  endpoint: "https://s3.{region}.acme.example"
  region: "{region}"
  sse_kms: false        # true only if the provider exposes a customer key per bucket
oks:
  endpoint: "https://api.{region}.oks.acme.example"
```

## 7. The contract: what is verified, what is not, what does not exist

The contract is the honesty layer, and it drives the assessment. For each normalized
type, one of three states:

```yaml
contrat:
  types:
    security_group_rule:
      etat: verifie          # read in the SDK/API and actually projected
      source: "GET /security-groups — SecurityGroupRule.{direction,protocol,from_port}"
      mapping:
        protocol: Protocol (tcp|udp|icmp)
    blockstorage_volume:
      etat: a_verifier       # plausible, unread: no pass will ever be asserted
      source: "to confirm in the SDK"
  non_applicable:
    - control: loadbalancer_http_redirect_to_https
      reason: "aucun mécanisme de redirection HTTP→HTTPS dans l'API des load balancers"
      reason_en: "no HTTP→HTTPS redirection mechanism in the load balancer API"
```

Three invariants are enforced by tests:

- `TestContractVerifiedTypesAreCollected` — a type declared `verifie` **must** be
  collected or mapped. Otherwise the rules that read it are dead: loaded, but never fed.
- `TestEveryContractJustificationIsBilingual` — every `non_applicable` justification
  carries its English counterpart, and the English one carries no accented characters.
  An auditor reads that reason facing a `not-applicable`; an N/A without a reason is not
  opposable, and one that flips language is not opposable either.
- `TestProvidersValid` — identity, auth, credentials and sources are non-empty.

`a_verifier` is not a failure, it is an honest state: it keeps `pass` out of reach and
makes the coverage matrix say `partial` with the reason. Declaring `verifie` to make a
cell green is the one thing that must never happen.

## 8. Fixtures: what is mocked, and what is measured

Fixtures are **realistic** API responses and plans, in the provider's native vocabulary:

```
examples/<name>/terraform/         # a deliberately non-compliant plan (plan.json)
examples/<name>/inventory.json     # a normalized inventory, for an offline scan
```

Be explicit about the two registers, because they do not carry the same weight:

- **Mocked** — a committed fixture proves the *mapping* and the *rule*. It proves
  nothing about the API's behaviour.
- **Measured** — a real run (a plan generated from real Terraform, or a live scan)
  proves the *contract*: that the API really returns that field, with that shape.

A contract entry moves to `verifie` on the strength of the second, never the first. Say
which one you did in the pull request; the difference is the whole value of the tool.

## 9. Regions and zones

`region_key` names the logical key that `--region` feeds, because providers do not agree
on the word: Exoscale reasons in zones, others in regions. It substitutes `{region}` or
`{zone}` in `base_url`, in the S3 endpoint and in paths. Declare a sane `defaults` value
so that a first scan works without ceremony.

## 10. Least-privilege permissions

A Pépin scan is **read-only**. The exact list of API calls the live collection makes is
derived from your descriptor and published on the provider's page — which makes it the
list of rights a read-only key must carry. Do not maintain that list by hand: it is
generated.

To give your provider its page, add it to `documentedProviders` in
[`internal/docgen/providers.go`](../../internal/docgen/providers.go) and create
`docs/providers/<name>.md` and `docs/providers/<name>.fr.md` with the generated regions
(`provider-<name>-identity`, `-credentials`, `-live`, `-terraform`, `-coverage`, `-na`,
`-onesource`, `-scan`). Copy an existing provider page: everything factual in it is
injected.

## 11. Register, build, and check

Descriptors are **embedded** in the binary (`embed.go`, `//go:embed all:providers`), so
adding a cloud requires a rebuild — but no Go code:

```bash
mise run build
./pepin provider list
./pepin scan <name> --terraform examples/<name>/terraform/plan.json
```

Then the gates:

```bash
mise run validate   # reference ↔ rules ↔ frozen SCSL index ↔ catalogue ↔ bilingualism
mise run test       # Go tests (-race) + Rego tests + documentation freshness
mise run audit      # vet + lint + gosec + govulncheck + osv
mise run gen-docs   # regenerate the coverage matrix, the control pages, the provider page
```

## 12. Validating on a real account, if you have one

Only if a live scan is the only way to confirm the contract:

1. Scan **read-only** first, provisioning nothing. Most contract questions are answered
   by what an existing tenant already returns.
2. If a resource must exist to observe a field, create the strict minimum, in an
   isolated project, and write down what you created.
3. Observe, record the field in the contract with its source.
4. **Destroy everything**, and confirm the teardown before concluding.

Never leave a test resource alive. And never claim a coverage a run has not shown: an
unvalidated control stays `fournisseurs: []` — written, tested, activation frozen.

---

## Provider release checklist

- [ ] `providers/<name>.yaml` exists, and **no** `.rego` file was added.
- [ ] Identity, `scope` and `region_key` are set; `pepin provider list` shows the
      provider.
- [ ] Sovereignty facts are filled, with the ownership chain **sourced**.
- [ ] Auth and credential resolution work from the environment **and** from the native
      configuration file.
- [ ] Every mapped attribute is anchored: read in the SDK or in the official docs, and
      cited (a page under `references/docs/<name>/`, or the contract's `source`).
- [ ] Derived attributes are marked as derived, with their formula.
- [ ] `TestProviderMappingsMatchSchema` was run **on a machine with Terraform**, and the
      mapping matches the real provider schema.
- [ ] Every type marked `verifie` is really collected or mapped; anything unread stays
      `a_verifier`.
- [ ] Every `non_applicable` carries its bilingual justification.
- [ ] Fixtures are realistic, and **contain no secret**.
- [ ] The pull request says explicitly what was **mocked** and what was **measured**.
- [ ] Any resource provisioned for a test was **destroyed**, and the teardown is
      confirmed.
- [ ] The provider page exists in both languages, with its generated regions.
- [ ] `mise run gen-docs` has been run and the result committed.
- [ ] `mise run validate`, `mise run test` and `mise run audit` are green.

## See also

- [Architecture](../project/architecture.md) — the source changes, the rule does not.
- [Adding a control](adding-a-control.md) — the other half: the rule.
- [Terraform plan vs live scan](../concepts/terraform-vs-live.md) — what each source can
  and cannot show.
- [Coverage matrix](../coverage.md) — where your provider will appear, and why a cell is
  not green.
- Existing descriptors, which are the real reference:
  [`providers/exoscale.yaml`](../../providers/exoscale.yaml),
  [`providers/outscale.yaml`](../../providers/outscale.yaml),
  [`providers/scaleway.yaml`](../../providers/scaleway.yaml).

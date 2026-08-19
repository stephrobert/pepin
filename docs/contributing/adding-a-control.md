> 🇬🇧 English · [🇫🇷 Français](adding-a-control.fr.md)

# Adding a control, end to end

This guide follows one real control of this repository, from the risk to the pull
request: **`database_service_not_open_to_internet`** — a managed database whose allowed
IP list admits a public CIDR. Every file quoted below exists; nothing here is a sketch.

Before anything else, two invariants decide how the work is shaped.

**One rule set, common to every provider.** A control is written **once**, in
`internal/commonrules/rules/`, and it evaluates the *normalized* model. What changes
from one cloud to the next is the **source** (the collector, the Terraform mapping),
never the rule. If a check seems to need provider-specific logic, the normalization is
what is missing, not a second rule. See [the architecture](../project/architecture.md).

**Everything a human reads is written twice.** French is the reference language of the
normative content, English its maintained translation. A control carries `titre_en`,
`description_en` and `remediation_en`; a rule carries `labels.message_en` and
`labels.remediation_en`. `mise run validate` refuses any of these missing — an English
report must never fall back to French mid-sentence.

You do **not** need a cloud account to follow this guide. Auditing a Terraform plan
provisions nothing, and it is enough to validate a mapping and a rule.

---

## 1. Identify the risk

State it as an observable configuration fact, not as an intention. "The database is
reachable from the internet" is observable; "the database is insecure" is not.

For our example: a managed database exposes an ACL (a list of allowed CIDRs). If that
list contains `0.0.0.0/0`, the service is reachable from the whole internet.

## 2. Find the normative reference — the index is frozen

Every control maps onto an **existing** SCSL requirement of the cloud posture module
(`CLD-*`). The index is **frozen**: you map onto a requirement, you never create one.

The frozen identifiers are listed in
[`referentiel/scsl-baseline.json`](../../referentiel/scsl-baseline.json):

```bash
grep -o 'CLD-[A-Z]*-[0-9]*' referentiel/scsl-baseline.json | sort -u
```

With the SCSL framework checked out next to Pépin, the full report is:

```bash
./pepin scsl                                          # default index: ../framework-scsl/api/v1/exigences.json
./pepin scsl --index /path/to/api/v1/exigences.json
```

Our example maps onto `CLD-NET-1` (the source of an exposed service is restricted to a
controlled CIDR, never `0.0.0.0/0`).

**If no frozen requirement covers the need, stop here.** The control stays in
[`referentiel/catalogue.yaml`](../../referentiel/catalogue.yaml) with
`statut: a_trier`, out of scope until SCSL freezes something for it. Inventing a
reference would make the whole report unopposable, which is the one failure mode this
project cannot afford.

## 3. Check applicability, provider by provider

For each provider you intend to activate, the native field must be **verified in the
SDK or the official API documentation** and recorded in `providers/<provider>.yaml`.
"Absent" is a finding too, and it is proven by reading the SDK, never assumed.

For Scaleway, the contract records the type as verified and names where the attribute
comes from:

```yaml
    managed_database:
      etat: verifie
      source: >
        Terraform scaleway_rdb_acl (mapping validé par un plan réel : acl_rules[].ip
        agrégés en ip_filter ; formes reprises de scaleway/dagster-scaleway et
        Qovery/engine, ip=0.0.0.0/0). API live api/rdb/v1 ACLRule.IP à câbler en collecte.
      mapping:
        database_id: instance_id du RDB parent
        ip_filter: '[acl_rules[].ip] (CIDR autorisés ; transform list)'
```

Three states exist, and they are not interchangeable: `verifie` (read in the SDK, and
projected), `a_verifier` (plausible, unread — no `pass` will be asserted), `absent`
(the mechanism does not exist, with its justification). The state gates the `pass`:
`internal/assess` refuses to assert compliance on a type the contract has not verified.

## 4. Add the control to the reference

`referentiel/controles.yaml` is the source of truth. A control is agnostic: its `code`
follows `<service>_<resource>_<check>` with a neutral service prefix (`network_`,
`compute_`, `objectstorage_`, `blockstorage_`, `iam_`, `kubernetes_`, `loadbalancer_`,
`governance_`). No provider name ever appears in a code.

```yaml
  - code: database_service_not_open_to_internet
    famille: reseau
    titre: Base de données managée joignable depuis Internet
    titre_en: Managed database reachable from the internet
    severite: high
    description: >
      La liste d'IP autorisées (ACL) d'une base de données managée admet un CIDR
      public : le service de base de données est joignable depuis Internet.
    description_en: >
      The allowed IP list (ACL) of a managed database admits a public CIDR: the
      database service is reachable from the internet.
    remediation: >
      Restreindre l'ACL de la base aux seuls CIDR applicatifs (réseau privé quand
      disponible) ; retirer 0.0.0.0/0.
    remediation_en: >
      Restrict the database ACL to the application CIDRs only (a private network
      where one is available); remove 0.0.0.0/0.
    scsl: [CLD-NET-1]
    frameworks:
      secnumcloud_3_2: ["13.2"]
      cis_controls_v8: ["4.4", "12.2"]
      iso_27001_2022: ["A.8.20", "A.8.22"]
    fournisseurs: [scaleway]
```

`fournisseurs` is the activation switch. Leave it **empty** until the mapping has been
validated on a real plan or a real scan: a control declared for a provider it cannot
actually measure is a false promise, and the coverage matrix will say so.

## 5. Determine the deciding attributes

This is the step that separates an honest control from a false green, and it is
detailed in [§ 6](#6-the-step-everyone-forgets-missing-data).

Ask one question: **if the attribute the rule reads was never collected, does the rule
stay silent?** For our example, yes: no `ip_filter`, no finding. Silence would then be
read as compliance, which is exactly wrong — a Scaleway database is open until an ACL
is set.

The attribute is therefore declared in the `requiredAttr` table of
[`internal/assess/assess.go`](../../internal/assess/assess.go):

```go
	// La base est ouverte tant qu'aucune ACL n'est posée (défaut d'API Scaleway) :
	// sans ip_filter collecté, « conforme » est exactement l'inverse de la réalité.
	"database_service_not_open_to_internet": {"ip_filter"},
```

Controls **not** listed are judged on the presence of a deviation: the absence of a bad
configuration genuinely means compliance. That is a decision to make deliberately, per
control, not a default to inherit.

## 6. The step everyone forgets: missing data

A rule that does not fire for lack of data must produce **`not-evaluated`**, never
`pass`. This is not a nuance; it is the difference between an audit and a reassurance.
An internal audit of this repository found **fourteen** controls concluding "compliant"
without ever having read the deciding attribute. A false green is invisible by
construction, which makes it the worst possible defect for a posture tool.

Two mechanisms work together, and both are needed:

1. **The capability guard, in the rule.** The rule reads the attribute only if it is
   present, so an uncollected attribute produces no false positive:

   ```rego
   "ip_filter" in object.keys(r.attributes)
   ```

2. **The `pass` lock, in `internal/assess`.** The `requiredAttr` entry turns the
   silence into `not-evaluated` when no resource of the targeted type carried the
   attribute. Without it, the guard would silently buy a green.

A test keeps the two in sync: `TestRequiredAttrGuardsExist`
(`internal/assess/requiredattr_test.go`) fails if a `requiredAttr` entry names a control
no rule emits, or an attribute the rule never reads — "a gate that protects nothing".

The generated table of every gated control, with its deciding attribute, is in
[the assessment model](../concepts/assessment-model.md).

## 7. Write the common rule

One file, `internal/commonrules/rules/<code>.rego`, `package pepin.rules`,
`import rego.v1`, and a `deny contains f if { … }`. The header cites the anchored
source; `labels.provider` comes **from the resource** through `provider_of(r)`, never
hardcoded.

```rego
# database_service_not_open_to_internet
#   Base de données managée dont la liste d'IP autorisées (ACL) admet un CIDR
#   public : le service de base de données est joignable depuis Internet.
# SCSL : CLD-NET-1 (source restreinte à un CIDR maîtrisé, jamais 0.0.0.0/0).
# Contrat : type normalisé agnostique `managed_database` ; attribut ip_filter
#   (liste de CIDR autorisés, DÉRIVÉ — Scaleway RDB ACLs / Exoscale DBaaS
#   ip-filter). Absent ⇒ pas de finding (garde de capacité).
package pepin.rules

import rego.v1

deny contains f if {
	some r in resources_of_type("managed_database")
	"ip_filter" in object.keys(r.attributes)
	some cidr in cidr_list(object.get(r.attributes, "ip_filter", []))
	is_public_cidr(cidr)
	id := object.get(r.attributes, "database_id", r.id)
	f := {
		"code": "database_service_not_open_to_internet",
		"severity": "high",
		"subject": id,
		"message": sprintf("Base de données managée « %s » : ACL autorisant un CIDR public (%s) — service exposé à Internet.", [id, cidr]),
		"remediation": "Restreindre l'ACL de la base aux seuls CIDR applicatifs (réseau privé quand disponible) ; retirer 0.0.0.0/0.",
		"labels": {
			"provider": provider_of(r),
			"category": "security",
			"message_en": sprintf("Managed database \"%s\": ACL allowing a public CIDR (%s) — the service is exposed to the internet.", [id, cidr]),
			"remediation_en": "Restrict the database ACL to the application CIDRs only (a private network where one is available); remove 0.0.0.0/0.",
		},
	}
}
```

Two details that are easy to get wrong:

- **Defensive access only** (`object.get`, `object.keys`): an inventory is untrusted
  input, and a rule must never crash on a missing field.
- **The severity must match the reference.** `TestRegoSeverityMatchesReferentiel`
  compares the two and fails on a divergence.

Reuse the shared helpers of `lib.rego` (`resources_of_type`, `provider_of`,
`is_public_cidr`, `cidr_list`) rather than re-implementing them: a helper fixed once is
fixed for every rule.

## 8. Add the fixtures

Fixtures are **realistic API or plan responses**, never invented shapes. Two mirrors
matter:

- the non-compliant one, under `examples/<provider>/terraform/` (or the inventory
  fixture), which makes the rule fire;
- the compliant one, under `examples/<provider>/terraform-fixed/` or as a remediation
  proof in `references/remediation/<provider>/<code>/`, which must **not** make it fire.

No secret ever goes into a fixture: credentials come from the environment, and
`mise run secrets` (gitleaks) scans the history for the sovereign key patterns.

## 9. Test the failure, the pass and the absence

The rule's test file, `<code>_test.rego`, must contain the three cases. This is the
real one, unabridged:

```rego
package pepin.rules

import rego.v1

_db(attrs) := {"resources": [{"provider": "scaleway", "type": "managed_database", "id": "db-1", "attributes": attrs}]}

# ✗ ACL avec 0.0.0.0/0 → finding.
test_db_open_to_all_denied if {
	some f in deny with input as _db({"database_id": "db-1", "ip_filter": ["0.0.0.0/0"]})
	f.code == "database_service_not_open_to_internet"
}

# ✗ ACL avec un /1 (moitié d'Internet) → finding (réutilise is_public_cidr).
test_db_half_internet_denied if {
	some f in deny with input as _db({"database_id": "db-1", "ip_filter": ["128.0.0.0/1"]})
	f.code == "database_service_not_open_to_internet"
}

# ✓ ACL restreinte à un CIDR privé → aucun finding.
test_db_restricted_ok if {
	count({f | some f in deny; f.code == "database_service_not_open_to_internet"}) == 0 with input as _db({"database_id": "db-1", "ip_filter": ["10.0.0.0/16"]})
}

# ✓ ip_filter non collecté (garde de capacité) → pas de faux positif.
test_db_uncollected_ok if {
	count({f | some f in deny; f.code == "database_service_not_open_to_internet"}) == 0 with input as _db({"database_id": "db-1"})
}
```

The last test is the one people skip. It proves the guard exists; the `requiredAttr`
entry of step 5 is what turns its silence into `not-evaluated` instead of `pass`. Both
are needed: the test alone would leave a false green, the table alone would leave a
false positive.

```bash
mise run test-rego     # opa test internal/commonrules/rules -v
mise run test          # Go tests (-race) + Rego tests
```

A test is only worth what its ability to fail is worth. Break the rule on purpose (drop
the guard, flip a comparison) and check the suite goes red. A suite that stays green
while the subject is broken is measuring its own failure, not the code.

## 10. Check `not-applicable` and `not-evaluated` on a real scan

Run the scan and read the statuses, rather than trusting the unit tests:

```bash
./pepin scan scaleway --terraform examples/scaleway/terraform/plan.json --format assessment
./pepin scan scaleway --terraform examples/scaleway/terraform-fixed/plan.json --format assessment
```

What to check, in order:

- the control appears in `results` (if it does not, the code is not in the reference or
  no rule emits it — `mise run validate` says which);
- it is `fail` on the non-compliant input, with a `subject` naming the offending
  resource;
- it is `pass` on the compliant input, and `evidence.observed` says **why** the pass is
  asserted;
- on an input where the deciding attribute is absent, it is `not-evaluated` **with its
  reason** — not `pass`.

If a provider genuinely has no such mechanism, that is not a gap in Pépin: record it as
`non_applicable` in `providers/<provider>.yaml` **with its justification**, bilingual.
The scan then returns `not-applicable` with that reason, which an auditor can act on. An
N/A without a justification is not opposable, and a test enforces the bilingual
justification.

## 11. Document the remediation

The `remediation` and `remediation_en` fields of step 4 are mandatory and gated by
`TestEveryFindingCarriesRemediation`. The layer above — a **deployable** proof under
`references/remediation/<provider>/<code>/` — is described in
[the remediation guide](../guides/remediation.md). It is the difference between telling
the reader what to do and showing them the setup that does it.

## 12. Document the change — the step the procedure now makes explicit

The canonical procedure (`CLAUDE.md` §10, and the same requirement in
[CONTRIBUTING.md](../../CONTRIBUTING.md)) ends on a seventh step: **documenting is part
of the change, not a follow-up**. It has three parts, and only the first is automatic.

**Regenerate.** The control catalogue, the coverage matrix and every command output
shown in `docs/` are **generated**. Never edit them by hand:

```bash
mise run gen-docs
```

Your control now has its two pages under `docs/controls/`, it appears in
`docs/coverage.md` with its status per provider and per source, and the figures move.
`TestGeneratedDocsAreUpToDate` fails on any documentation that lags behind, so
regenerating is not optional — but passing that test only proves the *generated* pages
are current.

**Re-read what your change makes false.** The generator cannot know that a sentence
written by hand has stopped being true. A new control usually falsifies at least one of:

- [`docs/known-limitations.md`](../known-limitations.md) — if your control closes a
  blind spot that page names, the page now understates what Pépin measures. Move the
  limitation to *Resolved limitations* with the version that lifted it.
- the page of each provider you activated
  ([`docs/providers/`](../providers/scaleway.md)) — its coverage prose and its list of
  API calls.
- [`docs/concepts/scope.md`](../concepts/scope.md) — if the control opens a family the
  scope page said Pépin did not cover.

And in **both languages**. A French page left behind is not a translation debt, it is a
page that tells a French reader something the tool no longer does.

**Add a CHANGELOG line.** Activating a control changes a verdict on a tenant nobody
touched: yesterday's report said nothing, today's says `fail`, and someone will have to
explain that to an auditor. That is exactly what the CHANGELOG exists for, so a new
active control always earns its line, in both languages. A control that stays dormant
(`fournisseurs: []`) moves no verdict and needs none.

The question that settles every case: *would someone reading the documentation without
reading the code be misled by this change?*

---

## Checklist before the pull request

- [ ] The SCSL requirement (`CLD-*`) **already existed** in the frozen index; none was
      invented. If none covers the need, the control stayed in `catalogue.yaml`.
- [ ] For every activated provider, the native field is recorded in
      `providers/<provider>.yaml` with `etat: verifie` and its source.
- [ ] The control is in `referentiel/controles.yaml` with an agnostic `code`, its
      severity, its `scsl`, its framework mappings and its `fournisseurs`.
- [ ] The six prose fields are filled **in both languages** (`titre`/`titre_en`,
      `description`/`description_en`, `remediation`/`remediation_en`).
- [ ] One rule in `internal/commonrules/rules/`, `package pepin.rules`, with
      `labels.provider: provider_of(r)` and `labels.message_en` /
      `labels.remediation_en`.
- [ ] The rule severity matches the reference.
- [ ] The rule carries its **capability guard**, and the deciding attribute is declared
      in `requiredAttr` when silence would otherwise buy a `pass`.
- [ ] `<code>_test.rego` covers the **failure**, the **pass** and the **missing
      attribute**.
- [ ] The suite was checked for its ability to fail: breaking the rule turns it red.
- [ ] A real scan shows `fail`, `pass` and `not-evaluated` where each is expected,
      never a `pass` on uncollected data.
- [ ] Any `non_applicable` is justified, bilingually, in the provider contract.
- [ ] `mise run gen-docs` has been run and the result committed.
- [ ] The **hand-written** pages the control makes wrong were re-read, in both
      languages: known limitations, the page of each activated provider, scope.
- [ ] A CHANGELOG line, in both languages, if the control is active — an activated
      control moves a verdict on an unchanged tenant.
- [ ] `mise run validate`, `mise run test` and `mise run audit` are green.
- [ ] The commit follows Conventional Commits, imperative, under 72 characters.

## See also

- [Architecture](../project/architecture.md) — why one rule set, and what changes per cloud.
- [Adding a provider](adding-a-provider.md) — the other half: the source.
- [Control catalogue](../controls/index.md) — what your control's page will look like.
- [The assessment model](../concepts/assessment-model.md) — what each status asserts.
- [CONTRIBUTING.md](../../CONTRIBUTING.md) — the quality gates and the non-negotiables.

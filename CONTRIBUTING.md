> 🇬🇧 English · [🇫🇷 Français](CONTRIBUTING.fr.md)

# Contributing to Pépin

Thanks for contributing. Pépin is an audit tool: the quality bar is high because a
false result defeats its very purpose (opposability).

## Quality gates

```bash
mise run build   # compile
mise run test    # go test ./... -race + Rego tests (opa)
mise run audit   # vet + lint (golangci-lint) + gosec + govulncheck + osv
```

Do not submit a change if `mise run test` or `mise run audit` fails. `opa test`,
`opa check --strict` and `opa fmt` must stay clean.

## Non-negotiable rules

1. **Anchor on the API contract.** Never invent a provider's resource model: the
   evaluated model reflects the native SDK/API contract. An unverified field is
   not used; a **derived** field is marked "DÉRIVÉ" with its formula. Cite the
   source in each rule's header.
2. **Never invent a normative reference.** Every SCSL requirement (`CLD-*`) and
   every norm mapping is checked against the official text. Tests
   (`TestSCSLReferencesExist`, `TestFrameworkReferencesExist`) reject a
   non-existent id; add the requirement to the source first if it is missing.
3. **Effective configuration.** A control queries the resolved state, never a
   service file. Every rule carries a **capability guard**: if the needed
   attribute was not collected, it does not fire (no false positive).
4. **No rule change without a real scan.** A new rule or a provider extension MUST
   be validated by a real scan (live collection, or auditing a Terraform plan)
   before any coverage is claimed. An unvalidated control stays `fournisseurs: []`
   (written and tested, activation frozen).
5. **No test resource left alive.** If a test provisions cloud resources, they
   MUST be destroyed afterwards (`terraform destroy`). Prefer the Terraform plan
   (`scan --terraform`), which provisions nothing.
6. **No hardcoded secrets** (keys, passwords): CLI options / environment /
   `random_password` in fixtures.

## Adding a provider

A provider is one `providers/<name>.yaml` file: identity, auth, credential
resolution, S3 endpoint, `collecte` (live API), `mapping_terraform`, and the
`contrat` (state per type: `verifie` / `a_verifier` / `absent`, with its source).

## Adding a control

1. Write the rule `internal/commonrules/rules/<code>.rego` (emits the neutral
   `code`, with a capability guard) and its `_test.rego`.
2. Declare the control in `referentiel/controles.yaml` (severity, SCSL, norm
   mappings, `fournisseurs`).
3. **Write its veracity scenarios** in `internal/veracity/testdata/scenarios/` —
   one file per control × provider × source path, proving the verdicts that path
   can reach end to end, through the binary. A Rego test proves one link; the unit
   that matters is the whole chain, and it is a *collector* that let the founding
   incident through, not a rule. What is not proven is **recorded**
   (`mise run veracity-update`), never hidden: the ledger is a gate in both
   directions, and a control added without its scenarios fails the build.
4. Validate with a real scan, then activate (`fournisseurs`).
5. **Re-read the reference tenants** (`references/tenants/`): real third-party
   configurations, pinned to a commit, replayed on every build. A fixture is
   written by the rule's author, so it is self-confirming; a tenant nobody wrote
   for Pépin is where a false positive shows up. If a verdict flips there, say
   whether the product improved or regressed **before** running
   `mise run tenants-update`. See
   [Reference tenants](docs/guides/reference-tenants.md).

Everything the user reads is **written twice**, side by side, because Pépin
detects the reader's language. A rule emits `message`/`remediation` in French and
`labels.message_en`/`labels.remediation_en` in English; a control carries
`titre_en`, `description_en` and `remediation_en`; a provider contract carries
`reason_en` next to `reason`. `mise run validate` refuses any of these missing —
an English report must never fall back to French mid-sentence. French is the
reference language of the normative content; English is its translation.

## Docs & commits

Repository docs are **bilingual**: English is primary (`README.md`), French is the
counterpart (`README.fr.md`), linked by a language switcher. Commits follow
Conventional Commits (`feat`/`fix`/`docs`/`refactor`/`test`/`chore`), imperative
subject.

**Documentation is part of the change, not a follow-up.** Most of `docs/` is
generated from the binary and the reference, and `TestGeneratedDocsAreUpToDate`
fails the build when it drifts. So, before opening a pull request:

- run `mise run gen-docs` and commit what it regenerates;
- re-read the pages your change makes **wrong**, not only the generated ones:
  `docs/known-limitations.md` when a blind spot closes, the provider page you
  touched, `docs/coverage.md`;
- keep both languages in step;
- add a CHANGELOG line, in both languages, whenever the change moves a **verdict**
  on an unchanged tenant, a parsable surface, or an exit code.

A page describing a product that no longer exists is worse than a missing page:
it earns trust it cannot honour. The question that settles it: *would someone
reading the documentation without reading the code be misled by this change?*

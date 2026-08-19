> 🇬🇧 English · [🇫🇷 Français](understanding-a-scan.fr.md)

# Understanding a scan

This page reads one real Pépin run end to end. The command:

```bash
./pepin scan scaleway --terraform examples/scaleway/terraform/plan.json
```

Everything reproduced below was captured from that exact command on this repository — the
terminal report, the assessment document, the exit code. Nothing is illustrative.

## The two streams

Pépin separates them on purpose, so that `pepin scan … > report.txt` gives you the report and
nothing else.

**`stderr`** carries the banner and the scope disclaimer:

<!-- pepin:gen scan-vulnerable-banner -->
```text

 ██████╗ ███████╗██████╗ ██╗ ███╗   ██╗
 ██╔══██╗██╔════╝██╔══██╗██║ ████╗  ██║
 ██████╔╝█████╗  ██████╔╝██║ ██╔██╗ ██║
 ██╔═══╝ ██╔══╝  ██╔═══╝ ██║ ██║╚██╗██║
 ██║     ███████╗██║     ██║ ██║ ╚████║
 ╚═╝     ╚══════╝╚═╝     ╚═╝ ╚═╝  ╚═══╝

 v<version>  · scanner de posture cloud (sécurité · conformité)


ⓘ Ce rapport évalue la configuration d'un tenant (périmètre commanditaire). Les correspondances normatives (SecNumCloud, ISO, CIS) sont indicatives : elles ne constituent pas une preuve de qualification/certification, laquelle porte sur le prestataire de service cloud.
```
<!-- /pepin:gen scan-vulnerable-banner -->

The version is shown as `<version>`: it is injected at build time from `git describe`, so it
differs between machines and commits, and freezing it here would make this page diverge on
every build without any behaviour changing. Everything else is the captured output.

The banner is printed *before* collection starts, not at the end: on a live scan against a
large tenant, you want to know the tool started. The closing line is
`assess.ScopeDisclaimer`, emitted on every single scan — see [Scope](../concepts/scope.md).

**`stdout`** carries the report:

<!-- pepin:gen scan-vulnerable-full -->
```text
──────────────────────────────────────────────────────────────────────────────
 Mode      scan scaleway (terraform)
 Source    examples/scaleway/terraform/plan.json
──────────────────────────────────────────────────────────────────────────────

──────────────────────────────────────────────────────────────────────────────
 ⚡ Immediate action — top 3 most severe deviations
──────────────────────────────────────────────────────────────────────────────

  1. 🔴 CRIT  CLD-CMP-1 — aucun filtrage réseau ne s'applique.
     subject: scaleway_instance_server.web
  2. 🔴 CRIT  CLD-STO-1 — Bucket « scaleway_object_bucket_acl.backups » accessible publiq…
     subject: scaleway_object_bucket_acl.backups
  3. 🟠 HIGH  CLD-CMP-9 — secret en clair dans user-data (mot de passe en clair).
     subject: scaleway_instance_server.web


──────────────────────────────────────────────────────────────────────────────
 CRITICAL  ·  CLD-CMP-1  ·  scaleway
 Machine sans filtrage réseau
──────────────────────────────────────────────────────────────────────────────
  Total deviations: 1

  Details:
      CRIT  scaleway_instance_server.web — aucun filtrage réseau ne s'applique.

  Remediation
    Attacher un groupe de sécurité restrictif (refus par défaut) à la VM.

  ↳ docs: https://stephane-robert.info/scsl/CLD-CMP-1

──────────────────────────────────────────────────────────────────────────────
 CRITICAL  ·  CLD-STO-1  ·  scaleway
 Stockage objet exposé publiquement
──────────────────────────────────────────────────────────────────────────────
  Total deviations: 1

  Details:
      CRIT  scaleway_object_bucket_acl.backups — Bucket « scaleway_object_bucket_acl.backups » accessible publiquement (ACL publique).

  Remediation
    Rendre le bucket privé (ACL private, retrait du grant AllUsers, suppression de la policy publique) ; servir via des URLs pré-signées si nécessaire.

  ↳ docs: https://stephane-robert.info/scsl/CLD-STO-1

──────────────────────────────────────────────────────────────────────────────
 HIGH  ·  CLD-CHF-2  ·  scaleway
 Base de données managée sans chiffrement au repos
──────────────────────────────────────────────────────────────────────────────
  Total deviations: 1

  Details:
      HIGH  pepin-test-rdb — Base de données managée « pepin-test-rdb » sans chiffrement au repos.

  Remediation
    Activer le chiffrement au repos de l'instance (à la création ou par mise à niveau).

  ↳ docs: https://stephane-robert.info/scsl/CLD-CHF-2

──────────────────────────────────────────────────────────────────────────────
 HIGH  ·  CLD-CMP-9  ·  scaleway
 Secret en clair dans les données utilisateur (user-data)
──────────────────────────────────────────────────────────────────────────────
  Total deviations: 1

  Details:
      HIGH  scaleway_instance_server.web — secret en clair dans user-data (mot de passe en clair).

  Remediation
    Bannir les secrets des données utilisateur ; utiliser un coffre de secrets et l'injection au démarrage. Révoquer le secret exposé.

  ↳ docs: https://stephane-robert.info/scsl/CLD-CMP-9

──────────────────────────────────────────────────────────────────────────────
 HIGH  ·  CLD-IAM-12  ·  scaleway
 Politique IAM permettant une élévation de privilèges
──────────────────────────────────────────────────────────────────────────────
  Total deviations: 1

  Details:
      HIGH  ci-deployer — confère la gestion de l'IAM (PermissionSet) — chemin d'élévation de privilèges.

  Remediation
    Réserver la gestion IAM à une politique d'administration dédiée ; retirer le PermissionSet de gestion des politiques d'usage.

  ↳ docs: https://stephane-robert.info/scsl/CLD-IAM-12

──────────────────────────────────────────────────────────────────────────────
 HIGH  ·  CLD-NET-1  ·  scaleway
 Base de données managée joignable depuis Internet
──────────────────────────────────────────────────────────────────────────────
  Total deviations: 2

  Details:
      HIGH  fr-par/11111111-1111-1111-1111-111111111111 — ACL autorisant un CIDR public (0.0.0.0/0) — service exposé à Internet.
      HIGH  scaleway_instance_security_group.web — SSH (port 22) accepté depuis/vers Internet.

  Remediation
    Restreindre l'ACL de la base aux seuls CIDR applicatifs (réseau privé quand disponible) ; retirer 0.0.0.0/0.

  ↳ docs: https://stephane-robert.info/scsl/CLD-NET-1

──────────────────────────────────────────────────────────────────────────────
 HIGH  ·  CLD-NET-2  ·  scaleway
 Politique entrante par défaut d'un groupe de sécurité en « accept »
──────────────────────────────────────────────────────────────────────────────
  Total deviations: 1

  Details:
      HIGH  sg-open-default — politique entrante par défaut « accept » — tout trafic non filtré est admis.

  Remediation
    Basculer la politique entrante par défaut sur « drop » et n'ouvrir que les flux légitimes par des règles explicites.

  ↳ docs: https://stephane-robert.info/scsl/CLD-NET-2

──────────────────────────────────────────────────────────────────────────────
 HIGH  ·  CLD-STO-3  ·  scaleway
 Sauvegardes automatiques d'une base managée désactivées
──────────────────────────────────────────────────────────────────────────────
  Total deviations: 1

  Details:
      HIGH  pepin-test-rdb — sauvegardes automatiques désactivées.

  Remediation
    Réactiver les sauvegardes automatiques et fixer une rétention adaptée au RPO.

  ↳ docs: https://stephane-robert.info/scsl/CLD-STO-3

──────────────────────────────────────────────────────────────────────────────
 MEDIUM  ·  CLD-GVN-1  ·  scaleway
 Inventaire et étiquetage incomplets
──────────────────────────────────────────────────────────────────────────────
  Total deviations: 1

  Details:
      MED   scaleway_instance_server.web — étiquettes de gouvernance manquantes (CostCenter, Project, Env, Owner).

  Remediation
    Ajouter les étiquettes obligatoires (CostCenter, Project, Env, Owner) sur la ressource.

  ↳ docs: https://stephane-robert.info/scsl/CLD-GVN-1

──────────────────────────────────────────────────────────────────────────────
 LOW  ·  CLD-STO-8  ·  scaleway
 Object Lock (immutabilité) désactivé sur le stockage objet
──────────────────────────────────────────────────────────────────────────────
  Total deviations: 1

  Details:
      LOW   backups-prod — objets non immuables (pas de protection WORM contre suppression/écrasement).

  Remediation
    Activer l'Object Lock (mode conformité/gouvernance) sur les buckets de sauvegarde et d'objets critiques.

  ↳ docs: https://stephane-robert.info/scsl/CLD-STO-8

  Controls
  ╭────────────┬──────────────────────────────────────────────────┬──────────┬──────────┬───╮
  │ Code       │ Control                                          │ Sev      │ Tier     │ # │
  ├────────────┼──────────────────────────────────────────────────┼──────────┼──────────┼───┤
  │ CLD-CMP-1  │ Machine sans filtrage réseau                     │ CRITICAL │ scaleway │ 1 │
  │ CLD-STO-1  │ Stockage objet exposé publiquement               │ CRITICAL │ scaleway │ 1 │
  │ CLD-CHF-2  │ Base de données managée sans chiffrement au rep… │ HIGH     │ scaleway │ 1 │
  │ CLD-CMP-9  │ Secret en clair dans les données utilisateur (u… │ HIGH     │ scaleway │ 1 │
  │ CLD-IAM-12 │ Politique IAM permettant une élévation de privi… │ HIGH     │ scaleway │ 1 │
  │ CLD-NET-1  │ Base de données managée joignable depuis Intern… │ HIGH     │ scaleway │ 2 │
  │ CLD-NET-2  │ Politique entrante par défaut d'un groupe de sé… │ HIGH     │ scaleway │ 1 │
  │ CLD-STO-3  │ Sauvegardes automatiques d'une base managée dés… │ HIGH     │ scaleway │ 1 │
  │ CLD-GVN-1  │ Inventaire et étiquetage incomplets              │ MEDIUM   │ scaleway │ 1 │
  │ CLD-STO-8  │ Object Lock (immutabilité) désactivé sur le sto… │ LOW      │ scaleway │ 1 │
  ╰────────────┴──────────────────────────────────────────────────┴──────────┴──────────┴───╯
──────────────────────────────────────────────────────────────────────────────
 Summary

 Verdict : NON CONFORME

 🔴 CRITICAL 2   🟠 HIGH 7   🟡 MEDIUM 1   🔵 LOW 1
──────────────────────────────────────────────────────────────────────────────
```
<!-- /pepin:gen scan-vulnerable-full -->

## Reading it section by section

### Header

<!-- pepin:gen scan-header -->
```text
──────────────────────────────────────────────────────────────────────────────
 Mode      scan scaleway (terraform)
 Source    examples/scaleway/terraform/plan.json
──────────────────────────────────────────────────────────────────────────────
[…]
```
<!-- /pepin:gen scan-header -->

`Mode` names the **provider** and the **source**. `(terraform)` means a plan was projected
through the provider's Terraform mapping; `(live)` would mean the API was queried, and `Source`
would then read `collecte live · profil … · région …` instead of a file path. Plain
`scan scaleway` with a file means a pre-normalized inventory export.

The distinction is not cosmetic. A plan carries the **planned** state and omits anything
`known after apply`; a live collection carries the **effective** configuration but depends on
the rights of the credentials you used.

### Immediate action

The three most severe deviations, ranked, with their subject. It exists so that the first
screen of a long report is already actionable.

### One block per control

<!-- pepin:gen scan-control-objectstore -->
```text
──────────────────────────────────────────────────────────────────────────────
 CRITICAL  ·  CLD-STO-1  ·  scaleway
 Stockage objet exposé publiquement
──────────────────────────────────────────────────────────────────────────────
  Total deviations: 1

  Details:
      CRIT  scaleway_object_bucket_acl.backups — Bucket « scaleway_object_bucket_acl.backups » accessible publiquement (ACL publique).

  Remediation
    Rendre le bucket privé (ACL private, retrait du grant AllUsers, suppression de la policy publique) ; servir via des URLs pré-signées si nécessaire.

  ↳ docs: https://stephane-robert.info/scsl/CLD-STO-1
```
<!-- /pepin:gen scan-control-objectstore -->

Read the header line first:

- **severity** — `critical`, `high`, `medium`, `low`, from the reference.
- **`CLD-STO-1`** — the **control** identifier, that is the frozen SCSL requirement. The rules
  themselves emit an agnostic check name (`objectstorage_bucket_public_access`); the scan
  resolves it against `referentiel/controles.yaml` and keeps the check name in `labels.check`.
- **`scaleway`** — the tier, taken from the resource itself (`labels.provider`), never
  hardcoded in a rule.
- the title, then `Total deviations`, then one `Details` line per offending subject.

`Details` is the **evidence**: the resource that failed, and the observed fact about it. Then
the **remediation**, and a `docs:` link to the SCSL page of the requirement.

### Controls table and summary

The table recaps code, title, severity, tier and the number of deviations. The summary carries
the **verdict** and the counts per severity.

The verdict phrasing is load-bearing. On a Terraform plan it reads *"périmètre déclaré (plan
Terraform, état planifié)"*; on a scan where nothing was measured it reads
`Verdict : INDÉTERMINÉ`, and the exit code follows.

### Exit code

```bash
echo $?   # 1
```

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

## The same run, as an assessment

The terminal report shows deviations. The **assessment** shows every control, including the
ones that did *not* deviate — which is what an auditor asks for.

```bash
./pepin scan scaleway --terraform examples/scaleway/terraform/plan.json --format assessment
```

### Provenance

<!-- pepin:gen assessment-run -->
```json
{
  "run": {
    "ruleset": {
      "digest": "\u003cempreinte règles + descripteurs + référentiel\u003e",
      "name": "pepin-config"
    },
    "scope": {
      "included": [
        "compute_instance",
        "governance_provider",
        "iam_policy",
        "managed_database",
        "object_storage_bucket",
        "security_group",
        "security_group_rule"
      ]
    },
    "source": "terraform-plan",
    "target": {
      "id": "scaleway",
      "platform": "scaleway",
      "provider": "scaleway"
    },
    "timestamp": "\u003chorodatage RFC3339 du scan\u003e",
    "tool": {
      "digest": "\u003cempreinte du binaire\u003e",
      "name": "pepin",
      "version": "\u003cversion\u003e"
    }
  }
}
```
<!-- /pepin:gen assessment-run -->

*(timestamp, version and digests are volatile by nature and are shown as placeholders here;
everything else is the captured document.)*

- `tool.digest` pins the **binary** that produced the verdict (VCS revision, `+modified` when
  the tree was dirty).
- `ruleset.digest` pins **everything that determines the result**: the embedded Rego rules, the
  provider descriptors, the reference, and the content of any `--policy-dir`.
- `scope.included` lists the resource types actually evaluated. A control about a type absent
  from that list cannot have been measured, and the assessment says so.

### The four statuses, on this one run

<!-- pepin:gen assessment-counts -->
| Status | Count |
|---|---:|
| `pass` | 6 |
| `fail` | 11 |
| `not-applicable` | 2 |
| `not-evaluated` | 8 |
<!-- /pepin:gen assessment-counts -->

A single account-free scan produces all four. Each is documented in
[the assessment model](../concepts/assessment-model.md); here is one of each, verbatim.

**`fail`** — a deviation, with its subject and evidence:

<!-- pepin:gen assessment-fail -->
```json
{
  "control": "objectstorage_bucket_public_access",
  "evidence": {
    "observed": "Bucket « scaleway_object_bucket_acl.backups » accessible publiquement (ACL publique).",
    "proves": [
      "",
      "",
      ""
    ],
    "source": "terraform-plan"
  },
  "labels": {
    "category": "security",
    "provider": "scaleway"
  },
  "references": [
    {
      "framework": "scsl",
      "id": "CLD-STO-1"
    },
    {
      "framework": "cis-v8",
      "id": "3.3"
    },
    {
      "framework": "iso-27001",
      "id": "A.5.15"
    },
    {
      "framework": "iso-27001",
      "id": "A.8.3"
    },
    {
      "framework": "iso-27017",
      "id": "CLD.9.5.1"
    },
    {
      "framework": "secnumcloud-3.2",
      "id": "9.7"
    },
    {
      "framework": "secnumcloud-3.2",
      "id": "13.2"
    }
  ],
  "remediation": "Rendre le bucket privé (ACL private, retrait du grant AllUsers, suppression de la policy publique) ; servir via des URLs pré-signées si nécessaire.",
  "severity": "critical",
  "status": "fail",
  "subject": "scaleway_object_bucket_acl.backups",
  "title": "Stockage objet exposé publiquement"
}
```
<!-- /pepin:gen assessment-fail -->

**`pass`** — note that `evidence.observed` names *what* was checked:

<!-- pepin:gen assessment-pass -->
```json
{
  "control": "network_securitygroup_allow_ingress_from_internet_to_all_ports",
  "evidence": {
    "observed": "aucune non-conformité détectée sur les ressources de type « security_group_rule » collectées (contrat vérifié)",
    "proves": [
      "",
      "",
      ""
    ],
    "source": "terraform-plan"
  },
  "references": [
    {
      "framework": "scsl",
      "id": "CLD-NET-2"
    },
    {
      "framework": "cis-v8",
      "id": "4.4"
    },
    {
      "framework": "cis-v8",
      "id": "12.2"
    },
    {
      "framework": "iso-27001",
      "id": "A.8.20"
    },
    {
      "framework": "iso-27001",
      "id": "A.8.22"
    },
    {
      "framework": "secnumcloud-3.2",
      "id": "13.2"
    }
  ],
  "severity": "critical",
  "status": "pass",
  "subject": "scaleway",
  "title": "Tout le trafic entrant autorisé depuis Internet (any/any)"
}
```
<!-- /pepin:gen assessment-pass -->

**`not-applicable`** — with the justification recorded in the provider contract:

<!-- pepin:gen assessment-na -->
```json
{
  "control": "blockstorage_volume_encryption",
  "references": [
    {
      "framework": "scsl",
      "id": "CLD-CHF-2"
    },
    {
      "framework": "iso-27001",
      "id": "A.8.24"
    },
    {
      "framework": "secnumcloud-3.2",
      "id": "10.1"
    }
  ],
  "severity": "high",
  "status": "not-applicable",
  "subject": "scaleway",
  "title": "Chiffrement au repos désactivé",
  "waiver": {
    "justification": "Chiffrement au repos des volumes block côté invité (LUKS/Cryptsetup), responsabilité du client (responsabilité partagée) ; l'API block n'expose aucun champ de chiffrement → non observable côté plateforme (CHF-2)."
  }
}
```
<!-- /pepin:gen assessment-na -->

**`not-evaluated`** — with the exact reason Pépin could not decide:

<!-- pepin:gen assessment-ne -->
```json
{
  "control": "compute_instance_public_ip_with_open_securitygroup",
  "evidence": {
    "observed": "attribut « public_ip » non collecté sur les ressources de type « compute_instance » (garde de capacité)",
    "proves": [
      "",
      "",
      ""
    ],
    "source": "terraform-plan"
  },
  "references": [
    {
      "framework": "scsl",
      "id": "CLD-NET-3"
    },
    {
      "framework": "cis-v8",
      "id": "4.4"
    },
    {
      "framework": "cis-v8",
      "id": "12.2"
    },
    {
      "framework": "iso-27001",
      "id": "A.8.20"
    },
    {
      "framework": "iso-27001",
      "id": "A.8.22"
    },
    {
      "framework": "iso-27017",
      "id": "CLD.9.5.2"
    },
    {
      "framework": "secnumcloud-3.2",
      "id": "13.2"
    }
  ],
  "severity": "critical",
  "status": "not-evaluated",
  "subject": "scaleway",
  "title": "Machine exposée publiquement sans filtrage restrictif"
}
```
<!-- /pepin:gen assessment-ne -->

## The chain, end to end

Take the object storage failure and follow it backwards through the repository:

| Link | Artifact | Content |
|---|---|---|
| **finding** | terminal report / assessment `fail` | `scaleway_object_bucket_acl.backups`, publicly readable |
| **evidence** | `examples/scaleway/terraform/plan.json` | `scaleway_object_bucket_acl.backups` declares `acl = "public-read"` |
| **projection** | `providers/scaleway.yaml`, `mapping_terraform` | `scaleway_object_bucket_acl` → type `object_storage_bucket`, attribute `acl` |
| **rule** | `internal/commonrules/rules/objectstorage_bucket_public_access.rego` | one common rule, `labels.provider` read from the resource |
| **control** | `referentiel/controles.yaml` | code `objectstorage_bucket_public_access`, severity `critical` |
| **normative references** | same entry | SCSL `CLD-STO-1`, plus SecNumCloud / CIS / ISO mappings |
| **remediation** | the rule's `remediation` field | make the bucket private, drop the `AllUsers` grant, use pre-signed URLs |

Each link is a file in this repository, and each is checked: `mise run validate` refuses a
control without a rule, a code emitted but not catalogued, or an SCSL reference absent from
the frozen index.

## Sealing it

```bash
./pepin scan scaleway --terraform examples/scaleway/terraform/plan.json --seal ./bundle
./pepin verify ./bundle
```

The bundle holds the evaluated inventory, the assessment, its OSCAL 1.1.2 rendering, a digested
manifest and a checksums file, so a third party can re-check integrity — and, with
`--re-derive`, re-run the evaluation on the sealed inventory.

**Without `--redact`, `input.json` embeds the raw inventory**: user-data, IAM policy documents,
policies. Pépin warns about it on stderr. Treat an unredacted bundle as sensitive, or produce a
shareable one with `--redact` (which replaces sensitive values by digests, and is therefore
incompatible with `verify --re-derive`).

## Other formats

| Flag | For |
|---|---|
| `--format table` | humans (the default, shown above) |
| `--format json` | `{"findings": …, "summary": …}` — scripts |
| `--format assessment` | the typed, referenced, provenanced document |
| `--format oscal` | OSCAL 1.1.2 assessment results, schema-validated in CI |
| `--format sarif` | code scanning (GitHub, GitLab) |

## See also

- [Quickstart](quickstart.md) — the same example, with a fix and a second scan.
- [The assessment model](../concepts/assessment-model.md) — what each status asserts.
- [Coverage matrix](../coverage.md) — what is measurable, per provider and per source.
- [Scope and non-goals](../concepts/scope.md) — what this report does not prove.

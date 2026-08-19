> 🇬🇧 English · [🇫🇷 Français](quickstart.fr.md)

# Quickstart — five minutes, no cloud account

This page takes you from nothing to a scan you can trust: an intentionally misconfigured
Terraform module, a real failure, the fix, and a second scan that returns a different verdict.

**No Scaleway, Outscale or Exoscale account is required.** Nothing is provisioned. Pépin reads
a Terraform *plan* — the state your code declares — and never talks to a cloud API here.

Every command below is executed by the documentation generator on the repository, and every
output shown is the output it captured. See [How this page stays true](#how-this-page-stays-true).

## 1. Get Pépin

Released binaries (checksummed, signed, with SLSA provenance), the container image, the GitHub
action and the GitLab template are documented in [install.md](../install.md). From source:

```bash
git clone https://github.com/stephrobert/pepin && cd pepin
go build -o pepin .          # Go 1.26+
./pepin provider list
```

<!-- pepin:gen provider-list -->
```text

// pépin  providers enregistrés
  exoscale  Exoscale (CH) — instances, security groups, block storage, SKS, SOS
  kubernetes  Kubernetes (in-cluster) — RBAC, Pod Security Standards, NetworkPolicy
  outscale  Outscale (3DS) — VM, BSU, OOS, EIM, security groups, OKS, LBU
  scaleway  Scaleway — object storage, instances, IAM, security groups
```
<!-- /pepin:gen provider-list -->

`kubernetes` is not a cloud: it audits the state *inside* a cluster. The three sovereign clouds
are the ones this walkthrough uses.

## 2. The deliberately misconfigured example

`examples/scaleway/terraform/main.tf` ships a small Scaleway module that is wrong on purpose:
a `public-read` bucket, SSH open to `0.0.0.0/0`, a managed database with encryption off and
backups disabled, a secret in cloud-init, an IAM policy that can grant itself IAM rights.

Its plan is committed as `examples/scaleway/terraform/plan.json`, so you can scan it right
away. To regenerate it yourself (this downloads the Scaleway provider, but still creates
nothing):

```bash
cd examples/scaleway/terraform
terraform init && terraform plan -out tfplan && terraform show -json tfplan > plan.json
cd -
```

## 3. Scan it

```bash
./pepin scan scaleway --terraform examples/scaleway/terraform/plan.json
```

The report opens with the three most severe deviations, so the first screen is already
actionable:

<!-- pepin:gen scan-vulnerable-head -->
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
[…]
```
<!-- /pepin:gen scan-vulnerable-head -->

and closes with the per-control table and the verdict:

<!-- pepin:gen scan-vulnerable-tail -->
```text
[…]
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
<!-- /pepin:gen scan-vulnerable-tail -->

Between the two, one block per control explains the deviation and its remediation. The full
output is walked through line by line in
[Understanding a scan](understanding-a-scan.md).

```bash
echo $?   # 1 — at least one critical or high deviation
```

## 4. Read one FAIL

Take `CLD-CHF-2`, *managed database without encryption at rest*, severity `high`. Its block,
lifted from the run above:

<!-- pepin:gen scan-control-encryption -->
```text
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
```
<!-- /pepin:gen scan-control-encryption -->

Read it backwards, which is how it was produced:

1. **The evidence.** The plan declares `encryption_at_rest = false` on
   `scaleway_rdb_instance.insecure`.
2. **The projection.** The Scaleway descriptor (`providers/scaleway.yaml`) maps that Terraform
   attribute onto the common normalized attribute `encryption_at_rest` of the
   `managed_database` type.
3. **The rule.** The common rule `database_encryption_at_rest_enabled` fires on any
   `managed_database` whose `encryption_at_rest` is false — the same rule for every provider.
4. **The control.** `referentiel/controles.yaml` maps that agnostic code onto the frozen SCSL
   requirement `CLD-CHF-2`, and from there onto SecNumCloud, ISO and CIS.

The report shows `CLD-CHF-2` because the finding was resolved against the reference; the
agnostic check name is kept in `labels.check`.

## 5. Fix it

`examples/scaleway/terraform-fixed/main.tf` is the same module with every deviation corrected.
For the database, the fix is two lines:

```diff
 resource "scaleway_rdb_instance" "insecure" {
   name               = "pepin-test-rdb"
   node_type          = "DB-DEV-S"
   engine             = "PostgreSQL-15"
   user_name          = "admin"
   password           = random_password.db.result
-  encryption_at_rest = false
-  disable_backup     = true
+  encryption_at_rest = true
+  disable_backup     = false
 }
```

The other corrections follow the same shape: `acl = "private"` instead of `"public-read"`,
`ip_range = "10.0.0.0/8"` on the SSH rule instead of an omitted range (which means *any*
origin), `inbound_default_policy = "drop"`, a cloud-init without a password, an IAM policy
without the `IAMManager` permission set, and the four governance tags on the instance.

## 6. Scan again

```bash
./pepin scan scaleway --terraform examples/scaleway/terraform-fixed/plan.json
```

<!-- pepin:gen scan-fixed-full -->
```text
──────────────────────────────────────────────────────────────────────────────
 Mode      scan scaleway (terraform)
 Source    examples/scaleway/terraform-fixed/plan.json
──────────────────────────────────────────────────────────────────────────────

  ✓ No deviations found in the audited scope.

──────────────────────────────────────────────────────────────────────────────
 Summary

 Verdict : conforme sur le périmètre déclaré (plan Terraform, état planifié) (aucune non-conformité détectée, 16 contrôles conformes)

 🔴 CRITICAL 0   🟠 HIGH 0   🟡 MEDIUM 0   🔵 LOW 0
──────────────────────────────────────────────────────────────────────────────
```
<!-- /pepin:gen scan-fixed-full -->

```bash
echo $?   # 0
```

The verdict wording matters. It says **"compliant on the declared scope (Terraform plan,
planned state)"**, not "compliant". A plan describes what your code *intends* to create; only
a live scan (`--live`) observes what actually runs. And it names the number of controls that
passed, because a scan that measured nothing must never look like a scan that found nothing.

## 7. Exit codes, observed

Pépin is meant to be a CI gate, so its exit codes are part of the contract. Each row below was
produced by running the command shown:

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

Two of these deserve attention:

- **`3` without `--strict`.** A scan that measured nothing returns 3, never 0. Expired
  credentials, insufficient rights, an empty region or a truncated inventory all produce the
  same empty result as a healthy tenant — and an empty result says nothing about posture.
- **`3` with `--strict`.** The stricter gate also refuses remaining medium/low deviations,
  which the default exit code ignores.

The two throwaway inventories used above are not repository fixtures — write them yourself.
`empty-inventory.json`, a tenant where nothing was collected:

<!-- pepin:gen fixture-empty-inventory -->
```json
{
  "provider": "scaleway",
  "resources": []
}
```
<!-- /pepin:gen fixture-empty-inventory -->

`tagless-inventory.json`, one instance missing its governance tags, which is a `medium`
deviation and nothing else:

<!-- pepin:gen fixture-tagless-inventory -->
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
        "tags": []
      }
    }
  ]
}
```
<!-- /pepin:gen fixture-tagless-inventory -->

## Where to go next

- [Understanding a scan](understanding-a-scan.md) — the full output, read line by line.
- [The assessment model](../concepts/assessment-model.md) — what `pass`, `fail`,
  `not-applicable` and `not-evaluated` actually assert.
- [Coverage matrix](../coverage.md) — what is measurable, per provider and per source.
- [Known limitations](../known-limitations.md) — what Pépin cannot see, and why.
- [Scope and non-goals](../concepts/scope.md) — what a Pépin report is *not*.

## How this page stays true

The command outputs above are not transcribed. `internal/docgen` runs the `pepin` binary on
the repository's own fixtures and injects what it captured between the `pepin:gen` markers.
`mise run gen-docs` rewrites them; `TestGeneratedDocsAreUpToDate` fails if what is committed
differs from what the binary produces today. When the product changes, this page breaks —
which is the point.

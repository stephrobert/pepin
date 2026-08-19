> [🇬🇧 English](github-actions.md) · 🇫🇷 Français

# Intégration GitHub Actions

Trois postures qu'un pipeline peut prendre, et ce guide les couvre toutes les trois :
**bloquer** une pull request sur le plan Terraform, **rapporter** les écarts dans Code Scanning
sans bloquer, et **surveiller** le tenant live périodiquement avec un bundle de preuve scellé.

Le workflow complet est committé dans `examples/github-actions/pepin.yml` et affiché en entier à
la fin de cette page. Il est contrôlé par `actionlint`.

## Installation : épingler l'action, et au minimum la v0.2.0

```yaml
- uses: stephrobert/pepin/.github/actions/pepin-scan@a02e42d054dc9d8a5a41ece5b46f6c1111659a70 # v0.2.0
  with:
    version: '0.2.0'
```

Deux épinglages, et ils ne font pas double emploi. Le **`@<sha>`** décide quel code d'action
s'exécute ; l'**entrée `version:`** décide quel binaire publié il installe. Une référence mutable
(`@main`, `@v1`) exécute ce que cette référence dira demain, c'est-à-dire exactement la forme de
supply chain que ce dépôt consacre un workflow de release à ne pas avoir.

> **La v0.2.0 est le minimum.** En v0.1.0 et v0.1.1, l'installateur de l'action appelait
> `gh attestation verify` sans jeton. L'API GitHub refuse cet appel dans un workflow,
> l'installateur y voyait une provenance non vérifiée, et **toute** installation était refusée.
> L'action fournit désormais `github.token` elle-même. Épingler un tag antérieur installe une
> action incapable d'installer quoi que ce soit.

### Ce que l'action vérifie avant d'exécuter un binaire

Pas seulement une empreinte. Dans l'ordre :

1. **La provenance.** `gh attestation verify --repo stephrobert/pepin --signer-workflow
   stephrobert/pepin/.github/workflows/release.yml` : le binaire doit porter une attestation de
   build produite par *ce* workflow, dans *ce* dépôt. Comparer un téléchargement à une liste de
   sommes publiée à côté de lui prouve seulement que deux fichiers de même origine concordent ;
   qui peut remplacer l'un remplace l'autre. C'est l'attestation qui tranche la question de
   l'auteur.
2. **L'intégrité.** Le SHA-256 de l'artefact téléchargé est ensuite comparé au `checksums.txt`
   de la release.
3. Alors seulement le binaire devient exécutable et rejoint le `PATH`.

Si `gh` est absent du runner (il est fourni sur les runners hébergés par GitHub), l'installateur
avertit bruyamment que la provenance n'a pas été vérifiée et retombe sur l'empreinte seule. Un
job de CI de ce dépôt corrompt un octet du téléchargement et exige que la vérification le
refuse.

## Bloquer une pull request sur le plan Terraform

Aucun identifiant, rien de provisionné, rien de facturé : `terraform plan` est rendu en JSON et
audité. C'est le job à rendre obligatoire dans un ruleset de branche.

```yaml
permissions: {}

jobs:
  terraform-plan:
    runs-on: ubuntu-24.04
    permissions:
      contents: read
    steps:
      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
        with:
          persist-credentials: false
      - uses: hashicorp/setup-terraform@dfe3c3f87815947d99a8997f908cb6525fc44e9e # v4.0.1
        with:
          terraform_wrapper: false
      - run: |
          terraform init
          terraform plan -out=tfplan
          terraform show -json tfplan > plan.json
      - uses: stephrobert/pepin/.github/actions/pepin-scan@a02e42d054dc9d8a5a41ece5b46f6c1111659a70 # v0.2.0
        with:
          version: '0.2.0'
          provider: scaleway
          terraform-plan: plan.json
```

`permissions: {}` au niveau du workflow, puis le minimum dont chaque job a besoin : celui-ci lit
le dépôt, et rien d'autre. `persist-credentials: false` garde le jeton du checkout hors de la
configuration git, où une étape ultérieure pourrait le réutiliser.

## Les codes de sortie, traités explicitement

C'est là qu'une porte de posture se gagne ou se perd. L'action projette le contrat ainsi :

| Code | Signification | `fail-on-nonconformity: 'true'` (défaut) | `'false'` |
|:-:|---|---|---|
| **0** | conforme | job en succès | job en succès |
| **1** | écart critical/high | **job en échec**, `::error::` | `::warning::`, job en succès |
| **3** | rien mesuré, ou medium/low sous `--strict` | **job en échec**, `::error::` | `::warning::`, job en succès |
| **2** | erreur technique | **job en échec**, `::error::` | **job en échec**, jamais dégradé |

**2 n'est jamais dégradé, par construction.** « Le tenant n'est pas conforme » est un verdict ;
« le scan n'a pas pu conclure » est une panne de la mesure, et un pipeline qui l'avale rapporte
une posture que personne n'a mesurée. La même règle vaut si l'on exécute le binaire directement
plutôt que l'action :

```yaml
- name: Bloquer sur la posture, en gardant 2 distinguable
  run: |
    pepin scan scaleway --terraform plan.json
    code=$?
    case "$code" in
      0) echo "::notice::conforme" ;;
      1|3) echo "::error::non conforme (code $code)" ; exit 1 ;;
      2) echo "::error::pepin n'a pas pu conclure (erreur technique)" ; exit 2 ;;
      *) echo "::error::code inattendu $code" ; exit 2 ;;
    esac
```

Jamais de `continue-on-error: true` sur l'étape de scan, jamais de `|| true` dans le script :
les deux effacent la distinction pour laquelle le tableau ci-dessus existe. Pour rapporter sans
bloquer, utiliser `fail-on-nonconformity: 'false'`, qui garde 2 fatal.

## Publier les écarts dans Code Scanning

SARIF est le format que lit l'onglet Code Scanning. Le téléversement demande exactement une
permission de plus.

```yaml
    permissions:
      contents: read
      security-events: write   # exigée par upload-sarif, et par rien d'autre

    steps:
      - uses: stephrobert/pepin/.github/actions/pepin-scan@a02e42d054dc9d8a5a41ece5b46f6c1111659a70 # v0.2.0
        with:
          version: '0.2.0'
          provider: scaleway
          terraform-plan: plan.json
          format: sarif
          output-file: pepin.sarif
          fail-on-nonconformity: 'false'
      - uses: github/codeql-action/upload-sarif@5595ccaf912efad79be6eef63a5619ff05969be3 # v4.37.6
        with:
          sarif_file: pepin.sarif
          category: pepin
```

`category: pepin` garde ces alertes dans leur propre espace, pour qu'un autre outil qui
téléverse du SARIF dans le même dépôt ne ferme pas les alertes de Pépin, et réciproquement. Les
alertes se posent sur le **fichier scanné**, le plan, pas sur une ligne d'un fichier `.tf` : une
ressource normalisée ne porte pas la ligne dont elle vient
([Formats de sortie](../reference/output-formats.fr.md#sarif--pour-github-code-scanning)).

Code Scanning est une surface de *rapport*. L'associer à `fail-on-nonconformity: 'false'` et
garder la décision bloquante dans le job de porte, sinon le même écart fera échouer la
construction deux fois.

## Scanner le tenant live

Un plan dit ce que le code déclare. Seul un scan live voit ce qui tourne réellement, y compris
ce que personne n'a écrit en Terraform
([Plan Terraform contre scan live](../concepts/terraform-vs-live.fr.md)). À lancer
périodiquement, pas sur chaque pull request.

```yaml
    env:
      SCW_ACCESS_KEY: ${{ secrets.SCW_ACCESS_KEY }}
      SCW_SECRET_KEY: ${{ secrets.SCW_SECRET_KEY }}
      SCW_DEFAULT_ORGANIZATION_ID: ${{ secrets.SCW_DEFAULT_ORGANIZATION_ID }}
      SCW_DEFAULT_REGION: fr-par
```

**Les identifiants ne sont jamais des entrées de l'action.** Pépin lit les variables
d'environnement natives de chaque fournisseur, et les faire transiter par des entrées
n'ajouterait qu'un endroit de plus où elles peuvent fuir :

| Fournisseur | Variables |
|---|---|
| Scaleway | `SCW_ACCESS_KEY`, `SCW_SECRET_KEY`, `SCW_DEFAULT_ORGANIZATION_ID`, `SCW_DEFAULT_REGION` |
| Outscale | `OSC_ACCESS_KEY`, `OSC_SECRET_KEY`, `OSC_REGION` |
| Exoscale | `EXOSCALE_API_KEY`, `EXOSCALE_API_SECRET`, `EXOSCALE_ZONE` |

Utiliser une clé en **lecture seule**, réduite au plus petit ensemble de permissions couvrant ce
que Pépin appelle : chaque page de fournisseur liste ces appels et les droits qu'ils exigent,
[Scaleway](../providers/scaleway.fr.md#permissions-minimales-pour-un-scan-live),
[Outscale](../providers/outscale.fr.md#permissions-minimales-pour-un-scan-live),
[Exoscale](../providers/exoscale.fr.md#permissions-minimales-pour-un-scan-live).

Une clé aux droits insuffisants ne produit pas un faux vert : ce qui n'a pas pu être collecté
revient en `not-evaluated`, et un scan qui n'a rien mesuré rend **3**
([Codes de sortie](../reference/exit-codes.fr.md)).

## Archiver le bundle de preuve

```yaml
      - uses: actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a # v7.0.1
        with:
          name: pepin-evidence
          path: |
            report.json
            bundle/
          retention-days: 7
```

**Un bundle nomme chaque ressource du tenant.** L'action passe `--redact` sauf si l'on pose
`redact: 'false'`, donc les données utilisateur et les documents de politique sont remplacés par
leur empreinte, mais l'inventaire lui-même, ressource par ressource, y est toujours. Garder une
rétention courte, et se rappeler que **les artefacts d'un dépôt public sont téléchargeables par
quiconque a un accès en lecture**. Ne renoncer à `--redact` que si le bundle doit supporter
`verify --re-derive`, et traiter alors l'artefact comme un secret
([Le bundle de preuve](evidence-bundles.fr.md)).

## Épingler la langue

```yaml
env:
  PEPIN_LANG: en
```

Codes de sortie, codes de contrôle, sévérités et statuts sont identiques dans les deux langues :
une porte n'a donc rien à épingler. Mais un job qui compare le *texte* d'un rapport entre deux
exécutions, ou qui archive des bundles et compare leurs empreintes, doit épingler `PEPIN_LANG` :
un runner dont la locale change produirait sinon une différence que personne n'a causée.

## Le workflow complet

À copier dans `.github/workflows/`. Tout y est épinglé par SHA de commit, et les trois jobs sont
les trois postures sur lesquelles cette page s'ouvre.

<!-- pepin:gen example-github-workflow -->
```yaml
# Gate a pipeline on the posture of a sovereign cloud — Pépin in GitHub Actions.
#
# Copy this into `.github/workflows/` of a repository holding a Terraform
# configuration for Exoscale, Outscale or Scaleway. The first two jobs need no
# credential at all: they audit the *plan*, so nothing is provisioned and there
# is no secret to create. The third shows the live variant, disabled by
# default, because it reads a real tenant with real credentials.
#
# The exit codes are the contract: 0 compliant, 1 non-compliance, 2 technical
# error, 3 nothing measured (or medium/low under --strict). The action fails
# the job on 1 and 3 unless you set `fail-on-nonconformity: 'false'`, and it
# fails on 2 whatever you set: a swallowed technical error would report a
# posture nobody measured.
#
# Everything is pinned by commit SHA, this repository's own action included.
# A mutable reference (@main, @v1) runs whatever that reference says tomorrow.
#
# Pin at least v0.2.0 of the action: in v0.1.0 and v0.1.1 its installer called
# `gh attestation verify` without a token, which refused EVERY installation.

name: cloud posture

on:
  pull_request:
  schedule:
    - cron: '17 4 * * 1'   # the live job below, once a week
  workflow_dispatch:

permissions: {}

env:
  # Codes, identifiers, severities and exit codes are identical in both
  # languages; titles, messages and remediations are not. A pipeline that
  # compares report TEXT between runs pins the language. One that reads exit
  # codes does not have to, and pinning costs nothing.
  PEPIN_LANG: en

jobs:
  # ---------------------------------------------------------------- blocking
  # No account, no secret, nothing billed: the plan is audited, not the cloud.
  # This job GATES: a critical/high deviation fails the pull request.
  terraform-plan:
    name: audit the Terraform plan (blocking)
    if: github.event_name == 'pull_request'
    runs-on: ubuntu-24.04
    timeout-minutes: 10
    permissions:
      contents: read
    steps:
      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
        with:
          persist-credentials: false

      - uses: hashicorp/setup-terraform@dfe3c3f87815947d99a8997f908cb6525fc44e9e # v4.0.1
        with:
          terraform_wrapper: false

      # `terraform plan` against these providers works offline once `init` has
      # the provider binaries; no credential is needed to *render* a plan for
      # most resources. Adjust to your own layout.
      - name: Render the plan as JSON
        run: |
          terraform init
          terraform plan -out=tfplan
          terraform show -json tfplan > plan.json

      - name: Pépin — the posture gate
        uses: stephrobert/pepin/.github/actions/pepin-scan@a02e42d054dc9d8a5a41ece5b46f6c1111659a70 # v0.2.0
        with:
          version: '0.2.0'
          provider: scaleway
          terraform-plan: plan.json

  # ------------------------------------------------------------- report only
  # The same audit, without gating: the verdict is reported and published to
  # Code Scanning, and only a TECHNICAL error fails the job.
  terraform-plan-report:
    name: audit the Terraform plan (report only)
    if: github.event_name == 'pull_request'
    runs-on: ubuntu-24.04
    timeout-minutes: 10
    permissions:
      contents: read
      security-events: write   # required by upload-sarif, and by nothing else
    steps:
      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
        with:
          persist-credentials: false

      - uses: hashicorp/setup-terraform@dfe3c3f87815947d99a8997f908cb6525fc44e9e # v4.0.1
        with:
          terraform_wrapper: false

      - name: Render the plan as JSON
        run: |
          terraform init
          terraform plan -out=tfplan
          terraform show -json tfplan > plan.json

      - name: Pépin — report the posture as SARIF
        id: scan
        uses: stephrobert/pepin/.github/actions/pepin-scan@a02e42d054dc9d8a5a41ece5b46f6c1111659a70 # v0.2.0
        with:
          version: '0.2.0'
          provider: scaleway
          terraform-plan: plan.json
          format: sarif
          output-file: pepin.sarif
          fail-on-nonconformity: 'false'   # 1 and 3 warn; 2 still fails the job

      # SARIF is the format the Code Scanning tab reads. Alerts land on the
      # scanned file (the plan), because a normalized resource does not carry
      # the line of the .tf file it came from.
      - name: Publish the findings to Code Scanning
        uses: github/codeql-action/upload-sarif@5595ccaf912efad79be6eef63a5619ff05969be3 # v4.37.6
        with:
          sarif_file: pepin.sarif
          category: pepin

      # The three codes, treated explicitly. 2 has already failed the step
      # above, whatever `fail-on-nonconformity` says; this is what a reader of
      # the job summary sees.
      - name: Read the verdict
        env:
          CODE: ${{ steps.scan.outputs.exit-code }}
          VERDICT: ${{ steps.scan.outputs.verdict }}
        run: |
          case "${CODE}" in
            0) echo "::notice::compliant (${VERDICT})" ;;
            1) echo "::warning::non-compliance: at least one critical/high deviation" ;;
            3) echo "::warning::nothing measured, or medium/low deviations under --strict" ;;
            *) echo "::error::pepin could not conclude (exit ${CODE})" ; exit 1 ;;
          esac

  # -------------------------------------------------------------------- live
  # Reads the real tenant. Credentials come from repository secrets, through
  # `env:`, in the provider's own variable names — never as action inputs,
  # never in the repository. Pépin does not print them, but the evidence
  # bundle embeds the tenant's INVENTORY: keep `redact` at its default, keep
  # the artifact retention short, and remember that artifacts of a public
  # repository are downloadable by anyone with read access.
  live-scan:
    name: scan the live tenant
    if: github.event_name == 'schedule' || github.event_name == 'workflow_dispatch'
    runs-on: ubuntu-24.04
    timeout-minutes: 15
    permissions:
      contents: read
    env:
      SCW_ACCESS_KEY: ${{ secrets.SCW_ACCESS_KEY }}
      SCW_SECRET_KEY: ${{ secrets.SCW_SECRET_KEY }}
      SCW_DEFAULT_ORGANIZATION_ID: ${{ secrets.SCW_DEFAULT_ORGANIZATION_ID }}
      SCW_DEFAULT_REGION: fr-par
    steps:
      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
        with:
          persist-credentials: false

      - name: Pépin — live scan, report only, sealed evidence
        id: scan
        uses: stephrobert/pepin/.github/actions/pepin-scan@a02e42d054dc9d8a5a41ece5b46f6c1111659a70 # v0.2.0
        with:
          version: '0.2.0'
          provider: scaleway
          live: 'true'
          format: json
          output-file: report.json
          seal: bundle
          fail-on-nonconformity: 'false'   # report the posture, gate elsewhere

      # The bundle is the auditable proof; it names every resource of the
      # tenant. Short retention, and redacted by default (the action passes
      # --redact unless told otherwise). A bundle meant for a third party
      # verifies with `pepin verify` against its cosign signature.
      - uses: actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a # v7.0.1
        with:
          name: pepin-evidence
          path: |
            report.json
            bundle/
          retention-days: 7

      - name: Read the verdict without failing the job
        env:
          VERDICT: ${{ steps.scan.outputs.verdict }}
          CODE: ${{ steps.scan.outputs.exit-code }}
        run: |
          echo "posture: ${VERDICT} (exit ${CODE})"
```
<!-- /pepin:gen example-github-workflow -->

## Aide-mémoire

- [ ] chaque `uses:` épinglé par SHA de commit, cette action comprise : pas de `@main`, pas de
      `@v1`
- [ ] `version:` de l'action à `0.2.0` ou plus
- [ ] `permissions: {}` au niveau du workflow, moindre privilège par job
- [ ] `security-events: write` uniquement sur le job qui téléverse le SARIF
- [ ] aucun `continue-on-error` ni `|| true` sur l'étape de scan
- [ ] `fail-on-nonconformity: 'false'` sur les jobs de rapport, jamais un contournement global
- [ ] identifiants en `env:` depuis les secrets, sous les noms natifs du fournisseur
- [ ] identifiants en lecture seule pour le scan live
- [ ] artefact du bundle de preuve : rétention courte, caviardé, pas sur un dépôt public
- [ ] `PEPIN_LANG` épinglé si quoi que ce soit compare du texte de rapport ou des empreintes de
      bundle

## Pour aller plus loin

- [Codes de sortie](../reference/exit-codes.fr.md) : le contrat sur lequel ce guide bloque.
- [Formats de sortie](../reference/output-formats.fr.md) : quoi téléverser, quoi archiver.
- [Le bundle de preuve](evidence-bundles.fr.md) : sceller, vérifier, partager.
- [GitLab CI](gitlab-ci.fr.md) : les trois mêmes postures, sur GitLab.
- [Installation](../install.fr.md) : vérifier une release à la main (cosign, provenance SLSA).

## Comment cette page reste vraie

Le workflow complet ci-dessus n'est pas une recopie : c'est le contenu de
`examples/github-actions/pepin.yml`, injecté par `internal/docgen`, pour que la page ne puisse
pas épingler un SHA différent de celui du fichier que le lecteur copie.
`TestGeneratedDocsAreUpToDate` échoue quand les deux divergent. Le workflow lui-même n'a pas été
exécuté par le générateur de documentation, un workflow ne s'exécute que sur GitHub, mais il est
validé par `actionlint`, et les promesses de l'action (vérification de provenance, refus d'un
binaire corrompu, projection des codes de sortie) sont éprouvées par le workflow `entrypoints`
de ce dépôt à chaque pull request.

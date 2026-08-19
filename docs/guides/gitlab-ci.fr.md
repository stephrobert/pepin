> [🇬🇧 English](gitlab-ci.md) · 🇫🇷 Français

# Intégration GitLab CI

Les trois mêmes postures que sur GitHub, bloquer une demande de fusion sur le plan Terraform,
rapporter sans bloquer, surveiller le tenant live, exprimées dans le vocabulaire de GitLab : un
template inclus, `allow_failure: exit_codes`, et des variables protégées.

Deux fichiers sont committés dans `examples/gitlab-ci/` et affichés en entier à la fin de cette
page : le template qu'on inclut, et un pipeline minimal qui s'en sert.

## Installation : un template inclus, épinglé à un tag

```yaml
include:
  - remote: 'https://raw.githubusercontent.com/stephrobert/pepin/v0.2.0/examples/gitlab-ci/pepin.gitlab-ci.yml'
```

**Épingler l'include à un tag.** Un include non épinglé (`.../main/...`) exécute ce que la
branche par défaut dira demain, dans le dépôt de quelqu'un d'autre, au moment où votre pipeline
démarre.

Le template télécharge le binaire publié et **vérifie son SHA-256 contre la liste de sommes de
la release avant que quoi que ce soit ne l'exécute** :

```yaml
  before_script:
    - wget -q "${PEPIN_BASE}/pepin-linux-amd64" "${PEPIN_BASE}/checksums.txt"
    - grep " pepin-linux-amd64$" checksums.txt | sha256sum -c -
    - chmod 0755 pepin-linux-amd64 && mv pepin-linux-amd64 /usr/local/bin/pepin
```

Un téléchargement corrompu arrête le job là, avec l'écart d'empreinte nommé dans le log. C'est
de l'intégrité, pas de l'authenticité : le binaire et la liste de sommes viennent de la même
origine, donc qui pourrait remplacer l'un remplacerait l'autre. La moitié forte de la chaîne, la
signature cosign de cette liste et l'attestation de provenance SLSA, exige `cosign` ou `gh` dans
l'image, et les commandes sont dans [Installation](../install.fr.md). Si votre image de runner
peut embarquer `cosign`, vérifiez aussi la signature dans le `before_script`.

La version tient dans une variable, donc une montée est une ligne :

```yaml
variables:
  PEPIN_VERSION: "0.2.0"
```

## Auditer un plan Terraform, sans aucun identifiant

Le job `plan` rend le plan ; le job Pépin l'audite. Rien n'est provisionné, aucun secret n'est
nécessaire, et le retour arrive sur la demande de fusion.

```yaml
stages: [plan, test]

plan:
  stage: plan
  image:
    name: hashicorp/terraform:1.15
    entrypoint: [""]
  script:
    - terraform init
    - terraform plan -out=tfplan
    - terraform show -json tfplan > plan.json
  artifacts:
    paths: [plan.json]
    expire_in: 1 day

pepin-terraform-plan:
  extends: .pepin
  stage: test
  script:
    - pepin scan scaleway --terraform plan.json --format json > pepin-report.json
  artifacts:
    when: always
    paths: [pepin-report.json]
    expire_in: 1 week
```

Tel quel, ce job **bloque** : tout code de sortie non nul fait échouer le pipeline.

## Les codes de sortie : `allow_failure: exit_codes: [1, 3]`, jamais 2

Pour rapporter la posture sans bloquer dessus, lister les codes de **verdict**, et eux seuls :

```yaml
pepin-terraform-plan:
  extends: .pepin
  allow_failure:
    exit_codes: [1, 3]
```

| Code | Signification | Job bloquant | `allow_failure: exit_codes: [1, 3]` |
|:-:|---|---|---|
| **0** | conforme | succès | succès |
| **1** | écart critical/high | **échec** | avertissement, le pipeline continue |
| **3** | rien mesuré, ou medium/low sous `--strict` | **échec** | avertissement, le pipeline continue |
| **2** | erreur technique | **échec** | **échec**, 2 n'est pas dans la liste |

**Ne jamais écrire `exit_codes: [1, 2, 3]`, ni `allow_failure: true`.** Les deux transforment
« le scan n'a pas pu conclure » en pipeline vert : API injoignable, identifiants expirés, plan
illisible, tout cela rapporté comme une posture que personne n'a mesurée. `allow_failure: true`
est la version brutale de la même erreur, puisqu'il couvre tous les codes d'un coup.

Noter que 3 est un verdict, pas une erreur, et qu'il mérite de l'attention plutôt que d'être
écarté : il signifie que l'exécution n'a rien mesuré (hors gouvernance), ce qui, sur un scan
live, pointe le plus souvent des identifiants ou des droits
([Codes de sortie](../reference/exit-codes.fr.md)).

## Scanner le tenant live

```yaml
pepin-live:
  extends: .pepin
  stage: test
  rules:
    - when: manual
  script:
    - pepin scan scaleway --live --format json --seal bundle --redact > pepin-report.json
  allow_failure:
    exit_codes: [1, 3]
```

Les identifiants viennent de variables CI/CD **masquées et protégées** (Settings → CI/CD →
Variables), sous les noms natifs du fournisseur, jamais dans le YAML, jamais affichés par une
ligne de script :

| Fournisseur | Variables |
|---|---|
| Scaleway | `SCW_ACCESS_KEY`, `SCW_SECRET_KEY`, `SCW_DEFAULT_ORGANIZATION_ID`, `SCW_DEFAULT_REGION` |
| Outscale | `OSC_ACCESS_KEY`, `OSC_SECRET_KEY`, `OSC_REGION` |
| Exoscale | `EXOSCALE_API_KEY`, `EXOSCALE_API_SECRET`, `EXOSCALE_ZONE` |

*Protégée* garde la variable hors des branches non protégées, où une demande de fusion venue de
n'importe où pourrait la lire. *Masquée* la garde hors des logs de job. Utiliser une clé en
**lecture seule**, réduite à ce que Pépin appelle réellement :
[Scaleway](../providers/scaleway.fr.md#permissions-minimales-pour-un-scan-live),
[Outscale](../providers/outscale.fr.md#permissions-minimales-pour-un-scan-live),
[Exoscale](../providers/exoscale.fr.md#permissions-minimales-pour-un-scan-live).

`rules: - when: manual` garde le scan live hors de chaque push. Un pipeline planifié (CI/CD →
Schedules) est l'autre déclencheur raisonnable ; un scan live à chaque commit lit un tenant réel
bien plus souvent que le tenant ne change.

## Archiver le bundle de preuve

```yaml
  artifacts:
    when: always
    paths:
      - pepin-report.json
      - bundle/
    expire_in: 1 week
    public: false
```

Quatre décisions, toutes délibérées :

- **`--redact`** dans le script : les données utilisateur et les documents de politique sont
  remplacés par leur empreinte. Le coût est qu'un bundle caviardé ne peut pas être re-dérivé
  ([Le bundle de preuve](evidence-bundles.fr.md#redact--un-bundle-quon-remet-à-un-tiers)).
- **`when: always`** : le rapport est archivé même quand le job échoue, ce qui est précisément
  l'exécution qu'on voudra lire.
- **`expire_in: 1 week`** : un bundle nomme chaque ressource du tenant. Ce n'est pas un artefact
  permanent de build ; s'il faut en conserver un, le déplacer vers un coffre de preuves avec
  contrôle d'accès.
- **non public** : garder l'artefact hors des pages de pipeline publiques.

## Épingler la langue

```yaml
variables:
  PEPIN_LANG: "en"
```

Les codes de sortie et les identifiants sont stables d'une langue à l'autre : une porte n'a rien
à épingler. Un job qui compare le *texte* d'un rapport entre deux exécutions, ou qui compare des
empreintes de bundle, doit épingler la langue : une image de runner dont la locale change
produirait sinon une différence que personne n'a causée.

## Le template complet

`examples/gitlab-ci/pepin.gitlab-ci.yml`, le fichier que vise l'`include` ci-dessus. Il embarque
les trois jobs : bloquant, rapport seul, et scan live manuel.

<!-- pepin:gen example-gitlab-template -->
```yaml
# Pépin in GitLab CI — an includable template.
#
#   include:
#     - remote: 'https://raw.githubusercontent.com/stephrobert/pepin/v0.2.0/examples/gitlab-ci/pepin.gitlab-ci.yml'
#
# then `extends: .pepin` in your own job (see .gitlab-ci.yml beside this file).
# The remote include is pinned to a tag on purpose: an unpinned include runs
# whatever the default branch says tomorrow.
#
# The released binary is downloaded and its SHA-256 verified against the
# release's checksum list BEFORE anything runs it. The stronger half of the
# chain (the cosign signature over that list, the SLSA provenance) needs
# cosign or gh; docs/install.md carries those commands.
#
# Exit codes are the contract: 0 compliant, 1 non-compliance, 2 technical
# error, 3 nothing measured (or medium/low under --strict). The default below
# fails the job on any non-zero code. To *report* the posture without gating
# on it, allow the verdict codes and only them — never 2, which means the scan
# could not conclude:
#
#   pepin-scan:
#     extends: .pepin
#     allow_failure:
#       exit_codes: [1, 3]

variables:
  PEPIN_VERSION: "0.2.0"
  # Codes, identifiers, severities and exit codes are identical in both
  # languages; titles, messages and remediations are not. A pipeline that
  # compares report TEXT between runs pins the language.
  PEPIN_LANG: "en"

.pepin:
  image: alpine:3.21
  variables:
    PEPIN_BASE: "https://github.com/stephrobert/pepin/releases/download/v${PEPIN_VERSION}"
  before_script:
    - wget -q "${PEPIN_BASE}/pepin-linux-amd64" "${PEPIN_BASE}/checksums.txt"
    # The bytes are checked before anything runs them; a corrupted download
    # stops the job here, with the checksum mismatch named in the log.
    - grep " pepin-linux-amd64$" checksums.txt | sha256sum -c -
    - chmod 0755 pepin-linux-amd64 && mv pepin-linux-amd64 /usr/local/bin/pepin

# Audit a Terraform plan: no credential, nothing provisioned. Expects a
# `plan.json` artifact from an earlier stage (`terraform show -json`).
# Override PEPIN_PROVIDER and the path to match your pipeline.
#
# This job GATES: any non-zero code fails the pipeline, and the three of them
# stay distinguishable in the job log and in its exit code.
pepin-terraform-plan:
  extends: .pepin
  stage: test
  variables:
    PEPIN_PROVIDER: scaleway
  script:
    - pepin scan "${PEPIN_PROVIDER}" --terraform plan.json --format json > pepin-report.json
  artifacts:
    when: always
    paths: [pepin-report.json]
    expire_in: 1 week

# The same audit, reporting instead of gating. `allow_failure.exit_codes`
# lists the VERDICT codes and only them: 2 is absent, so a technical error
# still fails the pipeline. Listing it would turn "the scan could not
# conclude" into a green pipeline.
pepin-terraform-plan-report:
  extends: .pepin
  stage: test
  variables:
    PEPIN_PROVIDER: scaleway
  script:
    - pepin scan "${PEPIN_PROVIDER}" --terraform plan.json --format json > pepin-report.json
  allow_failure:
    exit_codes: [1, 3]
  artifacts:
    when: always
    paths: [pepin-report.json]
    expire_in: 1 week

# Live scan of the real tenant — disabled until triggered by hand, because it
# reads a real account. Credentials come from masked, protected CI/CD
# variables in the provider's OWN names (Settings > CI/CD > Variables):
#
#   Scaleway  SCW_ACCESS_KEY, SCW_SECRET_KEY, SCW_DEFAULT_ORGANIZATION_ID,
#             SCW_DEFAULT_REGION
#   Outscale  OSC_ACCESS_KEY, OSC_SECRET_KEY, OSC_REGION
#   Exoscale  EXOSCALE_API_KEY, EXOSCALE_API_SECRET, EXOSCALE_ZONE
#
# Never write them in this file, and never echo them in a script line. Pépin
# reads them from the environment and does not print them.
#
# The evidence bundle embeds the tenant's inventory. It is sealed with
# --redact here (secrets like user-data and policy documents are replaced by
# their digest), the artifact expires after a week, and it is kept away from
# public pipeline pages. A bundle for a third party verifies with
# `pepin verify` against its cosign signature; only drop --redact when you
# need `verify --re-derive`, and then treat the bundle itself as a secret.
pepin-live:
  extends: .pepin
  stage: test
  rules:
    - when: manual
  variables:
    PEPIN_PROVIDER: scaleway
  script:
    - pepin scan "${PEPIN_PROVIDER}" --live --format json --seal bundle --redact > pepin-report.json
  allow_failure:
    exit_codes: [1, 3]
  artifacts:
    when: always
    paths:
      - pepin-report.json
      - bundle/
    expire_in: 1 week
    public: false
```
<!-- /pepin:gen example-gitlab-template -->

## Le pipeline qui s'en sert

`examples/gitlab-ci/.gitlab-ci.yml`, à copier dans votre dépôt Terraform.

<!-- pepin:gen example-gitlab-pipeline -->
```yaml
# Gate a GitLab pipeline on cloud posture with Pépin — minimal usage.
#
# Copy into a repository holding a Terraform configuration for Exoscale,
# Outscale or Scaleway. The `plan` job renders the plan; the included template
# audits it with no credential and nothing provisioned. The pipeline fails on
# a non-compliant posture (exit 1), on a scan that measured nothing (exit 3)
# and on a technical error (exit 2), and the three are distinguishable in the
# job log and in its exit code.

include:
  # Pinned to a tag: an unpinned include runs whatever the default branch
  # says tomorrow.
  - remote: 'https://raw.githubusercontent.com/stephrobert/pepin/v0.2.0/examples/gitlab-ci/pepin.gitlab-ci.yml'

stages: [plan, test]

plan:
  stage: plan
  image:
    name: hashicorp/terraform:1.15
    entrypoint: [""]
  script:
    - terraform init
    - terraform plan -out=tfplan
    - terraform show -json tfplan > plan.json
  artifacts:
    paths: [plan.json]
    expire_in: 1 day

# The template ships three jobs: `pepin-terraform-plan` (gating),
# `pepin-terraform-plan-report` (report only, allow_failure on the verdict
# codes 1 and 3 but never on 2) and `pepin-live` (manual). Keep the one you
# want and disable the others, for instance:
#
#   pepin-terraform-plan-report:
#     rules:
#       - when: never
```
<!-- /pepin:gen example-gitlab-pipeline -->

## Aide-mémoire

- [ ] `include: remote:` épinglé à un tag, jamais à une branche
- [ ] `PEPIN_VERSION` à `0.2.0` ou plus
- [ ] empreinte du binaire vérifiée avant exécution (le template le fait)
- [ ] `allow_failure: exit_codes: [1, 3]` sur les jobs de rapport, **jamais 2**, jamais
      `allow_failure: true`
- [ ] identifiants dans des variables CI/CD masquées et protégées, sous les noms natifs du
      fournisseur
- [ ] identifiants en lecture seule pour le scan live
- [ ] job live derrière `when: manual` ou un planificateur
- [ ] artefact du bundle : `--redact`, `expire_in` court, non public
- [ ] `PEPIN_LANG` épinglé si quoi que ce soit compare du texte de rapport ou des empreintes de
      bundle

## Pour aller plus loin

- [Codes de sortie](../reference/exit-codes.fr.md) : le contrat sur lequel `allow_failure`
  s'appuie.
- [Formats de sortie](../reference/output-formats.fr.md) : quoi archiver, et ce qui se parse
  sans risque.
- [Le bundle de preuve](evidence-bundles.fr.md) : ce qu'il y a dans un bundle, et comment le
  vérifier.
- [GitHub Actions](github-actions.fr.md) : les trois mêmes postures, sur GitHub.

## Comment cette page reste vraie

Les deux fichiers complets ci-dessus sont injectés depuis `examples/gitlab-ci/` par
`internal/docgen`, pour que la page ne puisse pas afficher un épinglage de version différent de
celui du fichier que le lecteur copie. `TestGeneratedDocsAreUpToDate` échoue quand ils
divergent. Les pipelines eux-mêmes n'ont pas été exécutés par le générateur de documentation,
ils ne tournent que sur GitLab, mais chaque commande `pepin` qu'ils contiennent est documentée,
avec sa sortie réelle et son code de sortie réel, dans
[Codes de sortie](../reference/exit-codes.fr.md) et
[Formats de sortie](../reference/output-formats.fr.md).

> [🇬🇧 English](CHANGELOG.md) · 🇫🇷 Français

# Journal des changements

Les changements notables, au format [Keep a Changelog](https://keepachangelog.com/fr/1.1.0/),
versionnés selon [Semantic Versioning](https://semver.org/).

Ce fichier est lu par le workflow de release : la section correspondant à un
tag devient le corps de sa GitHub Release. Une entrée absente d'ici est une
entrée qu'aucun lecteur téléchargeant un binaire ne verra jamais.

Deux sortes de changements méritent leur ligne quelle que soit leur taille,
parce que c'est sur elles qu'une chaîne de conformité bâtie sur Pépin est
jugée : **une surface qu'un pipeline consommateur parse** (la forme de
l'assessment, des findings, du bundle ou de l'OSCAL, un code de sortie, un
verbe ou un flag de la CLI), et **un verdict qui peut changer sur un tenant
inchangé** (une règle durcie ou assouplie, un contrôle activé ou retiré, un
mapping normatif retrié). La première casse leur parsing ; le second oblige
leur utilisateur à expliquer à un auditeur un changement qu'il n'a pas fait,
et c'est ici que cette explication commence. Un refactor qui ne change ni
l'une ni l'autre appartient au `git log`.

## [Unreleased]

### Ajouté

- **La surface publique est gelée par des tests, pas par de la prose.** Les
  verbes, flags et codes de sortie de la CLI, le document findings de
  `--format json`, le document assessment et la forme du bundle de preuve ont
  chacun une fixture committée sous `cmd/testdata/frozen/` : l'arbre de
  champs, jamais une valeur. Une forme qui bouge sans sa fixture fait échouer
  la CI ; une fixture régénérée sans que la version déclarée bouge aussi. La
  version du bundle voyage sur le fil comme suffixe `/vN` du champ `format`
  de `manifest.json` ; un vérificateur qui rencontre une version inconnue
  doit s'arrêter plutôt que deviner.
- **L'index SCSL est surveillé en dérive.** `mise run scsl-drift` compare
  l'index vivant de `framework-scsl` à une baseline committée dans
  `referentiel/scsl-baseline.json` et sort en 2 quand une exigence CLD a été
  ajoutée, retirée ou reformulée en amont sans qu'un humain retrie les
  mappings. La convention de sortie de l'outillage (0 ok, 1 erreur,
  **2 dérive**) est volontairement distincte de celle de `pepin scan` (où 2
  est l'erreur technique).
- **Une release se refuse avant le tag, pas se regrette après.**
  `mise run release-check -- vX.Y.Z` rejoue hors ligne tout ce qui doit
  tenir : arbre propre sur `main`, tag libre, tests et cohérence du
  référentiel, zéro dérive SCSL, les codes de sortie répondus par le binaire
  construit plutôt que lus dans une constante, un bundle scellé qui se
  vérifie, se re-dérive **et se refuse une fois altéré**, la version que les
  Conventional Commits impliquent (`.cz.toml`), et les deux CHANGELOG portant
  la section d'où le corps de la release se lit.
- **Un tag construit, atteste et signe la release.**
  `.github/workflows/release.yml` construit les binaires `linux`/`darwin` ×
  `amd64`/`arm64` avec le tag gravé dedans, génère les empreintes SHA-256 et
  un SBOM CycloneDX, enregistre la provenance de build SLSA, signe les
  empreintes en Cosign keyless, et publie la GitHub Release avec la section
  correspondante de ce fichier pour corps.
- **Une image de conteneur** (`ghcr.io/stephrobert/pepin`, un tag par
  release, pas de `latest`) : les binaires linux publiés sur une base
  distroless épinglée par digest, avec les certificats racines pour le TLS de
  `--live`, l'utilisateur 65532 et pas de shell. Rien n'est compilé dans le
  Dockerfile, donc les empreintes, le SBOM et la provenance de la release
  décrivent aussi le contenu de l'image ; l'image porte sa propre provenance
  SLSA, sa propre attestation de SBOM et une signature keyless, et la release
  refuse une image dont le `pepin version` n'est pas le tag ou dont les codes
  de sortie ont bougé à travers `docker run`.
- **Une action GitHub composite** (`.github/actions/pepin-scan`) qui vérifie
  le SHA-256 du binaire téléchargé contre la liste d'empreintes de la release
  avant de l'exécuter, scanne un plan Terraform, un inventaire ou l'API live,
  et traduit les codes de sortie en porte de CI :
  `fail-on-nonconformity: 'false'` rétrograde un verdict non conforme (1, ou
  3 sous strict) en avertissement, et ne rétrograde jamais une erreur
  technique (2). Les identifiants ne sont jamais des entrées de l'action ;
  les variables natives du provider passent par `env:`. La CI corrompt un
  octet du téléchargement et exige le refus (`entrypoints.yml`), et chaque
  release rejoue l'action contre ses propres artefacts publiés.
- **Un modèle GitLab CI et des exemples de CI**
  (`examples/gitlab-ci/`, `examples/github-actions/`) : même téléchargement
  vérifié, même contrat de codes de sortie, mode rapport via
  `allow_failure: exit_codes: [1, 3]`, jamais 2. L'installation et la
  vérification des quatre portes d'entrée sont documentées dans
  `docs/install.md` / `docs/install.fr.md`.

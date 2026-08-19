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

- **Vague 2 de la documentation produit : dix pages, générées partout où elles
  peuvent l'être.** Une référence CLI bâtie depuis la surface gelée et depuis de
  vraies exécutions de `--help`, le contrat des codes de sortie montré comme six
  exécutions avec le code que chacune a rendu, les cinq formats de sortie avec un
  document réel chacun, le plan contre le live avec deux divergences
  reproductibles, le cycle de vie du bundle de preuve (sceller, vérifier,
  re-dériver, altérer, caviarder) capturé de bout en bout, les intégrations
  GitHub Actions et GitLab CI dont les pipelines complets sont injectés depuis
  `examples/`, et une page par cloud souverain avec ses appels d'API et ses
  permissions minimales en lecture seule. `TestEveryPublicCLIFlagIsDocumented`
  échoue désormais quand un drapeau public manque à la référence CLI, dans l'une
  ou l'autre langue.

- Les exemples de CI publiés épinglent la **v0.2.0** et chaque action par SHA de
  commit. L'action des v0.1.0 et v0.1.1 n'installait rien du tout
  (`gh attestation verify` sans jeton) : ces tags ne doivent être épinglés par
  personne.

## [0.2.0] - 2026-08-19

### Ajouté

- **Pépin est bilingue, et détecte la langue.** Rapports, verdict, aide, erreurs
  et formats parsables (`json`, `sarif`, `oscal`, `assessment`) sortent en
  français ou en anglais. Ordre de résolution :
  `--lang=fr|en` → `PEPIN_LANG` → `LC_ALL` → `LANG` → repli `en` ; la première
  source non vide décide, et une locale inconnue retombe sur l'anglais sans
  erreur. Jusqu'ici l'ossature était anglaise et le contenu français : un lecteur
  recevait un rapport en deux langues dans la même phrase.
  Le français reste la langue de référence du contenu normatif : le référentiel
  et les règles s'écrivent en français d'abord, et là où une lecture juridique
  est en jeu, c'est la formulation française d'un contrôle qui fait foi.

- **Le projet a une marque.** `docs/assets/brand/` porte l'icône et les
  verrouillages en SVG et PNG, clair, sombre et monochrome, avec les générateurs
  qui les produisent (`scripts/generer-marque.py`,
  `scripts/generer-png-marque.py`) et les règles d'usage dans `docs/brand.fr.md`.
  Les deux README s'ouvrent dessus.

### Modifié

- **La prose d'un finding change avec la langue, ses clés non.** Codes (`CLD-*`),
  identifiants de check, sévérités, statuts, sujets et codes de sortie sont
  identiques dans les deux langues ; titres, messages, remédiations et preuves
  sont traduits. Un pipeline qui compare le *texte* d'un rapport d'une exécution
  à l'autre doit figer `PEPIN_LANG`. Un pipeline adossé aux codes et aux statuts
  n'est pas affecté.
- **Un bundle scellé porte la langue du scan qui l'a produit.**
  `verify --re-derive` rejoue les règles dans les deux langues et accepte la
  concordance de l'une : vérifier un bundle français depuis un shell anglais
  n'est plus signalé comme une falsification. À noter : l'empreinte du bundle
  dépend bien de la langue, puisque la prose de l'assessment fait partie de ce
  qui est scellé.
- **Surface CLI v1 → v2** : ajout du drapeau persistant `--lang`. Ajout pur :
  aucun verbe, aucun autre drapeau ni aucun code de sortie ne bouge.


- **`docs/doc-cache-brief.md` sort de la documentation produit.** C'était un
  mémo de mainteneur s'adressant à une machine (« déjà téléchargée sur cette
  machine »), décrivant un cache qu'un clone ne peut pas avoir, lié de nulle
  part, et portant six chemins absolus vers un répertoire personnel. Ce qu'il
  contenait de précieux rejoint `references/docs/README.md`, à côté du
  `sources.yaml` qu'il décrit, dont le piège qui compte le plus : la
  documentation n'est pas le contrat.

### Corrigé

- **L'action publiée installe de nouveau.** La vérification de provenance
  ajoutée en 0.1.0 appelait `gh attestation verify` sans jeton ; `gh` refuse de
  tourner dans un workflow sans `GH_TOKEN`, si bien que l'installateur tenait
  tout binaire pour invérifiable et le refusait, chez tous les consommateurs, en
  0.1.0 comme en 0.1.1. L'action fournit désormais `github.token` elle-même :
  personne ne devrait avoir à câbler un jeton pour installer un binaire.

  L'angle mort mérite d'être nommé. Le job de pull request servait `install.sh`
  en boucle locale avec la vérification sautée, donc le chemin public n'était
  jamais exercé avant le job d'après-publication, c'est-à-dire après que le tag
  existe. Un job appelle maintenant l'action contre une version déjà publiée, à
  chaque pull request.

## [0.1.1] - 2026-08-19

### Corrigé

- **Une instance Scaleway dont le groupe de sécurité est créé par le même plan
  n'est plus rapportée `CRITICAL` « VM sans groupe de sécurité ».** Au stade
  plan, `security_group_id` est *unknown after apply*, donc absent de
  `planned_values` ; le transform `list` fabriquait alors une collection vide,
  qui satisfaisait la garde de capacité de la règle, celle-là même qui existe
  pour empêcher ce cas. Un transform de collection ne s'applique désormais que
  si la clé source existe réellement. « Absent » signifie que la source n'expose
  pas l'information ; « présent et vide » est une information.

  Cela change un verdict sur un tenant inchangé : sur un plan Terraform, le
  contrôle passe de `fail` à `non évalué`. Sur un plan, une instance réellement
  dépourvue de groupe de sécurité est indistinguable d'une instance dont le
  groupe n'est pas encore connu : Pépin le dit maintenant au lieu de deviner. Le
  chemin live en bénéficie aussi, où une API omettant une clé produisait le même
  `[]` fabriqué.

  Trouvé en rejouant quinze stacks Terraform de tiers contre le binaire.

- **La porte de non-dérive de la documentation compile désormais ce qu'elle
  mesure.** Elle réutilisait un `./pepin` déjà présent à la racine : un binaire
  périmé pouvait donc valider des pages périmées, ce qui est arrivé, la porte
  annonçant « à jour » pendant que la doc affichait encore le finding ci-dessus.

### Ajouté

- **Documentation produit, générée plutôt que recopiée.** Six pages en anglais
  avec leur contrepartie française synchronisée : un démarrage en cinq minutes
  sans compte cloud, le modèle d'assessment (`pass` / `fail` / `non applicable` /
  `non évalué`), la matrice de couverture providers × contrôles, les limites
  connues, la lecture commentée d'un scan réel, et le périmètre exact avec ses
  non-objectifs. Toute sortie de commande est capturée d'une exécution réelle du
  binaire, la matrice est calculée depuis le référentiel et les descripteurs de
  providers, et une porte de CI échoue dès que l'un des deux dérive.

- **Fuzzing des entrées non fiables** : `FuzzParsePlan` et `FuzzInventoryWalk`,
  couvrant le plan Terraform et l'export d'inventaire. Il a immédiatement trouvé
  une ressource au type vide entrant dans le modèle, désormais écartée et
  conservée en régression.

### Sécurité

- `SECURITY.md` lie désormais son canal de signalement privé au lieu de
  seulement le décrire.

## [0.1.0] - 2026-08-19

### Sécurité

- **Une politique chargée à chaud n'a plus accès au réseau.** `--policy-dir`
  compilait du Rego tiers avec les capacités par défaut d'OPA, `http.send`
  compris : une règle de huit lignes suffisait à POSTer l'inventaire évalué —
  user-data des instances, documents de politique IAM, policies de bucket — vers
  un hôte arbitraire, ou à balayer le réseau interne du runner depuis l'intérieur
  du scanner. Corrigé en amont dans `scankit v0.2.2` ; une politique appelant un
  de ces builtins ne compile plus. L'évaluation reçoit aussi une borne de cinq
  minutes.
- **Les identifiants fournisseur ne survivent plus à une redirection HTTP.** Go
  ne retire, en cross-domain, que `Authorization`, `Cookie` et
  `WWW-Authenticate` — ni `X-Auth-Token` (clé secrète Scaleway) ni
  `AccessKey`/`SecretKey` (Outscale). Une seule 302 vers un hôte contrôlé les
  livrait. Le client de collecte ne suit plus les redirections.
- **`pepin verify` ne lit plus hors de son bundle.** Les noms d'artefacts
  venaient du manifeste, fourni par le tiers audité : `../secret` faisait de la
  vérification un oracle d'existence et de contenu.
- **`--seal --redact` n'emporte plus les clés du tenant.** Le caviardage ne
  couvrait que les documents libres, alors qu'`access_key` est un attribut à part
  entière du modèle normalisé et que `password`/`certificate` remontent des bases
  managées.
- **Toolchain en Go 1.26.6**, qui annule cinq avis de la bibliothèque standard
  atteignables depuis ce code (`net/url`, `crypto/tls`, `encoding/xml`,
  `encoding/asn1`, `net/http`).
- **L'action publiée vérifie l'authenticité, pas seulement l'intégrité.** Le
  binaire et `checksums.txt` viennent de la même origine : qui peut remplacer les
  assets d'une release remplace les deux. `install.sh` vérifie désormais la
  provenance via `gh attestation verify`.

### Corrigé

Chaque point ci-dessous peut changer un verdict sur un tenant inchangé.

- **Un scan qui n'a rien mesuré ne rend plus `0`.** Identifiants expirés, droits
  insuffisants ou inventaire tronqué produisaient le même résultat vide qu'un
  tenant sain, et la porte de CI passait au vert sur un périmètre jamais regardé.
  Le code `3` le dit maintenant, sans exiger `--strict`.
- **Quatorze contrôles ne concluent plus `pass` sans la donnée décisive.** Le
  verrou de capacité gagne treize entrées, et une collection vide ne compte plus
  comme collectée : le collecteur IAM pose toujours `statements`, à `[]` quand un
  document ne s'analyse pas, si bien que quatre contrôles critical/high
  concluaient « conforme » sur zéro information.
- **`authenticated-read` et `AuthenticatedUsers` sont détectés** comme exposition
  publique : les deux accordent la lecture à tout utilisateur authentifié de la
  plateforme, donc hors du tenant.
- **Un bucket rendu public par une `acl` en ligne** sur `scaleway_object_bucket`
  est enfin collecté ; il produisait auparavant zéro finding et un verdict
  « conforme ».
- **Les booléens transmis en chaîne sont honorés.** Un plan Terraform rend
  certains attributs de schéma en `"true"`/`"false"`, et `== false` est
  simplement faux pour `"false"` ; 25 comparaisons dans 16 règles passent
  désormais par `truthy()`.
- **Une région non cataloguée est signalée** au lieu de passer en silence : les
  tables de classification sont des listes blanches, et leur silence valait
  « en UE ».
- **Normalisation réseau** : `-1`, `any` et un protocole vide signifient tous
  « tout protocole », et un scalaire là où le modèle attend une liste ne rend
  plus la règle indéfinie — un export portant `"cidrs": "0.0.0.0/0"` n'était pas
  signalé.
- **Sévérités de `CLD-CHF-2` alignées** sur `high` pour ses trois contrôles : la
  sévérité pilote la porte de CI, et l'écart n'était pas justifié.

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

> [🇬🇧 English](cli.md) · 🇫🇷 Français

# Référence de la CLI

La ligne de commande de Pépin est une **interface publique**. Ses verbes, ses drapeaux et ses
codes de sortie sont une promesse sur laquelle des pipelines se branchent, et cette promesse
est tenue par un test, pas par une phrase : `cmd/testdata/frozen/cli.json` porte la forme
courante, `cmd/frozen_test.go` passe au rouge quand le code s'en écarte, et un second test
passe au rouge quand la fixture bouge sans que son numéro de version ait suivi.

Cette page est générée depuis cette même fixture et depuis le binaire lui-même. Les tableaux
de drapeaux viennent de la surface gelée ; chaque aide ci-dessous est la sortie de
`pepin <verbe> --help`, capturée en exécutant le binaire. Un drapeau public absent d'ici fait
échouer `TestEveryPublicCLIFlagIsDocumented`, dans `cmd/`.

## La surface gelée

<!-- pepin:gen cli-verbs -->
| Commande | Drapeaux |
|---|---|
| `pepin provider` | _(aucun drapeau propre)_ |
| `pepin provider list` | _(aucun drapeau propre)_ |
| `pepin provider new` | _(aucun drapeau propre)_ |
| `pepin provider validate` | _(aucun drapeau propre)_ |
| `pepin scan` | `--exceptions`, `--format` / `-f`, `--kubeconfig`, `--lang`, `--live`, `--policy-dir` / `-p`, `--profile`, `--redact`, `--region`, `--s3-endpoint`, `--seal`, `--strict`, `--terraform` / `-t` |
| `pepin scsl` | `--index` |
| `pepin verify` | `--bundle`, `--pubkey`, `--re-derive` |
| `pepin version` | _(aucun drapeau propre)_ |
<!-- /pepin:gen cli-verbs -->

`pepin provider` répond aussi à `pepin providers`, et `pepin provider list` à
`pepin provider ls`. Les alias sont un confort ; les noms ci-dessus sont la promesse.

Quatre formes sont versionnées séparément, parce qu'un pipeline consommateur les parse
séparément :

<!-- pepin:gen surface-versions -->
| Surface | Ce qui est gelé | Version |
|---|---|:-:|
| `cli` | verbes, drapeaux et codes de sortie | **v3** |
| `findings` | forme de `--format json` (`findings` + `summary`) | **v1** |
| `assessment` | forme du document `--format assessment` | **v1** |
| `bundle` | forme du bundle de preuve (fichiers, rôles, manifest) | **v2** |
| `inventory` | forme de l'inventaire normalisé (enveloppe, ressource, types et attributs) | **v2** |
<!-- /pepin:gen surface-versions -->

Un numéro monte à **tout** changement de forme, ajout compris : il signifie « la surface a
bougé », pas « la surface a cassé ». La procédure d'un changement délibéré (régénérer la
fixture, incrémenter la constante, écrire la ligne de CHANGELOG) est dans
[RELEASING.fr.md](../../RELEASING.fr.md).

## `--lang`, le drapeau persistant

Pépin est bilingue. La langue est résolue une seule fois, avant que la moindre aide ne soit
construite, dans cet ordre : la première source non vide décide.

`--lang=fr|en` → `PEPIN_LANG` → `LC_ALL` → `LANG` → repli `en`

Une locale inconnue retombe sur l'anglais sans erreur. Le drapeau est **persistant** : il vaut
pour la racine et pour toutes les sous-commandes.

Ce que la langue change, et ce qu'elle ne change pas :

| Stable dans les deux langues | Traduit |
|---|---|
| codes de contrôle (`CLD-*`), identifiants de check, sévérités, statuts, sujets, codes de sortie | titres, messages, remédiations, preuves, textes d'aide, formulation du verdict |

**Un pipeline qui compare le TEXTE d'un rapport entre deux exécutions doit épingler
`PEPIN_LANG`.** Sans quoi un runner dont le `LANG` change produira un diff qu'aucun changement
de configuration n'explique. Un pipeline adossé aux codes et aux statuts, lui, n'est pas
concerné. La même précaution vaut pour l'empreinte d'un bundle scellé : voir
[Le bundle de preuve](../guides/evidence-bundles.fr.md#lempreinte-dépend-de-la-langue).

<!-- pepin:gen cli-help-root -->
```text
Pépin — CSPM multi-cloud souverain.

Évalue la posture d'un cloud (OVH, Scaleway, Exoscale, Outscale…) contre un
référentiel commun ancré sur SCSL, SecNumCloud, CIS et ISO.

Usage:
  pepin [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  provider    Gérer les providers déclaratifs (lister, valider, créer)
  scan        Évaluer la posture d'un cloud contre les politiques
  scsl        Vérifier la cohérence avec l'index SCSL et piloter la roadmap
  verify      Vérifier l'intégrité (et la signature) d'un bundle de preuve
  version     Afficher la version

Flags:
  -h, --help          help for pepin
      --lang string   langue de l'interface : fr | en (défaut : PEPIN_LANG, puis LC_ALL/LANG, sinon en)

Use "pepin [command] --help" for more information about a command.
```
<!-- /pepin:gen cli-help-root -->

## `pepin scan`

Le verbe autour duquel tout tourne : il lit une source, évalue les règles communes sur le
modèle normalisé, et rend un rapport.

<!-- pepin:gen cli-help-scan -->
```text
Évalue un inventaire contre les règles embarquées du provider (+ règles
externes via --policy-dir). Trois sources : un export JSON normalisé, un plan
Terraform (--terraform), ou une collecte live de l'API (--live).

Usage:
  pepin scan <provider> [export.json] [flags]

Flags:
      --exceptions fichier               fichier YAML de dérogations (control, justification, expires_at, owner, approved_by) : un écart couvert passe au statut exempted, jamais conforme
  -f, --format string                    format de sortie : table | json | assessment | oscal | sarif (default "table")
  -h, --help                             help for scan
      --kubeconfig string                chemin d'un kubeconfig pour auditer l'état DANS un cluster Kubernetes (utiliser un accès en LECTURE SEULE, TTL court — jamais cluster-admin)
      --live                             collecter l'inventaire en direct via l'API du provider (identifiants requis)
  -p, --policy-dir stringArray           répertoire de règles externes (.rego), répétable — chargé sans recompilation
      --profile string                   profil d'identifiants pour la collecte live (ex. ~/.osc/config.json)
      --redact                           caviarder les valeurs sensibles (user-data, policies) de l'input.json du bundle — pour partage à un tiers ; INCOMPATIBLE avec verify --re-derive
      --region string                    région cible pour la collecte live
      --s3-endpoint string               endpoint S3 custom pour le stockage objet (collecte live ; ex. MinIO http://localhost:9000)
      --seal string                      écrire un bundle de preuve opposable (assessment + OSCAL + manifest + checksums) dans ce dossier
      --strict                           porte CI stricte : code de sortie ≠ 0 si aucun contrôle n'est mesuré (hors gouvernance) ou s'il subsiste un écart medium/low
  -t, --terraform terraform show -json   auditer un plan Terraform (terraform show -json) au lieu d'un export d'inventaire

Global Flags:
      --lang string   langue de l'interface : fr | en (défaut : PEPIN_LANG, puis LC_ALL/LANG, sinon en)
```
<!-- /pepin:gen cli-help-scan -->

### Une source, et une seule

| Source | Comment | Identifiants |
|---|---|---|
| Inventaire normalisé (export JSON) | `pepin scan <provider> inventaire.json` | aucun |
| Plan Terraform | `pepin scan <provider> --terraform plan.json` | aucun |
| API live du fournisseur | `pepin scan <provider> --live` | oui, dans les variables d'environnement natives du fournisseur |

Deux pièges méritent d'être nommés, et la CLI les refuse plutôt que de deviner :

- **`--terraform` est un interrupteur booléen, pas un chemin.** Le fichier de plan est
  l'argument positionnel. `--terraform=plan.json` échoue sur une erreur d'analyse
  (`strconv.ParseBool`), il ne lit pas le fichier.
- **`--live` et `--terraform` s'excluent mutuellement.** Poser les deux est refusé par le
  groupe de drapeaux, pas arbitré en silence en faveur de l'un d'eux.

Choisir entre un plan et un scan live est une vraie décision, et les deux sont légitimes :
[Plan Terraform contre scan live](../concepts/terraform-vs-live.fr.md).

### Drapeau par drapeau

| Drapeau | Défaut | Ce qu'il fait |
|---|---|---|
| `--format`, `-f` | `table` | format de sortie : `table`, `json`, `assessment`, `oscal`, `sarif`, voir [Formats de sortie](output-formats.fr.md) |
| `--terraform`, `-t` | `false` | lire le fichier positionnel comme un plan Terraform (`terraform show -json`) |
| `--live` | `false` | collecter l'inventaire depuis l'API du fournisseur |
| `--region` | défaut du fournisseur | région (ou zone) visée par la collecte live |
| `--profile` | aucun | profil d'identifiants pour la collecte live (ex. `~/.osc/config.json`) |
| `--s3-endpoint` | défaut du fournisseur | endpoint S3 personnalisé pour le stockage objet (collecte live) |
| `--kubeconfig` | aucun | auditer l'état **dans** un cluster Kubernetes ; accès en lecture seule, à durée courte, jamais `cluster-admin` |
| `--policy-dir`, `-p` | aucun | dossier de règles `.rego` externes, répétable, chargé sans recompiler |
| `--seal` | aucun | écrire un bundle de preuve dans ce dossier, voir [Le bundle de preuve](../guides/evidence-bundles.fr.md) |
| `--redact` | `false` | caviarder les valeurs sensibles de l'`input.json` du bundle ; **incompatible avec `verify --re-derive`** |
| `--strict` | `false` | porte de CI plus exigeante : code non nul s'il subsiste des écarts medium/low |
| `--lang` | détectée | langue de l'interface, `fr` ou `en` |

`--strict` ne commande **pas** la porte « rien n'a été mesuré » : un scan qui n'a rien conclu
rend déjà 3 sans lui. Voir [Codes de sortie](exit-codes.fr.md).

## `pepin verify`

La vérification tierce d'un bundle produit par `scan --seal`.

<!-- pepin:gen cli-help-verify -->
```text
Recalcule l'empreinte SHA-256 de chaque fichier listé dans checksums.txt et signale
toute altération. C'est la vérification tierce d'un bundle produit par `scan --seal`.

Sans --pubkey, seule l'intégrité (altération accidentelle) est vérifiée : un attaquant
peut régénérer fichiers + checksums. Avec --pubkey, la SIGNATURE cosign de checksums.txt
est vérifiée (non-répudiation) — l'opérateur ayant scellé le bundle avec (cosign 3.x) :
  cosign sign-blob --key cosign.key --bundle checksums.txt.bundle checksums.txt

Usage:
  pepin verify <dossier-bundle> [flags]

Flags:
      --bundle string   bundle de signature cosign (défaut : <dossier>/checksums.txt.bundle)
  -h, --help            help for verify
      --pubkey string   clé publique cosign pour vérifier la signature de checksums.txt
      --re-derive       rejouer les règles sur input.json et vérifier que l'assessment scellé en découle (opposabilité forte)

Global Flags:
      --lang string   langue de l'interface : fr | en (défaut : PEPIN_LANG, puis LC_ALL/LANG, sinon en)
```
<!-- /pepin:gen cli-help-verify -->

Trois niveaux d'assurance, qui ne sont pas interchangeables :

| Commande | Ce qu'elle établit |
|---|---|
| `pepin verify bundle` | cohérence interne : altération accidentelle seulement. Qui peut réécrire un fichier peut aussi réécrire `checksums.txt`. |
| `pepin verify bundle --pubkey cosign.pub` | non-répudiation : la signature cosign de `checksums.txt` est valide. Exige le binaire `cosign` dans le `PATH`. |
| `pepin verify bundle --re-derive` | opposabilité : les règles sont rejouées sur `input.json`, et l'assessment scellé doit en découler. |

## `pepin provider`

<!-- pepin:gen cli-help-provider -->
```text
Gérer les providers déclaratifs (lister, valider, créer)

Usage:
  pepin provider [flags]
  pepin provider [command]

Aliases:
  provider, providers

Available Commands:
  list        Lister les providers cloud disponibles
  new         Créer le squelette d'un provider (providers/<nom>.yaml)
  validate    Valider les providers d'un dossier (défaut : providers/) contre le contrat

Flags:
  -h, --help   help for provider

Global Flags:
      --lang string   langue de l'interface : fr | en (défaut : PEPIN_LANG, puis LC_ALL/LANG, sinon en)

Use "pepin provider [command] --help" for more information about a command.
```
<!-- /pepin:gen cli-help-provider -->

### `pepin provider list`

<!-- pepin:gen cli-help-provider-list -->
```text
Lister les providers cloud disponibles

Usage:
  pepin provider list [flags]

Aliases:
  list, ls

Flags:
  -h, --help   help for list

Global Flags:
      --lang string   langue de l'interface : fr | en (défaut : PEPIN_LANG, puis LC_ALL/LANG, sinon en)
```
<!-- /pepin:gen cli-help-provider-list -->

<!-- pepin:gen provider-list -->
```text

// pépin  providers enregistrés
  exoscale  Exoscale (CH) — instances, security groups, block storage, SKS, SOS
  kubernetes  Kubernetes (in-cluster) — RBAC, Pod Security Standards, NetworkPolicy
  outscale  Outscale (3DS) — VM, BSU, OOS, EIM, security groups, OKS, LBU
  scaleway  Scaleway — object storage, instances, IAM, security groups
```
<!-- /pepin:gen provider-list -->

### `pepin provider validate`

Contrôle les descripteurs d'un dossier (par défaut `providers/`) contre le contrat qu'ils
doivent tenir. Rend 1 quand un descripteur est invalide, ce qui en fait une porte de CI
utilisable sur une contribution qui ajoute un fournisseur.

<!-- pepin:gen cli-help-provider-validate -->
```text
Valider les providers d'un dossier (défaut : providers/) contre le contrat

Usage:
  pepin provider validate [dossier] [flags]

Flags:
  -h, --help   help for validate

Global Flags:
      --lang string   langue de l'interface : fr | en (défaut : PEPIN_LANG, puis LC_ALL/LANG, sinon en)
```
<!-- /pepin:gen cli-help-provider-validate -->

### `pepin provider new`

<!-- pepin:gen cli-help-provider-new -->
```text
Créer le squelette d'un provider (providers/<nom>.yaml)

Usage:
  pepin provider new <nom> [flags]

Flags:
  -h, --help   help for new

Global Flags:
      --lang string   langue de l'interface : fr | en (défaut : PEPIN_LANG, puis LC_ALL/LANG, sinon en)
```
<!-- /pepin:gen cli-help-provider-new -->

## `pepin scsl`

Rapport de cohérence avec l'index SCSL et la roadmap qu'il pilote. `--index` désigne l'API
statique du framework (`api/v1/exigences.json`) ; l'index est gelé, donc ce verbe rapporte, il
ne crée jamais d'exigence.

<!-- pepin:gen cli-help-scsl -->
```text
Vérifier la cohérence avec l'index SCSL et piloter la roadmap

Usage:
  pepin scsl [flags]

Flags:
  -h, --help           help for scsl
      --index string   chemin de l'API SCSL (api/v1/exigences.json du framework) (default "../framework-scsl/api/v1/exigences.json")

Global Flags:
      --lang string   langue de l'interface : fr | en (défaut : PEPIN_LANG, puis LC_ALL/LANG, sinon en)
```
<!-- /pepin:gen cli-help-scsl -->

## `pepin version`

<!-- pepin:gen cli-help-version -->
```text
Afficher la version

Usage:
  pepin version [flags]

Flags:
  -h, --help   help for version

Global Flags:
      --lang string   langue de l'interface : fr | en (défaut : PEPIN_LANG, puis LC_ALL/LANG, sinon en)
```
<!-- /pepin:gen cli-help-version -->

Le nom de l'outil s'écrit « pépin » en français et « pepin » en anglais : un script qui parse
cette sortie doit épingler `PEPIN_LANG`, ou couper sur l'espace.

## Codes de sortie

<!-- pepin:gen cli-exit-codes -->
| Code | Constante (`cmd/surface.go`) | Signification |
|:-:|---|---|
| **0** | `conforme` | aucun écart critical/high, et au moins un contrôle réellement mesuré |
| **1** | `non_conformite` | au moins un écart critical ou high |
| **2** | `erreur` | erreur technique : le scan n'a pas pu conclure |
| **3** | `strict` | le scan n'établit pas la conformité : rien n'a été mesuré, ou la collecte n'a pas pu lire tout le périmètre (les deux sans `--strict`), ou écarts medium/low restants avec `--strict` |
| **4** | `derogation` | tout écart critical/high restant est couvert par une dérogation datée et attribuée (`--exceptions`) |
<!-- /pepin:gen cli-exit-codes -->

La sémantique complète, situation par situation, avec les commandes qui produisent chaque
code : [Codes de sortie et porte de CI](exit-codes.fr.md).

## Pour aller plus loin

- [Codes de sortie](exit-codes.fr.md) : le contrat d'intégration d'une porte de CI.
- [Formats de sortie](output-formats.fr.md) : quel format parser, et ce qui est garanti.
- [Le bundle de preuve](../guides/evidence-bundles.fr.md) : `--seal`, `--redact`, `verify`.
- [Matrice de couverture](../coverage.fr.md) : ce qui est mesurable, par fournisseur et par
  source.

## Comment cette page reste vraie

Les tableaux de drapeaux sont lus dans la surface CLI gelée, et les blocs d'aide sont la sortie
standard de `pepin <verbe> --help`, capturée par `internal/docgen` qui exécute le binaire.
`mise run gen-docs` les réécrit ; `TestGeneratedDocsAreUpToDate` échoue quand ce qui est
committé diffère de ce que le binaire imprime aujourd'hui, et
`TestEveryPublicCLIFlagIsDocumented` échoue quand un drapeau public n'apparaît jamais sur cette
page.

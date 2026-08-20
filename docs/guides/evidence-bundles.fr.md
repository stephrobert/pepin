> [🇬🇧 English](evidence-bundles.md) · 🇫🇷 Français

# Le bundle de preuve

`pepin scan --seal <dossier>` écrit un **bundle de preuve** : l'inventaire évalué, l'assessment
typé, son rendu OSCAL, un manifest et une liste de sommes de contrôle. C'est ce qu'on remet à un
auditeur, ce qu'on attache à un ticket de changement, ou ce qu'on garde pour expliquer, six mois
plus tard, à quoi ressemblait le tenant le jour du scan.

## À lire avant d'en partager un

**Sans `--redact`, un bundle embarque l'inventaire BRUT.** `input.json` est l'objet exact contre
lequel les règles ont été évaluées : données utilisateur cloud-init, documents de politique IAM,
policies de bucket, identifiants de clés d'accès, chaînes de connexion. Un outil qui trouve un
mot de passe en clair dans les données utilisateur *possède* nécessairement ce mot de passe, et
le sceller sans caviardage le met dans le bundle.

Pépin le dit à chaque scellement, sur la sortie d'erreur :

<!-- pepin:gen bundle-seal -->
```console
$ ./pepin scan scaleway --terraform examples/scaleway/terraform/plan.json --seal bundle
pepin: ⚠ input.json embarque l'inventaire BRUT (peut contenir des secrets : user-data, policies). Traiter le bundle comme SENSIBLE, ou utiliser --redact pour le partager.
pepin: bundle de preuve écrit dans bundle — sceller : cosign sign-blob bundle/checksums.txt
$ echo $?
1
```
<!-- /pepin:gen bundle-seal -->

(Le code de sortie est `1` parce que le plan scanné est non conforme, pas parce que le
scellement a échoué : `--seal` ne change jamais le verdict. Voir
[Codes de sortie](../reference/exit-codes.fr.md).)

Traiter un bundle non caviardé comme **le secret du tenant qu'il décrit** : stockage d'artefacts
privé, rétention courte, aucune page de CI publique, aucune pièce jointe à un ticket qu'un
prestataire peut lire. [`--redact`](#--redact--un-bundle-quon-remet-à-un-tiers) existe pour les
bundles qui quittent le périmètre, et il a un coût que cette page énonce sans détour.

## Ce que contient un bundle

<!-- pepin:gen bundle-files -->
| Fichier | Rôle déclaré au manifest |
|---|---|
| `input.json` | `evaluated-input` |
| `assessment.json` | `assessment` |
| `assessment-oscal.json` | `oscal-assessment-results` |
| `manifest.json` | `manifest` |
| `checksums.txt` | `checksums` |
<!-- /pepin:gen bundle-files -->

La forme de cet ensemble (quels fichiers, quels rôles, quels champs de manifest) est une
**surface gelée** : `cmd/testdata/frozen/bundle.json`. Un vérificateur tiers peut s'y adosser,
et un changement de cette forme incrémente un numéro de version et reçoit sa ligne de CHANGELOG.

### `manifest.json`

<!-- pepin:gen bundle-manifest -->
```json
{
  "format": "pepin-assessment-bundle/v2",
  "inventory_schema": "pepin-inventory/v3",
  "disclaimer": "Ce rapport évalue la configuration d'un tenant (périmètre commanditaire). Les correspondances normatives (SecNumCloud, ISO, CIS) sont indicatives : elles ne constituent pas une preuve de qualification/certification, laquelle porte sur le prestataire de service cloud.",
  "generated": "<timestamp>",
  "tool": {
    "name": "pepin",
    "version": "<version>",
    "digest": "<provenance>"
  },
  "ruleset": {
    "name": "pepin-config",
    "digest": "sha256:<sha256>"
  },
  "target": {
    "id": "scaleway",
    "provider": "scaleway",
    "platform": "scaleway"
  },
  "source": "terraform-plan",
  "summary": {
    "fail": 10,
    "not-applicable": 2,
    "not-evaluated": 9,
    "pass": 6
  },
  "artifacts": [
    {
      "file": "input.json",
      "role": "evaluated-input",
      "sha256": "<sha256>",
      "bytes": "<bytes>"
    },
    {
      "file": "assessment.json",
      "role": "assessment",
      "sha256": "<sha256>",
      "bytes": "<bytes>"
    },
    {
      "file": "assessment-oscal.json",
      "role": "oscal-assessment-results",
      "sha256": "<sha256>",
      "bytes": "<bytes>"
    }
  ]
}
```
<!-- /pepin:gen bundle-manifest -->

Les marqueurs `<timestamp>`, `<sha256>`, `<version>`, `<provenance>` et `<bytes>` tiennent la
place de valeurs qui changent à chaque exécution ; un manifest réel les porte en entier. Trois
champs méritent l'attention :

- **`tool.digest`** identifie le binaire qui a produit le bundle (`vcs:<commit>`, suffixé
  `+modified` quand l'arbre de travail était sale).
- **`ruleset.digest`** couvre ensemble les règles, les descripteurs de fournisseurs et le
  référentiel. S'il diffère du vôtre au moment de re-dériver, vous rejouez avec un autre jeu de
  règles, et `pepin verify` le dit au lieu de conclure à une divergence.
- **`source`** consigne d'où venait la donnée : `terraform-plan`, `live` ou `export`. Deux
  bundles du même tenant issus de deux sources différentes ne se comparent pas ligne à ligne :
  [Plan Terraform contre scan live](../concepts/terraform-vs-live.fr.md).

### `checksums.txt`

<!-- pepin:gen bundle-checksums -->
```text
<sha256>  input.json
<sha256>  assessment.json
<sha256>  assessment-oscal.json
<sha256>  manifest.json
```
<!-- /pepin:gen bundle-checksums -->

Format `sha256sum` standard, donc vérifiable sans Pépin du tout
(`sha256sum -c checksums.txt`). C'est ce fichier que couvre une signature cosign : signer un
seul fichier qui nomme les empreintes des autres est ce qui fait qu'une signature couvre tout le
bundle.

### `exemptions.json` : seulement si une dérogation a été appliquée

Un dossier qui tait ce qu'il a écarté n'est pas opposable. Quand un scan reçoit
`--exceptions`, le bundle porte un sixième artefact, qui consigne la politique telle qu'elle a
été chargée et ce que chaque entrée a réellement produit (`applied`, `expired`, `orphan`).

C'est un artefact comme les autres : son empreinte figure dans `checksums.txt`, donc il est
couvert par la signature, et l'empreinte du bundle **dépend des dérogations**. Un dossier ne
peut pas les retirer sans échouer à sa propre vérification. Le manifeste en porte le résumé
(combien appliquées, échues, orphelines, et l'empreinte de la politique elle-même), pour qu'un
vérificateur le voie avant d'ouvrir quoi que ce soit d'autre.

`verify --re-derive` rejoue la politique scellée à l'instant d'évaluation scellé. Sans cela, un
bundle parfaitement fidèle serait déclaré falsifié pour la seule raison que le vérificateur ne
détient pas le fichier de l'opérateur, et une dérogation valide semblerait « expirer » entre le
scan et la vérification.

## Vérifier un bundle

Trois niveaux, et ils n'établissent pas la même chose. Ne pas les confondre.

### Niveau 1 : l'intégrité, accidentelle seulement

<!-- pepin:gen bundle-verify -->
```console
$ ./pepin verify bundle
⚠ bundle cohérent en interne : bundle
  (intégrité ACCIDENTELLE seulement — NON opposable sans --pubkey pour la signature cosign)
$ echo $?
0
```
<!-- /pepin:gen bundle-verify -->

Noter le `⚠` et l'absence de coche verte rassurante. Recalculer des empreintes prouve seulement
que les fichiers concordent avec `checksums.txt`, et qui peut réécrire un fichier peut aussi
réécrire cette liste. Ce niveau attrape un téléchargement tronqué, pas un adversaire.

Voici un bundle dont l'assessment scellé a été édité, un `fail` passé en `pass`, la
falsification la plus tentante qui soit :

<!-- pepin:gen bundle-tampered -->
```console
$ ./pepin verify bundle-tampered
erreur : empreinte invalide pour assessment.json (fichier altéré)
$ echo $?
2
```
<!-- /pepin:gen bundle-tampered -->

Code de sortie 2, et le fichier est nommé. Une vérification incapable de cela serait
décorative.

### Niveau 2 : la signature, pour la non-répudiation

`--pubkey` vérifie la signature cosign de `checksums.txt`, qui couvre transitivement chacun des
fichiers qu'il nomme. Elle passe par le binaire `cosign`, qui doit être dans le `PATH`.

```bash
# Une fois, côté scellement (cosign 3.x) :
cosign sign-blob --key cosign.key --bundle bundle/checksums.txt.bundle bundle/checksums.txt

# Côté vérification :
pepin verify bundle --pubkey cosign.pub
```

Sans `--bundle`, Pépin cherche `<dossier>/checksums.txt.bundle`. Quand ce fichier manque, il
refuse avec le code de sortie 2 et imprime la commande `sign-blob` exacte à exécuter : il ne
retombe pas sur une vérification non signée.

> **Cette page ne montre aucune sortie capturée d'une vérification signée.** Signer exige une
> clé privée et une configuration Sigstore que le générateur de documentation ne possède pas, et
> tous les autres blocs de cette page sont des exécutions réelles. Les commandes ci-dessus sont
> celles que documente `pepin verify --help`
> ([Référence de la CLI](../reference/cli.fr.md#pepin-verify)).

### Niveau 3 : la re-dérivation, la seule opposable

Une signature atteste des octets. Elle n'atteste pas que le verdict découle de l'entrée : un
bundle parfaitement signé peut porter un assessment que son propre `input.json` ne soutient pas.
`--re-derive` rejoue les règles sur `input.json` et compare le résultat à l'assessment scellé.

<!-- pepin:gen bundle-rederive -->
```console
$ ./pepin verify bundle --re-derive
⚠ bundle cohérent en interne : bundle
  (intégrité ACCIDENTELLE seulement — NON opposable sans --pubkey pour la signature cosign)
✓ re-dérivation FIDÈLE : l'assessment scellé découle bien de input.json
$ echo $?
0
```
<!-- /pepin:gen bundle-rederive -->

Elle re-rend aussi l'OSCAL depuis l'assessment re-dérivé et le compare, pour que
`assessment-oscal.json`, l'artefact qu'ingère un outil GRC, ne puisse pas être falsifié de son
côté. Et elle contrôle la cohérence de provenance : le scan grave un instant unique à la fois
dans `input.evaluated_at` et dans `run.timestamp`, et un écart entre les deux trahit un
antidatage.

Comme les règles sont rejouées avec le binaire **courant**, re-dériver avec un autre jeu de
règles imprime une note qui nomme les deux empreintes, au lieu de faire comme si le bundle était
fidèle.

## L'empreinte dépend de la langue

L'assessment et l'OSCAL portent de la prose : titres, messages, remédiations, preuves. La prose
est traduite, donc le même scan scellé en français et en anglais produit des octets différents,
et donc des empreintes différentes, pour chaque fichier qui porte du texte :

<!-- pepin:gen bundle-cross-lang -->
| Fichier | Mêmes octets dans les deux langues ? |
|---|---|
| `assessment-oscal.json` | ❌ diffère (l'empreinte change) |
| `assessment.json` | ❌ diffère (l'empreinte change) |
| `checksums.txt` | ❌ diffère (l'empreinte change) |
| `input.json` | ✅ identique |
| `manifest.json` | ❌ diffère (l'empreinte change) |
<!-- /pepin:gen bundle-cross-lang -->

`input.json` est identique, parce qu'un inventaire ne porte pas de prose. Tout le reste diffère.
(La comparaison neutralise l'horodatage du run, qui diffère entre deux exécutions quelles que
soient les langues.)

Deux conséquences pratiques.

**Épingler `PEPIN_LANG` au scellement**, si l'on archive des bundles et qu'on compare leurs
empreintes dans le temps. Sinon, un runner dont la locale change produira un bundle
« différent » à partir d'un tenant inchangé.

**La vérification, elle, s'en moque.** `verify --re-derive` rejoue les règles dans les deux
langues et accepte l'une ou l'autre concordance : ce qu'elle compare (statuts, sujets,
références, provenance) est identique, seule la formulation change. Voici le `pepin` français de
cette page vérifiant le bundle scellé en anglais, re-dérivation comprise :

<!-- pepin:gen bundle-cross-verify -->
```console
$ ./pepin verify bundle-en --re-derive
⚠ bundle cohérent en interne : bundle-en
  (intégrité ACCIDENTELLE seulement — NON opposable sans --pubkey pour la signature cosign)
✓ re-dérivation FIDÈLE : l'assessment scellé découle bien de input.json
$ echo $?
0
```
<!-- /pepin:gen bundle-cross-verify -->

Une accusation de falsification à tort est le pire verdict qu'un vérificateur puisse rendre :
ce cas se montre donc, il ne s'affirme pas.

## `--redact` : un bundle qu'on remet à un tiers

`--redact` remplace la valeur de chaque attribut sensible de l'inventaire embarqué par son
empreinte. Le finding reste, le secret part :

<!-- pepin:gen bundle-redact -->
```json
"user_data": "[REDACTED sha256:2bb6abea90eaa2eb]"
```
<!-- /pepin:gen bundle-redact -->

Les attributs caviardés sont ceux dont la valeur peut porter un secret : `user_data`,
`document`, `statements`, `policy`, `access_key`, `secret_key`, `password`, `token`, `ssh_key`,
`public_key`, `private_key`, `certificate`, `connection_string`.

**Le coût est explicite : un bundle caviardé ne peut pas être re-dérivé.** Les règles
rejoueraient sur des valeurs caviardées et aboutiraient à un autre verdict, ce que le
vérificateur signale comme une divergence :

<!-- pepin:gen bundle-redact-rd -->
```console
$ ./pepin verify bundle-redacted --re-derive
⚠ bundle cohérent en interne : bundle-redacted
  (intégrité ACCIDENTELLE seulement — NON opposable sans --pubkey pour la signature cosign)
erreur : re-dérivation DIVERGE de l'assessment scellé : le bundle n'atteste PAS fidèlement input.json (résultat fabriqué ou config différente)
$ echo $?
2
```
<!-- /pepin:gen bundle-redact-rd -->

Ce message est sévère à dessein, c'est le même qu'obtient un bundle fabriqué, pour que le
caviardage soit un choix délibéré et non un défaut qui désactive en silence la garantie la plus
forte.

| | Le garder en interne | Le remettre à un tiers |
|---|---|---|
| Drapeau | `--seal bundle` | `--seal bundle --redact` |
| `input.json` | inventaire brut, **sensible** | valeurs sensibles remplacées par leur empreinte |
| `verify` | oui | oui |
| `verify --pubkey` | oui | oui |
| `verify --re-derive` | oui | **non** |
| Ce sur quoi il repose | la re-dérivation | la signature cosign |

## Un cycle complet

```bash
# 1. Sceller (scan live du tenant réel, dans une langue épinglée)
PEPIN_LANG=fr pepin scan scaleway --live --region fr-par --seal bundle

# 2. Signer la liste de sommes, qui couvre chacun des fichiers qu'elle nomme
cosign sign-blob --key cosign.key --bundle bundle/checksums.txt.bundle bundle/checksums.txt

# 3. Côté réception : intégrité, signature, et le fait que le verdict découle de l'entrée
pepin verify bundle --pubkey cosign.pub --re-derive
```

En CI, garder le bundle privé et de courte durée : les étapes d'artefact des deux plateformes
sont dans [GitHub Actions](github-actions.fr.md#archiver-le-bundle-de-preuve) et
[GitLab CI](gitlab-ci.fr.md#archiver-le-bundle-de-preuve).

## Ce qu'un bundle ne prouve pas

Un bundle établit ce que Pépin a observé, quand, avec quelles règles, et que le verdict découle
de ce qu'il a observé. Il n'établit pas que le fournisseur de cloud est qualifié, ni que le
tenant est conforme à un référentiel : les correspondances normatives sont indicatives, et le
périmètre est celui du commanditaire. Voir
[Périmètre et non-objectifs](../concepts/scope.fr.md), et l'avertissement que le manifest porte
lui-même.

## Pour aller plus loin

- [Formats de sortie](../reference/output-formats.fr.md) : les documents assessment et OSCAL en
  détail.
- [Le modèle d'assessment](../concepts/assessment-model.fr.md) : ce qu'affirme chaque statut.
- [Référence de la CLI](../reference/cli.fr.md#pepin-verify) : tous les drapeaux de `verify`.
- [Codes de sortie](../reference/exit-codes.fr.md) : `verify` rend 2 quand il refuse un bundle.

## Comment cette page reste vraie

Chaque bloc console ci-dessus est une exécution réelle capturée par `internal/docgen` : le
bundle est scellé dans un dossier jetable, vérifié, re-dérivé, altéré puis re-vérifié, scellé à
nouveau avec `--redact`, et scellé une fois de plus dans l'autre langue pour le tableau de
comparaison. Les chemins affichés sont les chemins relatifs qu'un lecteur taperait. Horodatages,
empreintes, tailles et version de build sont marqués comme tels, parce qu'ils changent à chaque
exécution. `TestGeneratedDocsAreUpToDate` échoue quand la page committée ne correspond plus à ce
que fait le binaire.

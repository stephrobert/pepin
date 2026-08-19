> [🇬🇧 English](ROADMAP.md) · 🇫🇷 Français

# Feuille de route

Ce que Pépin mesure aujourd'hui, où va l'effort ensuite, et ce qu'il ne tentera pas.

Cette page ne porte **ni date ni engagement**. Elle donne une direction et un ordre,
pour que quelqu'un qui envisage de bâtir sur Pépin distingue ce qui existe de ce qui
est visé. Ce qui y est présenté comme une capacité est soit déjà mesurable, soit
nommé explicitement comme une intention.

Deux pages portent les faits, toutes deux générées depuis le référentiel et vérifiées
en CI : les [limites connues](docs/known-limitations.fr.md) pour les angles morts, et
la [matrice de couverture](docs/coverage.fr.md) pour ce qui est mesurable contrôle par
contrôle. Cette page ne les recopie pas.

## Où en est Pépin, v0.2.0

Quatre fournisseurs enregistrés : trois clouds souverains, plus un collecteur
Kubernetes intra-cluster.

<!-- pepin:gen provider-list -->
```text

// pépin  providers enregistrés
  exoscale  Exoscale (CH) — instances, security groups, block storage, SKS, SOS
  kubernetes  Kubernetes (in-cluster) — RBAC, Pod Security Standards, NetworkPolicy
  outscale  Outscale (3DS) — VM, BSU, OOS, EIM, security groups, OKS, LBU
  scaleway  Scaleway — object storage, instances, IAM, security groups
```
<!-- /pepin:gen provider-list -->

Le référentiel contre lequel ils sont évalués :

<!-- pepin:gen control-counts -->
| Chiffre | Nombre |
|---|---:|
| Contrôles au référentiel | 57 |
| Contrôles déclarés pour au moins un fournisseur | 56 |
| `critical` | 10 |
| `high` | 32 |
| `medium` | 13 |
| `low` | 2 |
<!-- /pepin:gen control-counts -->

Autour de cela : deux sources analysables (un plan Terraform, qui ne crée rien, et un
scan live en lecture seule), cinq formats de sortie, un contrat de codes de sortie
stable, un bundle de preuve scellé avec son export OSCAL, et un produit bilingue du
rapport jusqu'au référentiel lui-même. Le
[catalogue des contrôles](docs/controls/index.fr.md) donne une page par contrôle, et
l'[architecture](docs/project/architecture.fr.md) explique le trajet d'une ressource,
de l'API du cloud jusqu'au finding.

## La suite, dans l'ordre

### 1. Refermer les cases `◐` : collecter ce que les règles savent déjà lire

La matrice de couverture marque `◐` là où une source produit bien le type de ressource
mais ne projette pas l'**attribut décisif**. Pépin y rend `not-evaluated`, jamais un
`pass` silencieux : le verdict n'est donc pas faux. Mais un `◐` est un contrôle que le
lecteur croyait avoir. En refermer un n'ajoute **aucune règle** : cela ajoute un champ
à un descripteur de fournisseur. D'où la première place, et la couverture la moins
chère du projet.

### 2. Les preuves de remédiation déployables, un fournisseur à la fois

Chaque finding porte déjà une remédiation **textuelle**, et un test du référentiel
refuse un contrôle qui en manquerait. Ce qui est partiel, c'est la **preuve
déployable** : un module Terraform autonome et conforme sous `references/remediation/`.

<!-- pepin:gen remediation-coverage -->
| Fournisseur | Preuves de remédiation |
|---|---:|
| exoscale | 4 / 26 |
| kubernetes | 0 / 4 |
| outscale | 0 / 40 |
| scaleway | 0 / 25 |
| **Total** | **4 / 95** |
<!-- /pepin:gen remediation-coverage -->

`mise run check-remediation` est volontairement débranché de `mise run validate` : une
porte rouge en permanence est une porte qu'on apprend à ignorer. Le plan est d'amener un
fournisseur à 100 %, de rebrancher la porte pour celui-là, puis de recommencer. Voir le
[guide de remédiation](docs/guides/remediation.fr.md).

### 3. Les domaines de contrôles encore minces

Par ordre de priorité approximatif, et chacun conditionné à une exigence SCSL **gelée**,
jamais à une exigence inventée :

- **Bases de données managées**, au-delà du seul fournisseur qui les porte aujourd'hui.
- **Journalisation et piste d'audit au niveau organisation** : la famille la plus faible
  du référentiel, et la première dont un auditeur demande des nouvelles.
- **Services exposés par défaut** dans les couches PaaS : fonctions serverless,
  registres de conteneurs, espaces de noms d'images.
- **Cycle de vie cryptographique** : politique de rotation des clés, expiration des
  certificats, durée de vie bornée des secrets au-delà de l'IAM.

Quand aucune exigence SCSL gelée ne couvre un contrôle candidat, il reste au catalogue
de tri (`referentiel/catalogue.yaml`) plutôt que de devenir un contrôle dont on aurait
inventé la justification. Cette retenue est le sujet du projet, pas un effet de bord.

### 4. Fournisseurs

**OVHcloud** est le prochain cloud souverain de la liste. En ajouter un, c'est un
descripteur et **zéro règle** : les règles sont communes, seule la source change. C'est
exactement le parcours d'[ajouter un fournisseur](docs/contributing/adding-a-provider.fr.md).

Rien n'est déclaré couvert avant que son contrat soit vérifié champ par champ contre le
SDK ou la spécification d'API du fournisseur. Un fournisseur livré avec des champs non
vérifiés poserait dans la matrice une case verte que personne ne peut défendre en audit.

### 5. Lire le référentiel hors du dépôt

Le référentiel, la matrice de couverture et le catalogue des contrôles sont déjà générés
depuis le binaire. L'intention est de les publier sous forme de site bilingue alimenté
par ce même générateur, pour que les pages publiées ne puissent pas diverger de l'outil.
C'est une direction, pas une fonctionnalité livrée.

## Ce que Pépin ne fera pas

La page [périmètre et non-objectifs](docs/concepts/scope.fr.md) fait autorité. En bref :
Pépin lit le côté **client** d'un cloud, à un instant donné, en lecture seule. Ce n'est
pas un agent d'exécution, il ne garde aucun historique entre deux runs, il ne modifie
rien sur votre compte, et ses correspondances normatives sont indicatives : un rapport
Pépin n'est pas une preuve de qualification, laquelle porte sur le prestataire de cloud
et non sur le scan d'un tenant.

## Les limites qui dictent cet ordre

Trois, tirées des [limites connues](docs/known-limitations.fr.md), parce qu'elles
expliquent pourquoi la liste ci-dessus est ordonnée ainsi :

- **Le contrat d'API est consigné par type de ressource, pas par (type × source).**
  C'est ce qui produit les cases `◐`, et le point 1 ci-dessus y répond.
- **La colonne « live » de la matrice de couverture est dérivée des descripteurs, pas
  observée.** Elle dit ce qu'un descripteur projette, pas ce qu'une API a rendu lors
  d'une exécution mesurée.
- **Rien n'est mesuré entre deux runs.** Un résultat décrit un instant ; la posture
  continue est l'affaire de ce qui ordonnance Pépin.

## Comment cette page reste honnête

Les chiffres ci-dessus sont **générés** par `mise run gen-docs` depuis le référentiel,
les descripteurs de fournisseurs et le binaire lui-même ; `TestGeneratedDocsAreUpToDate`
casse la CI dès qu'ils dérivent. La prose est écrite à la main et relue à chaque
publication.

Le journal d'investigation détaillé (verdicts d'audit par fournisseur, bugs moteur,
constats champ par champ) est un document de travail de mainteneur, en français, tenu
hors de la documentation produit, dans `notes/roadmap-interne.fr.md`. Ce qu'il contient
qui concerne un utilisateur a sa place dans les
[limites connues](docs/known-limitations.fr.md) : on l'y déplace, on ne le résume pas
ici.

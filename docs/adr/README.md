# Registre des décisions d'architecture

> 🇫🇷 Français · [🇬🇧 English](README.en.md)

Le code décrit **l'état actuel**. Un ADR explique **pourquoi cet état existe** et
quelles décisions ne se remettent pas en cause sans décision explicite.

La règle qui gouverne ce registre :

> **Une issue décrit ce que nous voulons changer. Un ADR décrit les décisions
> déjà prises. Une issue ne peut pas annuler un ADR en silence.**

## Pourquoi ce registre existe

Ce projet a rejoué plusieurs fois les mêmes débats. Une solution écartée revient
sous un autre nom, un correctif local contredit une décision globale, un
commentaire vieilli devient faux, et deux parties du code appliquent des règles
divergentes sans que personne ne l'ait décidé.

Le risque est plus grand encore avec un agent : il produit vite une solution
techniquement plausible **et déjà rejetée**, parce que rien ne lui dit qu'elle
l'a été.

## Quand créer un ADR

Une modification de l'une de ces surfaces en demande un :

modèle de données · architecture · réseau · API · workflow · format de sortie ·
persistance · sécurité · génération · orchestration · **comportement public**.

Une correction de faute, un test manquant, un renommage local : **non**.

## Quand ne pas en créer

- Pour expliquer ce que fait une portion de code — c'est le rôle d'un commentaire.
- Pour consigner une préférence de style — c'est le rôle de `CONTRIBUTING.md`.
- Pour décrire un état chiffré du dépôt. Un ADR dit « tout module généré a un
  test », jamais « les 43 modules ont un test ». Les chiffres vivants viennent
  des rapports générés (`docs/detection-quality.md`, `docs/coverage.md`).

## Le contrat de revue, avant toute modification

1. Identifier les composants touchés.
2. Trouver les ADR applicables — par la table de portée ci-dessous, ou par le
   champ `scope:` de chaque ADR.
3. **Les lire en entier**, pas seulement leur titre.
4. Relever leurs invariants.
5. Vérifier si le changement respecte, étend, contredit ou rouvre une décision.

Ne pas lire tout le registre à chaque correctif. En revanche, un changement
transverse ou architectural justifie de le parcourir entièrement.

## Si le changement contredit un ADR

**Ne pas modifier le code.** Signaler d'abord :

```
Cette évolution entre en conflit avec ADR-XXXX : <résumé de la décision>.

Elle exige donc soit de respecter l'ADR, soit de prendre explicitement une
nouvelle décision d'architecture.
```

Si la décision doit vraiment changer, **ne pas réécrire l'ancien ADR**. En créer
un nouveau qui le remplace :

```
ADR-0023 supersedes ADR-0007
```

et passer l'ancien en `Status: Superseded by ADR-0023`. L'historique reste
visible : c'est lui qui empêche de refaire le tour trois fois.

## Si la solution proposée a déjà été rejetée

La section **Alternatives écartées** n'est pas décorative. Si une implémentation
correspond à une alternative explicitement rejetée, **s'arrêter** et le dire :

```
ADR-0014 a rejeté X pour les raisons A, B et C.
La solution proposée réintroduit X.
Avant d'implémenter, établir si A, B et C ont cessé d'être vraies.
```

## Table de portée

| Domaine | ADR |
|---|---|
| Architecture des règles, providers | [0001](0001-regles-communes-providers-collecteurs.md) |
| Moteur, findings, rendu, scoring | [0002](0002-moteur-partage-scankit.md) |
| Modèle de données, inventaire | [0003](0003-inventaire-contrat-gele.md), [0007](0007-provenance-index-parallele.md) |
| Référentiel, frameworks normatifs | [0004](0004-index-scsl-gele.md), [0009](0009-configuration-lie-mapping.md) |
| Codes de sortie, portes de CI | [0005](0005-codes-de-sortie.md), [0008](0008-derogation-nest-pas-conformite.md) |
| Assessment, verdicts, dégradation | [0006](0006-jamais-un-pass-non-prouve.md), [0014](0014-jamais-fabriquer-une-donnee-absente.md) |
| Tests, preuve, véracité | [0010](0010-dette-de-veracite-comptee.md) |
| Langue, documentation | [0011](0011-bilinguisme-francais-normatif.md) |
| Sécurité de la mesure, identifiants | [0012](0012-aucun-identifiant-en-ci.md) |
| Compatibilité des consommateurs | [0013](0013-un-code-de-controle-ne-se-renomme-pas.md) |

Chaque ADR porte aussi un `scope:` en tête, pour que cette identification soit
automatisable.

## Dérive entre les ADR et le code

`mise run adr-drift` produit un rapport : ADR citant un fichier disparu, ADR
`Accepted` dont l'invariant n'a plus de test, ADR remplacé mais non marqué.

Ce contrôle est **partiel par construction** : il détecte l'obsolescence de
forme, pas la contradiction de fond. Un rapport vide ne prouve pas qu'un ADR dit
encore la vérité.

## Format

Le gabarit est dans [`TEMPLATE.md`](TEMPLATE.md). La section **Invariants** est
obligatoire pour tout ADR structurant, et chaque invariant automatisable doit
nommer le test qui le garde. Quand il ne l'est pas, l'ADR dit pourquoi.

# ADR-0013 — Un code de contrôle ne se renomme pas sans migration

```yaml
status: Accepted
date: 2026-08-23
scope:
  - controls
  - compatibility
```

## Contexte

Plusieurs contrôles portent un nom qui promet plus que ce qu'ils mesurent. La
correction évidente est de les renommer.

## Décision

Un code de contrôle est une **surface de consommateur**. Il ne se renomme pas sans
migration documentée. Quand la promesse est fausse, on corrige d'abord le titre, la
description, le message et **la condition évaluée** — dans les deux langues.

## Justification

Un code voyage dans les `ruleId` SARIF des consommateurs, dans les assessments
archivés, et **dans les fichiers de dérogations**. Un renommage transforme du jour
au lendemain une dérogation valide en dérogation **orpheline** : l'opérateur voit
un avertissement apparaître et son écart réapparaître, sans avoir rien changé.

Déclencher en masse la détection d'orphelines qu'on vient soi-même de livrer serait
une régression de confiance.

## Alternatives écartées

**Renommer et laisser les consommateurs suivre.** Rejeté : ils ne le sauront qu'en
voyant leur porte de CI changer d'avis.

**Renommer avec un alias permanent.** Recevable, mais non retenu par défaut : deux
noms pour un contrôle sont deux noms qui divergeront dans les rapports.

## Conséquences

Certains noms restent imparfaits plus longtemps que leur contenu. Quand le **nom
lui-même** est ce qui ment — `cross_organization` pour ce qui mesure
`cross_account` —, le renommage redevient la bonne réponse, mais il se paie d'une
migration explicite et d'une ligne de CHANGELOG.

## Invariants

- Aucun renommage de code sans migration écrite.
- Un renommage retenu déclare son effet sur les dérogations existantes.

*Non automatisable : aucune garde ne peut savoir qu'un renommage était justifié.
C'est une décision de revue, pas de test.*

## Validation

Revue de PR. La détection des dérogations orphelines rend l'effet visible en
exécution.

## Liens

ADR-0008 · issue #109.

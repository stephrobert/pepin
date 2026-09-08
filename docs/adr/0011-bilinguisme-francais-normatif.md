# ADR-0011 — Bilinguisme : l'anglais est primaire, le français fait foi pour le normatif

```yaml
status: Accepted
date: 2026-08-23
scope:
  - docs
  - cli
  - referentiel
```

## Contexte

Pépin vise l'écosystème souverain francophone **et** une audience internationale.
Un outil de conformité dont le contenu normatif est traduit approximativement perd
sa valeur d'opposabilité.

## Décision

**L'anglais est la langue primaire du dépôt** : `README.md`, `SECURITY.md`,
`CONTRIBUTING.md`, avec leur contrepartie `*.fr.md`.

**Le français est la langue de référence du contenu normatif** : c'est lui qui fait
foi, l'anglais en est la traduction maintenue en parallèle.

La langue de la CLI est détectée : `--lang` → `PEPIN_LANG` → `LC_ALL` → `LANG` →
repli `en`. Elle vaut pour tout ce que l'outil imprime, formats parsables compris.

Le **code et les commentaires** restent en français ; les **commits** en anglais.

## Justification

Le contenu normatif est adossé à des textes français (SCSL, SecNumCloud). En faire
la référence évite qu'une nuance juridique se perde dans un aller-retour de
traduction.

## Alternatives écartées

**Tout en anglais.** Rejeté : perte de précision sur le normatif, et le public
premier est francophone.

**Tout en français.** Rejeté : ferme l'outil à l'international.

**Traduire à la volée.** Rejeté : une traduction non relue d'un texte normatif est
une affirmation que personne n'a validée.

## Conséquences

Tout texte utilisateur s'écrit **deux fois, côte à côte** : `i18n.T(fr, en)` en Go,
`message`/`labels.message_en` en Rego, `titre`/`titre_en` au référentiel,
`reason`/`reason_en` dans les contrats.

Les traductions voyagent dans `labels` parce que scankit ne se modifie pas depuis
ici (ADR-0002).

## Invariants

- Aucune traduction manquante.
- Une sortie `LANG=en` ne porte aucun mot accenté qui ne vienne de l'inventaire scanné.
- La sortie française publiée n'a pas bougé d'un octet sans décision.

*Gardes : `TestEveryControlIsBilingual`, `TestEveryFindingCarriesRemediation`,
`TestEveryContractJustificationIsBilingual`,
`TestFrenchScanOutputHasNotMovedOneCharacter`.*

## Validation

Les quatre portes sont dans `mise run validate` et `mise run test`.

## Liens

`CLAUDE.md` §1.2.

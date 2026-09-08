# ADR-0004 — L'index SCSL est gelé : on mappe, on n'invente jamais

```yaml
status: Accepted
date: 2026-08-23
scope:
  - referentiel
```

## Contexte

Pépin s'ancre sur SCSL, dont les exigences `CLD-*` sont publiées par un référentiel
externe. La tentation, en couvrant un contrôle nouveau, est de « créer l'exigence
qui manque ».

## Décision

L'index SCSL est **gelé**. Un contrôle se mappe sur une exigence `CLD-*`
**existante**. Si aucune ne couvre le besoin, le contrôle reste au catalogue en
`statut: a_trier` — hors périmètre tant que SCSL ne l'a pas figé.

## Justification

Un référentiel dont on invente les exigences ne vaut plus rien comme référentiel.
L'opposabilité que Pépin revendique repose entièrement sur le fait que la
correspondance renvoie à un texte que Pépin n'a pas écrit.

## Alternatives écartées

**Créer des exigences locales en attendant.** Rejeté : elles deviendraient
indiscernables des vraies dans un rapport lu par un auditeur.

**Mapper sur l'exigence « la plus proche ».** Rejeté : c'est la même invention,
avec un air de rigueur.

## Conséquences

Des contrôles utiles restent au catalogue sans être actifs. C'est le prix, et il
est assumé : un contrôle sans ancrage normatif reste mesurable, il n'est
simplement pas opposable.

## Invariants

- Tout `scsl:` d'un contrôle actif existe dans l'index gelé.
- Aucune exigence n'est créée depuis ce dépôt.

*Gardes : `TestSCSLReferencesExist` (tout `scsl:` existe dans l'index gelé),
`TestSCSLCoherence`, `TestFrameworkReferencesExist`.*

## Validation

`mise run validate`, et `scsl-drift` au préflight de release.

## Liens

`CLAUDE.md` §3 et §10 · ADR-0009 pour ce qu'un assouplissement fait perdre.

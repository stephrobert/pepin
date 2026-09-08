# ADR-0008 — Une dérogation écarte un écart, elle ne le déclare jamais conforme

```yaml
status: Accepted
date: 2026-08-23
scope:
  - assessment
  - ci
```

## Contexte

Une équipe qui ne peut pas déroger à un contrôle **supprime le contrôle**. Il
fallait donc un mécanisme de dérogation — sans qu'il devienne une porte vers le vert.

## Décision

Un cinquième statut, `exempted`, et un code de sortie dédié, `4`. Une dérogation
porte une justification, une échéance, un propriétaire et un approbateur.

L'écart dérogé **reste** dans `--format json`, dans le SARIF et dans les décomptes
de sévérité ; `summary.conforme` reste faux. Seule la **porte** bouge.

## Justification

`0` ferait de la dérogation un faux vert silencieux, exactement ce que le statut
existe pour empêcher. `1` la rendrait inutile. `4` est non nul — rien ne passe en
silence — et distinct, donc une chaîne qui l'accepte doit écrire le nombre, donc
savoir qu'il existe.

L'échéance s'évalue contre `evaluated_at`, pas contre l'horloge : un bundle rejoué
rend le même verdict, et une dérogation valide ne « périme » pas entre le scan et
la vérification.

## Alternatives écartées

**Rendre `0`.** Rejeté : faux vert.

**Rendre `1`.** Rejeté : mécanisme inutilisable, donc contrôle supprimé.

**Retirer l'écart dérogé du SARIF.** Rejeté : c'est le format le plus souvent
branché sur une porte de fusion. L'en retirer serait le faux vert que cette vague
combat. Le coût est réel et documenté (issue #94).

## Conséquences

`exemptions.json` est un artefact du bundle, sous `checksums.txt`, donc **sous
signature** : un dossier ne peut pas perdre ses dérogations sans échouer à sa
propre vérification.

Une dérogation **périmée** cesse de s'appliquer et le dit ; une dérogation
**orpheline** est signalée. Sous `--strict`, les deux font échouer la porte.

## Invariants

- `exempted` n'est jamais compté comme `pass`.
- Un scan avec dérogation ne rend jamais `0`.
- Seul un `fail` peut être dérogé.
- Une dérogation expirée ne s'applique plus.

*Gardes : `TestAnExemptionNeverTurnsAFailIntoAPass`,
`TestAnExemptionNeverProducesAZeroExitCode`,
`TestAnExpiredExemptionStopsApplyingAndSaysSo`, `TestOnlyAFailCanBeExempted`.*

## Validation

`mise run test`, plus la vérification qu'un bundle scellé avec dérogations se
re-dérive fidèlement.

## Liens

ADR-0005 · issue #94.

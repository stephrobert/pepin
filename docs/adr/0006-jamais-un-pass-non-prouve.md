# ADR-0006 — Un PASS non prouvé est un défaut : verrou de capacité et dégradation

```yaml
status: Accepted
date: 2026-08-23
scope:
  - assessment
```

## Contexte

L'incident fondateur : une policy EIM inline `Action: "*"` échappait à **tous** les
contrôles `iam_policy_*`. La règle Rego était juste. **La donnée n'arrivait jamais
jusqu'à elle.** Un test Rego parfait serait resté vert pendant que le scanner
produisait un faux vert.

## Décision

Deux mécanismes, tenus par l'assessment et non par les règles :

**Le verrou de capacité** (`requiredAttr`) — un contrôle ne peut pas rendre `pass`
si l'attribut qui décide n'a pas été collecté.

**La dégradation** (`assess.Degrade`) — une unité de collecte incomplète fait
passer les contrôles qui en dépendent de `pass` à `not-evaluated`, avec la raison.

La transition est **strictement directionnelle** et n'en produit qu'une :
`pass → not-evaluated`.

## Justification

Confier cette garde aux règles serait la confier à la mémoire de celui qui écrira
la cinquantième. L'assessment la tire une fois, pour toutes les règles présentes
et à venir.

Un `fail` est **conservé** : l'écart a été *vu*, la donnée qui l'établit est
arrivée, et l'effacer supprimerait une non-conformité vraie. Un `not-applicable`
est conservé : il vient du contrat du fournisseur, et rien de ce qu'un scan lit ou
ne lit pas ne peut le contredire.

## Alternatives écartées

**Une garde par règle.** Rejeté : c'est une garde qu'on oublie d'écrire.

**Avorter le scan sur un endpoint refusé.** Rejeté : sûr en apparence, mais cela
pousse l'opérateur à **élargir les droits** du compte qui exécute le CSPM. Un outil
qui incite à donner plus de privilèges travaille contre son propre objet.

**Dégrader aussi les `fail`.** Rejeté : cf. justification.

## Conséquences

Une seule exception à « ce qui a été lu est gardé » : un **agrégat** n'est jamais
calculé sur une liste tronquée, parce qu'un compte faux est pire qu'un compte
absent — une règle le compare à un seuil.

**Dette connue** : le verrou raisonne aujourd'hui par *type* et en *any-of*, pas par
ressource ni en *all-of*, et un contrôle ne déclare qu'un type. Voir #102 et #103 :
c'est la limite actuelle de cette décision, pas une remise en cause de sa direction.

## Invariants

- Aucun `pass` sans l'attribut qui décide.
- La dégradation ne produit que `pass → not-evaluated`.
- Un agrégat n'est jamais calculé sur une liste tronquée.

*Gardes : `TestAnIncompleteCollectionWithdrawsThePassAndOnlyThePass`,
`TestAnAggregateIsNotComputedOnATruncatedList`.*

## Validation

Campagne binaire avant/après sur les fixtures et le corpus à chaque évolution :
**aucun verdict ne doit devenir plus affirmatif**.

## Liens

`CLAUDE.md` §10 · ADR-0010 · issues #102, #103, #104.

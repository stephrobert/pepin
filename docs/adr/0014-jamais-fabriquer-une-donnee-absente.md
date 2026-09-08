# ADR-0014 — Une donnée absente ne se fabrique jamais, elle se déclare

```yaml
status: Accepted
date: 2026-08-23
scope:
  - assessment
  - collectors
  - output
```

## Contexte

Trois fois, la même tentation s'est présentée sous trois formes : donner une valeur
plausible à ce qu'on n'a pas observé, parce que l'absence est inconfortable à
afficher.

## Décision

Ce qui n'a pas été observé est **déclaré absent**, jamais estimé, jamais approché.

**Origine Terraform** : `terraform show -json` ne porte ni fichier ni ligne — vérifié
dans la source de Terraform. Le module se **lit** sur l'adresse, le fichier et la
ligne se **mesurent** dans les `.tf` à côté du plan. Sources absentes, module
distant, ou en-tête trouvé deux fois ⇒ **aucune origine**.

**Relevé de capacités** : il est dérivé de la collecte **réellement effectuée**,
jamais d'une sonde préalable.

**Permissions minimales** : chaque ligne dit `verifie` ou `a_verifier`.

## Justification

Une ligne fausse envoie quelqu'un corriger au mauvais endroit, **et elle est crue**.
Une sonde qui réussit là où l'appel réel échoue produit un relevé **qui ment** —
c'est la règle de provenance appliquée aux capacités.

Aucun drapeau ne pointe vers un autre arbre de sources : résoudre une origine contre
un arbre dont le plan ne vient pas donne une ligne **plausible et fausse**, ce qui
est le pire des deux.

## Alternatives écartées

**Approcher la ligne au plus proche.** Rejeté : plausible et faux.

**Sonder les endpoints avant le scan.** Rejeté : le relevé mentirait, et il
doublerait les appels contre les limites de débit qui causent l'incomplétude.

**Déduire les permissions de la documentation sans le dire.** Rejeté : chaque ligne
porte son état de vérification.

## Conséquences

Une origine manque plus souvent qu'elle ne pourrait. C'est le mode d'échec voulu.

**Dette connue** : plusieurs règles violent encore cette décision en défaussant un
attribut absent sur une valeur défavorable — `state` présumé actif, `is_enabled`
présumé faux, région inconnue traitée comme un écart. Voir #104, #106, #107, #109.
Ce sont des violations à corriger, pas des exceptions.

## Invariants

- Une origine absente n'est jamais fabriquée.
- Le relevé de capacités ne nomme que des appels réellement émis.
- Aucune règle ne produit un `fail` depuis un attribut absent défaussé.

*Gardes : `TestALiveOrExportedInventoryCarriesNoFabricatedOrigin`,
`TestAnAmbiguousBlockYieldsNoOrigin`, `TestTheRecordedCollectionStillHappens`. Le
troisième invariant n'a pas encore sa garde : c'est l'objet de #104.*

## Validation

`mise run test`, et la garde de rejeu des endpoints enregistrés.

## Liens

ADR-0003, ADR-0006, ADR-0007 · issues #104, #106, #107, #109.

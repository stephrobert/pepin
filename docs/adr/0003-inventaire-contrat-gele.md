# ADR-0003 — L'inventaire normalisé est un contrat gelé et versionné

```yaml
status: Accepted
date: 2026-08-23
scope:
  - model
```

## Contexte

L'inventaire est le point de passage de tout : la collecte y projette, les règles
s'y évaluent, l'assessment en dérive, le bundle le scelle. Chaque nouvel usage le
figeait un peu plus **par accident**, et une évolution du modèle cassait alors un
consommateur en silence.

## Décision

Le schéma est un contrat nommé, versionné (`model.InventoryFormat`) et gelé dans
`cmd/testdata/frozen/inventory.json`. Ce qui est garanti et ce qui ne l'est pas
sont écrits ; un changement de forme exige un bump, une entrée d'historique et une
ligne de CHANGELOG.

## Justification

Un contrat implicite est un contrat qu'on casse sans le savoir. Le nommer coûte un
`frozen-update` par changement ; ne pas le nommer coûte un consommateur cassé sans
avertissement.

## Alternatives écartées

**Laisser le schéma implicite et documenter les usages.** Rejeté : la
documentation dérive, le gel ne dérive pas.

**Versionner sans geler.** Rejeté : sans fixture, rien ne détecte le changement au
moment où il est fait.

## Conséquences

Ajouter un attribut à un descripteur rend la fixture rouge et coûte un
`frozen-update`. **C'est le but** : l'inventaire cesse d'être un détail
d'implémentation.

## Invariants

- Un attribut **absent** n'est jamais forcé à une valeur : « non collecté » et
  « collecté à faux » ne se confondent pas. *C'est l'invariant dont tout le modèle
  de confiance dépend.*
- `attributes` est une carte plate, en snake_case, agnostique du fournisseur.
- `evaluated_at` et `config` sont écrits **une fois** : un `input.json` rejoué
  garde les siens.
- L'ordre des ressources et l'exhaustivité de l'inventaire ne sont **pas** garantis.

*Garde : `TestTheFrozenSurfacesStillMatchTheirFixture`.*

## Validation

`mise run test` inclut le gel. `mise run frozen-update` régénère, et le diff se
relit — la porte attrape l'oubli, pas la négligence.

## Liens

`internal/model/schema.go` · ADR-0006, ADR-0007.

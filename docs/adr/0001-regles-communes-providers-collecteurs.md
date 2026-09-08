# ADR-0001 — Un seul jeu de règles commun, les providers ne sont que des collecteurs

```yaml
status: Accepted
date: 2026-08-23
scope:
  - rules
  - providers
```

## Contexte

Pépin est né de la généralisation d'`osc-policy`, écrit pour un seul cloud. La
question posée à la généralisation était : duplique-t-on les règles par
fournisseur, ou les mutualise-t-on sur un modèle commun ?

## Décision

**Toutes** les règles de posture sont communes et vivent dans
`internal/commonrules/rules/`. Elles s'évaluent sur un modèle normalisé agnostique.
Un provider est un **collecteur** : son seul rôle est de projeter sa source vers
ce modèle. Il ne porte aucune règle.

Le `code` d'un contrôle est agnostique (`network_securitygroup_…`,
`objectstorage_bucket_…`), jamais préfixé par un fournisseur.

## Justification

La duplication par fournisseur ne divise pas le travail, elle le multiplie : à
chaque correction de règle il faut retrouver ses N copies, et celle qu'on oublie
devient un faux verdict silencieux. Un modèle commun déplace le coût là où il est
irréductible — la différence entre les API — et l'y confine.

## Alternatives écartées

**Un paquet de règles par fournisseur.** Rejeté : N copies d'une même logique
divergent, et la divergence ne se voit qu'au moment où un verdict est faux.

**Des règles communes avec des exceptions par fournisseur dans la règle.** Rejeté :
c'est la duplication en pire, parce qu'elle est invisible dans l'arborescence.
`labels.provider` se tire de la ressource via `provider_of(r)`, jamais en dur.

## Conséquences

Ajouter un fournisseur, c'est un contrat plus un collecteur, et **zéro règle**.

En retour, tout écart d'API doit être absorbé par la normalisation, ce qui rend le
collecteur plus riche et le modèle commun plus contraint. Une règle ne se déclenche
que si des ressources du type visé existent, donc un fournisseur qui n'a pas ce
type ne produit pas de faux positif.

## Invariants

- Aucun fichier `.rego` de posture hors de `internal/commonrules/rules/`.
- Aucun code de contrôle ne nomme un fournisseur.
- `labels.provider` vient de la ressource, jamais d'une constante.

*Gardes : `TestActiveControlsHaveRule` (tout contrôle actif a une règle),
`TestRuleCodesAreControlled` (tout code émis est catalogué), `TestCatalogueCoherent`.*

*Le troisième invariant — `labels.provider` tiré de la ressource — n'a pas de garde :
une constante mise en dur produirait un rapport plausible que rien ne distingue.
Il tient par la revue et par le helper `provider_of`, qui est le seul chemin écrit.*

## Validation

`mise run validate` et `mise run test-rego`. Un contrôle actif sans règle, ou un
code émis hors catalogue, casse la validation.

## Liens

`CLAUDE.md` §0 et §3 · skills `nouvelle-regle`, `nouveau-provider`.

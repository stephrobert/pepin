# ADR-0009 — Un contrôle assoupli perd ses correspondances normatives

```yaml
status: Accepted
date: 2026-08-23
scope:
  - referentiel
  - controls
```

## Contexte

Rendre des contrôles configurables — étiquettes exigées, fraîcheur des snapshots,
seuil de confiance — donne à l'utilisateur une **poignée qui fabrique du vert** :
desserrer le seuil, puis continuer d'afficher la même correspondance CIS ou
SecNumCloud.

## Décision

`referentiel/controles.yaml` porte, à côté de `scsl:` et `frameworks:`, les
contraintes sous lesquelles ces correspondances valent (`config_requise`), en
quatre sens : `au_plus_le_defaut`, `superset_du_defaut`, `sous_ensemble_du_defaut`,
`au_moins_aussi_strict_que_le_defaut`.

Quand la configuration effective sort de la contrainte, le contrôle **perd ses
`references`** mais **garde son statut**, et l'assouplissement devient visible en
cinq endroits : terminal, assessment, `--format json`, bundle scellé, et `--strict`
qui rend `3`.

## Justification

Le statut ne change pas parce que le contrôle **a bien été évalué**, contre une
barre plus basse. On retire la seule chose qui a cessé d'être vraie — la
correspondance — et on garde la mesure. Marquer `not-evaluated` effacerait une
mesure réelle ; garder les `references` mentirait.

Vous pouvez abaisser la barre. Vous ne pouvez pas l'abaisser **et** garder le badge.

## Alternatives écartées

**Refuser la configurabilité.** Rejeté : une politique d'étiquetage figée produit
des faux positifs sur toute organisation qui n'a pas choisi la même convention.

**Configurer sans lier au mapping.** Rejeté : c'est la poignée à vert.

**Passer un contrôle assoupli en `not-evaluated`.** Rejeté : il a été mesuré.

Le quatrième sens de contrainte n'était pas prévu : la mesure a signalé qu'une
convention d'écriture différente — l'exemple même de l'issue d'origine — était
comptée comme un assouplissement alors qu'elle **durcit**. Un faux positif qui
punit le bon comportement.

## Conséquences

Un fichier de politique unique (`--policy`), dont `--exceptions` est le nom
historique ; les deux drapeaux sont mutuellement exclusifs, parce que deux fichiers
divergent.

La configuration voyage dans `input.config`, donc elle est **scellée** et le rejeu
n'applique jamais la politique du jour à un dossier d'hier.

## Invariants

- À configuration par défaut, aucun verdict ne bouge.
- Un contrôle assoupli n'affiche plus ses correspondances normatives.
- Un assouplissement est visible dans les cinq surfaces, bundle compris.
- Un durcissement n'est jamais signalé comme un assouplissement.

*Gardes : `TestAnExplicitDefaultPolicyMovesNoVerdict`,
`TestARelaxedConfigurationIsVisibleEverywhere`,
`TestAnotherWritingConventionIsNotARelaxation`.*

## Validation

Campagne de configurations éprouvées, avec ce que chacune déclenche et où elle
devient visible.

## Liens

ADR-0004, ADR-0005.

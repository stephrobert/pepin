# ADR-0005 — Sémantique des codes de sortie, et pourquoi il n'y en a pas un cinquième

```yaml
status: Accepted
date: 2026-08-23
scope:
  - cli
  - ci
```

## Contexte

Les codes de sortie sont la **porte de CI** : ce sont eux, et non le rapport, qui
décident si une chaîne passe. Chaque code ajouté coûte à tous les consommateurs une
relecture de leur `case $?`.

## Décision

`0` conforme · `1` non-conformité (≥ 1 écart critical/high) · `2` erreur technique ·
`3` **le scan n'établit pas la conformité** — rien de mesuré, collecte incomplète,
ou écarts medium/low avec `--strict` · `4` tout écart critical/high est couvert par
une dérogation valide.

Ordre de précédence : `2` > `1` > `3` (rien de mesuré) > `4` > `3` (`--strict`).

**Ni `3` ni `4` ne valent conformité.**

## Justification

Un cinquième code pour l'incomplétude a été envisagé puis rejeté : il ne pourrait
**jamais** primer sur `1` — masquer un écart critique réel parce que le reste
manquait serait exactement le faux vert que le modèle de confiance combat —, donc
il ne se déclencherait que là où `3` se déclenche déjà. Deux codes pour une même
position dans l'ordre de précédence sont un doublon, pas une distinction.

Ce qui distingue les cas est **lisible ailleurs** : relevé de capacités, motif de
chaque `not-evaluated`, clé `collection` de `--format json`.

## Alternatives écartées

**Un code dédié à l'incomplétude.** Rejeté, cf. ci-dessus.

**Rendre `0` sur une collecte partielle.** Rejeté : c'est la définition du faux vert.

**Rendre `1` sur une dérogation valide.** Rejeté : une équipe qui ne peut pas
déroger supprime le contrôle à la place.

## Conséquences

Un changement de cette sémantique touche **cinq lieux qui doivent bouger ensemble** :
`README.md`, `README.fr.md`, `CLAUDE.md` §6, et `docs/reference/exit-codes.{md,fr.md}`
qui est généré. L'oubli du résumé de page d'accueil s'est déjà produit.

## Invariants

- Un scan qui n'a rien collecté ne rend jamais `0`.
- Une dérogation ne rend jamais `0`.
- L'incomplétude n'efface jamais un écart observé : `1` prime sur `3`.

*Gardes : `TestAnExemptionNeverProducesAZeroExitCode`, et le préflight vérifie que
les codes répondent ce que le README promet.*

## Validation

`tools/release/preflight.sh` exécute les quatre cas nominaux avant tout tag.

## Liens

`CLAUDE.md` §6 · ADR-0008.

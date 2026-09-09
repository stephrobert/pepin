# ADR-0019 — Un profil filtre la porte, jamais le rapport

```yaml
status: Accepted
date: 2026-09-09
scope:
  - output
  - ci
```

## Contexte

Un premier scan doit provoquer « ah oui, ça c'est intéressant », pas « oui, je sais que
ma VM de test n'a pas de protection contre la suppression ».

Une VM publique dont SSH est ouvert à Internet et un volume sans snapshot récente sont
tous deux `high`. Le second est un faux positif **assumé** — la règle le documente
elle-même, un volume peut être sauvegardé autrement — et lui donner le poids du premier
fait douter du premier.

Mais changer ce qu'une porte accepte est **la façon la plus discrète de casser une
chaîne de CI** : une équipe qui échoue aujourd'hui passerait au vert après une mise à
jour, sans que personne ne l'ait décidé.

## Décision

Un profil (`scan --gate`) filtre **ce qui pèse dans le code de sortie**. Il ne retire
rien du rapport, dans aucun format — table, `json`, `sarif`, `assessment`, bundle
scellé.

Deux garanties, tenues par des gardes :

1. **Le défaut ne change pas** (`all`). Sans le drapeau, rien ne bouge.
2. **Aucun profil ne peut faire passer une chaîne de rouge à vert.** Un écart
   `critical`/`high` mis de côté rend `3` — « le scan n'établit pas la conformité » —
   jamais `0`. Un écart resté visible rend toujours `1`.

Chaque scan **imprime ce qu'il a mis de côté**, par code de contrôle.

## Justification

Ce n'est pas une invention : c'est la règle que ce dépôt applique **déjà** aux
dérogations et aux constats d'incertitude, écrite dans `cmd/scan.go` quelques lignes
au-dessus du point d'insertion — *« le rapport dit tout, seule la porte tient compte des
dérogations »*. Un profil est le troisième cas de la même doctrine.

Un profil qui retirerait des écarts du rapport transformerait le silence en faux vert,
et il le ferait depuis un **réglage** plutôt qu'un défaut de règle — donc invisible,
donc pire. C'est très exactement ce que la vague 4 a passé trois lots à combattre.

Le `3` n'est pas un choix par défaut : c'est la lecture honnête d'un scan
**volontairement partiel**, et l'ADR-0005 lui réserve déjà cette place. Aucun cinquième
code n'est requis.

## Alternatives écartées

**Changer le profil par défaut pour un profil peu bruyant.** Rejeté : une chaîne rouge
passerait au vert sans décision. Si ce défaut devait un jour changer, il faudrait une
version majeure et une ligne de CHANGELOG qui le dise en toutes lettres.

**Filtrer le rapport en même temps que la porte.** Rejeté : un écart caché ne se
retrouve pas. La demande d'un premier scan moins irritant est réelle, mais elle se
traite par le REGROUPEMENT des findings (issue #117), pas par leur disparition.

**Un cinquième code de sortie pour « partiel par profil ».** Rejeté par l'ADR-0005, dont
l'argument tient ici : il ne pourrait jamais primer sur `1`, donc il ne s'exprimerait
que là où `3` s'exprime déjà.

**Filtrer sur la seule sévérité.** Rejeté : cela mélange encore « grave » et « certain »,
qui est le défaut que la dimension de confiance existe pour corriger.

## Conséquences

Le drapeau ne s'appelle **pas** `--profile` : ce nom désigne déjà le profil
d'identifiants de la collecte live. Il s'appelle `--gate`, ce qui dit d'ailleurs mieux
ce qu'il fait.

Un finding dont le label `category` ou `confidence` manque est **toujours retenu**. Le
repli penche vers plus de sévérité : une règle qui oublierait son label ne doit pas
sortir silencieusement de toutes les portes.

## Invariants

- Le rapport est complet quel que soit le profil.
- Le profil par défaut ne filtre rien.
- Un profil ne rend jamais `0` là où le profil `all` rendrait `1`.

*Gardes : `TestNoGateProfileTurnsARedChainGreen` (mesurée sur le binaire, et éprouvée en
neutralisant la porte), `TestTheDefaultProfileFiltersNothing`,
`TestAnUnlabelledFindingIsAlwaysRetained`.*

## Validation

Matrice mesurée sur le binaire : inventaire non conforme → `1` ou `3` selon le profil,
jamais `0` ; plan corrigé → `0` partout.

## Liens

ADR-0005 (codes de sortie) · ADR-0008 (une dérogation n'est pas une conformité) ·
issues #111 (confiance), #112, #117 (regroupement).

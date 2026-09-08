# ADR-0015 — Une règle qui ne peut pas conclure le dit, et l'assessment en fait un `not-evaluated`

```yaml
status: Accepted
date: 2026-09-08
scope:
  - assessment
  - rules
```

## Contexte

`assess` sait déjà produire un `not-evaluated` dans deux cas : l'attribut décisif
n'a pas été collecté (verrou de capacité), ou l'unité de collecte est incomplète
(dégradation). Ces deux cas se décident **hors de la règle**, sur la présence de la
donnée.

Il en existe un troisième, que seule la règle peut voir : la donnée est **là**, elle
est lisible, et elle ne permet toujours pas de conclure. Une région renseignée mais
absente des tables de classification en est le cas type — Pépin sait qu'il ne sait
pas.

Le moteur ne connaît qu'un canal, `data.pepin.rules.deny`. Une règle n'a donc, à ce
jour, que deux moyens de traiter ce cas : se taire, ou émettre un écart. Les deux
sont faux.

**Se taire** est un fail-open structurel : les tables de classification sont des
listes blanches, donc leur silence vaudrait « conforme » pour toute région qu'elles
ignorent. Pour un outil de souveraineté, c'est le pire des défauts.

**Émettre un écart** est ce que fait le code aujourd'hui, et le message dit lui-même
« ni établie ni infirmée » tout en produisant un `fail`. C'est un faux rouge
programmé : il se déclenchera à chaque ouverture de région par un fournisseur.

## Décision

Une règle qui ne peut pas conclure émet un finding portant
`labels.inconclusive: "true"` et une raison.

L'assessment en fait un résultat **`not-evaluated`**, pas un `fail`, et ce finding
**ne compte pas** dans les décomptes de sévérité ni dans le code de sortie.

La règle **constate** ; l'assessment **statue**. C'est la division du travail que
l'ADR-0006 a déjà posée pour le verrou de capacité et la dégradation.

## Justification

Le troisième canal doit exister quelque part. Le mettre dans la règle est le seul
endroit possible — c'est la seule couche qui sait qu'une valeur présente est
inexploitable — mais **statuer** dans la règle referait l'erreur que l'ADR-0006 a
corrigée : une garde par règle est une garde qu'on oublie d'écrire.

Faire voyager l'information dans `labels` suit l'ADR-0002 : `finding.Finding` vient
de scankit, et les messages bilingues comme l'origine Terraform y voyagent déjà.

## Alternatives écartées

**Ajouter un second document Rego (`inconclusive`) interrogé par le moteur.**
Rejeté : `scankit/engine` n'interroge que `deny`, et l'ADR-0002 interdit un moteur
local. Cela exigerait une évolution amont pour un besoin qui n'est pas encore
partagé avec pitstop.

**Laisser la règle se taire et traiter le cas dans le collecteur.** Rejeté : le
collecteur a bien collecté. Rien ne manque à la collecte ; c'est la table de
classification de Pépin qui est incomplète, et le collecteur n'en sait rien.

**Étendre le verrou de capacité aux valeurs.** Rejeté : `requiredAttr` répond
« l'attribut a-t-il été collecté ». Lui faire juger la valeur mélangerait deux
questions et rendrait le verrou dépendant du contenu des tables.

**Émettre un `fail` de sévérité `low`.** Rejeté : c'est le faux rouge, atténué. Un
écart de faible gravité reste un écart, et le rapport continuerait d'affirmer ce
que la règle dit ne pas savoir.

## Conséquences

Un cas inconcluant **reste visible** : il produit une ligne `not-evaluated` avec sa
raison, et il apparaît au relevé de capacités et à la carte de qualité. Le rendre
invisible serait le fail-open que cette décision existe pour empêcher.

Le code de sortie peut passer de `1` à `3` sur un tenant dont les seuls écarts
étaient inconcluants. C'est le sens voulu : le scan n'établit pas la conformité,
il ne la nie pas non plus.

Coût assumé : une règle peut désormais mentir en se déclarant inconcluante pour
éviter un écart. Aucune garde automatique ne peut le détecter — seule la revue le
peut, et les scénarios de véracité l'attrapent chemin par chemin.

## Invariants

- Un finding `inconclusive` ne produit jamais un `fail`.
- Un finding `inconclusive` ne compte dans aucun décompte de sévérité, donc ne rend
  jamais `1`.
- Un cas inconcluant reste visible dans le rapport, avec sa raison.
- `inconclusive` ne se substitue jamais au verrou de capacité : un attribut **non
  collecté** relève de l'ADR-0006, pas d'ici.

*Gardes : `TestAnInconclusiveFindingIsNotEvaluated`,
`TestAnInconclusiveFindingNeverProducesExitOne`.*

## Validation

`mise run test`, et la campagne de non-régression : aucun verdict ne doit devenir
plus affirmatif.

## Liens

ADR-0006, ADR-0014 · issues #105, #106, #109.

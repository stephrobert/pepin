# ADR-0010 — Le contrat de véracité publie une dette comptée, jamais une matrice verte

```yaml
status: Accepted
date: 2026-08-23
scope:
  - testing
  - quality
```

## Contexte

L'unité de test qui compte pour un CSPM n'est pas `fixture → Rego → FAIL`, mais la
chaîne entière : `réponse d'API → collecteur → normalisation → verrou → Rego →
assessment → verdict`. L'incident EIM inline l'a démontré : la règle était juste,
la donnée n'arrivait jamais.

## Décision

`internal/veracity` dérive, pour chaque chemin contrôle × fournisseur × source, les
verdicts qu'il peut **réellement atteindre**, les compare aux scénarios versionnés
qui s'exécutent **contre le binaire**, et exige que la différence soit inscrite dans
un registre de dette.

Le registre est une porte **dans les deux sens** : effacer une dette sans preuve
échoue, en inventer une échoue, ajouter un contrôle sans scénarios échoue.

## Justification

178 chemins × quatre verdicts font environ sept cents cas. Personne ne peut
*éprouver* sept cents cas — les casser un par un pour vérifier qu'ils rougissent —
et une matrice engendrée par gabarit serait exactement le faux vert que tout le
reste combat.

Et « quatre verdicts partout » est faux : exiger un `not-applicable` d'un chemin où
le mécanisme existe demande d'inventer une non-applicabilité.

**Un compteur de dette honnête vaut mieux qu'une matrice verte.**

## Alternatives écartées

**Une matrice complète engendrée.** Rejeté : verte parce que creuse.

**Masquer la dette.** Rejeté : le chiffre laid est l'information.

**Compter un scénario par sa seule présence.** Rejeté : un filtre écarte le crédit
obtenu par simple absence d'un type de ressource.

## Conséquences

Les chiffres publiés sont laids et ils sont justes. La carte de qualité les affiche,
y compris « validé en live : 0 % », dont le zéro est **dérivé** et non écrit.

Un scénario `live` dont le type est produit par la spec de collecte entre par `api:`
— des réponses servies au collecteur **réel** —, jamais par `inventory:` : c'est le
collecteur qui a laissé passer l'incident fondateur.

## Invariants

- Aucun chiffre publié ne peut dépasser ce que le registre autorise.
- Un contrôle ajouté sans ses scénarios casse la CI.
- Une dette effacée sans preuve casse la CI.

*Gardes : `TestTheVeracityDebtLedgerIsExact`, `TestTheMapNeverExceedsTheLedger`,
`TestNoPublishedFigureFlattersTheMeasure`.*

## Validation

`mise run veracity-update` régénère le registre ; le diff se relit.

## Liens

`CLAUDE.md` §10 étape 6 · ADR-0006 · issue #115 (contre-exemples).

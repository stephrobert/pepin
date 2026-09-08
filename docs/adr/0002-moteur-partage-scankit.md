# ADR-0002 — Le moteur, le modèle de finding et le rendu viennent de scankit

```yaml
status: Accepted
date: 2026-08-23
scope:
  - engine
  - output
```

## Contexte

Pépin et **pitstop** évaluent des politiques et rendent des résultats. Écrits
séparément, ils auraient deux moteurs OPA, deux modèles de finding et deux rendus
qui divergeraient sans que personne ne le décide.

## Décision

Le moteur OPA, `finding.Finding`, le rendu (terminal, SARIF), le scoring et
l'assessment viennent du module publié
**`github.com/stephrobert/scankit`**, épinglé par version dans `go.mod`, consommé
en ligne — **sans `replace` local**.

Reste dans Pépin : les règles, les collecteurs, le référentiel, la marque et le
verdict.

## Justification

Une évolution de moteur ou de rendu profite alors aux deux outils, et le rendu
terminal reste identique entre eux — ce qui est une propriété visible par
l'utilisateur, pas un détail interne.

## Alternatives écartées

**Un moteur local à Pépin.** Rejeté : deux implémentations d'un même verdict
divergent, et la divergence se découvre par un rapport faux.

**Un `replace` local vers un clone de scankit.** Rejeté : il rend le dépôt
non reproductible pour quiconque n'a pas ce clone, et masque la version
réellement utilisée.

## Conséquences

Une évolution du modèle de finding ou du rendu se fait **en amont**. Ce coût est
réel : la vague 4 a dû faire voyager les traductions, l'origine Terraform et la
confiance dans `labels` plutôt que dans des champs de premier rang, parce que
`finding.Finding` ne se modifie pas depuis ici.

`labels` est donc la voie normale d'extension tant qu'un besoin n'est pas partagé
avec pitstop.

## Invariants

- Aucun moteur, rendu ou scoring local dans Pépin.
- Aucun `replace` vers scankit dans `go.mod`.
- Toute donnée propre à Pépin voyage dans `labels`, pas dans un champ de scankit.

*Non automatisable : on ne sait pas écrire un test qui distingue « du code qui
évalue une politique » de « du code qui fait autre chose ». Un moteur local
réintroduit se voit à la revue et à l'arborescence. L'absence de `replace` se lit
en une ligne de `go.mod`. C'est une limite assumée : cette décision repose sur la
revue.*

## Validation

*Aucune garde automatique.* Un moteur local réintroduit se verrait à la revue et à
l'arborescence, pas par un test : on ne sait pas écrire un test qui distingue « du code
qui évalue une politique » de « du code qui fait autre chose ». `go.mod` se relit,
et l'absence de `replace` y est visible en une ligne.

C'est une limite assumée de cette décision : elle repose sur la revue.

## Liens

`CLAUDE.md` §9 · limitation connue : SARIF ne sait pas exprimer une suppression
(issue #94), qui se règle en amont.

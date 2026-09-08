# ADR-0017 — La provenance atteste qu'un champ a été CHERCHÉ, et cette attestation peut faire conclure

```yaml
status: Accepted
date: 2026-09-08
scope:
  - assessment
  - model
```

## Contexte

L'ADR-0007 a posé la provenance comme index parallèle et lui a donné trois
invariants, dont le premier : **« La provenance ne modifie jamais un statut. »**

Cet invariant est **déjà faux**, et il l'est depuis la construction en deux passes
de la carte des attributs collectés. Mesuré sur deux inventaires qui ne diffèrent
que par une entrée de provenance :

| `user_data` absent des `attributes` de la seconde VM… | Verdict |
|---|---|
| …mais **présent en provenance** (cherché, non exposé) | `pass` |
| …et **absent de la provenance** (jamais cherché) | `not-evaluated` |

La garde censée protéger l'invariant, `TestProvenanceNeverMovesAVerdict`, ne l'a
pas vu : elle éprouve la passe d'ANNOTATION (`WithProvenance`), qui n'écrit que
`evidence`, et non le verrou de capacité, qui décide.

Ce n'est pas un accident à corriger. C'est la seule donnée du modèle qui porte une
distinction dont les verdicts ont besoin :

| Situation | Ce que la règle voit | Ce que ça vaut |
|---|---|---|
| Le pointeur était nul : aucune expiration définie | attribut absent | une **observation** |
| L'endpoint n'a pas été appelé, ou le champ n'est pas mappé | attribut absent | une **lacune** |

La première doit produire un écart ; la seconde ne le doit pas (ADR-0014). Sans la
provenance, elles sont indiscernables, et `iam_accesskey_expiration_set` émettait
un `critical` dans les deux cas — un écart fabriqué depuis un champ jamais cherché.

## Décision

**La provenance atteste qu'un champ a été CHERCHÉ.** Cette attestation est une
observation à part entière, et l'**assessment** a le droit d'en conclure.

L'invariant 1 de l'ADR-0007 est **révisé** et remplacé par :

> La provenance ne modifie jamais un statut **depuis une règle**. Seul
> l'assessment la lit, et il ne s'en sert que pour distinguer une absence
> ATTESTÉE (cherchée, non exposée) d'une absence NON ATTESTÉE (jamais cherchée).

Trois bornes, qui font que cette lecture n'est pas une porte ouverte :

1. **Aucune règle ne lit la provenance.** L'entrée des règles reste identique
   octet pour octet, ce qui était la raison d'être de l'index parallèle.
2. **Une absence de provenance ne conclut jamais seule.** Un inventaire reçu d'un
   tiers n'en porte aucune ; il ne doit donc rien perdre. La dégradation n'a lieu
   que si la ressource porte une provenance pour d'AUTRES attributs — c'est ce qui
   prouve que Pépin l'a collectée et savait quoi y chercher.
3. **Le sens de la conclusion est borné.** L'attestation ne peut que RETIRER une
   affirmation (un `fail` devient `not-evaluated`, un `pass` reste soumis au
   verrou). Elle ne crée jamais un écart, et ne transforme jamais un
   `not-evaluated` en `pass` sur sa seule foi.

Le reste de l'ADR-0007 est **conservé, non remplacé** : index parallèle plutôt
qu'enveloppe, source `api` nommant l'appel réellement servi, origine jamais
fabriquée.

## Justification

La distinction est déjà dans le modèle, déjà scellée dans le bundle, et déjà
documentée par l'ADR-0007 lui-même :

> Une clé peut y exister sans que l'attribut soit dans `attributes` : c'est un
> champ **cherché et non exposé** par la source.

Refuser de s'en servir ne préserverait pas l'invariant — il est déjà contredit —
mais laisserait un `critical` se déclencher sur un champ que personne n'a demandé.
Entre un invariant qui décrit mal le code et un faux positif `critical`, c'est
l'invariant qui doit bouger, et il doit bouger **par écrit**.

La séparation des deux passes porte la décision dans le code : `WithProvenance`
annote et ne décide pas ; `WithAttestedAbsence` décide et n'annote pas. Une
fonction dont le nom dit qu'elle déplace un verdict est plus honnête qu'un
invariant qui affirme le contraire de ce que fait le programme.

## Alternatives écartées

**Faire lire la provenance aux règles.** Rejeté, pour la raison exacte de
l'ADR-0007 : cinquante-neuf règles à réécrire, et une règle réécrite est une règle
qui peut changer de verdict. La distinction se traite là où les verdicts se
statuent déjà (ADR-0006, ADR-0015), pas dans la logique métier.

**Retirer le secours par provenance pour restaurer l'invariant.** Rejeté : le cas
mesuré (`user_data` cherché sur cinq VM, exposé sur trois) redeviendrait
`not-evaluated` alors qu'il n'y a rien à déclarer. On perdrait une conclusion juste
pour sauver une phrase.

**Remplacer entièrement l'ADR-0007.** Rejeté : deux de ses trois invariants et
toute sa décision structurelle restent vrais. Les réécrire dans un nouvel ADR
créerait deux textes à maintenir sur la même décision, ce que ce registre existe
pour éviter. La révision est donc PARTIELLE et nommée comme telle — l'ADR-0007
garde son statut `Accepted` et porte un renvoi sur l'invariant révisé.

**Exiger la présence de l'attribut (`requiredAttr`) pour ce contrôle.** Rejeté :
cela rendrait `iam_accesskey_expiration_set` aveugle au cas même qu'il existe pour
voir, puisque chez Scaleway l'absence EST l'expiration non définie
(`ExpiresAt *time.Time`, contrat `access_key`, `etat: verifie`).

## Conséquences

Un écart né d'une absence non attestée devient `not-evaluated` et dit pourquoi.
C'est un verdict de moins, et c'est voulu : il valait un `critical` inventé.

Un inventaire sans aucune provenance — export d'un tiers — ne change pas de
comportement. C'est la borne 2, et elle est mesurée plutôt que supposée.

Un collecteur qui cesse de mapper un champ ne produit plus une vague d'écarts
`critical` : il produit des `not-evaluated` qui nomment le champ manquant. La
panne devient lisible au lieu d'être bruyante.

## Invariants

- Aucune règle ne lit la provenance.
- L'attestation ne crée jamais un écart ; elle ne peut qu'en retirer un.
- Une ressource sans aucune provenance ne subit aucune dégradation.
- `WithProvenance` n'écrit que `evidence`, et ne déplace aucun statut.

*Gardes : `TestProvenanceNeverMovesAVerdict` (passe d'annotation),
`TestNoRuleReadsProvenance`, `TestAttestedAbsenceOnlyRemovesFindings`,
`TestInventoryWithoutProvenanceIsUntouched`.*

## Validation

Comparaison des verdicts avant/après sur toutes les fixtures du dépôt, et les deux
cas construits de l'issue #121 — l'absence attestée reste un écart, l'absence non
attestée devient `not-evaluated`.

## Liens

ADR-0006, ADR-0007 (invariant 1 révisé ici), ADR-0014, ADR-0015 · issue #121.

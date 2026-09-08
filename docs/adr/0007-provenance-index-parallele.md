# ADR-0007 — La provenance est un index parallèle, pas une enveloppe autour de chaque valeur

```yaml
status: Accepted
date: 2026-08-23
scope:
  - model
  - assessment
```

## Contexte

Pour dire ce que Pépin a **observé** plutôt que ce qu'il suppose, chaque attribut
devait porter son origine : appel d'API servi, littéral de descripteur, valeur
dérivée.

## Décision

`model.Resource` gagne une carte `provenance`, indexée par les **mêmes** noms
d'attributs, sœur de `attributes` — jamais imbriquée dans une valeur.

Une source `api` nomme **la requête réellement servie**, lue sur la requête après
une réponse valide, jamais celle que la spec déclare.

## Justification

Envelopper chaque valeur (`{"value": false, "observed": true, …}`) aurait imposé de
réécrire les 59 règles, et **une règle réécrite est une règle qui peut changer de
verdict**. Avec un index parallèle, l'entrée des règles est identique octet pour
octet : la non-régression devient structurelle au lieu d'être espérée.

Une provenance qui désigne un appel qui n'a pas eu lieu donnerait l'**apparence** de
la traçabilité, ce qui est pire que son absence.

## Alternatives écartées

**Envelopper chaque valeur.** Rejeté : réécriture des 59 règles.

**Dériver la provenance de la spec YAML.** Rejeté : elle nommerait un appel non
émis.

**Rendre la provenance optionnelle derrière un drapeau.** Rejeté : un dossier de
preuve dont la provenance est optionnelle est un dossier qu'on peut produire sans
elle.

## Conséquences

L'`input.json` d'un bundle grossit d'une entrée par attribut mappé. Coût assumé :
si la taille devient un problème, la correction honnête est une **compaction** de
l'attestation, pas sa désactivation.

La provenance **ne déplace aucun verdict**. Elle rend visible qu'un contrôle
franchit son verrou grâce à un littéral de descripteur plutôt qu'à une mesure —
l'écart devient *détectable*, ce qui est tout l'objet.

## Invariants

- ~~La provenance ne modifie jamais un statut.~~ **RÉVISÉ par l'ADR-0017.** Cet
  invariant était déjà contredit par le verrou de capacité, qui traite un attribut
  cherché et non exposé comme une observation. L'ADR-0017 le remplace par : *la
  provenance ne modifie jamais un statut depuis une RÈGLE ; seul l'assessment la
  lit, pour distinguer une absence attestée d'une absence jamais cherchée.* Le
  reste du présent ADR est inchangé.
- Une source `api` nomme un appel réellement servi.
- Une origine absente n'est jamais fabriquée.

*Gardes : `TestProvenanceNeverMovesAVerdict` (passe d'annotation seule, cf.
ADR-0017), `TestProvenanceNamesTheCallThatActuallyHappened`.*

## Validation

Comparaison des verdicts avant/après sur toutes les fixtures, dans les deux langues.

## Liens

ADR-0003, ADR-0014.

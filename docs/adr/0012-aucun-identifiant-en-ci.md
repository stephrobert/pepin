# ADR-0012 — Aucun identifiant cloud en CI ; l'émulateur est la surface de mesure

```yaml
status: Accepted
date: 2026-08-23
scope:
  - testing
  - security
```

## Contexte

Valider un chemin de collecte « live » demande une API réelle. Mettre des
identifiants cloud en CI les expose à toute la chaîne d'approvisionnement du dépôt
— actions tierces comprises.

## Décision

**Aucun identifiant cloud n'entre en CI.** Un scan live est un geste de mainteneur,
lancé localement, dont le résultat est **consigné et daté** ; le préflight peut
juger ce relevé périmé **sans jamais détenir de secret**.

La surface de mesure automatisable est l'**émulateur local** des trois clouds
souverains, atteint par un proxy d'enregistrement, plus les **plans Terraform**, qui
ne provisionnent rien.

## Justification

Un CSPM dont la CI porte les clés d'un tenant réel est une cible : il concentre en
un point ce qu'il est censé protéger.

Et le non-provisionnement est préférable au provisionnement encadré : un plan
Terraform suffit à valider un mapping, sans coût, sans surface d'exposition, sans
rien à détruire.

## Alternatives écartées

**Des identifiants à droits réduits en CI.** Rejeté : « réduits » n'est pas
« inoffensifs », et la réduction dérive.

**Un endpoint de collecte surchargeable** pour rediriger les appels. Rejeté :
chaque requête de collecte porte une clé en en-tête ; un endpoint configurable est
un moyen d'envoyer les clés du tenant vers un hôte arbitraire. L'audit de livraison
l'avait identifié comme tel. Le proxy obtient le même résultat **sans créer cette
surface**, parce que le client honore déjà `HTTPS_PROXY`.

## Conséquences

Ce qui n'est pas mesuré contre une API réelle est **déclaré comme tel** : la colonne
« live » de `docs/coverage.md`, les permissions minimales en `a_verifier`, la
classification d'un vrai `403`.

**Un émulateur prouve ce que Pépin fait, pas ce que le cloud répond.** Confondre les
deux fabriquerait la fausse confiance que tout le reste combat.

Corollaire de sécurité : toute ressource provisionnée pour un test **doit** être
détruite, et l'enregistrement d'une session porte l'inventaire réel d'un tenant —
aucun n'entre au dépôt sans relecture et assainissement, par **liste blanche** de ce
qui est conservé. Une liste noire a été éprouvée et a échoué.

## Invariants

- Aucun secret dans un fichier versionné.
- Aucune ressource cloud laissée derrière un test.
- Ce qui n'est pas observé est déclaré non observé.

*Gardes : `gitleaks` et `trufflehog` en CI ; le préflight juge la fraîcheur du relevé.*

## Validation

`mise run secrets` sur l'historique complet.

## Liens

`CLAUDE.md` §1.1 · skill `tracer-api` · ADR-0010.

# ADR-0023 — Un statut HTTP ne classe pas un refus ; le code d'erreur documenté, si

```yaml
status: Accepted
date: 2026-09-12
scope:
  - model
  - collect
```

## Contexte

L'état de collecte range chaque échec dans une **classe** — `permission_denied`,
`not_found`, `timeout`… — et cette classe commande la phrase que lit l'opérateur.
Elle était dérivée du **statut HTTP** seul, avec un repli sur `unavailable` pour
tout `4xx` non réclamé par une classe plus fine.

Trois observations, faites sur les plans de contrôle réels, ont montré que ce
raisonnement ne pouvait pas tenir.

**Le statut ne dit pas la même chose d'un fournisseur à l'autre.** Pour un même
refus, Scaleway répond `401`, Exoscale `403`, Outscale `400` (issue #91).

**Chez un même fournisseur, le statut range les deux erreurs à l'envers.** La
table officielle d'Outscale documente l'erreur d'**authentification** en `400`
(code `4120`, `ErrorAuthenticationexception`) et l'erreur d'**autorisation** en
`401` (code `5`, `ErrorNotAuthorized`). Classer par statut donne donc « service
indisponible » à la première et « privilège insuffisant » à la seconde : les deux
erreurs possibles, dans les deux sens.

**Le repli `unavailable` était un mensonge visible.** Le relevé de canari rangeait
les dix-huit endpoints Outscale en « service indisponible » alors que tous
avaient répondu — le `4xx` étant précisément la preuve que le service fonctionne.

La distinction signature/droit ne pouvait pas se trancher depuis le dépôt : des
identifiants synthétiques ne produisent jamais que le refus d'authentification.
Elle a été mesurée le 2026-09-12 sur `eu-west-2`, avec une identité EIM créée
pour la mesure, ne portant que `api:ReadVms`, puis détruite — un appel autorisé
joué d'abord comme témoin.

| Situation du compte de scan | HTTP | `Type` · `Code` |
|---|---|---|
| Clé d'accès inexistante | `400` | `InvalidParameterValue` · `4120` |
| Clé existante, signature invalide | `401` | `AccessDenied` · `1` |
| Clé valide, droit manquant | `403` | `AccessDenied` · `4` |

## Décision

La classe d'un refus se dérive du **code d'erreur que le fournisseur documente**
quand il y en a un, et du statut seulement à défaut. Un `4xx` non classé rend
`rejected` (« l'API a répondu et refusé »), jamais `unavailable`, qui ne désigne
plus que ce qui n'a pas répondu.

## Justification

L'opérateur ne lit pas une classe : il lit une phrase, et il agit. Les trois
gestes sont incompatibles — corriger des **identifiants**, corriger des
**droits**, attendre une **panne**. Une classe qui envoie vers le mauvais geste
coûte plus que pas de classe du tout, parce qu'elle inspire confiance.

Le statut HTTP est une convention que chaque API applique à sa façon ; le code
d'erreur est un **contrat écrit**, que le fournisseur publie et tient. Entre une
convention et un contrat, l'ancrage §2 tranche depuis toujours en faveur du
contrat.

## Alternatives écartées

**Mapper `400 → permission_denied` chez Outscale.** Rejeté par l'issue #91
elle-même, avant toute mesure : un vrai défaut de requête serait devenu un
problème de droits. C'était remplacer un mensonge par le mensonge inverse.

**Garder `unavailable` comme repli des `4xx`.** Rejeté : c'est la formulation qui
a produit le défaut. Un `4xx` prouve que le service répond.

**Introduire une seule classe « refus, cause indéterminable ».** C'était la piste
de repli de l'issue, valable tant que la distinction n'était pas mesurée. Elle
l'est : s'en tenir là aurait jeté une mesure qui a coûté un compte réel.

**Déduire `unauthenticated` d'un `401`, et `permission_denied` d'un `403`, par
sémantique HTTP.** Rejeté, et c'est le piège le plus séduisant : la table
d'Outscale place `ErrorNotAuthorized` — un droit manquant — en `401`. La
sémantique HTTP est ici démentie par la doc du fournisseur lui-même.

**Mapper le `403 · code 4` mesuré.** Rejeté : ce code n'est PAS dans la table
publiée. Il est mesuré, pas documenté, et l'inscrire dans le code reviendrait à
affirmer un contrat que personne ne tient (CLAUDE.md §2). Son statut `403` le
classe déjà correctement.

## Conséquences

Chaque fournisseur dont l'enveloppe d'erreur doit être lue coûte une entrée dans
la table de correspondance, et cette entrée exige une **source citée**. C'est le
coût voulu : il rend impossible d'ajouter une correspondance devinée.

L'éventail de valeurs de `error` s'élargit, donc `InventoryFormat` monte (v14) et
un consommateur qui les énumère doit être averti — ADR-0003.

Ce que la décision ne change pas : aucun verdict ne bouge sur un tenant inchangé.
Un refus dégradait un contrôle en `not-evaluated` ; il le fait toujours.

## Invariants

- Un `4xx` ne se classe **jamais** `unavailable` ; seul un service qui n'a pas
  répondu le mérite — *garde : `TestAnAnsweringServiceIsNeverCalledUnavailable`*
- Une correspondance code → classe cite la table officielle qui la documente ; un
  code non documenté retombe sur le statut — *garde :
  `TestTheThreeMeasuredOutscaleRefusalsDoNotMeanTheSameThing`*
- Un corps que la table ne reconnaît pas ne déplace aucune classe — *garde :
  `TestAnUnrecognizedBodyLeavesTheStatusInCharge`*
- Le droit requis ne se nomme que sur `permission_denied` : sur une clé inconnue,
  il enverrait élargir une politique attachée à une identité absente — *garde :
  `TestOnlyAMissingRightNamesTheRequiredGrant`*

## Validation

`mise run test`. Le relevé de canari (`references/canary/outscale.yaml`) rejoue la
mesure contre le vrai plan de contrôle à chaque qualification de release : ses
endpoints ressortent `unauthenticated`, ce qui est exact — le canari n'a pas de
clé — et ce qui prouve du même coup qu'il ne peut rien dire du refus de DROIT.

## Liens

Issue #91 · `internal/collect/status.go` · `internal/model/collection.go` ·
`internal/assess/collection.go` · <https://docs.outscale.com/api-errors.html> ·
ADR-0003, ADR-0006, ADR-0012.

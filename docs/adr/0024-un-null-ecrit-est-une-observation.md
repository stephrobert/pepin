# ADR-0024 — Un `null` écrit par un plan est une observation ; une clé absente ne l'est pas

```yaml
status: Accepted
date: 2026-09-21
scope:
  - collectors
  - model
```

## Contexte

Sur une réponse d'API, un champ absent peut vouloir dire deux choses — le service ne
l'expose pas, ou il ne l'a pas renvoyé cette fois-ci — et Pépin refuse donc de
conclure. C'est le verrou de capacité, et il est juste.

Sur un **plan Terraform**, la situation est différente, et mesurable. Terraform
distingue deux silences :

| Ce que l'auteur a fait | `planned_values` | `after_unknown` |
|---|---|---|
| Argument optionnel **non écrit** | clé présente, valeur `null` | — |
| Valeur **connue après apply** | clé **absente** | `true` |

Mesuré sur les plans complets du dépôt, croisé attribut par attribut avec le schéma
des providers : **62 valeurs nulles, toutes portées par un attribut `optional` sans
`computed`, zéro contre-exemple**. Aucun attribut sorti `null` n'était `computed`.

La règle qui en découle est donc établie, sans exception connue :

> **Un `null` dans `planned_values` signifie toujours « l'auteur n'a pas écrit cet
> argument ».**

Ce qu'il **vaut**, en revanche, n'est pas dans le plan : c'est ce que le provider
envoie à l'API lorsque l'argument est omis, et cela se lit dans son code.

## Décision

Un `null` écrit par le plan est une **observation**. La valeur qu'il vaut se
**déclare**, attribut par attribut, dans `mapping_terraform`, par le transform
`default:` déjà existant — et chaque déclaration **cite la source du provider** qui
l'établit.

Une clé **absente** ne se déclare jamais : le plan ne la connaît pas.

## Justification

Ce n'est pas une estimation. La source EST lue, elle dit `null`, et ce que cette
omission produit est écrit dans le code du provider. L'ADR-0014 interdit de fabriquer
une valeur qu'on n'a pas observée ; il n'interdit pas de lire une observation et de
lui appliquer un contrat publié — c'est exactement le raisonnement qu'ADR-0022 a tenu
pour les références déclarées.

Le cas qui tranche : `scaleway_rdb_instance.encryption_at_rest` est `Optional` sans
`Default`, et la création envoie **toujours**
`Encryption: &rdb.EncryptionAtRest{Enabled: d.Get(...)}` — donc `false` quand
l'argument est omis. Le plan **prouve** que la base sera créée non chiffrée, et Pépin
répondait « ne peut conclure ». Comparer avec `disable_backup`, qui porte un
`Default: false` au schéma : Terraform l'inscrit alors dans le plan, et aucune
déclaration n'est nécessaire. La capacité de conclure dépendait donc d'un accident
d'écriture du provider, pas de ce que le plan démontre.

## Alternatives écartées

**Une règle générale : tout `null` vaut le défaut du provider.** Rejeté, et c'est le
rejet le plus important. Sur les onze plans du dépôt, **vingt des vingt-cinq** cibles
d'une telle règle sont des champs de règles de groupe de sécurité, qui alimentent un
contrôle CRITICAL. Un `ip_range` non écrit ne vaut pas `0.0.0.0/0` par défaut : il
veut dire que la règle désigne un groupe de sécurité au lieu d'une plage. Une règle
générale fabriquerait des écarts critiques sur des tenants tiers inchangés.

**Lire `after_unknown` pour distinguer les deux silences.** Rejeté : ce bloc vit dans
`resource_changes`, que `CheckPlanShape` retire des plans de référence parce que
c'est là qu'un plan réel porte ses identifiants. Le distinguo est de toute façon
disponible dans `planned_values`, qui est le seul bloc que Pépin lit.

**Déclarer par descripteur quels champs sont « known after apply ».** Déjà rejeté par
ADR-0022, et le rejet tient : le schéma du provider le dit, et une déclaration de
plus pourrit en silence.

**Déclarer sans citer la source.** Rejeté après mesure : les quatre premières
déclarations du dépôt ont été écrites ainsi, et l'une d'elles portait sur un attribut
`required` — elle ne pouvait pas tirer, et personne ne s'en est aperçu. Une
déclaration fausse ne se voit pas : elle produit un verdict plausible.

**Ne rien construire, au motif qu'aucun plan du dépôt ne l'exerce.** Rejeté : le
corpus est écrit pour **exercer** les règles, donc il écrit les arguments. Il mesure
la présence, jamais l'omission, qui est le cas courant d'un HCL tiers. Confondre
corpus de test et usage réel conduirait à ne jamais couvrir le cas normal.

## Conséquences

Une déclaration coûte une source vérifiée dans le code du provider. C'est le coût
voulu : il rend impossible d'en poser une par analogie.

Les verdicts bougent sur des tenants inchangés, et cela s'écrit au CHANGELOG :
`database_encryption_at_rest_enabled` passe de `not-evaluated` à `fail` sur les
tenants de référence dont le plan omet l'argument.

Ce que la décision ne change pas : une clé absente ne projette toujours rien, des
deux côtés, et le verrou de capacité fait son travail.

**Un cas adjacent reste ouvert** : la spec de collecte **live** d'Exoscale porte un
`default:0` sur `max_session_ttl` qui n'est pas sourcé. La porte de sourçage ne
couvre que `mapping_terraform`, parce que la sémantique d'un `null` dans une réponse
d'API n'est pas celle d'un plan. Il est signalé ici pour ne pas être oublié.

## Invariants

- Un `default:` ne s'applique QUE si la clé est présente et sa valeur nulle ; une clé
  absente ne projette rien — *garde : `TestNoSpecFabricatesAnAttributeFromNothing`,
  qui couvre désormais les deux sources*
- Un `default:` sur un attribut `computed` est refusé : son silence part en
  `after_unknown`, donc le plan ne le connaît pas — *garde :
  `TestProviderMappingsMatchSchema`*
- Un `default:` sur un attribut `required` est refusé : il ne tire jamais, et un
  no-op trompeur fait croire qu'un cas est traité — *garde : idem*
- Toute déclaration de `mapping_terraform` cite sa source — *garde :
  `TestEveryTerraformDefaultCitesItsSource`*
- Un booléen garde le même type quelle que soit la source — *garde : transform
  `to_bool`, exercé par les scénarios de véracité*

## Validation

`mise run test`. Le contrôle `database_encryption_at_rest_enabled` sur le tenant
`ducklake` est la mesure de bout en bout : `not-evaluated` sans la déclaration,
`fail` avec, et la raison affichée nomme la base.

## Liens

Issue #243 · ADR-0003, ADR-0006, ADR-0014, ADR-0017, ADR-0022 ·
`internal/collect/engine.go` (`default:`, `to_bool`) · `internal/tfmap/schema.go`.

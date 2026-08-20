> [🇬🇧 English](control-configuration.md) · 🇫🇷 Français

# Configurer les contrôles — et ce qu'un assouplissement fait perdre

Certains contrôles doivent être réglables. Une convention d'étiquetage n'est pas une norme ;
une fenêtre de fraîcheur est une décision d'exploitation ; une heuristique générique de secret
est plus bruyante qu'un bloc PEM. Un contrôle qu'on ne peut pas régler finit désactivé, et un
contrôle désactivé ne mesure rien.

Mais chaque réglage est une poignée qui permet de fabriquer du vert. Desserrer un seuil,
retirer une étiquette exigée, allonger un délai — et continuer d'afficher la même
correspondance CIS ou SecNumCloud. Un `pass` obtenu en abaissant l'exigence est très
exactement le `PASS` non prouvé que ce projet refuse de produire.

Pépin lie donc la configuration à la correspondance normative. **Vous pouvez abaisser la
barre. Vous ne pouvez pas l'abaisser et garder le badge.**

## Un seul fichier, deux sections

Il n'y a qu'un fichier de politique. Il porte les réglages des contrôles et les dérogations,
parce que les deux répondent à la même question — *ce que cette organisation assume
sciemment* — et se relisent au même moment, par les mêmes personnes. Deux fichiers auraient
deux cycles de revue, et un utilisateur qui doit tenir trois fichiers de politique en fera
diverger deux.

```yaml
# pepin-policy.yaml
controls:
  tagging:
    required_tags: [CostCenter, Project, Env, Owner]
    network_required_tags: [Owner, Project, Env]
    aliases:
      Owner: [Owner, team, responsible, contact]
    resource_types: [compute_instance, blockstorage_volume, object_storage_bucket]
  snapshots:
    max_age_days: 7
    accepted_states: [completed, created]
  secrets:
    min_confidence: low

exceptions:
  - control: objectstorage_bucket_public_access
    resource: public-assets
    justification: "Bucket de diffusion publique assumé, contenu non sensible"
    expires_at: 2026-12-31
    owner: platform-security
    approved_by: security@example.org
```

```console
$ pepin scan scaleway --terraform plan.json --policy pepin-policy.yaml
```

`--exceptions` est le nom historique du même fichier et lit le même schéma : une invocation
existante continue de fonctionner, et un fichier de dérogations peut gagner une section
`controls:` sans changer de ligne de commande. **Les deux drapeaux sont mutuellement
exclusifs** — accepter deux fichiers de politique différents, c'est garantir que l'un des deux
dérivera.

Chaque section est facultative. Une section absente laisse le profil par défaut intact : un
fichier de politique partiel n'assouplit que ce qu'il nomme.

## Le profil par défaut est une recommandation, pas une norme

Aucune convention d'étiquetage ne fait autorité, et Pépin ne prétend pas le contraire. Ce que
le profil par défaut encode, ce sont les **questions auxquelles un inventaire doit savoir
répondre** — qui paye, pour quoi, à quel stade, qui répond —, jamais des mots précis.

| Réglage | Défaut | Ce qu'il désigne |
|---|---|---|
| `tagging.required_tags` | `CostCenter, Project, Env, Owner` | étiquettes exigées sur une ressource facturable |
| `tagging.network_required_tags` | `Owner, Project, Env` | étiquettes exigées sur un réseau (cartographie) |
| `tagging.aliases` | voir plus bas | autres écritures qu'un nom logique accepte |
| `tagging.resource_types` | 8 types, listés plus bas | où l'étiquetage est exigé |
| `snapshots.max_age_days` | `7` | fenêtre de fraîcheur d'une snapshot |
| `snapshots.accepted_states` | `completed, created` | états natifs d'une snapshot exploitable |
| `secrets.min_confidence` | `low` | tout signaler, heuristiques génériques comprises |

**La comparaison est insensible à la casse et aux séparateurs.** `cost-center`,
`cost_center`, `Cost Center` et `CostCenter` sont la même exigence — un outil qui crie au loup
sur une typographie finit désactivé. Par-dessus, les alias élargissent chaque nom logique :

| Nom logique | Écritures acceptées |
|---|---|
| `CostCenter` | `CostCenter`, `cost-center`, `cc`, `billing-code`, `billing` |
| `Project` | `Project`, `app`, `application`, `service` |
| `Env` | `Env`, `environment`, `stage` |
| `Owner` | `Owner`, `team`, `responsible`, `contact` |

Les noms du tableau sont des **noms d'affichage** : ce sont eux qu'un message de finding
énumère comme manquants. La correspondance, elle, passe par les alias normalisés.

**Quels types de ressources sont visés, et pourquoi.** Le critère est *facturable et
étiquetable* : une ressource qui coûte sans propriétaire connu est un coût orphelin autant
qu'un risque orphelin.

- Dans le périmètre : `compute_instance`, `blockstorage_volume`, `blockstorage_snapshot`,
  `compute_image`, `load_balancer`, `object_storage_bucket`, `managed_database`,
  `kubernetes_cluster`.
- Hors périmètre, avec les raisons : `network`, `subnet`, `network_peering`,
  `security_group*` (non facturés en propre ; le réseau a sa propre exigence de cartographie,
  CLD-NET-5) ; `iam_*`, `access_key`, `api_access_*` (ni facturés ni porteurs d'étiquettes) ;
  `governance_provider` (ressource synthétique, pas une ressource du tenant) ; `k8s_*`
  (portée intra-cluster, hors posture cloud).

## Ce qu'un assouplissement fait perdre

Chaque correspondance du référentiel porte les contraintes de configuration sous lesquelles
elle vaut :

```yaml
- code: blockstorage_volume_snapshots_exist
  scsl: [CLD-STO-3]
  frameworks:
    secnumcloud_3_2: ["12.5"]
    cis_controls_v8: ["11.2"]
    iso_27001_2022: ["A.8.13"]
  config_requise:
    - parametre: snapshots.max_age_days
      contrainte: au_plus_le_defaut
    - parametre: snapshots.accepted_states
      contrainte: sous_ensemble_du_defaut
```

Quatre sens de contrainte, chacun nommant le côté du défaut où la promesse survit :

| Contrainte | Tient tant que | L'assouplir signifie |
|---|---|---|
| `au_plus_le_defaut` | la valeur ne dépasse pas le défaut | une valeur au-delà tait ce que l'exigence demandait de voir |
| `superset_du_defaut` | la valeur contient au moins le défaut | en retirer un membre, c'est cesser de vérifier ce que l'exigence demande |
| `sous_ensemble_du_defaut` | la valeur reste contenue dans le défaut | l'élargir, c'est accepter ce que l'exigence rejette |
| `au_moins_aussi_strict_que_le_defaut` | chaque exigence du profil par défaut a encore une contrepartie au moins aussi stricte | une exigence a été retirée, ou ce qu'elle accepte a été élargi |

Le dernier est celui qu'emploie le profil d'étiquetage, et il ne revient pas à comparer des
noms. Le défaut exige « la question de l'environnement est répondue par l'une des écritures
`env`, `environment`, `stage` ». Une organisation qui écrit `environment` accepte **moins**
d'écritures, donc exige davantage : c'est un durcissement. Comparer les noms l'aurait signalée
comme assouplie et lui aurait retiré sa correspondance normative pour avoir resserré sa propre
convention — le faux positif le plus coûteux qui soit, puisqu'il punit le bon comportement.

**Durcir n'est pas assouplir.** Exiger une étiquette de plus, raccourcir la fenêtre, rétrécir
les états acceptés ou les écritures acceptées : la correspondance tient et rien n'est signalé.
La contrainte ne dit pas « ne touche à rien », elle dit de quel côté du défaut la promesse
survit.

Quand la configuration effective sort d'une contrainte, le contrôle est **assoupli**, et cinq
choses arrivent en même temps :

1. le résultat **perd ses références normatives** dans l'assessment — il ne prétend plus
   couvrir CLD-STO-3, CIS 11.2 ni ISO A.8.13 ;
2. le résultat porte `labels.config_relaxed`, `labels.config_relaxed_detail` et
   `labels.references_dropped` ;
3. la preuve (`evidence.observed`, donc l'OSCAL) énonce l'assouplissement en toutes lettres ;
4. le terminal imprime un bloc `CONFIGURATION ASSOUPLIE` nommant le réglage, le défaut, la
   valeur effective et les correspondances abandonnées ;
5. `--format json` publie `config.relaxations` et `config.dropped_references`, et un bundle
   scellé gagne `config.json` plus une entrée `config` à son manifeste — les deux couverts par
   `checksums.txt`, donc l'empreinte du dossier dépend des réglages.

Le bandeau de verdict change lui aussi, et `--strict` rend le code de sortie `3`. Un pipeline
qui vend de la conformité ne doit pas rendre `0` sur un contrôle qu'il a lui-même abaissé.

Le statut, lui, ne change **pas**. Le contrôle a bel et bien été évalué — contre une barre
plus basse. On retire la seule chose qui a cessé d'être vraie, la correspondance, et on garde
la mesure.

## Régler le seuil bloquant de la détection de secrets en CI

Chaque détection porte son niveau de confiance, dans `labels.confidence` :

| Niveau | Fondement | Exemple |
|---|---|---|
| `high` | confirmé par sa forme | `-----BEGIN … PRIVATE KEY-----` |
| `medium` | préfixe reconnu et format attendu | `ghp_…`, `AKIA…`, `SCW…`, `glpat-…`, JWT |
| `low` | heuristique générique | `password=…`, `api_key=…` |

Le défaut est `low` : tout est signalé. C'est le seul défaut défendable pour un détecteur de
secrets — taire par défaut ce qu'on ne sait pas confirmer, c'est échanger un faux positif
contre un faux négatif, sur le seul sujet où le faux négatif se paye en fuite.

Trier en CI sans changer le scan :

```console
$ pepin scan scaleway --terraform plan.json --format json \
  | jq '[.findings[] | select(.labels.check == "compute_instance_no_secrets_in_user_data")
        | {subject, confidence: .labels.confidence}]'
```

Rendre le scan lui-même plus silencieux — et en assumer le prix :

```yaml
controls:
  secrets:
    min_confidence: medium   # les heuristiques génériques ne sont plus signalées
```

Cela fait tomber la correspondance CLD-CMP-9 / SecNumCloud 10.5, et le rapport le dit.
« Aucun secret en clair » ne se prouve plus dès lors qu'on a choisi de ne pas regarder une
partie de ce qui a été trouvé.

**La valeur détectée n'apparaît jamais**, quel que soit le niveau. Le message n'interpole que
le libellé du motif, jamais ce qui a matché — un rapport voyage en SARIF, dans les artefacts
de CI et dans un bundle scellé, et un détecteur de secrets qui recopie le secret dans son
rapport transforme le rapport en fuite. C'est tenu par des tests, pas par une intention.

## Ce que le contrôle de snapshot ne prouve pas

`blockstorage_volume_snapshots_exist` mesure une chose : un volume en usage a-t-il, dans la
fenêtre configurée, au moins une snapshot dont l'état natif la dit terminée ? L'état est
vérifié, pas seulement la date — une snapshot en erreur ou en cours ne restaure rien.

Il ne prouve **pas** :

- que la snapshot soit **restaurable** — aucune restauration n'est tentée ;
- qu'elle soit **complète** au sens applicatif (bases à chaud, volumes multiples) ;
- qu'une **rétention** soit respectée — une seule snapshot suffit à satisfaire le contrôle ;
- qu'une **politique de sauvegarde** existe.

Un volume sauvegardé autrement — sauvegarde applicative, réplication, service de backup du
fournisseur, outil externe — sera signalé ici. C'est un faux positif assumé : il se traite par
une dérogation datée et justifiée, jamais en désactivant le contrôle. Ce contrôle ne participe
à aucune affirmation « sauvegarde conforme ».

Il n'est pas non plus **évaluable sur un plan Terraform** : `state` y arrive en
`after_unknown` et aucun `blockstorage_snapshot` n'y figure. Le scan rend `not-evaluated`,
avec son motif.

## La configuration voyage avec la preuve

La configuration effective est injectée dans l'inventaire évalué sous la clé `config`,
exactement comme `evaluated_at`. Elle atterrit donc dans l'`input.json` d'un bundle scellé, et
`verify --re-derive` rejoue le même verdict sous la même politique sans qu'on ait à lui
redonner le fichier. Un `input.json` rejoué garde sa propre `config` : le rejeu n'applique
jamais la politique du jour à un dossier d'hier.

Ce qui signifie aussi qu'un scan par défaut n'est jamais muet sur sa politique :
`--format json` publie toujours `config.policy_digest` et `config.effective`. Deux scans que
seul un réglage sépare ne peuvent pas porter la même empreinte.

## Voir aussi

- [Codes de sortie](../reference/exit-codes.fr.md) — dont `--strict` et les dérogations.
- [Bundles de preuve](evidence-bundles.fr.md) — ce qui est scellé, et comment le vérifier.
- [Limites connues](../known-limitations.fr.md) — les angles morts, nommés.

> [🇬🇧 English](outscale.md) · 🇫🇷 Français

# Outscale

Tout ce que cette page affirme sur ce que Pépin collecte est dérivé de
`providers/outscale.yaml`, le descripteur que le scanner lit lui-même. Il n'est pas nécessaire
d'ouvrir ce fichier pour comprendre la couverture.

<!-- pepin:gen provider-outscale-identity -->
| Champ du descripteur | Valeur |
|---|---|
| Description | Outscale (3DS) — VM, BSU, OOS, EIM, security groups, OKS, LBU |
| Portée | cloud |
| Clé de région (`--region`) | `region` |
| Authentification de l'API | `sigv4` |
| Juridiction du siège | FR |
| Établi dans l'UE | oui |
| Contrôle capitalistique | FR |
| SecNumCloud | `qualifie` |
| Exposition extraterritoriale | non |
| Sources de l'ancrage | 3ds.com/newsroom (Outscale 1er cloud SecNumCloud 3.2) ; en.outscale.com/our-certifications |
<!-- /pepin:gen provider-outscale-identity -->

Outscale est le seul des trois dont le descripteur consigne `secnumcloud: qualifie`. Cette
qualification porte sur le **prestataire**, jamais sur votre tenant : un rapport Pépin n'en dit
rien, ni dans un sens ni dans l'autre
([Périmètre et non-objectifs](../concepts/scope.fr.md)).

## Authentification

<!-- pepin:gen provider-outscale-credentials -->
| Clé logique | Variable d'environnement | Défaut |
|---|---|---|
| `access_key` | `OSC_ACCESS_KEY` | — |
| `region` | `OSC_REGION` | `eu-west-2` |
| `secret_key` | `OSC_SECRET_KEY` | — |
| fichier de configuration natif | `~/.osc/config.json` | `osc` |
<!-- /pepin:gen provider-outscale-credentials -->

- L'OAPI est appelée en **signature V4** (service `oapi`), ce qui est un protocole, pas une
  parenté : tout le reste ici est le vocabulaire natif d'Outscale.
- Les mêmes clé d'accès et clé secrète servent à OOS (stockage objet) et à OKS.
- Le fichier de configuration natif est lu quand les variables sont absentes ;
  `OSC_CONFIG_FILE` en surcharge l'emplacement, et `--profile` sélectionne un profil à
  l'intérieur.

```bash
export OSC_ACCESS_KEY=... OSC_SECRET_KEY=...
pepin scan outscale --live --region eu-west-2
```

## Ce qu'appelle un scan live

Chaque endpoint, y compris la liste parente d'une jointure. Tous les appels OAPI sont des
`POST`.

<!-- pepin:gen provider-outscale-live -->
| Type normalisé | Appel | Note |
|---|---|---|
| `access_key` | `POST /ReadAccessKeys` | — |
| `access_key` | `POST /ReadUsers` | liste parente d'une jointure (appelée en premier) |
| `api_access_policy` | `POST /ReadApiAccessPolicy` | — |
| `api_access_rule` | `POST /ReadApiAccessRules` | — |
| `api_access_summary` | `POST /ReadApiAccessRules` | — |
| `blockstorage_snapshot` | `POST /ReadAccounts` | liste parente d'une jointure (appelée en premier) |
| `blockstorage_snapshot` | `POST /ReadSnapshots` | — |
| `blockstorage_volume` | `POST /ReadVolumes` | — |
| `compute_image` | `POST /ReadAccounts` | liste parente d'une jointure (appelée en premier) |
| `compute_image` | `POST /ReadImages` | — |
| `compute_instance` | `POST /ReadVms` | — |
| `iam_policy` | `POST /ReadPolicies` | liste parente d'une jointure (appelée en premier) |
| `iam_policy` | `POST /ReadPolicyVersion` | — |
| `load_balancer` | `POST /ReadLoadBalancers` | — |
| `network` | `POST /ReadNets` | — |
| `network_peering` | `POST /ReadNetPeerings` | — |
| `security_group_rule` | `POST /ReadSecurityGroups` | — |
| `subnet` | `POST /ReadSubnets` | — |
| `object_storage_bucket` | `https://oos.{region}.outscale.com` | API S3 du stockage objet (collecteur Go) |
| `kubernetes_cluster` | `https://api.{region}.oks.outscale.com` | API du Kubernetes managé (collecteur Go) |

URL de base : `https://api.{region}.{host}/api/v1`
<!-- /pepin:gen provider-outscale-live -->

Trois collecteurs sont écrits en Go et n'apparaissent donc pas dans la spec de collecte du
descripteur :

- **Politiques EIM inline** (`internal/eimpolicy`) : `ReadUsers` → `ReadUserPolicies` →
  `ReadUserPolicy`, et `ReadUserGroups` → `ReadUserGroupPolicies`. Une politique inline
  `Action:* / Resource:*` accorde les pleins pouvoirs, et ne lire que les politiques gérées la
  raterait en silence.
- **Stockage objet** (`internal/objectstorage`) : `ListBuckets`, puis par bucket
  `GetBucketAcl`, `GetBucketVersioning`, `GetBucketPolicy`, `GetBucketTagging`,
  `GetObjectLockConfiguration`, `GetBucketEncryption`.
- **OKS** (`internal/oks`) : `GET /api/v2/clusters/all` sur un host distinct, authentifié par
  deux en-têtes simples plutôt qu'en SigV4.

`ReadImages` et `ReadSnapshots` sont **filtrés sur le compte** (`ReadAccounts` alimente
`Filters.AccountIds`). Sans ce filtre, `ReadImages` renvoie tout le catalogue public et chaque
image publique d'un tiers devient un finding : la tempête de faux positifs que le descripteur
existe pour empêcher.

## Permissions minimales pour un scan live

Le vocabulaire d'Outscale est une **politique EIM** : `Effect`, `Action` sous la forme
`<service>:<MéthodeAPI>`, et `Resource`. Refus par défaut, le refus explicite l'emporte.

La plus petite politique documentée couvrant tous les appels OAPI ci-dessus est celle
qu'Outscale publie lui-même, décrite comme « une politique EIM n'autorisant que les appels Read
de l'API OUTSCALE » :

```json
{
  "Statement": [
    {
      "Effect": "Allow",
      "Action": ["api:Read*"],
      "Resource": ["*"]
    }
  ]
}
```

Vérifié dans `docs.outscale.com/en/userguide/Managing-Access-for-Cloud-Automation.html` (la
politique) et `EIM-Policy-Elements.html` (la syntaxe `service:MéthodeAPI` et les codes de
service `api`, `ec2`, `elasticloadbalancing`, `iam`, `directconnect`).

Énumérer les actions plutôt que d'employer `api:Read*` donne la liste suivante, **déduite de la
syntaxe documentée et non recopiée d'une source** :
`ReadSecurityGroups`, `ReadVms`, `ReadImages`, `ReadNets`, `ReadNetPeerings`, `ReadSubnets`,
`ReadVolumes`, `ReadSnapshots`, `ReadAccounts`, `ReadUsers`, `ReadAccessKeys`, `ReadPolicies`,
`ReadPolicyVersion`, `ReadUserPolicies`, `ReadUserPolicy`, `ReadUserGroups`,
`ReadUserGroupPolicies`, `ReadApiAccessRules`, `ReadApiAccessPolicy`, `ReadLoadBalancers`.

**Ce qui n'a pas pu être vérifié, et qui compte.**

- **Le stockage objet n'est pas gouverné par EIM.** Aucun code de service de stockage objet ne
  figure dans les éléments de politique EIM, et la référence complète de l'API EIM ne mentionne
  jamais OOS. Avec les clés du propriétaire du compte, l'accès à OOS vient avec le compte ;
  comment l'accorder à un utilisateur EIM n'est **pas vérifié**. Par une bucket policy, les
  actions de lecture documentées sont `s3:GetBucketAcl`, `s3:GetBucketPolicy`,
  `s3:GetBucketTagging`, `s3:GetBucketVersioning`, `s3:GetEncryptionConfiguration`,
  `s3:GetBucketObjectLockConfiguration`, `s3:ListBucket`, `s3:HeadBucket`, sans aucune action
  pour *lister* les buckets d'un compte, qui est le premier appel que fait Pépin.
- **OKS.** Le modèle de permission de `GET /api/v2/clusters/all` n'est **pas vérifié** ; la
  documentation indique seulement qu'OKS est « partiellement compatible avec les utilisateurs
  EIM », et les deux rôles OKS documentés portent sur les volumes et les snapshots, pas sur les
  clusters. Un appel refusé se dégrade proprement : un avertissement sur la sortie d'erreur, et
  les contrôles Kubernetes reviennent en `not-evaluated`.
- **Les clés du compte racine.** `ReadAccessKeys` sans nom d'utilisateur renvoie les clés de
  l'identité appelante : auditer les clés du compte racine exige donc des identifiants
  propriétaires du compte. C'est consigné dans le descripteur comme une limite d'API ; nous ne
  l'avons pas retrouvé tel quel dans la documentation officielle.
- **Les règles d'accès à l'API.** Si le compte restreint l'accès à l'API par IP source,
  l'adresse du scanner doit y figurer, sinon toute la collecte échoue, avec le code de sortie 2,
  pas un faux vert.

## Ressources Terraform reconnues

<!-- pepin:gen provider-outscale-terraform -->
| Ressource Terraform | Type normalisé | Bloc éclaté |
|---|---|---|
| `outscale_image_launch_permission` | `compute_image` | — |
| `outscale_net` | `network` | — |
| `outscale_policy` | `iam_policy` | — |
| `outscale_security_group_rule` | `security_group_rule` | — |
| `outscale_security_group_rule` | `security_group_rule` | `rules[*]` |
| `outscale_subnet` | `subnet` | — |
| `outscale_vm` | `compute_instance` | — |
<!-- /pepin:gen provider-outscale-terraform -->

```bash
terraform plan -out tfplan && terraform show -json tfplan > plan.json
pepin scan outscale --terraform plan.json
```

Attention à une vraie différence de type entre les deux sources : un plan rend le
`global_permission` d'`outscale_image_launch_permission` en **chaîne** `"true"`, là où l'API
rend un booléen. Les règles normalisent les deux
([Plan Terraform contre scan live](../concepts/terraform-vs-live.fr.md#divergence-2--un-booléen-que-le-plan-rend-en-chaîne)).

## Couverture

<!-- pepin:gen provider-outscale-coverage -->
| Source | ✅ `supported` | ◐ `partial` | ∅ `not-applicable` | ✗ `unsupported` |
|---|---:|---:|---:|---:|
| terraform | 17 | 4 | 4 | 32 |
| live | 39 | 1 | 4 | 13 |
<!-- /pepin:gen provider-outscale-coverage -->

Contrôle par contrôle, avec le motif de chaque case qui n'est pas pleinement supportée, la
source de vérité est la [matrice de couverture](../coverage.fr.md).

### Déclarés non applicables

<!-- pepin:gen provider-outscale-na -->
| Contrôle | Justification consignée au contrat |
|---|---|
| `blockstorage_volume_encryption` | osc-sdk-go/v2 Volume n'expose aucun champ de chiffrement ; le chiffrement au repos est côté invité (EncFS/LUKS), responsabilité du client → non observable côté plateforme (CHF-2). |
| `iam_user_mfa_enabled` | type de ressource « iam_user » absent de l'API outscale |
| `loadbalancer_http_redirect_to_https` | Le LBU Outscale ne peut pas rediriger : `ListenerRule.Action` est documenté « always forward » au contrat OAPI (aucune action de redirection), et aucun attribut de redirection n'existe sur `Listener`. Le mécanisme est inexistant → contrôle non applicable (CHF-1). |
| `objectstorage_bucket_kms_encryption` | OOS chiffre côté serveur en AES256 avec une clé FOURNISSEUR ; il n'existe ni service KMS ni clé maître gérée par le client, donc pas de BYOK à auditer au niveau bucket (CHF-4). NB : l'activation du SSE elle-même est opt-in et observable — elle relève d'un contrôle distinct, pas de ce N/A. |
<!-- /pepin:gen provider-outscale-na -->

### Observables depuis une seule source

<!-- pepin:gen provider-outscale-onesource -->
| Contrôle | Observable uniquement via | Motif du côté aveugle |
|---|---|---|
| `blockstorage_snapshot_not_public` | live | cette source ne produit aucune ressource de type « blockstorage_snapshot » |
| `blockstorage_volume_snapshots_exist` | live | cette source ne produit aucune ressource de type « blockstorage_volume » |
| `compute_instance_deletion_protection` | live | attribut décisif « deletion_protection » non projeté par cette source : garde de capacité, le scan rend « not-evaluated » |
| `governance_resource_region_in_eu` | live | attribut décisif « region » non projeté par cette source : garde de capacité, le scan rend « not-evaluated » |
| `iam_accesskey_expiration_set` | live | cette source ne produit aucune ressource de type « access_key » |
| `iam_account_mfa_enforced` | live | cette source ne produit aucune ressource de type « api_access_policy » |
| `iam_apiaccesspolicy_max_key_expiration` | live | cette source ne produit aucune ressource de type « api_access_policy » |
| `iam_apiaccessrule_defined` | live | cette source ne produit aucune ressource de type « api_access_summary » |
| `iam_apiaccessrule_no_public_cidr` | live | cette source ne produit aucune ressource de type « api_access_rule » |
| `iam_no_root_access_key` | live | cette source ne produit aucune ressource de type « access_key » |
| `kubernetes_cluster_auto_upgrade_enabled` | live | cette source ne produit aucune ressource de type « kubernetes_cluster » |
| `kubernetes_cluster_control_plane_highly_available` | live | cette source ne produit aucune ressource de type « kubernetes_cluster » |
| `kubernetes_cluster_deletion_protection` | live | cette source ne produit aucune ressource de type « kubernetes_cluster » |
| `kubernetes_cluster_not_publicly_accessible` | live | cette source ne produit aucune ressource de type « kubernetes_cluster » |
| `loadbalancer_logging_enabled` | live | cette source ne produit aucune ressource de type « load_balancer » |
| `loadbalancer_ssl_listeners` | live | cette source ne produit aucune ressource de type « load_balancer » |
| `network_peering_cross_organization` | live | cette source ne produit aucune ressource de type « network_peering » |
| `network_securitygroup_default_restrict_traffic` | live | attribut décisif « security_group_name » non projeté par cette source : garde de capacité, le scan rend « not-evaluated » |
| `objectstorage_bucket_default_encryption` | live | cette source ne produit aucune ressource de type « object_storage_bucket » |
| `objectstorage_bucket_object_lock_enabled` | live | cette source ne produit aucune ressource de type « object_storage_bucket » |
| `objectstorage_bucket_public_access` | live | cette source ne produit aucune ressource de type « object_storage_bucket » |
| `objectstorage_bucket_versioning_enabled` | live | cette source ne produit aucune ressource de type « object_storage_bucket » |
<!-- /pepin:gen provider-outscale-onesource -->

## Un scan réel

`examples/outscale/terraform/` contient un module volontairement mal configuré. Son plan est
committé, donc ceci s'exécute sans aucun compte :

```bash
pepin scan outscale --terraform examples/outscale/terraform/plan.json
```

<!-- pepin:gen provider-outscale-scan -->
```text
[…]
  ├────────────┼──────────────────────────────────────────────────┼──────────┼──────────┼───┤
  │ CLD-IAM-1  │ Politique IAM à privilèges administratifs (acti… │ CRITICAL │ outscale │ 2 │
  │ CLD-IAM-12 │ Politique IAM permettant une élévation de privi… │ HIGH     │ outscale │ 1 │
  │ CLD-NET-1  │ SSH (port 22) ouvert à Internet                  │ HIGH     │ outscale │ 1 │
  │ CLD-STO-2  │ Image machine partagée publiquement              │ HIGH     │ outscale │ 1 │
  │ CLD-GVN-1  │ Inventaire et étiquetage incomplets              │ MEDIUM   │ outscale │ 1 │
  │ CLD-NET-3  │ Sous-réseau attribuant une IP publique par défa… │ MEDIUM   │ outscale │ 1 │
  │ CLD-NET-5  │ Réseau non documenté (cartographie non tenue)    │ LOW      │ outscale │ 1 │
  ╰────────────┴──────────────────────────────────────────────────┴──────────┴──────────┴───╯
──────────────────────────────────────────────────────────────────────────────
 Summary

 Verdict : NON CONFORME

 🔴 CRITICAL 1   🟠 HIGH 4   🟡 MEDIUM 2   🔵 LOW 1
──────────────────────────────────────────────────────────────────────────────
```
<!-- /pepin:gen provider-outscale-scan -->

## Limites

- **Les permissions OOS et OKS** sont les deux endroits où une clé strictement au moindre
  privilège n'est pas entièrement documentée (voir plus haut). Les deux se dégradent en
  `not-evaluated`, jamais en faux `pass`.
- **Le chiffrement au repos des volumes BSU** est côté invité et non observable depuis l'API :
  un `not-applicable` déclaré.
- **Le LBU ne sait pas rediriger** : le contrat OAPI documente `ListenerRule.Action` comme
  toujours en transfert, donc le contrôle de redirection HTTP vers HTTPS est non applicable
  plutôt qu'en échec.
- Les limites générales de l'outil sont dans [Limites connues](../known-limitations.fr.md).

## Pour aller plus loin

- [Matrice de couverture](../coverage.fr.md) : chaque contrôle, chaque source.
- [Plan Terraform contre scan live](../concepts/terraform-vs-live.fr.md) : choisir la source.
- [Scaleway](scaleway.fr.md) · [Exoscale](exoscale.fr.md) : les deux autres clouds souverains.
- [GitHub Actions](../guides/github-actions.fr.md) · [GitLab CI](../guides/gitlab-ci.fr.md).

## Comment cette page reste vraie

Les tableaux d'identité, d'identifiants, d'endpoints, de mapping Terraform, de couverture et de
non-applicabilité sont calculés depuis `providers/outscale.yaml` et depuis le référentiel par
`internal/docgen` ; la sortie de scan est une exécution réelle de la fixture committée. Les
appels des collecteurs Go et la section des permissions sont écrits à la main, le descripteur ne
les portant pas, et chacun indique la source où il a été vérifié, ou dit clairement qu'il ne
l'a pas été.

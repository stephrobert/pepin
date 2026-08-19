> [🇬🇧 English](scaleway.md) · 🇫🇷 Français

# Scaleway

Tout ce que cette page affirme sur ce que Pépin collecte est dérivé de
`providers/scaleway.yaml`, le descripteur que le scanner lit lui-même. Il n'est pas nécessaire
d'ouvrir ce fichier pour comprendre la couverture.

<!-- pepin:gen provider-scaleway-identity -->
| Champ du descripteur | Valeur |
|---|---|
| Description | Scaleway — object storage, instances, IAM, security groups |
| Portée | cloud |
| Clé de région (`--region`) | `region` |
| Authentification de l'API | `header` |
| Juridiction du siège | FR |
| Établi dans l'UE | oui |
| Contrôle capitalistique | FR |
| SecNumCloud | `en_cours` |
| Exposition extraterritoriale | non |
| Sources de l'ancrage | iliad.fr (entrée processus SecNumCloud) ; en.wikipedia.org/wiki/Scaleway (capital européen) |
<!-- /pepin:gen provider-scaleway-identity -->

Les champs de souveraineté ne sont pas décoratifs : ils alimentent le contrôle de gouvernance
`CLD-GVN-4`, et leurs sources sont consignées dans le descripteur. `secnumcloud: en_cours`
signifie que le processus de qualification est engagé, **pas** qu'il est obtenu.

## Authentification

Pépin lit les variables d'environnement natives du fournisseur. Il n'invente jamais ses propres
noms, et n'accepte jamais d'identifiants en argument de ligne de commande.

<!-- pepin:gen provider-scaleway-credentials -->
| Clé logique | Variable d'environnement | Défaut |
|---|---|---|
| `access_key` | `SCW_ACCESS_KEY` | — |
| `org` | `SCW_DEFAULT_ORGANIZATION_ID` | — |
| `region` | `SCW_DEFAULT_REGION` | `fr-par` |
| `secret_key` | `SCW_SECRET_KEY` | — |
| `zone` | `SCW_DEFAULT_ZONE` | `{region}-1` |
| fichier de configuration natif | `~/.config/scw/config.yaml` | `scw` |
<!-- /pepin:gen provider-scaleway-credentials -->

- L'API s'authentifie avec la **clé secrète** dans l'en-tête `X-Auth-Token`.
- Les mêmes clé d'accès et clé secrète servent au stockage objet (S3).
- `SCW_DEFAULT_ORGANIZATION_ID` est obligatoire : il est substitué dans les chemins IAM, et sans
  lui la collecte IAM ne peut pas s'exécuter.
- Le fichier de configuration natif est lu quand les variables sont absentes ;
  `SCW_CONFIG_PATH` en surcharge l'emplacement.
- `--region` fixe la région (et la zone vaut par défaut `{region}-1`).

```bash
export SCW_ACCESS_KEY=... SCW_SECRET_KEY=... SCW_DEFAULT_ORGANIZATION_ID=...
pepin scan scaleway --live --region fr-par
```

## Ce qu'appelle un scan live

Chaque endpoint, y compris la liste parente d'une jointure. C'est exactement la surface qu'une
clé en lecture seule doit couvrir.

<!-- pepin:gen provider-scaleway-live -->
| Type normalisé | Appel | Note |
|---|---|---|
| `access_key` | `GET /iam/v1alpha1/api-keys?organization_id={org}` | — |
| `compute_instance` | `GET /instance/v1/zones/{zone}/servers` | — |
| `iam_user` | `GET /iam/v1alpha1/users?organization_id={org}` | — |
| `security_group_rule` | `GET /instance/v1/zones/{zone}/security_groups` | liste parente d'une jointure (appelée en premier) |
| `security_group_rule` | `GET /instance/v1/zones/{zone}/security_groups/{sg.id}/rules` | — |
| `object_storage_bucket` | `https://s3.{region}.scw.cloud` | API S3 du stockage objet (collecteur Go) |

URL de base : `https://api.scaleway.com`
<!-- /pepin:gen provider-scaleway-live -->

Le collecteur de stockage objet (`internal/objectstorage`) émet `ListBuckets`, puis par bucket
`GetBucketAcl`, `GetBucketVersioning`, `GetBucketPolicy`, `GetBucketTagging`,
`GetObjectLockConfiguration` et `GetBucketEncryption`. Chacun est best-effort : un appel refusé
laisse l'attribut non collecté, et le contrôle rend `not-evaluated`, jamais un `pass`
silencieux.

Noter ce qui n'est **pas** collecté en live : `network` et `kubernetes_cluster` n'existent que
dans le mapping Terraform. Un scan live d'un tenant Scaleway ne dit rien des VPC ni des clusters
Kapsule.

## Permissions minimales pour un scan live

Le vocabulaire de Scaleway est une **policy IAM** faite de règles, chacune portant des
**permission sets** et une portée. Créer une Application dédiée, lui donner une clé d'API, et y
attacher une policy à deux règles :

| Portée de la règle | Permission set | Ce qu'elle couvre |
|---|---|---|
| Organisation | `IAMReadOnly` | `/iam/v1alpha1/users`, `/iam/v1alpha1/api-keys` |
| Projet | `InstancesReadOnly` | groupes de sécurité, leurs règles, serveurs |
| Projet | `ObjectStorageReadOnly` | `ListBuckets`, `GetBucketAcl`, `GetBucketVersioning`, `GetBucketTagging`, `GetObjectLockConfiguration` |

Vérifié dans les sources de documentation de Scaleway elles-mêmes : le catalogue des permission
sets (`pages/iam/reference-content/permission-sets.mdx`) et la table d'équivalence des actions
S3 (`pages/object-storage/reference-content/s3-iam-permissions-equivalence.mdx`) de
`scaleway/docs-content`.

**Deux trous, énoncés plutôt que maquillés.** `GetBucketPolicy` et `GetBucketEncryption`
n'apparaissent dans **aucun** permission set en lecture seule de la table d'équivalence que nous
avons pu lire ; le seul ensemble qui nomme les bucket policies
(`ObjectStorageBucketPolicyFullAccess`) n'est pas en lecture seule. Nous n'avons pas vérifié
quelle permission en lecture les accorde, et nous ne devinons pas. En pratique, ces deux appels
sont best-effort : un `403` coûte de la couverture (`objectstorage_bucket_public_access` peut
perdre son signal de policy, `objectstorage_bucket_kms_encryption` rend `not-evaluated`), jamais
de l'exactitude.

`ObjectStorageBucketsRead` seul ne suffit **pas** : sa table d'équivalence n'inclut pas
`GetObjectLockConfiguration`.

Une clé restreinte à un projet ne voit que les buckets et les instances de ce projet.

## Ressources Terraform reconnues

<!-- pepin:gen provider-scaleway-terraform -->
| Ressource Terraform | Type normalisé | Bloc éclaté |
|---|---|---|
| `scaleway_iam_api_key` | `access_key` | — |
| `scaleway_iam_policy` | `iam_policy` | — |
| `scaleway_instance_security_group` | `security_group` | — |
| `scaleway_instance_security_group` | `security_group_rule` | `inbound_rule[*]` |
| `scaleway_instance_security_group` | `security_group_rule` | `outbound_rule[*]` |
| `scaleway_instance_security_group_rules` | `security_group_rule` | `inbound_rule[*]` |
| `scaleway_instance_security_group_rules` | `security_group_rule` | `outbound_rule[*]` |
| `scaleway_instance_server` | `compute_instance` | — |
| `scaleway_object_bucket` | `object_storage_bucket` | — |
| `scaleway_object_bucket_acl` | `object_storage_bucket` | — |
| `scaleway_rdb_acl` | `managed_database` | `acl_rules[*]` |
| `scaleway_rdb_instance` | `managed_database` | — |
| `scaleway_vpc_private_network` | `network` | — |
<!-- /pepin:gen provider-scaleway-terraform -->

```bash
terraform plan -out tfplan && terraform show -json tfplan > plan.json
pepin scan scaleway --terraform plan.json
```

## Couverture

<!-- pepin:gen provider-scaleway-coverage -->
| Source | ✅ `supported` | ◐ `partial` | ∅ `not-applicable` | ✗ `unsupported` |
|---|---:|---:|---:|---:|
| terraform | 18 | 6 | 2 | 31 |
| live | 16 | 3 | 2 | 36 |
<!-- /pepin:gen provider-scaleway-coverage -->

Contrôle par contrôle, avec le motif de chaque case qui n'est pas pleinement supportée, la
source de vérité est la [matrice de couverture](../coverage.fr.md). Elle est calculée depuis le
même descripteur que cette page.

### Déclarés non applicables

Un `not-applicable` est une affirmation : il porte donc sa justification, tirée du contrat du
fournisseur.

<!-- pepin:gen provider-scaleway-na -->
| Contrôle | Justification consignée au contrat |
|---|---|
| `blockstorage_snapshot_not_public` | Les snapshots block Scaleway (api/block/v1) n'exposent aucun mécanisme de partage ou d'export public : le risque d'exposition publique est structurellement absent, conforme par construction (STO-2). |
| `blockstorage_volume_encryption` | Chiffrement au repos des volumes block côté invité (LUKS/Cryptsetup), responsabilité du client (responsabilité partagée) ; l'API block n'expose aucun champ de chiffrement → non observable côté plateforme (CHF-2). |
<!-- /pepin:gen provider-scaleway-na -->

### Observables depuis une seule source

<!-- pepin:gen provider-scaleway-onesource -->
| Contrôle | Observable uniquement via | Motif du côté aveugle |
|---|---|---|
| `compute_instance_no_secrets_in_user_data` | terraform | attribut décisif « user_data » non projeté par cette source : garde de capacité, le scan rend « not-evaluated » |
| `compute_instance_public_ip_with_open_securitygroup` | live | attribut décisif « public_ip » non projeté par cette source : garde de capacité, le scan rend « not-evaluated » |
| `database_backup_enabled` | terraform | cette source ne produit aucune ressource de type « managed_database » |
| `database_encryption_at_rest_enabled` | terraform | cette source ne produit aucune ressource de type « managed_database » |
| `database_service_not_open_to_internet` | terraform | cette source ne produit aucune ressource de type « managed_database » |
| `iam_policy_no_privilege_escalation` | terraform | cette source ne produit aucune ressource de type « iam_policy » |
| `iam_user_mfa_enabled` | live | cette source ne produit aucune ressource de type « iam_user » |
| `network_securitygroup_default_deny` | terraform | cette source ne produit aucune ressource de type « security_group » |
| `objectstorage_bucket_kms_encryption` | live | attribut décisif « sse_kms_enabled » non projeté par cette source : garde de capacité, le scan rend « not-evaluated » |
| `objectstorage_bucket_versioning_enabled` | live | attribut décisif « versioning » non projeté par cette source : garde de capacité, le scan rend « not-evaluated » |
<!-- /pepin:gen provider-scaleway-onesource -->

C'est le tableau à lire avant de comparer un scan de plan et un scan live du même tenant :
certaines de ces différences sont de la couverture, pas de la dérive
([Plan Terraform contre scan live](../concepts/terraform-vs-live.fr.md)).

## Un scan réel

`examples/scaleway/terraform/` contient un module volontairement mal configuré. Son plan est
committé, donc ceci s'exécute sans aucun compte :

```bash
pepin scan scaleway --terraform examples/scaleway/terraform/plan.json
```

<!-- pepin:gen provider-scaleway-scan -->
```text
[…]
  │ CLD-CHF-2  │ Base de données managée sans chiffrement au rep… │ HIGH     │ scaleway │ 1 │
  │ CLD-CMP-9  │ Secret en clair dans les données utilisateur (u… │ HIGH     │ scaleway │ 1 │
  │ CLD-IAM-12 │ Politique IAM permettant une élévation de privi… │ HIGH     │ scaleway │ 1 │
  │ CLD-NET-1  │ Base de données managée joignable depuis Intern… │ HIGH     │ scaleway │ 2 │
  │ CLD-NET-2  │ Politique entrante par défaut d'un groupe de sé… │ HIGH     │ scaleway │ 1 │
  │ CLD-STO-3  │ Sauvegardes automatiques d'une base managée dés… │ HIGH     │ scaleway │ 1 │
  │ CLD-GVN-1  │ Inventaire et étiquetage incomplets              │ MEDIUM   │ scaleway │ 1 │
  │ CLD-STO-8  │ Object Lock (immutabilité) désactivé sur le sto… │ LOW      │ scaleway │ 1 │
  ╰────────────┴──────────────────────────────────────────────────┴──────────┴──────────┴───╯
──────────────────────────────────────────────────────────────────────────────
 Summary

 Verdict : NON CONFORME

 🔴 CRITICAL 1   🟠 HIGH 7   🟡 MEDIUM 1   🔵 LOW 1
──────────────────────────────────────────────────────────────────────────────
```
<!-- /pepin:gen provider-scaleway-scan -->

Le même module, corrigé, est dans `examples/scaleway/terraform-fixed/`, et le scanner rend le
code de sortie 0. La visite guidée est le
[démarrage rapide](../getting-started/quickstart.fr.md).

## Limites

- **Pas de collecte live des VPC ni de Kubernetes.** Les deux types ne sont mappés que depuis
  Terraform.
- **La policy et le chiffrement d'un bucket** peuvent être refusés à une clé strictement en
  lecture seule (voir plus haut) ; les contrôles concernés rendent alors `not-evaluated`.
- **Le chiffrement au repos des volumes block** est côté invité et non observable depuis
  l'API : c'est un `not-applicable` déclaré, pas un oubli.
- Les limites générales de l'outil, tous fournisseurs confondus, sont dans
  [Limites connues](../known-limitations.fr.md).

## Pour aller plus loin

- [Matrice de couverture](../coverage.fr.md) : chaque contrôle, chaque source.
- [Plan Terraform contre scan live](../concepts/terraform-vs-live.fr.md) : choisir la source.
- [Outscale](outscale.fr.md) · [Exoscale](exoscale.fr.md) : les deux autres clouds souverains.
- [GitHub Actions](../guides/github-actions.fr.md) · [GitLab CI](../guides/gitlab-ci.fr.md).

## Comment cette page reste vraie

Les tableaux d'identité, d'identifiants, d'endpoints, de mapping Terraform, de couverture et de
non-applicabilité sont calculés depuis `providers/scaleway.yaml` et depuis le référentiel par
`internal/docgen` ; la sortie de scan est une exécution réelle de la fixture committée. Ajouter
un type de ressource collecté change cette page, et `TestGeneratedDocsAreUpToDate` échoue tant
qu'elle n'a pas été régénérée. La section des permissions, elle, est écrite à la main, parce que
le descripteur ne porte pas les noms de permission sets de Scaleway : elle cite la source où
chaque nom a été vérifié, et dit clairement ce qui ne l'a pas été.

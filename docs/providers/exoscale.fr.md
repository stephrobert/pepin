> [🇬🇧 English](exoscale.md) · 🇫🇷 Français

# Exoscale

Tout ce que cette page affirme sur ce que Pépin collecte est dérivé de
`providers/exoscale.yaml`, le descripteur que le scanner lit lui-même. Il n'est pas nécessaire
d'ouvrir ce fichier pour comprendre la couverture.

<!-- pepin:gen provider-exoscale-identity -->
| Champ du descripteur | Valeur |
|---|---|
| Description | Exoscale (CH) — instances, security groups, block storage, SKS, SOS |
| Portée | cloud |
| Clé de région (`--region`) | `zone` |
| Authentification de l'API | `exoscale-hmac` |
| Juridiction du siège | CH |
| Établi dans l'UE | non |
| Contrôle capitalistique | extra_ue |
| SecNumCloud | `non` |
| Exposition extraterritoriale | non |
| Sources de l'ancrage | exoscale.com/about-us ; newsroom.a1.com (A1 Digital acquiert Exoscale) ; América Móvil 60,8 % / ÖBAG 28,4 % du groupe A1 Telekom Austria (09/2025) |
<!-- /pepin:gen provider-exoscale-identity -->

Lire ces champs attentivement : Exoscale est **suisse, non établi dans l'UE**, et son contrôle
capitalistique est hors Union européenne. Ce n'est pas une disqualification, la Suisse bénéficie
d'une décision d'adéquation et Exoscale exploite des zones européennes, mais c'est un fait que
le contrôle de gouvernance `CLD-GVN-4` rapporte, et un fait que quiconque construit un argument
de souveraineté doit traiter explicitement. Le descripteur en consigne les sources.

## Authentification

<!-- pepin:gen provider-exoscale-credentials -->
| Clé logique | Variable d'environnement | Défaut |
|---|---|---|
| `access_key` | `EXOSCALE_API_KEY` | — |
| `secret_key` | `EXOSCALE_API_SECRET` | — |
| `zone` | `EXOSCALE_ZONE` | `ch-gva-2` |
| fichier de configuration natif | `~/.config/exoscale/exoscale.toml` | `exoscale` |
<!-- /pepin:gen provider-exoscale-credentials -->

- L'API s'authentifie avec le schéma **HMAC** propre à Exoscale (`exoscale-hmac`).
- Les mêmes clé et secret d'API servent à SOS (stockage objet).
- Le fichier de configuration natif est lu quand les variables sont absentes ;
  `EXOSCALE_CONFIG` en surcharge l'emplacement.
- **La clé de région est `zone`.** `--region` fixe donc la zone, et le host de l'API la porte
  lui-même (`api-{zone}.exoscale.com`) : **un scan couvre une zone**. Scanner plusieurs zones
  demande plusieurs exécutions.

```bash
export EXOSCALE_API_KEY=... EXOSCALE_API_SECRET=...
pepin scan exoscale --live --region ch-gva-2
```

## Ce qu'appelle un scan live

Chaque endpoint, y compris la liste parente d'une jointure.

<!-- pepin:gen provider-exoscale-live -->
| Type normalisé | Appel | Note |
|---|---|---|
| `blockstorage_snapshot` | `GET /block-storage-snapshot` | — |
| `blockstorage_volume` | `GET /block-storage` | — |
| `compute_instance` | `GET /instance` | liste parente d'une jointure (appelée en premier) |
| `compute_instance` | `GET /instance/{vm.id}` | — |
| `iam_role` | `GET /iam-role` | — |
| `iam_user` | `GET /user` | — |
| `kubernetes_cluster` | `GET /sks-cluster` | — |
| `network` | `GET /private-network` | — |
| `security_group_rule` | `GET /security-group` | — |
| `object_storage_bucket` | `https://sos-{zone}.exo.io` | API S3 du stockage objet (collecteur Go) |

URL de base : `https://api-{zone}.exoscale.com/v2`
<!-- /pepin:gen provider-exoscale-live -->

Le collecteur de stockage objet (`internal/objectstorage`) émet `ListBuckets`, puis par bucket
`GetBucketAcl`, `GetBucketVersioning`, `GetBucketPolicy`, `GetBucketTagging`,
`GetObjectLockConfiguration` et `GetBucketEncryption`.

Exoscale est le seul des trois pour lequel un scan live atteint le Kubernetes managé (SKS) par
la spec de collecte du descripteur elle-même.

## Permissions minimales pour un scan live

Dérivé du descripteur du fournisseur, si bien que ce tableau et ce que rapporte le scan ne
peuvent pas diverger : quand un appel est refusé, le relevé de capacités et le motif du
`not-evaluated` nomment le droit listé ici. **Confirmé** signifie que le droit est énoncé dans
la source officielle citée en regard, pas qu'un scan a été lancé avec un rôle délibérément
réduit. Ce dépôt ne détient aucun identifiant cloud et aucun contrôle automatisé n'atteint une
API de fournisseur.

<!-- pepin:gen provider-exoscale-permissions -->
| Unité de collecte | Droit minimal | Confirmé | Source |
|---|---|:-:|---|
| `security_group_rule` | `compute:list-security-groups` | oui | Exoscale IAM: compute / iam / sos operation catalogues + CEL policy grammar |
| `network` | `compute:list-private-networks` | oui | Exoscale IAM: compute / iam / sos operation catalogues + CEL policy grammar |
| `compute_instance` | `compute:list-instances, compute:get-instance` | oui | Exoscale IAM: compute / iam / sos operation catalogues + CEL policy grammar |
| `iam_user` | `iam:list-users` | oui | Exoscale IAM: compute / iam / sos operation catalogues + CEL policy grammar |
| `kubernetes_cluster` | `compute:list-sks-clusters` | oui | Exoscale IAM: compute / iam / sos operation catalogues + CEL policy grammar |
| `iam_role` | `iam:list-iam-roles` | oui | Exoscale IAM: compute / iam / sos operation catalogues + CEL policy grammar |
| `blockstorage_volume` | `compute:list-block-storage-volumes` | oui | Exoscale IAM: compute / iam / sos operation catalogues + CEL policy grammar |
| `blockstorage_snapshot` | `compute:list-block-storage-snapshots` | oui | Exoscale IAM: compute / iam / sos operation catalogues + CEL policy grammar |
| `object_storage_bucket` | `sos:list-buckets, sos:get-bucket-acl, sos:get-bucket-versioning, sos:get-bucket-policy, sos:get-bucket-tagging, sos:get-object-lock-configuration, sos:get-bucket-encryption` | oui | Exoscale IAM: compute / iam / sos operation catalogues + CEL policy grammar |
<!-- /pepin:gen provider-exoscale-permissions -->

Le vocabulaire d'Exoscale est un **rôle IAM** portant une policy, et une **clé d'API liée à ce
rôle** (`exo iam role create`, puis `exo iam api-key create <clé> <rôle>` ; une clé ne peut pas
être réaffectée à un autre rôle ensuite).

La policy refuse par défaut et autorise exactement les opérations que Pépin appelle :

```json
{
  "default-service-strategy": "deny",
  "services": {
    "compute": {
      "type": "rules",
      "rules": [{
        "action": "allow",
        "expression": "operation in ['list-security-groups','list-private-networks','list-instances','get-instance','list-sks-clusters','list-block-storage-volumes','list-block-storage-snapshots']"
      }]
    },
    "iam": {
      "type": "rules",
      "rules": [{
        "action": "allow",
        "expression": "operation in ['list-users','list-iam-roles']"
      }]
    },
    "sos": {
      "type": "rules",
      "rules": [{
        "action": "allow",
        "expression": "operation in ['list-buckets','get-bucket-acl','get-bucket-versioning','get-bucket-policy','get-bucket-tagging','get-object-lock-configuration','get-bucket-encryption']"
      }]
    }
  }
}
```

Chaque nom d'opération ci-dessus a été vérifié dans les références IAM officielles d'Exoscale
(les catalogues d'opérations `compute`, `iam` et `sos`), et la grammaire CEL des policies dans
le guide officiel. Exoscale est le seul des trois fournisseurs à documenter une opération IAM
pour lire la policy d'un bucket et sa configuration de chiffrement, les deux appels dont nous
n'avons pas pu établir la permission en lecture seule chez Scaleway.

**Un piège, documenté par Exoscale elle-même** : si **aucune règle ne correspond**, la requête
est refusée *quelle que soit la stratégie de service par défaut*. Chaque bloc de service doit
donc contenir une règle qui correspond effectivement, ce que fait la forme
`operation in [...]` ci-dessus. Une variante plus lâche mais tout aussi valide est
`operation.startsWith('get-') || operation.startsWith('list-')`.

Deux durcissements facultatifs, lus dans la même grammaire de policy : `source_ip.inIpRange(…)`
pour lier la clé à l'adresse du scanner, et un `max-session-ttl` court sur le rôle. Pépin lit
ces deux notions comme des attributs d'`iam_role`, ce qui lui permet précisément de rapporter
sur vos *autres* rôles.

## Ressources Terraform reconnues

<!-- pepin:gen provider-exoscale-terraform -->
| Ressource Terraform | Type normalisé | Bloc éclaté |
|---|---|---|
| `exoscale_block_storage_volume` | `blockstorage_volume` | — |
| `exoscale_compute_instance` | `compute_instance` | — |
| `exoscale_iam_role` | `iam_role` | — |
| `exoscale_private_network` | `network` | — |
| `exoscale_security_group_rule` | `security_group_rule` | — |
| `exoscale_sks_cluster` | `kubernetes_cluster` | — |
<!-- /pepin:gen provider-exoscale-terraform -->

```bash
terraform plan -out tfplan && terraform show -json tfplan > plan.json
pepin scan exoscale --terraform plan.json
```

## Couverture

<!-- pepin:gen provider-exoscale-coverage -->
| Source | ✅ `supported` | ◐ `partial` | ∅ `not-applicable` | ✗ `unsupported` |
|---|---:|---:|---:|---:|
| terraform | 21 | 1 | 5 | 30 |
| live | 25 | 1 | 5 | 26 |
<!-- /pepin:gen provider-exoscale-coverage -->

Contrôle par contrôle, avec le motif de chaque case qui n'est pas pleinement supportée, la
source de vérité est la [matrice de couverture](../coverage.fr.md).

### Déclarés non applicables

<!-- pepin:gen provider-exoscale-na -->
| Contrôle | Justification consignée au contrat |
|---|---|
| `blockstorage_snapshot_not_public` | Snapshots block-storage Exoscale non exportables/partageables (doc officielle) : le risque d'exposition publique est structurellement absent, conforme par construction (STO-2). |
| `loadbalancer_http_redirect_to_https` | type de ressource « load_balancer » absent de l'API exoscale |
| `loadbalancer_logging_enabled` | type de ressource « load_balancer » absent de l'API exoscale |
| `loadbalancer_ssl_listeners` | type de ressource « load_balancer » absent de l'API exoscale |
| `objectstorage_bucket_kms_encryption` | SOS chiffre au repos par défaut (SSE-SOS, clés gérées par Exoscale, type SSE-S3) mais n'expose pas de BYOK/KMS géré par le client au niveau bucket (SSE-C reste par-objet, non observable) → le contrôle BYOK-au-bucket est sans objet (CHF-4). |
<!-- /pepin:gen provider-exoscale-na -->

### Observables depuis une seule source

<!-- pepin:gen provider-exoscale-onesource -->
| Contrôle | Observable uniquement via | Motif du côté aveugle |
|---|---|---|
| `iam_user_mfa_enabled` | live | cette source ne produit aucune ressource de type « iam_user » |
| `objectstorage_bucket_object_lock_enabled` | live | cette source ne produit aucune ressource de type « object_storage_bucket » |
| `objectstorage_bucket_public_access` | live | cette source ne produit aucune ressource de type « object_storage_bucket » |
| `objectstorage_bucket_versioning_enabled` | live | cette source ne produit aucune ressource de type « object_storage_bucket » |
<!-- /pepin:gen provider-exoscale-onesource -->

## Un scan réel

`examples/exoscale/terraform/` contient un module volontairement mal configuré. Son plan est
committé, donc ceci s'exécute sans aucun compte :

```bash
pepin scan exoscale --terraform examples/exoscale/terraform/plan.json
```

<!-- pepin:gen provider-exoscale-scan -->
```text
[…]
  │ CLD-IAM-1  │ Rôle IAM aux privilèges excessifs                │ HIGH     │ exoscale │ 1 │
  │ CLD-IAM-12 │ Politique IAM permettant une élévation de privi… │ HIGH     │ exoscale │ 1 │
  │ CLD-IAM-4  │ Rôle IAM sans restriction d'IP source            │ HIGH     │ exoscale │ 2 │
  │ CLD-K8S-2  │ Plan de contrôle Kubernetes non hautement dispo… │ HIGH     │ exoscale │ 1 │
  │ CLD-NET-1  │ SSH (port 22) ouvert à Internet                  │ HIGH     │ exoscale │ 2 │
  │ CLD-GVN-1  │ Inventaire et étiquetage incomplets              │ MEDIUM   │ exoscale │ 1 │
  │ CLD-K8S-3  │ Mises à jour automatiques du cluster Kubernetes… │ MEDIUM   │ exoscale │ 1 │
  │ CLD-GVN-3  │ Ressource hébergée hors Union européenne         │ LOW      │ exoscale │ 3 │
  ╰────────────┴──────────────────────────────────────────────────┴──────────┴──────────┴───╯
──────────────────────────────────────────────────────────────────────────────
 Summary

 Verdict : NON CONFORME

 🔴 CRITICAL 2   🟠 HIGH 11   🟡 MEDIUM 2   🔵 LOW 3
──────────────────────────────────────────────────────────────────────────────
```
<!-- /pepin:gen provider-exoscale-scan -->

## Limites

- **Une zone par scan.** Le host de l'API porte la zone ; un tenant multi-zones demande une
  exécution par zone, et chacune ne connaît que la sienne.
- **Pas de type load balancer.** Les contrôles qui lisent `load_balancer` sont déclarés non
  applicables, avec cette justification au contrat.
- **SOS chiffre au repos par défaut** avec des clés gérées par le fournisseur, et n'expose
  aucune clé client au niveau du bucket : le contrôle BYOK est donc non applicable plutôt qu'en
  échec.
- Les limites générales de l'outil sont dans [Limites connues](../known-limitations.fr.md).

## Pour aller plus loin

- [Matrice de couverture](../coverage.fr.md) : chaque contrôle, chaque source.
- [Plan Terraform contre scan live](../concepts/terraform-vs-live.fr.md) : choisir la source.
- [Scaleway](scaleway.fr.md) · [Outscale](outscale.fr.md) : les deux autres clouds souverains.
- [GitHub Actions](../guides/github-actions.fr.md) · [GitLab CI](../guides/gitlab-ci.fr.md).

## Comment cette page reste vraie

Les tableaux d'identité, d'identifiants, d'endpoints, de mapping Terraform, de couverture et de
non-applicabilité sont calculés depuis `providers/exoscale.yaml` et depuis le référentiel par
`internal/docgen` ; la sortie de scan est une exécution réelle de la fixture committée. La
section des permissions est écrite à la main, parce que le descripteur ne porte pas le
vocabulaire IAM d'Exoscale : chaque nom d'opération qu'elle contient a été lu dans une référence
officielle, et ce qui n'aurait pas pu être vérifié serait signalé comme tel.

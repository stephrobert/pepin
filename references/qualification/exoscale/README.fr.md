> [🇬🇧 English](README.md) · 🇫🇷 Français

# Tenant de qualification Exoscale

Une organisation Exoscale délibérément fautive : **une faute par ressource, un
contre-exemple par contrôle**, appliquée sur un compte réel, scannée en `--live` dans
les cinq formats, scellée, vérifiée, rescannée en `--terraform` sur le même plan,
détruite, prouvée détruite, et comparée à [`expected.yaml`](expected.yaml). Toute
différence est NO-GO. La doctrine et le runner sont décrits dans
[`../README.fr.md`](../README.fr.md) ; ce qui suit est ce qui est propre à Exoscale.

```bash
PROVIDER=exoscale mise run qualify:plan                       # plan + scan --terraform, rien n'est créé
PEPIN_GATE_LIVE=1 PROVIDER=exoscale mise run qualify           # apply, scan, scelle, détruit, prouve, compare
```

Le compte se confirme avant tout apply : l'API rattache la clé à une organisation
(`GET /api-key/{key}` → `org-id` — `GET /organization` est refusé à une clé dont le
rôle ne couvre pas l'organisation, et une clé de qualification n'a aucune raison de le
couvrir), et cet identifiant doit être égal à **`PEPIN_QUAL_EXO_ORG`**. La variable
absente est un refus, rien n'est appliqué
([ADR-0012](../../docs/adr/0012-aucun-identifiant-en-ci.md) : aucun identifiant de
compte n'est committé). Les identifiants natifs sont ceux de `~/.config/exoscale/exoscale.toml`
ou de `EXOSCALE_API_KEY`/`EXOSCALE_API_SECRET` ; le provider Terraform ne lisant pas le
fichier, le crochet `variables` les lui passe par l'environnement du run, sans les
écrire.

## Ce que la stack crée

| Famille | Nombre | Ressource fautive → contrôle | Contre-exemple (doit rester muet) |
|---|---:|---|---|
| Groupes de sécurité | 9 (+ 15 règles) | `ssh_open` → `…tcp_port_22` · `rdp_open` → `…tcp_port_3389` · `db_open` (5432) → `…high_risk_tcp_ports` · `snmp_open` (UDP 161) → `…high_risk_udp_ports` · `any_open` (tout TCP et tout UDP) → les quatre précédents · `quartet` (quatre `/2`) → `…tcp_port_22` par évasion · `egress_any` (sortie TCP 1-65535 vers Internet) → `network_securitygroup_unrestricted_egress` · `undocumented_flow` (règle sans description, port 443) → `network_flow_matrix_documented` | `hardened` (SSH depuis `10.0.0.0/8`, HTTPS depuis Internet, sortie TCP 443, chaque flux décrit) |
| Réseaux privés | 2 | `undocumented` → `network_documented` | `documented` (Owner, Project, Env, description) |
| Instances | 4 appliquées + 1 sur le plan | `exposed` (IP publique, SSH ouvert) → `compute_instance_public_ip_with_open_securitygroup` · `untagged` → `governance_resource_required_tags` · `secrets` (cloud-init avec mot de passe) → `compute_instance_no_secrets_in_user_data` · `swiss` (zone `ch-gva-2`, **plan seulement**) → `governance_resource_region_in_eu` (low : hors UE, espace européen de confiance) | `hardened` (IP publique derrière un groupe fermé, étiquetée, `de-fra-1`, cloud-init anodin) ; `untagged` est aussi le contre-exemple de NET-3 : même SSH ouvert qu'`exposed`, mais `private = true`, donc sans IP publique |
| Volumes block storage | 2 (+ 1 snapshot) | `unsnapshotted` (attaché à `secrets`, jamais sauvegardé) → `blockstorage_volume_snapshots_exist` | `snapshotted` (attaché à `hardened`, une snapshot terminée du jour) ; les deux chiffrés par construction (`blockstorage_volume_encryption` passe) |
| Clusters SKS | 2, sans nodepool | `weak` (`starter`, sans auto-upgrade, audit désactivé) → `kubernetes_cluster_control_plane_highly_available`, `…auto_upgrade_enabled`, `…audit_logging_enabled` | `strong` (`pro`, auto-upgrade, endpoint d'audit) |
| Rôles IAM | 4 | `admin` (`allow` par défaut) → `iam_role_no_admin_privileges` · `no_source_ip` → `iam_role_source_ip_restricted` · `unbounded` (sans `duration(`) → `iam_role_key_lifetime_bounded` | `restricted` (`deny` par défaut, `source_ip`, `duration(`) |
| Buckets SOS (crochet `extra`, hors Terraform) | 3 | `public` (ACL `public-read`) → `objectstorage_bucket_public_access` · `unversioned` → `objectstorage_bucket_versioning_enabled` · `public` et `unversioned` (sans verrou) → `objectstorage_bucket_object_lock_enabled` | `hardened` (privé, créé avec Object Lock — par l'API S3, `exo storage` ne le sait pas —, donc versionné) |
| Fournisseur | — | `governance_provider_sovereignty` échoue sur `exoscale` lui-même : siège en Suisse, contrôle capitalistique extra-UE — des faits du descripteur, pas du tenant | — |

Ce qui est propre à Exoscale, et qui a dessiné cette stack :

- **Quatre instances.** C'est le quota de l'organisation (`GET /quota` : `instance` 4),
  découvert au premier apply (`Usage of resource 'instance' has been exceeded`, deux
  instances refusées, tout le reste détruit et prouvé détruit). D'où un contre-exemple
  porté par une ressource déjà fautive ailleurs (`untagged`, privée), un volume fautif
  porté par `secrets`, et `swiss` sur le plan seulement (`terraform_only_resources`,
  que le runner passe à `false` pour appliquer).
- **Une zone scannée.** Les ressources sont zonales et `scan --live --region de-fra-1`
  ne lit que cette zone : une instance de `ch-gva-2` lui serait invisible, et
  `governance_resource_region_in_eu` n'est prouvé en écart que sur le plan. L'inventaire
  de la preuve de destruction, lui, parcourt les deux zones (`zones:` de `tenant.yaml`).
- **Pas de protocole « tout ».** Une règle de groupe n'accepte que `tcp`, `udp`,
  `icmp`, `icmpv6`, `ah`, `esp`, `gre`, `ipip` (provider 0.71.0 et API v2).
  `network_securitygroup_allow_ingress_from_internet_to_all_ports` est donc **non
  applicable** chez Exoscale (#202, #206) — `any_open` ouvre tout TCP et tout UDP, et
  ce sont les quatre contrôles par famille de ports qui parlent. La sortie « tout »
  s'écrit `tcp 1-65535 → 0.0.0.0/0`, et `network_securitygroup_unrestricted_egress`
  reconnaît cette forme (#206) : `egress_any` est un écart.
- **Les buckets SOS ne s'étiquettent pas.** SOS accepte `PutBucketTagging` (200, par
  SigV4 comme par la CLI `aws`) et ne persiste rien : `GetBucketTagging` rend
  `NoSuchTagSet` juste après. Un bucket SOS n'a donc jamais d'étiquette, quoi qu'on
  fasse, et `governance_resource_required_tags` échoue sur les trois buckets **par
  construction** — épinglé tel que mesuré dans `expected.yaml`, faux positif non
  remédiable, issue #208.
- **Un cluster SKS crée un rôle IAM `sks-ccm-<cluster>`** (éditable, `deny` par
  défaut, sans borne d'origine ni de durée) et le supprime avec lui : chaque cluster
  apporte deux écarts `iam_role_*` par construction, hors tenant ici (sujets non
  préfixés), rapportés et non gardés — issue #213. Le rôle de la clé qui
  qualifie (`deny` par défaut, sans règle CEL) porte les mêmes deux écarts.
- **Une instance privée n'a pas de groupe de sécurité.** `private = true`
  (`public-ip-assignment: none`) : l'API ignore les groupes déclarés et rend
  `security-groups: []`. `compute_instance_has_security_group` en fait un écart, avec
  une remédiation inexécutable — épinglé tel que mesuré sur `untagged`, issue
  #212.
- **Faux vert live sur l'audit SKS.** L'API rend `audit: {}` pour un cluster dont
  l'audit est désactivé ; la collecte dérive `audit_enabled` de `audit.endpoint`,
  absent, ne projette rien sur `weak`, et `kubernetes_cluster_audit_logging_enabled`
  conclut `pass` sur le seul cluster observé (`observed=1/2`) — le plan, lui, échoue
  sur `weak`. Épinglé comme **défaut connu** (#209) ; la correction fera
  rougir cette porte sciemment.
- **Faux vert live sur l'étiquetage des instances.** La collecte live ne projette pas
  `labels` sur les instances, volumes et clusters (le contrat les dit `a_verifier`) ;
  seuls les buckets portent `tags`, la règle saute le reste en silence, et `untagged`
  passe — issue #211. Le plan, lui, l'attrape.
- **`region` n'est projetée sur rien en live** : `governance_resource_region_in_eu`
  rend `not-evaluated` sur toute organisation Exoscale (dette de collecte nommée,
  ADR-0010, issue #210).

## Ce que le provider amont est connu pour laisser derrière lui au `destroy`

Lu avant de choisir les types de ressources, dans `exoscale/terraform-provider-exoscale`
(0.71.0 épinglé, lockfile committé). Ce que chacune a changé dans cette stack :

| Amont | État | Ce que ça fait | Ce que ce tenant en fait |
|---|---|---|---|
| [#453](https://github.com/exoscale/terraform-provider-exoscale/issues/453) le provider ne détache pas avant de supprimer (IP élastiques, groupes de sécurité) ; [#537](https://github.com/exoscale/terraform-provider-exoscale/issues/537) `Cannot delete group when it's in use by virtual machines` | #453 fermée 2025-08 (IP corrigée en 0.69.2, [#460](https://github.com/exoscale/terraform-provider-exoscale/pull/460)) ; #537 ouverte | Un groupe référencé par une instance ne se détruit ni ne se renomme. | Aucune IP élastique ; chaque groupe n'est attaché qu'à des instances Terraform, détruites d'abord par dépendance. Le nettoyage de secours supprime les instances, attend, puis les groupes. |
| [#375](https://github.com/exoscale/terraform-provider-exoscale/issues/375) une instance trop petite pour un volume block storage disparaît de l'état (`Instance size must be at least small`) | ouverte | Une instance `micro` avec `block_storage_volume_ids` est créée, puis perdue par Terraform : un reste. | Les deux instances qui portent un volume sont `standard.small` (`instance_type_with_volume`). |
| [#207](https://github.com/exoscale/terraform-provider-exoscale/issues/207) le destroy n'attend pas les ressources dépendantes ; [#210](https://github.com/exoscale/terraform-provider-exoscale/issues/210) un NLB créé par le CCM d'un cluster SKS survit au destroy ; [#240](https://github.com/exoscale/terraform-provider-exoscale/issues/240) destroy d'un cluster en `context deadline exceeded` | fermées (2022-2023) | Ce qu'un cluster crée hors Terraform lui survit. | Clusters **sans nodepool** et `create_default_security_group = false` : rien n'est créé hors Terraform, et l'inventaire liste NLB, pools et groupes pour le prouver. |
| [#438](https://github.com/exoscale/terraform-provider-exoscale/issues/438) `aws_s3_bucket` ne termine jamais sur SOS | fermée 2025-06 | Un bucket par le provider `aws` n'est ni fiable ni détruit proprement. | Les buckets passent par `exo storage` (CLI de référence) et l'API S3, hors Terraform, dans le crochet `extra` ; supprimés récursivement et relistés. |
| [#76](https://github.com/exoscale/terraform-provider-exoscale/issues/76) destroy en échec après suppression manuelle d'un réseau privé | fermée 2020 | — | Rien n'est supprimé à la main pendant un run ; le nettoyage de secours ne s'exécute qu'après le destroy. |
| Mesuré ici : **`GET /private-network` sur `api-de-fra-1` reste parfois sans réponse** (une fois sur trois ou quatre), puis répond en 0,2 s | à signaler en amont | Un listing de preuve qui rendrait « inconnu » sur un aléa. | Le crochet réessaie trois fois (60 s de délai) avant de conclure. |
| Mesuré ici : **SOS active le versioning d'un bucket créé avec Object Lock** | — | Le contre-exemple des buckets est versionné par construction. | `hardened` sert de contre-exemple aux trois contrôles de bucket. |
| Mesuré ici : **le quota `instance` compte encore, plusieurs minutes après un destroy réussi, des instances qu'aucun listing ne montre** (`GET /quota` : usage 2, 0 instance listée, 5 min après) | à signaler à Exoscale | L'apply suivant est refusé (`Usage of resource 'instance' has been exceeded`) alors que le compte est vide : un run 4 l'a subi. | Le crochet `variables` attend (borné à 15 min) que le compteur retombe à zéro avant que le runner n'applique. |
| Mesuré ici (une fois, sur un cluster hors tenant) : **la clé d'API `sks-ccm-<cluster>` que SKS crée pour le CCM a survécu une vingtaine de minutes à son cluster** ; son rôle, lui, était supprimé, et `DELETE /api-key` répondait 403 `API Key not in organization`, par l'API comme par `exo iam api-key delete` ; elle a fini par disparaître d'elle-même | — (suppression asynchrone, côté plateforme) | Pendant ce délai, une clé inerte (son rôle n'existe plus) est listée, et l'organisation ne peut pas la retirer. Les clusters du tenant n'en ont laissé aucune dans la fenêtre de la preuve (delta `api_keys` vide à chaque run). | L'inventaire liste les clés d'API : une survivante apparaît dans le delta, et le nettoyage de secours tente la suppression et dit le 403 sans bloquer. |

La preuve de destruction liste 15 familles par l'API v2 (instances, réseaux privés,
volumes, snapshots, clusters SKS, IP élastiques, pools d'instances, NLB, modèles
privés, clés SSH, groupes d'anti-affinité, groupes de sécurité, rôles IAM, clés d'API)
et les buckets par `exo storage list`, dans les deux zones, filtrées sur l'étiquette et
le préfixe du tenant, et compare le listing entier à celui pris avant apply. `exo`, la
CLI de référence du fournisseur, est ce qu'un mainteneur rejoue à la main :

```bash
exo compute instance list -O table
exo compute security-group list -O table
exo compute private-network list -O table
exo compute block-storage list -O table
exo compute block-storage snapshot list -O table
exo compute sks list -O table
exo compute elastic-ip list -O table
exo iam role list -O table
exo iam api-key list -O table
exo storage list -O table
```

Chaque commande liste toutes les zones par défaut (`-z` pour une seule).

## Ce que ce tenant ne peut pas exercer, et pourquoi

`tenant.yaml`, `not_exercised:` — `…_all_ports` (non applicable : aucun protocole
« tout »), `governance_resource_region_in_eu` en live (`region` non collectée ; et une
seule zone lue, quatre instances au quota), le MFA des utilisateurs (des personnes), les
écarts du rôle de la clé qui qualifie et des rôles `sks-ccm-*`, l'étiquetage des
buckets SOS (impossible par construction), la souveraineté du fournisseur (un fait du
descripteur).

## Coût et budget

Tarifs horaires dans [`tenant.yaml`](tenant.yaml), lus sur l'API de tarification du
portail (EUR) : 2 × `standard.micro` à 0,0073 EUR/h, 2 × `standard.small` à 0,0233 EUR/h,
40 Go de disque local et 30 Go de block storage à 0,00014 EUR/Go/h, un plan de contrôle
SKS Pro à 0,055 EUR/h (Starter est gratuit) — environ **0,13 EUR par heure** de vie du
tenant. Budget : 20 minutes mur, sinon NO-GO.

## Mesuré

<!-- qualification:measured -->
Run du 2026-09-09 (`GO`), sur l'organisation que l'environnement nomme :

| | |
|---|---|
| Ressources | 40 dans le plan complet, 39 appliquées (`swiss` n'est que sur le plan), plus 3 ressource(s) hors Terraform : pepin-qual-<org>-public, pepin-qual-<org>-unversioned, pepin-qual-<org>-hardened |
| `apply` | 40.6 s |
| `scan --live` × 5 formats + scellement | 11.3 s, code de sortie 1 dans chaque format |
| bundle | `verify --re-derive` passe ; le bundle altéré d'un octet est refusé |
| `scan --terraform` | code de sortie 1 |
| `destroy` | 50.7 s — terraform destroy rc=0 ; état vide ; 2 fichier(s) d'état purgé(s) |
| preuve de destruction | 15 familles listées, aucune ressource du tenant, aucun delta avant/après |
| comparaison | live : 46 résultats, 0 différence ; terraform : 37 résultats, 0 différence |
| falsification | 3 attentes cassées en mémoire sur chaque source, toutes NO-GO |
| durée mur | 183.8 s (budget 20 min) |
| coût estimé | 0.0037 EUR (0.029 h × 0.126 EUR/h) |

Écarts hors tenant rapportés par le scan live et non gardés : 7
(l'utilisateur de l'organisation sans MFA, le rôle de la clé qui qualifie et les deux rôles
`sks-ccm-*` des clusters, sans borne d'origine ni de durée — #213).
<!-- /qualification:measured -->

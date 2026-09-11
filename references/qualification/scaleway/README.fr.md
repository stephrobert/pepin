> [🇬🇧 English](README.md) · 🇫🇷 Français

# Tenant de qualification Scaleway

40 ressources dans le plan, **dont 29 appliquées**, dans `fr-par` / `fr-par-1`, sur
le projet épinglé dans [`expected.yaml`](expected.yaml). Une faute par ressource, un
contre-exemple par contrôle. Appliqué, scanné, scellé, détruit et comparé par
`PEPIN_GATE_LIVE=1 mise run qualify` (voir [le README générique](../README.fr.md)).

**Deux passes qui ne mesurent pas la même chose.** Un scan live Scaleway collecte
cinq types (`access_key`, `iam_user`, `compute_instance`, `security_group_rule`,
`object_storage_bucket`, contrat de `providers/scaleway.yaml`) ; un plan en porte
huit (plus `managed_database`, `security_group`, `iam_policy`, `network`).
Provisionner une base managée sur le compte réel coûterait de l'argent qu'aucun scan
live ne mesure : les ressources que seul un plan lit sont derrière
`terraform_only_resources` (défaut `true`) ; le runner applique avec `false` et
scanne le plan complet en `--terraform`. `expected.yaml` épingle les deux sources
séparément.

## Prérequis : un projet dédié, jamais celui par défaut

Cette stack applique des ressources **délibérément exposées** — SSH ouvert à tout
Internet, buckets publics, politiques permissives — et sa preuve de destruction repose
sur un delta avant/après pour ce qui ne porte pas d'étiquette. Une ressource **tierce**
qui bouge pendant le run fausse ce delta, dans un sens ou dans l'autre.

Chez Scaleway, les ressources vivent dans un **Projet** et n'en sortent pas, tandis que
l'IAM et les quotas restent au niveau de l'**Organization**. Le projet est donc la seule
frontière que ce tenant puisse utiliser — et le **projet par défaut n'en est pas une** :
il porte l'ID de l'Organization, ne peut être ni supprimé ni transféré, et c'est là que
tout atterrit quand personne n'a choisi.

Mesuré le 2026-09-10 : le projet par défaut portait une VM `pavois-repro-…` lancée deux
heures plus tôt, issue d'un autre travail. Le run n'a pu avoir lieu qu'après vidage
complet du compte, ce qui n'est pas une procédure. D'où l'issue #240, et le refus que le
crochet `identity` oppose désormais.

```bash
scw account project create name=pepin-qualification     description="Tenant de qualification Pepin — vide entre deux runs"
scw config set default-project-id=<ID_DU_PROJET>
export PEPIN_QUAL_SCW_PROJECT=<ID_DU_PROJET>
```

Le projet n'est **pas recréé** à chaque run : il est **vidé**. Un projet réutilisé garde
son historique de consommation, ne consomme qu'une fois le plafond de 25 projets par
organisation, et n'a besoin qu'une fois de son VPC (un projet créé depuis le
13 mai 2025 n'en reçoit plus par défaut).

## Ce que la stack crée

| Famille | Nombre | Ressource fautive → contrôle | Contre-exemple (doit rester muet) |
|---|---:|---|---|
| Applications IAM | 1 (+1 plan seul) | — (`keys` porte les clés et aucune politique ; `admin`, plan seul, porte les politiques et aucune clé : le chemin d'élévation n'est jamais un secret en circulation) | — |
| Clés d'API IAM | 2 (+1 plan seul) | `expired` (échéance quelques minutes après l'apply ; le runner l'attend avant de scanner, et une clé expirée **reste listée** par l'API) → `iam_accesskey_expiration_set` (high) · `no_expiry`, **plan seul** : l'organisation refuse de la créer — `organization security settings require an expiration date for API keys`, mesuré le 2026-09-09 — → le même contrôle (critical) | `expiring` (échéance à J+2, posée par le runner) |
| Politiques IAM | 2, plan seul | `iam_manager` (PermissionSet `IAMManager`, portée organisation) → `iam_policy_no_privilege_escalation` | `read_only` (`InstancesReadOnly`, portée projet) |
| Groupes de sécurité | 9 | `ssh_open` → `…tcp_port_22` · `rdp_open` → `…tcp_port_3389` · `db_open` (5432) → `…high_risk_tcp_ports` · `snmp_open` (UDP 161) → `…high_risk_udp_ports` · `any_open` → `…all_ports` **et les quatre précédents** · `quartet` (quatre `/2`) → `…tcp_port_22` par évasion · `egress_any` → `network_securitygroup_unrestricted_egress` · `default_accept` → `network_securitygroup_default_deny` | `hardened` : refus par défaut, SSH depuis `10.0.0.0/8`, HTTPS depuis Internet, sortie TCP 443 seulement |
| IP flexibles | 2 | — | — |
| Serveurs (DEV1-S) | 5 | `exposed` (IP publique + `ssh_open`) → `compute_instance_public_ip_with_open_securitygroup` · `untagged` → `governance_resource_required_tags` · `secrets` (cloud-init avec mot de passe) → `compute_instance_no_secrets_in_user_data` | `private` (même groupe ouvert, pas d'IP publique) · `hardened` (IP publique, groupe restrictif) |
| VPC | 1, plan seul | — | — |
| Réseaux privés | 2, plan seul | `undocumented` → `network_documented` | `documented` (Owner, Project, Env) |
| Buckets (vides) | 7 | `public` (ACL `public-read`) et `policy` (politique `Principal: *`) → `objectstorage_bucket_public_access` · `unversioned` → `…versioning_enabled` **et** `…object_lock_enabled` (pas de verrou sans versioning) · `unlocked` → `…object_lock_enabled` · `sensitive` (`classification=confidential`, sans SSE-KMS) → `…kms_encryption` · tout bucket sans SSE → `objectstorage_bucket_default_encryption`, contrôle que le référentiel ne déclare que pour Outscale (**défaut connu**, épinglé tel quel, #192) | `hardened` (privé, versionné, verrouillé, classé public) · `encrypted` (SSE-ONE configuré : muet sur le chiffrement par défaut) |
| Bases managées (db-dev-s) | 2, plan seul | `exposed` (ACL `0.0.0.0/0`, sans chiffrement au repos, sauvegardes désactivées) → `database_service_not_open_to_internet`, `database_encryption_at_rest_enabled`, `database_backup_enabled` | `hardened` (chiffrée sur volume Block, sauvegardes quotidiennes, ACL `10.0.0.0/8`) |

Toute ressource étiquetable porte le tag `pepin-qual`, tout nom commence par
`pepin-qual-`, et les noms de buckets embarquent les huit premiers caractères de
l'identifiant du projet : c'est là-dessus que la preuve de destruction filtre.

## Ce que le provider amont est connu pour laisser derrière lui au `destroy`

Lu avant de choisir les types de ressources, dans `scaleway/terraform-provider-scaleway`
et dans les notes de terrain du mainteneur
([Terraform sur Scaleway : provisionner, et surtout détruire](https://blog.stephane-robert.info/docs/cloud/scaleway/iac/terraform-provider-scaleway/),
2026-09-09). Ce que chacune a changé dans cette stack :

| Amont | État | Ce que ça fait | Ce que ce tenant en fait |
|---|---|---|---|
| [#4338](https://github.com/scaleway/terraform-provider-scaleway/issues/4338) `scaleway_instance_private_nic` : destroy échoue en `412 Can't delete a private network interface attached to a server` depuis la v2.81.0 (migration API v2) | **ouverte**, confirmée sur 2.82.0 par le mainteneur de Pépin le 2026-09-09 (2.80.0 : code 0, 7 s, 0 reste ; 2.81.0 et 2.82.0 : code 1, 4 restes sur 4). Arrêter le serveur ou le cibler ne change rien ; Terraform ne peut pas inverser l'ordre. | Bloque tout destroy d'un serveur avec une interface privée attachée. | **Aucune interface privée.** Les réseaux privés existent seuls, rien n'y est attaché. Provider épinglé à l'exact `2.82.0`, lockfile committé ; la parade de secours est `scw instance private-nic delete` (route v1) si une interface entrait un jour. Conséquence : la machine à deux cartes de l'audit externe (#E) ne peut pas être exercée sur Scaleway aujourd'hui. |
| [#2853](https://github.com/scaleway/terraform-provider-scaleway/issues/2853) instance : les volumes SBS ne sont pas supprimés au destroy | fermée le 2024-12-19 par `fix(instance_server): delete_after_termination for sbs volumes` | Un volume Block orphelin, facturé, que personne ne retrouve. | Volumes racine sur le stockage **local** de la gamme, `delete_on_termination = true` écrit plutôt que présumé, aucun volume additionnel ; volumes Instance **et** Block sont dans le listing de sortie. |
| [#2869](https://github.com/scaleway/terraform-provider-scaleway/issues/2869) un bucket avec des objets ne se supprime pas | fermée le 2025-01-10 (`force_destroy` vide le bucket depuis 2.49.0) | Un bucket non vide, versionné ou verrouillé bloque sa propre suppression. | Les buckets restent **vides**, `force_destroy = true` partout. |
| [#2821](https://github.com/scaleway/terraform-provider-scaleway/issues/2821) le groupe de sécurité créé par le produit bloque la destruction | fermée le 2025-02-14 : « explicitly create a security group and use it with the instance » | Un serveur tombé dans le groupe par défaut le retient. | Neuf groupes **explicites**, chaque serveur attaché explicitement à l'un d'eux. |
| [#4262](https://github.com/scaleway/terraform-provider-scaleway/issues/4262) IPv4 orpheline quand `ip_ids` est posé sur un load balancer | ouverte | Une IP flexible non suivie, facturée. | Pas de load balancer ; les deux IP flexibles sont des ressources Terraform, et le nettoyage de secours supprime les serveurs avec `with-ip=true` (l'option que les notes de terrain isolent). |
| [#2125](https://github.com/scaleway/terraform-provider-scaleway/issues/2125) impossible de détruire `scaleway_vpc_private_network` · [#3243](https://github.com/scaleway/terraform-provider-scaleway/issues/3243) `precondition failed: resource is still in use` | fermées (mise à jour du provider ; pas de reproduction) | Un réseau privé qui a encore quelque chose d'attaché. | VPC explicite, jamais rien d'attaché aux réseaux privés. |
| [#4320](https://github.com/scaleway/terraform-provider-scaleway/issues/4320) `iam_api_key` éphémère empile des clés non gérées | ouverte | Des clés recréées à chaque apply, inconnues de l'état. | `scaleway_iam_api_key` gérée, ordinaire ; le listing IAM filtre sur les applications du tenant et sur le préfixe de description. |
| [#2426](https://github.com/scaleway/terraform-provider-scaleway/issues/2426) `account_project` : 412 à la suppression | fermée | Un projet qui refuse de partir. | Aucun projet n'est créé : le tenant vit dans le projet épinglé. |
| b_ssd volumes are no longer supported (notes de terrain) | — | `scaleway_instance_volume` est refusé par l'API. | Aucun volume Instance ; un volume de données serait un `scaleway_block_volume`. |

La preuve de destruction, ce sont les onze listes des notes de terrain, plus les
familles IAM et Object Storage dont elles n'ont pas besoin : [`hooks.py inventory`](hooks.py)
liste serveurs, groupes de sécurité, IP flexibles, volumes Instance, snapshots
Instance, images Instance, volumes Block, snapshots Block, bases managées, leurs
sauvegardes et snapshots, réseaux privés, VPC, load balancers, passerelles
publiques, applications, politiques et clés IAM, et buckets, par les mêmes routes
d'API que la CLI `scw`. Le runner exige que le sous-ensemble du tenant de chaque
famille soit vide **et** que le listing entier soit égal à celui pris avant apply.

## Ce que ce tenant ne peut pas exercer, et pourquoi

- `governance_resource_region_in_eu` : toutes les régions Scaleway sont dans l'UE,
  aucun écart n'est constructible ; seul le `pass` est prouvé.
- `iam_user_mfa_enabled` : un utilisateur IAM est une personne réelle invitée dans
  l'organisation, hors périmètre d'un tenant jetable. Le contrôle s'observe sur les
  utilisateurs existants de l'organisation, et `expected.yaml` n'épingle que le fait
  qu'il **conclut**.
- `compute_instance_has_security_group` : un serveur Scaleway porte toujours un groupe.
- `iam_no_root_access_key` : `root_owned` n'est pas encore dérivé par le collecteur ;
  le contrôle reste `not-evaluated` sur les deux sources, et c'est épinglé.
- Neuf contrôles sont `not-evaluated` sur la source **live** (#195) parce que le collecteur
  ne lit pas encore la donnée (politique par défaut des groupes, réseaux privés,
  bases managées, user-data des serveurs, politiques IAM) : chacun est épinglé tel
  quel dans `expected.yaml`, donc un collecteur qui se met à les lire déplace un
  verdict sciemment, avec sa ligne de CHANGELOG.
- La machine à deux cartes (audit externe #E) : voir #4338 ci-dessus.

## Coût et budget

Tarifs horaires dans [`tenant.yaml`](tenant.yaml) pour ce qui est appliqué : 5 ×
DEV1-S à 0,008976 EUR/h et 2 IPv4 flexibles à 0,004 EUR/h — environ **0,053 EUR par
heure** de vie du tenant ; tout le reste appliqué (IAM, groupes de sécurité, buckets
vides) est gratuit, et les bases managées (0,0347 EUR/h pièce) ne sont jamais
provisionnées. Le runner multiplie les tarifs par la durée apply→destroy et écrit
l'estimation au rapport. Budget : 25 minutes mur, sinon NO-GO.

## Mesuré

<!-- qualification:measured -->
Run du 2026-09-09 (`GO`), sur le projet épinglé dans `expected.yaml` :

| | |
|---|---|
| Ressources | 40 dans le plan, 29 appliquées |
| `apply` | 11.2 s |
| attente de l'échéance de la clé expirée | 184.9 s |
| `scan --live` × 5 formats + scellement | 37.3 s, code de sortie 1 dans chaque format |
| bundle | `verify --re-derive` passe ; le bundle altéré d'un octet est refusé |
| `scan --terraform` | code de sortie 1 |
| `destroy` | 18.3 s, `terraform destroy` code 0, état vide, fichiers d'état purgés |
| preuve de destruction | 19 familles listées, aucune ressource du tenant, aucun delta avant/après |
| comparaison | live : 40 résultats, 0 différence ; terraform : 34 résultats, 0 différence |
| falsification | 3 attentes cassées en mémoire sur chaque source, toutes NO-GO |
| durée mur | 259.9 s (budget 25 min) |
| coût estimé | 0.0037 EUR (0.07 h × 0.0529 EUR/h) |

Lignes d'information du run (non gardées) : `iam_user_mfa_enabled` échoue sur un
utilisateur de l'organisation (hors périmètre du tenant), et
`objectstorage_bucket_default_encryption` est épinglé comme défaut connu (#192, voir
`expected.yaml`).
<!-- /qualification:measured -->

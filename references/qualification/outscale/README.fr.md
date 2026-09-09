> [🇬🇧 English](README.md) · 🇫🇷 Français

# Tenant de qualification Outscale

79 ressources Terraform, plus 6 buckets OOS et une politique EIM inline créés par le
crochet `extra`, en `eu-west-2` / `eu-west-2a`, sur le compte que l'environnement
nomme (`PEPIN_QUAL_OSC_ACCOUNT`, confirmé par `ReadAccounts` avant tout apply). Une
faute par ressource, un contre-exemple par contrôle. Appliqué, scanné, scellé,
détruit et comparé par `PEPIN_GATE_LIVE=1 PROVIDER=outscale mise run qualify` (voir
[le README générique](../README.fr.md)).

À la différence de Scaleway, un scan live Outscale lit **tout ce que cette stack
crée** (16 types au contrat de `providers/outscale.yaml`) : les deux passes voient le
même tenant, et c'est ici que se trouvent les cas que Scaleway ne sait pas exercer en
live — la machine à deux cartes, la clé root, la famille `iam_policy_*` par les
documents EIM, les snapshots, l'OMI publique, les load balancers.

## Ce que la stack crée

| Famille | Nombre | Ressource fautive → contrôle | Contre-exemple (doit rester muet) |
|---|---:|---|---|
| Groupes de sécurité (tous dans le Net) | 10 (+ 14 règles) | `ssh_open` → `…tcp_port_22` · `rdp_open` → `…tcp_port_3389` · `db_open` (5432) → `…high_risk_tcp_ports` · `snmp_open` (UDP 161) → `…high_risk_udp_ports` · `any_open` → `…all_ports` **et les quatre précédents** · `quartet` (quatre `/2`) → `…tcp_port_22` par évasion · `egress_any` (sortie explicite `-1 → 0.0.0.0/0`) → `network_securitygroup_unrestricted_egress` · `net_ssh_open` → `…tcp_port_22` (le groupe de la carte secondaire) · le groupe **`default` du Net**, créé par le produit et lu par une source de données → `network_securitygroup_default_restrict_traffic` (règle d'usine) et `unrestricted_egress` (sa règle sortante par défaut) | `hardened` (SSH depuis `10.0.0.0/8`, HTTPS depuis Internet, sortie TCP 443) · `net_closed` · tous les groupes ont `remove_default_outbound_rule = true` |
| VM (`tinav6.c2r4p2`) | 8 | `exposed` (IP publique + `ssh_open`) → `compute_instance_public_ip_with_open_securitygroup` · `two_nics` (carte primaire privée et fermée, carte **secondaire** publique dans `net_ssh_open`) → le même contrôle, par l'appariement des cartes de #187 · `untagged` → `governance_resource_required_tags` · `secrets` (cloud-init avec mot de passe, `UserData` base64) → `compute_instance_no_secrets_in_user_data` · `unprotected` (`Env=prod`, sans protection) → `compute_instance_deletion_protection` | `private` (même groupe ouvert, pas d'IP) · `hardened` (IP publique, groupe restrictif, `Env=prod`, **protégée**) · `two_nics_inverse` (primaire publique et fermée, secondaire privée et ouverte : aplatie elle paraîtrait exposée, appariée elle ne l'est pas) · `untagged` reçoit aussi un *non concluant* sur la protection (pas d'environnement lisible, ADR-0015) |
| Cartes secondaires, IP publiques | 2 + 4 | — | `outscale_nic` + `outscale_nic_link`, jamais le bloc `nics` (voir plus bas) |
| Volumes | 2 (+ 8 racines) | `unsnapshotted` et les huit volumes racine → `blockstorage_volume_snapshots_exist` | `snapshotted` (snapshot privée terminée, source des deux OMI) · volumes racine étiquetés par `outscale_tag` |
| Snapshots | 2 | `public` (`GlobalPermission`) → `blockstorage_snapshot_not_public` | `private` |
| Images (OMI) | 2 | `public` (permission de lancement globale) → `compute_image_not_public` | `private` |
| Nets, sous-réseaux | 2 + 2 | `undocumented` → `network_documented` · `autoip` (`MapPublicIpOnLaunch`) → `network_subnet_no_public_ip_by_default` | `documented` · `main` · un **appairage entre les deux Nets** (même compte) est le contre-exemple de `network_peering_cross_organization` |
| Load balancers (LBU) | 2 | `http` (écouteur HTTP seul, sans journal) → `loadbalancer_ssl_listeners`, `loadbalancer_logging_enabled` · `tcp443` (écouteur TCP 443) → *non concluant* sur `loadbalancer_ssl_listeners` (candidat passthrough TLS), écart `loadbalancer_logging_enabled` | — (un journal activé exige un bucket OOS cible ; non construit) |
| EIM | 1 utilisateur, 3 clés, 5 politiques | `no_expiry` → `iam_accesskey_expiration_set` · `root` (clé du compte appelant) → `iam_no_root_access_key` · `admin` (`Action: *`) → `iam_policy_no_administrative_privileges` · `wildcard`, `notaction`, `escalation` (`Resource: *`) → `iam_policy_no_wildcard_resource` · `notaction` → `iam_policy_no_notaction_notresource` · `escalation` (`CreateAccessKey`, `LinkPolicy`) → `iam_policy_no_privilege_escalation` | `expiring` (échéance à J+2) · `scoped` (ressource OOS nommée) · toutes les clés sont créées **INACTIVES** : les règles ne lisent pas l'état, et aucun secret utilisable ne circule |
| Règles d'accès API | 2 | `public` (`0.0.0.0/0`) → `iam_apiaccessrule_no_public_cidr` | `private` (`10.0.0.0/8`) ; les règles s'ajoutent, aucune n'enferme personne dehors |
| Buckets OOS (crochet `extra`, `aws s3api` sur `oos.eu-west-2.outscale.com`) | 6 | `public` (ACL) et `policy` (`Principal: *`) → `objectstorage_bucket_public_access` · `unversioned` → `…versioning_enabled` et `…object_lock_enabled` · `unlocked` → `…object_lock_enabled` · `unencrypted` (sans SSE) → `objectstorage_bucket_default_encryption` | `hardened` (privé, verrouillé, SSE, étiqueté) |
| Politique EIM inline (crochet `extra`, `PutUserPolicy`) | 1 | `pepin-qual-inline-admin` (`Action: *` sur l'utilisateur du tenant, l'incident fondateur de l'ADR-0006) → `iam_policy_no_administrative_privileges` | — |

Toute ressource étiquetable porte le tag `pepin-qual`, tout nom commence par
`pepin-qual-`, et les noms de buckets embarquent les huit premiers chiffres du compte.

## Ce que le provider amont est connu pour laisser derrière lui au `destroy`

Lu avant de choisir les types de ressources, dans `outscale/terraform-provider-outscale`
(1.8.0 épinglé, lockfile committé). Ce que chacune a changé dans cette stack :

| Amont | État | Ce que ça fait | Ce que ce tenant en fait |
|---|---|---|---|
| [#88](https://github.com/outscale/terraform-provider-outscale/issues/88) deletion_protection : une VM protégée fait échouer le destroy (`Error deleting the VM`) | ouverte, confirmée par les mainteneurs en 2024-08 (« you must apply the change first ») | Le contre-exemple protégé bloquerait tout le destroy. | `deletion_protection = var.deletion_protection`, et `tenant.yaml` déclare `pre_destroy_vars: {deletion_protection: false}` : le runner **applique** la déprotection avant de détruire. Le nettoyage de secours fait `UpdateVm DeletionProtection=false` d'abord. |
| [#28](https://github.com/outscale/terraform-provider-outscale/issues/28) / [#137](https://github.com/outscale/terraform-provider-outscale/issues/137) `outscale_nic_private_ip` : 400/409 au destroy, carte et IP primaire non supprimées | ouvertes | Des IP privées secondaires bloquent la destruction de la carte. | Aucune `outscale_nic_private_ip` : une IP privée primaire par carte. |
| [#778](https://github.com/outscale/terraform-provider-outscale/issues/778), [#424](https://github.com/outscale/terraform-provider-outscale/issues/424), [#50](https://github.com/outscale/terraform-provider-outscale/issues/50), [#448](https://github.com/outscale/terraform-provider-outscale/issues/448) blocs `nics` / `primary_nic` d'`outscale_vm` : VM remplacée à chaque plan, carte « oubliée » | #778 fermée 2026-07 (ne sera pas corrigée : « use `primary_nic` with a separate `nic_link` ») | Une seconde carte déclarée en ligne ne se détruit pas proprement. | Les cartes secondaires sont des `outscale_nic` + `outscale_nic_link` (+ `outscale_public_ip_link` sur la carte), la parade recommandée par les mainteneurs. |
| [#622](https://github.com/outscale/terraform-provider-outscale/issues/622) un groupe de sécurité encore attaché ne se détruit pas | fermée 2026-02 | Un groupe référencé par une VM ou un LBU retient. | Chaque groupe n'est attaché qu'à des VM Terraform, détruites d'abord par dépendance. |
| [#663](https://github.com/outscale/terraform-provider-outscale/issues/663) destruction d'une politique de stickiness LBU en échec | fermée | — | Aucune politique de stickiness. |
| [#6](https://github.com/outscale/terraform-provider-outscale/issues/6) IP publique relâchée trop tôt | fermée 2022 | — | Ressources `outscale_public_ip` + `outscale_public_ip_link`, ordonnées par dépendance. |
| Mesuré ici : **`terraform destroy` rend 0 alors qu'un appairage de Nets survit** | à signaler en amont | Un `outscale_net_peering` accepté (+ `_acceptation`) était encore `active` après un destroy réussi. | Le nettoyage de secours le supprime (`DeleteNetPeering`) et la preuve de destruction l'attrape. |
| Mesuré ici : **les suppressions sont asynchrones** | — | 26 s après un destroy réussi, les deux LBU, dix volumes et deux snapshots étaient encore listés ; deux minutes plus tard, ils « n'existaient plus ». | La preuve de destruction **reliste jusqu'au vide**, bornée par `settle_seconds` (240 s par défaut) : ce qui survit au délai est un reste. |
| Mesuré ici : un groupe du cloud public ne peut pas retirer sa règle sortante par défaut (`remove_default_outbound_rule` exige `net_id`) | — | Tout groupe du cloud public porte `unrestricted_egress` par construction. | Tous les groupes vivent dans le Net. |
| Mesuré ici : OOS active le versioning d'un bucket Object Lock et **le fige** (`InvalidBucketState` sur `PutBucketVersioning`) | — | — | Le crochet n'active le versioning explicitement que sur `unlocked`. |
| Mesuré ici (trois runs) : **`LinkPublicIp` par `VmId` est refusé (400 `InvalidResource`) sur une VM qui a déjà deux cartes** | — | Le contre-exemple `two_nics_inverse` ne se construisait pas quand la seconde carte était attachée d'abord. | L'IP publique est liée pendant que la VM n'a que sa carte primaire ; l'`outscale_nic_link` de la seconde carte dépend de ce lien. |
| Mesuré ici : **`DeleteImage` ne supprime pas la snapshot qu'une OMI créée depuis une VM implique** | — | Deux snapshots orphelines après un destroy réussi, attrapées par le delta avant/après. | Les deux OMI sont construites depuis la snapshot gérée du tenant (`block_device_mappings`) : rien d'implicite n'est créé. |

La preuve de destruction liste 20 familles par l'OAPI (VM, cartes, IP publiques,
groupes, volumes, snapshots, images, Nets, sous-réseaux, services Internet, NAT,
tables de routage, appairages, LBU, keypairs, règles d'accès API, utilisateurs,
politiques, clés d'accès, buckets), filtrées sur le tag et le préfixe du tenant, et
compare le listing entier à celui pris avant apply. `octl`, la CLI de référence du
fournisseur, est ce qu'un mainteneur rejoue à la main :

```bash
octl iaas vm list --filter 'Tags[].Key:pepin-qual' -o table
octl iaas securitygroup list -o table --filter 'SecurityGroupName:pepin-qual-'
octl iaas publicip list -o table
octl iaas volume list -o table
octl iaas snapshot list -o table
octl iaas image list -o table --filter 'ImageName:pepin-qual-'
octl iaas net list -o table
octl iaas netpeering list -o table
octl iaas loadbalancer list -o table
octl iaas user list -o table
octl iaas policy list -o table
octl storage bucket list -o table
```

Jamais `octl profile current` : il imprime la clé secrète en clair. Et lire les
états : un moment après un destroy, `octl iaas vm list` montre encore les VM du
tenant en `terminated` et `octl iaas netpeering list` son appairage en `deleted` (32
et 4 après quatre runs). Ce sont des entrées terminales, pas des ressources ;
l'inventaire API de la preuve exclut ces états.

## Ce que ce tenant ne peut pas exercer, et pourquoi

`tenant.yaml`, `not_exercised:` — une ressource hors UE (un compte Outscale vit dans
une région), une clé de plus de 90 jours, un appairage entre comptes, la politique
d'accès API du compte (MFA imposée, plafond d'échéance des clés : les forcer
enfermerait la clé qui qualifie), un tenant sans aucune règle d'accès API, un journal
LBU activé, un groupe `default` vidé de sa règle d'usine, les clusters OKS.

## Coût et budget

Tarifs horaires dans [`tenant.yaml`](tenant.yaml) : 8 × `tinav6.c2r4p2` à 0,09 EUR/h
(le type que le mainteneur a donné pour son compte), 2 LBU à 0,03 EUR/h, 4 IP
publiques, ~100 Go de volumes — environ **0,82 EUR par heure** de vie du tenant.
Budget : 40 minutes mur, sinon NO-GO.

## Mesuré

<!-- qualification:measured -->
Run du 2026-09-09 (`GO`), sur le compte que l'environnement nomme :

| | |
|---|---|
| Ressources | 79 dans le plan, toutes appliquées, plus 7 ressource(s) hors Terraform : pepin-qual-30044641-public, pepin-qual-30044641-policy, pepin-qual-30044641-unversioned, pepin-qual-30044641-unlocked, pepin-qual-30044641-unencrypted, pepin-qual-30044641-hardened, pepin-qual-user/pepin-qual-inline-admin |
| `apply` | 210.3 s |
| `scan --live` × 5 formats + scellement | 50.8 s, code de sortie 1 dans chaque format |
| bundle | `verify --re-derive` passe ; le bundle altéré d'un octet est refusé |
| `scan --terraform` | code de sortie 1 |
| `destroy` | 155.3 s — pré-destroy (deletion_protection=False) rc=0 ; terraform destroy rc=0 ; état vide ; 2 fichier(s) d'état purgé(s) |
| preuve de destruction | 20 familles listées, aucune ressource du tenant, aucun delta avant/après (après 40s de suppressions asynchrones) |
| comparaison | live : 145 résultats, 0 différence ; terraform : 54 résultats, 0 différence |
| falsification | 3 attentes cassées en mémoire sur chaque source, toutes NO-GO |
| durée mur | 511.5 s (budget 40 min) |
| coût estimé | 0.0989 EUR (0.121 h × 0.815 EUR/h) |

Écarts hors tenant rapportés par le scan live et non gardés : 74 (clés,
politiques, règles d'accès API, buckets et groupes par défaut propres au compte).
<!-- /qualification:measured -->

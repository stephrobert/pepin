> [🇬🇧 English](README.md) · 🇫🇷 Français

# Tenant de qualification Exoscale (source Terraform seulement)

**Aucun compte Exoscale n'est disponible.** Ce tenant se qualifie donc en
`--plan-only` : 37 ressources sont planifiées, rien n'est créé, et
[`expected.yaml`](expected.yaml) épingle ce que `pepin scan exoscale --terraform`
rend sur ce plan — le chemin qu'un utilisateur suit sans compte
([ADR-0012](../../docs/adr/0012-aucun-identifiant-en-ci.md) le préfère). Cela a déjà
de la valeur : les verdicts Terraform de 26 contrôles sont fixés, avec leurs
contre-exemples.

Ce qu'il ne fait **pas**, et le dit (`tenant.yaml` : `live: unavailable`) :
appliquer, scanner en `--live`, sceller un bundle, détruire, prouver la destruction.
`mise run qualify` sur ce tenant est refusé ; `mise run qualify:plan`
(`PROVIDER=exoscale`) est le seul mode. [`hooks.py`](hooks.py) fournit les
identifiants factices qu'un plan exige et refuse tout le reste. La lecture des
issues de `exoscale/terraform-provider-exoscale` sur `destroy` est due avant le
premier apply, pas avant le premier plan.

```bash
PROVIDER=exoscale mise run qualify:plan
```

## Ce que le plan porte

| Famille | Nombre | Ressource fautive → contrôle | Contre-exemple (doit rester muet) |
|---|---:|---|---|
| Groupes de sécurité | 9 (+ 15 règles) | `ssh_open` → `…tcp_port_22` · `rdp_open` → `…tcp_port_3389` · `db_open` (5432) → `…high_risk_tcp_ports` · `snmp_open` (UDP 161) → `…high_risk_udp_ports` · `any_open` (tout TCP et tout UDP) → les quatre précédents · `quartet` (quatre `/2`) → `…tcp_port_22` par évasion · `undocumented_flow` (règle sans description, port 443) → `network_flow_matrix_documented` | `hardened` (SSH depuis `10.0.0.0/8`, HTTPS depuis Internet, sortie TCP 443, chaque flux décrit) |
| Réseaux privés | 2 | `undocumented` → `network_documented` | `documented` (Owner, Project, Env, description) |
| Instances | 4 | `untagged` → `governance_resource_required_tags` · `secrets` (cloud-init avec mot de passe) → `compute_instance_no_secrets_in_user_data` · `swiss` (zone `ch-gva-2`) → `governance_resource_region_in_eu` (low : hors UE, espace européen de confiance) | `hardened` (étiquetée, `de-fra-1`, cloud-init anodin) |
| Volume block storage | 1 | — (chiffré par construction : `blockstorage_volume_encryption` passe) | — |
| Clusters SKS | 2 | `weak` (`starter`, sans auto-upgrade, audit désactivé) → `kubernetes_cluster_control_plane_highly_available`, `…auto_upgrade_enabled`, `…audit_logging_enabled` | `strong` (`pro`, auto-upgrade, endpoint d'audit) |
| Rôles IAM | 4 | `admin` (`allow` par défaut) → `iam_role_no_admin_privileges` · `no_source_ip` → `iam_role_source_ip_restricted` · `unbounded` (sans `duration(`) → `iam_role_key_lifetime_bounded` | `restricted` (`deny` par défaut, `source_ip`, `duration(`) |
| Fournisseur | — | `governance_provider_sovereignty` échoue sur `exoscale` lui-même : siège en Suisse, contrôle capitalistique extra-UE — des faits du descripteur, pas du tenant | — |

## Ce que ce tenant ne peut pas exercer, et pourquoi

- **`network_securitygroup_allow_ingress_from_internet_to_all_ports` et
  `network_securitygroup_unrestricted_egress` ne peuvent pas se déclencher chez
  Exoscale**, sur aucune source : une règle de groupe n'a pas de protocole « tout »
  (`tcp`, `udp`, `icmp`, `icmpv6`, `ah`, `esp`, `gre`, `ipip` seulement, provider
  0.71.0 et API), et les deux règles communes exigent `protocol == all`. `any_open`
  ouvre tout TCP et tout UDP, les quatre autres contrôles d'entrée parlent ; ces deux-là
  sont épinglés `pass` par absence, ce que le référentiel et la matrice de couverture
  devraient cesser de présenter comme de la couverture (#202).
- **`network_flow_matrix_documented` rend `pass` sur un plan dont la règle
  `undocumented_flow` n'a pas de description** : une description non écrite est
  *calculée* sur un plan, l'attribut n'est pas projeté, la garde de capacité se tait,
  et l'assessment conclut « conforme » — le faux vert que l'intersection de
  l'[ADR-0006](../../docs/adr/0006-jamais-un-pass-non-prouve.md) existe pour empêcher.
  Épinglé comme **défaut connu** (#201) ; la correction fera rougir cette porte
  sciemment.
- L'IP publique d'une instance et l'état d'usage d'un volume sont calculés à
  l'apply : `compute_instance_public_ip_with_open_securitygroup` et
  `blockstorage_volume_snapshots_exist` attendent la moitié live.
- Buckets SOS : le provider n'a pas de ressource de bucket (`exoscale_sos_bucket_policy`
  seulement) ; ils seront créés par un crochet `extra`, comme chez Outscale, quand un
  compte existera.
- Les utilisateurs IAM sont des personnes réelles, pas des ressources Terraform.

## Mesuré

Run plan seul (`PROVIDER=exoscale mise run qualify:plan`) : 37 ressources planifiées,
`scan --terraform` en code 1, 26 contrôles épinglés sur la source Terraform, GO, avec
une attente cassée en mémoire qui rend NO-GO. Rien de créé, rien à détruire, aucun
coût.

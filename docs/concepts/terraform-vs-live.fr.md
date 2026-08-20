> [🇬🇧 English](terraform-vs-live.md) · 🇫🇷 Français

# Plan Terraform contre scan live

Pépin évalue un seul modèle normalisé, et ce modèle peut être alimenté par trois sources : un
plan Terraform, une collecte live via l'API du fournisseur, ou un export d'inventaire
normalisé. Les règles ne savent jamais laquelle elles lisent, [c'est l'architecture](../coverage.fr.md).
Ce qui change, c'est **ce que la source sait voir**, et la différence n'est pas cosmétique : le
même contrôle, sur la même ressource, peut conclure d'un côté et refuser de conclure de
l'autre.

Aucune des deux sources n'est supérieure. Un plan audite ce qui n'existe pas encore ; un scan
live audite ce qui tourne réellement, y compris tout ce que personne n'a écrit en Terraform.

## Les deux sources, côte à côte

| | Plan Terraform | Scan live |
|---|---|---|
| Identifiants | aucun | oui, les variables d'environnement natives du fournisseur |
| Ce qui est audité | l'état que le code déclare | la configuration effective |
| Ressources créées hors du code | invisibles | vues |
| Dérive entre le code et la réalité | invisible par construction | c'est ce qu'il mesure |
| Avant déploiement | oui, sur une pull request | non |
| Données de runtime (attributs calculés à l'apply) | partiellement inconnues | disponibles |
| Coût, rayon d'impact | rien provisionné, rien facturé | lit un tenant réel |
| Usage typique | la porte sur une demande de fusion | le contrôle de posture périodique |

```bash
pepin scan scaleway --terraform plan.json     # aucun compte, rien de créé
pepin scan scaleway --live --region fr-par    # lit le tenant réel
```

Les deux drapeaux s'excluent mutuellement, et Pépin refuse les deux plutôt que d'en choisir un
([Référence de la CLI](../reference/cli.fr.md#une-source-et-une-seule)).

## Divergence 1 : un attribut inconnu au stade du plan

Ce n'est pas une hypothèse. C'est le faux positif corrigé en **v0.1.1**, et il se rejoue depuis
le plan committé dans ce dépôt.

Le plan déclare une instance et un groupe de sécurité. Le groupe de l'instance est créé par le
même plan, donc son identifiant n'existe pas encore : Terraform le marque `unknown after apply`
et `planned_values` n'a tout simplement pas la clé.

<!-- pepin:gen drift-plan-unknown -->
```json
{
  "address": "scaleway_instance_server.web",
  "change": {
    "after": {},
    "after_unknown": {
      "id": true,
      "public_ips": true,
      "security_group_id": true
    }
  },
  "type": "scaleway_instance_server"
}
```
<!-- /pepin:gen drift-plan-unknown -->

Trois clés, toutes inconnues : l'identifiant, les adresses publiques et le groupe de sécurité.
Le contrôle qui lit le filtrage de l'instance ne peut donc pas décider, et il le dit plutôt que
de deviner :

<!-- pepin:gen drift-plan-status -->
```json
{
  "control": "compute_instance_has_security_group",
  "evidence": {
    "attribute": "security_group_ids",
    "observed": "attribut « security_group_ids » non collecté sur les ressources de type « compute_instance » (garde de capacité)",
    "proves": [
      "",
      "",
      ""
    ],
    "source": "security_group_ids=terraform-plan:scaleway_instance_server observed=0/1"
  },
  "references": [
    {
      "framework": "scsl",
      "id": "CLD-CMP-1"
    },
    {
      "framework": "cis-v8",
      "id": "4.4"
    },
    {
      "framework": "iso-27001",
      "id": "A.8.20"
    },
    {
      "framework": "iso-27017",
      "id": "CLD.9.5.2"
    },
    {
      "framework": "secnumcloud-3.2",
      "id": "13.2"
    }
  ],
  "severity": "critical",
  "status": "not-evaluated",
  "subject": "scaleway",
  "title": "Machine sans filtrage réseau"
}
```
<!-- /pepin:gen drift-plan-status -->

Avant la v0.1.1, une transformation de collecte fabriquait une liste vide pour cette clé
absente, la garde de capacité de la règle était satisfaite par cette liste vide, et le scan
signalait une VM « sans groupe de sécurité » en `CRITICAL` alors qu'elle était correctement
rattachée. La transformation ne s'applique désormais que si la clé source existe : **absent
signifie que la source n'expose pas l'information ; présent-et-vide est une information.**

Sur une source où l'attribut est collecté, le même contrôle conclut. Voici la même instance
dans un inventaire à la forme que produit le collecteur live, `vm_id` depuis `Server.ID` et
`security_group_ids` depuis le groupe de sécurité du serveur, selon
`providers/scaleway.yaml` :

<!-- pepin:gen drift-live-fixture -->
```json
{
  "provider": "scaleway",
  "resources": [
    {
      "provider": "scaleway",
      "type": "compute_instance",
      "id": "3f2b1c00-0000-4a00-9000-000000000001",
      "name": "web",
      "region": "fr-par",
      "attributes": {
        "vm_id": "3f2b1c00-0000-4a00-9000-000000000001",
        "state": "running",
        "security_group_ids": ["b1a2c3d4-0000-4a00-9000-000000000002"],
        "tags": [{"key": "env", "value": "prod"}]
      }
    }
  ]
}
```
<!-- /pepin:gen drift-live-fixture -->

<!-- pepin:gen drift-live-status -->
```json
{
  "control": "compute_instance_has_security_group",
  "evidence": {
    "observed": "aucune non-conformité détectée sur les ressources de type « compute_instance » collectées (contrat vérifié)",
    "proves": [
      "",
      "",
      ""
    ],
    "source": "export"
  },
  "references": [
    {
      "framework": "scsl",
      "id": "CLD-CMP-1"
    },
    {
      "framework": "cis-v8",
      "id": "4.4"
    },
    {
      "framework": "iso-27001",
      "id": "A.8.20"
    },
    {
      "framework": "iso-27017",
      "id": "CLD.9.5.2"
    },
    {
      "framework": "secnumcloud-3.2",
      "id": "13.2"
    }
  ],
  "severity": "critical",
  "status": "pass",
  "subject": "scaleway",
  "title": "Machine sans filtrage réseau"
}
```
<!-- /pepin:gen drift-live-status -->

> **Aucun compte live n'a servi à produire cette page.** L'inventaire ci-dessus est une fixture
> écrite à la forme que produit le collecteur, et le scan qui en est fait est réel. Un scan live
> contre un tenant Scaleway emprunterait le même chemin de code depuis le même attribut ; rien
> ici ne prétend l'avoir exécuté.

`not-evaluated` puis `pass`, sur le même contrôle, le même tenant, le même jour. Le verdict n'a
pas changé parce que la configuration a changé : il a changé parce que la source voyait un
attribut de plus. C'est tout l'objet de cette page.

## Divergence 2 : un booléen que le plan rend en chaîne

La seconde divergence est plus discrète, parce qu'elle ne change pas un statut : elle change un
**type**.

La permission de lancement d'une image machine Outscale est un booléen côté API
(`PermissionsToLaunch.GlobalPermission`), et une *chaîne* dans un plan Terraform
(`permission_additions[].global_permission`). Dans le plan committé :

<!-- pepin:gen drift-bool-plan -->
```json
{
  "address": "outscale_image_launch_permission.public_omi",
  "change": {
    "after": {
      "image_id": "ami-12345678",
      "permission_additions": [
        {
          "account_ids": null,
          "global_permission": "true"
        }
      ]
    },
    "after_unknown": {
      "permission_additions": [
        {}
      ]
    }
  },
  "type": "outscale_image_launch_permission"
}
```
<!-- /pepin:gen drift-bool-plan -->

`"true"`, avec les guillemets. Une règle écrite `attributes.public == true` raterait en silence
toutes les images partagées publiquement dans un plan, tout en les attrapant en live. Les
règles communes passent donc par un helper partagé (`truthy`, dans
`internal/commonrules/rules/lib.rego`) qui accepte le booléen `true` comme la chaîne `"true"`,
insensiblement à la casse. L'écart est bien trouvé :

<!-- pepin:gen drift-bool-finding -->
```json
{
  "code": "CLD-STO-2",
  "labels": {
    "category": "security",
    "check": "compute_image_not_public",
    "provider": "outscale"
  },
  "message": "Image machine « ami-12345678 » partagée publiquement (lancement autorisé à tous).",
  "remediation": "Retirer le partage public de l'image ; la réserver aux comptes légitimes.",
  "severity": "high",
  "subject": "ami-12345678",
  "title": "Image machine partagée publiquement"
}
```
<!-- /pepin:gen drift-bool-finding -->

La leçon se généralise : un plan est un *rendu* de la configuration par le schéma d'un provider
Terraform, pas la charge utile de l'API. Les types peuvent différer, les noms peuvent différer,
et des attributs calculés peuvent manquer entièrement. C'est pourquoi la normalisation vit dans
le collecteur et le mapper, par fournisseur et par source, et jamais dans les règles.

## Les contrôles qu'une seule source sait observer

Ce tableau est calculé depuis les descripteurs de fournisseurs et le référentiel : pour ces
couples, une source produit le type de ressource et son attribut décisif, et l'autre non.

<!-- pepin:gen single-source -->
| Contrôle | Fournisseur | Observable uniquement via | Motif |
|---|---|---|---|
| `blockstorage_snapshot_not_public` | outscale | live | cette source ne produit aucune ressource de type « blockstorage_snapshot » |
| `blockstorage_volume_snapshots_exist` | outscale | live | cette source ne produit aucune ressource de type « blockstorage_volume » |
| `compute_instance_deletion_protection` | outscale | live | attribut décisif « deletion_protection » non projeté par cette source : garde de capacité, le scan rend « not-evaluated » |
| `compute_instance_no_secrets_in_user_data` | scaleway | terraform | attribut décisif « user_data » non projeté par cette source : garde de capacité, le scan rend « not-evaluated » |
| `compute_instance_public_ip_with_open_securitygroup` | scaleway | live | attribut décisif « public_ip » non projeté par cette source : garde de capacité, le scan rend « not-evaluated » |
| `database_backup_enabled` | scaleway | terraform | cette source ne produit aucune ressource de type « managed_database » |
| `database_encryption_at_rest_enabled` | scaleway | terraform | cette source ne produit aucune ressource de type « managed_database » |
| `database_service_not_open_to_internet` | scaleway | terraform | cette source ne produit aucune ressource de type « managed_database » |
| `governance_resource_region_in_eu` | outscale | live | attribut décisif « region » non projeté par cette source : garde de capacité, le scan rend « not-evaluated » |
| `iam_accesskey_expiration_set` | outscale | live | cette source ne produit aucune ressource de type « access_key » |
| `iam_account_mfa_enforced` | outscale | live | cette source ne produit aucune ressource de type « api_access_policy » |
| `iam_apiaccesspolicy_max_key_expiration` | outscale | live | cette source ne produit aucune ressource de type « api_access_policy » |
| `iam_apiaccessrule_defined` | outscale | live | cette source ne produit aucune ressource de type « api_access_summary » |
| `iam_apiaccessrule_no_public_cidr` | outscale | live | cette source ne produit aucune ressource de type « api_access_rule » |
| `iam_no_root_access_key` | outscale | live | cette source ne produit aucune ressource de type « access_key » |
| `iam_policy_no_privilege_escalation` | scaleway | terraform | cette source ne produit aucune ressource de type « iam_policy » |
| `iam_user_mfa_enabled` | exoscale | live | cette source ne produit aucune ressource de type « iam_user » |
| `iam_user_mfa_enabled` | scaleway | live | cette source ne produit aucune ressource de type « iam_user » |
| `kubernetes_cluster_auto_upgrade_enabled` | outscale | live | cette source ne produit aucune ressource de type « kubernetes_cluster » |
| `kubernetes_cluster_control_plane_highly_available` | outscale | live | cette source ne produit aucune ressource de type « kubernetes_cluster » |
| `kubernetes_cluster_deletion_protection` | outscale | live | cette source ne produit aucune ressource de type « kubernetes_cluster » |
| `kubernetes_cluster_not_publicly_accessible` | outscale | live | cette source ne produit aucune ressource de type « kubernetes_cluster » |
| `loadbalancer_logging_enabled` | outscale | live | cette source ne produit aucune ressource de type « load_balancer » |
| `loadbalancer_ssl_listeners` | outscale | live | cette source ne produit aucune ressource de type « load_balancer » |
| `network_peering_cross_organization` | outscale | live | cette source ne produit aucune ressource de type « network_peering » |
| `network_securitygroup_default_deny` | scaleway | terraform | cette source ne produit aucune ressource de type « security_group » |
| `network_securitygroup_default_restrict_traffic` | outscale | live | attribut décisif « security_group_name » non projeté par cette source : garde de capacité, le scan rend « not-evaluated » |
| `objectstorage_bucket_default_encryption` | outscale | live | cette source ne produit aucune ressource de type « object_storage_bucket » |
| `objectstorage_bucket_kms_encryption` | scaleway | live | attribut décisif « sse_kms_enabled » non projeté par cette source : garde de capacité, le scan rend « not-evaluated » |
| `objectstorage_bucket_object_lock_enabled` | exoscale | live | cette source ne produit aucune ressource de type « object_storage_bucket » |
| `objectstorage_bucket_object_lock_enabled` | outscale | live | cette source ne produit aucune ressource de type « object_storage_bucket » |
| `objectstorage_bucket_public_access` | exoscale | live | cette source ne produit aucune ressource de type « object_storage_bucket » |
| `objectstorage_bucket_public_access` | outscale | live | cette source ne produit aucune ressource de type « object_storage_bucket » |
| `objectstorage_bucket_versioning_enabled` | exoscale | live | cette source ne produit aucune ressource de type « object_storage_bucket » |
| `objectstorage_bucket_versioning_enabled` | outscale | live | cette source ne produit aucune ressource de type « object_storage_bucket » |
| `objectstorage_bucket_versioning_enabled` | scaleway | live | attribut décisif « versioning » non projeté par cette source : garde de capacité, le scan rend « not-evaluated » |
<!-- /pepin:gen single-source -->

Les totaux par fournisseur, et le motif de chaque case non ✅, sont dans la
[matrice de couverture](../coverage.fr.md).

## Comparer deux scans honnêtement

Deux rapports du même tenant, l'un depuis un plan et l'autre depuis le live, ne sont **pas**
interchangeables. Avant de lire un écart comme une dérive, vérifier les trois choses qui
l'expliquent sans qu'aucune configuration n'ait bougé.

**1. La source.** Chaque assessment la porte, et elle est scellée dans le bundle de preuve. Les
deux documents derrière cette page la déclarent ainsi :

<!-- pepin:gen drift-source-provenance -->
```json
{"run": {"source": "terraform-plan"}}
{"run": {"source": "export"}}
```
<!-- /pepin:gen drift-source-provenance -->

**2. La couverture.** Un contrôle `not-evaluated` d'un côté et `pass` de l'autre peut être une
différence de couverture, pas une différence de posture. Le champ de preuve dit laquelle : il
nomme l'attribut ou le type de ressource qui lui a manqué.

**3. La langue.** Codes, statuts et sévérités sont stables d'une langue à l'autre ; titres,
messages et preuves ne le sont pas. Un pipeline qui compare le *texte* d'un rapport doit
épingler `PEPIN_LANG`
([Formats de sortie](../reference/output-formats.fr.md#formats-et-langue)).

## Le flux de travail qui utilise les deux

- **Sur une pull request : le plan.** Aucun identifiant, rien de provisionné, et le retour
  arrive avant que la ressource n'existe. Bloquer sur le code de sortie 1
  ([GitHub Actions](../guides/github-actions.fr.md), [GitLab CI](../guides/gitlab-ci.fr.md)).
- **Périodiquement : le live.** Une fois par jour ou par semaine, contre le tenant réel, avec
  une clé en lecture seule. C'est ce qui attrape ce que personne n'a écrit en Terraform, ce
  qu'un clic en console a changé, et ce qu'un identifiant expiré masque, ce dernier cas
  ressortant en code de sortie 3 plutôt qu'en faux vert
  ([Codes de sortie](../reference/exit-codes.fr.md)).
- **Sceller l'exécution live.** C'est le scan live qui mérite `--seal` : il a observé un tenant
  réel à un instant réel ([Le bundle de preuve](../guides/evidence-bundles.fr.md)).

Un pipeline qui ne fait que du plan est aveugle à la dérive. Un pipeline qui ne fait que du
live trouve les problèmes une fois déployés. Les deux répondent à des questions différentes, et
un programme de posture sérieux pose les deux.

## Pour aller plus loin

- [Le modèle d'assessment](assessment-model.fr.md) : pourquoi `not-evaluated` est un résultat à
  part entière.
- [Matrice de couverture](../coverage.fr.md) : par contrôle, par fournisseur, par source.
- [Limites connues](../known-limitations.fr.md) : ce qu'aucune des deux sources ne voit.
- [Périmètre et non-objectifs](scope.fr.md) : ce qu'un rapport Pépin n'est pas.

## Comment cette page reste vraie

Les extraits de plan sont lus dans les fixtures committées sous `examples/`, et les statuts sont
la sortie de scans réels de ces fixtures, capturée par `internal/docgen`. Si la garde de ce
contrôle disparaissait, ou si le plan cessait de marquer l'attribut comme inconnu, cette page
cesserait de correspondre à ce que le binaire produit et `TestGeneratedDocsAreUpToDate`
échouerait.

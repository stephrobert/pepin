> [🇬🇧 English](objectstorage_bucket_public_access.md) · 🇫🇷 Français

<!-- PAGE GÉNÉRÉE : ne pas éditer à la main. Régénérer avec `mise run gen-docs`. -->

# `objectstorage_bucket_public_access`

**Stockage objet exposé publiquement**

[Retour au catalogue](index.fr.md)

| Champ | Valeur |
|---|---|
| Code | `objectstorage_bucket_public_access` |
| Famille | `stockage` |
| Sévérité | `critical` |
| Exigence SCSL (index gelé) | `CLD-STO-1` |
| Type de ressource lu | `object_storage_bucket` |
| Attribut décisif | `acl` / `acl_grants` / `public_via_acl` |
| État | actif |
| Déclaré pour | `exoscale`, `outscale`, `scaleway` |
| Preuves de remédiation | 1 / 3 |

## Le risque

Un bucket de stockage objet est accessible publiquement (ACL au groupe AllUsers, ACL « canned » public-read/-write, ou politique publique).

Cette description vient du référentiel : c'est le texte que le rapport cite, dans la
langue du lecteur.

## Correspondances normatives

Reprises telles quelles du référentiel. L'exigence SCSL provient de l'index **gelé** :
un contrôle se rattache à une exigence existante, jamais à une exigence créée pour lui.
Ces correspondances sont **indicatives** : un rapport Pépin n'est pas une preuve de
qualification.

| Cadre | Références |
|---|---|
| `scsl` | `CLD-STO-1` |
| `cis_controls_v8` | `3.3` |
| `iso_27001_2022` | `A.5.15`, `A.8.3` |
| `iso_27017` | `CLD.9.5.1` |
| `secnumcloud_3_2` | `9.7`, `13.2` |

## Où Pépin sait le mesurer

Une case ✅ signifie que la source produit le type visé, que le contrat du fournisseur
le déclare `verifie` et que l'attribut décisif est projeté. ◐ signifie « Pépin ne peut
pas décider depuis cette source », ∅ « non testable, avec justification », ✗ « non
déclaré, ou type absent de cette source ».

| Fournisseur | Plan Terraform | Collecte live |
|---|:-:|:-:|
| exoscale | ✗ | ✅ |
| outscale | ✗ | ✅ |
| scaleway | ✅ | ✅ |
| kubernetes | sans objet | ✗ |

Chaque case qui n'est pas ✅, **alors que le contrôle est déclaré pour ce fournisseur**,
porte son motif :

| Fournisseur | Source | Statut | Motif |
|---|---|---|---|
| exoscale | terraform | ✗ `unsupported` | cette source ne produit aucune ressource de type « object_storage_bucket » |
| outscale | terraform | ✗ `unsupported` | cette source ne produit aucune ressource de type « object_storage_bucket » |

## Ce que Pépin peut conclure

| Statut | Ce que le statut affirme | Atteignable depuis |
|---|---|---|
| `fail` | un écart a été détecté sur une ressource réelle | exoscale / live · outscale / live · scaleway / terraform · scaleway / live |
| `pass` | la donnée décisive a été collectée, et elle est conforme | exoscale / live · outscale / live · scaleway / terraform · scaleway / live |
| `not-applicable` | le contrat du fournisseur déclare le contrôle non testable, avec sa justification | aucun |
| `not-evaluated` | le contrôle est implémenté, mais la donnée dont il dépend n'a pas été confirmée | aucun |

Un contrôle observable rend tout de même `not-evaluated` sur un inventaire qui ne
contient aucune ressource du type visé : « rien à voir » n'est pas « conforme ».

## Comment enquêter

- Type de ressource normalisé lu par la règle : `object_storage_bucket`
- Attribut dont la décision dépend : `acl` / `acl_grants` / `public_via_acl`
- Sans cet attribut sur une ressource du type visé, le scan rend `not-evaluated` et non `pass` (`internal/assess`, table `requiredAttr`).
- Ce que chaque source projette se lit dans le descripteur : [`providers/exoscale.yaml`](../../providers/exoscale.yaml) · [`providers/outscale.yaml`](../../providers/outscale.yaml) · [`providers/scaleway.yaml`](../../providers/scaleway.yaml)
- La règle qui émet ce code vit dans [`internal/commonrules/rules/`](../../internal/commonrules/rules) : elle est **commune** à tous les fournisseurs, seule la source change.

## Comment corriger

Retirer l'accès public de l'ACL/la politique ; servir le contenu via des URLs signées si un accès externe est requis.

| Fournisseur | Montage déployable |
|---|---|
| exoscale | [`references/remediation/exoscale/objectstorage_bucket_public_access`](../../references/remediation/exoscale/objectstorage_bucket_public_access) |
| outscale | _aucune preuve déposée à ce jour_ |
| scaleway | _aucune preuve déposée à ce jour_ |

Une preuve de remédiation est un module Terraform autonome, **conforme**, qui se déploie
tel quel, ou une note ancrée sur la documentation officielle. Voir
[le guide de remédiation](../guides/remediation.fr.md).

## Comment vérifier la correction

```bash
# depuis un plan Terraform : aucune ressource n'est créée
./pepin scan scaleway --terraform plan.json --format assessment

# depuis l'API du fournisseur : configuration effective
./pepin scan exoscale --live --format assessment
```

Dans la sortie `assessment`, chercher `"control": "objectstorage_bucket_public_access"` : son `status` doit être
`pass`. S'il reste `not-evaluated`, la donnée décisive n'a pas été collectée, et la
correction n'est **pas** démontrée : le tableau des motifs ci-dessus dit pourquoi.

## Voir aussi

- [Le modèle d'assessment](../concepts/assessment-model.fr.md) : ce que chaque statut affirme.
- [Matrice de couverture](../coverage.fr.md) : la même information, tous contrôles confondus.
- [Plan Terraform ou scan live](../concepts/terraform-vs-live.fr.md) : choisir la source.
- [Ajouter un contrôle](../contributing/adding-a-control.fr.md) : la procédure de bout en bout.

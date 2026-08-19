> [🇬🇧 English](objectstorage_bucket_kms_encryption.md) · 🇫🇷 Français

<!-- PAGE GÉNÉRÉE : ne pas éditer à la main. Régénérer avec `mise run gen-docs`. -->

# `objectstorage_bucket_kms_encryption`

**Clé de chiffrement gérée par le client absente (BYOK) sur un bucket sensible**

[Retour au catalogue](index.fr.md)

| Champ | Valeur |
|---|---|
| Code | `objectstorage_bucket_kms_encryption` |
| Famille | `chiffrement` |
| Sévérité | `medium` |
| Exigence SCSL (index gelé) | `CLD-CHF-4` |
| Type de ressource lu | `object_storage_bucket` |
| Attribut décisif | `sse_kms_enabled` |
| État | actif |
| Déclaré pour | `scaleway` |
| Preuves de remédiation | 0 / 1 |

## Le risque

Un bucket de stockage objet portant des données classées sensibles n'utilise pas de clé de chiffrement gérée par le client (SSE-KMS / BYOK) : ses données restent sous le contrôle exclusif du fournisseur. Observable seulement là où l'API expose le chiffrement par défaut du bucket (Scaleway, via Key Manager) ; non applicable chez les fournisseurs sans clé client au niveau objet.

Cette description vient du référentiel : c'est le texte que le rapport cite, dans la
langue du lecteur.

## Correspondances normatives

Reprises telles quelles du référentiel. L'exigence SCSL provient de l'index **gelé** :
un contrôle se rattache à une exigence existante, jamais à une exigence créée pour lui.
Ces correspondances sont **indicatives** : un rapport Pépin n'est pas une preuve de
qualification.

| Cadre | Références |
|---|---|
| `scsl` | `CLD-CHF-4` |
| `iso_27001_2022` | `A.8.24` |
| `secnumcloud_3_2` | `10.1`, `10.5` |

## Où Pépin sait le mesurer

Une case ✅ signifie que la source produit le type visé, que le contrat du fournisseur
le déclare `verifie` et que l'attribut décisif est projeté. ◐ signifie « Pépin ne peut
pas décider depuis cette source », ∅ « non testable, avec justification », ✗ « non
déclaré, ou type absent de cette source ».

| Fournisseur | Plan Terraform | Collecte live |
|---|:-:|:-:|
| exoscale | ∅ | ∅ |
| outscale | ∅ | ∅ |
| scaleway | ◐ | ✅ |
| kubernetes | sans objet | ✗ |

Chaque case qui n'est pas ✅, **alors que le contrôle est déclaré pour ce fournisseur**,
porte son motif :

| Fournisseur | Source | Statut | Motif |
|---|---|---|---|
| exoscale | terraform | ∅ `not-applicable` | SOS chiffre au repos par défaut (SSE-SOS, clés gérées par Exoscale, type SSE-S3) mais n'expose pas de BYOK/KMS géré par le client au niveau bucket (SSE-C reste par-objet, non observable) → le contrôle BYOK-au-bucket est sans objet (CHF-4). |
| exoscale | live | ∅ `not-applicable` | SOS chiffre au repos par défaut (SSE-SOS, clés gérées par Exoscale, type SSE-S3) mais n'expose pas de BYOK/KMS géré par le client au niveau bucket (SSE-C reste par-objet, non observable) → le contrôle BYOK-au-bucket est sans objet (CHF-4). |
| outscale | terraform | ∅ `not-applicable` | OOS chiffre côté serveur en AES256 avec une clé FOURNISSEUR ; il n'existe ni service KMS ni clé maître gérée par le client, donc pas de BYOK à auditer au niveau bucket (CHF-4). NB : l'activation du SSE elle-même est opt-in et observable — elle relève d'un contrôle distinct, pas de ce N/A. |
| outscale | live | ∅ `not-applicable` | OOS chiffre côté serveur en AES256 avec une clé FOURNISSEUR ; il n'existe ni service KMS ni clé maître gérée par le client, donc pas de BYOK à auditer au niveau bucket (CHF-4). NB : l'activation du SSE elle-même est opt-in et observable — elle relève d'un contrôle distinct, pas de ce N/A. |
| scaleway | terraform | ◐ `partial` | attribut décisif « sse_kms_enabled » non projeté par cette source : garde de capacité, le scan rend « not-evaluated » |

## Ce que Pépin peut conclure

| Statut | Ce que le statut affirme | Atteignable depuis |
|---|---|---|
| `fail` | un écart a été détecté sur une ressource réelle | scaleway / terraform · scaleway / live |
| `pass` | la donnée décisive a été collectée, et elle est conforme | scaleway / live |
| `not-applicable` | le contrat du fournisseur déclare le contrôle non testable, avec sa justification | exoscale / terraform · exoscale / live · outscale / terraform · outscale / live |
| `not-evaluated` | le contrôle est implémenté, mais la donnée dont il dépend n'a pas été confirmée | scaleway / terraform |

Un contrôle observable rend tout de même `not-evaluated` sur un inventaire qui ne
contient aucune ressource du type visé : « rien à voir » n'est pas « conforme ».

## Comment enquêter

- Type de ressource normalisé lu par la règle : `object_storage_bucket`
- Attribut dont la décision dépend : `sse_kms_enabled`
- Sans cet attribut sur une ressource du type visé, le scan rend `not-evaluated` et non `pass` (`internal/assess`, table `requiredAttr`).
- Ce que chaque source projette se lit dans le descripteur : [`providers/scaleway.yaml`](../../providers/scaleway.yaml)
- La règle qui émet ce code vit dans [`internal/commonrules/rules/`](../../internal/commonrules/rules) : elle est **commune** à tous les fournisseurs, seule la source change.

## Comment corriger

Créer une clé dans le gestionnaire de clés du fournisseur (Key Manager) et l'associer au bucket comme clé de chiffrement par défaut (SSE-KMS).

| Fournisseur | Montage déployable |
|---|---|
| scaleway | _aucune preuve déposée à ce jour_ |

Une preuve de remédiation est un module Terraform autonome, **conforme**, qui se déploie
tel quel, ou une note ancrée sur la documentation officielle. Voir
[le guide de remédiation](../guides/remediation.md).

## Comment vérifier la correction

```bash
# depuis un plan Terraform : aucune ressource n'est créée
./pepin scan scaleway --terraform plan.json --format assessment

# depuis l'API du fournisseur : configuration effective
./pepin scan scaleway --live --format assessment
```

Dans la sortie `assessment`, chercher `"control": "objectstorage_bucket_kms_encryption"` : son `status` doit être
`pass`. S'il reste `not-evaluated`, la donnée décisive n'a pas été collectée, et la
correction n'est **pas** démontrée : le tableau des motifs ci-dessus dit pourquoi.

**Une des deux sources ne sait pas lever le verrou du « pass »** pour ce contrôle :
le fournisseur cité y produit bien le type visé, mais le scan y rendra `not-evaluated`.
Le tableau des motifs dit laquelle, et pourquoi.

## Voir aussi

- [Le modèle d'assessment](../concepts/assessment-model.md) : ce que chaque statut affirme.
- [Matrice de couverture](../coverage.md) : la même information, tous contrôles confondus.
- [Plan Terraform ou scan live](../concepts/terraform-vs-live.md) : choisir la source.
- [Ajouter un contrôle](../contributing/adding-a-control.md) : la procédure de bout en bout.

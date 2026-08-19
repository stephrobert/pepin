> [🇬🇧 English](iam_user_mfa_enabled.md) · 🇫🇷 Français

<!-- PAGE GÉNÉRÉE : ne pas éditer à la main. Régénérer avec `mise run gen-docs`. -->

# `iam_user_mfa_enabled`

**MFA non activée sur un compte**

[Retour au catalogue](index.fr.md)

| Champ | Valeur |
|---|---|
| Code | `iam_user_mfa_enabled` |
| Famille | `iam` |
| Sévérité | `high` |
| Exigence SCSL (index gelé) | `CLD-IAM-3` |
| Type de ressource lu | `iam_user` |
| Attribut décisif | `mfa_enabled` |
| État | actif |
| Déclaré pour | `exoscale`, `scaleway` |
| Preuves de remédiation | 0 / 2 |

## Le risque

Un compte utilisateur cloud n'a pas l'authentification multifacteur (MFA) activée : un mot de passe compromis suffit à accéder à la console ou à l'API.

Cette description vient du référentiel : c'est le texte que le rapport cite, dans la
langue du lecteur.

## Correspondances normatives

Reprises telles quelles du référentiel. L'exigence SCSL provient de l'index **gelé** :
un contrôle se rattache à une exigence existante, jamais à une exigence créée pour lui.
Ces correspondances sont **indicatives** : un rapport Pépin n'est pas une preuve de
qualification.

| Cadre | Références |
|---|---|
| `scsl` | `CLD-IAM-3` |
| `cis_controls_v8` | `6.5` |
| `iso_27001_2022` | `A.5.17`, `A.8.5` |
| `secnumcloud_3_2` | `9.5`, `9.6` |

## Où Pépin sait le mesurer

Une case ✅ signifie que la source produit le type visé, que le contrat du fournisseur
le déclare `verifie` et que l'attribut décisif est projeté. ◐ signifie « Pépin ne peut
pas décider depuis cette source », ∅ « non testable, avec justification », ✗ « non
déclaré, ou type absent de cette source ».

| Fournisseur | Plan Terraform | Collecte live |
|---|:-:|:-:|
| exoscale | ✗ | ✅ |
| outscale | ∅ | ∅ |
| scaleway | ✗ | ✅ |
| kubernetes | sans objet | ✗ |

Chaque case qui n'est pas ✅, **alors que le contrôle est déclaré pour ce fournisseur**,
porte son motif :

| Fournisseur | Source | Statut | Motif |
|---|---|---|---|
| exoscale | terraform | ✗ `unsupported` | cette source ne produit aucune ressource de type « iam_user » |
| outscale | terraform | ∅ `not-applicable` | type de ressource « iam_user » absent de l'API outscale |
| outscale | live | ∅ `not-applicable` | type de ressource « iam_user » absent de l'API outscale |
| scaleway | terraform | ✗ `unsupported` | cette source ne produit aucune ressource de type « iam_user » |

## Ce que Pépin peut conclure

| Statut | Ce que le statut affirme | Atteignable depuis |
|---|---|---|
| `fail` | un écart a été détecté sur une ressource réelle | exoscale / live · scaleway / live |
| `pass` | la donnée décisive a été collectée, et elle est conforme | exoscale / live · scaleway / live |
| `not-applicable` | le contrat du fournisseur déclare le contrôle non testable, avec sa justification | outscale / terraform · outscale / live |
| `not-evaluated` | le contrôle est implémenté, mais la donnée dont il dépend n'a pas été confirmée | aucun |

Un contrôle observable rend tout de même `not-evaluated` sur un inventaire qui ne
contient aucune ressource du type visé : « rien à voir » n'est pas « conforme ».

## Comment enquêter

- Type de ressource normalisé lu par la règle : `iam_user`
- Attribut dont la décision dépend : `mfa_enabled`
- Sans cet attribut sur une ressource du type visé, le scan rend `not-evaluated` et non `pass` (`internal/assess`, table `requiredAttr`).
- Ce que chaque source projette se lit dans le descripteur : [`providers/exoscale.yaml`](../../providers/exoscale.yaml) · [`providers/scaleway.yaml`](../../providers/scaleway.yaml)
- La règle qui émet ce code vit dans [`internal/commonrules/rules/`](../../internal/commonrules/rules) : elle est **commune** à tous les fournisseurs, seule la source change.

## Comment corriger

Activer la MFA sur tous les comptes, en priorité les accès d'administration et la console ; l'imposer par politique au niveau de l'organisation.

| Fournisseur | Montage déployable |
|---|---|
| exoscale | _aucune preuve déposée à ce jour_ |
| scaleway | _aucune preuve déposée à ce jour_ |

Une preuve de remédiation est un module Terraform autonome, **conforme**, qui se déploie
tel quel, ou une note ancrée sur la documentation officielle. Voir
[le guide de remédiation](../guides/remediation.md).

## Comment vérifier la correction

```bash
# depuis l'API du fournisseur : configuration effective
./pepin scan exoscale --live --format assessment
```

Dans la sortie `assessment`, chercher `"control": "iam_user_mfa_enabled"` : son `status` doit être
`pass`. S'il reste `not-evaluated`, la donnée décisive n'a pas été collectée, et la
correction n'est **pas** démontrée : le tableau des motifs ci-dessus dit pourquoi.

## Voir aussi

- [Le modèle d'assessment](../concepts/assessment-model.md) : ce que chaque statut affirme.
- [Matrice de couverture](../coverage.md) : la même information, tous contrôles confondus.
- [Plan Terraform ou scan live](../concepts/terraform-vs-live.md) : choisir la source.
- [Ajouter un contrôle](../contributing/adding-a-control.md) : la procédure de bout en bout.

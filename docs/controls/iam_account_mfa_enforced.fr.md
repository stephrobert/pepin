> [🇬🇧 English](iam_account_mfa_enforced.md) · 🇫🇷 Français

<!-- PAGE GÉNÉRÉE : ne pas éditer à la main. Régénérer avec `mise run gen-docs`. -->

# `iam_account_mfa_enforced`

**MFA non imposée au niveau du compte**

[Retour au catalogue](index.fr.md)

| Champ | Valeur |
|---|---|
| Code | `iam_account_mfa_enforced` |
| Famille | `iam` |
| Sévérité | `high` |
| Exigence SCSL (index gelé) | `CLD-IAM-3` |
| Type de ressource lu | `api_access_policy` |
| Attribut décisif | `require_trusted_env` |
| État | actif |
| Déclaré pour | `outscale` |
| Preuves de remédiation | 0 / 1 |

## Le risque

La politique d'accès API du compte n'impose pas la MFA à l'ensemble des utilisateurs (environnement de confiance désactivé) : un mot de passe compromis suffit. Angle compte, complémentaire du MFA par utilisateur.

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
| exoscale | ✗ | ✗ |
| outscale | ✗ | ✅ |
| scaleway | ✗ | ✗ |
| kubernetes | sans objet | ✗ |

Chaque case qui n'est pas ✅, **alors que le contrôle est déclaré pour ce fournisseur**,
porte son motif :

| Fournisseur | Source | Statut | Motif |
|---|---|---|---|
| outscale | terraform | ✗ `unsupported` | cette source ne produit aucune ressource de type « api_access_policy » |

## Ce que Pépin peut conclure

| Statut | Ce que le statut affirme | Atteignable depuis |
|---|---|---|
| `fail` | un écart a été détecté sur une ressource réelle | outscale / live |
| `pass` | la donnée décisive a été collectée, et elle est conforme | outscale / live |
| `not-applicable` | le contrat du fournisseur déclare le contrôle non testable, avec sa justification | aucun |
| `not-evaluated` | le contrôle est implémenté, mais la donnée dont il dépend n'a pas été confirmée | aucun |

Un contrôle observable rend tout de même `not-evaluated` sur un inventaire qui ne
contient aucune ressource du type visé : « rien à voir » n'est pas « conforme ».

## Comment enquêter

- Type de ressource normalisé lu par la règle : `api_access_policy`
- Attribut dont la décision dépend : `require_trusted_env`
- Sans cet attribut sur une ressource du type visé, le scan rend `not-evaluated` et non `pass` (`internal/assess`, table `requiredAttr`).
- Ce que chaque source projette se lit dans le descripteur : [`providers/outscale.yaml`](../../providers/outscale.yaml)
- La règle qui émet ce code vit dans [`internal/commonrules/rules/`](../../internal/commonrules/rules) : elle est **commune** à tous les fournisseurs, seule la source change.

## Comment corriger

Activer l'exigence d'environnement de confiance (RequireTrustedEnv) pour imposer la MFA (WebAuthn/OTP) à tous les utilisateurs du compte.

| Fournisseur | Montage déployable |
|---|---|
| outscale | _aucune preuve déposée à ce jour_ |

Une preuve de remédiation est un module Terraform autonome, **conforme**, qui se déploie
tel quel, ou une note ancrée sur la documentation officielle. Voir
[le guide de remédiation](../guides/remediation.fr.md).

## Comment vérifier la correction

```bash
# depuis l'API du fournisseur : configuration effective
./pepin scan outscale --live --format assessment
```

Dans la sortie `assessment`, chercher `"control": "iam_account_mfa_enforced"` : son `status` doit être
`pass`. S'il reste `not-evaluated`, la donnée décisive n'a pas été collectée, et la
correction n'est **pas** démontrée : le tableau des motifs ci-dessus dit pourquoi.

## Voir aussi

- [Le modèle d'assessment](../concepts/assessment-model.fr.md) : ce que chaque statut affirme.
- [Matrice de couverture](../coverage.fr.md) : la même information, tous contrôles confondus.
- [Plan Terraform ou scan live](../concepts/terraform-vs-live.fr.md) : choisir la source.
- [Ajouter un contrôle](../contributing/adding-a-control.fr.md) : la procédure de bout en bout.

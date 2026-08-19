> [🇬🇧 English](iam_role_key_lifetime_bounded.md) · 🇫🇷 Français

<!-- PAGE GÉNÉRÉE : ne pas éditer à la main. Régénérer avec `mise run gen-docs`. -->

# `iam_role_key_lifetime_bounded`

**Rôle IAM sans borne de durée de vie des accès**

[Retour au catalogue](index.fr.md)

| Champ | Valeur |
|---|---|
| Code | `iam_role_key_lifetime_bounded` |
| Famille | `iam` |
| Sévérité | `critical` |
| Exigence SCSL (index gelé) | `CLD-IAM-2` |
| Type de ressource lu | `iam_role` |
| Attribut décisif | `max_session_ttl` / `policy_has_expiration` |
| État | actif |
| Déclaré pour | `exoscale` |
| Preuves de remédiation | 1 / 1 |

## Le risque

La politique d'un rôle IAM ne borne pas la durée de vie des accès (ni TTL de session maximal, ni expiration dans la politique) : une clé assumant ce rôle conserve indéfiniment ses droits, contrairement au principe de secrets à durée limitée.

Cette description vient du référentiel : c'est le texte que le rapport cite, dans la
langue du lecteur.

## Correspondances normatives

Reprises telles quelles du référentiel. L'exigence SCSL provient de l'index **gelé** :
un contrôle se rattache à une exigence existante, jamais à une exigence créée pour lui.
Ces correspondances sont **indicatives** : un rapport Pépin n'est pas une preuve de
qualification.

| Cadre | Références |
|---|---|
| `scsl` | `CLD-IAM-2` |
| `iso_27001_2022` | `A.5.17` |
| `secnumcloud_3_2` | `9.5`, `10.5` |

## Où Pépin sait le mesurer

Une case ✅ signifie que la source produit le type visé, que le contrat du fournisseur
le déclare `verifie` et que l'attribut décisif est projeté. ◐ signifie « Pépin ne peut
pas décider depuis cette source », ∅ « non testable, avec justification », ✗ « non
déclaré, ou type absent de cette source ».

| Fournisseur | Plan Terraform | Collecte live |
|---|:-:|:-:|
| exoscale | ✅ | ✅ |
| outscale | ✗ | ✗ |
| scaleway | ✗ | ✗ |
| kubernetes | sans objet | ✗ |

Chaque case qui n'est pas ✅, **alors que le contrôle est déclaré pour ce fournisseur**,
porte son motif :

_Aucune : toutes les cases déclarées sont pleinement observables._

## Ce que Pépin peut conclure

| Statut | Ce que le statut affirme | Atteignable depuis |
|---|---|---|
| `fail` | un écart a été détecté sur une ressource réelle | exoscale / terraform · exoscale / live |
| `pass` | la donnée décisive a été collectée, et elle est conforme | exoscale / terraform · exoscale / live |
| `not-applicable` | le contrat du fournisseur déclare le contrôle non testable, avec sa justification | aucun |
| `not-evaluated` | le contrôle est implémenté, mais la donnée dont il dépend n'a pas été confirmée | aucun |

Un contrôle observable rend tout de même `not-evaluated` sur un inventaire qui ne
contient aucune ressource du type visé : « rien à voir » n'est pas « conforme ».

## Comment enquêter

- Type de ressource normalisé lu par la règle : `iam_role`
- Attribut dont la décision dépend : `max_session_ttl` / `policy_has_expiration`
- Sans cet attribut sur une ressource du type visé, le scan rend `not-evaluated` et non `pass` (`internal/assess`, table `requiredAttr`).
- Ce que chaque source projette se lit dans le descripteur : [`providers/exoscale.yaml`](../../providers/exoscale.yaml)
- La règle qui émet ce code vit dans [`internal/commonrules/rules/`](../../internal/commonrules/rules) : elle est **commune** à tous les fournisseurs, seule la source change.

## Comment corriger

Borner la durée de vie : définir un TTL de session maximal sur le rôle ou une condition d'expiration dans sa politique (au-delà d'une durée, les accès sont refusés).

| Fournisseur | Montage déployable |
|---|---|
| exoscale | [`references/remediation/exoscale/iam_role_key_lifetime_bounded`](../../references/remediation/exoscale/iam_role_key_lifetime_bounded) |

Une preuve de remédiation est un module Terraform autonome, **conforme**, qui se déploie
tel quel, ou une note ancrée sur la documentation officielle. Voir
[le guide de remédiation](../guides/remediation.fr.md).

## Comment vérifier la correction

```bash
# depuis un plan Terraform : aucune ressource n'est créée
./pepin scan exoscale --terraform plan.json --format assessment

# depuis l'API du fournisseur : configuration effective
./pepin scan exoscale --live --format assessment
```

Dans la sortie `assessment`, chercher `"control": "iam_role_key_lifetime_bounded"` : son `status` doit être
`pass`. S'il reste `not-evaluated`, la donnée décisive n'a pas été collectée, et la
correction n'est **pas** démontrée : le tableau des motifs ci-dessus dit pourquoi.

## Voir aussi

- [Le modèle d'assessment](../concepts/assessment-model.fr.md) : ce que chaque statut affirme.
- [Matrice de couverture](../coverage.fr.md) : la même information, tous contrôles confondus.
- [Plan Terraform ou scan live](../concepts/terraform-vs-live.fr.md) : choisir la source.
- [Ajouter un contrôle](../contributing/adding-a-control.fr.md) : la procédure de bout en bout.

> [🇬🇧 English](network_flow_matrix_documented.md) · 🇫🇷 Français

<!-- PAGE GÉNÉRÉE : ne pas éditer à la main. Régénérer avec `mise run gen-docs`. -->

# `network_flow_matrix_documented`

**Flux entrant sans justification dans la matrice des flux**

[Retour au catalogue](index.fr.md)

| Champ | Valeur |
|---|---|
| Code | `network_flow_matrix_documented` |
| Famille | `reseau` |
| Sévérité | `medium` |
| Exigence SCSL (index gelé) | `CLD-NET-5` |
| Type de ressource lu | `security_group_rule` |
| Attribut décisif | `description` |
| État | actif |
| Déclaré pour | `exoscale` |
| Preuves de remédiation | 0 / 1 |

## Le risque

Une règle de flux entrant n'a pas de justification (description) : la matrice des flux autorisés (services, protocoles, ports + justification) n'est pas tenue, ce qui empêche la revue et l'imputabilité des ouvertures réseau.

Cette description vient du référentiel : c'est le texte que le rapport cite, dans la
langue du lecteur.

## Correspondances normatives

Reprises telles quelles du référentiel. L'exigence SCSL provient de l'index **gelé** :
un contrôle se rattache à une exigence existante, jamais à une exigence créée pour lui.
Ces correspondances sont **indicatives** : un rapport Pépin n'est pas une preuve de
qualification.

| Cadre | Références |
|---|---|
| `scsl` | `CLD-NET-5` |
| `iso_27001_2022` | `A.8.20`, `A.8.22` |
| `secnumcloud_3_2` | `13.1`, `13.2` |

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

- Type de ressource normalisé lu par la règle : `security_group_rule`
- Attribut dont la décision dépend : `description`
- Sans cet attribut sur une ressource du type visé, le scan rend `not-evaluated` et non `pass` (`internal/assess`, table `requiredAttr`).
- Ce que chaque source projette se lit dans le descripteur : [`providers/exoscale.yaml`](../../providers/exoscale.yaml)
- La règle qui émet ce code vit dans [`internal/commonrules/rules/`](../../internal/commonrules/rules) : elle est **commune** à tous les fournisseurs, seule la source change.

## Comment corriger

Documenter chaque règle de security group (service desservi, raison) via sa description ; tenir à jour la matrice des flux et la réviser périodiquement.

| Fournisseur | Montage déployable |
|---|---|
| exoscale | _aucune preuve déposée à ce jour_ |

Une preuve de remédiation est un module Terraform autonome, **conforme**, qui se déploie
tel quel, ou une note ancrée sur la documentation officielle. Voir
[le guide de remédiation](../guides/remediation.md).

## Comment vérifier la correction

```bash
# depuis un plan Terraform : aucune ressource n'est créée
./pepin scan exoscale --terraform plan.json --format assessment

# depuis l'API du fournisseur : configuration effective
./pepin scan exoscale --live --format assessment
```

Dans la sortie `assessment`, chercher `"control": "network_flow_matrix_documented"` : son `status` doit être
`pass`. S'il reste `not-evaluated`, la donnée décisive n'a pas été collectée, et la
correction n'est **pas** démontrée : le tableau des motifs ci-dessus dit pourquoi.

## Voir aussi

- [Le modèle d'assessment](../concepts/assessment-model.md) : ce que chaque statut affirme.
- [Matrice de couverture](../coverage.md) : la même information, tous contrôles confondus.
- [Plan Terraform ou scan live](../concepts/terraform-vs-live.md) : choisir la source.
- [Ajouter un contrôle](../contributing/adding-a-control.md) : la procédure de bout en bout.

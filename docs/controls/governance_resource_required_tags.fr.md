> [🇬🇧 English](governance_resource_required_tags.md) · 🇫🇷 Français

<!-- PAGE GÉNÉRÉE : ne pas éditer à la main. Régénérer avec `mise run gen-docs`. -->

# `governance_resource_required_tags`

**Inventaire et étiquetage incomplets**

[Retour au catalogue](index.fr.md)

| Champ | Valeur |
|---|---|
| Code | `governance_resource_required_tags` |
| Famille | `gouvernance` |
| Sévérité | `medium` |
| Exigence SCSL (index gelé) | `CLD-GVN-1` |
| Type de ressource lu | _aucun : contrôle transverse_ |
| Attribut décisif | _aucun : jugé à la présence d'un écart_ |
| État | actif |
| Déclaré pour | `exoscale`, `outscale`, `scaleway` |
| Preuves de remédiation | 0 / 3 |

## Le risque

Des ressources n'ont pas les étiquettes de gouvernance (propriétaire, projet, environnement) : l'inventaire et la responsabilité ne sont pas tenus.

Cette description vient du référentiel : c'est le texte que le rapport cite, dans la
langue du lecteur.

## Correspondances normatives

Reprises telles quelles du référentiel. L'exigence SCSL provient de l'index **gelé** :
un contrôle se rattache à une exigence existante, jamais à une exigence créée pour lui.
Ces correspondances sont **indicatives** : un rapport Pépin n'est pas une preuve de
qualification.

| Cadre | Références |
|---|---|
| `scsl` | `CLD-GVN-1` |
| `cis_controls_v8` | `1.1` |
| `iso_27001_2022` | `A.5.9`, `A.8.9` |
| `secnumcloud_3_2` | `8.1` |

## Où Pépin sait le mesurer

Une case ✅ signifie que la source produit le type visé, que le contrat du fournisseur
le déclare `verifie` et que l'attribut décisif est projeté. ◐ signifie « Pépin ne peut
pas décider depuis cette source », ∅ « non testable, avec justification », ✗ « non
déclaré, ou type absent de cette source ».

| Fournisseur | Plan Terraform | Collecte live |
|---|:-:|:-:|
| exoscale | ◐ | ◐ |
| outscale | ◐ | ◐ |
| scaleway | ◐ | ◐ |
| kubernetes | sans objet | ✗ |

Chaque case qui n'est pas ✅, **alors que le contrôle est déclaré pour ce fournisseur**,
porte son motif :

| Fournisseur | Source | Statut | Motif |
|---|---|---|---|
| exoscale | terraform | ◐ `partial` | aucun type de ressource visé et le contrôle ne lit pas le descripteur du fournisseur : le verrou du « pass » ne peut pas être levé, le scan rend « not-evaluated » tant qu'aucun écart n'est détecté |
| exoscale | live | ◐ `partial` | aucun type de ressource visé et le contrôle ne lit pas le descripteur du fournisseur : le verrou du « pass » ne peut pas être levé, le scan rend « not-evaluated » tant qu'aucun écart n'est détecté |
| outscale | terraform | ◐ `partial` | aucun type de ressource visé et le contrôle ne lit pas le descripteur du fournisseur : le verrou du « pass » ne peut pas être levé, le scan rend « not-evaluated » tant qu'aucun écart n'est détecté |
| outscale | live | ◐ `partial` | aucun type de ressource visé et le contrôle ne lit pas le descripteur du fournisseur : le verrou du « pass » ne peut pas être levé, le scan rend « not-evaluated » tant qu'aucun écart n'est détecté |
| scaleway | terraform | ◐ `partial` | aucun type de ressource visé et le contrôle ne lit pas le descripteur du fournisseur : le verrou du « pass » ne peut pas être levé, le scan rend « not-evaluated » tant qu'aucun écart n'est détecté |
| scaleway | live | ◐ `partial` | aucun type de ressource visé et le contrôle ne lit pas le descripteur du fournisseur : le verrou du « pass » ne peut pas être levé, le scan rend « not-evaluated » tant qu'aucun écart n'est détecté |

## Ce que Pépin peut conclure

| Statut | Ce que le statut affirme | Atteignable depuis |
|---|---|---|
| `fail` | un écart a été détecté sur une ressource réelle | exoscale / terraform · exoscale / live · outscale / terraform · outscale / live · scaleway / terraform · scaleway / live |
| `pass` | la donnée décisive a été collectée, et elle est conforme | aucun |
| `not-applicable` | le contrat du fournisseur déclare le contrôle non testable, avec sa justification | aucun |
| `not-evaluated` | le contrôle est implémenté, mais la donnée dont il dépend n'a pas été confirmée | exoscale / terraform · exoscale / live · outscale / terraform · outscale / live · scaleway / terraform · scaleway / live |

Un contrôle observable rend tout de même `not-evaluated` sur un inventaire qui ne
contient aucune ressource du type visé : « rien à voir » n'est pas « conforme ».

## Comment enquêter

- Contrôle transverse : il ne lit pas un type de ressource particulier.
- Aucun verrou d'attribut : le contrôle se juge à la présence d'un écart, l'absence de mauvaise configuration valant conformité.
- Ce que chaque source projette se lit dans le descripteur : [`providers/exoscale.yaml`](../../providers/exoscale.yaml) · [`providers/outscale.yaml`](../../providers/outscale.yaml) · [`providers/scaleway.yaml`](../../providers/scaleway.yaml)
- La règle qui émet ce code vit dans [`internal/commonrules/rules/`](../../internal/commonrules/rules) : elle est **commune** à tous les fournisseurs, seule la source change.

## Comment corriger

Imposer des étiquettes obligatoires (propriétaire, projet, environnement) ; contrôler leur présence à la création.

| Fournisseur | Montage déployable |
|---|---|
| exoscale | _aucune preuve déposée à ce jour_ |
| outscale | _aucune preuve déposée à ce jour_ |
| scaleway | _aucune preuve déposée à ce jour_ |

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

Dans la sortie `assessment`, chercher `"control": "governance_resource_required_tags"` : son `status` doit être
`pass`. S'il reste `not-evaluated`, la donnée décisive n'a pas été collectée, et la
correction n'est **pas** démontrée : le tableau des motifs ci-dessus dit pourquoi.

**Une des deux sources ne sait pas lever le verrou du « pass »** pour ce contrôle :
le fournisseur cité y produit bien le type visé, mais le scan y rendra `not-evaluated`.
Le tableau des motifs dit laquelle, et pourquoi.

## Voir aussi

- [Le modèle d'assessment](../concepts/assessment-model.fr.md) : ce que chaque statut affirme.
- [Matrice de couverture](../coverage.fr.md) : la même information, tous contrôles confondus.
- [Plan Terraform ou scan live](../concepts/terraform-vs-live.fr.md) : choisir la source.
- [Ajouter un contrôle](../contributing/adding-a-control.fr.md) : la procédure de bout en bout.

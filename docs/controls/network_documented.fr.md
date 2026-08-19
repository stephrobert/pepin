> [🇬🇧 English](network_documented.md) · 🇫🇷 Français

<!-- PAGE GÉNÉRÉE : ne pas éditer à la main. Régénérer avec `mise run gen-docs`. -->

# `network_documented`

**Réseau non documenté (cartographie non tenue)**

[Retour au catalogue](index.fr.md)

| Champ | Valeur |
|---|---|
| Code | `network_documented` |
| Famille | `reseau` |
| Sévérité | `low` |
| Exigence SCSL (index gelé) | `CLD-NET-5` |
| Type de ressource lu | `network` |
| Attribut décisif | _aucun : jugé à la présence d'un écart_ |
| État | actif |
| Déclaré pour | `exoscale`, `outscale`, `scaleway` |
| Preuves de remédiation | 1 / 3 |

## Le risque

Un réseau (VPC / Net / réseau privé) n'a pas de nom ou d'étiquettes de gouvernance : la cartographie réseau (inventaire, propriétaire, projet) n'est pas tenue à jour, ce qui nuit à la revue et à la réponse aux incidents.

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
| `iso_27001_2022` | `A.5.9` |
| `secnumcloud_3_2` | `13.1` |

## Où Pépin sait le mesurer

Une case ✅ signifie que la source produit le type visé, que le contrat du fournisseur
le déclare `verifie` et que l'attribut décisif est projeté. ◐ signifie « Pépin ne peut
pas décider depuis cette source », ∅ « non testable, avec justification », ✗ « non
déclaré, ou type absent de cette source ».

| Fournisseur | Plan Terraform | Collecte live |
|---|:-:|:-:|
| exoscale | ✅ | ✅ |
| outscale | ✅ | ✅ |
| scaleway | ◐ | ✗ |
| kubernetes | sans objet | ✗ |

Chaque case qui n'est pas ✅, **alors que le contrôle est déclaré pour ce fournisseur**,
porte son motif :

| Fournisseur | Source | Statut | Motif |
|---|---|---|---|
| scaleway | terraform | ◐ `partial` | contrat du fournisseur : le type « network » n'est pas déclaré `verifie` (état : a_verifier) |
| scaleway | live | ✗ `unsupported` | cette source ne produit aucune ressource de type « network » |

## Ce que Pépin peut conclure

| Statut | Ce que le statut affirme | Atteignable depuis |
|---|---|---|
| `fail` | un écart a été détecté sur une ressource réelle | exoscale / terraform · exoscale / live · outscale / terraform · outscale / live · scaleway / terraform |
| `pass` | la donnée décisive a été collectée, et elle est conforme | exoscale / terraform · exoscale / live · outscale / terraform · outscale / live |
| `not-applicable` | le contrat du fournisseur déclare le contrôle non testable, avec sa justification | aucun |
| `not-evaluated` | le contrôle est implémenté, mais la donnée dont il dépend n'a pas été confirmée | scaleway / terraform |

Un contrôle observable rend tout de même `not-evaluated` sur un inventaire qui ne
contient aucune ressource du type visé : « rien à voir » n'est pas « conforme ».

## Comment enquêter

- Type de ressource normalisé lu par la règle : `network`
- Aucun verrou d'attribut : le contrôle se juge à la présence d'un écart, l'absence de mauvaise configuration valant conformité.
- Ce que chaque source projette se lit dans le descripteur : [`providers/exoscale.yaml`](../../providers/exoscale.yaml) · [`providers/outscale.yaml`](../../providers/outscale.yaml) · [`providers/scaleway.yaml`](../../providers/scaleway.yaml)
- La règle qui émet ce code vit dans [`internal/commonrules/rules/`](../../internal/commonrules/rules) : elle est **commune** à tous les fournisseurs, seule la source change.

## Comment corriger

Nommer et étiqueter chaque réseau (propriétaire, projet, environnement) ; tenir à jour la cartographie réseau et la réviser périodiquement.

| Fournisseur | Montage déployable |
|---|---|
| exoscale | [`references/remediation/exoscale/network_documented`](../../references/remediation/exoscale/network_documented) |
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

Dans la sortie `assessment`, chercher `"control": "network_documented"` : son `status` doit être
`pass`. S'il reste `not-evaluated`, la donnée décisive n'a pas été collectée, et la
correction n'est **pas** démontrée : le tableau des motifs ci-dessus dit pourquoi.

## Voir aussi

- [Le modèle d'assessment](../concepts/assessment-model.fr.md) : ce que chaque statut affirme.
- [Matrice de couverture](../coverage.fr.md) : la même information, tous contrôles confondus.
- [Plan Terraform ou scan live](../concepts/terraform-vs-live.fr.md) : choisir la source.
- [Ajouter un contrôle](../contributing/adding-a-control.fr.md) : la procédure de bout en bout.

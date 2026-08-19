> [🇬🇧 English](compute_instance_public_ip_with_open_securitygroup.md) · 🇫🇷 Français

<!-- PAGE GÉNÉRÉE : ne pas éditer à la main. Régénérer avec `mise run gen-docs`. -->

# `compute_instance_public_ip_with_open_securitygroup`

**Machine exposée publiquement sans filtrage restrictif**

[Retour au catalogue](index.fr.md)

| Champ | Valeur |
|---|---|
| Code | `compute_instance_public_ip_with_open_securitygroup` |
| Famille | `reseau` |
| Sévérité | `critical` |
| Exigence SCSL (index gelé) | `CLD-NET-3` |
| Type de ressource lu | `compute_instance` |
| Attribut décisif | `public_ip` |
| État | actif |
| Déclaré pour | `exoscale`, `outscale`, `scaleway` |
| Preuves de remédiation | 0 / 3 |

## Le risque

Une instance porte une IP publique et un groupe de sécurité ouvert à 0.0.0.0/0 : elle est directement atteignable depuis Internet.

Cette description vient du référentiel : c'est le texte que le rapport cite, dans la
langue du lecteur.

## Correspondances normatives

Reprises telles quelles du référentiel. L'exigence SCSL provient de l'index **gelé** :
un contrôle se rattache à une exigence existante, jamais à une exigence créée pour lui.
Ces correspondances sont **indicatives** : un rapport Pépin n'est pas une preuve de
qualification.

| Cadre | Références |
|---|---|
| `scsl` | `CLD-NET-3` |
| `cis_controls_v8` | `4.4`, `12.2` |
| `iso_27001_2022` | `A.8.20`, `A.8.22` |
| `iso_27017` | `CLD.9.5.2` |
| `secnumcloud_3_2` | `13.2` |

## Où Pépin sait le mesurer

Une case ✅ signifie que la source produit le type visé, que le contrat du fournisseur
le déclare `verifie` et que l'attribut décisif est projeté. ◐ signifie « Pépin ne peut
pas décider depuis cette source », ∅ « non testable, avec justification », ✗ « non
déclaré, ou type absent de cette source ».

| Fournisseur | Plan Terraform | Collecte live |
|---|:-:|:-:|
| exoscale | ✅ | ✅ |
| outscale | ✅ | ✅ |
| scaleway | ◐ | ✅ |
| kubernetes | sans objet | ✗ |

Chaque case qui n'est pas ✅, **alors que le contrôle est déclaré pour ce fournisseur**,
porte son motif :

| Fournisseur | Source | Statut | Motif |
|---|---|---|---|
| scaleway | terraform | ◐ `partial` | attribut décisif « public_ip » non projeté par cette source : garde de capacité, le scan rend « not-evaluated » |

## Ce que Pépin peut conclure

| Statut | Ce que le statut affirme | Atteignable depuis |
|---|---|---|
| `fail` | un écart a été détecté sur une ressource réelle | exoscale / terraform · exoscale / live · outscale / terraform · outscale / live · scaleway / terraform · scaleway / live |
| `pass` | la donnée décisive a été collectée, et elle est conforme | exoscale / terraform · exoscale / live · outscale / terraform · outscale / live · scaleway / live |
| `not-applicable` | le contrat du fournisseur déclare le contrôle non testable, avec sa justification | aucun |
| `not-evaluated` | le contrôle est implémenté, mais la donnée dont il dépend n'a pas été confirmée | scaleway / terraform |

Un contrôle observable rend tout de même `not-evaluated` sur un inventaire qui ne
contient aucune ressource du type visé : « rien à voir » n'est pas « conforme ».

## Comment enquêter

- Type de ressource normalisé lu par la règle : `compute_instance`
- Attribut dont la décision dépend : `public_ip`
- Sans cet attribut sur une ressource du type visé, le scan rend `not-evaluated` et non `pass` (`internal/assess`, table `requiredAttr`).
- Ce que chaque source projette se lit dans le descripteur : [`providers/exoscale.yaml`](../../providers/exoscale.yaml) · [`providers/outscale.yaml`](../../providers/outscale.yaml) · [`providers/scaleway.yaml`](../../providers/scaleway.yaml)
- La règle qui émet ce code vit dans [`internal/commonrules/rules/`](../../internal/commonrules/rules) : elle est **commune** à tous les fournisseurs, seule la source change.

## Comment corriger

Retirer l'IP publique si inutile ; restreindre le groupe de sécurité ; placer la machine derrière un répartiteur ou un VPN.

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

Dans la sortie `assessment`, chercher `"control": "compute_instance_public_ip_with_open_securitygroup"` : son `status` doit être
`pass`. S'il reste `not-evaluated`, la donnée décisive n'a pas été collectée, et la
correction n'est **pas** démontrée : le tableau des motifs ci-dessus dit pourquoi.

## Voir aussi

- [Le modèle d'assessment](../concepts/assessment-model.fr.md) : ce que chaque statut affirme.
- [Matrice de couverture](../coverage.fr.md) : la même information, tous contrôles confondus.
- [Plan Terraform ou scan live](../concepts/terraform-vs-live.fr.md) : choisir la source.
- [Ajouter un contrôle](../contributing/adding-a-control.fr.md) : la procédure de bout en bout.

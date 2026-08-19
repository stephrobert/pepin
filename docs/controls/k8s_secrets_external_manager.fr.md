> [🇬🇧 English](k8s_secrets_external_manager.md) · 🇫🇷 Français

<!-- PAGE GÉNÉRÉE : ne pas éditer à la main. Régénérer avec `mise run gen-docs`. -->

# `k8s_secrets_external_manager`

**Secrets Kubernetes sans coffre externe**

[Retour au catalogue](index.fr.md)

| Champ | Valeur |
|---|---|
| Code | `k8s_secrets_external_manager` |
| Famille | `chiffrement` |
| Sévérité | `high` |
| Exigence SCSL (index gelé) | `CLD-K8S-10` |
| Type de ressource lu | `k8s_crd` |
| Attribut décisif | _aucun : jugé à la présence d'un écart_ |
| État | actif |
| Déclaré pour | `kubernetes` |
| Preuves de remédiation | 0 / 1 |

## Le risque

Aucun gestionnaire de secrets externes n'est déployé dans le cluster : les secrets reposent sur le stockage natif de Kubernetes (encodés en base64 dans etcd), sans coffre, sans rotation et exposés à quiconque peut les lire.

Cette description vient du référentiel : c'est le texte que le rapport cite, dans la
langue du lecteur.

## Correspondances normatives

Reprises telles quelles du référentiel. L'exigence SCSL provient de l'index **gelé** :
un contrôle se rattache à une exigence existante, jamais à une exigence créée pour lui.
Ces correspondances sont **indicatives** : un rapport Pépin n'est pas une preuve de
qualification.

| Cadre | Références |
|---|---|
| `scsl` | `CLD-K8S-10` |
| `iso_27001_2022` | `A.8.24` |
| `secnumcloud_3_2` | `10.5` |

## Où Pépin sait le mesurer

Une case ✅ signifie que la source produit le type visé, que le contrat du fournisseur
le déclare `verifie` et que l'attribut décisif est projeté. ◐ signifie « Pépin ne peut
pas décider depuis cette source », ∅ « non testable, avec justification », ✗ « non
déclaré, ou type absent de cette source ».

| Fournisseur | Plan Terraform | Collecte live |
|---|:-:|:-:|
| exoscale | ✗ | ✗ |
| outscale | ✗ | ✗ |
| scaleway | ✗ | ✗ |
| kubernetes | sans objet | ✅ |

Chaque case qui n'est pas ✅, **alors que le contrôle est déclaré pour ce fournisseur**,
porte son motif :

| Fournisseur | Source | Statut | Motif |
|---|---|---|---|
| kubernetes | terraform | ✗ `unsupported` | cette source ne produit aucune ressource de type « k8s_crd » |

## Ce que Pépin peut conclure

| Statut | Ce que le statut affirme | Atteignable depuis |
|---|---|---|
| `fail` | un écart a été détecté sur une ressource réelle | kubernetes / live |
| `pass` | la donnée décisive a été collectée, et elle est conforme | kubernetes / live |
| `not-applicable` | le contrat du fournisseur déclare le contrôle non testable, avec sa justification | aucun |
| `not-evaluated` | le contrôle est implémenté, mais la donnée dont il dépend n'a pas été confirmée | aucun |

Un contrôle observable rend tout de même `not-evaluated` sur un inventaire qui ne
contient aucune ressource du type visé : « rien à voir » n'est pas « conforme ».

## Comment enquêter

- Type de ressource normalisé lu par la règle : `k8s_crd`
- Aucun verrou d'attribut : le contrôle se juge à la présence d'un écart, l'absence de mauvaise configuration valant conformité.
- Ce que chaque source projette se lit dans le descripteur : [`providers/kubernetes.yaml`](../../providers/kubernetes.yaml)
- La règle qui émet ce code vit dans [`internal/commonrules/rules/`](../../internal/commonrules/rules) : elle est **commune** à tous les fournisseurs, seule la source change.

## Comment corriger

Déployer un gestionnaire de secrets externes (External Secrets Operator ou Secrets Store CSI) et monter les secrets depuis le coffre ; ne jamais placer de secret en clair dans un manifest ou une variable d'environnement.

| Fournisseur | Montage déployable |
|---|---|
| kubernetes | _aucune preuve déposée à ce jour_ |

Une preuve de remédiation est un module Terraform autonome, **conforme**, qui se déploie
tel quel, ou une note ancrée sur la documentation officielle. Voir
[le guide de remédiation](../guides/remediation.md).

## Comment vérifier la correction

```bash
# depuis l'API du fournisseur : configuration effective
./pepin scan kubernetes --live --format assessment
```

Dans la sortie `assessment`, chercher `"control": "k8s_secrets_external_manager"` : son `status` doit être
`pass`. S'il reste `not-evaluated`, la donnée décisive n'a pas été collectée, et la
correction n'est **pas** démontrée : le tableau des motifs ci-dessus dit pourquoi.

## Voir aussi

- [Le modèle d'assessment](../concepts/assessment-model.md) : ce que chaque statut affirme.
- [Matrice de couverture](../coverage.md) : la même information, tous contrôles confondus.
- [Plan Terraform ou scan live](../concepts/terraform-vs-live.md) : choisir la source.
- [Ajouter un contrôle](../contributing/adding-a-control.md) : la procédure de bout en bout.

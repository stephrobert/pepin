> [🇬🇧 English](loadbalancer_ssl_listeners.md) · 🇫🇷 Français

<!-- PAGE GÉNÉRÉE : ne pas éditer à la main. Régénérer avec `mise run gen-docs`. -->

# `loadbalancer_ssl_listeners`

**Chiffrement en transit absent**

[Retour au catalogue](index.fr.md)

| Champ | Valeur |
|---|---|
| Code | `loadbalancer_ssl_listeners` |
| Famille | `chiffrement` |
| Sévérité | `high` |
| Exigence SCSL (index gelé) | `CLD-CHF-1` |
| Type de ressource lu | `load_balancer` |
| Attribut décisif | `load_balancer_type` |
| État | actif |
| Déclaré pour | `outscale` |
| Preuves de remédiation | 0 / 1 |

## Le risque

Un répartiteur de charge exposé n'impose pas de listener TLS/HTTPS : le trafic transite en clair.

Cette description vient du référentiel : c'est le texte que le rapport cite, dans la
langue du lecteur.

## Correspondances normatives

Reprises telles quelles du référentiel. L'exigence SCSL provient de l'index **gelé** :
un contrôle se rattache à une exigence existante, jamais à une exigence créée pour lui.
Ces correspondances sont **indicatives** : un rapport Pépin n'est pas une preuve de
qualification.

| Cadre | Références |
|---|---|
| `scsl` | `CLD-CHF-1` |
| `cis_controls_v8` | `3.10`, `16.11` |
| `iso_27001_2022` | `A.8.24`, `A.8.21` |
| `secnumcloud_3_2` | `10.2` |

## Où Pépin sait le mesurer

Une case ✅ signifie que la source produit le type visé, que le contrat du fournisseur
le déclare `verifie` et que l'attribut décisif est projeté. ◐ signifie « Pépin ne peut
pas décider depuis cette source », ∅ « non testable, avec justification », ✗ « non
déclaré, ou type absent de cette source ».

| Fournisseur | Plan Terraform | Collecte live |
|---|:-:|:-:|
| exoscale | ∅ | ∅ |
| outscale | ◐ | ✅ |
| scaleway | ✗ | ✗ |
| kubernetes | sans objet | ✗ |

Chaque case qui n'est pas ✅, **alors que le contrôle est déclaré pour ce fournisseur**,
porte son motif :

| Fournisseur | Source | Statut | Motif |
|---|---|---|---|
| exoscale | terraform | ∅ `not-applicable` | type de ressource « load_balancer » absent de l'API exoscale |
| exoscale | live | ∅ `not-applicable` | type de ressource « load_balancer » absent de l'API exoscale |
| outscale | terraform | ◐ `partial` | attribut décisif « load_balancer_type » déclaré par le mapping mais ABSENT des plans de référence : soit la valeur n'existe qu'après `apply`, soit l'argument est optionnel et le HCL réel ne l'écrit pas — garde de capacité, le scan rend « not-evaluated » |

## Ce que Pépin peut conclure

| Statut | Ce que le statut affirme | Atteignable depuis |
|---|---|---|
| `fail` | un écart a été détecté sur une ressource réelle | outscale / terraform · outscale / live |
| `pass` | la donnée décisive a été collectée, et elle est conforme | outscale / live |
| `not-applicable` | le contrat du fournisseur déclare le contrôle non testable, avec sa justification | exoscale / terraform · exoscale / live |
| `not-evaluated` | le contrôle est implémenté, mais la donnée dont il dépend n'a pas été confirmée | outscale / terraform |

Un contrôle observable rend tout de même `not-evaluated` sur un inventaire qui ne
contient aucune ressource du type visé : « rien à voir » n'est pas « conforme ».

## Comment enquêter

- Type de ressource normalisé lu par la règle : `load_balancer`
- Attribut dont la décision dépend : `load_balancer_type`
- Sans cet attribut sur une ressource du type visé, le scan rend `not-evaluated` et non `pass` (`internal/assess`, table `requiredAttr`).
- Ce que chaque source projette se lit dans le descripteur : [`providers/outscale.yaml`](../../providers/outscale.yaml)
- La règle qui émet ce code vit dans [`internal/commonrules/rules/`](../../internal/commonrules/rules) : elle est **commune** à tous les fournisseurs, seule la source change.

## Comment corriger

Configurer un listener HTTPS/SSL (TLS ≥ 1.2) ; rediriger le trafic en clair vers HTTPS.

| Fournisseur | Montage déployable |
|---|---|
| outscale | _aucune preuve déposée à ce jour_ |

Une preuve de remédiation est un module Terraform autonome, **conforme**, qui se déploie
tel quel, ou une note ancrée sur la documentation officielle. Voir
[le guide de remédiation](../guides/remediation.fr.md).

## Comment vérifier la correction

```bash
# depuis un plan Terraform : aucune ressource n'est créée
./pepin scan outscale --terraform plan.json --format assessment

# depuis l'API du fournisseur : configuration effective
./pepin scan outscale --live --format assessment
```

Dans la sortie `assessment`, chercher `"control": "loadbalancer_ssl_listeners"` : son `status` doit être
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

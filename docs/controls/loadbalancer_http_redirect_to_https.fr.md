> [🇬🇧 English](loadbalancer_http_redirect_to_https.md) · 🇫🇷 Français

<!-- PAGE GÉNÉRÉE : ne pas éditer à la main. Régénérer avec `mise run gen-docs`. -->

# `loadbalancer_http_redirect_to_https`

**Listener HTTP sans redirection HTTPS**

[Retour au catalogue](index.fr.md)

| Champ | Valeur |
|---|---|
| Code | `loadbalancer_http_redirect_to_https` |
| Famille | `chiffrement` |
| Sévérité | `medium` |
| Exigence SCSL (index gelé) | `CLD-CHF-1` |
| Type de ressource lu | `load_balancer` |
| Attribut décisif | `redirect_to_https` |
| État | dormant (déclaré pour aucun fournisseur) |
| Déclaré pour | aucun |
| Preuves de remédiation | 0 / 0 |

## Le risque

Un répartiteur de charge expose un listener HTTP (port 80) : sans redirection vers HTTPS, le trafic peut transiter en clair.

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
| `cis_controls_v8` | `3.10` |
| `iso_27001_2022` | `A.8.24` |
| `secnumcloud_3_2` | `10.2` |

## Où Pépin sait le mesurer

Une case ✅ signifie que la source produit le type visé, que le contrat du fournisseur
le déclare `verifie` et que l'attribut décisif est projeté. ◐ signifie « Pépin ne peut
pas décider depuis cette source », ∅ « non testable, avec justification », ✗ « non
déclaré, ou type absent de cette source ».

| Fournisseur | Plan Terraform | Collecte live |
|---|:-:|:-:|
| exoscale | ∅ | ∅ |
| outscale | ∅ | ∅ |
| scaleway | ✗ | ✗ |
| kubernetes | sans objet | ✗ |

Chaque case qui n'est pas ✅, **alors que le contrôle est déclaré pour ce fournisseur**,
porte son motif :

| Fournisseur | Source | Statut | Motif |
|---|---|---|---|
| exoscale | terraform | ∅ `not-applicable` | type de ressource « load_balancer » absent de l'API exoscale |
| exoscale | live | ∅ `not-applicable` | type de ressource « load_balancer » absent de l'API exoscale |
| outscale | terraform | ∅ `not-applicable` | Le LBU Outscale ne peut pas rediriger : `ListenerRule.Action` est documenté « always forward » au contrat OAPI (aucune action de redirection), et aucun attribut de redirection n'existe sur `Listener`. Le mécanisme est inexistant → contrôle non applicable (CHF-1). |
| outscale | live | ∅ `not-applicable` | Le LBU Outscale ne peut pas rediriger : `ListenerRule.Action` est documenté « always forward » au contrat OAPI (aucune action de redirection), et aucun attribut de redirection n'existe sur `Listener`. Le mécanisme est inexistant → contrôle non applicable (CHF-1). |

## Ce que Pépin peut conclure

| Statut | Ce que le statut affirme | Atteignable depuis |
|---|---|---|
| `fail` | un écart a été détecté sur une ressource réelle | aucun |
| `pass` | la donnée décisive a été collectée, et elle est conforme | aucun |
| `not-applicable` | le contrat du fournisseur déclare le contrôle non testable, avec sa justification | exoscale / terraform · exoscale / live · outscale / terraform · outscale / live |
| `not-evaluated` | le contrôle est implémenté, mais la donnée dont il dépend n'a pas été confirmée | aucun |

Un contrôle observable rend tout de même `not-evaluated` sur un inventaire qui ne
contient aucune ressource du type visé : « rien à voir » n'est pas « conforme ».

## Comment enquêter

- Type de ressource normalisé lu par la règle : `load_balancer`
- Attribut dont la décision dépend : `redirect_to_https`
- Sans cet attribut sur une ressource du type visé, le scan rend `not-evaluated` et non `pass` (`internal/assess`, table `requiredAttr`).
- La règle qui émet ce code vit dans [`internal/commonrules/rules/`](../../internal/commonrules/rules) : elle est **commune** à tous les fournisseurs, seule la source change.

## Comment corriger

Mettre en place une redirection 301 du listener HTTP:80 vers HTTPS.

_Contrôle dormant : aucun fournisseur déclaré, donc aucune preuve attendue._

Une preuve de remédiation est un module Terraform autonome, **conforme**, qui se déploie
tel quel, ou une note ancrée sur la documentation officielle. Voir
[le guide de remédiation](../guides/remediation.fr.md).

## Comment vérifier la correction

Aucune source ne sait aujourd'hui conclure `pass` sur ce contrôle : un scan
peut faire disparaître l'écart, il ne peut pas **démontrer** la conformité. Le tableau
des motifs ci-dessus dit ce qui manque.
## Voir aussi

- [Le modèle d'assessment](../concepts/assessment-model.fr.md) : ce que chaque statut affirme.
- [Matrice de couverture](../coverage.fr.md) : la même information, tous contrôles confondus.
- [Plan Terraform ou scan live](../concepts/terraform-vs-live.fr.md) : choisir la source.
- [Ajouter un contrôle](../contributing/adding-a-control.fr.md) : la procédure de bout en bout.

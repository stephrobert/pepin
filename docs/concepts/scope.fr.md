> [🇬🇧 English](scope.md) · 🇫🇷 Français

# Périmètre et non-objectifs

Pépin imprime ceci à **chaque** scan, sur `stderr`, avant que le rapport puisse être mal lu :

<!-- pepin:gen scope-disclaimer -->
```text
Ce rapport évalue la configuration d'un tenant (périmètre commanditaire). Les
correspondances normatives (SecNumCloud, ISO, CIS) sont indicatives : elles ne
constituent pas une preuve de qualification/certification, laquelle porte sur le
prestataire de service cloud.
```
<!-- /pepin:gen scope-disclaimer -->

Cette page est cette phrase, développée. Elle n'en est pas une version adoucie : là où les deux
pourraient se lire différemment, c'est la constante `assess.ScopeDisclaimer` qui fait foi, et
cette page qui a tort.

## Ce que Pépin évalue : la posture d'un tenant

Un cloud a deux côtés, et ils sont audités par deux choses différentes.

| | **Prestataire** de service cloud | **Client** du cloud (tenant) |
|---|---|---|
| Détient | centres de données, hyperviseurs, plan de contrôle, procédures d'exploitation, contrôle du personnel | comptes, IAM, réseaux, instances, buckets, bases de données, leur configuration |
| Audité par | un organisme de qualification ou de certification (ANSSI pour SecNumCloud, auditeur accrédité pour ISO 27001) | des outils comme Pépin, plus vos propres procédures |
| Produit | une qualification, un certificat, une attestation | des preuves sur une configuration, à un instant donné |

**Pépin ne regarde jamais que la colonne de droite.** Il lit votre compte via l'API du
fournisseur, ou votre code d'infrastructure via un plan Terraform. Il n'a aucune visibilité sur
la façon dont le fournisseur exploite sa plateforme, et n'en affirme rien.

## De la configuration observable, et rien d'autre

Pépin mesure des **faits de configuration qu'une API ou un plan expose**. Cette frontière a des
conséquences nettes, et elles sont documentées plutôt qu'escamotées :

- Une propriété que l'API n'expose pas ne peut pas être mesurée. Là où un contrat de
  fournisseur le consigne (`etat: absent`, ou une entrée `contrat.non_applicable`), le contrôle
  rend `not-applicable` **avec la justification consignée**.
- Une propriété qui existe mais n'a pas été collectée sur ce run rend `not-evaluated`, avec le
  motif. Jamais `pass`.
- Un chiffrement réalisé **dans l'invité** (LUKS sur un volume block, chiffrement applicatif)
  est invisible de l'API de la plateforme, par construction. Pépin le dit ; il ne devine pas.

Toute la taxonomie est dans [le modèle d'assessment](assessment-model.fr.md), et les
conséquences par fournisseur dans les [limites connues](../known-limitations.fr.md).

## Deux sources, deux affirmations différentes

| Source | Commande | Ce qu'elle affirme |
|---|---|---|
| Plan Terraform | `--terraform plan.json` | ce que votre code **déclare** vouloir créer. Rien sur la dérive, rien sur ce qui tourne déjà, rien sur les attributs encore `known after apply`. Le verdict le dit : « périmètre déclaré (plan Terraform, état planifié) ». |
| API live | `--live` | la configuration **effective** du tenant à l'instant du scan, dans la limite de ce que les identifiants ont pu lire. Des droits insuffisants ressortent en `not-evaluated`, jamais en conformité. |

Aucune des deux sources n'affirme quoi que ce soit sur l'intervalle entre deux scans.

## Une correspondance n'est pas une certification

Chaque contrôle de `referentiel/controles.yaml` porte des correspondances : une exigence SCSL
(`CLD-*`, issue de l'index gelé), et des renvois vers SecNumCloud 3.2, CIS Controls v8 et
ISO/IEC 27001:2022 / 27017. Ces correspondances existent pour qu'un résultat puisse être
*classé* sous un référentiel qu'un auditeur connaît déjà.

Elles ne signifient pas, et ne peuvent pas signifier :

- que réussir les contrôles associés satisfait l'exigence du référentiel : une exigence couvre
  d'ordinaire de l'organisation, de la procédure et de la preuve, dont la configuration n'est
  qu'une part ;
- que votre tenant est qualifié SecNumCloud : la **qualification porte sur le prestataire**,
  elle est délivrée par l'ANSSI, et aucun scanner ne la délivre ;
- que votre fournisseur est qualifié : les faits de souveraineté de `providers/<nom>.yaml` sont
  **déclarés** depuis des sources publiques et cités comme tels, ils ne sont pas vérifiés par
  Pépin.

Ce dernier point est visible dans la sortie elle-même : le contrôle
`governance_provider_sovereignty` porte la preuve « conforme selon les faits de souveraineté
déclarés au descripteur du fournisseur (attestation, non mesuré sur le tenant) ». Il dit
« attestation » parce que c'en est une.

## Ce que « opposable » veut dire ici

Dans le vocabulaire de Pépin, « opposable » est une propriété du **document de résultat**, pas
un statut juridique. Un résultat est opposable quand un tiers peut le contester sur des faits
plutôt que sur la confiance. Concrètement :

- **des statuts typés** : `pass` / `fail` / `not-applicable` / `not-evaluated`, pour qu'« aucun
  finding » ne soit jamais présenté comme « conforme » ;
- **une justification sur chaque non-mesure** : un `not-applicable` non justifié n'est pas
  produit du tout ;
- **des références normatives exactes** sur chaque résultat, pas un nom de référentiel vague ;
- **une provenance** : l'empreinte du binaire, et une empreinte couvrant les règles, les
  descripteurs de fournisseurs et le référentiel, pour que deux configurations différentes ne
  puissent pas produire la même empreinte sous le même résultat ;
- **un bundle scellable** : `--seal` écrit l'inventaire évalué, l'assessment, son rendu OSCAL
  1.1.2 et un manifest à empreintes, que `pepin verify` revérifie et sait re-dériver.

Cela ne veut **pas** dire que le résultat est recevable où que ce soit, ni qu'il remplace un
audit.

## Non-objectifs

Pépin ne fait pas :

- **délivrer, prouver ou prédire une qualification ou une certification**, quelle qu'elle soit ;
- **auditer le fournisseur de cloud** : sa plateforme, ses procédures, son personnel, ses
  sous-traitants ;
- **auditer l'intérieur d'un cluster Kubernetes comme s'il s'agissait d'un plan de contrôle
  cloud** : le provider `kubernetes` existe pour l'état in-cluster (RBAC, Pod Security,
  NetworkPolicy) et reste délibérément hors de toute comparaison de parité avec les clouds,
  aucune des deux portées ne pouvant couvrir l'autre ;
- **scanner les charges de travail** : ni analyse de vulnérabilités d'images, ni agent
  d'exécution, ni SAST/DAST ;
- **lire les données applicatives** : il lit des métadonnées de configuration, et l'inventaire
  évalué est précisément ce que vous pouvez inspecter dans un bundle scellé ;
- **remédier** : il n'écrit jamais dans une API cloud. Un scan live demande des identifiants en
  lecture seule, et l'outil ne saurait rien faire de plus même si vous lui en donniez plus ;
- **remplacer un audit, une analyse de risque ou une politique de sécurité** ;
- **mesurer quoi que ce soit entre deux runs** : un résultat décrit un instant.

## Terminologie, tenue cohérente

| Terme | Chez Pépin |
|---|---|
| tenant | le compte ou l'organisation côté client, celui qui est scanné. Jamais le fournisseur. |
| contrôle | une entrée agnostique de `referentiel/controles.yaml`, reliée à une exigence SCSL gelée |
| check | le code agnostique qu'une règle Rego émet, conservé dans `labels.check` |
| source | `terraform-plan`, `live-api`, ou `export` |
| finding | un écart, sur un sujet |
| assessment | le document typé, référencé et à provenance, couvrant tous les contrôles |
| bundle de preuve | le dossier scellé produit par `--seal` |

## Le référentiel, en chiffres

<!-- pepin:gen control-counts -->
| Chiffre | Nombre |
|---|---:|
| Contrôles au référentiel | 57 |
| Contrôles déclarés pour au moins un fournisseur | 56 |
| `critical` | 10 |
| `high` | 32 |
| `medium` | 13 |
| `low` | 2 |
<!-- /pepin:gen control-counts -->

Le détail, par fournisseur et par source, est dans la
[matrice de couverture](../coverage.fr.md).

## Voir aussi

- [Le modèle d'assessment](assessment-model.fr.md) : ce qu'affirme chaque statut.
- [Limites connues](../known-limitations.fr.md) : les angles morts, nommés.
- [Matrice de couverture](../coverage.fr.md) : ce qui est mesurable aujourd'hui.

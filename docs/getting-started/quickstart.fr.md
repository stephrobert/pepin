> [🇬🇧 English](quickstart.md) · 🇫🇷 Français

# Démarrage rapide : cinq minutes, aucun compte cloud

Cette page va de zéro à un scan auquel se fier : un module Terraform volontairement mal
configuré, un échec réel, sa correction, puis un second scan qui rend un verdict différent.

**Aucun compte Scaleway, Outscale ou Exoscale n'est nécessaire.** Rien n'est provisionné.
Pépin lit un *plan* Terraform (l'état que votre code déclare) et n'appelle ici aucune API de
fournisseur.

Toutes les commandes ci-dessous sont exécutées par le générateur de documentation sur le
dépôt, et chaque sortie affichée est celle qu'il a capturée. Voir
[Comment cette page reste vraie](#comment-cette-page-reste-vraie).

## 1. Récupérer Pépin

Les binaires publiés (avec sommes de contrôle, signature et provenance SLSA), l'image de
conteneur, l'action GitHub et le template GitLab sont documentés dans
[install.fr.md](../install.fr.md). Depuis les sources :

```bash
git clone https://github.com/stephrobert/pepin && cd pepin
go build -o pepin .          # Go 1.26+
./pepin provider list
```

<!-- pepin:gen provider-list -->
```text

// pépin  providers enregistrés
  exoscale  Exoscale (CH) — instances, security groups, block storage, SKS, SOS
  kubernetes  Kubernetes (in-cluster) — RBAC, Pod Security Standards, NetworkPolicy
  outscale  Outscale (3DS) — VM, BSU, OOS, EIM, security groups, OKS, LBU
  scaleway  Scaleway — object storage, instances, IAM, security groups
```
<!-- /pepin:gen provider-list -->

`kubernetes` n'est pas un cloud : il audite l'état *dans* un cluster. Ce parcours s'appuie sur
les trois clouds souverains.

## 2. L'exemple volontairement non conforme

`examples/scaleway/terraform/main.tf` livre un petit module Scaleway faux à dessein : un
bucket en `public-read`, SSH ouvert à `0.0.0.0/0`, une base managée sans chiffrement au repos
et sans sauvegardes, un secret dans le cloud-init, une politique IAM capable de s'octroyer des
droits IAM.

Son plan est committé sous `examples/scaleway/terraform/plan.json` : le scan est donc
immédiat. Pour le régénérer vous-même (cela télécharge le provider Scaleway, mais ne crée
toujours rien) :

```bash
cd examples/scaleway/terraform
terraform init && terraform plan -out tfplan && terraform show -json tfplan > plan.json
cd -
```

## 3. Le scanner

```bash
./pepin scan scaleway --terraform examples/scaleway/terraform/plan.json
```

Le rapport s'ouvre sur les trois écarts les plus graves : le premier écran est donc déjà
actionnable.

<!-- pepin:gen scan-vulnerable-head -->
```text
──────────────────────────────────────────────────────────────────────────────
 Mode      scan scaleway (terraform)
 Source    examples/scaleway/terraform/plan.json
──────────────────────────────────────────────────────────────────────────────

──────────────────────────────────────────────────────────────────────────────
 ⚡ Immediate action — top 3 most severe deviations
──────────────────────────────────────────────────────────────────────────────

  1. 🔴 CRIT  CLD-STO-1 — Bucket « scaleway_object_bucket_acl.backups » accessible publiq…
     subject: scaleway_object_bucket_acl.backups
  2. 🟠 HIGH  CLD-CMP-9 — secret en clair dans user-data (mot de passe en clair).
     subject: scaleway_instance_server.web
  3. 🟠 HIGH  CLD-STO-3 — sauvegardes automatiques désactivées.
     subject: pepin-test-rdb


──────────────────────────────────────────────────────────────────────────────
 CRITICAL  ·  CLD-STO-1  ·  scaleway
 Stockage objet exposé publiquement
[…]
```
<!-- /pepin:gen scan-vulnerable-head -->

Il se referme sur le tableau par contrôle et le verdict :

<!-- pepin:gen scan-vulnerable-tail -->
```text
[…]

  Controls
  ╭────────────┬──────────────────────────────────────────────────┬──────────┬──────────┬───╮
  │ Code       │ Control                                          │ Sev      │ Tier     │ # │
  ├────────────┼──────────────────────────────────────────────────┼──────────┼──────────┼───┤
  │ CLD-STO-1  │ Stockage objet exposé publiquement               │ CRITICAL │ scaleway │ 1 │
  │ CLD-CHF-2  │ Base de données managée sans chiffrement au rep… │ HIGH     │ scaleway │ 1 │
  │ CLD-CMP-9  │ Secret en clair dans les données utilisateur (u… │ HIGH     │ scaleway │ 1 │
  │ CLD-IAM-12 │ Politique IAM permettant une élévation de privi… │ HIGH     │ scaleway │ 1 │
  │ CLD-NET-1  │ Base de données managée joignable depuis Intern… │ HIGH     │ scaleway │ 2 │
  │ CLD-NET-2  │ Politique entrante par défaut d'un groupe de sé… │ HIGH     │ scaleway │ 1 │
  │ CLD-STO-3  │ Sauvegardes automatiques d'une base managée dés… │ HIGH     │ scaleway │ 1 │
  │ CLD-GVN-1  │ Inventaire et étiquetage incomplets              │ MEDIUM   │ scaleway │ 1 │
  │ CLD-STO-8  │ Object Lock (immutabilité) désactivé sur le sto… │ LOW      │ scaleway │ 1 │
  ╰────────────┴──────────────────────────────────────────────────┴──────────┴──────────┴───╯
──────────────────────────────────────────────────────────────────────────────
 Summary

 Verdict : NON CONFORME

 🔴 CRITICAL 1   🟠 HIGH 7   🟡 MEDIUM 1   🔵 LOW 1
──────────────────────────────────────────────────────────────────────────────
```
<!-- /pepin:gen scan-vulnerable-tail -->

Entre les deux, un bloc par contrôle détaille l'écart et sa remédiation. La sortie complète est
commentée ligne à ligne dans [Lire un scan](understanding-a-scan.fr.md).

```bash
echo $?   # 1 : au moins un écart critical ou high
```

## 4. Lire un FAIL

Prenons `CLD-CHF-2`, « base de données managée sans chiffrement au repos », sévérité `high`.
Son bloc, extrait du run ci-dessus :

<!-- pepin:gen scan-control-encryption -->
```text
──────────────────────────────────────────────────────────────────────────────
 HIGH  ·  CLD-CHF-2  ·  scaleway
 Base de données managée sans chiffrement au repos
──────────────────────────────────────────────────────────────────────────────
  Total deviations: 1

  Details:
      HIGH  pepin-test-rdb — Base de données managée « pepin-test-rdb » sans chiffrement au repos.

  Remediation
    Activer le chiffrement au repos de l'instance (à la création ou par mise à niveau).

  ↳ docs: https://stephane-robert.info/scsl/CLD-CHF-2
```
<!-- /pepin:gen scan-control-encryption -->

Il se lit à rebours, dans l'ordre où il a été produit :

1. **La preuve.** Le plan déclare `encryption_at_rest = false` sur
   `scaleway_rdb_instance.insecure`.
2. **La projection.** Le descripteur Scaleway (`providers/scaleway.yaml`) projette cet attribut
   Terraform sur l'attribut normalisé commun `encryption_at_rest` du type `managed_database`.
3. **La règle.** La règle commune `database_encryption_at_rest_enabled` se déclenche sur toute
   `managed_database` dont `encryption_at_rest` est faux : la même règle pour tous les
   fournisseurs.
4. **Le contrôle.** `referentiel/controles.yaml` relie ce code agnostique à l'exigence SCSL
   gelée `CLD-CHF-2`, et de là à SecNumCloud, ISO et CIS.

Le rapport affiche `CLD-CHF-2` parce que le finding a été résolu contre le référentiel ; le
nom du check agnostique reste, lui, dans `labels.check`.

## 5. Corriger

`examples/scaleway/terraform-fixed/main.tf` est le même module, chaque écart corrigé. Pour la
base de données, la correction tient en deux lignes :

```diff
 resource "scaleway_rdb_instance" "insecure" {
   name               = "pepin-test-rdb"
   node_type          = "DB-DEV-S"
   engine             = "PostgreSQL-15"
   user_name          = "admin"
   password           = random_password.db.result
-  encryption_at_rest = false
-  disable_backup     = true
+  encryption_at_rest = true
+  disable_backup     = false
 }
```

Les autres corrections suivent la même forme : `acl = "private"` au lieu de `"public-read"`,
`ip_range = "10.0.0.0/8"` sur la règle SSH au lieu d'une plage omise (qui vaut *toute*
origine), `inbound_default_policy = "drop"`, un cloud-init sans mot de passe, une politique IAM
sans le PermissionSet `IAMManager`, et les quatre étiquettes de gouvernance sur l'instance.

## 6. Rescanner

```bash
./pepin scan scaleway --terraform examples/scaleway/terraform-fixed/plan.json
```

<!-- pepin:gen scan-fixed-full -->
```text
──────────────────────────────────────────────────────────────────────────────
 Mode      scan scaleway (terraform)
 Source    examples/scaleway/terraform-fixed/plan.json
──────────────────────────────────────────────────────────────────────────────

  ✓ No deviations found in the audited scope.

──────────────────────────────────────────────────────────────────────────────
 Summary

 Verdict : conforme sur le périmètre déclaré (plan Terraform, état planifié) (aucune non-conformité détectée, 16 contrôles conformes)

 🔴 CRITICAL 0   🟠 HIGH 0   🟡 MEDIUM 0   🔵 LOW 0
──────────────────────────────────────────────────────────────────────────────
```
<!-- /pepin:gen scan-fixed-full -->

```bash
echo $?   # 0
```

La formulation du verdict compte. Il dit **« conforme sur le périmètre déclaré (plan
Terraform, état planifié) »**, et non « conforme ». Un plan décrit ce que votre code *entend*
créer ; seul un scan live (`--live`) observe ce qui tourne réellement. Et il annonce le nombre
de contrôles conformes, parce qu'un scan qui n'a rien mesuré ne doit jamais ressembler à un
scan qui n'a rien trouvé.

## 7. Codes de sortie, observés

Pépin est fait pour tenir une porte de CI : ses codes de sortie font partie du contrat. Chaque
ligne ci-dessous a été produite en exécutant la commande affichée.

<!-- pepin:gen exit-codes -->
| Situation | Commande | Code de sortie |
|---|---|:-:|
| Aucun écart sur le périmètre évalué | `./pepin scan scaleway --terraform examples/scaleway/terraform-fixed/plan.json` | **0** |
| Au moins un écart critical ou high | `./pepin scan scaleway --terraform examples/scaleway/terraform/plan.json` | **1** |
| Erreur technique (fichier illisible, provider inconnu, API injoignable) | `./pepin scan scaleway examples/scaleway/plan-absent.json` | **2** |
| Rien n'a été mesuré (inventaire vide) : **sans avoir à demander `--strict`** | `./pepin scan scaleway empty-inventory.json` | **3** |
| Écarts medium/low seulement, sans `--strict` | `./pepin scan scaleway tagless-inventory.json` | **0** |
| Écarts medium/low seulement, avec `--strict` | `./pepin scan scaleway tagless-inventory.json --strict` | **3** |
| Aucun écart, mais une unité de collecte n'a pas pu être lue | `./pepin scan scaleway partial-inventory.json` | **3** |
| Tout écart critical/high est couvert par une dérogation valide | `./pepin scan scaleway bastion-inventory.json --exceptions exceptions.yaml` | **4** |
| La même dérogation, échue : elle ne s'applique plus | `./pepin scan scaleway bastion-inventory.json --exceptions exceptions-expired.yaml` | **1** |
<!-- /pepin:gen exit-codes -->

Deux d'entre eux méritent l'attention :

- **`3` sans `--strict`.** Un scan qui n'a rien mesuré rend 3, jamais 0. Des identifiants
  expirés, des droits insuffisants, une région vide ou un inventaire tronqué produisent le
  même résultat vide qu'un tenant sain, et un résultat vide ne dit rien de la posture.
- **`3` avec `--strict`.** La porte stricte refuse en plus les écarts medium/low subsistants,
  que le code de sortie normal ignore.

Les deux inventaires jetables utilisés ci-dessus ne sont pas des fixtures du dépôt : écrivez-les
vous-même. `empty-inventory.json`, un tenant dont rien n'a été collecté :

<!-- pepin:gen fixture-empty-inventory -->
```json
{
  "provider": "scaleway",
  "resources": []
}
```
<!-- /pepin:gen fixture-empty-inventory -->

`tagless-inventory.json`, une instance à qui il manque ses étiquettes de gouvernance, ce qui
est un écart `medium` et rien d'autre :

<!-- pepin:gen fixture-tagless-inventory -->
```json
{
  "provider": "scaleway",
  "resources": [
    {
      "provider": "scaleway",
      "type": "compute_instance",
      "id": "srv-demo",
      "name": "srv-demo",
      "region": "fr-par",
      "attributes": {
        "vm_id": "srv-demo",
        "security_group_ids": ["sg-front"],
        "tags": []
      }
    }
  ]
}
```
<!-- /pepin:gen fixture-tagless-inventory -->

## Pour aller plus loin

- [Lire un scan](understanding-a-scan.fr.md) : la sortie complète, commentée ligne à ligne.
- [Le modèle d'assessment](../concepts/assessment-model.fr.md) : ce que `pass`, `fail`,
  `not-applicable` et `not-evaluated` affirment réellement.
- [Matrice de couverture](../coverage.fr.md) : ce qui est mesurable, par fournisseur et par
  source.
- [Limites connues](../known-limitations.fr.md) : ce que Pépin ne sait pas voir, et pourquoi.
- [Périmètre et non-objectifs](../concepts/scope.fr.md) : ce qu'un rapport Pépin n'est *pas*.

## Comment cette page reste vraie

Les sorties ci-dessus ne sont pas recopiées. `internal/docgen` exécute le binaire `pepin` sur
les fixtures du dépôt et injecte ce qu'il a capturé entre les marqueurs `pepin:gen`.
`mise run gen-docs` les réécrit ; `TestGeneratedDocsAreUpToDate` échoue si ce qui est committé
diffère de ce que le binaire produit aujourd'hui. Quand le produit change, cette page casse,
et c'est précisément l'objectif.

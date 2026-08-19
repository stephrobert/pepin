> [🇬🇧 English](understanding-a-scan.md) · 🇫🇷 Français

# Lire un scan

Cette page commente un run Pépin réel, de bout en bout. La commande :

```bash
./pepin scan scaleway --terraform examples/scaleway/terraform/plan.json
```

Tout ce qui est reproduit ci-dessous a été capturé depuis cette commande exacte, sur ce dépôt :
le rapport du terminal, le document d'assessment, le code de sortie. Rien n'est illustratif.

## Les deux flux

Pépin les sépare à dessein, pour que `pepin scan … > rapport.txt` vous donne le rapport et rien
d'autre.

**`stderr`** porte le bandeau et l'avertissement de portée :

<!-- pepin:gen scan-vulnerable-banner -->
```text

 ██████╗ ███████╗██████╗ ██╗ ███╗   ██╗
 ██╔══██╗██╔════╝██╔══██╗██║ ████╗  ██║
 ██████╔╝█████╗  ██████╔╝██║ ██╔██╗ ██║
 ██╔═══╝ ██╔══╝  ██╔═══╝ ██║ ██║╚██╗██║
 ██║     ███████╗██║     ██║ ██║ ╚████║
 ╚═╝     ╚══════╝╚═╝     ╚═╝ ╚═╝  ╚═══╝

 v<version>  · scanner de posture cloud (sécurité · conformité)


ⓘ Ce rapport évalue la configuration d'un tenant (périmètre commanditaire). Les correspondances normatives (SecNumCloud, ISO, CIS) sont indicatives : elles ne constituent pas une preuve de qualification/certification, laquelle porte sur le prestataire de service cloud.
```
<!-- /pepin:gen scan-vulnerable-banner -->

La version est affichée `<version>` : elle est injectée au build depuis `git describe`, elle
diffère donc d'une machine et d'un commit à l'autre, et la figer ici ferait diverger cette page
à chaque build sans qu'aucun comportement n'ait bougé. Tout le reste est la sortie capturée.

Le bandeau est imprimé *avant* le début de la collecte, pas à la fin : sur un scan live d'un
gros tenant, vous voulez savoir que l'outil a démarré. La dernière ligne est
`assess.ScopeDisclaimer`, émis à chaque scan sans exception (voir
[Périmètre](../concepts/scope.fr.md)).

**`stdout`** porte le rapport :

<!-- pepin:gen scan-vulnerable-full -->
```text
──────────────────────────────────────────────────────────────────────────────
 Mode      scan scaleway (terraform)
 Source    examples/scaleway/terraform/plan.json
──────────────────────────────────────────────────────────────────────────────

──────────────────────────────────────────────────────────────────────────────
 ⚡ Immediate action — top 3 most severe deviations
──────────────────────────────────────────────────────────────────────────────

  1. 🔴 CRIT  CLD-CMP-1 — aucun filtrage réseau ne s'applique.
     subject: scaleway_instance_server.web
  2. 🔴 CRIT  CLD-STO-1 — Bucket « scaleway_object_bucket_acl.backups » accessible publiq…
     subject: scaleway_object_bucket_acl.backups
  3. 🟠 HIGH  CLD-CMP-9 — secret en clair dans user-data (mot de passe en clair).
     subject: scaleway_instance_server.web


──────────────────────────────────────────────────────────────────────────────
 CRITICAL  ·  CLD-CMP-1  ·  scaleway
 Machine sans filtrage réseau
──────────────────────────────────────────────────────────────────────────────
  Total deviations: 1

  Details:
      CRIT  scaleway_instance_server.web — aucun filtrage réseau ne s'applique.

  Remediation
    Attacher un groupe de sécurité restrictif (refus par défaut) à la VM.

  ↳ docs: https://stephane-robert.info/scsl/CLD-CMP-1

──────────────────────────────────────────────────────────────────────────────
 CRITICAL  ·  CLD-STO-1  ·  scaleway
 Stockage objet exposé publiquement
──────────────────────────────────────────────────────────────────────────────
  Total deviations: 1

  Details:
      CRIT  scaleway_object_bucket_acl.backups — Bucket « scaleway_object_bucket_acl.backups » accessible publiquement (ACL publique).

  Remediation
    Rendre le bucket privé (ACL private, retrait du grant AllUsers, suppression de la policy publique) ; servir via des URLs pré-signées si nécessaire.

  ↳ docs: https://stephane-robert.info/scsl/CLD-STO-1

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

──────────────────────────────────────────────────────────────────────────────
 HIGH  ·  CLD-CMP-9  ·  scaleway
 Secret en clair dans les données utilisateur (user-data)
──────────────────────────────────────────────────────────────────────────────
  Total deviations: 1

  Details:
      HIGH  scaleway_instance_server.web — secret en clair dans user-data (mot de passe en clair).

  Remediation
    Bannir les secrets des données utilisateur ; utiliser un coffre de secrets et l'injection au démarrage. Révoquer le secret exposé.

  ↳ docs: https://stephane-robert.info/scsl/CLD-CMP-9

──────────────────────────────────────────────────────────────────────────────
 HIGH  ·  CLD-IAM-12  ·  scaleway
 Politique IAM permettant une élévation de privilèges
──────────────────────────────────────────────────────────────────────────────
  Total deviations: 1

  Details:
      HIGH  ci-deployer — confère la gestion de l'IAM (PermissionSet) — chemin d'élévation de privilèges.

  Remediation
    Réserver la gestion IAM à une politique d'administration dédiée ; retirer le PermissionSet de gestion des politiques d'usage.

  ↳ docs: https://stephane-robert.info/scsl/CLD-IAM-12

──────────────────────────────────────────────────────────────────────────────
 HIGH  ·  CLD-NET-1  ·  scaleway
 Base de données managée joignable depuis Internet
──────────────────────────────────────────────────────────────────────────────
  Total deviations: 2

  Details:
      HIGH  fr-par/11111111-1111-1111-1111-111111111111 — ACL autorisant un CIDR public (0.0.0.0/0) — service exposé à Internet.
      HIGH  scaleway_instance_security_group.web — SSH (port 22) accepté depuis/vers Internet.

  Remediation
    Restreindre l'ACL de la base aux seuls CIDR applicatifs (réseau privé quand disponible) ; retirer 0.0.0.0/0.

  ↳ docs: https://stephane-robert.info/scsl/CLD-NET-1

──────────────────────────────────────────────────────────────────────────────
 HIGH  ·  CLD-NET-2  ·  scaleway
 Politique entrante par défaut d'un groupe de sécurité en « accept »
──────────────────────────────────────────────────────────────────────────────
  Total deviations: 1

  Details:
      HIGH  sg-open-default — politique entrante par défaut « accept » — tout trafic non filtré est admis.

  Remediation
    Basculer la politique entrante par défaut sur « drop » et n'ouvrir que les flux légitimes par des règles explicites.

  ↳ docs: https://stephane-robert.info/scsl/CLD-NET-2

──────────────────────────────────────────────────────────────────────────────
 HIGH  ·  CLD-STO-3  ·  scaleway
 Sauvegardes automatiques d'une base managée désactivées
──────────────────────────────────────────────────────────────────────────────
  Total deviations: 1

  Details:
      HIGH  pepin-test-rdb — sauvegardes automatiques désactivées.

  Remediation
    Réactiver les sauvegardes automatiques et fixer une rétention adaptée au RPO.

  ↳ docs: https://stephane-robert.info/scsl/CLD-STO-3

──────────────────────────────────────────────────────────────────────────────
 MEDIUM  ·  CLD-GVN-1  ·  scaleway
 Inventaire et étiquetage incomplets
──────────────────────────────────────────────────────────────────────────────
  Total deviations: 1

  Details:
      MED   scaleway_instance_server.web — étiquettes de gouvernance manquantes (CostCenter, Project, Env, Owner).

  Remediation
    Ajouter les étiquettes obligatoires (CostCenter, Project, Env, Owner) sur la ressource.

  ↳ docs: https://stephane-robert.info/scsl/CLD-GVN-1

──────────────────────────────────────────────────────────────────────────────
 LOW  ·  CLD-STO-8  ·  scaleway
 Object Lock (immutabilité) désactivé sur le stockage objet
──────────────────────────────────────────────────────────────────────────────
  Total deviations: 1

  Details:
      LOW   backups-prod — objets non immuables (pas de protection WORM contre suppression/écrasement).

  Remediation
    Activer l'Object Lock (mode conformité/gouvernance) sur les buckets de sauvegarde et d'objets critiques.

  ↳ docs: https://stephane-robert.info/scsl/CLD-STO-8

  Controls
  ╭────────────┬──────────────────────────────────────────────────┬──────────┬──────────┬───╮
  │ Code       │ Control                                          │ Sev      │ Tier     │ # │
  ├────────────┼──────────────────────────────────────────────────┼──────────┼──────────┼───┤
  │ CLD-CMP-1  │ Machine sans filtrage réseau                     │ CRITICAL │ scaleway │ 1 │
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

 🔴 CRITICAL 2   🟠 HIGH 7   🟡 MEDIUM 1   🔵 LOW 1
──────────────────────────────────────────────────────────────────────────────
```
<!-- /pepin:gen scan-vulnerable-full -->

## Lecture, section par section

### En-tête

```
 Mode      scan scaleway (terraform)
 Source    examples/scaleway/terraform/plan.json
```

`Mode` nomme le **fournisseur** et la **source**. `(terraform)` signifie qu'un plan a été
projeté par le mapping Terraform du fournisseur ; `(live)` signifierait que l'API a été
interrogée, et `Source` afficherait alors `collecte live · profil … · région …` au lieu d'un
chemin de fichier. Un `scan scaleway` nu suivi d'un fichier désigne un export d'inventaire déjà
normalisé.

La distinction n'est pas cosmétique. Un plan porte l'état **planifié** et omet tout ce qui est
`known after apply` ; une collecte live porte la configuration **effective**, mais dépend des
droits des identifiants employés.

### Action immédiate

Les trois écarts les plus graves, classés, avec leur sujet. Cette section existe pour que le
premier écran d'un long rapport soit déjà actionnable.

### Un bloc par contrôle

```
 CRITICAL  ·  CLD-STO-1  ·  scaleway
 Stockage objet exposé publiquement
```

- **sévérité** : `critical`, `high`, `medium`, `low`, issue du référentiel.
- **`CLD-STO-1`** : l'identifiant du **contrôle**, c'est-à-dire l'exigence SCSL gelée. Les
  règles, elles, émettent un nom de check agnostique
  (`objectstorage_bucket_public_access`) ; le scan le résout contre
  `referentiel/controles.yaml` et conserve le nom du check dans `labels.check`.
- **`scaleway`** : le palier, tiré de la ressource elle-même (`labels.provider`), jamais codé
  en dur dans une règle.
- le titre, puis `Total deviations`, puis une ligne `Details` par sujet fautif.

`Details` porte la **preuve** : la ressource en écart, et le fait observé à son sujet.

```
      CRIT  scaleway_object_bucket_acl.backups — Bucket « … » accessible publiquement (ACL publique).
```

Vient ensuite la **remédiation**, puis un lien `docs:` vers la page SCSL de l'exigence.

### Tableau des contrôles et résumé

Le tableau récapitule code, titre, sévérité, palier et nombre d'écarts. Le résumé porte le
**verdict** et les compteurs par sévérité.

La formulation du verdict est porteuse. Sur un plan Terraform, elle dit « périmètre déclaré
(plan Terraform, état planifié) » ; sur un scan où rien n'a été mesuré, elle dit
`Verdict : INDÉTERMINÉ`, et le code de sortie suit.

### Code de sortie

```bash
echo $?   # 1
```

<!-- pepin:gen exit-codes -->
| Situation | Commande | Code de sortie |
|---|---|:-:|
| Aucun écart sur le périmètre évalué | `./pepin scan scaleway --terraform examples/scaleway/terraform-fixed/plan.json` | **0** |
| Au moins un écart critical ou high | `./pepin scan scaleway --terraform examples/scaleway/terraform/plan.json` | **1** |
| Erreur technique (fichier illisible, provider inconnu, API injoignable) | `./pepin scan scaleway examples/scaleway/plan-absent.json` | **2** |
| Rien n'a été mesuré (inventaire vide) : **sans avoir à demander `--strict`** | `./pepin scan scaleway empty-inventory.json` | **3** |
| Écarts medium/low seulement, sans `--strict` | `./pepin scan scaleway tagless-inventory.json` | **0** |
| Écarts medium/low seulement, avec `--strict` | `./pepin scan scaleway tagless-inventory.json --strict` | **3** |
<!-- /pepin:gen exit-codes -->

## Le même run, en assessment

Le rapport du terminal montre les écarts. L'**assessment** montre tous les contrôles, y compris
ceux qui n'ont *pas* dévié, ce qui est précisément ce qu'un auditeur demande.

```bash
./pepin scan scaleway --terraform examples/scaleway/terraform/plan.json --format assessment
```

### Provenance

<!-- pepin:gen assessment-run -->
```json
{
  "run": {
    "ruleset": {
      "digest": "\u003cempreinte règles + descripteurs + référentiel\u003e",
      "name": "pepin-config"
    },
    "scope": {
      "included": [
        "compute_instance",
        "governance_provider",
        "iam_policy",
        "managed_database",
        "object_storage_bucket",
        "security_group",
        "security_group_rule"
      ]
    },
    "source": "terraform-plan",
    "target": {
      "id": "scaleway",
      "platform": "scaleway",
      "provider": "scaleway"
    },
    "timestamp": "\u003chorodatage RFC3339 du scan\u003e",
    "tool": {
      "digest": "\u003cempreinte du binaire\u003e",
      "name": "pepin",
      "version": "\u003cversion\u003e"
    }
  }
}
```
<!-- /pepin:gen assessment-run -->

*(horodatage, version et empreintes sont volatils par nature et remplacés ici par des
marqueurs ; tout le reste est le document capturé.)*

- `tool.digest` épingle le **binaire** qui a produit le verdict (révision VCS, `+modified` si
  l'arbre était sale).
- `ruleset.digest` épingle **tout ce qui détermine le résultat** : les règles Rego embarquées,
  les descripteurs de fournisseurs, le référentiel, et le contenu des éventuels
  `--policy-dir`.
- `scope.included` liste les types de ressources réellement évalués. Un contrôle portant sur un
  type absent de cette liste ne peut pas avoir été mesuré, et l'assessment le dit.

### Les quatre statuts, sur ce seul run

<!-- pepin:gen assessment-counts -->
| Statut | Nombre |
|---|---:|
| `pass` | 6 |
| `fail` | 11 |
| `not-applicable` | 2 |
| `not-evaluated` | 8 |
<!-- /pepin:gen assessment-counts -->

Un unique scan sans compte cloud produit les quatre. Chacun est documenté dans
[le modèle d'assessment](../concepts/assessment-model.fr.md) ; en voici un de chaque, mot pour
mot.

**`fail`**, un écart, avec son sujet et sa preuve :

<!-- pepin:gen assessment-fail -->
```json
{
  "control": "objectstorage_bucket_public_access",
  "evidence": {
    "observed": "Bucket « scaleway_object_bucket_acl.backups » accessible publiquement (ACL publique).",
    "proves": [
      "",
      "",
      ""
    ],
    "source": "terraform-plan"
  },
  "labels": {
    "category": "security",
    "provider": "scaleway"
  },
  "references": [
    {
      "framework": "scsl",
      "id": "CLD-STO-1"
    },
    {
      "framework": "cis-v8",
      "id": "3.3"
    },
    {
      "framework": "iso-27001",
      "id": "A.5.15"
    },
    {
      "framework": "iso-27001",
      "id": "A.8.3"
    },
    {
      "framework": "iso-27017",
      "id": "CLD.9.5.1"
    },
    {
      "framework": "secnumcloud-3.2",
      "id": "9.7"
    },
    {
      "framework": "secnumcloud-3.2",
      "id": "13.2"
    }
  ],
  "remediation": "Rendre le bucket privé (ACL private, retrait du grant AllUsers, suppression de la policy publique) ; servir via des URLs pré-signées si nécessaire.",
  "severity": "critical",
  "status": "fail",
  "subject": "scaleway_object_bucket_acl.backups",
  "title": "Stockage objet exposé publiquement"
}
```
<!-- /pepin:gen assessment-fail -->

**`pass`** : remarquez que `evidence.observed` nomme *ce qui* a été vérifié.

<!-- pepin:gen assessment-pass -->
```json
{
  "control": "network_securitygroup_allow_ingress_from_internet_to_all_ports",
  "evidence": {
    "observed": "aucune non-conformité détectée sur les ressources de type « security_group_rule » collectées (contrat vérifié)",
    "proves": [
      "",
      "",
      ""
    ],
    "source": "terraform-plan"
  },
  "references": [
    {
      "framework": "scsl",
      "id": "CLD-NET-2"
    },
    {
      "framework": "cis-v8",
      "id": "4.4"
    },
    {
      "framework": "cis-v8",
      "id": "12.2"
    },
    {
      "framework": "iso-27001",
      "id": "A.8.20"
    },
    {
      "framework": "iso-27001",
      "id": "A.8.22"
    },
    {
      "framework": "secnumcloud-3.2",
      "id": "13.2"
    }
  ],
  "severity": "critical",
  "status": "pass",
  "subject": "scaleway",
  "title": "Tout le trafic entrant autorisé depuis Internet (any/any)"
}
```
<!-- /pepin:gen assessment-pass -->

**`not-applicable`**, avec la justification consignée au contrat du fournisseur :

<!-- pepin:gen assessment-na -->
```json
{
  "control": "blockstorage_volume_encryption",
  "references": [
    {
      "framework": "scsl",
      "id": "CLD-CHF-2"
    },
    {
      "framework": "iso-27001",
      "id": "A.8.24"
    },
    {
      "framework": "secnumcloud-3.2",
      "id": "10.1"
    }
  ],
  "severity": "high",
  "status": "not-applicable",
  "subject": "scaleway",
  "title": "Chiffrement au repos désactivé",
  "waiver": {
    "justification": "Chiffrement au repos des volumes block côté invité (LUKS/Cryptsetup), responsabilité du client (responsabilité partagée) ; l'API block n'expose aucun champ de chiffrement → non observable côté plateforme (CHF-2)."
  }
}
```
<!-- /pepin:gen assessment-na -->

**`not-evaluated`**, avec le motif exact qui a empêché Pépin de décider :

<!-- pepin:gen assessment-ne -->
```json
{
  "control": "compute_instance_public_ip_with_open_securitygroup",
  "evidence": {
    "observed": "attribut « public_ip » non collecté sur les ressources de type « compute_instance » (garde de capacité)",
    "proves": [
      "",
      "",
      ""
    ],
    "source": "terraform-plan"
  },
  "references": [
    {
      "framework": "scsl",
      "id": "CLD-NET-3"
    },
    {
      "framework": "cis-v8",
      "id": "4.4"
    },
    {
      "framework": "cis-v8",
      "id": "12.2"
    },
    {
      "framework": "iso-27001",
      "id": "A.8.20"
    },
    {
      "framework": "iso-27001",
      "id": "A.8.22"
    },
    {
      "framework": "iso-27017",
      "id": "CLD.9.5.2"
    },
    {
      "framework": "secnumcloud-3.2",
      "id": "13.2"
    }
  ],
  "severity": "critical",
  "status": "not-evaluated",
  "subject": "scaleway",
  "title": "Machine exposée publiquement sans filtrage restrictif"
}
```
<!-- /pepin:gen assessment-ne -->

## La chaîne, de bout en bout

Prenons l'écart de stockage objet et remontons-le à travers le dépôt.

| Maillon | Artefact | Contenu |
|---|---|---|
| **finding** | rapport terminal / assessment `fail` | `scaleway_object_bucket_acl.backups`, lisible publiquement |
| **preuve** | `examples/scaleway/terraform/plan.json` | `scaleway_object_bucket_acl.backups` déclare `acl = "public-read"` |
| **projection** | `providers/scaleway.yaml`, `mapping_terraform` | `scaleway_object_bucket_acl` vers le type `object_storage_bucket`, attribut `acl` |
| **règle** | `internal/commonrules/rules/objectstorage_bucket_public_access.rego` | une règle commune, `labels.provider` lu sur la ressource |
| **contrôle** | `referentiel/controles.yaml` | code `objectstorage_bucket_public_access`, sévérité `critical` |
| **références normatives** | même entrée | SCSL `CLD-STO-1`, plus les correspondances SecNumCloud, CIS et ISO |
| **remédiation** | champ `remediation` de la règle | rendre le bucket privé, retirer le grant `AllUsers`, servir par URL pré-signée |

Chaque maillon est un fichier de ce dépôt, et chacun est vérifié : `mise run validate` refuse un
contrôle sans règle, un code émis mais non catalogué, ou une référence SCSL absente de l'index
gelé.

## Le sceller

```bash
./pepin scan scaleway --terraform examples/scaleway/terraform/plan.json --seal ./bundle
./pepin verify ./bundle
```

Le bundle contient l'inventaire évalué, l'assessment, son rendu OSCAL 1.1.2, un manifest à
empreintes et un fichier de sommes de contrôle : un tiers peut donc en revérifier l'intégrité
et, avec `--re-derive`, rejouer l'évaluation sur l'inventaire scellé.

**Sans `--redact`, `input.json` embarque l'inventaire BRUT** : user-data, documents de
politique IAM, policies. Pépin l'annonce sur stderr. Traitez un bundle non caviardé comme
sensible, ou produisez-en un partageable avec `--redact` (qui remplace les valeurs sensibles
par des empreintes, et devient de ce fait incompatible avec `verify --re-derive`).

## Autres formats

| Option | Pour |
|---|---|
| `--format table` | les humains (par défaut, montré ci-dessus) |
| `--format json` | `{"findings": …, "summary": …}`, pour les scripts |
| `--format assessment` | le document typé, référencé, à provenance |
| `--format oscal` | résultats d'évaluation OSCAL 1.1.2, validés par schéma en CI |
| `--format sarif` | analyse de code (GitHub, GitLab) |

## Voir aussi

- [Démarrage rapide](quickstart.fr.md) : le même exemple, avec sa correction et un second scan.
- [Le modèle d'assessment](../concepts/assessment-model.fr.md) : ce qu'affirme chaque statut.
- [Matrice de couverture](../coverage.fr.md) : ce qui est mesurable, par fournisseur et par
  source.
- [Périmètre et non-objectifs](../concepts/scope.fr.md) : ce que ce rapport ne prouve pas.

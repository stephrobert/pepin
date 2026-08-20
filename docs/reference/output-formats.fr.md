> [🇬🇧 English](output-formats.md) · 🇫🇷 Français

# Formats de sortie

`pepin scan` rend la même évaluation sous cinq formes. Ce ne sont pas cinq rendus de la même
information : deux d'entre elles savent dire « rien n'a été mesuré », les trois autres non.
Choisir la mauvaise, c'est se retrouver avec une chaîne de conformité incapable de distinguer
un tenant propre d'un tenant non collecté.

Tous les exemples ci-dessous sont le même scan du même plan Terraform, dans le format que la
section nomme.

| Format | Destinataire | Le parser ? | Sait dire `pass` / `not-evaluated` ? |
|---|---|---|---|
| `table` (défaut) | un humain, dans un terminal | **non** | non, mais la ligne de verdict le dit en toutes lettres |
| `json` | un pipeline, un tableau de bord | oui, gelé | non : les écarts seulement |
| `assessment` | une chaîne de conformité, un auditeur | oui, gelé | **oui** |
| `oscal` | un outil GRC qui ingère de l'OSCAL | oui, standard | oui, via les observations |
| `sarif` | GitHub Code Scanning, un IDE | oui, standard | non : les écarts seulement |

Le code de sortie est le même dans les cinq cas : le format change ce qui est écrit, jamais le
verdict. Voir [Codes de sortie](exit-codes.fr.md).

## Quelles formes sont gelées

<!-- pepin:gen surface-versions -->
| Surface | Ce qui est gelé | Version |
|---|---|:-:|
| `cli` | verbes, drapeaux et codes de sortie | **v4** |
| `findings` | forme de `--format json` (`findings` + `summary`) | **v1** |
| `assessment` | forme du document `--format assessment` | **v1** |
| `bundle` | forme du bundle de preuve (fichiers, rôles, manifest) | **v3** |
| `inventory` | forme de l'inventaire normalisé (enveloppe, ressource, types et attributs) | **v3** |
<!-- /pepin:gen surface-versions -->

« Gelé » signifie qu'un test échoue quand la forme bouge sans que sa version ait suivi : les
chemins de champs et les types JSON sont la promesse, les valeurs non. `table` est absent de
cette liste à dessein : sa mise en page est faite pour évoluer avec le rapport terminal, ce qui
est précisément la raison de ne rien en parser.

## `table` : le défaut, pour les humains

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

Le bandeau et l'avertissement de portée partent sur la **sortie d'erreur**, le rapport sur la
sortie standard. La couleur est retirée quand la sortie n'est pas un terminal, ou quand
`NO_COLOR` est posé. Une lecture commentée, ligne à ligne, de cette sortie est dans
[Lire un scan](../getting-started/understanding-a-scan.fr.md).

## `json` : les écarts et leur décompte

Forme gelée : `{"findings": [...], "summary": {...}}`.

<!-- pepin:gen format-json-summary -->
```json
{
  "conforme": false,
  "critical": 1,
  "high": 7,
  "low": 1,
  "medium": 1,
  "total": 10
}
```
<!-- /pepin:gen format-json-summary -->

<!-- pepin:gen format-json-finding -->
```json
{
  "code": "CLD-STO-1",
  "labels": {
    "category": "security",
    "check": "objectstorage_bucket_public_access",
    "provider": "scaleway",
    "tf_file": "main.tf",
    "tf_line": "81"
  },
  "message": "Bucket « scaleway_object_bucket_acl.backups » accessible publiquement (ACL publique).",
  "remediation": "Rendre le bucket privé (ACL private, retrait du grant AllUsers, suppression de la policy publique) ; servir via des URLs pré-signées si nécessaire.",
  "severity": "critical",
  "subject": "scaleway_object_bucket_acl.backups",
  "title": "Stockage objet exposé publiquement"
}
```
<!-- /pepin:gen format-json-finding -->

`code` est l'identifiant normatif (`CLD-*`), et `labels.check` l'identifiant de check
agnostique, commun à tous les fournisseurs. Les deux sont stables d'une langue à l'autre ;
`title`, `message` et `remediation` sont traduits.

Trois clés additives se tiennent à côté de `findings` et `summary`, et elles existent pour
qu'un pipeline lise ce qu'il accepte au lieu de le déduire :

- `config` : **toujours présente** — l'empreinte de la configuration effective des contrôles,
  la configuration elle-même, et, quand un réglage sort d'une contrainte normative, les
  `relaxations` et les `dropped_references` qu'elles coûtent. Un scan par défaut n'est jamais
  muet sur sa politique : un lecteur doit pouvoir vérifier qu'un scan a bien tourné sous les
  réglages attendus, pas seulement constater qu'il n'a rien dit. Voir
  [Configurer les contrôles](../guides/control-configuration.fr.md) ;
- `exemptions` : présente quand `--exceptions` (ou `--policy`) a fourni des dérogations
  (voir ci-dessous) ;
- `collection` : présente quand Pépin a **mesuré** l'inventaire — l'état de chaque unité de
  collecte, et les types de ressources que la source portait sans qu'aucune spec ne les
  projette. Sa forme est décrite dans
  [l'inventaire normalisé](inventory.fr.md#létat-de-collecte).

Un finding de détection de secret porte en outre `labels.confidence` (`high` | `medium` |
`low`), pour qu'un pipeline trie les heuristiques génériques à part des secrets confirmés sans
changer le scan. La valeur détectée, elle, n'apparaît jamais, quel que soit le niveau.

**Ce que ce format ne sait pas dire** : il liste des écarts, donc un tableau `findings` vide
signifie « aucun écart trouvé », ce qui recouvre aussi bien « le tenant est propre » que « rien
n'a été collecté ». Le booléen `summary.conforme` ne dit rien non plus de la couverture. Si
cette distinction compte, et pour une porte de conformité elle compte, lire le code de sortie
(3 signifie que le scan n'établit pas la conformité : rien de mesuré, ou un périmètre lu en
partie seulement), lire `collection`, ou utiliser le format assessment.

### `config` : toujours présente

```json
{
  "config": {
    "policy_digest": "sha256:…",
    "effective": { "tagging": { "…": "…" }, "snapshots": { "…": "…" }, "secrets": { "…": "…" } },
    "relaxations": [
      {
        "control": "blockstorage_volume_snapshots_exist",
        "parameter": "snapshots.max_age_days",
        "constraint": "au_plus_le_defaut",
        "default": "7 jours",
        "effective": "90 jours",
        "dropped_references": ["scsl:CLD-STO-3", "cis-v8:11.2", "iso-27001:A.8.13", "secnumcloud-3.2:12.5"]
      }
    ],
    "dropped_references": ["cis-v8:11.2", "iso-27001:A.8.13", "scsl:CLD-STO-3", "secnumcloud-3.2:12.5"]
  }
}
```

`relaxations` et `dropped_references` sont absentes sous le profil par défaut. Quand elles
apparaissent, les résultats correspondants du format `assessment` ont **perdu leurs
`references`** et portent `labels.config_relaxed` : le contrôle a bien été évalué, mais contre
une barre plus basse que l'exigence qu'il citait, donc il ne prétend plus la couvrir.

### `exemptions` : présent seulement avec `--exceptions`

Quand un scan reçoit un fichier de dérogations, le document porte une troisième clé de premier
niveau, à côté de `findings` et `summary` :

```json
{
  "exemptions": {
    "policy_digest": "sha256:…",
    "exceptions": [ { "control": "…", "justification": "…", "expires_at": "…", "owner": "…", "approved_by": "…" } ],
    "records": [ { "control": "…", "effect": "applied", "subjects": ["…"] } ]
  }
}
```

`effect` vaut `applied`, `expired` ou `orphan`. Rien n'est retiré de `findings` ni de `summary`
au motif d'une dérogation : une exemption déplace le **code de sortie**, jamais le dossier. Un
pipeline qui veut refuser toute dérogation vérifie que `exemptions` est absent.

## `assessment` : le document typé et opposable

C'est le format bâti pour une chaîne de conformité : une entrée par contrôle, avec un statut
typé, une preuve, les références normatives et la provenance de l'exécution.

<!-- pepin:gen assessment-counts -->
| Statut | Nombre |
|---|---:|
| `pass` | 6 |
| `fail` | 10 |
| `not-applicable` | 2 |
| `not-evaluated` | 9 |
<!-- /pepin:gen assessment-counts -->

Quatre statuts mesurés, et les deux que les autres formats ne savent pas exprimer sont
`not-applicable` (le contrat du fournisseur déclare le contrôle non testable, avec sa
justification) et `not-evaluated` (Pépin n'a pas pu décider, et dit sur quoi il a buté). Un
cinquième, `exempted`, apparaît quand `--exceptions` fournit une dérogation ; il n'est jamais
une conformité. Leur sens exact est dans
[Le modèle d'assessment](../concepts/assessment-model.fr.md).

`evidence.attribute` et `evidence.source` portent la **provenance de l'attribut décisif**
quand Pépin en a une : quel attribut le contrôle lit, d'où sa valeur vient (`api:` suivi de la
requête réellement servie, `terraform-plan:` suivi du type de ressource du plan, ou `derived:`
suivi de l'élément de descripteur), et sur combien de ressources la source la portait vraiment
(`observed=n/m`). Une source `derived:` est le nom honnête d'une valeur qu'aucun appel d'API
n'a produite.

Un résultat :

<!-- pepin:gen assessment-fail -->
```json
{
  "control": "objectstorage_bucket_public_access",
  "evidence": {
    "attribute": "acl",
    "observed": "Bucket « scaleway_object_bucket_acl.backups » accessible publiquement (ACL publique).",
    "proves": [
      "",
      "",
      ""
    ],
    "source": "acl=terraform-plan:scaleway_object_bucket + terraform-plan:scaleway_object_bucket_acl observed=2/2"
  },
  "labels": {
    "category": "security",
    "provider": "scaleway",
    "tf_file": "main.tf",
    "tf_line": "81"
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

Et l'enveloppe de provenance que porte chaque document :

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

Les champs volatils sont marqués comme tels ici parce qu'ils changent à chaque exécution ; dans
un document réel, ils portent l'horodatage du scan, l'empreinte de build du binaire et
l'empreinte des règles, descripteurs et référentiel qui ont produit le verdict. C'est ce
document que `scan --seal` écrit dans un bundle de preuve, et que `verify --re-derive` rejoue :
voir [Le bundle de preuve](../guides/evidence-bundles.fr.md).

## `oscal` : des résultats d'évaluation pour un outil GRC

Le même assessment, rendu en document OSCAL 1.1.2 `assessment-results`.

<!-- pepin:gen format-oscal-head -->
```json
{
  "assessment-results": {
    "uuid": "<uuid>",
    "metadata": {
      "title": "pepin assessment of scaleway",
      "last-modified": "<timestamp>",
      "version": "<version>",
      "oscal-version": "1.1.2",
      "props": [
        {
          "name": "tool-name",
          "value": "pepin",
          "ns": "https://github.com/stephrobert/scankit/ns/oscal"
        },
        {
          "name": "tool-version",
          "value": "<version>",
          "ns": "https://github.com/stephrobert/scankit/ns/oscal"
        },
        {
          "name": "tool-digest",
          "value": "<provenance>",
          "ns": "https://github.com/stephrobert/scankit/ns/oscal"
        },
[…]
```
<!-- /pepin:gen format-oscal-head -->

À utiliser quand le consommateur est une plateforme GRC qui ingère déjà de l'OSCAL : le format
est un standard du NIST, pas une invention de Pépin, et la projection est faite par `scankit`,
partagé avec les autres outils de la même famille. Rappel : les correspondances normatives sont
**indicatives**. Un document OSCAL produit par Pépin décrit la configuration d'un tenant, pas
une qualification du fournisseur de cloud
([Périmètre et non-objectifs](../concepts/scope.fr.md)).

## `sarif` : pour GitHub Code Scanning

SARIF 2.1.0, le format que lit l'onglet Code Scanning de GitHub. C'est celui à téléverser avec
`github/codeql-action/upload-sarif`.

<!-- pepin:gen format-sarif-head -->
```json
{
  "$schema": "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/sarif-2.1/schema/sarif-schema-2.1.0.json",
  "runs": [
    {
      "results": [
        {
          "level": "error",
          "locations": [
            {
              "physicalLocation": {
                "artifactLocation": {
                  "uri": "main.tf"
                },
                "region": {
                  "startLine": 81
                }
              }
            }
          ],
          "message": {
            "text": "Bucket « scaleway_object_bucket_acl.backups » accessible publiquement (ACL publique)."
          },
[…]
```
<!-- /pepin:gen format-sarif-head -->

Un résultat :

<!-- pepin:gen format-sarif-result -->
```json
{
  "level": "error",
  "locations": [
    {
      "physicalLocation": {
        "artifactLocation": {
          "uri": "main.tf"
        },
        "region": {
          "startLine": 81
        }
      }
    }
  ],
  "message": {
    "text": "Bucket « scaleway_object_bucket_acl.backups » accessible publiquement (ACL publique)."
  },
  "ruleId": "CLD-STO-1"
}
```
<!-- /pepin:gen format-sarif-result -->

Deux choses à savoir avant de le brancher :

- **Les localisations désignent le bloc `.tf` quand Pépin a pu le mesurer.** Sur un plan
  Terraform dont les sources sont posées à côté, `artifactLocation.uri` est le fichier `.tf` et
  `region.startLine` la ligne du bloc `resource` : une alerte Code Scanning se pose sur la
  déclaration à corriger. Quand l'origine n'a pas pu être établie — un plan transmis sans ses
  sources, un module tiré d'un registre, un en-tête de bloc ambigu — la localisation retombe sur
  le fichier scanné, sans `region`, et rien n'est deviné. Sur une collecte live, la notion
  n'existe pas du tout ; voir
  [d'où vient un finding](../concepts/terraform-vs-live.fr.md#doù-vient-un-finding).
- **`level`** est dérivé de la sévérité : `error` pour `critical` et `high`, `warning` pour
  `medium`, `note` pour `low`.

L'étape de téléversement et les permissions qu'elle demande sont dans
[GitHub Actions](../guides/github-actions.fr.md#publier-les-écarts-dans-code-scanning).

## Formats et langue

Les codes, les identifiants de check, les sévérités et les statuts sont identiques en français
et en anglais. Les titres, messages, remédiations et preuves sont traduits, dans `json`,
`assessment`, `oscal` et `sarif` pareillement, puisque la prose voyage avec le finding.

Deux conséquences pour un pipeline :

1. **Comparer le texte d'un rapport entre deux exécutions exige d'épingler `PEPIN_LANG`.**
   Sinon, un runner dont le `LANG` change produira un diff que personne n'a causé.
2. **Se brancher sur `code`, `labels.check`, `status` et `severity` n'exige d'épingler rien du
   tout.**

## Pour aller plus loin

- [L'inventaire normalisé](inventory.fr.md) : la forme dont tous ces documents dérivent, et sa
  version gelée.

- [Codes de sortie](exit-codes.fr.md) : sur quoi bloquer.
- [Le modèle d'assessment](../concepts/assessment-model.fr.md) : ce que chaque statut affirme.
- [Le bundle de preuve](../guides/evidence-bundles.fr.md) : sceller l'assessment et l'OSCAL.
- [Référence de la CLI](cli.fr.md) : `--format` et ses voisins.

## Comment cette page reste vraie

Chaque document ci-dessus a été produit en exécutant le binaire sur la fixture de plan
Terraform du dépôt, et capturé par `internal/docgen`. Horodatages, UUID, empreintes et version
de build sont remplacés par des marqueurs explicites, parce qu'ils changent à chaque exécution
et feraient sinon diverger cette page sans qu'aucun comportement n'ait bougé.
`mise run gen-docs` réécrit les blocs ; `TestGeneratedDocsAreUpToDate` échoue quand la page
committée ne correspond plus à ce que le binaire produit.

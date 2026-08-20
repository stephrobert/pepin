> [🇬🇧 English](remediation.md) · 🇫🇷 Français

# Corriger ce que Pépin trouve

Un scanner qui se contente de nommer les problèmes déplace le travail, il ne le réduit
pas. Cette page décrit comment Pépin passe de *détecter* à *agir* : ce que chaque écart
porte déjà, ce à quoi une remédiation doit répondre, comment la correction est
**vérifiée** plutôt que supposée, et jusqu'où vont exactement les preuves déployables
aujourd'hui.

## Chaque écart porte déjà sa remédiation

Un écart n'est jamais émis sans la phrase qui dit quoi en faire. Ce n'est pas une
habitude éditoriale, c'est une porte : `TestEveryFindingCarriesRemediation`
(`referentiel/validate_test.go`, exécuté par `mise run validate`) refuse toute règle
dont la `remediation` manque, et exige au passage ses contreparties anglaises
(`labels.message_en`, `labels.remediation_en`). Un rapport anglais ne doit jamais
retomber en français au milieu d'une phrase.

Le référentiel suit la même règle : chaque contrôle porte `remediation` et
`remediation_en` dans `referentiel/controles.yaml`. C'est ce texte que le rapport
imprime, et que chaque [page de contrôle](../controls/index.fr.md) cite.

La remédiation **textuelle** est donc un problème résolu, et couvert par un test. Ce qui
suit porte sur la couche du dessus : la *preuve déployable*.

## Les quatre questions auxquelles une remédiation doit répondre

| Question | Où vit la réponse |
|---|---|
| **Pourquoi c'est un risque** | la description du contrôle, tirée du référentiel |
| **Comment enquêter** | le type de ressource normalisé lu, et l'attribut dont la décision dépend |
| **Comment corriger** | le texte de remédiation du référentiel, plus le montage déployable quand il existe |
| **Comment vérifier** | la commande `pepin scan` exacte, et le statut qui doit changer |

Chaque page de contrôle répond aux quatre, et rien n'y est écrit à la main : les pages
sont générées depuis le référentiel, les descripteurs de fournisseurs et le verrou du
« pass ». Le point d'entrée est le
[catalogue des contrôles](../controls/index.fr.md).

## La boucle, mesurée

Une remédiation non vérifiée est une affirmation. Pépin ferme la boucle avec la commande
qui a trouvé le problème, sur la même nature d'entrée. Les deux extraits ci-dessous sont
**capturés par exécution réelle** sur les plans d'exemple du dépôt : celui qui est
volontairement mal configuré, puis le même module corrigé.

Avant, sur `examples/scaleway/terraform/plan.json` :

<!-- pepin:gen remediation-before -->
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
<!-- /pepin:gen remediation-before -->

Après, sur `examples/scaleway/terraform-fixed/plan.json` :

<!-- pepin:gen remediation-after -->
```json
{
  "control": "objectstorage_bucket_public_access",
  "evidence": {
    "attribute": "acl",
    "observed": "aucune non-conformité détectée sur les ressources de type « object_storage_bucket » collectées (contrat vérifié)",
    "proves": [
      "",
      "",
      ""
    ],
    "source": "acl=terraform-plan:scaleway_object_bucket + terraform-plan:scaleway_object_bucket_acl observed=2/2"
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
  "severity": "critical",
  "status": "pass",
  "subject": "scaleway",
  "title": "Stockage objet exposé publiquement"
}
```
<!-- /pepin:gen remediation-after -->

Trois choses ont changé, et les trois comptent :

- `status` est passé de `fail` à `pass`.
- `subject` est passé de la ressource fautive au périmètre du fournisseur : il n'y a
  plus de ressource à nommer.
- `evidence.observed` dit **pourquoi** le « pass » est affirmé, et nomme le verrou
  franchi : le contrat est vérifié, donc l'absence d'écart vaut conformité mesurée, et
  non absence de mesure.

Ce dernier point est toute la différence entre une correction et un angle mort. Si le
statut était passé de `fail` à `not-evaluated`, rien n'aurait été démontré : l'écart
aurait simplement cessé d'être visible. Voir
[le modèle d'assessment](../concepts/assessment-model.fr.md) pour ce que chaque statut
affirme, et [les codes de sortie](../reference/exit-codes.fr.md) pour ce qu'un pipeline
en lit.

## Ce qu'est une preuve déployable

Une preuve de remédiation est un **module Terraform autonome et conforme** : il se
déploie tel quel, dans le vocabulaire natif du fournisseur, et un scan de ce montage ne
déclenche pas la règle. C'est le miroir des fixtures non conformes de
`examples/<fournisseur>/terraform/`.

```
references/remediation/<fournisseur>/<code>/      # module Terraform autonome (préféré)
references/remediation/<fournisseur>/<code>.md    # note ancrée sur la doc officielle, si Terraform n'est pas pertinent
```

Quatre règles, aucune cosmétique :

1. **Un dossier par règle.** Terraform fusionne tous les `.tf` d'un même dossier :
   plusieurs preuves au même endroit deviendraient indéployables et inauditables
   séparément. Un montage qui satisfait des règles voisines est dupliqué, chaque copie
   cadrée sur sa règle.
2. **Déployable, et vérifié comme tel.** `terraform init -backend=false` puis
   `terraform validate` doivent passer : configuration complète, variables déclarées,
   champs réels du schéma. Un extrait inapplicable est du pseudo-code, et du pseudo-code
   présenté comme une configuration est exactement ce que ce dépôt refuse.
3. **Un en-tête obligatoire** nommant le `code`, l'exigence SCSL, le fournisseur,
   **pourquoi le montage est conforme**, et la **source ancrée** (une page de
   `references/docs/<fournisseur>/`, ou le contrat de `providers/<fournisseur>.yaml`).
4. **Aucun secret.** Les identifiants viennent de l'environnement ou de variables,
   jamais du fichier.

La convention complète vit dans
[`references/remediation/README.md`](../../references/remediation/README.md).

## La couverture d'aujourd'hui, et elle est partielle

Le compte ci-dessous est calculé depuis le dépôt au moment de la génération, sur les
couples (contrôle, fournisseur) que le référentiel déclare réellement :

<!-- pepin:gen remediation-coverage -->
| Fournisseur | Preuves de remédiation |
|---|---:|
| exoscale | 26 / 26 |
| kubernetes | 0 / 4 |
| outscale | 0 / 40 |
| scaleway | 0 / 25 |
| **Total** | **26 / 95** |
<!-- /pepin:gen remediation-coverage -->

Ce tableau compte des **preuves déployables**, pas des remédiations. La remédiation
textuelle est à 100 % et tenue par un test ; les preuves sont un chantier, et l'honnête
est de le dire plutôt que d'arrondir.

Pour reproduire le chiffre, et obtenir la liste des manquantes par fournisseur :

```bash
mise run check-remediation                     # tous les fournisseurs
python3 scripts/check-remediation.py exoscale  # un seul
```

## La porte, maintenant qu'un fournisseur est complet

`mise run check-remediation` reste volontairement **découplé** de `mise run validate`.
Tous fournisseurs confondus, il est encore rouge, et une porte rouge en permanence est
une porte qu'on apprend à ignorer : une porte de qualité qu'on contourne par habitude
est pire que pas de porte, parce qu'elle enseigne le contournement.

Ce qui a changé, c'est que la condition de rebranchement écrite à côté de la tâche dans
`mise.toml`, **un fournisseur à 100 %**, est désormais tenue. Chaque contrôle déclaré
par exoscale porte sa preuve, et cet acquis est gardé par un test plutôt que par une
bonne intention : `TestExoscaleRemediationCoverageStaysComplete` (`internal/docgen`)
échoue dès qu'un contrôle exoscale arrive sans la sienne, en nommant ce qui manque. Il
tourne avec `mise run test`, donc il siège dans la porte de release.

Les autres fournisseurs restent hors de cette garde tant que leur couverture est
partielle, et chacun l'intègre le jour où il atteint 100 %. En attendant, leur chiffre
est publié, ci-dessus et sur chaque page de contrôle, plutôt qu'imposé.

## Déposer une preuve

1. Ancrer le montage d'abord : le champ, son type et ses valeurs acceptées viennent du
   SDK ou de la documentation officielle du fournisseur, jamais de mémoire. Mettre la
   page en cache sous `references/docs/<fournisseur>/` (`mise run fetch-docs`) et la
   citer.
2. Écrire `references/remediation/<fournisseur>/<code>/main.tf` avec l'en-tête
   obligatoire.
3. Vérifier qu'il se déploie : `terraform init -backend=false && terraform validate`
   dans ce dossier.
4. Vérifier qu'il est **conforme** : en produire un plan et le scanner, la règle ne doit
   pas se déclencher.

   ```bash
   terraform plan -out tfplan && terraform show -json tfplan > plan.json
   ./pepin scan <fournisseur> --terraform plan.json --format assessment
   ```

5. Régénérer la documentation dérivée : `mise run gen-docs`. La page du contrôle lie
   désormais votre preuve, et les chiffres de couverture bougent.
6. `mise run validate` et `mise run test` restent au vert.

## Voir aussi

- [Catalogue des contrôles](../controls/index.fr.md) : les quatre questions, contrôle
  par contrôle.
- [Le modèle d'assessment](../concepts/assessment-model.fr.md) : ce qu'un `pass` affirme.
- [Limites connues](../known-limitations.fr.md) : les angles morts, nommés.
- [Ajouter un contrôle](../contributing/adding-a-control.fr.md) : la procédure de bout
  en bout.

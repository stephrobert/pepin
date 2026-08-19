# Preuves de remédiation

> **État : chantier en cours, couverture partielle.** 4 preuves sur 95 attendues
> (exoscale 4/26 · outscale 0/40 · scaleway 0/25 · kubernetes 0/4). Le chiffre à
> jour vient de `mise run check-remediation`. Ce qui suit décrit la **cible** et la
> forme attendue d'une preuve — pas l'état du répertoire. La remédiation *textuelle*,
> elle, est garantie : chaque finding émis en porte une, et un test du référentiel
> le vérifie.

Cible : pour **chaque règle** (`code` agnostique) et **chaque provider** qui
l'implémente, un artefact qui montre **comment être conforme** : de préférence du **Terraform
déployable** (vocabulaire natif du provider), à défaut une **note** pointant la
doc officielle. Ces artefacts sont la matière première de la future **doc de
remédiation** : la règle dit *ce qui ne va pas*, la preuve montre *le bon montage*.

Pendant logique du registre [questions-providers](../questions-providers/) : on ne
documente une remédiation qu'une fois le bon montage **ancré** (doc/SDK, cf. §2).

## Structure

```
references/remediation/<provider>/<code>/      # module Terraform AUTONOME (préféré)
references/remediation/<provider>/<code>.md    # note + doc, si pas de Terraform pertinent
```

**Un dossier = une preuve = un module Terraform autonome** (`<code>/main.tf` avec son
bloc `terraform`/`provider` et toutes ses variables). Terraform fusionne tous les
`.tf` d'un même dossier : mettre plusieurs preuves dans un seul dossier rendrait les
ressources indéployables/inauditables séparément — d'où **un dossier par règle**.

Le `code` est commun à tous les providers, mais le contenu est **propre au provider**
(ressources `exoscale_*`, `outscale_*`, `scaleway_*` ; vocabulaire natif). Un même
montage peut satisfaire plusieurs règles voisines (un `exoscale_iam_role` bien
configuré couvre `iam_role_no_admin_privileges`, `_source_ip_restricted` et
`_key_lifetime_bounded`) : on duplique alors le module, chacun cadré sur sa règle.

## Exemples fonctionnels et complets

Chaque module DOIT être **déployable tel quel** : `terraform init -backend=false`
puis `terraform validate` passent (configuration complète, variables déclarées,
champs requis du schéma réel présents). Le montage est CONFORME — scanné par Pépin,
il ne déclenche pas la règle visée (cas ✓, miroir des fixtures non conformes de
`examples/<provider>/terraform/`).

## En-tête obligatoire

Chaque fichier rappelle, en commentaire de tête : le `code`, l'exigence SCSL
(`CLD-*`), le provider, **pourquoi c'est conforme** (ce que la règle vérifie), et la
**source** ancrée (page `references/docs/<provider>/…` et/ou contrat
`providers/<provider>.yaml`).

## Statut de couverture

`mise run check-remediation` liste, par provider, les règles **sans** preuve de
remédiation, et rend 1 s'il en manque.

Ce contrôle est **volontairement découplé** de `mise run validate` : le rebrancher
aujourd'hui rendrait la porte de qualité rouge en permanence, ce qui la ferait
ignorer. Condition de rebranchement (`depends` dans `mise.toml`) : couverture à
100 % pour au moins un provider complet, puis pour les autres au fil de l'eau. Le miroir non conforme de ces
montages vit dans `examples/<provider>/terraform/` (fixtures de test).

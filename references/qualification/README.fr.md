> [🇬🇧 English](README.md) · 🇫🇷 Français

# Tenants de qualification

Un **tenant de qualification** est une stack Terraform, écrite par nous,
**délibérément mal configurée**, réellement **appliquée** sur un compte cloud,
scannée, scellée, **détruite**, puis comparée à un résultat attendu committé. C'est
l'étape 3 de la porte de release (issue #178).

C'est le miroir d'un [tenant de référence](../tenants/) : un tenant de référence est
une configuration tierce sur laquelle Pépin doit rester **muet** (c'est là qu'un faux
positif se voit) ; un tenant de qualification porte des **mauvaises pratiques
connues**, une ressource par contrôle et un contre-exemple par contrôle, et le scan
doit les trouver **toutes, et rien d'autre**, à travers la chaîne entière : API
réelle → collecteur → règles → assessment → bundle → verdict.

```
references/qualification/
  README.md · README.fr.md          cette page
  <fournisseur>/
    *.tf                            la stack : une faute par ressource, un contre-exemple par contrôle
    .terraform.lock.hcl             versionné : le binaire de provider qui a été qualifié
    tenant.yaml                     métadonnées : région, tag du tenant, version épinglée, budget, tarifs horaires
    expected.yaml                   le contrat : contrôle × source × sujet → statut, et le compte épinglé
    hooks.py                        crochets du fournisseur : identité, variables, extra (ce que Terraform ne sait pas créer), inventaire par famille, nettoyage de secours
    README.md · README.fr.md        ce que la stack crée, ce qui bloque destroy en amont, résultats mesurés
tools/qualification/qualify.py      le runner générique (apply, scan, scelle, vérifie, détruit, prouve, compare)
```

## Ce qu'un run fait

```bash
PEPIN_GATE_LIVE=1 mise run qualify            # PROVIDER=scaleway par défaut
mise run qualify:plan                         # le même tenant, sans rien créer : plan + scan --terraform
mise run qualify:compare                      # recompare le dernier run à expected.yaml
mise run qualify:selftest                     # la comparaison se prouve sur des cas synthétiques
```

1. **Préflight** : les outils, le lockfile épingle la version de provider que
   `tenant.yaml` déclare, le binaire est compilé depuis l'arbre qu'on qualifie.
2. **Identité** : les identifiants sont ceux du fournisseur (environnement ou son
   fichier de configuration, jamais le dépôt, [ADR-0012](../../docs/adr/0012-aucun-identifiant-en-ci.md)).
   Le runner demande à l'**API** quel compte ils ouvrent, et **refuse de démarrer** si
   le projet et l'organisation ne sont pas ceux qu'`expected.yaml` épingle. Un tenant
   qui ouvre SSH au monde ne s'applique jamais sur une production par erreur.
3. **Inventaire avant** : chaque famille est listée ; un reste d'un run précédent
   refuse le départ.
4. **Deux plans, un apply** : un scan live ne collecte pas tous les types qu'un plan
   porte (sur Scaleway : cinq types en live, huit sur un plan). Le plan **complet** est
   celui scanné avec `--terraform` ; ce qui est **appliqué** est le sous-ensemble qu'un
   scan live collecte, parce que provisionner le reste coûterait de l'argent qu'aucun
   scan live ne mesure. `expected.yaml` épingle les deux sources séparément, et les
   variables du tenant disent quelles ressources sont « plan seulement ».
5. **`pepin scan --live`** dans tous les formats (`table`, `json`, `assessment`,
   `oscal`, `sarif`), scellé par `--seal` ; `verify --re-derive` doit passer, et un
   bundle altéré d'un octet doit être refusé ([ADR-0018](../../docs/adr/0018-ce-quun-bundle-prouve-de-lui-meme.md)).
6. **`pepin scan --terraform`** sur le même plan.
7. **Destroy**, dans un `finally` : il tourne même quand une étape précédente a
   échoué. Ce qu'il faut défaire d'abord (une protection contre la suppression, que le
   provider Outscale ne sait pas traverser, #88) est appliqué avant lui (`tenant.yaml`
   `pre_destroy_vars`) ; ce que le crochet `extra` a créé hors Terraform est retiré
   avant lui aussi. Si `terraform destroy` ne finit pas, les crochets du fournisseur
   suppriment par l'API ce qui porte le tag du tenant, et destroy est relancé.
8. **Preuve de destruction** : `Destroy complete!` n'est pas une preuve. Le compte est
   relisté famille par famille, filtré sur le tag et le préfixe de nom du tenant,
   **et** comparé à l'inventaire pris avant apply : ce qui est apparu entre les deux,
   étiqueté ou non (un volume racine renommé, une sauvegarde automatique), est un
   reste et un NO-GO. Les suppressions sont asynchrones chez plus d'un fournisseur :
   le listing est répété jusqu'au vide, bornée (`settle_seconds`, 240 s) — ce qui
   survit au délai est un reste, et le nettoyage de secours est tenté dessus.
9. **Comparaison** à `expected.yaml`, puis **falsification** : une attente est cassée
   en mémoire et la comparaison doit dire NO-GO. Une porte qu'on n'a jamais vue rouge
   ne garde rien.

La sortie va dans `release-gate/` (ignoré par git) : `qualification-<fournisseur>.json`
(verdict, preuves, durées, coût estimé) et le dossier du run avec chaque artefact.
Ces artefacts portent les identifiants de ressources d'un compte réel : rien n'en
entre au dépôt sans relecture.

## Ce que NO-GO veut dire

| Différence | Ce que c'est |
|---|---|
| un `fail` attendu qui manque | un **faux vert** : la faute est là et le scan ne l'a pas vue |
| un `fail` sur un sujet du tenant que rien n'attend | un **faux positif** |
| le contre-exemple parle | la règle se déclenche mais **ne discrimine pas** |
| un `not-evaluated` devenu `pass` | un **pass que personne n'a prouvé** ([ADR-0006](../../docs/adr/0006-jamais-un-pass-non-prouve.md)) : le collecteur n'a pas changé, la donnée manque toujours |
| un `pass` ou un `fail` devenu `not-evaluated` | une **régression de couverture** |
| un contrôle absent d'`expected.yaml` | un verdict non épinglé |
| un code de sortie autre que celui épinglé | la porte de CI elle-même a bougé ([ADR-0005](../../docs/adr/0005-codes-de-sortie.md)) |
| une ressource qui survit au destroy | le seul résultat inacceptable de tout l'exercice |

Trois mots de plus qu'`expected.yaml` peut employer : `inconclusive:` liste les sujets
du tenant sur lesquels une règle dit ne pas savoir conclure (un `not-evaluated` à sujet,
ADR-0015) ; `known_defect:` épingle un comportement mesuré et faux, avec l'issue qui le
suit — la porte reste GO et la correction la fait rougir sciemment ; `evaluated` accepte
`pass` ou `fail` pour un contrôle dont les sujets sont hors du tenant (les utilisateurs
du compte, sa politique d'accès API). Un tenant sans compte dit `live: unavailable` dans
`tenant.yaml` : il ne tourne qu'en `--plan-only`, et un run réel est refusé.

`expected.yaml` n'est **pas une matrice verte** ([ADR-0010](../../docs/adr/0010-dette-de-veracite-comptee.md)) :
chacun de ses `not-evaluated` est une **dette de collecte nommée**, et il change quand,
et seulement quand, le collecteur apprend à lire la donnée. Un tel changement déplace
un verdict sur un tenant inchangé : il a sa ligne au CHANGELOG.

## Ajouter un fournisseur

1. Copier la structure de `scaleway/` ; garder les mêmes noms de fichiers, le runner
   s'y adosse (`tenant.yaml`, `expected.yaml`, `hooks.py`).
2. **Lire d'abord les issues Terraform du fournisseur sur `destroy`**, et écrire ce
   qu'on a trouvé dans le README du fournisseur, avec les liens : un type de ressource
   connu pour bloquer un destroy est évité, ou son nettoyage manuel est automatisé dans
   `hooks.py cleanup`.
3. Une ressource par contrôle, un contre-exemple par contrôle. Étiqueter **toute**
   ressource étiquetable avec le tag du tenant, et préfixer chaque nom : c'est sur eux
   que la preuve de destruction filtre.
4. Épingler la version du provider à l'exact, et committer le lockfile.
5. Écrire `expected.yaml` depuis un run `--plan-only` d'abord (source Terraform), puis
   depuis un run réel (source live), et **justifier chaque statut** par la règle et le
   collecteur : un statut s'épingle parce qu'il est juste, pas parce qu'il a été mesuré.
6. Écrire l'épinglage du compte, puis lancer `PEPIN_GATE_LIVE=1 mise run qualify` et
   lire la preuve de destruction avant toute autre chose.

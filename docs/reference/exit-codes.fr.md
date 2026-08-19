> [🇬🇧 English](exit-codes.md) · 🇫🇷 Français

# Codes de sortie et porte de CI

Une porte de CI lit un seul nombre. Les codes de sortie de Pépin sont donc un **contrat
d'intégration** : ils sont gelés dans `cmd/testdata/frozen/cli.json`, testés un par un, et
changer leur signification est une rupture qui a sa propre ligne de CHANGELOG.

<!-- pepin:gen cli-exit-codes -->
| Code | Constante (`cmd/surface.go`) | Signification |
|:-:|---|---|
| **0** | `conforme` | aucun écart critical/high, et au moins un contrôle réellement mesuré |
| **1** | `non_conformite` | au moins un écart critical ou high |
| **2** | `erreur` | erreur technique : le scan n'a pas pu conclure |
| **3** | `strict` | rien n'a été mesuré (sans `--strict`), ou écarts medium/low restants avec `--strict` |
<!-- /pepin:gen cli-exit-codes -->

La distinction dont dépend toute cette page : **1 et 3 sont des verdicts sur un tenant, 2 est
une panne de la mesure elle-même.** Un pipeline peut légitimement décider de rapporter un
verdict sans bloquer dessus. Il ne peut jamais le faire avec 2 : une erreur technique avalée
rapporte une posture que personne n'a mesurée.

## Un tableau, six situations, six exécutions réelles

Chaque commande de ce tableau a été exécutée par le générateur de documentation, et le code de
la dernière colonne est celui que le processus a rendu.

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

Les deux dernières lignes sont le même inventaire, avec et sans `--strict`. Les deux qui les
précèdent portent la distinction la plus importante, et c'est l'objet de la suite.

## `0` : conforme

Aucun écart critical ou high, **et** au moins un contrôle réellement mesuré.

<!-- pepin:gen exit-run-clean -->
```console
$ ./pepin scan scaleway --terraform examples/scaleway/terraform-fixed/plan.json
[…]
 Summary

 Verdict : conforme sur le périmètre déclaré (plan Terraform, état planifié) (aucune non-conformité détectée, 16 contrôles conformes)

 🔴 CRITICAL 0   🟠 HIGH 0   🟡 MEDIUM 0   🔵 LOW 0
──────────────────────────────────────────────────────────────────────────────
$ echo $?
0
```
<!-- /pepin:gen exit-run-clean -->

La ligne de verdict nomme le périmètre qu'elle couvre. « Conforme » y signifie « aucun écart
détecté sur le périmètre déclaré », jamais « ce tenant est certifié » : voir
[Périmètre et non-objectifs](../concepts/scope.fr.md).

## `1` : non-conformité

Au moins un écart `critical` ou `high`.

<!-- pepin:gen exit-run-nonconformity -->
```console
$ ./pepin scan scaleway --terraform examples/scaleway/terraform/plan.json
[…]
 Summary

 Verdict : NON CONFORME

 🔴 CRITICAL 1   🟠 HIGH 7   🟡 MEDIUM 1   🔵 LOW 1
──────────────────────────────────────────────────────────────────────────────
$ echo $?
1
```
<!-- /pepin:gen exit-run-nonconformity -->

Des écarts medium et low seuls ne produisent **pas** 1. C'est délibéré : une porte qui bloque
sur le moindre écart low est une porte qu'on désactive dans la semaine. Pour les inclure,
demander `--strict`, qui rend 3.

## `2` : erreur technique

Le scan n'a pas pu conclure : fichier illisible, provider inconnu, API injoignable,
identifiants invalides, dossier de règles qui ne compile pas.

<!-- pepin:gen exit-run-error -->
```console
$ ./pepin scan scaleway examples/scaleway/plan-absent.json

 ██████╗ ███████╗██████╗ ██╗ ███╗   ██╗
 ██╔══██╗██╔════╝██╔══██╗██║ ████╗  ██║
 ██████╔╝█████╗  ██████╔╝██║ ██╔██╗ ██║
 ██╔═══╝ ██╔══╝  ██╔═══╝ ██║ ██║╚██╗██║
 ██║     ███████╗██║     ██║ ██║ ╚████║
 ╚═╝     ╚══════╝╚═╝     ╚═╝ ╚═╝  ╚═══╝

 v<version>  · scanner de posture cloud (sécurité · conformité)

erreur : open examples/scaleway/plan-absent.json: no such file or directory
$ echo $?
2
```
<!-- /pepin:gen exit-run-error -->

Le bandeau part sur la sortie d'erreur, et le message d'erreur aussi. **2 n'est jamais un
verdict de posture.** Aucun `allow_failure`, aucun `continue-on-error`, aucun `|| true` ne doit
le couvrir.

## `3` : rien de mesuré, ou la porte stricte

Deux situations partagent ce code, et toutes deux signifient « ne lisez pas cette exécution
comme un feu vert ».

### Rien n'a été mesuré, et `--strict` n'est pas nécessaire

Depuis la v0.1.0, un scan qui n'a mesuré aucun contrôle (hors gouvernance) rend **3**, sans
qu'il faille demander `--strict`.

<!-- pepin:gen exit-run-nothing -->
```console
$ ./pepin scan scaleway empty-inventory.json
[…]
 Summary

 Verdict : INDÉTERMINÉ — aucun contrôle mesuré sur des ressources (le périmètre évalué est vide ou non collecté)

 🔴 CRITICAL 0   🟠 HIGH 0   🟡 MEDIUM 0   🔵 LOW 0
──────────────────────────────────────────────────────────────────────────────
$ echo $?
3
```
<!-- /pepin:gen exit-run-nothing -->

L'inventaire qui le produit est un inventaire valide, et vide :

<!-- pepin:gen fixture-empty-inventory -->
```json
{
  "provider": "scaleway",
  "resources": []
}
```
<!-- /pepin:gen fixture-empty-inventory -->

« Aucun écart » et « rien vu » produisent le même ensemble vide, et pourtant seul le premier
dit quelque chose de la posture. Des identifiants expirés, des droits insuffisants, une région
vide ou un inventaire tronqué rendraient sinon une porte de CI verte sur un périmètre que
personne n'a regardé. La ligne de verdict annonce `INDÉTERMINÉ`, et le code de sortie la suit.

### Il subsiste des écarts medium/low, avec `--strict`

Le même inventaire, deux fois. Sans `--strict`, les écarts medium et low ne bloquent pas :

<!-- pepin:gen exit-run-medium-plain -->
```console
$ ./pepin scan scaleway tagless-inventory.json
[…]
 Summary

 Verdict : aucun écart critique/haut, mais 1 écart(s) medium/low sur le périmètre évalué (3 conformes)

 🔴 CRITICAL 0   🟠 HIGH 0   🟡 MEDIUM 1   🔵 LOW 0
──────────────────────────────────────────────────────────────────────────────
$ echo $?
0
```
<!-- /pepin:gen exit-run-medium-plain -->

Avec `--strict`, ils bloquent :

<!-- pepin:gen exit-run-strict -->
```console
$ ./pepin scan scaleway tagless-inventory.json --strict
[…]
 Summary

 Verdict : aucun écart critique/haut, mais 1 écart(s) medium/low sur le périmètre évalué (3 conformes)

 🔴 CRITICAL 0   🟠 HIGH 0   🟡 MEDIUM 1   🔵 LOW 0
──────────────────────────────────────────────────────────────────────────────
$ echo $?
3
```
<!-- /pepin:gen exit-run-strict -->

`--strict` n'ajoute donc qu'un seul comportement : les écarts medium/low deviennent bloquants.
Il ne crée pas la porte « rien n'a été mesuré », qui existe sans lui.

## Ce qui ne change pas le code de sortie

- **Le format de sortie.** `--format json`, `sarif`, `oscal` ou `assessment` changent ce qui
  est écrit sur la sortie standard, jamais le code rendu. Un pipeline peut donc bloquer sur le
  code et archiver le document.
- **La langue.** `--lang fr` et `--lang en` rendent les mêmes codes sur la même entrée. Codes,
  identifiants, sévérités et statuts sont stables d'une langue à l'autre ; seule la prose est
  traduite. Un pipeline qui compare le **texte** d'un rapport entre deux exécutions doit
  épingler `PEPIN_LANG`, mais un pipeline qui se branche sur `$?` n'a rien à épingler.
- **`--seal`.** Écrire un bundle de preuve ne modifie pas le verdict.

## En shell

```bash
pepin scan scaleway --terraform plan.json
code=$?
case "$code" in
  0) echo "conforme" ;;
  1) echo "non-conformité : au moins un écart critical/high" ; exit 1 ;;
  3) echo "rien mesuré, ou écarts medium/low sous --strict" ; exit 1 ;;
  2) echo "erreur technique : le scan n'a pas pu conclure" ; exit 2 ;;
  *) echo "code inattendu $code" ; exit 2 ;;
esac
```

Ne jamais écrire `pepin scan … || true` : cela efface les trois codes qui portent de
l'information, 2 compris.

## GitHub Actions

L'action publiée implémente déjà ce contrat : elle fait échouer le job sur 1 et 3 (sauf
`fail-on-nonconformity: 'false'`), et toujours sur 2.

```yaml
- name: Pépin, porte de posture
  id: scan
  uses: stephrobert/pepin/.github/actions/pepin-scan@<sha-du-commit>   # épinglé par SHA
  with:
    version: '0.2.0'
    provider: scaleway
    terraform-plan: plan.json
    # fail-on-nonconformity: 'false'   # rapporter le verdict sans bloquer ; 2 échoue toujours
```

Lancer le binaire à la main dans une étape demande le `case` ci-dessus, puisqu'un code non nul
fait déjà échouer l'étape : l'enjeu est de garder 2 distinguable de 1 et 3 dans le résumé du
job. Le pipeline complet est dans [GitHub Actions](../guides/github-actions.fr.md).

## GitLab CI

```yaml
pepin-terraform-plan:
  script:
    - pepin scan scaleway --terraform plan.json --format json > pepin-report.json
  # Rapporter la posture sans bloquer dessus : autoriser les codes de VERDICT, et eux seuls.
  allow_failure:
    exit_codes: [1, 3]
```

`allow_failure: exit_codes: [1, 3]`, **jamais 2**. Y inscrire 2 transforme « le scan n'a pas pu
conclure » en pipeline vert. Le pipeline complet est dans [GitLab CI](../guides/gitlab-ci.fr.md).

## Pour aller plus loin

- [Référence de la CLI](cli.fr.md) : les verbes et drapeaux qui produisent ces codes.
- [Le modèle d'assessment](../concepts/assessment-model.fr.md) : pourquoi `not-evaluated` n'est
  pas un échec, et pourquoi il ne bloque pas.
- [GitHub Actions](../guides/github-actions.fr.md) · [GitLab CI](../guides/gitlab-ci.fr.md).

## Comment cette page reste vraie

Les codes de sortie des tableaux sont lus dans la surface CLI gelée ; chaque bloc console est
une exécution réelle capturée par `internal/docgen`, avec le code que le processus a rendu.
`TestTheGeneratorActuallyRunsTheBinary` vérifie chacun de ces codes indépendamment, et
`TestEveryExitCodeIsDocumented` échoue si un code de la surface gelée n'apparaît jamais ici.

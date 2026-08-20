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
| **3** | `strict` | le scan n'établit pas la conformité : rien n'a été mesuré, ou la collecte n'a pas pu lire tout le périmètre (les deux sans `--strict`), ou écarts medium/low restants avec `--strict` |
| **4** | `derogation` | tout écart critical/high restant est couvert par une dérogation datée et attribuée (`--exceptions`) |
<!-- /pepin:gen cli-exit-codes -->

La distinction dont dépend toute cette page : **1 et 3 sont des verdicts sur un tenant, 2 est
une panne de la mesure elle-même.** Un pipeline peut légitimement décider de rapporter un
verdict sans bloquer dessus. Il ne peut jamais le faire avec 2 : une erreur technique avalée
rapporte une posture que personne n'a mesurée.

## Un tableau, huit situations, huit exécutions réelles

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
| Aucun écart, mais une unité de collecte n'a pas pu être lue | `./pepin scan scaleway partial-inventory.json` | **3** |
| Tout écart critical/high est couvert par une dérogation valide | `./pepin scan scaleway bastion-inventory.json --exceptions exceptions.yaml` | **4** |
| La même dérogation, échue : elle ne s'applique plus | `./pepin scan scaleway bastion-inventory.json --exceptions exceptions-expired.yaml` | **1** |
<!-- /pepin:gen exit-codes -->

Les lignes cinq et six sont le même inventaire, avec et sans `--strict` ; les deux dernières
sont le même inventaire et la même dérogation, valide puis échue. Les deux lignes qui portent
la distinction la plus importante sont la troisième et la quatrième, et c'est l'objet de la suite.

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

## `3` : le scan n'établit pas la conformité

Trois situations partagent ce code, et toutes trois signifient « ne lisez pas cette exécution
comme un feu vert » : rien n'a été mesuré, la collecte n'a pas pu lire tout le périmètre, ou
la porte stricte a rattrapé des écarts medium/low.

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

`--strict` n'ajoute donc que deux comportements, et seulement deux : les écarts medium/low
deviennent bloquants, et une **correspondance normative tombée sous un réglage assoupli**
l'est aussi.

Un réglage assoupli est un réglage qui sort de la contrainte sous laquelle la correspondance
d'un contrôle vaut — une fenêtre de snapshot allongée, une étiquette exigée retirée, un seuil
de détection de secrets relevé. Le contrôle est toujours évalué, mais contre une barre plus
basse que l'exigence qu'il cite, donc il ne prétend plus la couvrir. Un pipeline qui vend de
la conformité ne doit pas rendre `0` sur un contrôle qu'il a lui-même abaissé. Voir
[Configurer les contrôles](../guides/control-configuration.fr.md).
Il ne crée pas la porte « rien n'a été mesuré », qui existe sans lui.

### La collecte n'a pas pu lire tout le périmètre

Une unité de collecte qui répond `403`, qui dépasse son délai ou dont la pagination s'interrompt
laisse une part du périmètre non lue. L'inventaire ci-dessous porte cet état — c'est la forme
que produit une collecte live, et celle que rejoue l'`input.json` d'un bundle scellé :

<!-- pepin:gen fixture-partial-inventory -->
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
        "tags": [
          {"key": "CostCenter", "value": "R-42"},
          {"key": "Project", "value": "pepin"},
          {"key": "Env", "value": "prod"},
          {"key": "Owner", "value": "platform"}
        ]
      }
    }
  ],
  "collection": {
    "units": [
      {
        "unit": "compute_instance",
        "types": ["compute_instance"],
        "attempted": true,
        "complete": true
      },
      {
        "unit": "security_group_rule",
        "types": ["security_group_rule"],
        "attempted": true,
        "complete": false,
        "error": "permission_denied",
        "detail": "HTTP 403 - GET https://api.scaleway.com/instance/v1/zones/fr-par-1/security_groups - insufficient permissions"
      }
    ]
  }
}
```
<!-- /pepin:gen fixture-partial-inventory -->

Le scan annonce ce qu'il a pu et n'a pas pu observer **avant** tout verdict, sur la sortie
d'erreur :

<!-- pepin:gen capability-report -->
```text
Relevé de capacités du collecteur
  ✓ compute_instance
  ✗ security_group_rule — privilège insuffisant du compte de scan
    droit requis : InstancesReadOnly (Project scope)
    HTTP 403 - GET https://api.scaleway.com/instance/v1/zones/fr-par-1/security_groups - insufficient permissions
Résultat : 6 contrôle(s) ne pourront pas être évalués sur ce périmètre.
  · network_securitygroup_allow_ingress_from_internet_to_all_ports
  · network_securitygroup_allow_ingress_from_internet_to_high_risk_tcp_ports
  · network_securitygroup_allow_ingress_from_internet_to_high_risk_udp_ports
  · network_securitygroup_allow_ingress_from_internet_to_tcp_port_22
  · network_securitygroup_allow_ingress_from_internet_to_tcp_port_3389
  · network_securitygroup_unrestricted_egress
```
<!-- /pepin:gen capability-report -->

Chaque contrôle qui lit un type de ressource alimenté par l'unité en échec devient
`not-evaluated`, avec l'unité manquante nommée, et l'exécution ne rend pas `0` :

<!-- pepin:gen exit-run-partial -->
```console
$ ./pepin scan scaleway partial-inventory.json
[…]
 Summary

 Verdict : INCOMPLET — 6 contrôle(s) non évaluables faute d'une collecte complète, 0 écart(s) medium/low sur ce qui a pu être lu

 🔴 CRITICAL 0   🟠 HIGH 0   🟡 MEDIUM 0   🔵 LOW 0
──────────────────────────────────────────────────────────────────────────────
$ echo $?
3
```
<!-- /pepin:gen exit-run-partial -->

Le raisonnement est celui qui vaut à un inventaire vide son `3` : un contrôle qui n'a pas pu
être évalué ne dit rien de la posture, et une porte qui verdit sur un périmètre que personne
n'a regardé est le faux vert que cet outil existe pour empêcher. Un `fail` observé sur la part
qui, elle, a été lue rend toujours **1** : un écart observé reste observé, et l'incomplétude ne
l'efface jamais.

**Pourquoi pas un cinquième code.** Un code propre à l'incomplétude ne pourrait jamais primer
sur `1` : masquer un écart critique réel au motif que le reste manque serait exactement le faux
vert que l'on combat. Il ne s'exprimerait donc que là où `3` s'exprime déjà, et deux codes pour
une seule position dans l'ordre de priorité sont un doublon, pas une distinction — au prix
d'une relecture de son `case $?` pour chaque consommateur, sans nouvelle décision à la clé. Ce
qui sépare les situations reste lisible là où c'est utile : le relevé de capacités nomme
l'unité et la classe d'échec, chaque contrôle touché porte son motif, et `--format json` publie
une clé `collection`.

## `4` : tout écart restant est sous dérogation

Un CSPM utilisé en production doit permettre des exceptions. Sans elles, une équipe désactive
le contrôle, ou cesse de lire l'outil : deux issues pires que l'écart lui-même. Mais une
dérogation ne doit jamais rendre une porte verte en silence, d'où un code qui n'appartient
qu'à elle.

On donne au scan un fichier de dérogations versionné, avec `--exceptions` :

<!-- pepin:gen fixture-exceptions -->
```yaml
exceptions:
  - control: network_securitygroup_allow_ingress_from_internet_to_tcp_port_22
    resource: sg-bastion
    justification: "Bastion administre, acces restreint par IP source en amont"
    expires_at: 2099-12-31
    owner: platform-security
    approved_by: security@example.org
```
<!-- /pepin:gen fixture-exceptions -->

Appliqué à un inventaire dont le seul écart high est celui qu'il couvre :

<!-- pepin:gen fixture-bastion-inventory -->
```json
{
  "provider": "scaleway",
  "resources": [
    {
      "provider": "scaleway",
      "type": "security_group_rule",
      "id": "vm-bastion",
      "name": "vm-bastion",
      "region": "fr-par",
      "attributes": {
        "security_group_id": "sg-bastion",
        "direction": "inbound",
        "action": "accept",
        "protocol": "tcp",
        "port_from": 22,
        "port_to": 22,
        "cidrs": ["0.0.0.0/0"],
        "description": "Acces d administration du bastion"
      }
    }
  ]
}
```
<!-- /pepin:gen fixture-bastion-inventory -->

<!-- pepin:gen exit-run-exempted -->
```console
$ ./pepin scan scaleway bastion-inventory.json --exceptions exceptions.yaml
[…]
  │ CLD-NET-1 │ SSH (port 22) ouvert à Internet │ HIGH │ scaleway │ 1 │
  ╰───────────┴─────────────────────────────────┴──────┴──────────┴───╯
──────────────────────────────────────────────────────────────────────────────
 Summary

 Verdict : NON CONFORME sous dérogation, 1 écart(s) critique/haut tous couverts par une dérogation datée et attribuée

 🔴 CRITICAL 0   🟠 HIGH 1   🟡 MEDIUM 0   🔵 LOW 0
──────────────────────────────────────────────────────────────────────────────

DÉROGATIONS APPLIQUÉES : écarts assumés, NON conformes
  · network_securitygroup_allow_ingress_from_internet_to_tcp_port_22 (sg-bastion)
    Bastion administre, acces restreint par IP source en amont
    jusqu'au 2099-12-31 · responsable platform-security · approuvé par security@example.org
$ echo $?
4
```
<!-- /pepin:gen exit-run-exempted -->

La ligne de verdict se lit telle quelle : **NON CONFORME sous dérogation**. L'écart n'a pas
disparu, il a été écarté. Le statut du contrôle dans `--format assessment` est `exempted`,
jamais `pass` ; le finding reste dans `--format json`, dans le SARIF et dans le décompte par
sévérité ; seule la porte bouge.

Pourquoi 4 plutôt que 0 ou 1. Rendre **0** ferait d'une exemption un faux vert silencieux,
c'est-à-dire exactement ce que le statut `exempted` existe pour empêcher. Rendre **1**
rendrait la dérogation inutile, et une équipe qui ne peut pas déroger à un contrôle finit par
le supprimer. **4** est non nul, donc rien ne passe en silence, et distinct, donc un pipeline
qui choisit de l'accepter doit écrire le chiffre, et par conséquent savoir qu'il existe.

### Une dérogation expirée n'ouvre pas la porte

Le même fichier, avec une date passée. La dérogation ne s'applique plus, l'écart redevient un
écart, et l'expiration est signalée sur la sortie d'erreur :

<!-- pepin:gen exit-run-expired -->
```console
$ ./pepin scan scaleway bastion-inventory.json --exceptions exceptions-expired.yaml

 ██████╗ ███████╗██████╗ ██╗ ███╗   ██╗
 ██╔══██╗██╔════╝██╔══██╗██║ ████╗  ██║
 ██████╔╝█████╗  ██████╔╝██║ ██╔██╗ ██║
 ██╔═══╝ ██╔══╝  ██╔═══╝ ██║ ██║╚██╗██║
 ██║     ███████╗██║     ██║ ██║ ╚████║
 ╚═╝     ╚══════╝╚═╝     ╚═╝ ╚═╝  ╚═══╝

 v<version>  · scanner de posture cloud (sécurité · conformité)

pepin: ⚠ dérogation EXPIRÉE le 2020-01-01 sur network_securitygroup_allow_ingress_from_internet_to_tcp_port_22 / sg-bastion : elle ne s'applique plus, l'écart redevient un écart (platform-security)

ⓘ Ce rapport évalue la configuration d'un tenant (périmètre commanditaire). Les correspondances normatives (SecNumCloud, ISO, CIS) sont indicatives : elles ne constituent pas une preuve de qualification/certification, laquelle porte sur le prestataire de service cloud.
$ echo $?
1
```
<!-- /pepin:gen exit-run-expired -->

Une dérogation qui nomme un contrôle ou une ressource inexistants est signalée de la même
façon, comme `ORPHAN` : c'est le symptôme d'une exception oubliée après un renommage ou un
retrait. Sous `--strict`, une dérogation expirée ou orpheline suffit à refuser la porte
(code 3) : un pipeline qui demande la rigueur demande que son fichier de dérogations soit revu.

### Ordre de priorité

Quand plusieurs situations se présentent en même temps, les codes se décident dans cet ordre :

1. **2** : une erreur technique, rien d'autre n'est un verdict.
2. **1** : au moins un écart critical/high **non** couvert par une dérogation valide.
3. **3** : rien n'a été mesuré (hors gouvernance).
4. **3** : la collecte était incomplète, et au moins un contrôle y a perdu son `pass` faute
   d'avoir pu lire la donnée dont il dépend.
5. **4** : au moins une dérogation a été appliquée.
6. **3** : `--strict` et des écarts medium/low subsistent, le fichier de dérogations est
   périmé, ou un réglage assoupli a fait tomber une correspondance normative.
7. **0** : aucun des cas ci-dessus.

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
  3) echo "le scan n'établit pas la conformité : rien mesuré, collecte incomplète, ou medium/low sous --strict" ; exit 1 ;;
  4) echo "des écarts subsistent, tous sous dérogation datée" ; exit 1 ;;
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

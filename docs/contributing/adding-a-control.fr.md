> [🇬🇧 English](adding-a-control.md) · 🇫🇷 Français

# Ajouter un contrôle, de bout en bout

Ce guide suit un contrôle réel de ce dépôt, du risque à la pull request :
**`database_service_not_open_to_internet`**, une base de données managée dont la liste
d'IP autorisées admet un CIDR public. Chaque fichier cité ci-dessous existe ; rien ici
n'est une esquisse.

Avant toute chose, deux invariants décident de la forme du travail.

**Un seul jeu de règles, commun à tous les fournisseurs.** Un contrôle s'écrit **une
fois**, dans `internal/commonrules/rules/`, et il évalue le modèle *normalisé*. Ce qui
change d'un cloud à l'autre, c'est la **source** (le collecteur, le mapping Terraform),
jamais la règle. Si un contrôle semble exiger une logique propre à un fournisseur, c'est
la normalisation qui manque, pas une seconde règle. Voir
[l'architecture](../project/architecture.fr.md).

**Tout ce qu'un humain lit est écrit deux fois.** Le français est la langue de référence
du contenu normatif, l'anglais en est la traduction maintenue. Un contrôle porte
`titre_en`, `description_en` et `remediation_en` ; une règle porte `labels.message_en`
et `labels.remediation_en`. `mise run validate` refuse l'absence de l'un d'eux : un
rapport anglais ne doit jamais retomber en français au milieu d'une phrase.

Vous n'avez **pas** besoin d'un compte cloud pour suivre ce guide. Auditer un plan
Terraform ne provisionne rien, et cela suffit à valider un mapping et une règle.

---

## 1. Identifier le risque

Le formuler comme un fait de configuration observable, pas comme une intention. « La
base est joignable depuis Internet » est observable ; « la base n'est pas sécurisée » ne
l'est pas.

Pour notre exemple : une base de données managée expose une ACL (une liste de CIDR
autorisés). Si cette liste contient `0.0.0.0/0`, le service est joignable depuis
Internet tout entier.

## 2. Trouver la référence normative : l'index est gelé

Chaque contrôle se rattache à une exigence SCSL **existante** du module de posture cloud
(`CLD-*`). L'index est **gelé** : on s'y rattache, on n'en crée jamais.

Les identifiants gelés sont listés dans
[`referentiel/scsl-baseline.json`](../../referentiel/scsl-baseline.json) :

```bash
grep -o 'CLD-[A-Z]*-[0-9]*' referentiel/scsl-baseline.json | sort -u
```

Avec le framework SCSL cloné à côté de Pépin, le rapport complet est :

```bash
./pepin scsl                                            # index par défaut : ../framework-scsl/api/v1/exigences.json
./pepin scsl --index /chemin/vers/api/v1/exigences.json
```

Notre exemple se rattache à `CLD-NET-1` (la source d'un service exposé est restreinte à
un CIDR maîtrisé, jamais `0.0.0.0/0`).

**Si aucune exigence gelée ne couvre le besoin, on s'arrête ici.** Le contrôle reste au
[`referentiel/catalogue.yaml`](../../referentiel/catalogue.yaml) avec `statut: a_trier`,
hors périmètre tant que SCSL n'a rien figé pour lui. Inventer une référence rendrait tout
le rapport inopposable, c'est-à-dire la seule panne que ce projet ne peut pas se
permettre.

## 3. Vérifier l'applicabilité, fournisseur par fournisseur

Pour chaque fournisseur que vous comptez activer, le champ natif doit être **vérifié
dans le SDK ou la documentation officielle de l'API**, puis consigné dans
`providers/<fournisseur>.yaml`. « Absent » est un constat lui aussi, et il se prouve en
lisant le SDK, jamais en le supposant.

Côté Scaleway, le contrat consigne le type comme vérifié et nomme d'où vient l'attribut :

```yaml
    managed_database:
      etat: verifie
      source: >
        Terraform scaleway_rdb_acl (mapping validé par un plan réel : acl_rules[].ip
        agrégés en ip_filter ; formes reprises de scaleway/dagster-scaleway et
        Qovery/engine, ip=0.0.0.0/0). API live api/rdb/v1 ACLRule.IP à câbler en collecte.
      mapping:
        database_id: instance_id du RDB parent
        ip_filter: '[acl_rules[].ip] (CIDR autorisés ; transform list)'
```

Trois états existent, et ils ne sont pas interchangeables : `verifie` (lu dans le SDK, et
projeté), `a_verifier` (plausible, non lu : aucun « pass » ne sera affirmé), `absent` (le
mécanisme n'existe pas, avec sa justification). L'état commande le « pass » :
`internal/assess` refuse d'affirmer une conformité sur un type que le contrat n'a pas
vérifié.

## 4. Ajouter le contrôle au référentiel

`referentiel/controles.yaml` est la source de vérité. Un contrôle est agnostique : son
`code` suit `<service>_<resource>_<check>` avec un préfixe de service neutre
(`network_`, `compute_`, `objectstorage_`, `blockstorage_`, `iam_`, `kubernetes_`,
`loadbalancer_`, `governance_`). Aucun nom de fournisseur n'apparaît jamais dans un code.

```yaml
  - code: database_service_not_open_to_internet
    famille: reseau
    titre: Base de données managée joignable depuis Internet
    titre_en: Managed database reachable from the internet
    severite: high
    description: >
      La liste d'IP autorisées (ACL) d'une base de données managée admet un CIDR
      public : le service de base de données est joignable depuis Internet.
    description_en: >
      The allowed IP list (ACL) of a managed database admits a public CIDR: the
      database service is reachable from the internet.
    remediation: >
      Restreindre l'ACL de la base aux seuls CIDR applicatifs (réseau privé quand
      disponible) ; retirer 0.0.0.0/0.
    remediation_en: >
      Restrict the database ACL to the application CIDRs only (a private network
      where one is available); remove 0.0.0.0/0.
    scsl: [CLD-NET-1]
    frameworks:
      secnumcloud_3_2: ["13.2"]
      cis_controls_v8: ["4.4", "12.2"]
      iso_27001_2022: ["A.8.20", "A.8.22"]
    fournisseurs: [scaleway]
```

`fournisseurs` est l'interrupteur d'activation. Le laisser **vide** tant que le mapping
n'a pas été validé sur un plan réel ou un scan réel : un contrôle déclaré pour un
fournisseur qu'il ne sait pas mesurer est une promesse fausse, et la matrice de
couverture le dira.

## 5. Déterminer les attributs décisifs

C'est l'étape qui sépare un contrôle honnête d'un faux vert, et elle est détaillée au
[§ 6](#6-létape-quon-oublie--la-donnée-manquante).

Une seule question à se poser : **si l'attribut que la règle lit n'a jamais été
collecté, la règle reste-t-elle muette ?** Pour notre exemple, oui : pas d'`ip_filter`,
pas d'écart. Le silence se lirait alors comme une conformité, ce qui est exactement
faux : une base Scaleway est ouverte tant qu'aucune ACL n'est posée.

L'attribut est donc déclaré dans la table `requiredAttr` de
[`internal/assess/assess.go`](../../internal/assess/assess.go) :

```go
	// La base est ouverte tant qu'aucune ACL n'est posée (défaut d'API Scaleway) :
	// sans ip_filter collecté, « conforme » est exactement l'inverse de la réalité.
	"database_service_not_open_to_internet": {"ip_filter"},
```

Les contrôles **non** listés se jugent à la présence d'un écart : l'absence de mauvaise
configuration y vaut réellement conformité. C'est une décision à prendre délibérément,
contrôle par contrôle, pas un défaut dont on hérite.

## 6. L'étape qu'on oublie : la donnée manquante

Une règle qui ne se déclenche pas faute de donnée doit produire **`not-evaluated`**,
jamais `pass`. Ce n'est pas une nuance, c'est la différence entre un audit et une
réassurance. Un audit interne de ce dépôt a trouvé **quatorze** contrôles qui
concluaient « conforme » sans avoir jamais lu l'attribut décisif. Un faux vert est
invisible par construction, ce qui en fait le pire défaut possible pour un outil de
posture.

Deux mécanismes travaillent ensemble, et les deux sont nécessaires :

1. **La garde de capacité, dans la règle.** La règle ne lit l'attribut que s'il est
   présent : un attribut non collecté ne produit donc aucun faux positif.

   ```rego
   "ip_filter" in object.keys(r.attributes)
   ```

2. **Le verrou du « pass », dans `internal/assess`.** L'entrée `requiredAttr` transforme
   le silence en `not-evaluated` quand aucune ressource du type visé ne portait
   l'attribut. Sans elle, la garde achèterait un vert en silence.

Un test tient les deux synchronisés : `TestRequiredAttrGuardsExist`
(`internal/assess/requiredattr_test.go`) échoue si une entrée `requiredAttr` nomme un
contrôle qu'aucune règle n'émet, ou un attribut que la règle ne lit jamais, autrement dit
« un gate qui ne protège rien ».

La table générée de tous les contrôles gatés, avec leur attribut décisif, est dans
[le modèle d'assessment](../concepts/assessment-model.fr.md).

## 7. Écrire la règle commune

Un fichier, `internal/commonrules/rules/<code>.rego`, `package pepin.rules`,
`import rego.v1`, et un `deny contains f if { … }`. L'en-tête cite la source ancrée ;
`labels.provider` est tiré **de la ressource** via `provider_of(r)`, jamais codé en dur.

```rego
# database_service_not_open_to_internet
#   Base de données managée dont la liste d'IP autorisées (ACL) admet un CIDR
#   public : le service de base de données est joignable depuis Internet.
# SCSL : CLD-NET-1 (source restreinte à un CIDR maîtrisé, jamais 0.0.0.0/0).
# Contrat : type normalisé agnostique `managed_database` ; attribut ip_filter
#   (liste de CIDR autorisés, DÉRIVÉ — Scaleway RDB ACLs / Exoscale DBaaS
#   ip-filter). Absent ⇒ pas de finding (garde de capacité).
package pepin.rules

import rego.v1

deny contains f if {
	some r in resources_of_type("managed_database")
	"ip_filter" in object.keys(r.attributes)
	some cidr in cidr_list(object.get(r.attributes, "ip_filter", []))
	is_public_cidr(cidr)
	id := object.get(r.attributes, "database_id", r.id)
	f := {
		"code": "database_service_not_open_to_internet",
		"severity": "high",
		"subject": id,
		"message": sprintf("Base de données managée « %s » : ACL autorisant un CIDR public (%s) — service exposé à Internet.", [id, cidr]),
		"remediation": "Restreindre l'ACL de la base aux seuls CIDR applicatifs (réseau privé quand disponible) ; retirer 0.0.0.0/0.",
		"labels": {
			"provider": provider_of(r),
			"category": "security",
			"message_en": sprintf("Managed database \"%s\": ACL allowing a public CIDR (%s) — the service is exposed to the internet.", [id, cidr]),
			"remediation_en": "Restrict the database ACL to the application CIDRs only (a private network where one is available); remove 0.0.0.0/0.",
		},
	}
}
```

Deux détails faciles à rater :

- **Accès défensif uniquement** (`object.get`, `object.keys`) : un inventaire est une
  entrée non fiable, et une règle ne doit jamais paniquer sur un champ absent.
- **La sévérité doit correspondre au référentiel.**
  `TestRegoSeverityMatchesReferentiel` compare les deux et échoue sur un écart.

Réutiliser les helpers partagés de `lib.rego` (`resources_of_type`, `provider_of`,
`is_public_cidr`, `cidr_list`) plutôt que de les réécrire : un helper corrigé une fois
est corrigé pour toutes les règles.

## 8. Ajouter les fixtures

Les fixtures sont des **réponses d'API ou de plan réalistes**, jamais des formes
inventées. Deux miroirs comptent :

- le non conforme, sous `examples/<fournisseur>/terraform/` (ou la fixture
  d'inventaire), qui déclenche la règle ;
- le conforme, sous `examples/<fournisseur>/terraform-fixed/` ou comme preuve de
  remédiation dans `references/remediation/<fournisseur>/<code>/`, qui ne doit **pas** la
  déclencher.

Aucun secret ne va jamais dans une fixture : les identifiants viennent de
l'environnement, et `mise run secrets` (gitleaks) scanne l'historique à la recherche des
motifs de clés souveraines.

## 9. Tester l'échec, le succès et l'absence

Le fichier de test de la règle, `<code>_test.rego`, doit contenir les trois cas. Voici le
vrai, sans coupe :

```rego
package pepin.rules

import rego.v1

_db(attrs) := {"resources": [{"provider": "scaleway", "type": "managed_database", "id": "db-1", "attributes": attrs}]}

# ✗ ACL avec 0.0.0.0/0 → finding.
test_db_open_to_all_denied if {
	some f in deny with input as _db({"database_id": "db-1", "ip_filter": ["0.0.0.0/0"]})
	f.code == "database_service_not_open_to_internet"
}

# ✗ ACL avec un /1 (moitié d'Internet) → finding (réutilise is_public_cidr).
test_db_half_internet_denied if {
	some f in deny with input as _db({"database_id": "db-1", "ip_filter": ["128.0.0.0/1"]})
	f.code == "database_service_not_open_to_internet"
}

# ✓ ACL restreinte à un CIDR privé → aucun finding.
test_db_restricted_ok if {
	count({f | some f in deny; f.code == "database_service_not_open_to_internet"}) == 0 with input as _db({"database_id": "db-1", "ip_filter": ["10.0.0.0/16"]})
}

# ✓ ip_filter non collecté (garde de capacité) → pas de faux positif.
test_db_uncollected_ok if {
	count({f | some f in deny; f.code == "database_service_not_open_to_internet"}) == 0 with input as _db({"database_id": "db-1"})
}
```

Le dernier test est celui qu'on saute. Il prouve que la garde existe ; c'est l'entrée
`requiredAttr` de l'étape 5 qui transforme son silence en `not-evaluated` plutôt qu'en
`pass`. Les deux sont nécessaires : le test seul laisserait un faux vert, la table seule
laisserait un faux positif.

```bash
mise run test-rego     # opa test internal/commonrules/rules -v
mise run test          # tests Go (-race) + tests Rego
```

Un test ne vaut que ce que vaut sa capacité à échouer. Casser la règle exprès (retirer la
garde, inverser une comparaison) et vérifier que la suite passe au rouge. Une suite qui
reste verte alors que son sujet est cassé ne mesure plus son sujet, elle mesure sa propre
panne.

## 10. Vérifier `not-applicable` et `not-evaluated` sur un scan réel

Lancer le scan et lire les statuts, plutôt que de faire confiance aux tests unitaires :

```bash
./pepin scan scaleway --terraform examples/scaleway/terraform/plan.json --format assessment
./pepin scan scaleway --terraform examples/scaleway/terraform-fixed/plan.json --format assessment
```

Ce qu'il faut vérifier, dans l'ordre :

- le contrôle apparaît dans `results` (sinon, le code n'est pas au référentiel ou aucune
  règle ne l'émet : `mise run validate` dit lequel des deux) ;
- il est `fail` sur l'entrée non conforme, avec un `subject` qui nomme la ressource
  fautive ;
- il est `pass` sur l'entrée conforme, et `evidence.observed` dit **pourquoi** le « pass »
  est affirmé ;
- sur une entrée où l'attribut décisif est absent, il est `not-evaluated` **avec son
  motif**, et non `pass`.

Si un fournisseur n'a réellement pas ce mécanisme, ce n'est pas un trou de Pépin : le
consigner en `non_applicable` dans `providers/<fournisseur>.yaml` **avec sa
justification**, bilingue. Le scan rend alors `not-applicable` accompagné de ce motif, sur
lequel un auditeur peut agir. Un N/A sans justification n'est pas opposable, et un test
impose la justification bilingue.

## 11. Documenter la remédiation

Les champs `remediation` et `remediation_en` de l'étape 4 sont obligatoires et tenus par
`TestEveryFindingCarriesRemediation`. La couche du dessus, une preuve **déployable** sous
`references/remediation/<fournisseur>/<code>/`, est décrite dans
[le guide de remédiation](../guides/remediation.fr.md). C'est la différence entre dire au
lecteur quoi faire et lui montrer le montage qui le fait.

## 12. Régénérer la documentation dérivée

Le catalogue des contrôles, la matrice de couverture et toutes les sorties de commande
montrées dans `docs/` sont **générées**. Ne jamais les éditer à la main :

```bash
mise run gen-docs
```

Votre contrôle a désormais ses deux pages sous `docs/controls/`, il apparaît dans
`docs/coverage.fr.md` avec son statut par fournisseur et par source, et les chiffres
bougent. `TestGeneratedDocsAreUpToDate` échoue sur toute documentation en retard : la
régénération n'est pas facultative.

---

## Checklist avant la pull request

- [ ] L'exigence SCSL (`CLD-*`) **existait déjà** dans l'index gelé ; aucune n'a été
      inventée. Si aucune ne couvre le besoin, le contrôle est resté au `catalogue.yaml`.
- [ ] Pour chaque fournisseur activé, le champ natif est consigné dans
      `providers/<fournisseur>.yaml` avec `etat: verifie` et sa source.
- [ ] Le contrôle est dans `referentiel/controles.yaml` avec un `code` agnostique, sa
      sévérité, son `scsl`, ses correspondances de frameworks et ses `fournisseurs`.
- [ ] Les six champs de prose sont remplis **dans les deux langues**
      (`titre`/`titre_en`, `description`/`description_en`,
      `remediation`/`remediation_en`).
- [ ] Une règle dans `internal/commonrules/rules/`, `package pepin.rules`, avec
      `labels.provider: provider_of(r)` et `labels.message_en` /
      `labels.remediation_en`.
- [ ] La sévérité de la règle correspond à celle du référentiel.
- [ ] La règle porte sa **garde de capacité**, et l'attribut décisif est déclaré dans
      `requiredAttr` dès que le silence achèterait un `pass`.
- [ ] `<code>_test.rego` couvre l'**échec**, le **succès** et l'**attribut manquant**.
- [ ] La suite a été éprouvée sur sa capacité à échouer : casser la règle la fait passer
      au rouge.
- [ ] Un scan réel montre `fail`, `pass` et `not-evaluated` là où chacun est attendu,
      jamais un `pass` sur une donnée non collectée.
- [ ] Tout `non_applicable` est justifié, en deux langues, dans le contrat du
      fournisseur.
- [ ] `mise run gen-docs` a été lancé et le résultat committé.
- [ ] `mise run validate`, `mise run test` et `mise run audit` sont au vert.
- [ ] Le commit suit Conventional Commits, en anglais, à l'impératif, sous 72 caractères.

## Voir aussi

- [Architecture](../project/architecture.fr.md) : pourquoi un seul jeu de règles, et ce
  qui change d'un cloud à l'autre.
- [Ajouter un fournisseur](adding-a-provider.fr.md) : l'autre moitié, la source.
- [Catalogue des contrôles](../controls/index.fr.md) : à quoi ressemblera la page de
  votre contrôle.
- [Le modèle d'assessment](../concepts/assessment-model.fr.md) : ce que chaque statut
  affirme.
- [CONTRIBUTING.fr.md](../../CONTRIBUTING.fr.md) : les portes de qualité et les points
  non négociables.

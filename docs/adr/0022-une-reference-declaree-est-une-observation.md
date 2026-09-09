# ADR-0022 — Une référence déclarée est une observation ; une ambiguïté est un silence

```yaml
status: Accepted
date: 2026-09-09
scope:
  - collectors
  - model
  - assessment
```

## Contexte

Le commentaire de tête d'`internal/tfparse` affirmait que `planned_values` porte des
valeurs « entièrement résolues », « ce qui permet aux règles de corrélation de
fonctionner ». **C'était faux**, et la matrice de couverture le répétait en marquant
✅ des cellules que rien ne pouvait honorer.

Ce qu'un plan ne peut pas connaître — l'identifiant d'une ressource que le même plan
va créer — n'est pas résolu : il est **absent**. Mesuré sur un tenant de référence
Outscale construit depuis du HCL tiers :

| attribut | présent | absent |
|---|---:|---:|
| `compute_instance.vm_id` | 0 | 5 |
| `compute_instance.public_ip` | 0 | 5 |
| `compute_instance.security_group_ids` | 0 | 5 |
| `security_group_rule.security_group_id` | 0 | 14 |

C'est-à-dire **tout ce sur quoi les règles se joignent**. Le verdict restait honnête
(`not-evaluated`, jamais un `pass` : l'ADR-0014 et l'ADR-0006 tenaient), mais la
corrélation ne fonctionnait sur aucun plan réel — seulement sur les plans écrits à la
main pour les tests, où l'identifiant est un littéral. Une capacité éprouvée
uniquement par des fixtures qui la confirment n'est pas éprouvée.

Le plan porte pourtant le fait ailleurs. `configuration` garde la **relation** que
l'exploitant a écrite :

```json
"vm_id": {"references": ["outscale_vm.web.vm_id", "outscale_vm.web"]}
```

## Décision

**Une référence déclarée dans `configuration` est une observation du plan**, et elle
comble l'attribut que `planned_values` laisse vide. La valeur projetée est l'**adresse
d'instance** de la ressource référencée — celle-là même qui identifie cette ressource
dans l'inventaire, donc celle sur laquelle la jointure se fait.

**Une référence ambiguë ne résout rien.** Une déclaration démultipliée par `count` ou
`for_each` produit N instances que la configuration ne distingue pas : le silence est
le seul verdict honnête.

Trois bornes complètent la décision : une valeur **présente gagne toujours** ; seuls
les **chemins simples** sont comblés (un chemin composé désigne une structure de
`values`, pas un argument de la configuration) ; et seule une adresse correspondant à
une ressource **réellement présente dans le plan** est projetée.

## Justification

Ce n'est pas une estimation, et c'est tout l'argument. La relation est **écrite** par
l'exploitant, lue telle quelle, et rattachée à une ressource qu'on a vue. L'ADR-0014
interdit de fabriquer une valeur qu'on n'a pas observée ; il n'interdit pas de lire
une observation là où elle se trouve. Ce qui reste inconnu — la valeur de
l'identifiant — reste inconnu : on ne projette pas une IP, on projette l'adresse de la
ressource qui la porte.

L'ambiguïté mérite d'être nommée séparément parce que la tentation de la trancher est
forte et l'erreur invisible. Joindre la VM n° 0 au groupe n° 1 poserait un écart sur
une ressource qui ne le porte pas : un faux positif sur une configuration tierce
inchangée, la panne la plus coûteuse de ce dépôt.

## Alternatives écartées

**Se contenter de dire la vérité dans la matrice (◐ partout).** Rejeté comme
suffisant, retenu comme complément : marquer ◐ décrit honnêtement une capacité
absente, mais la capacité était à portée. Le décompte de couverture reste à mesurer
plutôt qu'à déclarer — c'est le lot suivant, et il portera sur ce qui RESTE ouvert.

**Trancher l'ambiguïté en prenant la première instance.** Rejeté : plausible et faux,
exactement la formule que l'ADR-0014 refuse. Cette proposition reviendra, parce qu'elle
fait « marcher » un cas de plus ; la raison du rejet est qu'elle fait mentir tous les
autres.

**Résoudre les références en écrasant les valeurs présentes.** Rejeté : un état
appliqué (`values`) porte des valeurs RÉELLES. Les remplacer par des adresses
échangerait une observation contre une déclaration, dans le mauvais sens.

**Déclarer, dans chaque descripteur, quels champs sont « known after apply ».**
Rejeté : une déclaration de plus à tenir à jour, qui pourrit en silence dès qu'un
provider change de schéma. Le plan sait déjà ce qu'il ne sait pas.

**Ne pas franchir les frontières de module.** Rejeté après mesure : à l'intérieur d'un
module, l'argument référence `var.x`, et l'adresse réelle est dans l'appel. Sans ce
saut, la fonctionnalité n'existe que sur les plans à plat — c'est-à-dire sur aucun plan
sérieux.

## Conséquences

**Une adresse projetée devient une identité.** Un descripteur dont l'`id:` porte sur un
champ désormais comblé change le sujet de ses findings : sur un plan Scaleway, une ACL
de bucket a pour sujet le bucket qu'elle configure, et non plus la ressource ACL. C'est
le sujet que l'on cherche, et il rejoint celui de la collecte live — mais une
dérogation écrite sur l'ancien sujet cesse de correspondre, et cela s'écrit au
CHANGELOG.

**Le corpus de référence doit porter les références.** La réduction des plans de
tenants ne gardait que `planned_values` et la `source` des modules. Elle garde
désormais aussi les `references` de `configuration` — et **jamais** les
`constant_value`, qui sont le contenu écrit par l'exploitant et donc l'endroit exact où
vit un secret en dur. La distinction est la raison d'être de la réduction, et une garde
la tient sur le texte des plans committés plutôt que sur l'intention du script.

**La corrélation reste partielle.** Une IP publique attachée par une ressource de LIEN
(`outscale_public_ip_link`) n'est toujours pas rattachée à sa VM : c'est une jointure
INVERSE, qui écrit sur une autre ressource que celle qu'on projette, et elle demande un
autre mécanisme. Le contrôle correspondant reste `not-evaluated` sur cette source, ce
qui est honnête, et fait l'objet de son propre suivi.

## Invariants

- Une valeur présente n'est jamais remplacée par une référence — *garde :
  `TestAPresentValueAlwaysWinsOverAReference`*.
- Une référence qui ne désigne pas une ressource du plan ne projette rien — *garde :
  `TestAReferenceToSomethingThatIsNotAResourceResolvesToNothing`*.
- Une référence désignant plusieurs instances ne projette rien — *garde :
  `TestAnAmbiguousReferenceResolvesToNothing`*.
- Un attribut comblé par une référence est attesté `derived`, avec `configuration`
  pour source — *garde : `TestAReferenceIsAttestedAsDerivedFromTheConfiguration`*.
- Aucun plan de tenant de référence ne porte de `constant_value` — *garde :
  `TestNoReferenceTenantPlanCarriesAConstantValue`*.

## Validation

Mesuré sur le tenant `ztiac-two-tier`, modulaire et démultiplié par `count` : chaque VM
est rattachée à SON groupe de sécurité, à travers la frontière de module, et les deux
instances d'une même déclaration reçoivent le même groupe — ce qui est le cas exact.
Un seul verdict bouge sur tout le corpus, `compute_instance_has_security_group` de
`not-evaluated` à `pass`, et ce `pass` est prouvé par ce que le plan déclare.

## Liens

ADR-0006 (jamais un `pass` non prouvé) · ADR-0014 (une donnée absente ne se fabrique
pas) · ADR-0017 (la provenance atteste une recherche) · issue #175.

> [🇬🇧 English](detection-quality.md) · 🇫🇷 Français

<!-- Page GÉNÉRÉE par internal/docgen. Ne pas éditer à la main. -->

# Carte de qualité de détection

Ce que Pépin peut PROUVER de ses propres verdicts, et ce qu'il ne peut pas.
Chaque chiffre de cette page est dérivé des artefacts du dépôt — obligations
calculées depuis la matrice de couverture, scénarios de véracité, tenants de
référence, relevés de canari et de qualification. Aucun n'est saisi.

## La règle

**Aucun chiffre publié ici ne peut être meilleur que ce qui est mesuré.** Un
pourcentage sans mesure derrière est un faux vert déplacé dans un tableau de
bord, et il y est pire qu'ailleurs : personne ne relit un tableau de bord.

Les chiffres sont donc laids, et c'est le point. « 58 contrôles » ne dit rien de
la qualité d'une détection ; « 111 verdicts prouvés sur 478 » dit où en est le
produit, et rétrécit dans le bon sens à chaque scénario écrit.

## Les chiffres

| Chiffre | Nombre |
|---|---:|
| Contrôles au référentiel | 58 |
| Chemins contrôle × fournisseur × source sur lesquels Pépin conclut | 190 |
| Chemins dont TOUS les verdicts atteignables sont prouvés de bout en bout | 38 |
| Verdicts à prouver au total | 478 |
| Verdicts prouvés | 111 |

## Couverture de véracité, par verdict

Un chemin doit prouver les verdicts qu'il peut réellement ATTEINDRE, pas quatre
partout : exiger un `not-applicable` d'un chemin où le mécanisme existe
demanderait d'inventer une non-applicabilité.

| Verdict | Ce qu'il met en scène | À prouver | Prouvés | % |
|---|---|---:|---:|---:|
| `fail` | une configuration vulnérable est détectée | 144 | 29 | 20 |
| `pass` | une configuration réellement correcte est confirmée | 144 | 41 | 28 |
| `not-evaluated` | l'attribut décisif manque, et le scan refuse de conclure | 166 | 29 | 17 |
| `not-applicable` | le contrat du fournisseur déclare le mécanisme inexistant | 24 | 12 | 50 |
| **Total** | | **478** | **111** | **23** |

## Validé en live

Un scan canari interroge le VRAI plan de contrôle d'un fournisseur, mais **sans
identifiant** : il prouve qu'un endpoint existe et refuse, jamais qu'un droit
*suffisant* rende `200` sur un tenant réel. Il ne vaut donc pas validation live
d'un contrôle.

Un chemin compte lorsqu'au moins un verdict `pass` ou `fail` requis a été
explicitement comparé sur un tenant authentifié, pour ce contrôle et ce
fournisseur. Cela ne prouve pas tous les verdicts de ce chemin et ne modifie
pas le registre de véracité. Les contre-exemples silencieux, attentes hors tenant
et défauts connus ne constituent pas une preuve de `pass`.

Les relevés sont des faits historiques réévalués avec les attentes dont le
hachage est consigné. Ils indiquent le binaire réellement exécuté, pas celui
de la release. Le bundle vérifie l'intégrité, pas l'identité de son auteur.
Un tenant que nous avons écrit ne prouve pas la justesse universelle d'une
règle ; les tenants tiers la complètent.

Instantané historique au 2026-09-11T07:34:27Z. Chaque relevé expire 90 jours après son
horodatage ; le binaire retire les preuves périmées au moment de l'exécution ;
le préflight refuse un fournisseur dont les relevés sont tous non frais.

| Chiffre | Nombre |
|---|---:|
| Chemins dont la source est une collecte live | 104 |
| Chemins avec au moins un verdict live validé | 40 |
| Verdicts distincts validés en live | 40 |
| Validé en live | **38 %** |

| Fournisseur | Relevé le | Expire à (UTC) | Binaire qualifié | Relevé |
|---|---|---|---|---|
| `exoscale` | 2026-09-11T07:34:27Z | 2026-12-10T07:34:27Z | `0.1.0-dev` (`vcs:2cf48ca9cd0967a16aa6b2f5396749503a6a04dd+modified`) | [JSON](../references/qualification/evidence/exoscale-2026-09-11.json) |
| `scaleway` | 2026-09-11T05:13:16Z | 2026-12-10T05:13:16Z | `0.1.0-dev` (`vcs:74ebd40a490ddf0905957982e814949a0eb20584+modified`) | [JSON](../references/qualification/evidence/scaleway-2026-09-11.json) |

## Ce que les vrais plans de contrôle ont répondu

Une requête non authentifiée par endpoint déclaré, à la qualification de release.
Un endpoint qui répond existe et se résout ; un `moved` (404) dit qu'il a bougé.

| Fournisseur | Relevé le | Ont répondu | Déplacés | Injoignables |
|---|---|---:|---:|---:|
| `exoscale` | 2026-08-21 | 9 | 0 | 0 |
| `outscale` | 2026-09-12 | 18 | 0 | 0 |
| `scaleway` | 2026-09-21 | 7 | 0 | 0 |

## Précision des règles high/critical

Détecter et se TAIRE sont deux mesures différentes, et elles se publient ensemble :
« 21 chemins de détection prouvés » se lit dans le sens le plus flatteur tant que
rien ne dit sur combien de configurations légitimes ces mêmes règles se sont tues.
Une règle qui se déclenche sur tout est parfaitement sensible.

Un CONTRE-EXEMPLE est un couple sur un même chemin contrôle × fournisseur ×
source : un cas fautif, et un cas correct qui lui ressemble. Les contrôles qui
n'en ont pas encore sont comptés dans
`internal/veracity/testdata/counterexamples-debt.txt`.

| Chiffre | Nombre |
|---|---:|
| Contrôles high/critical actifs | 43 |
| Dont un chemin de détection est prouvé de bout en bout | 22 |
| Dont un contre-exemple légitime est prouvé | 20 |
| Faux positifs mesurés sur les contre-témoins | 0 |

Il n'y a pas de ligne « faux négatifs », et son absence est le chiffre le plus
honnête de cette page. Aucun artefact du dépôt ne les mesure : il faudrait un
corpus de configurations fautives dont on SAIT qu'elles échappent aux règles,
c'est-à-dire savoir ce qu'on ne sait pas. Publier « 0 » serait le faux vert exact
que cette page combat — zéro mesuré n'est pas zéro existant.

## Faux positifs

Le dépôt ne tient pas de registre de faux positifs, et en publier un compte serait
exactement la saisie que cette page refuse. Ce qui est MESURÉ, c'est le
contre-témoin : un tenant tiers déclaré durci sur lequel Pépin ne relève aucun
écart `critical`/`high`. C'est le seul endroit où un faux positif se voit, et une
porte le vérifie à chaque build.

| Chiffre | Nombre |
|---|---:|
| Tenants tiers durcis sans écart critical/high (contre-témoins) | 2 |
| Tenants de référence au total | 6 |

## Mesures hors de portée

Elles sont documentées plutôt que comblées : cf. [Limites connues](known-limitations.fr.md)
et le registre de dette `internal/veracity/testdata/debt.txt`, qui nomme ligne à
ligne chaque verdict restant à prouver.

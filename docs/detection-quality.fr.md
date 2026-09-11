> [🇬🇧 English](detection-quality.md) · 🇫🇷 Français

<!-- Page GÉNÉRÉE par internal/docgen. Ne pas éditer à la main. -->

# Carte de qualité de détection

Ce que Pépin peut PROUVER de ses propres verdicts, et ce qu'il ne peut pas.
Chaque chiffre de cette page est dérivé des artefacts du dépôt — obligations
calculées depuis la matrice de couverture, scénarios de véracité, tenants de
référence, relevés de canari. Aucun n'est saisi.

## La règle

**Aucun chiffre publié ici ne peut être meilleur que ce qui est mesuré.** Un
pourcentage sans mesure derrière est un faux vert déplacé dans un tableau de
bord, et il y est pire qu'ailleurs : personne ne relit un tableau de bord.

Les chiffres sont donc laids, et c'est le point. « 58 contrôles » ne dit rien de
la qualité d'une détection ; « 98 verdicts prouvés sur 467 » dit où en est le
produit, et rétrécit dans le bon sens à chaque scénario écrit.

## Les chiffres

| Chiffre | Nombre |
|---|---:|
| Contrôles au référentiel | 58 |
| Chemins contrôle × fournisseur × source sur lesquels Pépin conclut | 185 |
| Chemins dont TOUS les verdicts atteignables sont prouvés de bout en bout | 33 |
| Verdicts à prouver au total | 467 |
| Verdicts prouvés | 98 |

## Couverture de véracité, par verdict

Un chemin doit prouver les verdicts qu'il peut réellement ATTEINDRE, pas quatre
partout : exiger un `not-applicable` d'un chemin où le mécanisme existe
demanderait d'inventer une non-applicabilité.

| Verdict | Ce qu'il met en scène | À prouver | Prouvés | % |
|---|---|---:|---:|---:|
| `fail` | une configuration vulnérable est détectée | 141 | 25 | 17 |
| `pass` | une configuration réellement correcte est confirmée | 141 | 37 | 26 |
| `not-evaluated` | l'attribut décisif manque, et le scan refuse de conclure | 161 | 24 | 14 |
| `not-applicable` | le contrat du fournisseur déclare le mécanisme inexistant | 24 | 12 | 50 |
| **Total** | | **467** | **98** | **20** |

## Validé en live

Un scan canari interroge le VRAI plan de contrôle d'un fournisseur, mais **sans
identifiant** : il prouve qu'un endpoint existe et refuse, jamais qu'un droit
*suffisant* rende `200` sur un tenant réel. Il ne vaut donc pas validation live
d'un contrôle.

Ce compteur ne s'incrémente que sur un relevé **authentifié**, et il n'en existe
aucun. Le zéro est dérivé, pas écrit : le jour où un mainteneur consigne un
relevé authentifié, il montera tout seul.

| Chiffre | Nombre |
|---|---:|
| Chemins dont la source est une collecte live | 103 |
| Validé en live | **0 %** |

## Ce que les vrais plans de contrôle ont répondu

Une requête non authentifiée par endpoint déclaré, à la qualification de release.
Un endpoint qui répond existe et se résout ; un `moved` (404) dit qu'il a bougé.

| Fournisseur | Relevé le | Ont répondu | Déplacés | Injoignables |
|---|---|---:|---:|---:|
| `exoscale` | 2026-08-21 | 9 | 0 | 0 |
| `outscale` | 2026-08-21 | 17 | 0 | 0 |
| `scaleway` | 2026-08-21 | 5 | 0 | 0 |

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
| Dont un chemin de détection est prouvé de bout en bout | 21 |
| Dont un contre-exemple légitime est prouvé | 19 |
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

# ADR-0018 — Un bundle recoupe ce qu'il porte ; seule la signature ferme la chaîne

```yaml
status: Accepted
date: 2026-09-09
scope:
  - security
  - persistence
  - output
```

## Contexte

Le bundle de preuve est la promesse centrale du produit : un dossier qu'un tiers
rouvre et oppose. `verify` déclarait « cohérent en interne » un bundle qui se
contredisait lui-même.

Mesuré : réécrire quatre résultats `fail` en `pass`, puis recalculer l'empreinte du
seul fichier touché, produisait un dossier accepté avec le code 0 — pendant que son
propre `manifest.json` annonçait encore `"fail": 4`.

```
manifest.json dit   {"fail": 4, "not-applicable": 2, "not-evaluated": 14, "pass": 7}
assessment.json a   {           "not-applicable": 2, "not-evaluated": 14, "pass": 11}
```

L'information qui contredisait le dossier était **dans le dossier**, et rien ne la
regardait.

## Décision

Un bundle **recoupe tout ce qu'il porte en double**, et le refus est un échec dur :

- le résumé du manifeste est confronté aux statuts réellement présents dans
  `assessment.json` ;
- la **taille** déclarée de chaque artefact est vérifiée à côté de son empreinte ;
- `checksums.txt` est lu **strictement** : une ligne illisible ou dupliquée est un
  refus.

Et la frontière, qui compte autant que les vérifications : **aucune donnée interne au
bundle ne peut ancrer `checksums.txt`.** Seule la signature détachée ferme la chaîne.
`verify` ne prétend donc pas à davantage que ce qu'il peut établir, et
`--require-signature` permet à un appelant d'exiger la propriété forte.

## Justification

Ces recoupements ne coûtent rien — un parcours des résultats, une taille de fichier,
une lecture stricte — et attrapent l'altération **la plus tentante de toutes** : celle
qui change le verdict. Ne pas les faire revenait à ne pas regarder ce qu'on avait sous
les yeux.

La frontière doit être écrite parce qu'elle est **contre-intuitive**. Il paraît naturel
de vouloir « fermer la chaîne » en enregistrant l'empreinte de `checksums.txt` quelque
part dans le bundle. C'est impossible : qui réécrit un fichier réécrit aussi l'ancre.
Un dispositif qui en donnerait l'apparence serait pire que son absence, parce qu'on
cesserait de réclamer la signature.

## Alternatives écartées

**Enregistrer l'empreinte de `checksums.txt` dans `manifest.json`, et celle de
`manifest.json` dans `checksums.txt`.** Rejeté : **circulaire**. Écrire l'un invalide
l'empreinte que l'autre vient de prendre. Cette proposition reviendra, parce qu'elle
paraît évidente — la raison du rejet est arithmétique, pas une préférence.

**Un hachage racine stocké dans le bundle.** Rejeté pour la même raison, un cran plus
haut : l'ancre est dans la zone que l'attaquant contrôle. Un ancrage n'a de valeur
qu'HORS du bundle, ce qu'est exactement la signature détachée.

**Faire échouer `verify` par défaut sur un bundle non signé.** Rejeté : cela ferait
échouer, en silence, des chaînes qui passent aujourd'hui. Le drapeau est donc opt-in —
même raisonnement qu'à l'ADR-0019 pour le profil de porte.

**Se contenter d'avertir sur la contradiction.** Rejeté : l'avertissement était déjà
sur stdout et le code de sortie disait « réussi ». L'automatisation lit le code.

## Conséquences

Un bundle produit par une version antérieure reste vérifiable : un manifeste sans
résumé n'est pas recoupé, parce qu'on ne fabrique pas ce qu'il ne dit pas (ADR-0014).

`verify` refuse désormais des dossiers qu'il acceptait. C'est le mode d'échec voulu :
ceux qu'il refuse se contredisaient.

Le message d'erreur nomme la contradiction — ce que le manifeste annonce, ce que
l'assessment porte — plutôt que de dire « altéré ». Un dossier refusé sans motif ne
permet pas de distinguer une corruption d'une falsification.

## Invariants

- Un bundle dont deux artefacts se contredisent est refusé.
- `verify` sans signature n'affirme jamais plus que l'intégrité accidentelle.
- Aucun mécanisme interne au bundle n'est présenté comme fermant la chaîne.

*Gardes : `TestVerifyRefusesEveryTamperingPattern` (table par motif d'altération, qui
commence par vérifier un bundle INTACT — sans quoi une correction refusant tout
passerait), `TestRequireSignatureRefusesAnUnsignedBundle`.*

## Validation

Quatre motifs d'altération, chacun devant rendre non nul ; deux d'entre eux passaient
avant cette décision.

## Liens

ADR-0005 (codes de sortie) · ADR-0014 (une donnée absente ne se fabrique pas) ·
ADR-0016 (chaîne d'approvisionnement) · issue #156.

# ADR-0020 — L'origine déclarée d'un inventaire fait foi : on refuse, on n'avertit pas

```yaml
status: Accepted
date: 2026-09-09
scope:
  - model
  - assessment
```

## Contexte

Scanner un inventaire Scaleway avec le jeu de règles Exoscale était accepté **sans un
mot**, et produisait un rapport plausible contenant six verdicts `pass`.

```
exoscale lisant l'inventaire scaleway :  6 fail   5 n/a   17 not-evaluated   6 pass
scaleway lisant le même fichier       :  4 fail   2 n/a   14 not-evaluated   7 pass
```

Ces `pass` ne sont pas faux par accident : ils sont **vides de sens**. Les formes de
ressources se recouvrent juste assez pour que des règles s'évaluent et concluent sur des
données que ce jeu de règles n'a jamais eu à lire. Et la discordance ne change pas
seulement la formulation — elle **déplace les verdicts, dans les deux sens**.

Le mode d'échec est silencieux et réaliste : une faute de frappe dans un pipeline, un
job copié-collé, et le rapport a l'air parfaitement normal — un mélange crédible de
statuts et un code de sortie non nul qui suggère même que le scan a travaillé.

L'information était **des deux côtés** et n'était simplement jamais confrontée.

## Décision

Un inventaire qui **déclare** un fournisseur différent de celui demandé est **refusé**,
avec le code `2` — celui déjà employé pour un export illisible. La déclaration est lue à
la racine de l'export **et** sur ses ressources.

Un inventaire qui **ne déclare rien** n'est pas refusé.

## Justification

Un inventaire dont l'origine ne correspond pas au jeu de règles n'est pas un scan aux
résultats surprenants : **c'est un scan qui n'aurait pas dû tourner.** Le code `2` dit
exactement cela, et il est déjà celui d'une entrée que l'outil ne sait pas lire.

Avertir ne suffit pas. Le contrat central du produit est qu'un `pass` n'est affirmé que
si la donnée qui l'adosse a réellement été collectée ; ici, six contrôles affirmaient
`pass` contre un inventaire que le jeu de règles n'était pas censé lire. Un avertissement
laisse le rapport, le bundle et le code de sortie inchangés.

Ne rien exiger d'un inventaire muet découle de l'ADR-0014 : une origine absente ne
s'invente pas, et l'exiger casserait tout export écrit à la main.

## Alternatives écartées

**Avertir sans refuser.** Rejeté : l'avertissement serait sur stderr, le rapport
resterait plausible, le bundle scellé enregistrerait une cible contredisant son propre
inventaire, et le code de sortie ne bougerait pas.

**Un drapeau de lecture croisée délibérée** (`--allow-cross-provider`), avec mention de
la dérogation dans `run`. Rejeté : personne n'a produit de cas d'usage, et un drapeau
qui autorise un rapport dénué de sens est un drapeau qu'on finit par poser dans un
pipeline pour faire taire une erreur. Si un besoin réel apparaît, il exigera une
nouvelle décision, pas un simple ajout.

**Exiger que tout inventaire déclare son fournisseur.** Rejeté : ADR-0014, et ça
casserait les exports écrits à la main sans rien prouver de plus.

## Conséquences

La garde est la **matrice croisée**, jamais un couple : chaque fixture lue par chaque
autre fournisseur doit sortir en `2`, et lue par le sien doit scanner normalement.
Vérifier un seul couple laisserait passer une correction qui refuse tout.

Un `verify --re-derive` n'est pas affecté : l'`input.json` scellé porte le fournisseur
du scan qui l'a produit.

Cette décision a immédiatement révélé un défaut dans le harnais de mesure de ce dépôt —
quatre tenants du corpus déclarent `outscale` et étaient scannés avec les règles
`scaleway`. Une décision qui attrape un défaut chez celui qui l'écrit est une décision
qui mesure quelque chose.

## Invariants

- Un inventaire qui déclare une origine autre que celle demandée n'est jamais évalué.
- Un inventaire muet n'est jamais refusé pour son silence.

*Garde : `TestAProviderMismatchIsRefused` (matrice croisée, diagonale comprise).*

## Validation

Neuf couples fournisseur × inventaire : six refus, trois scans normaux.

## Liens

ADR-0005 (codes de sortie) · ADR-0006 (jamais un `pass` non prouvé) · ADR-0014 · issue
#148.

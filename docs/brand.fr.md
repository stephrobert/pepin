> [🇬🇧 English](brand.md) · 🇫🇷 Français

# La marque

![Pépin](assets/brand/pepin-lockup-light.svg)

Un écu, et un pépin dedans. Pas devant, pas à côté : **dedans**, en creux, parce
que c'est là que Pépin le trouve. Le nom dit déjà les deux choses à la fois, la
graine et l'ennui qu'on découvre, et le signe ne fait que les tenir ensemble.

Le pépin n'est pas une forme claire posée sur l'écu, c'est un **vide** qui le
traverse. Sur n'importe quel fond, c'est le fond qui apparaît au travers. Cela
tient à un `fill-rule`, et c'est ce qui distingue un logo d'une image de logo.

## Une réserve, écrite plutôt que tue

Le bouclier est le signe le plus employé de toute la sécurité informatique. Il
dit « je protège », alors que Pépin ne protège rien : **il constate**. Il lit une
configuration effective, la confronte à un référentiel et rend un verdict
opposable, avec ses références normatives et une preuve scellée. Un auditeur ne
protège pas, il regarde et il signe.

Le concept retenu assume donc un signe familier au prix d'une promesse
légèrement décalée. C'est un choix défendable, la familiarité étant aussi ce qui
rend un signe lisible en une seconde, et il est noté ici pour qu'il reste un
choix et non un oubli.

Ce qui rattrape le bouclier, c'est le creux : un écu percé n'est pas tout à fait
un écu, et c'est exactement ce que dit l'outil.

## Les fichiers

Tout est dans [`assets/brand/`](assets/brand/). **Les fichiers SVG font foi** ;
les PNG en dérivent et se régénèrent, ils ne s'éditent pas.

| Fichier | Usage |
|---|---|
| `pepin-lockup-light.svg` | le défaut, sur fond clair |
| `pepin-lockup-dark.svg` | sur fond sombre |
| `pepin-lockup-mono.svg` | hérite de `currentColor` |
| `pepin-lockup-vertical-light.svg` / `-dark.svg` | la composition verticale, quand la largeur manque |
| `pepin-icon.svg` / `pepin-icon-dark.svg` | le signe seul, à partir de 20 px |
| `pepin-icon-mono.svg` | le signe seul, encre unique, pour les favicons et les terminaux |

### Les PNG, pour ce qui n'accepte pas le vecteur

[`assets/brand/png/`](assets/brand/png/) contient les versions rastérisées :
l'aperçu social de GitHub, une présentation, une vignette, un rapport d'audit
exporté par un outil qui ignore le SVG.

| Fichier | Taille | Usage |
|---|---|---|
| `pepin-icon-256/512/1024.png` | carré, transparent | avatars, vignettes, empaquetage |
| `pepin-icon-dark-512/1024.png` | carré, transparent | les mêmes, sur fond sombre |
| `pepin-lockup-light-1024/2048.png` | horizontal, transparent | présentations, articles |
| `pepin-lockup-dark-1024/2048.png` | idem | les mêmes, sur fond sombre |
| `pepin-lockup-vertical-light-1024.png` / `-dark-1024.png` | vertical, transparent | quand la largeur manque |
| `pepin-social-preview.png` | 1280×640 | GitHub Settings → Social preview |
| `pepin-social-preview-dark.png` | 1280×640 | la variante sombre |

Régénérez-les par `python3 scripts/generer-png-marque.py` plutôt que d'exporter
à la main. Le script rastérise **à la résolution visée**, en calculant `-density`
depuis le `viewBox` de chaque fichier : `convert fichier.svg -resize 1024x`
rastériserait d'abord à la taille du `viewBox`, quelques dizaines de pixels, puis
agrandirait ce bitmap, ce qui est flou et se voit à 100 %. Une densité constante
ne vaudrait pas mieux : elle n'est juste que pour le `viewBox` pour lequel on l'a
choisie, et l'icône carrée n'a pas le même que le logotype.

## Comment il s'intègre

GitHub bascule sur le thème du lecteur par `<picture>` :

```html
<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/assets/brand/pepin-lockup-dark.svg">
  <img src="docs/assets/brand/pepin-lockup-light.svg" alt="Pépin" width="200">
</picture>
```

**Le mot est en courbes, pas en texte**, et l'accent aigu en est la raison. Un
SVG qui porte `<text font-family="Poppins">` s'affiche avec ce que le navigateur
du lecteur possède, parce que GitHub ne lui sert aucune webfont ; et une police
de repli peut décaler l'accent, le recomposer, ou le rendre comme un caractère
séparé. Sur un nom de cinq lettres dont l'accent porte l'identité, cela se voit.

Les cinq glyphes sont du Poppins SemiBold converti en courbes. Poppins est sous
licence SIL Open Font License : seules ces formes circulent dans ce dépôt,
jamais la police.

## L'employer

- **Zone de respiration** : la largeur du bouclier sur chaque côté. Rien n'y entre.
- **Tailles minimales** : 24 px pour le logotype, 20 px pour le signe seul. En
  dessous, le creux du pépin se referme et il ne reste qu'un écu plein.
- **Le pépin est un vide**, pas une forme blanche. Ne le remplissez pas, même
  pour « corriger » un rendu sur fond coloré : c'est justement ce qu'il gère.
- **Sur fond sombre, prenez le fichier sombre.**

Ne coupez pas l'écu du mot dans le logotype, ne composez pas le mot dans une
autre police, n'ajoutez ni signature ni effet, et ne redressez pas les deux
coins hauts : leur arrondi répond à la pointe du bas, et deux angles vifs
au-dessus d'une pointe arrondie se lisent comme deux dessins cousus ensemble.

## Le modifier

**Rien ne s'édite à la main.** Les SVG sortent de
`scripts/generer-marque.py`, les PNG sortent des SVG : modifiez le script, et
régénérez.

```bash
python3 -m pip install -r scripts/requirements.txt
python3 scripts/generer-marque.py --verifier   # les boîtes, sans rien écrire
python3 scripts/generer-marque.py
python3 scripts/generer-png-marque.py
```

Le signe est **un seul tracé** : l'écu puis le pépin, dans le même `d`, avec
`fill-rule="evenodd"`. Les séparer en deux formes, l'une remplie de blanc,
casserait la transparence et rendrait le logo inutilisable sur autre chose que
du blanc.

**La mise en page est calculée depuis la fonte, pas jugée à l'œil**, et c'est ce
qui a rattrapé le défaut le plus coûteux de ce dessin : avec une baseline posée
à une constante choisie de tête, le jambage des « p » sortait du `viewBox`, donc
il était rogné. Le contrôle tient en une ligne, le logotype étant serré sur son
encre : `identify -format '%@' fichier.png` doit rendre exactement les
dimensions du canevas, à l'origine `+0+0`.

## Ce que ce dépôt ne peut pas versionner

- **Le réglage de l'aperçu social.** L'image, elle, est versionnée, dans
  `assets/brand/png/pepin-social-preview.png` ; ce qui ne l'est pas, c'est le
  réglage qui l'indique à GitHub. Déposez-la à la main : Settings → Social
  preview.
- **L'avatar du compte**, que GitHub prend sur le propriétaire et non sur le
  dépôt.

## Licence

**Le nom *Pépin* et le logo ne sont pas couverts par la licence Apache 2.0** qui
couvre le code. Ils peuvent être employés pour désigner ce projet, dans un
article, une conférence, une comparaison, une liste d'outils, sans rien
demander. Ils ne peuvent pas servir de marque à un fork, à un produit ou à un
service, ni d'une façon qui laisserait croire que le projet approuve ce qu'il
n'approuve pas.

Cette distinction compte davantage ici qu'ailleurs : Pépin rend des verdicts de
conformité, et une marque reprise par un tiers laisserait croire à une caution
que le projet n'a pas donnée.

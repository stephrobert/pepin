# Cache de doc fournisseur — comment le RÉUTILISER

La doc officielle de trois clouds est déjà téléchargée et figée en Markdown sur cette
machine. **Ne la retéléchargez pas** : pointez dessus.

> ⚠️ **Le cache n'est PAS versionné** (`.gitignore` : `references/docs/*/` ; seul
> `sources.yaml` l'est — 39 Mo de contenu généré n'ont rien à faire dans git). Cloner ce
> dépôt ailleurs ne ramènera donc **aucune** page. Sur cette machine : lien symbolique ou
> copie. Sur une autre machine ou en CI : il faut **régénérer** (voir plus bas), ou
> transporter le dossier hors git (archive, stockage objet).

```
/home/bob/Projets/pepin/references/docs/
├── outscale/   18 Mo   ~1500 pages   (docs.outscale.com)
├── scaleway/   15 Mo                 (scaleway.com/en/docs)
└── exoscale/   6,2 Mo                (community.exoscale.com)
```

## Brancher un autre projet dessus

Au choix, du plus léger au plus autonome :

```bash
# 1. Lien symbolique (recommandé : une seule copie, mise à jour partagée)
ln -s /home/bob/Projets/pepin/references/docs ./references/docs

# 2. Variable d'environnement, si l'outil sait la lire
export DOC_CACHE=/home/bob/Projets/pepin/references/docs

# 3. Copie d'un seul fournisseur (projet autonome / portable)
cp -r /home/bob/Projets/pepin/references/docs/outscale ./references/docs/
```

Le lien symbolique évite de dupliquer 39 Mo et fait profiter les deux projets d'un
rafraîchissement. La copie est préférable si le projet doit rester autonome (CI, autre
machine).

## Trouver la bonne page

Le nom de fichier est **le chemin de l'URL**, non alphanumériques remplacés par `-` :

```
https://docs.outscale.com/en/userguide/About-OKS.html
        →  en-userguide-about-oks-html.md
```

Donc on cherche d'abord **par nom**, c'est instantané :

```bash
D=/home/bob/Projets/pepin/references/docs

ls $D/outscale/ | grep -i oks              # toutes les pages OKS (35)
ls $D/outscale/ | grep -iE "polic|iam"     # IAM / policies
ls $D/scaleway/ | grep -i "object-storage"
```

Puis **par contenu** quand le nom ne suffit pas :

```bash
grep -ril "readonly\|rbac" $D/outscale/ | head      # fichiers contenant le terme
grep -i -A5 "cluster-admin" $D/outscale/en-userguide-managing-access-for-oks-users-html.md
```

Astuce : les pages existent souvent en **EN et FR** (`en-userguide-…` / `fr-userguide-…`).
Filtrez sur `en-` pour éviter les doublons.

## Cas particulier : la doc d'un CLI

Un outil en ligne de commande a **deux** documentations, et la meilleure n'est pas sur le web.
Exemple mesuré pour `oks-cli` : le portail ne décrit qu'une douzaine de commandes, alors que
l'aide intégrée en expose **605 lignes** — dont des options totalement absentes du web
(`--nacl`, `--wide`). L'aide du binaire a en plus l'avantage d'être **toujours à jour avec la
version installée**.

Elle est donc figée dans le cache, avec le même en-tête de provenance :

```
$D/outscale/cli-oks-cli-fullhelp.md      # oks-cli fullhelp, version notée dans l'en-tête
```

Recette réutilisable pour n'importe quel CLI :

```bash
{ echo "<!-- source : \`montool --help\` (aide intégrée, $(montool version))"
  echo "     capturé le $(date +%F) — généré, ne pas éditer à la main. -->"; echo
  echo '```'; montool fullhelp 2>&1; echo '```'
} > $D/<fournisseur>/cli-montool.md
```

Règle : **le web pour les procédures et les intentions, l'aide du binaire pour les options
exactes**. Et penser à re-capturer après une mise à jour de l'outil (la version est dans
l'en-tête, justement pour repérer le décalage).

## Vérifier la fraîcheur avant de citer

Chaque fichier porte son en-tête de provenance :

```bash
head -3 $D/outscale/en-userguide-about-oks-html.md
# <!-- source : https://docs.outscale.com/en/userguide/About-OKS.html
#      récupéré le 2026-06-12 via trafilatura — généré, ne pas éditer à la main.
#      régénérer : python3 scripts/fetch-docs.py outscale -->
```

Lisez cette date avant de vous appuyer sur la page. **Ne l'éditez jamais à la main** : un
rafraîchissement l'écraserait, et surtout elle perdrait sa valeur de preuve.

## Rafraîchir (rarement, et de façon ciblée)

Le collecteur est **idempotent** : une page déjà en cache est sautée. Relancer ne
retélécharge donc rien, ça ne fait que combler les manques.

```bash
cd /home/bob/Projets/pepin
python3 scripts/fetch-docs.py outscale          # complète les pages absentes
REFRESH=1 python3 scripts/fetch-docs.py outscale # force la réécriture (long)
```

Pour ajouter un **autre** fournisseur, déclarez-le dans `references/docs/sources.yaml`
(mode `pages:` pour des URLs ciblées, `crawl:` pour tout un sitemap) puis relancez.
Dépendances : `trafilatura`, `pyyaml`.

## Le piège à connaître

**La doc n'est pas le contrat.** Elle décrit l'intention, pas toujours le comportement.
Vécu ici : la doc Outscale présente le paramètre de pagination `FirstItem` comme
« *using its ordinal number* » (donc 1-basé) alors que l'API réelle est **0-basée** — une
correction fondée sur la doc seule aurait introduit un bug.

Règle de conduite : pour les **noms de champs, types et sémantiques exactes**, la source
est le contrat machine (OpenAPI/SDK) ; en cas de divergence, **un appel réel tranche**.
La doc reste irremplaçable là où le contrat est muet : procédures, intentions, exemples
de commandes (c'est elle qui a donné le manifeste RBAC lecture seule d'OKS).

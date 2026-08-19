# Cache de documentation fournisseur

La doc officielle d'Outscale, Scaleway et Exoscale, figée en Markdown, pour ancrer
les contrats et les contrôles sur une source **vérifiable et hors ligne** (règle d'or
§2 du [CLAUDE.md](../../CLAUDE.md) : ne jamais inventer le modèle de ressources d'un
provider).

> **Le cache n'est pas versionné.** Le `.gitignore` exclut `references/docs/*/` :
> une quarantaine de mégaoctets de contenu tiers généré n'a rien à faire dans git.
> Seul [`sources.yaml`](sources.yaml) l'est. Un clone frais ne contient donc **aucune
> page** : il faut les régénérer.

## Régénérer

```bash
python3 -m pip install -r ../../scripts/requirements.txt
python3 ../../scripts/fetch-docs.py outscale            # complète les pages absentes
REFRESH=1 python3 ../../scripts/fetch-docs.py outscale  # force la réécriture (long)
```

Le collecteur est **idempotent** : une page déjà présente est sautée, relancer ne
retélécharge rien et ne fait que combler les manques.

Pour ajouter un fournisseur, le déclarer dans `sources.yaml` (`pages:` pour des URLs
ciblées, `crawl:` pour tout un sitemap), puis relancer.

## Trouver une page

Le nom de fichier **est le chemin de l'URL**, les caractères non alphanumériques
remplacés par `-` :

```
https://docs.outscale.com/en/userguide/About-OKS.html
    →  en-userguide-about-oks-html.md
```

Chercher d'abord par nom, c'est instantané ; par contenu ensuite, quand le nom ne
suffit pas :

```bash
ls outscale/ | grep -i oks                    # toutes les pages OKS
grep -ril 'readonly\|rbac' outscale/ | head   # par contenu
```

Les pages existent souvent en anglais **et** en français (`en-…` / `fr-…`) : filtrer
sur `en-` évite les doublons.

## Vérifier la fraîcheur avant de citer

Chaque fichier porte son en-tête de provenance :

```
<!-- source : https://docs.outscale.com/en/userguide/About-OKS.html
     récupéré le 2026-06-12 via trafilatura — généré, ne pas éditer à la main.
     régénérer : python3 scripts/fetch-docs.py outscale -->
```

Lire cette date avant de s'appuyer sur la page, et **ne jamais l'éditer à la main** :
un rafraîchissement l'écraserait, et la page perdrait surtout sa valeur de preuve.

## La doc d'un CLI ne se trouve pas sur le web

Un outil en ligne de commande a deux documentations, et la meilleure n'est pas en
ligne. Mesuré sur `oks-cli` : le portail décrit une douzaine de commandes, l'aide
intégrée en expose 605 lignes, dont des options absentes du web (`--nacl`, `--wide`).
L'aide du binaire a de plus l'avantage d'être toujours à jour avec la version
installée.

Recette, réutilisable pour n'importe quel outil :

```bash
{ echo "<!-- source : \`montool --help\` (aide intégrée, $(montool version))"
  echo "     capturé le $(date +%F) — généré, ne pas éditer à la main. -->"; echo
  echo '```'; montool fullhelp 2>&1; echo '```'
} > <fournisseur>/cli-montool.md
```

Règle : **le web pour les procédures et les intentions, l'aide du binaire pour les
options exactes**. Re-capturer après une mise à jour de l'outil ; la version figure
dans l'en-tête, précisément pour repérer le décalage.

## Le piège : la doc n'est pas le contrat

Elle décrit l'intention, pas toujours le comportement.

Cas vécu sur ce projet : la doc Outscale présente le paramètre de pagination
`FirstItem` comme « *using its ordinal number* », donc 1-basé, alors que l'API réelle
est **0-basée**. Une correction fondée sur la doc seule aurait introduit un bug.

Pour les **noms de champs, les types et les sémantiques exactes**, la source est le
contrat machine (OpenAPI, SDK) ; en cas de divergence, **un appel réel tranche**. La
doc reste irremplaçable là où le contrat est muet : procédures, intentions, exemples
de commandes.

#!/usr/bin/env bash
# Tout ce qui doit être vrai avant qu'un tag existe.
#
# Ce contrôle tourne en local plutôt que dans le workflow pour une raison de
# calendrier : les gardes du workflow ne parlent qu'APRÈS le push du tag, et un
# tag poussé doit être supprimé des deux côtés ; une release publiée avec un
# mauvais binaire ne se rejoue pas du tout. Donc tout ce qui peut se vérifier
# hors ligne se vérifie ici, avant l'irréversible.
#
# Il rapporte chaque verdict au lieu de s'arrêter au premier, parce que
# s'arrêter au premier signifie le relancer cinq fois de suite.
#
# Convention de sortie des OUTILS de release (distincte de celle de `pepin
# scan`, où 2 est l'erreur technique) : ici 0 prêt à taguer, 1 quelque chose ne
# l'est pas ; et scripts/scsl-drift.py rend 2 pour une dérive non triée.
#
# Usage : tools/release/preflight.sh vX.Y.Z
set -uo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/../.." || exit 1

VERSION="${1:-}"
GREEN=$'\033[32m'; RED=$'\033[31m'; DIM=$'\033[2m'; OFF=$'\033[0m'
[ -t 1 ] || { GREEN=""; RED=""; DIM=""; OFF=""; }

failures=0
ok()   { printf "  %s✔%s %s\n" "$GREEN" "$OFF" "$1"; }
ko()   { printf "  %s✘%s %s\n    %s%s%s\n" "$RED" "$OFF" "$1" "$DIM" "$2" "$OFF"; failures=$((failures + 1)); }

if [ -z "$VERSION" ]; then
  echo "usage : tools/release/preflight.sh vX.Y.Z" >&2
  exit 1
fi
case "$VERSION" in
  v[0-9]*.[0-9]*.[0-9]*) ;;
  *) echo "ÉCHEC : $VERSION n'est pas un tag vX.Y.Z" >&2; exit 1 ;;
esac

echo "preflight pour $VERSION"

# --- le dépôt est dans un état qui mérite un tag ------------------------------

if [ -z "$(git status --porcelain)" ]; then
  ok "l'arbre de travail est propre"
else
  ko "l'arbre de travail est sale" "le tag nommerait un commit qui n'est pas ce que vous avez testé"
fi

branch="$(git rev-parse --abbrev-ref HEAD)"
if [ "$branch" = "main" ]; then
  ok "sur main"
else
  ko "sur $branch, pas main" "taguer la branche d'où les releases se coupent, ou dire pourquoi pas"
fi

if git rev-parse "$VERSION" >/dev/null 2>&1; then
  ko "$VERSION existe déjà en local" "git tag -d $VERSION, si c'était une erreur"
else
  ok "$VERSION est libre en local"
fi

if git remote | grep -q .; then
  if git ls-remote --tags origin "$VERSION" 2>/dev/null | grep -q .; then
    ko "$VERSION existe déjà sur origin" "un tag publié ne se déplace jamais"
  else
    ok "$VERSION est libre sur origin"
  fi
else
  ko "aucun remote git" "rien ne se publie depuis un dépôt sans origin"
fi

# --- les portes que le projet possède déjà ------------------------------------

if mise run test >/dev/null 2>&1; then
  ok "mise run test (Go -race + Rego + surfaces gelées)"
else
  ko "mise run test échoue" "les tests Go, les tests Rego, ou une surface gelée a bougé"
fi

if mise run validate >/dev/null 2>&1; then
  ok "mise run validate (codes ↔ règles ↔ index SCSL gelé ↔ catalogue)"
else
  ko "mise run validate échoue" "le référentiel est incohérent : rien à publier tant qu'il ment"
fi

if mise run vet >/dev/null 2>&1; then
  ok "mise run vet"
else
  ko "mise run vet échoue" "l'analyse statique du compilateur refuse ce code"
fi

# 2 est une dérive, 1 une erreur, et aucune des deux ne s'embarque : une release
# publiée avec un index SCSL non retrié cite des références normatives que
# l'amont a déjà déplacées.
mise run scsl-drift >/dev/null 2>&1
case $? in
  0) ok "l'index SCSL est celui que la baseline décrit" ;;
  2) ko "l'index SCSL a dérivé depuis la baseline" "python3 scripts/scsl-drift.py pour la liste ; trier, puis mise run scsl-drift-update" ;;
  *) ko "scsl-drift n'a pas pu tourner" "cloner framework-scsl à côté du dépôt (l'index est requis pour publier)" ;;
esac

# --- le canari : ce que les vrais plans de contrôle ont répondu, et quand ------
#
# La colonne « live » de la matrice est DÉRIVÉE des descripteurs : elle dit ce que
# Pépin croit savoir collecter, jamais ce qu'il a observé. Le canari est la seule
# mesure du dépôt qui interroge le vrai plan de contrôle d'un fournisseur, et il
# le fait SANS identifiant — il envoie des valeurs synthétiques et mesure le refus.
#
# Ce contrôle-ci ne détient donc aucun secret : il lit une DATE dans un fichier
# committé. C'est précisément ce que l'issue #59 demande — le geste est local et
# manuel, le préflight n'exige que sa fraîcheur.
#
# La complétude (chaque fournisseur cloud a son relevé, lisible, substantiel) est
# vérifiée par internal/canary, donc par `mise run test` ci-dessus : elle ne dépend
# pas de la date et n'a rien à faire dans une porte qui ne parle qu'au moment d'un
# tag. Ici, seule la FRAÎCHEUR se juge.
#
# 90 jours : la même valeur que canary.MaxAge, et TestThePreflightCitesTheSameFreshnessWindow
# refuse que les deux divergent.
CANARY_MAX_AGE_DAYS=90
canary_dir=references/canary
if [ ! -d "$canary_dir" ] || ! ls "$canary_dir"/*.yaml >/dev/null 2>&1; then
  ko "aucun relevé de canari dans $canary_dir" \
     "lancer tools/release/canary.sh : une release ne promeut pas une capacité live que rien n'a vue répondre"
else
  now_s=$(date +%s)
  stale=""
  undated=""
  for rec in "$canary_dir"/*.yaml; do
    p=$(basename "$rec" .yaml)
    recorded=$(sed -n 's/^recorded:[[:space:]]*//p' "$rec" | head -1 | tr -d '"'"'"' ')
    rec_s=$(date -d "$recorded" +%s 2>/dev/null)
    if [ -z "$recorded" ] || [ -z "$rec_s" ]; then
      undated="$undated $p"
      continue
    fi
    age_days=$(( (now_s - rec_s) / 86400 ))
    if [ "$age_days" -gt "$CANARY_MAX_AGE_DAYS" ]; then
      stale="$stale $p(${age_days}j)"
    fi
  done
  if [ -n "$undated" ]; then
    ko "relevé de canari sans date lisible :$undated" \
       "un relevé qu'on ne sait pas dater n'atteste pas sa fraîcheur ; le refaire"
  elif [ -n "$stale" ]; then
    ko "relevé de canari plus vieux que $CANARY_MAX_AGE_DAYS jours :$stale" \
       "relancer tools/release/canary.sh, relire le relevé, le committer"
  else
    ok "les relevés de canari ont moins de $CANARY_MAX_AGE_DAYS jours"
  fi
fi

# --- les promesses que le README publie, éprouvées en les déclenchant ---------
#
# Les codes de sortie et le bundle sont la surface que les CI des consommateurs
# branchent. On ne les lit pas dans une constante : on lance le binaire qui va
# être publié et on regarde ce qu'il répond. Un contrôle de comportement, pas de
# forme (« vérifier, ce n'est pas parser »).

if mise run build >/dev/null 2>&1 && [ -x ./pepin ]; then
  ok "le binaire se construit"

  ./pepin scan scaleway examples/scaleway/inventory.json --format json >/dev/null 2>&1
  rc_bad=$?
  ./pepin scan scaleway examples/scaleway/inventory-ok.json --format json >/dev/null 2>&1
  rc_good=$?
  ./pepin scan fournisseur-inconnu export.json >/dev/null 2>&1
  rc_err=$?
  if [ "$rc_bad" -eq 1 ] && [ "$rc_good" -eq 0 ] && [ "$rc_err" -eq 2 ]; then
    ok "les codes de sortie répondent ce que le README promet (0 conforme, 1 écart, 2 erreur)"
  else
    ko "les codes de sortie ont bougé : écart=$rc_bad (attendu 1), conforme=$rc_good (attendu 0), erreur=$rc_err (attendu 2)" \
       "toute CI consommatrice teste \$? ; voir cmd/surface.go et la fixture cli.json"
  fi

  workdir="$(mktemp -d)"
  if ./pepin scan scaleway examples/scaleway/inventory.json --format json \
       --seal "$workdir/bundle" >/dev/null 2>&1 || [ $? -eq 1 ]; then
    if ./pepin verify "$workdir/bundle" --re-derive >/dev/null 2>&1; then
      ok "un bundle scellé par ce binaire se vérifie et se re-dérive"
      # Et la vérification doit pouvoir échouer : un verify qui accepte un
      # bundle altéré ne prouve rien du tout.
      printf ' ' >> "$workdir/bundle/assessment.json"
      if ./pepin verify "$workdir/bundle" >/dev/null 2>&1; then
        ko "verify accepte un bundle ALTÉRÉ" "l'opposabilité repose sur ce refus ; voir internal/assess/bundle.go"
      else
        ok "verify refuse le même bundle altéré d'un octet"
      fi
    else
      ko "le bundle scellé ne se vérifie pas" "./pepin verify --re-derive sur un bundle frais doit rendre 0"
    fi
  else
    ko "scan --seal n'a pas produit de bundle" "la promesse d'opposabilité ne part pas sans lui"
  fi
  rm -rf "$workdir"
else
  ko "mise run build échoue" "rien d'autre ne se mesure sans binaire"
fi

# --- le numéro que les commits impliquent -------------------------------------
#
# La version n'est pas une préférence : les Conventional Commits disent
# l'incrément (.cz.toml, version_provider = "scm"). Rapporté plutôt que refusé
# quand commitizen est inaccessible : un mainteneur sans l'outillage Python doit
# pouvoir couper une release ; mais uvx suffit, sans rien installer.
#
# La version est épinglée ici parce qu'aucun hook du dépôt ne l'épingle encore :
# deux versions du même outil en désaccord sur un incrément est un défaut que
# personne n'irait chercher. Si un hook commitizen arrive un jour dans
# .pre-commit-config.yaml, c'est sa version qui devra faire foi.
CZ_PIN="4.16.5"
cz_cmd=""
if command -v cz >/dev/null 2>&1; then
  cz_cmd="cz"
elif command -v uvx >/dev/null 2>&1; then
  cz_cmd="uvx --from commitizen==$CZ_PIN cz"
fi

if [ -n "$cz_cmd" ]; then
  implied="$($cz_cmd bump --dry-run --yes 2>/dev/null | grep -oE 'v?[0-9]+\.[0-9]+\.[0-9]+' | tail -1)"
  if [ -z "$implied" ]; then
    ok "aucun incrément à dériver des commits"
  elif [ "v${implied#v}" = "$VERSION" ]; then
    ok "$VERSION est ce que les commits depuis le dernier tag impliquent"
  else
    ko "les commits impliquent v${implied#v}, pas $VERSION" \
       "lire : $cz_cmd bump --dry-run. Taguer ce que les commits disent, ou dire pourquoi dans le CHANGELOG"
  fi
else
  printf "  %s·%s ni cz ni uvx : la version impliquée par les commits n'a pas été dérivée\n" "$DIM" "$OFF"
fi

# --- ce qu'un lecteur de la release cherchera ---------------------------------
#
# Les deux langues, parce que le dépôt promet leur synchronisation : une section
# présente en anglais et absente en français est une release que la moitié des
# lecteurs ne verra pas.

for f in CHANGELOG.md CHANGELOG.fr.md; do
  if grep -q "^## \[${VERSION#v}\]" "$f" 2>/dev/null; then
    ok "$f a une section pour ${VERSION#v}"
  else
    ko "$f n'a pas de section ## [${VERSION#v}]" \
       "y déplacer les entrées Unreleased ; le corps de la GitHub Release se lit là"
  fi
done

if [ "$(grep -o '^## \[[0-9][^]]*\]' CHANGELOG.md 2>/dev/null)" = "$(grep -o '^## \[[0-9][^]]*\]' CHANGELOG.fr.md 2>/dev/null)" ]; then
  ok "les deux CHANGELOG portent les mêmes versions"
else
  ko "CHANGELOG.md et CHANGELOG.fr.md ne listent pas les mêmes versions" "les deux langues se publient ensemble"
fi

# --- le GIF du README montre la version qu'on publie ----------------------------
#
# Le GIF est la première chose que voit un lecteur du README, et il affiche un
# `pepin version`. S'il porte une version antérieure, la page d'accueil du projet
# annonce durablement une démo d'un produit qui n'est plus celui qu'on livre.
#
# On ne lit évidemment pas la version DANS l'image : `mise run demo` consigne à
# côté celle qu'il a injectée au build, et c'est ce fichier qu'on compare. Le GIF
# et son relevé sont écrits par la même commande, donc l'un ne peut pas mentir
# sans l'autre.
#
# Les DEUX GIF sont exigés : un lecteur anglophone qui ouvre README.md et un
# lecteur francophone qui ouvre README.fr.md doivent chacun voir une démo dans
# leur langue. N'en contrôler qu'un laisserait l'autre se périmer en silence.
demo_version_file=docs/assets/quickstart.version
if [ ! -f docs/assets/quickstart.gif ] || [ ! -f docs/assets/quickstart.fr.gif ] || [ ! -f "$demo_version_file" ]; then
  ko "un GIF de démonstration est absent (anglais et français sont tous deux requis)" \
     "lancer PEPIN_DEMO_VERSION=$VERSION mise run demo"
elif [ "$(tr -d ' \n' <"$demo_version_file")" = "$VERSION" ]; then
  ok "les GIF du README montrent $VERSION"
else
  ko "les GIF du README montrent $(tr -d ' \n' <"$demo_version_file"), pas $VERSION" \
     "lancer PEPIN_DEMO_VERSION=$VERSION mise run demo et committer le résultat"
fi

# --- la CI sur ce commit exact --------------------------------------------------
#
# Non relancé ici : audit (gosec, govulncheck, OSV) et le schéma OSCAL du NIST
# demandent le réseau et des outils que cette machine peut ne pas avoir. La CI
# les fait tourner sur chaque push ; ceci vérifie qu'elle l'a fait sur CE
# commit. Une mesure qui échoue et un résultat absent sont deux réponses
# différentes, et une seule des deux parle du dépôt.
# TOUS les jobs qui gardent la livraison, pas seulement ceux de ci.yml. Le filtre
# précédent ne retenait que « Build + Test » et « golangci-lint » : CodeQL, l'audit,
# la revue de dépendances, le scan de secrets et les deux jobs entrypoints pouvaient
# être ROUGES pendant que le preflight annonçait « la CI est verte » et autorisait le
# tag. Les lignes ci-dessus délèguent l'audit à la CI : la délégation doit pointer
# vers ce qu'elle prétend déléguer.
# NB : ce motif est passé à awk via -v, donc awk le lit comme une CHAÎNE avant
# d'en faire une regex : les échappements doivent être DOUBLÉS. Avec un simple
# `\(`, awk émet « escape sequence treated as plain ( » et la parenthèse devient
# un GROUPE — le motif ne matche plus aucun nom de job, et le preflight conclut
# « la CI est 'null' » au lieu de lire son état.
ci_jobs='^(Build \\+ Test|golangci-lint|Audit \\(gosec \\+ govulncheck\\)|Analyze \\(go\\)|Dependency Review|Secret scanning|Secret scanning \\(pattern-based\\)|The image scans for real|The action installs only what it verified)$'
if git remote | grep -q . && command -v gh >/dev/null 2>&1; then
  sha="$(git rev-parse HEAD)"
  if runs="$(gh api "repos/{owner}/{repo}/commits/$sha/check-runs?per_page=100" \
             --jq '.check_runs[] | "\(.name)\t\(.conclusion)"' 2>/dev/null)"; then
    state="$(printf '%s\n' "$runs" \
             | awk -F'\t' -v re="$ci_jobs" '$1 ~ re { print $2 }' \
             | sort -u | paste -sd, -)"
    case "$state" in
      success)   ok "la CI est verte sur ce commit" ;;
      "")        ko "aucun run de CI trouvé pour ce commit" "le pousser et attendre la CI" ;;
      *)         ko "la CI est '$state' sur ce commit" "on ne tague pas ce que la CI refuse" ;;
    esac
  else
    ko "impossible d'interroger la CI sur ce commit" "gh api a échoué : vérifier l'authentification, pas le commit"
  fi
else
  printf "  %s·%s CI non vérifiée (pas de remote, ou gh absent)\n" "$DIM" "$OFF"
fi

echo
if [ "$failures" -gt 0 ]; then
  printf "%s%d contrôle(s) en échec ; rien n'a été tagué.%s\n" "$RED" "$failures" "$OFF" >&2
  exit 1
fi
cat <<EOF
${GREEN}prêt.${OFF} Alors, et alors seulement :

  git tag -a $VERSION -m "$VERSION"
  git push origin $VERSION

Pousser le tag est ce qui publie. Cela ne s'annule pas discrètement : un tag se
supprime des deux côtés, et une release qui a atteint le monde a été téléchargée.
EOF

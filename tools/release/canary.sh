#!/usr/bin/env bash
#
# LE CANARI : la seule mesure du dépôt qui interroge le VRAI plan de contrôle
# d'un fournisseur souverain.
#
#   tools/release/canary.sh                 # les trois fournisseurs
#   tools/release/canary.sh scaleway        # un seul
#
# ─── LE PROBLÈME QU'IL RÉSOUT ────────────────────────────────────────────────
#
# La colonne « live » de docs/coverage.md est DÉRIVÉE des descripteurs : elle dit
# ce que Pépin croit savoir collecter, jamais ce qu'il a observé. Une release qui
# promeut une capacité live validée uniquement sur des fixtures et un émulateur
# promeut une croyance.
#
# ─── LE CANARI NE DÉTIENT AUCUN IDENTIFIANT, ET C'EST SA FORCE ───────────────
#
# Il n'en a pas besoin : il envoie des valeurs SYNTHÉTIQUES, que le fournisseur
# refusera. Ce qui se mesure est le REFUS : un endpoint qui répond 401 ou 403
# EXISTE, répond, et sa classe de refus est celle que Pépin publie. Un endpoint
# déplacé répondrait 404 — c'est précisément la régression qu'un descripteur ne
# peut pas voir venir.
#
# Le script COMMENCE par effacer toute variable d'identifiants de l'environnement.
# Un mainteneur qui le lance depuis son shell de travail n'envoie donc PAS ses
# clés : c'est une propriété du script, pas une consigne d'usage. Et rien de ce
# qu'il écrit ne peut porter un secret, puisqu'aucun secret n'entre.
#
# ─── CE QU'IL PROUVE, ET CE QU'IL NE PROUVE PAS ──────────────────────────────
#
#   Il établit                              | Il n'établit pas
#   ----------------------------------------|-----------------------------------
#   que l'hôte compilé dans le descripteur  | les NOMS et TYPES des champs du
#   se résout et répond                     | contrat natif
#   que le chemin déclaré existe encore     | ce qu'un tenant contient
#   la classe qu'un refus reçoit            | qu'un droit SUFFISANT rende 200
#
# Un refus de SIGNATURE n'est pas un refus de DROIT. Que le fournisseur réponde
# la même chose à une clé valide sans permission reste dû à un scan authentifié,
# et le relevé le dit à sa place.
#
# ─── CE QU'IL ENVOIE, ET À QUELLE FRÉQUENCE ──────────────────────────────────
#
# Une requête par endpoint déclaré, une fois par qualification de release. C'est
# un appel non authentifié à une API publique, pas une campagne.
set -uo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/../.." || exit 1

GREEN=$'\033[32m'; RED=$'\033[31m'; DIM=$'\033[2m'; OFF=$'\033[0m'
[ -t 1 ] || { GREEN=""; RED=""; DIM=""; OFF=""; }

OUTDIR=references/canary
mkdir -p "$OUTDIR"

command -v python3 >/dev/null 2>&1 || { echo "python3 absent du PATH" >&2; exit 2; }

# --- l'effacement, AVANT toute autre chose ------------------------------------
#
# Tout ce qu'un descripteur sait lire dans l'environnement. La liste est longue à
# dessein : une variable oubliée ici est une clé de mainteneur qui part sur le
# réseau, et le canari deviendrait exactement la fuite qu'il prétend éviter.
unset SCW_ACCESS_KEY SCW_SECRET_KEY SCW_DEFAULT_ORGANIZATION_ID SCW_DEFAULT_PROJECT_ID \
      SCW_API_URL SCW_PROFILE SCW_CONFIG_PATH \
      OSC_ACCESS_KEY OSC_SECRET_KEY OSC_ENDPOINT_API OSC_PROFILE OSC_SECRET_KEY_ID \
      EXOSCALE_API_KEY EXOSCALE_API_SECRET EXOSCALE_API_ENDPOINT \
      AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY AWS_SESSION_TOKEN AWS_PROFILE

# Des jetons SYNTHÉTIQUES, de la forme que les fournisseurs valident, et dont le
# contenu dit ce qu'ils sont. Aucun n'ouvre quoi que ce soit.
export SCW_ACCESS_KEY='SCWCANARYCANARYCANAR'
export SCW_SECRET_KEY='00000000-0000-4000-8000-000000000000'
export SCW_DEFAULT_ORGANIZATION_ID='00000000-0000-4000-8000-000000000000'
export SCW_DEFAULT_PROJECT_ID='00000000-0000-4000-8000-000000000000'
export SCW_DEFAULT_REGION="${SCW_DEFAULT_REGION:-fr-par}"
export SCW_DEFAULT_ZONE="${SCW_DEFAULT_ZONE:-fr-par-1}"
export OSC_ACCESS_KEY='CANARYCANARYCANARYCA'
export OSC_SECRET_KEY='CANARYCANARYCANARYCANARYCANARYCANARYCANA'
export OSC_REGION="${OSC_REGION:-eu-west-2}"
export EXOSCALE_API_KEY='EXOCANARYCANARYCANARYCAN'
export EXOSCALE_API_SECRET='CANARYCANARYCANARYCANARYCANARYCANARYCANA'
export EXOSCALE_ZONE="${EXOSCALE_ZONE:-ch-dk-2}"
# --- le binaire mesuré, compilé AVANT que HOME ne bouge ------------------------
#
# L'ordre compte. `go build` dérive GOPATH — donc le cache de modules — de HOME :
# compilé sous le HOME jetable, il retéléchargerait tout le graphe de dépendances
# à chaque canari, puis échouerait à l'effacer (le cache de modules est posé en
# lecture seule). La compilation ne lit aucun identifiant de fournisseur ; c'est
# le SCAN qu'il faut isoler, et lui seul.
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo dev)
go build -trimpath -ldflags "-s -w -X github.com/stephrobert/pepin/cmd.version=$VERSION" \
   -o ./pepin . || { echo "compilation impossible" >&2; exit 2; }

# HOME jetable : aucun ~/.config/scw/config.yaml ni ~/.osc/config.json ne doit
# être lu. La résolution d'identifiants lit ces fichiers quand l'environnement
# est muet, et l'environnement ci-dessus pourrait ne pas couvrir un futur champ.
CANARY_HOME=$(mktemp -d)
export HOME="$CANARY_HOME"

PROVIDERS=("$@")
[ ${#PROVIDERS[@]} -gt 0 ] || PROVIDERS=(scaleway outscale exoscale)

work=$(mktemp -d)
trap 'rm -rf "$CANARY_HOME" "$work"' EXIT

rc=0
for p in "${PROVIDERS[@]}"; do
  printf '%s' "${DIM}canari $p …${OFF} "
  # Un code de sortie non nul est un RÉSULTAT : sans droits, rien ne se conclut,
  # donc 3 est attendu. Seule l'absence de document JSON est une panne.
  timeout 180 ./pepin scan "$p" --live --format json --lang en \
    > "$work/$p.json" 2> "$work/$p.err"
  if ! python3 tools/release/canary-record.py "$p" "$work/$p.json" "$VERSION" "$OUTDIR/$p.yaml"; then
    echo "${RED}✘${OFF} relevé non écrit (voir $work/$p.err)"
    rc=1
    continue
  fi
  echo "${GREEN}✔${OFF} $OUTDIR/$p.yaml"
done

echo
echo "Relire chaque relevé AVANT de committer : il ne doit porter ni identifiant,"
echo "ni nom de ressource, ni rien d'un tenant. Le canari n'en produit pas — le"
echo "vérifier est ce qui fait qu'on peut l'affirmer."
exit "$rc"

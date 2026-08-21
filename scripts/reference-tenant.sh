#!/usr/bin/env bash
#
# Régénère le plan d'un TENANT DE RÉFÉRENCE depuis sa provenance.
#
#   ./scripts/reference-tenant.sh references/tenants/scaleway/ducklake
#   ./scripts/reference-tenant.sh --all
#
# ─── CE QUE CETTE PROCÉDURE FAIT, ET NE FAIT PAS ─────────────────────────────
#
# Elle clone la configuration TIERCE au commit consigné, en génère un plan
# Terraform, et n'en garde que ce que Pépin lit. `terraform plan` NE PROVISIONNE
# RIEN : aucune ressource cloud n'est créée, donc rien n'est à détruire
# (CONTRIBUTING.md, règle 5). Les identifiants passés aux providers sont
# FACTICES et publics ; aucun compte n'est joint, aucun `refresh` n'est fait.
#
# Elle ne fabrique donc PAS un tenant réel. Un plan porte l'état PLANIFIÉ ; ce
# qu'un fournisseur RÉPOND reste dû à une collecte live, et la carte de qualité
# de détection le dit à sa place.
#
# ─── POURQUOI RÉGÉNÉRER PLUTÔT QUE CROIRE ────────────────────────────────────
#
# Un plan committé sans moyen de le refaire est une affirmation. Cette procédure
# est ce qui permet à un relecteur de vérifier que le tenant vient bien du dépôt
# tiers annoncé, et non d'un fichier que Pépin a fini par s'écrire à lui-même —
# c'est-à-dire exactement la fixture auto-confirmante qu'on cherche à quitter.
#
# Requiert terraform et l'accès au registre de providers. Le plan régénéré doit
# être IDENTIQUE à celui qui est committé : s'il diverge, l'amont a bougé sous un
# commit épinglé (impossible), ou la version du provider a changé la forme du
# plan — dire laquelle avant de committer.
set -uo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.." || exit 1

HERE="$PWD"
GREEN=$'\033[32m'; RED=$'\033[31m'; DIM=$'\033[2m'; OFF=$'\033[0m'
[ -t 1 ] || { GREEN=""; RED=""; DIM=""; OFF=""; }

command -v terraform >/dev/null 2>&1 || { echo "terraform absent du PATH" >&2; exit 2; }
command -v python3   >/dev/null 2>&1 || { echo "python3 absent du PATH" >&2; exit 2; }

# La socket que le greffon Terraform ouvre vit dans TMPDIR, et une socket unix
# est bornée à 108 octets. Un TMPDIR profond fait échouer le plan avec un
# « bind: invalid argument » qui ne parle pas de sa cause.
TMPDIR="${TMPDIR:-/tmp}"
case "${#TMPDIR}" in
  ?|??|???) ;;
  *) TMPDIR=/tmp ;;
esac
export TMPDIR TF_IN_AUTOMATION=1 CHECKPOINT_DISABLE=1

one() {
  local tenant="$1"
  local manifest="$tenant/tenant.yaml"
  [ -f "$manifest" ] || { echo "$RED✘$OFF $manifest introuvable"; return 1; }

  local repo commit path
  repo=$(   sed -n 's/^  repo:[[:space:]]*//p'   "$manifest" | head -1)
  commit=$( sed -n 's/^  commit:[[:space:]]*//p' "$manifest" | head -1)
  path=$(   sed -n 's/^  path:[[:space:]]*//p'   "$manifest" | head -1)
  [ -n "$repo" ] && [ -n "$commit" ] || { echo "$RED✘$OFF $manifest : provenance incomplète"; return 1; }

  local work; work=$(mktemp -d)
  # Le piège capture la valeur d'AUJOURD'HUI, pas celle du moment où il se déclenche :
  # $work change à chaque tenant, et un piège tardif effacerait le mauvais dossier.
  # shellcheck disable=SC2064
  trap "rm -rf '$work'" RETURN

  echo "${DIM}clone $repo @ ${commit:0:8}${OFF}"
  git -C "$work" init -q . >/dev/null 2>&1 || return 1
  git -C "$work" remote add origin "$repo" >/dev/null 2>&1
  git -C "$work" fetch -q --depth 1 origin "$commit" >/dev/null 2>&1 || {
    # Un serveur qui refuse le fetch par SHA : cloner puis se placer dessus.
    rm -rf "$work"; mkdir -p "$work"
    git clone -q "$repo" "$work" >/dev/null 2>&1 || { echo "$RED✘$OFF clone impossible"; return 1; }
  }
  git -C "$work" checkout -q "$commit" 2>/dev/null || git -C "$work" checkout -q FETCH_HEAD 2>/dev/null || {
    echo "$RED✘$OFF commit $commit introuvable dans $repo"; return 1; }

  local dir="$work/$path"
  [ -d "$dir" ] || { echo "$RED✘$OFF chemin « $path » absent de l'amont"; return 1; }

  ( cd "$dir" || exit 1
    python3 "$HERE/scripts/tenant-plan.py" placeholders . >/dev/null || exit 1
    terraform init  -input=false                          >/dev/null 2>&1 || exit 1
    terraform plan  -input=false -refresh=false -lock=false -out=tfplan.bin >/dev/null 2>&1 || exit 2
    terraform show  -json tfplan.bin > plan-full.json      2>/dev/null     || exit 3
    python3 "$HERE/scripts/tenant-plan.py" reduce plan-full.json "$HERE/$tenant/plan.json" >/dev/null ) || {
      echo "$RED✘$OFF $tenant : le plan n'a pas pu être régénéré (rc=$?)"; return 1; }

  echo "$GREEN✔$OFF $tenant"
}

if [ "${1:-}" = "--all" ]; then
  rc=0
  for t in references/tenants/*/*; do
    [ -d "$t" ] || continue
    one "$t" || rc=1
  done
  echo
  echo "Relire le diff, puis : mise run tenants-update && mise run veracity-update"
  exit "$rc"
fi

[ $# -eq 1 ] || { echo "usage : $0 <references/tenants/<fournisseur>/<nom>> | --all" >&2; exit 1; }
one "${1%/}"
